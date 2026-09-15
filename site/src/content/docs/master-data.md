---
title: Writing your Master Data
description: The file shapes behind every tailored CV, and how to write Entries that select well.
---

Master Data is plain files in your checkout. Edit them in the web app or in your
editor; both write the same thing.

## `data/profile.yaml`

Contact details plus the **Static Sections** — education, publications, awards,
activities, languages. These are always included in full, never selected or
rewritten.

```yaml
name: Jane Doe
location: Example City, Country
email: jane.doe@example.com
phone: "+1 555 0100"
linkedin: janedoe
github: janedoe
```

## `data/experience/*.md` and `data/projects/*.md`

One file per Entry: YAML frontmatter, then Markdown bullets.

```markdown
---
employer: Example Consulting S.p.A.
role: Data Engineer
client: Example Client A
location: Example City
start: "2024-10"
end: null            # null means "current"
flagship: true
tags:
  - AI Platform
  - MCP
---

- Built the ingestion service feeding the recommendation models, moving it
  from nightly batches to streaming.
- Led the migration of 40+ dashboards onto the new semantic layer.
```

### One file per client engagement

A consultancy role with three clients is **three Entries**, not one. Selection
works at Entry level, so splitting lets one client's work be surfaced for a job
where it is relevant while the others stay out.

### Tags matter

`tags` are the raw material for the Tech Stack section and a strong signal
during Selection. Keep spellings consistent — the app has a tag-lint view that
finds near-duplicates ("Postgres" vs "PostgreSQL").

### Write bullets you would defend

Rewrite may re-word a bullet, never invent one. A bullet with a real number in
it can be re-emphasised; a bullet with no number will never grow one. If a
result is worth claiming, put the evidence in Master Data.

## `data/cover-letter-snippets/*.md`

Optional. Reusable paragraphs — an opening, a closing, a "why this company"
pattern — that cover-letter drafting can draw on.

```markdown
---
kind: opening
tags: [data-engineering]
---

I have spent the last four years making data pipelines boring: predictable,
observable, and cheap to run.
```

## What is not Master Data

`output/` is derived (safe to delete), and `data/jobs/` and
`data/applications/` are your tracking records — gitignored, because they are
about your job search rather than your history.
