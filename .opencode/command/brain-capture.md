---
description: End-of-session capture — distill this session's knowledge into the vault (gotchas, learnings, ADRs, architecture updates).
---

You are running the end-of-session knowledge capture for lazypost.

Paths:
- Vault: /Users/dread-code/Documents/Vault/lazy-post
- Repo: /Users/dread-code/Dev/postgo

Session notes from the user: $ARGUMENTS

Steps:
1. Read `00 Vault Guide.md` (conventions, metadata, linking, templates) and `01 Inbox/Inbox.md`.
2. Reconstruct what happened this session: the conversation so far, the user's notes above, and repo state (`git status`/`git diff` if it is a git repo; otherwise recently modified files in the repo).
3. Identify candidates, then create/update notes using the templates in `90 Templates/`:
   - Decision made → ADR in `05 ADRs/` (next number in the sequence)
   - Surprise or trap discovered → gotcha in `08 Gotchas/` (include exact error messages)
   - New understanding → learning in `07 Learnings/`
   - Repeated procedure → playbook in `10 Playbooks/`
   - Feature thought → idea in `13 Ideas/`
   - Subsystem changed → update the matching `03 Architecture/` note (current truth!)
4. Empty `01 Inbox/` — file or delete every scrap note there.
5. For every note created/updated: set frontmatter (`created`/`updated` = today), add a `## Related` link to the relevant MOC, and link related notes to each other.
6. Update `00 Home.md` ("Current state", "Recent decisions", "Fresh from the trenches") if any of it changed.
7. Verify: every [[link]] in notes you touched resolves to a real note.
8. Report: notes created/updated (one line each) and anything you deliberately skipped.

Rules: atomic notes (one question per note); titles state the answer; link instead of duplicating; never rewrite history — supersede ADRs instead.
