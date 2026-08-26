# CV Reporter

A personal tool that turns one person's complete career history (Master Data) into a tailored, one-page CV (PDF), generated on demand for a specific job application. It is not a generic resume builder.

See `CONTEXT.md` for the domain vocabulary (Master Data, Entry, Client Engagement, Selection, Rewrite, Tailored CV, Job Listing, Application, ...) and `docs/adr/` for why the architecture looks like this.

## Two parts

**1. The tailoring pipeline** — a Claude Code skill (`.claude/skills/tailor-cv/`), not a build script. Run it with a job description (pasted text or URL) or nothing (Default Mode). It walks: Selection → Rewrite → Text Review (approval required) → Render → Visual Review (approval required). Rendering is done with [Typst](https://typst.app) (`typst` must be on `PATH`); there's no Node/JS involved in this part.

**2. The web app** — a standalone local app (Go backend + React/TypeScript/Vite frontend, run via `docker-compose`, localhost-only, no auth) for browsing and editing Master Data, and for tracking Job Listings and Applications. See `docs/adr/0004-standalone-web-app.md` and `docs/adr/0009-go-backend-react-frontend.md` for why.

## Repo layout

- `data/profile.yaml` — contact info + Static Sections (education, publications, awards, activities, languages). Always included in full, never selected or rewritten.
- `data/experience/*.md`, `data/projects/*.md` — Master Data. One file per Entry: YAML frontmatter (`employer`/`client`/dates/`tags`/...) + Markdown bullets.
- `template/cv.typ` — pure presentation. Reads one assembled JSON file and renders it; contains no selection/relevance logic.
- `output/` — gitignored. Rendered PDFs and per-Generation assembled JSON are derived artifacts, not Master Data.
- `.claude/skills/tailor-cv/` — the skill that drives the tailoring pipeline.
- `backend/` — Go HTTP API serving/editing the Master Data files under `data/` (see `backend/internal/api`).
- `frontend/` — React + TypeScript + Vite app consuming that API.
- `docs/adr/` — architecture decision records.

## Running the tailoring pipeline

Invoke the `tailor-cv` skill in Claude Code with a job description (or nothing, for Default Mode). To render manually once a tailored data file exists:

```
typst compile --root . template/cv.typ output/<slug>/cv.pdf --input data=output/<slug>/data.json
```

Adding a new job or project means adding a new Markdown file under `data/experience/` or `data/projects/` following the existing frontmatter shape — not writing code.

## Running the web app

```
docker-compose up
```

- Backend: `http://127.0.0.1:8080` (reads/writes `./data`, mounted into the container)
- Frontend: `http://127.0.0.1:5173`

### Backend API

| Method | Path | |
|---|---|---|
| GET | `/api/healthz` | health check |
| GET | `/api/master-data/entries` | list Entries |
| POST | `/api/master-data/entries` | create an Entry |
| GET | `/api/master-data/entries/{id}` | get an Entry |
| PUT | `/api/master-data/entries/{id}` | update an Entry |
| DELETE | `/api/master-data/entries/{id}` | delete an Entry |
| GET | `/api/master-data/profile` | get profile + Static Sections |
| PUT | `/api/master-data/profile` | update profile + Static Sections |
| GET | `/api/master-data/cover-letter-snippets` | list Cover Letter Snippets |
| POST | `/api/master-data/cover-letter-snippets` | create a Cover Letter Snippet |
| GET | `/api/master-data/cover-letter-snippets/{id}` | get a Cover Letter Snippet |
| PUT | `/api/master-data/cover-letter-snippets/{id}` | update a Cover Letter Snippet |
| DELETE | `/api/master-data/cover-letter-snippets/{id}` | delete a Cover Letter Snippet |

Backend tests are Go `testing`-package HTTP integration tests, run with `go test ./...` from `backend/`.

### Frontend dev loop

From `frontend/`: `npm run dev` (served by the `frontend` service above inside Docker), `npm run build`, `npm run lint`.
