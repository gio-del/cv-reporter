# ADR-0038: All Master Data is local-only; the repo ships `data/examples/`

- Status: accepted
- Date: 2026-09-16
- Extends: [ADR-0037](0037-profile-is-local-only.md)

## Context

ADR-0037 made `data/profile.yaml` gitignored, reasoning that it is "the one
file holding the user's identity" while "every other piece of Master Data
describes *work*".

That split does not survive contact with a real career. The Entry files name
the employer, every client, the projects those clients paid for, and what was
built for them. A consultancy's client list is commercially sensitive, and an
engagement that was never announced publicly is not the author's to publish.
The same goes for Cover Letter Snippets, which are first-person prose about
motivation and salary-adjacent context.

The repo's own history shows the failure mode: real employer and client names
were committed in the first commit, then removed a month later by
`8852a6d6` ("Scrub personal data from master data, replace with stub
examples") — a scrub that cannot undo a public push, and that also left one
invented bullet behind in the stub it created.

The stubs still need to exist. A self-hoster cloning this repo has to see the
shape of an Entry, and the app has to have something to render on first run.
But "the tracked file is also the file you edit" is precisely the trap
ADR-0037 identified.

## Decision

**Every file under `data/` that describes the installation owner is
gitignored. The repo tracks stub copies under `data/examples/` instead.**

```
data/examples/profile.yaml                 → data/profile.yaml
data/examples/experience/*.md              → data/experience/*.md
data/examples/projects/*.md                → data/projects/*.md
data/examples/cover-letter-snippets/*.md   → data/cover-letter-snippets/*.md
```

Setup is one copy:

```sh
cp -r data/examples/. data/
```

`data/profile.example.yaml` moves to `data/examples/profile.yaml` so there is
one convention and one setup command rather than two. ADR-0037's decision
stands unchanged in substance; only the path moves.

No code changes. `masterdata.ListEntries` and `masterdata.ListSnippets`
already treat a missing directory as an empty list (`os.IsNotExist` → `nil,
nil`), so a fresh clone with no copy performed yet reads as "no Entries", the
same ordinary empty state as a fresh clone with no `profile.yaml`.

## Consequences

- Nothing in `data/` that a self-hoster edits is ever staged by `git add -A`.
  The prompt to "replace the stubs with your own history" no longer means
  "edit a tracked file".
- The examples are now genuinely examples: they can be edited to teach the
  format without anyone's real history being the teaching material.
- Master Data loses git history. It was never much of a benefit — these are
  append-mostly files, and the app shows `lastModified` from git only when
  history exists — but `masterdata.EntryLastModified` will now return nil for
  Entries in a normal install, and the UI already handles that.
- Backing up Master Data becomes the user's problem, the same as it already
  was for `data/jobs/`, `data/applications/` and `profile.yaml`. A private
  repo or a synced folder covers it.
- Upgrading an existing install: move your own files out of the way, pull,
  and keep using them in place. Nothing needs to be regenerated, because the
  paths the app and the skill read are unchanged.
