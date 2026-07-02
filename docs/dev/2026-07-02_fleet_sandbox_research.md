# Sandboxing the fleet code-executor — research + recommendation

> **TL;DR / вывод.** For our deployment reality (fly.io Firecracker VMs + bare-metal Alpine, `CGO_ENABLED=0`, pure-Go binary) the best single default is a **thin re-exec child launcher** that, just before `exec`, applies: an **empty network namespace** (net-off by default), **`go-landlock`** filesystem confinement (read-only system dirs + writable workdir only), **rlimits** (CPU/FSIZE/NOFILE/NPROC/AS), and drops privileges + sets `NO_NEW_PRIVS`. All of that is pure-Go, no CGO, no daemon, no external binary, and works unprivileged inside a Fly VM. When an operator wants stronger isolation on a self-hosted box, **shell out to `nsjail`** (one apk-installable binary that bundles namespaces + cgroups v2 + rlimits + seccomp — the exact shape Windmill ships in production). Reserve **gVisor `runsc` (systrap platform)** for the untrusted-role-author tier; it runs on Fly without `/dev/kvm` and the simplepanel already exposes it. WASM/wazero is **not viable** for fleet's data work (WASI has no `dlopen`, so numpy/pandas cannot load).

This report builds on and refines the earlier tiered design in [`process_isolation.md`](./process_isolation.md); read that first for the primitive-by-primitive detail. Here the focus is: (1) grounding in our actual code, (2) how real Go projects solve this, and (3) a concrete, deployment-aware integration plan.

---

## 1. Current state & threat model

### What ships today

The single exec choke point is `agentruntime.RunBlock` in [`internal/agentruntime/coderun.go`](../../internal/agentruntime/coderun.go). Both entry paths funnel through it:

- **`executor: code`** roles → `RunCode` ([`runcode.go`](../../internal/agentruntime/runcode.go)) → `RunBlock`.
- The **LLM `exec` tool** → `runtime.go:371` → `RunBlock`.

Wiring: `internal/fleet/handler.go` passes `f.cfg.AllowedPrograms` (CLI `--allowed-programs`, [`cmd/fleet/main.go`](../../cmd/fleet/main.go)) into both. Interpreters are declared in [`interpreters.json`](../../internal/agentruntime/interpreters.json): `python3`, `node`, `bash`, `ruby`, `php`, `perl` — real native binaries spawned via `exec.CommandContext`.

Isolation currently in `RunBlock`:

- **Secret-scrubbed env** — `buildChildEnv` builds a minimal `PATH` + `FLEET_INPUT` allowlist; parent env is never inherited (hard invariant against leaking JWT/LLM keys).
- **Per-run throwaway workdir** — `os.MkdirTemp` → `cmd.Dir`, `RemoveAll` on return.
- **Context timeout** — `exec.CommandContext` with optional `spec.Timeout`.
- **stdout/stderr byte cap** — `limitedBuffer` (default 1 MiB).
- **Off by default** — empty `--allowed-programs` disables code execution entirely.
- **Write-path scope** — applied *after* the run: `NewScopedKB(nil, WritePatterns)` enforces `write_patterns` on the returned changes, same as `write_note`. This gates *what the KB persists*, not what the child process can touch on the host.

### Threat model

Self-hosted, typically Linux. Role *notes* are operator-trusted, but the *code they render* can be adversarial or prompt-injected. The child runs a real interpreter with the executor's uid and full host network/filesystem access.

**What the baseline does NOT stop:** unbounded CPU/RAM/PIDs/disk within the timeout window, arbitrary **network** access (exfiltration), reading anything the executor uid can read (the temp cwd is not a boundary — the child can `chdir` away or use absolute paths), writing anywhere the uid can write, and dangerous syscalls. The write-scope only sanitizes the *reported changes*; it does nothing about side effects the process performs directly.

### Deployment constraints to respect

