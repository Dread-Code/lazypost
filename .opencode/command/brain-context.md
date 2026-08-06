---
description: Load the project's context from the vault into this session — run at the start of a new session.
---

You are starting work on postgo in a fresh session. Absorb the project context from the vault so nothing has to be rediscovered.

Paths:
- Vault (project brain): /Users/dread-code/Documents/Vault/lazy-post
- Repo: /Users/dread-code/Dev/postgo

Optional focus area: $ARGUMENTS

Steps:
1. Read and internalize: `00 Home.md`, `00 Vault Guide.md` (conventions), `02 Vision/Project Vision.md`, `02 Vision/Roadmap.md`, `03 Architecture/Architecture Overview.md`.
2. Skim every ADR in `05 ADRs/` — know each decision and its reason.
3. Read every note in `08 Gotchas/` — these are traps that already cost time once.
4. If a focus area is given: read the matching MOC in `15 MOCs/` and follow its links to the relevant architecture, learning, and playbook notes. If not: read the MOC list and the subsystem notes in `03 Architecture/`.
5. Check repo state: git status if available, plus the files relevant to the focus area.
6. Do NOT dump the vault back at the user. Reply with a short confirmation:
   - One line: what the project is and its current state
   - Top 3 next steps (from Roadmap/Home)
   - Gotchas relevant to the focus area (one line each; all of them if no focus)
   - Then ask what to work on — unless the user already said.

From this point in the session: follow the vault conventions (templates, metadata, linking) for any note you write, and use `10 Playbooks/Playbook - Set up the Go toolchain.md` before building.
