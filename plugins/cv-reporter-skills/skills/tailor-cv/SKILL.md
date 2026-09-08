---
name: tailor-cv
description: Generate a tailored, one-page CV (PDF) from this repo's Master Data, for a specific job (paste text or a URL), for a role already tracked as a Job Listing/Application in the web app (by company or job title), or, with no job given, a general-purpose Default Mode CV. Use when the user wants to apply for a job, update their CV, or asks to run/build/tailor a CV.
---

# Tailor CV

Read `CONTEXT.md` at the repo root first — it defines the vocabulary used below (Master Data, Entry, Client Engagement, Tailoring, Selection, Rewrite, Static Section, Generation, Tailored CV, Job Listing, Application, Status). Follow ADRs under `docs/adr/`.

## Inputs

Work out which of three modes this run is in, in this order:

1. **Targeting a tracked Application** — the user refers to a company/role they've already saved via the web app or the LinkedIn extension (e.g. "generate a CV for the Acme Corp application", "tailor for the Senior Backend Engineer role at Acme"). Go to "Targeting a tracked Application" below before starting the Pipeline.
2. **Job Description given directly** — pasted text, or a URL (fetch it).
3. **Default Mode** — no input at all.

If it's genuinely unclear whether the user means a tracked Application or is just naming a company as part of pasted/URL text, ask rather than guessing.

## Targeting a tracked Application

This mode needs the backend running (`docker-compose up` — `http://127.0.0.1:8080`); the other two modes don't. If any backend call below fails for any reason (connection refused, 404, unexpected status), stop immediately with a clear message to the user — do not fall back to treating their company/title text as literal Job Description text, and do not proceed to Render believing a Generation is tracked when it isn't.

1. **Look up the Application.** Run:
   ```
   curl -sf http://127.0.0.1:8080/api/job-listings
   ```
   This returns every tracked Job Listing paired 1:1 with its Application: `[{ "jobListing": {...}, "application": {...} }, ...]`. A connection error or non-2xx here means the backend isn't running — stop and tell the user to run `docker-compose up`. Otherwise, match the user's text case-insensitively against each entry's `jobListing.company` and `jobListing.title`.
   - **No match**: stop and tell the user you couldn't find it — don't silently fall through to Default Mode or treat their text as a Job Description.
   - **One match**: proceed with it.
   - **More than one match**: list the candidates (company, title, `application.status`) and ask the user which one they mean before doing anything else.

   `jobListing` and `application` share the same `id` (they're 1:1) — keep it, it's needed in steps 4–5 below.

2. **Use its Job Description.** The matched `jobListing.jobDescription` is this run's Job Description — feed it into Selection/Rewrite in the Pipeline below exactly as pasted/URL text would be. For the Pipeline's `<slug>` (step 5), default to a kebab-case slug of the company name (e.g. `acme-corp`) unless the user prefers another.

3. Run the Pipeline below in full (Load Master Data → Selection → Rewrite → Text Review → Assemble → Render → Visual Review) — nothing about it changes for this mode. Come back here once Visual Review is approved.

4. **Record the Generation.** POST the rendered result to the same endpoint the web app's own Generate button uses, so it appears in the Application's history there (`<id>` is the id from step 1):
   ```
   curl -sf -X POST http://127.0.0.1:8080/api/applications/<id>/generations \
     -H 'Content-Type: application/json' \
     -d '{"slug": "<slug>", "cvPath": "output/<slug>/cv.pdf"}'
   ```
   Omit `coverLetterPath` — this skill doesn't draft Cover Letters. A connection error or non-2xx response means the Generation was **not** recorded — stop and tell the user; don't report the run as complete.

5. **Offer the Status move.** If `application.status` (from step 1) was `"saved"`, ask the user whether to move it to `"tailoring"`. If they say yes:
   ```
   curl -sf -X PATCH http://127.0.0.1:8080/api/applications/<id>/status \
     -H 'Content-Type: application/json' \
     -d '{"status": "tailoring"}'
   ```
   If `application.status` was already past `"saved"` (`tailoring`, `sent`, `interviewing`, `rejected`, `offer`), skip this ask entirely — don't touch Status.

## Pipeline

1. **Load Master Data.** Read `data/profile.yaml` (contact info + Static Sections: education, publications, awards, activities, languages — always included verbatim, never selected or rewritten) and every Entry file under `data/experience/*.md` and `data/projects/*.md` (YAML frontmatter + Markdown bullets).

2. **Selection.** Choose which Entries, and which of their bullets, are relevant to the Job Description, and in what order. In Default Mode, favor the most recent and most representative Entries (e.g. the `flagship: true` Entry) instead of matching a Job Description. The result must be trimmable enough to satisfy the one-page constraint in step 4 — err on the side of cutting a marginal Entry or bullet rather than keeping everything.

3. **Rewrite.** Adjust bullet phrasing to better match the Job Description's language and emphasis. Do not introduce facts, tools, or claims that aren't present in the source Entry — Rewrite may reword, not invent.

4. **Text Review (HITL, required).** Present the Selection + Rewrite result to the user as text — what was kept, dropped, reordered, and reworded (a diff against the source bullets is more useful than just the final text). Wait for approval or corrections before rendering. Do not proceed to Render until the user explicitly approves.

5. **Assemble the data file.** Merge the approved tailored content with the static parts of `data/profile.yaml` into a single JSON object matching the shape `template/cv.typ` expects (see the comment at the top of that file): `name`, `location`, `email`, `phone`, `linkedin`, `github`, `education`, `experience` (grouped-by-employer array, each item has `employer`, `role`, `client`, `location`, `start`, `end`, `bullets`), `projects`, `tech_stack` (derived — collect the `tags` of every selected Entry, deduplicated), `publications`, `awards`, `activities`, `languages`. Write it to `output/<slug>/data.json`, where `<slug>` is a short kebab-case name for this Generation (e.g. the company applied to, or `default`).

6. **Render.** Run:
   ```
   typst compile --root . template/cv.typ output/<slug>/cv.pdf --input data=output/<slug>/data.json
   ```

7. **Visual Review (HITL, required).** Tell the user the PDF is ready at `output/<slug>/cv.pdf` and ask them to check it — page count (must be one page), overflow, awkward breaks. If they report an issue, fix the data or trim Selection and re-render; don't guess silently. If this run is targeting a tracked Application, once approved here go back to "Targeting a tracked Application" step 4 to record it.

## Notes

- `output/` is gitignored — it's derived, not Master Data. Never edit it as if it were a source of truth.
- To add new Master Data (a new job, a new project), create a new file under `data/experience/` or `data/projects/` following the frontmatter shape of the existing files — this is normal manual editing, not something this skill automates.
- This skill never drafts Cover Letters, edits Master Data, or captures new Job Listings — those stay website/extension-only (see the FE's own Generate button, ADR-0005).
