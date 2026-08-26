# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A personal tool that generates a tailored, one-page CV (PDF) for a specific job application, and tracks the Job Listings/Applications built around it. It is not a generic resume builder — it's built around one person's career history and produces a *Tailored CV* per *Job Description* via a Claude Code skill, not a build script. A standalone local web app sits alongside the skill for browsing/editing Master Data and tracking Job Listings and Applications.

Read `CONTEXT.md` first for the domain vocabulary (Master Data, Entry, Client Engagement, Selection, Rewrite, Tailored CV, Job Listing, Application, etc.) — the rest of this file assumes it. Check `docs/adr/` for why the architecture looks like this before proposing to change it.

## Running it

The tailoring "build" is the `tailor-cv` skill (`.claude/skills/tailor-cv/SKILL.md`) — invoke it with a job description (pasted text or URL) or nothing (Default Mode). It walks: Selection → Rewrite → Text Review (approval required) → Render → Visual Review (approval required).

Rendering requires the `typst` CLI on `PATH`. To render manually once a tailored data file exists:
```
typst compile --root . template/cv.typ output/<slug>/cv.pdf --input data=output/<slug>/data.json
```

The web app (backend + frontend) runs via `docker-compose up` — backend on `127.0.0.1:8080`, frontend on `127.0.0.1:5173`, both localhost-only with no auth. Generation calls the Claude API directly (ADR-0005; set `ANTHROPIC_API_KEY`); Render shells out to the same `typst` CLI the skill uses, so the backend needs `template/` and `output/` alongside `data/` under one `PROJECT_ROOT` (see ADR-0012). See `README.md` for the API surface and frontend dev commands.

Note: ADR-0001 ("no Node.js/JS toolchain") only ever applied to the tailoring pipeline itself and has since been superseded by ADR-0004, which added the standalone web app (Go backend + React/TS/Vite frontend, see ADR-0009). The tailoring pipeline still has no Node/JS involvement — that part of ADR-0001's reasoning stands — but the repo as a whole now does.

## Architecture

- `data/profile.yaml` — contact info + Static Sections (education, publications, awards, activities, languages). Always included in full, never selected or rewritten.
- `data/experience/*.md`, `data/projects/*.md` — Master Data. One file per Entry: YAML frontmatter (`employer`/`client`/dates/`tags`/…) + Markdown bullets. A single employer can have multiple Entries (one per Client Engagement, e.g. `data/experience/quantyca-*.md`) so Selection can surface one client's work independently of another's.
- `data/cover-letter-snippets/*.md` — optional Master Data. One file per Cover Letter Snippet: YAML frontmatter (`kind`, optional `tags`) + a Markdown paragraph body.
- `template/cv.typ` — pure presentation. Reads one assembled JSON file (path passed via `--input data=...`) and renders it; contains no relevance/selection logic. The Tech Stack section it renders is expected to already be derived (deduplicated `tags` of the selected Entries) by whoever assembled the JSON — the template just prints it.
- `output/` — gitignored. Rendered PDFs and the per-Generation assembled JSON are derived artifacts, not Master Data.
- `.claude/skills/tailor-cv/` — the skill that drives the whole tailoring pipeline described above.
- `backend/` — Go HTTP API (see `backend/internal/api`) that reads/writes the same Master Data files under `data/` that the skill uses.
- `frontend/` — React + TypeScript + Vite app consuming that API.

Adding a new job or project means adding a new Markdown file under `data/experience/` or `data/projects/` following the existing frontmatter shape — not writing code.
