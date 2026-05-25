# AGENTS.md — Mioru

This file is intentionally thin. The single source of truth for agents — repository
layout, workflow rules, priorities, development principles, backend/frontend
architecture, security posture, env vars, and deployment — lives in **[CLAUDE.md](./CLAUDE.md)**.

Read `CLAUDE.md` first.

## The non-negotiables (full detail in CLAUDE.md)

- **Workflow:** plan → user approves → make change → user checks → user says commit → agent commits. No commits without approval. The agent commits/branches but **cannot push** (SSH passphrase); the user pushes. Remote: `git@github.com:rem22p/mioru.git`.
- **Priorities (in order):** 1) reliability of financial operations, 2) safety of user data, 3) speed/performance.
- **Principles:** TDD, YAGNI, DRY, OWASP / security-by-default, modular decomposition, architectural integrity.
