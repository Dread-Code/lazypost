---
description: Start the design flow — turn an idea into a Design note (Idea → Design → ADR).
---

Design: $ARGUMENTS

Vault: /Users/dread-code/Documents/Vault/lazy-post

Steps:
1. Read `90 Templates/tpl-design.md` and `02 Vision/Roadmap.md`. If $ARGUMENTS names an idea, read its note from `13 Ideas/`; otherwise treat $ARGUMENTS as the idea text. Check `04 Designs/` for duplicates — if one exists, enrich it instead of creating a new note.
2. Create `04 Designs/Design - <short name>.md` from the template, `status: draft`, following the vault Feature flow (Idea → Design → ADR, [[00 Vault Guide]]): Problem (evidence from the idea note), Proposal (concrete — data structures, key flows, sketches, file:line in `internal/`), Alternatives considered (table), Impact (which subsystems change and which notes in `03 Architecture/` must be updated when it lands), Open questions, Related (Idea link; "Will become ADR: [[ADR-…]] (create when decided)" — or "none expected" for a pure refactor, as in [[Design - decompose root model]]; MOC links).
3. Update the source idea note: `status: parked` → `committed`, add "Becomes: [[Design - <name>]]" to Related, bump `updated`.
4. Link it: append "(design in progress → [[Design - <name>]])" to the idea's line in `02 Vision/Roadmap.md`; move the idea out of "Parked ideas" into an "In design" section of `15 MOCs/MOC - Features.md` (create the section if absent). Bump `updated` on both.
5. Do NOT create the ADR — that is `/brain-decide`'s job once the design is decided. Verify all wikilinks resolve.
6. Reply with the note path and one sentence on where the design landed.
