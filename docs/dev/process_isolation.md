# Process isolation for the fleet code-executor

> **TL;DR / вывод.** The fleet code-executor already does the cheap, portable basics right (scrubbed env, temp cwd, context timeout, stdout cap). The next gains are tiered:
>
> 1. **Now, cheap + (mostly) portable** — add POSIX **rlimits** (`RLIMIT_CPU`/`FSIZE`/`NOFILE`/`NPROC`, `AS` as a coarse guard) and **network-off-by-default** via a Linux net namespace, plus run the child under a dedicated unprivileged uid. Go-native, no daemon, no root (with user namespaces).
> 2. **Self-hosted-Linux hardening** — wrap the allowlisted program in **`nsjail`** (or `bubblewrap`): one binary that bundles namespaces + cgroups v2 + rlimits + seccomp. Equivalent hand-rolled stack: `SysProcAttr` namespaces + `containerd/cgroups` + `elastic/go-seccomp-bpf` + `go-landlock`, applied in a **thin re-exec child launcher** (never in the daemon).
> 3. **Untrusted role-authors** — **gVisor (`runsc`)** for the best strength/effort ratio (user-space kernel, no VM, OCI drop-in), or a **microVM (Firecracker / Kata)** for true kernel-level isolation. If role code can be compiled to Wasm, **wazero** is the strongest *and* cheapest option (in-process capability sandbox, pure Go, no root).
>
> Two load-bearing facts for Go: **namespaces are first-class** in `syscall.SysProcAttr` (free isolation, no external binary), but **seccomp and tight rlimits do NOT compose with the multithreaded Go parent** — they must be applied in the child just before `exec`, not in the long-running daemon.

---

## 1. Baseline (already shipped) and threat model

The executor spawns operator-allowlisted programs with `exec.CommandContext` on the role's rendered code. Already in place:

- **Secret-scrubbed minimal env** — `PATH` + `FLEET_INPUT` + only role-declared `env_passthrough` / `env_prefix`.
- **Per-run throwaway temp workdir** (`os.MkdirTemp` → `Cmd.Dir`, deleted after).
- **Role timeout** (`exec.CommandContext` cancellation).
- **stdout byte cap** (limited writer).
- **Off by default** (`--allowed-programs`).

**Threat model.** Self-hosted, typically Linux. Role *notes* are operator-trusted, but the *code* they render can be adversarial or prompt-injected. So the boundary we care about is: a hostile allowlisted-program invocation must not exfiltrate secrets, reach the network, escape the workdir, exhaust the host, or pivot to the host kernel/other tenants.

**What the baseline does *not* yet stop:** unbounded CPU/RAM/PIDs/disk inside the timeout window, network access, reading anything the executor's uid can read (the temp cwd is not a security boundary — the child can `chdir` away or use absolute paths), and dangerous syscalls.

---

## 2. Universal / Go-native layer (portable)

What `os/exec` + the standard library + `golang.org/x/sys/unix` give you **without any external tool or root**. This is the cheap layer to maximize first.

### Already portable and shipped
- **Minimal env** (`Cmd.Env`), **context timeout** (`exec.CommandContext`), **temp cwd** (`Cmd.Dir`), **output caps** (wrap the reader/`io.LimitReader`/limited writer). All cross-platform (Linux/macOS/Windows).
- **Working-dir scoping** is *not* a confinement boundary on its own — it only sets the initial cwd. Real fs confinement needs Landlock / mount-ns / chroot (below).

### Resource limits — `syscall.Setrlimit` / `unix.Prlimit`
POSIX resource limits are the portable-ish way to cap a single process:

