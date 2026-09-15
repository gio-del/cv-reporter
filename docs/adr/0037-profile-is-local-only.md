# ADR-0037: The user's profile is local-only; the repo ships an example

- Status: accepted
- Date: 2026-09-16

## Context

`data/profile.yaml` holds the parts of a CV that identify a person: full name,
email, phone number, location, LinkedIn and GitHub handles, education,
publications, awards. Every other piece of Master Data describes *work*;
this one describes *who you are*.

It was tracked in git, with the stub content (Jane Doe) carrying a comment
warning the reader not to put real details in it because the repo is public.
That warning is the whole problem: the tool's first real setup step is
"replace the stubs with your own history", and the one file you must not
replace honestly is the one the app asks you for first. Sooner or later
somebody commits their home address.

The rest of the personal corpus is already handled: `data/jobs/` and
`data/applications/` are gitignored (ADR-0008) because they describe a job
search rather than a career, and `output/` is derived. `profile.yaml` was the
odd one out.

## Decision

**`data/profile.yaml` is gitignored. The repo tracks
`data/profile.example.yaml` instead**, holding the same stub content it always
had. Setup copies the example once:

```sh
cp data/profile.example.yaml data/profile.yaml
```

A missing `profile.yaml` is therefore an ordinary state of a fresh clone, not a
fault, and it is reported as one: `masterdata.GetProfile` returns
`ErrNoProfile`, whose message names the copy command, and the API answers
`404` rather than `500`.

Two alternatives were rejected:

- **A `profile.local.yaml` that overrides a tracked `profile.yaml`.** Two files
  for one record, and the app's own `PUT /api/master-data/profile` would have
  to decide which one it writes — a fork in the write path for no gain.
- **`git update-index --skip-worktree`.** Invisible local state that survives
  in one clone and not another, and that silently reverts on a fresh checkout.

The Entry files under `data/experience/` and `data/projects/` stay tracked.
They are the part a user might reasonably want in version control — history,
diffs, review — and they describe work rather than identity. A user who wants
those private too can gitignore them in their own fork; nothing in the app
depends on them being tracked.

## Consequences

- An existing clone keeps working untouched: the file is already on disk and
  simply stops being tracked. `git rm --cached data/profile.yaml` is the only
  cleanup, and only if it was committed with real content — in which case the
  history still holds it, and rewriting that history is the user's call.
- The first-run experience gains one copy command, documented in the README
  quickstart and the docs site.
- Screenshots and demos keep working from the example file.
- The tailoring pipeline is unaffected: it reads `data/profile.yaml` exactly as
  before.