- Pure-Go, **`CGO_ENABLED=0`** → prefer libraries that call the kernel via `golang.org/x/sys/unix` (no libseccomp/libcap C deps).
- Base image **Alpine (musl)**.
- Deploys on **fly.io** (each app is a Firecracker microVM) and **bare-metal Linux**.
- simplepanel already exposes a **gVisor `runsc`** runtime option for agents — the runtime is a known, available quantity.

---

## 2. Options survey

### 2a. Deployment-compatibility facts (fly.io + Alpine + no-CGO)

These determine which options are even usable for us; each is cited in §6.

- **Fly kernel = 5.15.x LTS.** ≥ 5.13 → Landlock **filesystem** rules (ABI v1–v3) work. **< 6.7 → Landlock network (TCP) rules do NOT** — so "network off" on Fly must come from a **net namespace**, not Landlock.
- **Fly = full VM, you run as root inside it.** `CLONE_NEWUSER` and the other namespaces are available; there is no outer container seccomp filter blocking `unshare`.
- **No `/dev/kvm` inside a Fly machine** → you cannot nest Firecracker/Kata/QEMU. gVisor's KVM platform is out, but its **systrap** (ptrace+seccomp) platform works and is what gVisor recommends inside a VM.
- **cgroups on Fly are a broken hybrid v1+v2 layout.** Container runtimes (podman/runsc) fail to auto-configure limits; `cgroup_no_v1=all` **crashes the VM at boot**. A Go program can still write a v2 subtree directly (`memory.max`, `pids.max`) as root-in-VM, but do not rely on runtime auto-detection.
- **Alpine packages:** `bubblewrap` (main), `nsjail` (community), `prlimit` via `util-linux-misc` (main); `runsc` is a **static Go binary** (download, no apk) — all musl-clean.
- **Pure-Go libs are CGO-free and musl-clean:** `landlock-lsm/go-landlock` and `elastic/go-seccomp-bpf` both use raw syscalls, no libc. Alpine `linux-lts` enables Landlock by default; whether **Fly's custom 5.15 kernel** compiles in Landlock (`CONFIG_SECURITY_LANDLOCK` + `lsm=`) is **undocumented** → must probe at runtime and degrade with `BestEffort()`.

### 2b. Techniques table

| Technique | Isolates (fs / net / cpu / mem / pids / syscalls) | No-CGO? | Alpine / fly.io OK? | Effort |
|---|---|---|---|---|
| **rlimits** (`unix.Prlimit` in re-exec child, or `prlimit(1)`) | cpu, file-size, fds, procs, coarse-mem (AS) | Yes (x/sys/unix) | Yes / Yes (Unix-gated) | Low |
| **Net namespace** (`SysProcAttr.Cloneflags CLONE_NEWNET` + `CLONE_NEWUSER`) | **net (full off)** | Yes (stdlib) | Yes / Yes (root-in-VM or userns) | Low |
| **Drop uid + `NO_NEW_PRIVS`** (`SysProcAttr.Credential`, `Prctl`) | privilege-gain, cross-uid reads | Yes | Yes / Yes | Low |
| **go-landlock** (`RODirs`/`RWDirs`, `BestEffort`) | **fs** (open/read/write/exec/create) | Yes | Yes (5.13+) / Fly kernel Landlock unverified → BestEffort | Low-Med |
| **cgroups v2** (write `memory.max`/`pids.max`/`cpu.max`) | **real mem, pids, cpu** | Yes (fs writes) | Yes bare-metal / Fly: direct writes only, not via runtime | Med |
| **seccomp-bpf** (`elastic/go-seccomp-bpf` in child) | **syscalls** (allow/deny) | Yes | Yes / Yes | Med (must be in child, needs NO_NEW_PRIVS) |
| **bubblewrap** (`bwrap` binary) | fs, net, pid, ipc, uts, caller-supplied seccomp; **no** cgroup/rlimit | shell-out | apk main / yes | Med |
| **nsjail** (`nsjail` binary) | **ns + cgroups v2 + rlimits + seccomp**, one tool | shell-out | apk community / yes | Med |
| **firejail** | ns + seccomp; **setuid-root** | shell-out | — | avoid on server (LPE CVE history) |
| **gVisor `runsc`** (systrap platform) | **all** (userspace kernel intercepts syscalls) | shell-out (runsc is Go) | static binary / **yes, systrap (no KVM needed)** | Med-High |
| **Firecracker / Kata microVM** | **all** (own guest kernel) | SDK/OCI | **NO on fly.io (no /dev/kvm)**; bare-metal only | High |
| **wazero + CPython WASI** | capability model, in-process | Yes (pure Go) | Yes / Yes — **but breaks on native deps** | — (see below) |
| **Embedded Go interpreters** (risor/yaegi/starlark/tengo/goja/gopher-lua) | in-process; no real python | Yes | Yes | — (wrong language) |

