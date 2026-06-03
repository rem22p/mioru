# AGENTS.md

Single source of truth for agents working in this repo. Read **[CLAUDE.md](./CLAUDE.md)** first.

## The non-negotiables (full detail in CLAUDE.md)

- **Workflow:** plan → user approves → make change → user checks → user says commit → agent commits. No commits without approval. Agent pushes branches and creates PRs but never merges.
- **Priorities (in order):** 1) reliability of financial operations, 2) safety of user data, 3) speed/performance.
- **Principles:** TDD, YAGNI, DRY, OWASP / security-by-default, modular decomposition, architectural integrity.
- **Remote:** `git@github.com:rem22p/mioru.git` (SSH) for the user. Agent uses HTTPS with PAT.
