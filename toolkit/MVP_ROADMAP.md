# MVP Roadmap — Mainvec Platform

> Feature roadmap for the MVP toolkit (mvgen, mvpgo, ugo).
>
> For current features and usage, see [MVP_SKILL.md](MVP_SKILL.md).

---

## Phase 1 — Spec Completeness (Foundation)

> Goal: Bring the MVP spec to feature parity with what developers expect from any API definition format.

| Feature | Description | Scope |
|---------|-------------|-------|
| **Enum type** | Named enum values in spec → Go `const` + `type`, proto `enum`, TS `enum` | Spec schema + Go/JS codegen |
| **Deprecation markers** | `"deprecated": true` on commands and fields → `// Deprecated:` Go comments, `@deprecated` JSDoc | Spec schema + Go/JS codegen |
| **Required/optional semantics** | Promote `tags: ["required"]` to first-class `"required": true` boolean | Spec schema |
| **Default values** | `"default": "value"` on fields → generated constructors and zero-values honor it | Spec schema + Go/JS codegen |
| **Field-level docs passthrough** | Ensure `title`/`desc` from spec appear as comments in all generated Go and JS/TS output | Go/JS codegen templates |

## Phase 2 — Developer Experience (Tooling)

> Goal: Make MVP specs self-documenting and safe to evolve.

| Feature | Description | Scope |
|---------|-------------|-------|
| **`mvp docs`** | Generate markdown or HTML API reference from a spec file | New mvp command |
| **Validation rules** | `min`, `max`, `minLen`, `maxLen`, `pattern` on fields → generated `Validate()` methods in Go, `validate()` in JS | Spec schema + Go/JS codegen |
| **Error definitions** | `"errors"` section in spec with named error codes → typed error constructors, client-side error matching | Spec schema + Go/JS codegen + runtime |
| **Breaking change detection** | `mvp diff --old v1.json --new v2.json` → reports removed fields, changed fnums, type changes | New mvp command |
| **Testing harness** | `mvp test-scaffold` → generates integration test file with HTTP client setup + one test per command | New mvp command + Go template |

## Phase 3 — Ecosystem Interop (Reach)

> Goal: Let MVP projects participate in the broader API ecosystem without maintaining parallel specs.

| Feature | Description | Scope |
|---------|-------------|-------|
| **OpenAPI export** | `mvp export --format openapi` → generates `openapi.yaml` from MVP spec | New mvp command |
| **Mock server** | `mvp mock --in spec.json --port 8080` → serves fake responses with example data for frontend dev | New mvp command |
| **Postman collection export** | Generate Postman/Insomnia collection from spec | New mvgen command |

## Phase 4 — Multi-Language (Scale)

> Goal: Extend MVP beyond Go and JS to enable cross-team adoption.

| Feature | Description | Scope |
|---------|-------------|-------|
| **Python codegen** | `--lang python` → dataclasses + typed client + package utilities | New codegen templates |
| **Rust codegen** | `--lang rust` → structs with serde + client | New codegen templates |
| **Swift codegen** | `--lang swift` → Codable structs + async client | New codegen templates |

## Phase 5 — Advanced Capabilities (Differentiation)

> Goal: Move beyond request-response into patterns that gRPC and OpenAPI handle differently.

| Feature | Description | Scope |
|---------|-------------|-------|
| **Streaming** | `"stream": true` on `resultFields` → SSE/WebSocket server handler + client consumer | Spec schema + codegen + runtime |
| **Multi-spec composition** | `$ref` across spec files for shared records between microservices | Spec schema + mvgen resolver |
| **Pagination pattern** | `"paginated": true` on commands → auto-generates cursor/limit fields + iterator client helpers | Spec schema + codegen |
| **Event/webhook definitions** | `"events"` section in spec for async notifications | Spec schema + codegen |

## Priority Order

```
Now          Soon         Next         Later
─────────    ─────────    ─────────    ──────────
Enums        mvp docs     OpenAPI      Streaming
Deprecation  Validation   Mock server  Multi-spec
Defaults     Error defs   Python       Pagination
Required     Diff/break   Postman      Events
Doc comments Test harness Rust/Swift
```
