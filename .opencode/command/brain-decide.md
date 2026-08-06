---
description: Draft an ADR — capture a decision with context and alternatives before the reasons evaporate.
---

Decision to record: $ARGUMENTS

Paths:
- Vault: /Users/dread-code/Documents/Vault/lazy-post
- Repo: /Users/dread-code/Dev/postgo

Steps:
1. Read `00 Vault Guide.md` and `90 Templates/tpl-adr.md`. Check existing `05 ADRs/` for the next number and for related or superseded decisions.
2. Gather context: relevant architecture notes, the code involved, and the conversation so far.
3. If the decision, its context, or the alternatives are unclear, ask the user — max 3 sharp questions.
4. Write the ADR to `05 ADRs/ADR-NNNN <short title>.md` using the template. Status: `accepted` if the decision is already made, `proposed` if still open. Context and consequences must be honest about trade-offs; the alternatives table needs real pros/cons, not strawmen.
5. Link it: add the ADR under "Decisions" in `15 MOCs/MOC - Architecture.md` (or the relevant MOC), and reference it from affected `03 Architecture/` notes ("Decided in [[ADR-…]]").
6. If this supersedes an older ADR, set `supersedes`/`superseded_by` on both and change the old one's status to `superseded` — never edit its decision text.
7. Verify all [[links]] resolve; report the ADR path and a one-paragraph summary.
