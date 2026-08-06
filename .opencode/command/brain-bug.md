---
description: Bug workflow — investigate, then turn it into vault knowledge (gotcha, postmortem, playbook, ADR).
---

Bug: $ARGUMENTS

Paths:
- Vault: /Users/dread-code/Documents/Vault/lazy-post
- Repo: /Users/dread-code/Dev/postgo

Steps:
1. Read `00 Vault Guide.md`. Search the vault FIRST — the bug or a sibling may already be known (grep `08 Gotchas/` and `12 Postmortems/` for the error message). If it is a known gotcha, say so and apply the documented fix.
2. Investigate in the repo: reproduce, isolate the root cause. Fix the code if the fix is small and obvious; otherwise stop at diagnosis and state what is needed.
3. Capture per the bug lifecycle (Vault Guide §5):
   - Always: a gotcha in `08 Gotchas/` (symptom with exact error text, cause, fix, prevention)
   - If it cost >30 min or risks recurrence: a postmortem in `12 Postmortems/` using `90 Templates/tpl-postmortem.md`, and update `12 Postmortems/Postmortems Index.md`
   - If a design decision caused it: a new ADR (or supersede the responsible one)
   - If the fix is a reusable procedure: a playbook in `10 Playbooks/`
   - Update any `03 Architecture/` note whose current truth changed
4. Add a regression test in the repo when practical; run `go vet ./... && go test ./...` (toolchain setup: `10 Playbooks/Playbook - Set up the Go toolchain.md`).
5. Link everything (gotcha ↔ postmortem ↔ playbook ↔ MOC), set frontmatter dates, verify all [[links]] resolve.
6. Report: root cause in one sentence, notes created/updated, test status.