### 2c. Why WASM and embedded interpreters do NOT fit fleet

- **wazero + CPython (WASI):** wazero is a genuine pure-Go capability sandbox and CPython has a `wasm32-wasi` target, so pure-Python-stdlib scripts run (filesystem via preopens, stdin/stdout, ~2–5× slower). **But WASI has no `dlopen`/`dlsym`**, so CPython cannot load C-extension `.so` files — `import numpy` fails with a hard `ModuleNotFoundError`. Fleet's KB/data-processing work is exactly numpy/pandas territory, so this is a dealbreaker. Pyodide *has* numpy/pandas but is **Emscripten**, not WASI — it does not run under wazero at all (and Cohere's Pyodide-based "Terrarium" shipped a CVSS-9.3 sandbox escape, CVE-2026-5752 — Pyodide was never a security boundary). Verdict: keep wazero on the radar as a *parallel track for pure-compute Wasm plugins*, not as the native-interpreter path.
- **Embedded Go interpreters** (risor, yaegi, starlark-go, tengo, goja, gopher-lua): they run in-process without `exec`, which is attractive, but **none run real Python/bash/node** and most run in the *same address space* as the daemon (no memory isolation, no CGO barrier). They solve a different problem (scripting the host in a Go-ish DSL), not "run the operator's allowlisted interpreters on adversarial code." Not a fit.

---

## 3. How popular Go projects actually do it

Named primitive per project (sources in §6):

- **Windmill** (the closest analogue — a worker that spawns python/bash/deno via subprocess): **nsjail** wraps each job (user+pid+mount+net namespaces, cgroups, rlimits, seccomp via Kafel), with a lighter fallback of `unshare --user --map-root-user --pid --fork --mount-proc` (blocks `/proc/$pid/mem` and `/proc/$pid/environ` snooping). Isolation is **off by default**; Deno is trusted via its own V8 capability flags. **Забрать:** the whole model — nsjail-per-subprocess with a per-language config, plus the namespace-only fallback for when nsjail isn't installed. This maps 1:1 onto our `RunBlock`.
- **Google nsjail + kCTF:** nsjail was purpose-built by Google for adversarial code (CTF contestants). Confirms nsjail is the battle-tested single-binary answer for untrusted subprocess execution. **Забрать:** trust the tool; borrow their Kafel seccomp posture as a starting allowlist.
- **Google gVisor** (Cloud Run, App Engine 2nd-gen, GKE Sandbox, new "GKE Agent Sandbox" for AI agents): userspace Linux kernel (`runsc`), intercepts ~300 syscalls in Go. **Забрать:** the runtime we already expose in simplepanel; use systrap platform on Fly.
- **OpenAI Code Interpreter** and **Modal** sandboxes: both use **gVisor `runsc`** for LLM-generated code at scale (Modal reported 250k concurrent). **Забрать:** validation that gVisor is the right strength/effort point for LLM-authored code short of full microVMs.
- **E2B** (open-source, Apache-2.0): **Firecracker microVM** per sandbox (<200ms boot, own kernel); used by HuggingFace smolagents, Perplexity. **Забрать:** the "sandbox is an external service you call" architecture — relevant only for bare-metal since Fly has no nested KVM.
- **HuggingFace smolagents:** its `LocalPythonInterpreter` is explicitly documented as **not** a security boundary; production uses `executor_type="e2b"` or `"docker"`. **Забрать:** the honesty — an in-process Python AST filter is not isolation; don't pretend otherwise.
- **Dagger:** trusted BuildKit daemon runs steps as OCI containers via **runc + containerd namespaces** (exposes AppArmor/SELinux/idmapping). **Coder/envbox:** **sysbox-runc** for rootless nested containers via deep user-namespace support. **Woodpecker/Drone, nektos/act, Gitea act_runner:** **one Docker container per step/job** (act's `host` mode = *no* isolation; Gitea's mounted `docker.sock` is a known escape). **Забрать:** container-per-unit is the CI floor, but it's heavier than we need for short-lived interpreter runs and assumes a Docker daemon.
- **Temporal:** does **not** sandbox activity code (Go activities are plain goroutines; the Python "sandbox" only enforces determinism). Their guidance for agent code is to delegate to an external sandbox (E2B/Modal) inside a durable activity. **Забрать:** the pattern of wrapping the sandbox call, not the (absent) isolation.
- **Fermyon Spin / wazero embedders (Trivy, Dapr, Redpanda):** **Wasmtime/wazero** capability sandbox for code compiled to Wasm. **Забрать:** confirms wazero is the Go-idiomatic sandbox *when you control the compile target* — which we don't for arbitrary python.