| rlimit | Caps | Notes / caveats |
|---|---|---|
| `RLIMIT_CPU` | CPU **seconds** | Backstop for the wall-clock timeout (a fork-bomb child can dodge a `context` deadline). |
| `RLIMIT_AS` | **Virtual** address space | Coarse only — limits *virtual*, not resident, memory. Go and modern allocators reserve huge virtual ranges → false OOM. Use **cgroup `memory.max`** for a real RAM cap. |
| `RLIMIT_DATA` | Heap/data segment | Similar virtual-memory caveat. |
| `RLIMIT_FSIZE` | Max file **size** the process can create | Cheap disk-write guard. |
| `RLIMIT_NOFILE` | Open file descriptors | **Cached by the Go runtime** and restored for children before `exec` (see Go #66797/#67184). |
| `RLIMIT_NPROC` | Processes **per real uid** | Anti-fork-bomb, but it's a *global per-uid* count, not per-sandbox → fragile if the uid is shared. Prefer the **pids cgroup**. |
| `RLIMIT_CORE` / `RLIMIT_STACK` | Core dumps / stack | Minor hardening. |

**Portability.** `Setrlimit`/`Getrlimit`/`Prlimit` exist on **Linux, macOS, BSD** via `x/sys/unix`; there is **no Windows or plan9 equivalent** — gate the code behind `GOOS`/build tags. Behavior also differs per-OS: macOS historically clamped `RLIMIT_NOFILE` and ignores some limits; `RLIMIT_AS` enforcement is weak/absent on Darwin; `RLIMIT_CPU` soft-limit semantics are Linux-specific. (Go `syscall/rlimit.go`; golang/go #30401, #5949.)

**The Go footgun — limiting the *child only*.** `syscall.Setrlimit` applies to the **calling process**, and limits are **inherited across `fork`/`exec`**. `SysProcAttr` has **no rlimit field**. So:

- Calling `Setrlimit` in the daemon limits the **daemon too** (bad — it's long-running).
- Cleanest options:
  1. **`prlimit(1)` wrapper** as argv[0]: `prlimit --cpu=10 --as=... --nofile=64 --nproc=16 -- <program> <args>`.
  2. **Re-exec helper pattern** (idiomatic Go, what runc does): the daemon re-execs `/proc/self/exe` with a hidden subcommand; the child sets its own rlimits via `unix.Setrlimit`, then `exec`s the real program. This is also where you apply seccomp/Landlock/NO_NEW_PRIVS (see §5).
  3. `unix.Prlimit(childPID, …)` from the parent after `Start()` — racy (window between exec and the call) and needs `CAP_SYS_RESOURCE` when uids differ.

### Network-off, the portable-on-Linux way
There is **no cross-platform "no network" switch** in `os/exec`. On Linux it is one flag (§4, net namespace). On macOS/Windows you'd need a higher-level sandbox or firewall. So "network off by default" is effectively a **Linux** feature for the fleet.

**Bottom line for this layer:** rlimits (via a `prlimit` wrapper or re-exec helper) and dropping to an unprivileged uid (`SysProcAttr.Credential`) are the only genuinely portable hardening steps. Real memory/PID/network/filesystem confinement is Linux-specific.

---

## 3. Linux-specific primitives — what each one actually restricts

| Primitive | Restricts | Go integration | Root needed? |
|---|---|---|---|
| **Namespaces** (`CLONE_NEW*`) | Process/host visibility, mounts, network, IPC, hostname, uids | **First-class**: `syscall.SysProcAttr.{Cloneflags,Unshareflags,UidMappings,GidMappings,Credential,Chroot}` | No, if user namespaces are enabled |
| **cgroups v2** | **CPU, real memory, PIDs, IO** quotas | `containerd/cgroups/v2`, or write `cpu.max`/`memory.max`/`pids.max` + add PID to `cgroup.procs` | Delegated cgroup (systemd) or root |
| **seccomp-bpf** | **Syscall** allow/deny list | `elastic/go-seccomp-bpf` (pure Go, no cgo) or `seccomp/libseccomp-golang` (cgo) | No (needs `NO_NEW_PRIVS` first) |
| **Landlock LSM** | **Filesystem** access; TCP bind/connect (ABI v4+) | `landlock-lsm/go-landlock` with `BestEffort()` | **No** (unprivileged by design) |
| **no_new_privs** | Blocks privilege gain via setuid/setcap exec | `unix.Prctl(PR_SET_NO_NEW_PRIVS,1,…)` in the child | No |
| **Capability dropping** | Ambient kernel capabilities | Best: run as unprivileged uid (`Credential`); `SysProcAttr.AmbientCaps` adds, libcap drops | No |
| **pivot_root / chroot** | Filesystem **view** | `SysProcAttr.Chroot`; `pivot_root` via mount ns | chroot needs root; pivot_root works inside a user+mount ns |

### Namespaces (the cheap structural isolation)
- **net** (`CLONE_NEWNET`): a fresh net namespace with **no veth = no connectivity at all** (loopback only). The simplest, strongest "network off" — and it's just a flag in `SysProcAttr.Cloneflags`.
- **pid** (`CLONE_NEWPID`): child can't see or signal host processes; remount `/proc` to hide them.
- **mount** (`CLONE_NEWNS`): private mount table; pair with `pivot_root` onto a minimal rootfs for real fs confinement.
- **user** (`CLONE_NEWUSER` + Uid/Gid mappings): maps "root-in-namespace" to an unprivileged host uid. This is what makes **all of the above usable without real root** (and what bubblewrap/rootless-podman rely on). Some distros gate unprivileged userns (`kernel.unprivileged_userns_clone`).
- **ipc / uts**: isolate SysV/POSIX IPC and hostname.

Go can unshare *all* of these at spawn time natively — no external binary — which is the standout "surprisingly easy in Go" finding.

### cgroups v2 (the real quota layer)
Unified hierarchy. The controllers that matter: `cpu.max` (CPU quota), `memory.max` + `memory.high` (**real RSS** cap with OOM-kill — the correct memory limit, unlike `RLIMIT_AS`), `pids.max` (process count — the correct anti-fork-bomb), `io.max` (disk throughput/IOPS). Needs either root or a systemd-delegated cgroup subtree.

### seccomp-bpf (syscall attack-surface reduction)
Allowlist/denylist of syscalls; a denied call gets `SIGSYS`/`EPERM`. Blocks the dangerous escape primitives (`mount`, `ptrace`, `keyctl`, `bpf`, `clone` of new user ns, `kexec`, etc.). `elastic/go-seccomp-bpf` is pure Go (no libseccomp), calls `PR_SET_NO_NEW_PRIVS` for you, and uses `SECCOMP_FILTER_FLAG_TSYNC` to sync across the Go runtime's threads. **Footgun:** installing a filter in the daemon filters the *whole Go runtime* — apply it only in the thin child just before `exec`.

### Landlock LSM (unprivileged filesystem confinement)
The most attractive Linux-native primitive for this use case: **no root**, stackable, no unmount needed. `go-landlock` with `BestEffort()` degrades gracefully on older kernels.

- Kernel **5.13+** = filesystem rules; **6.7+** (ABI v4) = TCP **bind/connect**; later ABIs add ioctl-on-device, IPC scoping, audit, UNIX-socket path rules.
- **Limits:** can restrict open/read/write/exec/create/remove/rename/truncate; **cannot** restrict `chdir`, `stat`, `chmod`/`chown`, `setxattr`, `fcntl`, `access`. Network is **TCP only** (no UDP).
- **Go gotcha:** Landlock only covers "classic" TCP, not Multipath TCP — and since **Go 1.24 `net.Listen()` defaults to MPTCP**, a Go listener can't be restricted by Landlock unless you disable MPTCP.

Minimal usage:
```go
err := landlock.V4.BestEffort().RestrictPaths(
    landlock.RODirs("/usr", "/bin"),
    landlock.RWDirs(workdir),
)
```

---

## 4. Higher-level sandboxes

| Sandbox | Isolation strength | Enforces | Go integration | Overhead / cold-start | Root / setup |
|---|---|---|---|---|---|
| **bubblewrap** (`bwrap`) | Medium (ns + caller-supplied seccomp) | namespaces, bind-mount rootfs, seccomp (you provide BPF); **no** cgroups/rlimits built in | **Spawn the binary** | Near-zero (just a process), ms start | Unprivileged (user ns); used by Flatpak |
| **nsjail** (Google) | Medium-high (purpose-built for untrusted code) | namespaces **+ cgroups + rlimits + seccomp (Kafel)** in one tool | **Spawn the binary** (config or flags) | Near-zero, ms start | Unprivileged or privileged modes; single binary |
| **firejail** | Medium, but **setuid-root** = added attack surface (LPE CVE history) | namespaces, seccomp, AppArmor; large desktop profile library | Spawn the binary | Low | **setuid root** — *not recommended for server untrusted-code* |
| **runc / podman** | Low-medium (**shared host kernel**) | cgroups + namespaces + seccomp + caps; OCI | Spawn / OCI libs; podman rootless | ~100s of ms; image mgmt | Rootless possible; a kernel 0-day escapes |
| **gVisor** (`runsc`) | **High** — user-space kernel; app syscalls never reach host kernel | full ns/cgroup quotas + intercepted syscalls (Systrap, seccomp `SIGSYS`, since 2023) | `runsc` **binary** / OCI `RuntimeClass` (written in Go, *not* a library) | ~0 CPU; **I/O & syscall-heavy 10–30%**, up to ~125% on heavy DB; cold-start lower than a microVM (tens–hundreds ms) | Install runtime; some syscall-compat gaps |
| **Firecracker** microVM | **Very high** — true KVM VM, own guest kernel | everything (VM); host shielded by KVM | `firecracker-go-sdk` drives the VMM over its API socket | **~125 ms boot, <5 MiB** overhead | Needs `/dev/kvm` (bare-metal or nested virt) + kernel/rootfs images + TAP networking |
| **Kata Containers** | **Very high** (VM, container UX) | everything (VM) via QEMU/Cloud Hypervisor | OCI `RuntimeClass` (drop-in) | ~150–300 ms start, ~17% runtime overhead | Needs KVM |
| **WASM (wazero)** | **High for pure compute** — capability-based, in-process, memory-safe | no ambient authority: **no fs/net/env unless explicitly granted**; mem cap via `RuntimeConfig`/`ModuleConfig`; CPU via context deadline | **Pure-Go library, no cgo, no external process** | Lowest (in-process), fully portable | **No root, no extra process** — but role code must compile to Wasm (WASI p1) |

Notes:
- **bubblewrap vs nsjail:** bwrap is a minimal namespace+mount builder (you add cgroups/rlimits separately, e.g. via `systemd-run`); **nsjail bundles cgroups + rlimits + seccomp**, which is exactly the untrusted-code-execution shape the fleet needs. For a single-tool Linux answer, **nsjail is the better fit**; bubblewrap if you want the smallest, most-audited primitive.
- **gVisor** is the sweet spot when you want strong isolation without managing VM images/KVM: it presents as a normal OCI runtime, so it slots under Docker/containerd/k8s with a `RuntimeClass`. The cost is I/O latency and occasional syscall-compat gaps.
- **microVMs (Firecracker/Kata)** are the industry standard for *multi-tenant untrusted* code (Lambda, Fargate, E2B, Vercel Sandbox), but bring real operational weight: guest kernel + rootfs images, warm pools, and TAP/CNI networking (which can dominate the 125 ms boot at scale).

---

## 5. Tiered recommendation for the fleet

### Tier 1 — universal baseline to add now (cheap, ~portable)
On top of what's shipped:
1. **rlimits on the child** via a `prlimit(1)` wrapper or a re-exec helper: `RLIMIT_CPU` (backstop the timeout), `RLIMIT_FSIZE`, `RLIMIT_NOFILE`, `RLIMIT_NPROC`, and `RLIMIT_AS` as a coarse memory guard. Unix-only — gate behind `GOOS`.
2. **Network off by default** (Linux): `SysProcAttr.Cloneflags |= CLONE_NEWNET` (empty net ns). Pair with `CLONE_NEWUSER` + uid/gid mappings so it works without root. Make network an explicit per-role opt-in.
3. **Drop to a dedicated unprivileged uid** (`SysProcAttr.Credential`) if the daemon runs as root, so the child can't read the executor's secrets/files.
4. Set **`PR_SET_NO_NEW_PRIVS`** in the child (prerequisite for Tier 2 seccomp and good hygiene regardless).

Cheap, mostly Go-native; the only non-portable parts (rlimits, net ns) are cleanly `GOOS`-gated.

### Tier 2 — Linux hardening for the common self-hosted-Linux deploy
Pick **one** of:
- **Wrap the program in `nsjail`** (recommended) or `bubblewrap`. nsjail gives namespaces + cgroups v2 (real memory/PID/CPU caps) + rlimits + seccomp from a single binary built for running untrusted code. Lowest engineering cost, no daemon, unprivileged-capable.
- **Hand-roll in a thin re-exec child launcher**: `SysProcAttr` namespaces (pid/net/mount/user/ipc/uts) + `containerd/cgroups/v2` (`memory.max`/`pids.max`/`cpu.max`) + `elastic/go-seccomp-bpf` (syscall allowlist) + `go-landlock` (fs confinement + workdir-only writes). Apply all of it **in the child after fork, before exec** — never in the daemon.

Either path delivers: real RAM/PID/CPU caps (cgroups), no network unless granted (net ns), filesystem confined to the temp workdir + read-only system dirs (Landlock/mount-ns), and a syscall allowlist (seccomp).

### Tier 3 — strongest, when role-authors may be untrusted
- **gVisor (`runsc`)** — best strength/effort ratio: user-space kernel, no VM images, OCI drop-in. Accept the I/O overhead and occasional syscall-compat gaps.
- **Firecracker or Kata microVM** — when you need true kernel-level isolation / hostile multi-tenant. Requires `/dev/kvm` and image+network+warm-pool plumbing. Drive Firecracker from Go via `firecracker-go-sdk`; Kata via OCI `RuntimeClass`.
- **wazero (Wasm)** — if (and only if) the role's code can be compiled to Wasm: strongest *and* cheapest, fully portable, in-process, no root. Doesn't fit "run arbitrary operator-allowlisted native programs," so it's a parallel track for pure-compute roles, not a replacement for the native-program path.

### Cheap+portable vs needs-Linux+privileges, at a glance
- **Cheap + portable:** scrubbed env, timeout, temp cwd, output cap (done); rlimits (`GOOS`-gated); unprivileged uid; wazero (if Wasm).
- **Linux, no root:** user+net+mount+pid namespaces, Landlock, seccomp, bubblewrap/nsjail (with unprivileged userns), rootless podman.
- **Linux + privileges/setup:** cgroups v2 (delegation), gVisor (runtime install), Firecracker/Kata (`/dev/kvm` + images).

---

## 6. Two Go-specific findings worth remembering

- **Surprisingly easy in Go:** Linux **namespaces** are first-class via `syscall.SysProcAttr` — pid/net/mount/user isolation with **zero external binaries**. **wazero** gives a genuine capability sandbox as a *pure-Go library* (no cgo, no extra process). **go-landlock** is a clean, unprivileged fs/network confinement library with graceful kernel fallback.
- **Surprisingly hard / footguns in Go:** **seccomp and tight rlimits don't compose with the Go parent.** You cannot safely install a seccomp filter or a hard `RLIMIT` in the long-running, multithreaded Go daemon, and `SysProcAttr` has no seccomp/rlimit fields — the only correct place is a **thin child launcher** (`/proc/self/exe` re-exec, or `nsjail`/`bwrap`) that sets `NO_NEW_PRIVS` + seccomp + Landlock + rlimits and then `exec`s the allowlisted program. Also: **`RLIMIT_AS` ≠ real memory** (use cgroup `memory.max`), **`RLIMIT_NPROC` is per-uid global** (use the pids cgroup), and **Landlock can't restrict `net.Listen()`** because Go 1.24 defaults to MPTCP.

---

## Sources

- gVisor — [Performance Guide](https://gvisor.dev/docs/architecture_guide/performance/), [Releasing Systrap (2023-04-28)](https://gvisor.dev/blog/2023/04/28/systrap-release/), [Optimizing seccomp usage (2024-02-01)](https://gvisor.dev/blog/2024/02/01/seccomp/), [Platforms](https://gvisor.dev/docs/architecture_guide/platforms/)
- Kata vs gVisor vs Firecracker — [Northflank: Kata vs gVisor](https://northflank.com/blog/kata-containers-vs-gvisor), [Northflank: Kata vs Firecracker vs gVisor](https://northflank.com/blog/kata-containers-vs-firecracker-vs-gvisor), [KubeBlocks runC/Kata/gVisor DB benchmark](https://kubeblocks.io/blog/does-containerization-affect-the-performance-of-databases), [onidel (2025)](https://onidel.com/blog/gvisor-kata-firecracker-2025)
- Firecracker — [Northflank: What is AWS Firecracker](https://northflank.com/blog/what-is-aws-firecracker), [cloudrps: Firecracker microVMs explained](https://cloudrps.com/blog/firecracker-microvm-serverless-isolation/), [E2B: Firecracker vs QEMU](https://e2b.dev/blog/firecracker-vs-qemu)
- AI agent sandboxing landscape (2026) — [manveerc: How to sandbox AI agents in 2026](https://manveerc.substack.com/p/ai-agent-sandboxing-guide), [particula: SmolVM vs Firecracker vs Docker](https://particula.tech/blog/smolvm-vs-firecracker-sandbox-ai-generated-code)
- Landlock — [Kernel: Landlock LSM](https://docs.kernel.org/security/landlock.html), [Kernel: unprivileged access control](https://docs.kernel.org/userspace-api/landlock.html), [landlock.io](https://landlock.io/), [go-landlock pkg.go.dev](https://pkg.go.dev/github.com/landlock-lsm/go-landlock/landlock), [Sandboxing network tools with Landlock (2025-12-06)](https://domcyrus.github.io/systems-programming/security/linux/2025/12/06/landlock-sandboxing-network-tools.html), [LWN: Toward unprivileged sandboxing (2017)](https://lwn.net/Articles/731478/)
- bubblewrap / firejail — [containers/bubblewrap](https://github.com/containers/bubblewrap), [netblue30/firejail](https://github.com/netblue30/firejail), [Firejail blog: Sandbox Linux apps (2025-08-20)](https://firejail.wordpress.com/2025/08/20/how-to-sandbox-linux-apps-with-firejail-and-bubblewrap/), [ArchWiki: Bubblewrap](https://wiki.archlinux.org/title/Bubblewrap)
- nsjail — [google/nsjail](https://github.com/google/nsjail), [nsjail.dev](https://nsjail.dev/)
- cgroups v2 — [containerd/cgroups/v2 pkg.go.dev](https://pkg.go.dev/github.com/containerd/cgroups/v2), [Kernel: Control Group v2](https://docs.kernel.org/admin-guide/cgroup-v2.html)
- seccomp in Go — [elastic/go-seccomp-bpf](https://github.com/elastic/go-seccomp-bpf), [seccomp/libseccomp-golang](https://pkg.go.dev/github.com/seccomp/libseccomp-golang), [Kernel: seccomp_filter](https://www.kernel.org/doc/Documentation/prctl/seccomp_filter.txt)
- Go rlimits / exec — [Go src: syscall/rlimit.go](https://go.dev/src/syscall/rlimit.go), [golang/go #30401 (macOS Setrlimit)](https://github.com/golang/go/issues/30401), [golang/go #66797 / #67184 (NOFILE cache)](https://github.com/golang/go/issues/66797), [Go src: syscall/exec_linux.go](https://go.dev/src/syscall/exec_linux.go)
- wazero / Wasm — [wazero docs](https://wazero.io/docs/), [Wazero hardening for Go embedders](https://www.systemshardening.com/articles/wasm/wazero-hardening/), [eunomia: WASI & Component Model status (2025-02-16)](https://eunomia.dev/blog/2025/02/16/wasi-and-the-webassembly-component-model-current-status/)

_Reference compiled 2026-06-30. Verify kernel/library versions against the actual deploy box before relying on a given ABI/feature._
