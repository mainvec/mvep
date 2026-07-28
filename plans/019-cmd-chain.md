# Plan 019 — CmdChain batch command with optional atomicity

> **STATUS: DRAFT — NOT APPROVED FOR IMPLEMENTATION.**
> The design below captures decisions reached in review, but several open
> questions remain unresolved (see [Open Questions](#open-questions)). The task
> breakdown is provisional and exists to size the work, not to authorize it.
> Do not create the branch or begin implementation until this plan leaves draft.

- **Issue**: [#19](https://github.com/mainvec/mvep/issues/19) — feat(runtime/go): CmdChain batch command with optional atomicity (DRAFT)
- **Branch**: `feat/019-cmd-chain` (not yet created)
- **Target release**: undecided; not before `runtime/go/v0.9.0`
- **Depends on**: [#18](https://github.com/mainvec/mvep/issues/18) for request body limits

## Problem / Goal

Every MVEP command costs a round trip. Callers performing a sequence of related
operations pay latency proportional to the number of commands, and there is no
way to express that the operations belong together.

`CmdChain` is a runtime built-in command carrying an ordered list of commands.
The server executes them in sequence, stops at the first failure, and returns a
positional list of results. Atomicity is opt-in per chain and is supplied by an
application-level interceptor rather than by the runtime.

The interesting property is that this falls out of MVEP's command-as-type model
almost for free: a chain is just another command whose payload happens to be a
list of commands. AWS models batching the same way (`BatchWriteItem`,
`TransactWriteItems` are ordinary API operations), but per-service and
hand-written. Because MVEP collapses method identity into the request type, the
chain can be generic and heterogeneous instead.

## Goals

- Execute an ordered list of commands from a single request.
- Return one result entry per input command, correlated by position.
- Stop at the first failure and report the remainder as skipped.
- Allow each inner command to carry its own `auth` header, validated
  independently.
- Keep inner payloads readable in JSON rather than base64-nested.
- Support opt-in atomicity, where a failure rolls back the whole chain, without
  introducing any transaction concept into the runtime.
- Require no spec change, no codegen change, and no toolkit release.

## Non-goals

- Result forwarding between commands. A chain is a batch, not a pipeline.
  OData, Microsoft Graph, and Salesforce Composite all grew reference syntax
  eventually; if that demand arrives, the positional result list gives a natural
  addressing scheme, but it is out of scope here.
- Parallel execution. Sequence is the only ordering guarantee offered.
- Nested chains. Depth is fixed at one.
- Cross-package chains. The command endpoint is per-package.
- Distributed atomicity. Sagas and compensating commands are a separate design,
  required only when the per-topic pub/sub transport exists.
- Two-phase commit or XA.
- A runtime-owned transaction API.
- Protobuf encoder support in v1.
- TypeScript runtime support in v1.

## Proposed Design

### Placement: a runtime built-in, not a generated command

Chain execution needs the dispatcher itself. The generated
`PkgCommandRunner.RunCmd` type switch holds only typed handler funcs and has no
reference to the `PackageHandler`, so a generated chain command could not
dispatch its own children. `PackageHandler` already holds the `Package`, the
`CommandRunner`, and the `Interceptor`.

Handling the chain inside `ServeCmdReq` therefore requires no spec change, no
codegen change, and no toolkit release, and works identically for every existing
package. This would establish the first reserved command name in MVEP; none
exist today.

### The load-bearing constraint

Interceptors run on the raw `*CmdReq` **before** decode:

```
ServeHTTP -> ServeCmdReq (interceptors) -> executeCmd (InstanceOf -> decode -> RunCmd)
```

The chain executor must loop calling **`h.ServeCmdReq`**, never `h.executeCmd`.
Calling `executeCmd` directly would let any caller bypass `AuthInterceptor`
combined with `OnlyCommands` by wrapping a privileged command inside a chain —
a privilege-escalation hole. Recursing through `ServeCmdReq` closes it, and has
two further benefits: rate-limiting interceptors count inner commands rather than
seeing one request, and per-command auth is genuinely evaluated per command.

The branch belongs in `ServeCmdReq`'s core-handler closure, so that outer
interceptors still wrap the chain itself while inner recursion re-applies them
per command.

### Types

Hand-written in a new `runtime/go/mvep/chain.go`:

```go
const CmdChainName = "CmdChain"

type RawPayload []byte // verbatim passthrough marshalers => inline JSON

type ChainEntry struct {
    Cmd     string
    Headers map[string]string
    Payload RawPayload
}

type CmdChain struct {
    Cmds   []ChainEntry
    Atomic bool
}

type ChainEntryStatus string // "ok" | "error" | "skipped" | "rolled_back"

type ChainEntryResult struct {
    Cmd     string
    Status  ChainEntryStatus
    Headers map[string]string
    Payload RawPayload
    Error   *ErrorInfo
}

type CmdChainResult struct {
    Results []ChainEntryResult
}
```

`RawPayload` is the mechanism for inline payloads. Its `MarshalJSON` and
`UnmarshalJSON` pass bytes through verbatim, so a chain serializes as
`"payload":{...}` rather than a base64 string, using one generic type and no
per-encoder branching in the chain logic. CBOR needs equivalent methods. The
protobuf encoder expects a `proto.Message` and cannot carry this type, so
v1 maintains an explicit encoder allowlist and returns
`chain_unsupported_encoder` otherwise.

### Execution and guards

In order:

1. Decode the outer payload into `CmdChain`.
2. Empty list → `empty_chain`.
3. Length over `MaxChainLength` → `chain_too_long`.
4. Any entry naming `CmdChain` → `nested_chain_forbidden`.
5. `Atomic` requested without transaction support → `atomic_not_supported`.
6. Apply `ChainTimeout` via `context.WithTimeout`.
7. Per entry: check `ctx.Err()`; copy the outer header map into a fresh map;
   overlay the entry's headers, including `auth`; build the inner `*CmdReq`;
   call `h.ServeCmdReq` recursively.
8. On failure: record the error, mark the remainder `skipped`, and if the chain
   is atomic relabel prior successes `rolled_back`; break.

The executor must never return a nil `*CmdResp`, because `RequestIDInterceptor`
dereferences the response unguarded. It must not alias caller-supplied maps.

### Atomicity without a runtime transaction API

The runtime cannot roll back arbitrary handler side effects. Atomicity is only
ever as strong as handler participation in a shared transactional resource. So
the runtime exposes exactly one seam:

```go
func ContextWithAtomicChain(ctx context.Context) context.Context
func IsAtomicChain(ctx context.Context) bool
```

An application-supplied `TransactionInterceptor` begins a transaction, sets the
marker, and lets the existing recursion carry the context to every inner
command. On return it inspects `resp.Error` and commits or rolls back. Fail-fast
already guarantees the chain's response carries an error when any inner command
failed, so **the chain executor needs no transaction awareness at all** beyond
the marker.

The interceptor must be registered with the existing `OnlyCommands(CmdChainName)`
helper, so that the per-inner-command recursion does not open a nested
transaction for each child.

If a commit fails after the executor has already produced a successful response,
the interceptor rewrites the response to `commit_failed`. A failed rollback is
reported distinctly, with both errors.

This is the Spring `@Transactional` / .NET `TransactionScope` / JTA model:
ambient transaction propagated through context, enlisted by cooperating
handlers. Side effects that do not enlist — email, external HTTP calls, file
writes — are not rolled back. That is a documented handler contract; the runtime
makes no attempt to detect it.

### Configuration

`ServerConfig` and `PackageHandler` gain `EnableCmdChain bool`,
`MaxChainLength int`, and `ChainTimeout time.Duration`, plumbed through
`RegisterPackage` beside the existing `Interceptor` assignment. `EnableCmdChain`
defaults to off.

### Client API

```go
NewChainBuilder(pkg).Add(cmd).AddWithHeaders(cmd, hdrs).Atomic(true).Build()
(*PackageClient).SendChain(ctx, cmds ...any) ([]ChainResult, error)
(*PackageClient).SendChainReq(ctx, chain) ([]ChainResult, error)
```

Result decoding reuses the existing `InstanceOf(cmdName + "Result")` convention.

## Affected Files

| File | Change |
| --- | --- |
| `runtime/go/mvep/chain.go` | New — types, `RawPayload` codec, `executeChain`, context seam |
| `runtime/go/mvep/chain_test.go` | New — batching, security, and atomicity coverage |
| `runtime/go/mvep/mvepackge.go` | Branch in `ServeCmdReq`; new `PackageHandler` fields |
| `runtime/go/mvep/middleware.go` | No change; `OnlyCommands` is reused as-is |
| `runtime/go/mvep/server/server.go` | Config fields; plumb through `RegisterPackage` |
| `runtime/go/mvep/client/client.go` | Chain builder and send methods |
| `runtime/go/mvep/example_transaction_test.go` | New — runnable `TransactionInterceptor` example |
| `runtime/go/mvep/server/SERVER.md` | Chain semantics, atomicity contract, local-trust caveat |
| `CHANGELOG.md` | `[Unreleased]` entries |

## Risks and Compatibility

- **Amplification.** One request becomes N handler executions. `MaxChainLength`
  bounds it, but this compounds the unbounded body reads tracked in #18, which
  is why that issue is a prerequisite.
- **Per-command auth is inert on trusted listeners.** `LocalTrustMiddleware`
  marks the context trusted and `AuthInterceptor` short-circuits on it, so every
  inner command inherits trust regardless of its own `auth` header. Defensible,
  but it will be assumed to work everywhere unless documented prominently.
- **Observability regression.** Access logs show one request for N commands.
  Per-inner-command logging must land in the same change, not as a follow-up.
  The `ServeCmdReq` recursion makes this nearly free.
- **Lock duration.** An open transaction holds database locks for the entire
  chain, which makes `ChainTimeout` mandatory rather than optional.
- **Partial atomicity is worse than none if misunderstood.** A chain that mixes
  enlisted database writes with non-enlisted side effects will roll back only
  part of its work while reporting `rolled_back` for everything.
- **Reserved-name shadowing.** A package that already defines a command named
  `CmdChain` would be shadowed when the feature is enabled.
- **Header injection.** Inner `x-mvep-*` headers are unfiltered today, and a
  chain widens the surface by letting one request carry many header sets.

## Verification

1. `go test ./runtime/go/mvep/... -count=1`.
2. `go test -race ./runtime/go/mvep/... -count=1`.
3. `go test -run TestChain_AuthPolicyNotBypassable ./runtime/go/mvep/` — the
   security gate; must fail if the executor is changed to call `executeCmd`.
4. `go test -run TestChain_Atomic ./runtime/go/mvep/` with a fake transaction
   manager asserting exactly one `Begin` and one `Rollback` for an N-command
   chain, and no nested `Begin`.
5. An interceptor call-counter equal to N+1 for an N-command chain.
6. Manual `curl` of a two-command chain confirming inline JSON objects rather
   than base64 strings in both request and response.

## Rollout

1. Ship batching (T1–T9) and atomicity (T10–T12) as separate PRs against this
   issue, or split atomicity into its own issue if the first half lands cleanly.
2. Keep `EnableCmdChain` off by default through at least one release.
3. Follow-up issues, not part of this plan:
   - TypeScript client-side chain construction and result decoding.
   - `mvep validate` rejecting specs that define a command named `CmdChain`.
   - Saga/compensation design for distributed transports.

## Decision Log

- **Fail-fast, not continue-on-error.** Matches Salesforce Composite `allOrNone`
  and Microsoft Graph `dependsOn`. JSON-RPC batch is continue-on-error, but
  JSON-RPC also permits reordering, which we do not.
- **Positional results, not id correlation.** Matches Redis MULTI/EXEC and is
  stricter than JSON-RPC, which correlates by `id` and explicitly allows the
  server to reorder. Positional removes an entire class of correlation bug.
- **No result forwarding.** Keeps v1 a batch. The alternative is a reference
  syntax and a resolver, which is Cap'n Proto promise-pipelining territory.
- **Inline payloads over nested bytes.** Readable JSON and readable logs, at the
  cost of dropping protobuf support in v1.
- **Per-command auth override allowed.** No established framework does this —
  Graph, OData change sets, and AWS batch operations all inherit a single outer
  identity. It is only safe because the `ServeCmdReq` recursion re-runs
  `AuthInterceptor` per command.
- **Transactions in an interceptor, not the runtime.** Keeps the dispatch layer
  storage-agnostic and composable. The runtime concedes one boolean context
  marker, purely so it can reject unsupported atomic requests and label results
  honestly.
- **Atomicity opt-in and rejected when unsupported.** The dangerous mode is a
  client requesting atomicity from a server with no transaction manager wired up
  and silently not getting it.
- **`rolled_back` is a distinct status.** Reporting `ok` for work that was rolled
  back would be actively misleading.
- **Built-in over spec-generated.** No toolkit release, no regeneration, works
  for every existing package.
- **Redis MULTI/EXEC is not cited as precedent for rollback.** It executes
  queued commands atomically but does not undo them on a runtime error; it is
  commonly and wrongly cited as transactional.

## Open Questions

These must be resolved before this plan leaves draft.

1. **Is batching worth its observability cost?** Two significant retreats are
   worth weighing: MCP added JSON-RPC batching in its 2025-03-26 revision and
   **removed it** in 2025-06-18, citing implementation complexity and
   error-handling ambiguity; Google deprecated its global HTTP batch endpoints.
   The common thread is that HTTP/2 multiplexing shrank the round-trip savings
   while the costs — ambiguous partial failure, collapsed tracing, undercounted
   rate limits — stayed fixed. What is the concrete latency problem this solves,
   and is it measured?
2. **Does the per-topic pub/sub goal invalidate the design?** If commands become
   NATS or MQTT subjects served by independent subscribers, a chain has no
   obvious owner and a shared transaction is impossible. Is `CmdChain` a
   local-transport-only capability, or does it need a distributed story before
   the shape is fixed?
3. **Should the encoder allowlist be a capability interface instead?** The
   current design hardcodes which encoders support `RawPayload`. A
   `RawEmbedder` interface on the encoding registry would be more extensible but
   adds surface area to `ugo/oencoding`.
4. **Should atomicity ship at all in v1?** Batching is useful without it, and the
   transaction seam is the part most likely to be misunderstood in production.
5. **How should chained commands appear in traces and access logs?** One span
   per inner command with the chain as parent is the obvious answer, but MVEP
   has no tracing integration today, so this may be premature.

## Progress

> Provisional. Do not begin until this plan leaves draft.

- [ ] T1 — Add `RawPayload` with inline-JSON round-trip tests
- [ ] T2 — Define chain envelope types and the encoder allowlist
- [ ] T3 — Add failing tests for ordering, fail-fast, and the chain guards
- [ ] T4 — Branch chain dispatch into `ServeCmdReq`'s core handler
- [ ] T5 — Implement `executeChain` with header copying and recursive dispatch
- [ ] T6 — Plumb `EnableCmdChain`, `MaxChainLength`, and `ChainTimeout` through config
- [ ] T7 — Add the auth-bypass regression test and per-command auth coverage
- [ ] T8 — Add per-inner-command logging
- [ ] T9 — Add the client chain builder and send methods with a round-trip test
- [ ] T10 — Add the atomic context seam and `atomic_not_supported` rejection
- [ ] T11 — Implement `rolled_back` relabeling on atomic failure
- [ ] T12 — Ship the `TransactionInterceptor` example and atomicity documentation

## Tasks

### T1 — `RawPayload`

- **Outcome**: A single type carries pre-encoded inner payloads and serializes
  inline for JSON.
- **Verification**: Round-trip test asserting the encoded bytes contain
  `"payload":{` and no base64.
- **Notes**: Needs `MarshalCBOR`/`UnmarshalCBOR` if CBOR support is required at
  the same time.

### T2 — Chain envelope types

- **Outcome**: Request and result types exist with the four-value status enum.
- **Verification**: Compiles; encoder allowlist returns
  `chain_unsupported_encoder` for protobuf.
- **Notes**: See Open Question 3 on whether the allowlist should be a capability
  interface.

### T3 — Failing execution tests

- **Outcome**: Ordering, fail-fast, and every guard are specified before
  implementation.
- **Verification**: Tests fail because `CmdChain` is not dispatched.
- **Notes**: Cover execution order via a side-effect log, skipped tail, nested
  chain, length limit, empty chain, mid-chain cancellation, unknown inner
  command, header merge, and outer-map immutability.

### T4 — Dispatch branch

- **Outcome**: `CmdChain` is recognized before `executeCmd` while remaining
  inside the outer interceptor chain.
- **Verification**: A chain request reaches the executor; outer interceptors run
  exactly once for the chain itself.
- **Notes**: The branch must live in the core-handler closure, not in
  `ServeHTTP` and not in `executeCmd`.

### T5 — `executeChain`

- **Outcome**: Commands execute in order through recursive `ServeCmdReq` calls,
  with all guards enforced.
- **Verification**: The T3 tests pass; the response is never nil.
- **Notes**: Copy the outer header map per entry; never alias caller maps.

### T6 — Configuration

- **Outcome**: The feature is off by default and its bounds are configurable.
- **Verification**: A chain sent to a server with `EnableCmdChain: false` is
  rejected as an unknown command.
- **Notes**: Mirror the fields on both `ServerConfig` and `PackageHandler`.

### T7 — Security coverage

- **Outcome**: Wrapping a command in a chain cannot bypass command-scoped
  interceptor policy.
- **Verification**: `AuthInterceptor` with `OnlyCommands("AdminCmd")` still
  rejects `AdminCmd` inside a chain; an inner command with its own valid token
  succeeds and with an invalid token fails.
- **Notes**: This test is the reason the recursion design exists. It must fail if
  someone later optimizes the executor into calling `executeCmd`.

### T8 — Observability

- **Outcome**: Each inner command is individually logged.
- **Verification**: An N-command chain produces N inner log lines plus one for
  the chain.
- **Notes**: Nearly free given recursion; must not be deferred to a follow-up.

### T9 — Client API

- **Outcome**: Callers can build and send chains with typed results.
- **Verification**: A real `Client` to `Server` round trip decoding two typed
  results — which also fills the missing client/server integration test gap.
- **Notes**: Reuses `InstanceOf(cmdName + "Result")`.

### T10 — Atomic seam

- **Outcome**: The runtime can tell whether transaction support is present
  without knowing what a transaction is.
- **Verification**: `Atomic: true` without the marker returns
  `atomic_not_supported`.
- **Notes**: Two exported functions and one context key; nothing more.

### T11 — Rollback labeling

- **Outcome**: Results never claim success for rolled-back work.
- **Verification**: An atomic chain failing at entry 3 reports entries 1 and 2 as
  `rolled_back`, 3 as `error`, and the remainder as `skipped`.
- **Notes**: Non-atomic chains keep reporting `ok` for completed entries.

### T12 — Transaction example and docs

- **Outcome**: The transaction pattern is demonstrated and its limits are stated
  plainly.
- **Verification**: The example test runs; a fake manager records one `Begin`
  and one `Rollback` with no nested `Begin`.
- **Notes**: Must state that non-enlisted side effects are not rolled back, that
  the interceptor requires `OnlyCommands(CmdChainName)`, and that per-command
  auth is inert behind `LocalTrustMiddleware`.