**Dominant patterns:** (1) containers are the CI floor (runc namespaces); (2) for a worker spawning interpreters, **nsjail / `unshare` per subprocess** is the proven answer (Windmill); (3) **gVisor** is the syscall-level defense of choice for LLM-generated code (OpenAI, Modal, Google); (4) **Firecracker microVMs** are the gold standard for fully-hostile code (E2B/Lambda); (5) **WASM** only where you own the compile target. Recurring lesson: in-process language "sandboxes" (smolagents local, Pyodide/Terrarium) are **not** security boundaries.

---

## 4. Recommendation (ranked)

Each tier composes with the existing write-scope + `--allowed-programs` and can be gated behind config so the default stays safe but overridable.

### Tier 0 — quick win, ships now (pure-Go, portable, unprivileged)

Add a **re-exec child launcher** to `RunBlock` (idiomatic Go, the runc pattern: daemon re-execs `/proc/self/exe` with a hidden subcommand; the child sets its own limits then `exec`s the interpreter). In that child, before `exec`:

1. **Empty net namespace** — `SysProcAttr.Cloneflags |= CLONE_NEWNET` (+ `CLONE_NEWUSER` with uid/gid maps so it works unprivileged / on bare-metal non-root). This is the single strongest cheap win: **no network at all** (loopback only), off by default, per-role opt-in for network.
   - *Isolates:* net. *Effort:* low. *Fly/Alpine/no-CGO:* all yes (root-in-VM on Fly; userns on bare-metal).
2. **rlimits** via `unix.Prlimit` in the child: `RLIMIT_CPU` (backstop the wall-clock timeout against fork-bomb children), `RLIMIT_FSIZE`, `RLIMIT_NOFILE`, `RLIMIT_NPROC`, `RLIMIT_AS` (coarse mem guard). `GOOS`-gated (Unix only).
   - *Isolates:* cpu/mem-coarse/pids/fds/disk-write. *Effort:* low. *All-compat:* yes.
3. **`go-landlock` filesystem confinement** with `BestEffort()`: `RODirs("/usr","/bin","/lib", interpreter dirs)`, `RWDirs(workDir)`. Degrades to no-op on kernels without Landlock (covers the "Fly kernel Landlock unverified" risk).
   - *Isolates:* fs (workdir-only writes, read-only system). *Effort:* low-med. *Compat:* yes on 5.13+; BestEffort elsewhere.
4. **Drop to a dedicated unprivileged uid** (`SysProcAttr.Credential`) when the daemon runs as root, and set **`PR_SET_NO_NEW_PRIVS`** (hygiene + prerequisite for Tier-1 seccomp).

**Забрать:** the re-exec+namespace shape from runc/Windmill's `unshare` fallback; `go-landlock` and rlimit idioms straight from `process_isolation.md` §2–3.

### Tier 1 — operator-selectable hardening (self-hosted Linux)

