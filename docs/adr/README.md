# Architecture Decision Records

This directory contains lightweight ADRs (Architecture Decision Records) capturing
significant design decisions and migration plans for `ansible-mmv1`. Each ADR
describes the context behind a decision, the plan itself, and its outcome, so
future contributors (human or agent) don't have to reconstruct the reasoning from
scratch.

## Status values

- **Proposed** - plan written and reviewed, not yet executed
- **Accepted** - plan approved, execution in progress or scheduled
- **Executed** - plan fully carried out; kept for historical context
- **Superseded by ADR-NNNN** - decision replaced by a later ADR

## Index

| ADR | Title | Status |
|---|---|---|
| [0001](0001-mmv1-upgrade-plan.md) | Magic-Modules v1 (MMv1) Upgrade Plan | Executed (2026-07-14) |
| [0002](0002-examples-to-samples-migration.md) | Migrate Ansible Overlay Content from `examples:` to `samples:` | Proposed (2026-07-16) |

## Adding a new ADR

1. Create `docs/adr/NNNN-short-title.md` using the next sequential number.
2. Start the file with `# ADR NNNN: <Title>` followed by a `**Status:** Proposed (<date>)` line.
3. Add a row to the index table above.
4. Update the `Status:` line (and the index) as the ADR moves through its lifecycle.
