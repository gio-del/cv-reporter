<p align="center">
  <img src="brand/logo-lockup.svg" alt="CV Reporter" width="320">
</p>

<p align="center">
  A personal tool that turns one person's complete career history into a tailored, one-page CV — generated on demand for a specific job application.
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white">
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-3178C6?logo=typescript&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-20232A?logo=react&logoColor=61DAFB">
  <img alt="Typst" src="https://img.shields.io/badge/Typst-239DAD?logo=typst&logoColor=white">
</p>

<p align="center">
  <img src="docs/screenshots/job-listings.png" alt="Job Listings tracking view" width="90%">
</p>
<p align="center">
  <img src="docs/screenshots/generate.png" alt="Generation flow" width="90%">
</p>

It is not a generic resume builder — it's built around one person's career history, and produces a *Tailored CV* per *Job Description*. This stays a personal tool, not a product pitched at other users.

See `CONTEXT.md` for the domain vocabulary (Master Data, Entry, Client Engagement, Selection, Rewrite, Tailored CV, Job Listing, Application, ...) and `docs/adr/` for why the architecture looks like this.

## Two parts

**1. The tailoring pipeline** — a Claude Code skill (`plugins/cv-reporter-skills/skills/tailor-cv/`), not a build script. Run it with a job description (pasted text or URL), a reference to an already-tracked Application, or nothing (Default Mode). It walks: Selection → Rewrite (plus a Cover Letter draft, when a job is given) → Text Review (approval required) → Render → Visual Review (approval required), running the same automated quality checks the web app runs along the way (see below). A run with a job produces both a Tailored CV and a Cover Letter, drawn from your Cover Letter Snippets when you have any; Default Mode produces the CV only. Rendering is done with [Typst](https://typst.app) (`typst` must be on `PATH`); there's no Node/JS involved in this part.

**2. The web app** — a standalone local app (Go backend + React/TypeScript/Vite frontend, run via `docker-compose`, localhost-only, no auth) for browsing and editing Master Data, and for tracking Job Listings and Applications. See `docs/adr/0004-standalone-web-app.md` and `docs/adr/0009-go-backend-react-frontend.md` for why.

Job Listings can be added to the web app three ways: manual paste (URL/text), pulling from an ATS's public job-board API (Greenhouse/Lever/Ashby), or a browser extension (`extension/`) that captures the LinkedIn or Indeed job posting you're currently viewing — see `docs/adr/0007-job-sourcing.md` for why it's scoped this way and [`extension/README.md`](extension/README.md) for how to load it and how capture works.

## Repo layout

- `data/profile.yaml` — contact info + Static Sections (education, publications, awards, activities, languages). Always included in full, never selected or rewritten.
- `data/experience/*.md`, `data/projects/*.md` — Master Data. One file per Entry: YAML frontmatter (`employer`/`client`/dates/`tags`/...) + Markdown bullets.
- `data/cover-letter-snippets/*.md` — optional Master Data. One file per Cover Letter Snippet: YAML frontmatter (`kind`, optional `tags`) + a Markdown paragraph body.
- `template/cv.typ`, `template/cover-letter.typ` — pure presentation. Each reads one assembled JSON file and renders it; they contain no selection/relevance/drafting logic.
- `output/` — gitignored. Rendered PDFs and per-Generation assembled JSON are derived artifacts, not Master Data. Each Generation gets its own `output/<label>-<UTC yyyymmdd-hhmmss>/` directory (with a `-2`, `-3`, … suffix if that name is already taken), so a later Generation never overwrites an earlier one's files; both the web app and the skill follow this scheme. Clearing `output/` is safe — Application/Generation records survive, and the app shows a "no longer on disk" note for their files. The skill also writes its Selection + Rewrite result into that directory as `selection.json`, so a groundedness verdict can be re-derived later.
- `.claude-plugin/marketplace.json` + `plugins/cv-reporter-skills/` — a repo-local Claude Code plugin marketplace holding this repo's own skill(s), currently just `tailor-cv` (`plugins/cv-reporter-skills/skills/tailor-cv/`), the skill that drives the tailoring pipeline (see `docs/adr/0015-tailor-cv-distributed-as-repo-local-plugin.md` for why).
- `backend/` — Go HTTP API serving/editing the Master Data files under `data/`, and tracking Job Listings/Applications under `data/jobs/` and `data/applications/` (see `backend/internal/api`). `backend/cmd/cvcheck` is a small offline CLI over the same Generation quality checks, which the `tailor-cv` skill shells out to (ADR-0028).
- `frontend/` — React + TypeScript + Vite app consuming that API.
- `extension/` — browser extension that captures the LinkedIn or Indeed job posting you're viewing into the app as a Job Listing (see [`extension/README.md`](extension/README.md)).
- `brand/` — logo (full lockup + icon-only mark) and color palette; the shared identity the web app's UI is meant to match.
- `docs/adr/` — architecture decision records.

## Running the tailoring pipeline

`tailor-cv` is served by this repo's own plugin marketplace (`.claude-plugin/marketplace.json`), not `.claude/skills/`. On a fresh clone, enable it once with:

```
claude plugin marketplace add .
claude plugin install cv-reporter-skills@cv-reporter-local
```

(or set `extraKnownMarketplaces`/`enabledPlugins` in `.claude/settings.json` to auto-enable it for every trusted checkout — see ADR-0015). Once installed, invoke it in Claude Code as `/cv-reporter-skills:tailor-cv`, with a job description, a reference to an already-tracked Application, or nothing (Default Mode). To render manually once a tailored data file exists:

```
typst compile --root . template/cv.typ output/<slug>/cv.pdf --input data=output/<slug>/data.json
typst compile --root . template/cover-letter.typ output/<slug>/cover-letter.pdf --input data=output/<slug>/cover-letter-data.json
```

The skill runs the web app's automated quality checks through the app's own Go code, via `plugins/cv-reporter-skills/skills/tailor-cv/scripts/quality-check.sh` (a wrapper that builds and runs `backend/cmd/cvcheck`; needs Go on `PATH`, no running backend — ADR-0028). Before Text Review it writes the Selection + Rewrite result to `output/<slug>/selection.json` and checks every rewritten bullet's groundedness against its source bullet:

```
./plugins/cv-reporter-skills/skills/tailor-cv/scripts/quality-check.sh groundedness --selection output/<slug>/selection.json [--json]
```

and, before Visual Review, checks the rendered PDF's page count, ATS-parsability (its text layer via `pdftotext`, which must be on `PATH` for that part — otherwise it reports itself unavailable) and the assembled data's `lang`:

```
./plugins/cv-reporter-skills/skills/tailor-cv/scripts/quality-check.sh pdf --pdf output/<slug>/cv.pdf --data output/<slug>/data.json [--json]
```

Checks are advisory — exit `0` clean, `1` flagged, `2` couldn't run — and never stop the pipeline.

Adding a new job or project means adding a new Markdown file under `data/experience/` or `data/projects/` following the existing frontmatter shape — not writing code.

## Running the web app

Generation calls the Claude API directly (ADR-0005), so copy `.env.example` to `.env` and fill in `ANTHROPIC_API_KEY` first — `docker-compose.yml` loads `.env` automatically and forwards it into the backend container. Without it the app still runs (Master Data browsing works); only Generation requests fail.

```
docker-compose up
```

- Backend: `http://127.0.0.1:8080` (reads/writes `./data`; Render also reads `./template` and writes `./output` — the whole project root is mounted into the container, see ADR-0012).
- Frontend: `http://127.0.0.1:5173`

#### Claude models per call site

Each kind of Claude API request the backend makes runs on a model chosen for that kind of work (ADR-0025). Calls whose output you review — Selection, Rewrite, Cover Letter drafting — and the two web-research calls run on Claude Sonnet 5; calls that only read structured fields out of a previous call's notes, or whose result is one click from correction, run on Claude Haiku 4.5:

| Call site | Default model | What it does |
|---|---|---|
| `selection_rewrite` | `claude-sonnet-5` | Selection + Rewrite for a Generation |
| `selection_preview` | `claude-sonnet-5` | Selection-only preview |
| `cover_letter` | `claude-sonnet-5` | Cover Letter drafting |
| `ral_research` | `claude-sonnet-5` | RAL Range web research (first of two calls, ADR-0011) |
| `ral_extraction` | `claude-haiku-4-5` | RAL Range extraction from the research notes |
| `application_method_inference` | `claude-haiku-4-5` | Application Method classification |
| `contact_research` | `claude-sonnet-5` | Contact web research (first of two calls) |
| `contact_extraction` | `claude-haiku-4-5` | Contact extraction from the research notes |

To override a model, set a variable in `.env` (listed in `.env.example`, forwarded by `docker-compose.yml`) and restart the backend:

- `CV_REPORTER_MODEL_<CALL_SITE>` overrides one call site, e.g. `CV_REPORTER_MODEL_SELECTION_REWRITE=claude-opus-5` to try Opus 5 on Rewrite alone and compare at Text Review.
- `CV_REPORTER_MODEL_DEFAULT` overrides every call site at once.

A per-call-site variable wins over `CV_REPORTER_MODEL_DEFAULT`, which wins over the built-in default; an unset or empty variable means "use the default". Values go to the API unchanged — an unknown model id doesn't stop the backend starting, the affected calls fail with the API's own error.

The per-call usage breakdown keeps its existing call-type labels (`ral_estimation` and `contact_suggestion` each cover both of their calls); each recorded call's `model` shows what actually ran. The Selection preview's call is recorded too, as `selection_preview`, in the standalone usage log (it is never persisted against an Application). Cost estimates come from the hand-maintained price table in `backend/internal/claude/pricing.go` — a model missing from it estimates at $0 and logs a warning.

#### Stopping it

`docker compose down` (or Ctrl-C in a local `go run ./cmd/server`) is a graceful stop: the backend stops accepting connections, gives requests already in flight up to 15 seconds to finish, and only then exits — so a Job Listing save that is one Claude call from completing lands instead of being thrown away. Anything still running when that window expires has its request context cancelled, which stops its outbound Claude call rather than leaving it to bill for a response nobody will receive. `docker-compose.yml` sets `stop_grace_period: 30s` so Docker's 10-second default doesn't cut the drain short.

A Generation that has only just started will not survive a stop — the drain is a bound, not a queue; there is no resume. The connection limits that come with this (10s to send request headers, 60s to send a body, 120s idle keep-alive) are deliberately paired with *no* total-response cap, because a Generation legitimately runs for minutes; those routes are bounded by a 10-minute request deadline of their own instead. See `docs/adr/0021-no-total-response-timeout-bounded-drain-instead.md` for the reasoning.

#### Optional: LAN-reachable mode

By default the web app is localhost-only with no auth, per ADR-0004. To check or update Job Listings/Applications from another device on your home network (e.g. a phone), opt in explicitly by setting `BIND_ADDR` and `LAN_AUTH_TOKEN` in `.env` (see `.env.example`) and starting with the `lan` profile:

```
docker compose --profile lan up
```

This binds both services to `0.0.0.0` instead of `127.0.0.1`, and requires every `/api/*` request to carry `LAN_AUTH_TOKEN`'s value in an `X-CV-Reporter-Token` header — the backend rejects requests without it with `401`. Leaving `LAN_AUTH_TOKEN` unset (the default) skips this check entirely, so plain `docker compose up` behaves exactly as before.

This is a single static shared secret, not a login/session system — proportional to a personal, single-user tool, not a multi-user auth model. There is no TLS/HTTPS termination: the token travels in plaintext over your LAN, so only enable this on a network you trust. There's also no frontend UI yet for entering/storing the token per device — until that lands, attach the header manually from whatever client you use to reach the app over the LAN.

#### Migrating Job Listing and Application records

Every Job Listing and Application record carries a `schemaVersion` (issue #100, [ADR-0034](docs/adr/0034-record-schema-version-and-one-shot-migration.md)); records saved before it existed have none and read as the legacy version. The app keeps working on legacy records, but run the one-shot migration once to bring them current. It is a dry run by default and only writes when given `-write`:

```
cd backend
go run ./cmd/migrate-records -data-dir ../data          # dry run: report only, writes nothing
cp -r ../data ../data-backup                            # data/jobs and data/applications are gitignored
go run ./cmd/migrate-records -data-dir ../data -write   # apply
```

(No Go locally? `docker run --rm -v "$PWD":/src -w /src/backend golang:1.26 go run ./cmd/migrate-records -data-dir ../data` from the repo root, adding `-write` to apply.) `-data-dir` defaults to `$DATA_DIR`, else `./data`. Exit status: `0` nothing pending (or `-write` applied it), `3` a dry run found records still to migrate (usable as a check), `1` an error, `2` a bad flag.

What it backfills, because the value is provably on disk already:
- `freshnessStatus: not-yet-checked` on a Job Listing without one.
- `statusUpdatedAt` and a single `saved` `statusHistory` entry, both from the Job Listing's `savedAt`, on an Application **still at Status Saved** — no Status moves back to Saved, so its saved date is the date its Status was set.

What it deliberately leaves empty, and names in the report as unknowable:
- `statusUpdatedAt` and `statusHistory` on an Application past Saved. It never invents a history: the funnel's time-in-stage is computed from consecutive history entries, and the stale nudges from `statusUpdatedAt`.
- `sourceSnippetIds`, `entryIds`, `usage` and `language` on existing Generations, which stay legacy (no `schemaVersion`) for good. Newly recorded Generations are stamped, so on them an absent field means genuinely none.

Every record is read and checked before anything is written, so an unparseable record or one at a newer `schemaVersion` stops the run naming the file, with nothing changed. A Job Listing without an Application file (or the reverse) is reported, not skipped. The Job Description body is kept byte for byte, keys the migration does not know are kept, Company Logo files are never touched, each rewrite goes through a temporary file and a rename, and running it again on a migrated corpus reports nothing to do.

### Backend API

| Method | Path | |
|---|---|---|
| GET | `/api/healthz` | health check |
| GET | `/api/master-data/entries` | list Entries |
| POST | `/api/master-data/entries` | create an Entry |
| GET | `/api/master-data/entries/{id}` | get an Entry |
| PUT | `/api/master-data/entries/{id}` | update an Entry |
| DELETE | `/api/master-data/entries/{id}` | delete an Entry |
| GET | `/api/master-data/tag-lint` | scan every Entry's Tags for near-duplicate spellings (case/alias-table "confident" matches, edit-distance "suggested" matches) — read-only, never rewrites Master Data |
| GET | `/api/master-data/profile` | get profile + Static Sections |
| PUT | `/api/master-data/profile` | update profile + Static Sections |
| GET | `/api/master-data/cover-letter-snippets` | list Cover Letter Snippets |
| POST | `/api/master-data/cover-letter-snippets` | create a Cover Letter Snippet |
| GET | `/api/master-data/cover-letter-snippets/{id}` | get a Cover Letter Snippet |
| PUT | `/api/master-data/cover-letter-snippets/{id}` | update a Cover Letter Snippet |
| DELETE | `/api/master-data/cover-letter-snippets/{id}` | delete a Cover Letter Snippet |
| GET | `/api/job-listings` | list Job Listings (with their Application) as `[{jobListing, application}]`, where `jobListing` is a summary: every Job Listing field except the Job Description text, replaced by a `hasJobDescription` boolean (issue #97; fetch `/api/job-listings/{id}` for the text) — optional `sort=ral&order=asc\|desc` and `ral_min`/`ral_max`/`ral_currency` (defaults to `EUR`) query params sort/filter by RAL Range (issue #51); `n/a`/`unresolved`/`conflict` listings always trail a sort and are excluded from a filter — optional `archived=exclude\|only\|all` picks the archived view (issue #98): absent means `exclude`, so archived Job Listings are left out by default, any other value is a 400, and it composes with the `status`/`company`/`savedFrom`/`savedTo` filters and the RAL sort/filter above |
| POST | `/api/job-listings` | save a Job Listing (URL/text) — creates its Application; RAL Range + Application Method are best-effort (see ADR-0011; failures persist as `unresolved`, never block the save) |
| POST | `/api/job-listings/from-extension` | save a Job Listing captured by the browser extension (`extension/`) — same best-effort save as above |
| GET | `/api/job-listings/{id}` | get a Job Listing with its Application — the list endpoint's `{jobListing, application}` row shape (stale-Entry information included), but with the whole Job Listing, Job Description text included. Both halves carry their own `version` token (see Lost-update protection below) |
| POST | `/api/job-listings/{id}/resolve` | retry RAL Range/Application Method resolution for whatever is still `unresolved` on a Job Listing (no-op if both already resolved) |
| POST | `/api/job-listings/{id}/suggest-contact` | suggest an Application contact for a Job Listing |
| POST | `/api/job-listings/{id}/check-freshness` | on-demand check of whether a Job Listing's source URL is still live (`live`/`unreachable`/`unknown`), persisting the result and a checked-at timestamp |
| POST | `/api/job-listings/{id}/archive` | archive a Job Listing (issue #98): sets `archived: true` in its frontmatter, returning `{jobListing}` with a fresh `version`. Idempotent; never touches the Application, deletes nothing, and archived listings still count in `/api/applications/stats`. Optional `If-Match` with the Job Listing's token: 409 if the file changed since |
| POST | `/api/job-listings/{id}/unarchive` | unarchive a Job Listing, removing the `archived` key from its frontmatter and returning `{jobListing}` with a fresh `version`. Idempotent. Optional `If-Match` as on archive |
| GET | `/api/applications` | the Applications view (issue #95): every tracked Application grouped under its Status as `{total, groups: [{status, count, items: [{jobListing, application}]}]}`. Always one group per Status in pipeline order (`saved`, `tailoring`, `sent`, `interviewing`, `offer`, `rejected`, `withdrawn`), empty groups included with `count: 0` and `items: []`. Items use the `/api/job-listings` row shape (Job Listing summary, no Job Description text), version tokens included, so a row's Status move can present its Application's token. Within a group, stale Applications (`isStale`, the 14-day read-time flag) come first, then least-recently-changed `statusUpdatedAt` first; a missing or unparseable `statusUpdatedAt` sorts last. Optional `archived=exclude\|only\|all` as on `/api/job-listings`: absent means `exclude`, any other value is a 400 |
| PATCH | `/api/applications/{id}/status` | move an Application's status (state machine — see `tracking.allowedTransitions`) |
| PATCH | `/api/applications/{id}/method` | correct an Application's Application Method |
| PATCH | `/api/applications/{id}/contact` | correct an Application's contact |
| GET | `/api/applications/{id}/mailto` | get a `mailto:` link prefilled for an Application |
| POST | `/api/applications/{id}/generations` | record a Generation against an Application |
| POST | `/api/applications/{id}/notes` | add a Note (issue #96) from `{"body": "<markdown>"}`, returning 201 with the updated Application, whose `notes` come newest-first as `{id, createdAt, editedAt?, body}`. The id and `createdAt` are server-assigned; an empty or whitespace-only body is a 400, an unknown Application a 404. Never touches Status, Status history or staleness. Like the two Note routes below, takes an optional `If-Match` with the Application's token (409 if the file changed since; an empty body is still a 400 and a missing Application still a 404) and answers with a fresh `version` |
| PATCH | `/api/applications/{id}/notes/{noteId}` | correct a Note's body from `{"body": "<markdown>"}`, returning 200 with the updated Application. `createdAt` never changes; `editedAt` is set when the body actually changes. Empty body is a 400; an unknown Application or Note id a 404 |
| DELETE | `/api/applications/{id}/notes/{noteId}` | hard-delete one Note (no tombstone), returning 200 with the updated Application; an unknown Application or Note id is a 404. Deleting the last Note removes the `notes` key from the Application file |
| POST | `/api/generations` | run Selection+Rewrite (+ Cover Letter, + RAL Range if a Job Description is given) |
| POST | `/api/generations/render` | render approved Text Review content to a Tailored CV PDF (+ Cover Letter PDF); the request `slug` is a label, the response `slug` is the unique `<label>-<UTC yyyymmdd-hhmmss>[-N]` directory actually written |
| GET | `/api/usage` | running-total Claude API usage/estimated cost across every Generation plus standalone calls (RAL Range/Application Method resolution, Contact suggestion) from `data/usage-log.json` — the usage totals plus `incomplete: true` and a human-readable `incompleteReason` when some usage is known to be missing (unreadable log, or a failed log write); both fields are omitted when the total is complete (issue #102) |
| GET | `/api/generations/{slug}/{file}` | fetch a rendered file (`cv.pdf`, `cover-letter.pdf`, `cover-letter.txt`) for preview/download |
| GET | `/api/ats/{provider}/{slug}/listings` | list public job-board listings from an ATS (Greenhouse/Lever/Ashby) |
| GET | `/api/ats/tracked-boards` | list tracked ATS boards |
| POST | `/api/ats/tracked-boards` | track an ATS board |
| DELETE | `/api/ats/tracked-boards/{id}` | stop tracking an ATS board |
| GET | `/api/export` | download a zip archive of Job Listing/Application data (`data/jobs/`, `data/applications/`) — a manual backup, since that data is gitignored (ADR-0008) unlike Master Data |

**Lost-update protection (issue #89).** The web app and the `tailor-cv` skill both write the same files, so every read of an editable record (Entry, Cover Letter Snippet, Profile, Application, Job Listing) carries an opaque `version` field — a hash of the record file's bytes, computed on read and never persisted, and unrelated to the persisted `schemaVersion`. A Job Listing and its Application are two files with two tokens: `/api/job-listings` rows, the `/api/job-listings/{id}` pair and `/api/applications` items carry both. Send the token back as an `If-Match` header on the Entry/Snippet/Profile `PUT`s, the Entry/Snippet `DELETE`s, the three Application `PATCH`es (Status, Method, Contact), the three Note routes (add, edit, delete) and archive/unarchive (the Job Listing's token); the Job Listing `DELETE` takes the Job Listing's token as `If-Match` and its Application's as `Application-If-Match`. If the file changed since that read, the write is refused with `409 Conflict` and nothing is written (a record that is gone is still `404`). The check runs inside the store right before its atomic write. The headers are optional: a request without them writes unconditionally, so the skill and other non-FE callers are unaffected. Every write that honours them answers with fresh tokens, as do the routes that stay unconditional: recording a Generation and the server-computed Job Listing routes (save, resolve, check-freshness).

## Licence

Sumisura is licensed under the [GNU AGPL-3.0](LICENSE).

**Your output is yours.** The CVs, cover letters and application records you
produce with it are your own work, not derivative works of this software: the
AGPL places no obligation on them, and neither the maintainer nor this project
claims any right over your Master Data or anything rendered from it.

The **name and logo** are trademarks and are not covered by the AGPL — see
[`TRADEMARKS.md`](TRADEMARKS.md). Fork freely; give your fork its own name.

Third-party code and fonts keep their own licences, listed in
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the dev loop, the test and lint
commands, the API contract fixtures, and how architecture decisions get
recorded. Bug reports and questions are welcome; by contributing you agree to
the [Code of Conduct](CODE_OF_CONDUCT.md).

## Further reading

- [`CONTEXT.md`](CONTEXT.md) — domain vocabulary (ubiquitous language)
- [`docs/adr/`](docs/adr/) — architecture decision records
- [`extension/README.md`](extension/README.md) — how the LinkedIn/Indeed capture extension works and how to load it
- [`brand/palette.md`](brand/palette.md) — the color palette behind the logo, applied across the frontend's shadcn/ui theme
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — dev loop, tests, lint, contract fixtures
- [`SECURITY.md`](SECURITY.md) — how to report a vulnerability
