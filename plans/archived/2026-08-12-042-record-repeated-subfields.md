# 042 — Record sub-field flags ignore `Repeated` (superseded)

**Superseded by [plan 041](041-cli-pipe-input-output.md), task T1.**

This plan was folded into 041 on 2026-08-12. Both plans addressed the same
defect from opposite sides — nested and repeated command data being unreachable
from the CLI — and they share a single invariant: every
(`FieldType` × `Repeated` × depth) combination must be reachable, and top-level
and sub-field binding must agree.

Kept separate, 041's per-field `--<field>-file` hatch would have been registered
at the top level only, re-opening the exact top-level / sub-field drift this
plan closes. The fix and the hatch have to land against one design.

Sequencing is preserved: **T1 in plan 041 is this fix**, shipped in its own PR
and its own `runtime/go` patch tag ahead of the rest of that plan, so the
downstream `zirafa runtime register` blocker is not queued behind the larger
feature work.

See plan 041 for the problem statement, design, verification, and the retained
decision log entries covering the runtime-binder-over-codegen choice and the
`-json` suffix convention.