Config option `--sandbox=nsjail` (or `bwrap`): wrap the interpreter argv in **`nsjail`** with a per-language config. One apk-installable binary gives namespaces + **real** cgroup-v2 mem/pids/cpu caps + rlimits + seccomp — the whole stack without hand-rolling. This is exactly Windmill's production posture. `bubblewrap` is the smaller alternative if you'd rather add cgroups via `systemd-run`.

- *Isolates:* fs/net/cpu/mem/pids/syscalls. *Effort:* med (config + detect binary + per-lang profiles). *Compat:* Alpine apk yes; Fly — namespaces/seccomp work, but **cgroup caps are unreliable on Fly's hybrid layout** (fine on bare-metal). So nsjail is primarily the **bare-metal** hardening tier.

Alternative if you refuse the external binary: extend the Tier-0 child with `elastic/go-seccomp-bpf` (syscall allowlist, applied in the child after `NO_NEW_PRIVS`) + a direct cgroup-v2 subtree writer (`memory.max`/`pids.max`). More code, same result, still no-CGO. Note the footgun from `process_isolation.md`: **seccomp and hard rlimits must be set in the child, never in the long-running Go daemon.**

### Tier 2 — untrusted role-authors / strongest available on our infra

**gVisor `runsc` (systrap platform).** Best strength/effort ratio and — critically — it runs on fly.io **without `/dev/kvm`** (systrap uses ptrace+seccomp; gVisor recommends it inside VMs). It's a static Go binary on Alpine, and **simplepanel already exposes runsc**, so the operational surface is known. Wrap the interpreter run as a `runsc`-executed OCI bundle for roles flagged untrusted-author.

- *Isolates:* all (userspace kernel). *Effort:* med-high. *Compat:* Fly yes (systrap), Alpine yes (static binary). Accept ~10–30% syscall/IO overhead.

**Firecracker/Kata microVM** = true kernel isolation, but **not possible on fly.io** (no nested KVM). Bare-metal-only; only pursue if a hostile-multi-tenant requirement appears. For that case, borrow **E2B's** call-an-external-sandbox architecture (and Temporal's durable-wrapper pattern).

### Single best default for our reality

**Tier 0 as the always-on default**, because it is the only tier that is simultaneously pure-Go/`CGO_ENABLED=0`, unprivileged, works identically on fly.io Firecracker VMs and bare-metal Alpine, needs no external binary, and needs no writable cgroup layout. Its default posture — **no network + read-only-FS-except-workdir + rlimits + dropped uid** — closes the biggest holes (exfiltration, host-file reads, resource exhaustion) at near-zero operational cost. Layer nsjail (bare-metal) or gVisor-systrap (untrusted authors, already available) on top by config.

---

## 5. Integration sketch for `coderun.go`

The wrapping goes **inside `RunBlock`**, at the point where `cmd` is constructed (coderun.go:170), because `RunBlock` is the sole exec choke point (both `RunCode` and the LLM `exec` tool reach it). Nothing above it changes; the write-scope in `RunCode` and `--allowed-programs` in `handler.go` stay exactly as-is and compose cleanly — sandboxing constrains the *process's side effects*, the write-scope constrains *persisted changes*, `--allowed-programs` constrains *which binary runs*. Three independent gates.

Shape (Tier 0, self-launcher):

```go
// coderun.go — new file sandbox_linux.go (+ sandbox_other.go no-op for !linux)

// applySandbox mutates cmd for the current SandboxPolicy just before Run().
// On non-Linux it is a no-op (GOOS build tags).
func applySandbox(cmd *exec.Cmd, workDir string, p SandboxPolicy) error {
    if p.Mode == SandboxOff {
        return nil
    }
    // Re-exec self as the confined child; the child sets rlimits + landlock
    // + NO_NEW_PRIVS after clone, before exec of the real interpreter.
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Cloneflags:  syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET | // net off
                     syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
        UidMappings: idMap(), GidMappings: idMap(),
        // Credential: &syscall.Credential{Uid: sandboxUID} when daemon is root
    }
    if p.Network { cmd.SysProcAttr.Cloneflags &^= syscall.CLONE_NEWNET }
    return nil // rlimits + landlock applied in the re-exec child stub
}
```

