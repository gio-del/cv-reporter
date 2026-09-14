# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A personal tool that generates a tailored, one-page CV (PDF) for a specific job application, and tracks the Job Listings/Applications built around it. It is not a generic resume builder — it's built around one person's career history and produces a *Tailored CV* per *Job Description* via a Claude Code skill, not a build script. A standalone local web app sits alongside the skill for browsing/editing Master Data and tracking Job Listings and Applications.

Read `CONTEXT.md` first for the domain vocabulary (Master Data, Entry, Client Engagement, Selection, Rewrite, Tailored CV, Job Listing, Application, etc.) — the rest of this file assumes it. Check `docs/adr/` for why the architecture looks like this before proposing to change it.

## Running it

The tailoring "build" is the `tailor-cv` skill (`plugins/cv-reporter-skills/skills/tailor-cv/SKILL.md`, invoked as `/cv-reporter-skills:tailor-cv`) — invoke it with a job description (pasted text or URL), a reference to an already-tracked Application, or nothing (Default Mode). It walks: Selection → Rewrite → Text Review (approval required) → Render → Visual Review (approval required), recording the Generation against the tracked Application afterward when that's the input mode used.

Rendering requires the `typst` CLI on `PATH`. To render manually once a tailored data file exists:
```
typst compile --root . template/cv.typ output/<slug>/cv.pdf --input data=output/<slug>/data.json
```

The web app (backend + frontend) runs via `docker-compose up` — backend on `127.0.0.1:8080`, frontend on `127.0.0.1:5173`, both localhost-only with no auth. Generation calls the Claude API directly (ADR-0005; copy `.env.example` to `.env` and set `ANTHROPIC_API_KEY` — `docker-compose.yml` forwards it into the backend container). Each kind of Claude call (call site) runs on its own default model — Haiku 4.5 for the RAL/Contact extraction calls and Application Method inference, Sonnet 5 for everything else — overridable per call site with `CV_REPORTER_MODEL_<CALL_SITE>` or for all of them with `CV_REPORTER_MODEL_DEFAULT` (ADR-0025; the table lives in `backend/internal/claude/models.go` and README's "Claude models per call site"; any model a call site can default to must also be priced in `backend/internal/claude/pricing.go`). Render shells out to the same `typst` CLI the skill uses, so the backend needs `template/` and `output/` alongside `data/` under one `PROJECT_ROOT` (see ADR-0012). See `README.md` for the API surface and frontend dev commands.

An opt-in LAN-reachable mode (issue #57) exists for reaching the app from other devices on the same network: set `BIND_ADDR=0.0.0.0` and `LAN_AUTH_TOKEN` in `.env` and run `docker compose --profile lan up`. This is an explicit, opt-in exception to ADR-0004's "localhost-only, no auth" default, not a reversal of it — the default `docker-compose up` invocation is unchanged. See `README.md`'s "Optional: LAN-reachable mode" section for the tradeoffs (shared static secret via `X-CV-Reporter-Token`, no TLS, single trusted network assumed).

Job Listing and Application records carry a `schemaVersion` (ADR-0034, `backend/internal/tracking/schema.go`). Records written before it existed read as legacy, and `backend/cmd/migrate-records` migrates them: `cd backend && go run ./cmd/migrate-records -data-dir ../data` is a dry run that only reports (exit 3 when anything is pending); add `-write` to apply. It backfills only `freshnessStatus` on Job Listings and, for Applications still at Saved, `statusUpdatedAt` plus a single Saved `statusHistory` entry from the Job Listing's `savedAt`; it never synthesises history past Saved, and never fills a legacy Generation's `sourceSnippetIds`/`entryIds`/`usage`/`language`. `data/jobs/` and `data/applications/` are gitignored, so never run it with `-write` against a real `data/` from a test or an experiment — use a temp copy. When adding a field to either record, bump `CurrentSchemaVersion` and teach `MigrateRecords` the step (see the comment in `schema.go`) instead of adding another "absent in older records" tolerance to a reader. See `README.md`'s "Migrating Job Listing and Application records".

Note: ADR-0001 ("no Node.js/JS toolchain") only ever applied to the tailoring pipeline itself and has since been superseded by ADR-0004, which added the standalone web app (Go backend + React/TS/Vite frontend, see ADR-0009). The tailoring pipeline still has no Node/JS involvement — that part of ADR-0001's reasoning stands — but the repo as a whole now does.

## Architecture

- `data/profile.yaml` — contact info + Static Sections (education, publications, awards, activities, languages). Always included in full, never selected or rewritten.
- `data/experience/*.md`, `data/projects/*.md` — Master Data. One file per Entry: YAML frontmatter (`employer`/`client`/dates/`tags`/…) + Markdown bullets. A single employer can have multiple Entries (one per Client Engagement, e.g. `data/experience/example-client-*.md`) so Selection can surface one client's work independently of another's.
- `data/cover-letter-snippets/*.md` — optional Master Data. One file per Cover Letter Snippet: YAML frontmatter (`kind`, optional `tags`) + a Markdown paragraph body.
- `template/cv.typ` — pure presentation. Reads one assembled JSON file (path passed via `--input data=...`) and renders it; contains no relevance/selection logic. The Tech Stack section it renders is expected to already be derived (deduplicated `tags` of the selected Entries) by whoever assembled the JSON — the template just prints it.
- `output/` — gitignored. Rendered PDFs and the per-Generation assembled JSON are derived artifacts, not Master Data. Every Generation writes to its own `output/<label>-<UTC yyyymmdd-hhmmss>/` directory (`-2`, `-3`, … if taken) and never into an existing one (issue #105) — the rule lives in `generation.Render` for the app and is mirrored in prose by SKILL.md step 5. Records may outlive their files; a missing directory is expected, not an error.
- `.claude-plugin/marketplace.json` + `plugins/cv-reporter-skills/` — a repo-local Claude Code plugin marketplace holding this repo's own skill(s), currently just `tailor-cv` (`plugins/cv-reporter-skills/skills/tailor-cv/SKILL.md`), the skill that drives the whole tailoring pipeline described above. Kept separate from the author's general-purpose personal skills (`develop`, `tdd`, `domain-modeling`, …), which stay synced in via `skills-lock.json` from an external repo and are untouched by this one.
- `backend/` — Go HTTP API (see `backend/internal/api`) that reads/writes the same Master Data files under `data/` that the skill uses.
- `frontend/` — React + TypeScript + Vite app consuming that API.

Adding a new job or project means adding a new Markdown file under `data/experience/` or `data/projects/` following the existing frontmatter shape — not writing code.

## Keeping docs in sync

Whenever a change alters something `README.md`, `CONTEXT.md`, or `docs/adr/` documents (a new/changed API route, a new top-level directory, a new running-it step, a superseded decision), update that documentation in the same session/commit as the code change — don't leave it for a later pass.
