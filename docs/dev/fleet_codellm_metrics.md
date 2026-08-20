# fleet & codellm metrics

Both binaries serve Prometheus on a second, loopback-only listener, mirroring
the monolith's internal listener (`cmd/server`, `127.0.0.1:8082`): `/metrics`,
`/debug/pprof/*`, `/healthz`, `/livez`, `/readyz`. Nothing on that port is
authenticated — never bind it off-box.

| Binary | Flag | Env | Default |
|--------|------|-----|---------|
| codellm | `--metrics-addr` | `CODELLM_METRICS_ADDR` | `127.0.0.1:18087` |
| fleet | `--metrics-addr` | `TRIP2G_FLEET_METRICS_ADDR` | `127.0.0.1:19090` |

Empty disables the listener; every record call site is nil-safe, so a disabled
listener costs nothing. The ports follow the deployment convention of a
`1`-prefixed twin of the service port (`infra/site.yml`: `8087` → `19087`).

Collectors live in `cmd/codellm/internal/codellmmetrics` and
`cmd/fleet/internal/fleetmetrics`. Each owns a **private registry** rather than
the global default, so neither binary depends on the monolith's
`internal/metrics` (which registers globally) and tests can gather a clean
snapshot.

## Who spends tokens

codellm spends none. It is a fake LLM: it executes fenced code and always
answers with `usage: {}`. Token spend is fleet's, recorded in
`agentruntime` from the provider's usage — which is why `fleet_llm_tokens_total`
is a fleet metric and a codellm-backed fleet reports zero on it.

## codellm

**Requests**

| Metric | Notes |
|---|---|
| `codellm_requests_total{endpoint,status}` | endpoint = `chat_completions` \| `graphql` \| `graphql_playground` |
| `codellm_request_duration_seconds{endpoint}` | |
| `codellm_requests_in_flight` | each in-flight request may fork an interpreter child |
| `codellm_auth_total{lane,result}` | lane = `apikey` \| `cookie`. Denials on an endpoint that executes code are a probing signal |

**Execution**

| Metric | Notes |
|---|---|
| `codellm_blocks_total{program,outcome}` | outcome = `ok` \| `nonzero_exit` \| `timeout` \| `start_failed` |
| `codellm_block_exit_codes_total{program,exit_code}` | `-1` = the child never produced an exit status |
| `codellm_block_duration_seconds{program}` | |
| `codellm_block_max_rss_bytes{program}` | from the child's rusage |
| `codellm_block_stdout_bytes{program}` | only the block whose stdout is buffered reports a size — in a pipeline the intermediate blocks stream into the next block's stdin |
| `codellm_block_stdout_truncated_total{program}` | stdout hit the cap and the overflow was dropped; downstream this reads as a parse error |
| `codellm_sandbox_fallbacks_total{reason}` | a `besteffort` policy degraded to unsandboxed execution. Enforcing mode refuses instead, and shows up as `codellm_exec_errors_total{kind="sandbox_refused"}` |
| `codellm_exec_errors_total{kind}` | `no_blocks`, `unknown_fence`, `disallowed_program`, `sandbox_refused`, `setup_failed`, `start_failed`, `timeout`, `nonzero_exit`, `parse_error` |
| `codellm_request_blocks` | blocks executed per request |
| `codellm_changes_total{kind}` | `write` \| `patch` |
| `codellm_config_info{sandbox_mode,sandbox_network,allowed_programs}` | always 1; the posture this process runs with |

`coderun` records nothing itself. It measures and reports through
`CodeInput.Observe` (`coderun.BlockStats`) and classifies failures with
`coderun.ExecError` / `coderun.ErrorKind`; the codellm layer decides what to
count. That keeps the execution core free of a metrics dependency.

## fleet

**Deliveries**

| Metric | Notes |
|---|---|
| `fleet_deliveries_total{role,kind,status}` | kind = `change` \| `cron` |
| `fleet_delivery_duration_seconds{role,kind}` | includes the agent run |
| `fleet_delivery_auth_failures_total{reason}` | `unknown_key` \| `bad_signature` \| `bad_payload` \| `read_body` — a rotated secret or a stale reconcile shows up here |
| `fleet_runs_in_flight` | deliveries are handled synchronously, so this is also the drain backlog |
| `fleet_fanout_items{role}` / `fleet_fanout_item_errors_total{role}` | trip2g records a partial batch as success, so the error counter is the only signal a `for_each` item failed |

**Runs**

| Metric | Notes |
|---|---|
| `fleet_runs_total{role,status}` | `completed` \| `capped` \| `max_steps` \| `error` |
| `fleet_run_steps{role}` / `fleet_run_duration_seconds{role}` | drift toward the step ceiling precedes hitting it |
| `fleet_llm_tokens_total{model,role,kind}` | one series answers both cost-per-model and cost-per-role |
| `fleet_tool_calls_total{tool,outcome}` | `ok` \| `denied` \| `invalid_args` \| `error` \| `apply_failed` \| `not_permitted` |
| `fleet_denials_total{kind}` | `read` \| `write` \| `not_permitted`; a steady rate usually means misconfigured patterns |
| `fleet_apply_failures_total{role,tool}` | under `HardFailApply` each one kills its run |

**Upstream LLM**

| Metric | Notes |
|---|---|
| `fleet_llm_requests_total{lane,model,status}` | lane = `llm` (role's model) \| `exec` (codellm), so a degrading code executor stays distinguishable |
| `fleet_llm_request_duration_seconds{lane,model}` | retries included |
| `fleet_llm_retries_total{lane,reason}` | `429` \| `5xx` \| `network` |

**Control plane**

| Metric | Notes |
|---|---|
| `fleet_syncs_total{status}` + `fleet_sync_duration_seconds` | one discovery+reconcile cycle |
| `fleet_last_successful_sync_timestamp_seconds` | only a fully clean cycle advances it |
| `fleet_roles_registered` / `fleet_roles_skipped_total` | |
| `fleet_roles_write_scope_misconfigured` | roles declaring write tools with no `write_patterns` — the deny-all trap, as a gauge instead of one warning at startup |
| `fleet_webhook_actions_total{action,status}` / `fleet_webhooks_owned{kind}` | a nonzero steady-state create/update rate means two fleets share a `fleet_id` and are re-pointing each other's webhooks |
| `fleet_config_info{fleet_id,default_model,exec_enabled}` | always 1 |

`/readyz` reports ready once the **first sync attempt completes**, not once it
succeeds. Gating on success would make the fleet's readiness cascade from a
trip2g outage and invite restart loops; staleness is what
`fleet_last_successful_sync_timestamp_seconds` is for.

## What to alert on

1. `fleet_last_successful_sync_timestamp_seconds` older than ~3× the poll
   interval — the failure mode that hides all the others: the fleet keeps
   serving a stale registry and otherwise looks healthy.
2. `rate(fleet_runs_total{status="capped"})` — money burning.
3. `rate(fleet_llm_retries_total)` — upstream degrading, on either lane.
4. `codellm_requests_total{status="422"}` as a share of all requests — the exec
   path is broken; split by `codellm_exec_errors_total{kind}` to see how.
5. `increase(codellm_sandbox_fallbacks_total)` > 0 — the security posture
   silently degraded to unsandboxed execution.
6. `fleet_runs_in_flight` / `codellm_requests_in_flight` growing without
   draining — a stuck run or a fork pile-up.

## Cardinality

`role` is the role note path, bounded by the number of role notes in the agents
folder. If that ever grows large, drop `role` from the histograms
(`fleet_run_steps`, `fleet_run_duration_seconds`, `fleet_delivery_duration_seconds`)
and keep it on the counters.