- **CodeSpec** gains a `Sandbox SandboxPolicy` field; `CodeInput`/`fleet.Config` expose it (CLI: `--sandbox=off|native|nsjail|runsc`, `--sandbox-network` opt-in, rlimit knobs). Default = `native` (Tier 0).
- **Re-exec child stub:** a hidden `fleet __coderun-child` subcommand that (1) `unix.Setrlimit(...)` each limit, (2) `landlock.V3.BestEffort().RestrictPaths(RODirs(sysDirs...), RWDirs(workDir))`, (3) `unix.Prctl(PR_SET_NO_NEW_PRIVS,1,0,0,0)`, (4) `syscall.Exec(interp, argv, env)`. Env stays the scrubbed `buildChildEnv` output; `cmd.Dir` stays `workDir`.
- **nsjail/runsc modes:** instead of self-re-exec, prepend `nsjail -C <lang>.cfg --` or run via `runsc` OCI bundle; the interpreter argv and scrubbed env are unchanged.
- **Default safe posture** shipped: `SandboxNative` = net-off + landlock RW-workdir-only + rlimits + `NO_NEW_PRIVS` + drop-uid. Network and extra paths are per-role opt-ins.

A single `RunBlock` test matrix (net-denied, write-outside-workdir-denied, CPU-limit-trips) verifies the policy; keep the `GOOS`-gated files so non-Linux dev builds compile with a no-op sandbox.

---

## 6. Open questions

1. **Does Fly's custom 5.15 kernel compile in Landlock** (`CONFIG_SECURITY_LANDLOCK`, `lsm=landlock,...`)? Undocumented — probe at boot (`landlock.V1.RestrictPaths()` on a temp dir) and log the ABI; rely on `BestEffort()` so a Landlock-less kernel degrades instead of erroring.
2. **Unprivileged userns on bare-metal targets** — some distros gate `kernel.unprivileged_userns_clone`. If disabled, Tier-0 namespaces need either the sysctl flipped or the daemon running as root (then use `Credential` to drop). Document the sysctl in deploy notes.
3. **cgroup mem/pids caps on Fly** — Tier 0 uses only `RLIMIT_AS` (coarse). If a real RSS cap is needed on Fly, test a direct v2-subtree writer against the hybrid layout; do **not** use `cgroup_no_v1=all` (crashes the VM).
4. **MPTCP + Landlock (future):** if we ever adopt Landlock network rules (needs kernel 6.7, not Fly's 5.15), Go 1.24's default MPTCP listeners bypass Landlock's TCP rules — disable MPTCP or stick to net-namespace for network-off.
5. **gVisor syscall-compat gaps** for the heavier interpreters (node native addons, python C-extensions doing exotic syscalls) — validate the target workloads under `runsc` before making it a role default.
6. **Per-language nsjail/seccomp profiles** — python vs bash vs node need different allowlists; borrow and adapt Windmill's committed nsjail configs rather than authoring from scratch.

---

## Sources

Deployment (fly/Alpine/no-CGO):
- [Fly default kernel version (5.15.x) — community](https://community.fly.io/t/updated-default-kernel-version/13786), [Fly infra log](https://fly.io/infra-log/)
- [Fly Machines run as root-in-VM — community](https://community.fly.io/t/why-are-fly-machines-running-as-firecracker-microvms-configured-with-privileged-true/26380)
- [Podman & gVisor on Fly (no KVM, systrap, hybrid cgroups) — community](https://community.fly.io/t/podman-and-gvisor/26529), [Fly sandboxing blog](https://fly.io/blog/sandboxing-and-workload-isolation/)
- [gVisor platforms (systrap in VMs)](https://gvisor.dev/docs/architecture_guide/platforms/), [gVisor install](https://gvisor.dev/docs/user_guide/install/)
- [bubblewrap (Alpine main)](https://pkgs.alpinelinux.org/package/edge/main/x86/bubblewrap), [nsjail (Alpine community)](https://pkgs.alpinelinux.org/package/v3.22/community/x86/nsjail), [util-linux-misc/prlimit (Alpine main)](https://pkgs.alpinelinux.org/package/edge/main/x86/util-linux-misc), [GVisor on Alpine wiki](https://wiki.alpinelinux.org/wiki/GVisor)
- [go-landlock source (CGO-free)](https://github.com/landlock-lsm/go-landlock/blob/main/landlock/restrict.go), [go-landlock pkg.go.dev](https://pkg.go.dev/github.com/landlock-lsm/go-landlock/landlock), [elastic/go-seccomp-bpf (no libseccomp)](https://github.com/elastic/go-seccomp-bpf/blob/main/README.md)
- [Alpine aports MR !31556 — Landlock in linux-lts](https://gitlab.alpinelinux.org/alpine/aports/-/merge_requests/31556), [landlock.io distro support](https://landlock.io/), [Landlock kernel docs](https://docs.kernel.org/userspace-api/landlock.html)
- WASM/Python: [wasi-wheels numpy/dlopen blocker](https://github.com/dicej/wasi-wheels/issues/4), [python-wazero POC](https://github.com/rgl/python-wazero-poc), [pyodide](https://github.com/pyodide/pyodide), [benbrandt/wasi-wheels](https://github.com/benbrandt/wasi-wheels)
- [cgroup v2 kernel docs](https://docs.kernel.org/admin-guide/cgroup-v2.html)

How popular Go projects do it:
- [Windmill security/isolation (nsjail + unshare)](https://www.windmill.dev/docs/advanced/security_isolation), [windmill repo](https://github.com/windmill-labs/windmill)
- [google/nsjail](https://github.com/google/nsjail), [kCTF](https://google.github.io/kctf/introduction.html)
- [gVisor](https://gvisor.dev/), [Google open-sourcing gVisor](https://cloud.google.com/blog/products/identity-security/open-sourcing-gvisor-a-sandboxed-container-runtime), [GKE Sandbox](https://docs.cloud.google.com/kubernetes-engine/docs/concepts/sandbox-pods)
- [OpenAI code-execution uses gVisor](https://itnext.io/openais-code-execution-runtime-replicating-sandboxing-infrastructure-a2574e22dc3c), [Modal sandboxes (gVisor)](https://modal.com/resources/best-code-execution-sandboxes-ai-agents)
- [E2B (Firecracker)](https://e2b.dev/), [E2B Firecracker vs QEMU](https://e2b.dev/blog/firecracker-vs-qemu)
- [smolagents secure execution (local ≠ sandbox)](https://huggingface.co/docs/smolagents/tutorials/secure_code_execution), [Cohere Terrarium/Pyodide CVE](https://thehackernews.com/2026/04/cohere-ai-terrarium-sandbox-flaw.html)
- [Dagger BuildKit engine](https://pkg.go.dev/github.com/dagger/dagger/engine/buildkit), [Coder envbox/sysbox](https://github.com/coder/envbox), [nestybox/sysbox](https://github.com/nestybox/sysbox)
- [Gitea act_runner (docker.sock risk)](https://gitea.com/gitea/runner/issues/167), [nektos/act runners](https://nektosact.com/usage/runners.html), [Woodpecker CI](https://github.com/woodpecker-ci/woodpecker)
- [Temporal Python workflow sandbox (determinism, not security)](https://python.temporal.io/temporalio.worker.workflow_sandbox.html), [Temporal agent-sandbox orchestration](https://temporal.io/blog/temporal-sandbox-orchestration-harness-the-missing-layer-for-running-agents)
- [Fermyon Spin v3 (Wasmtime)](https://www.fermyon.com/blog/introducing-spin-v3), [wazero users](https://wazero.io/community/users/), [wazero hardening](https://www.systemshardening.com/articles/wasm/wazero-hardening/)

_Compiled 2026-07-02. Verify kernel/library/package versions against the actual deploy box before relying on a given ABI/feature. Companion doc: [`process_isolation.md`](./process_isolation.md)._
