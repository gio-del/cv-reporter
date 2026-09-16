---
title: Upgrading
description: How versions work, and what each release asks of you.
---

One version covers the whole product — backend, frontend, extension and plugin.
They are only guaranteed to work together at the same commit, so they ship
together.

Before 1.0, **a breaking change bumps the minor version**, and "breaking" means
*you* have to do something: a renamed environment variable, a record migration,
a removed route, a plugin reinstall. Release notes say so explicitly.

## The general upgrade

```sh
git fetch --tags
git checkout v0.2.0            # whatever the newest release is
docker compose build           # rebuild the images
docker compose up
```

Then, if the release notes mention the skill or the extension:

```sh
claude plugin install sumisura@sumisura-local   # reinstall the skill
```

and reload the unpacked extension in your browser.

Back up first if you are nervous. `data/jobs/` and `data/applications/` are not
in git — use **Export** in the app, or copy the `data/` directory.

## Migrating records

Some releases add fields to Job Listing and Application records. Records written
earlier read as legacy and keep working, but a one-shot migration brings them
current. It is a dry run unless you pass `-write`:

```sh
cd backend
go run ./cmd/migrate-records -data-dir ../data          # report only
cp -r ../data ../data-backup                            # these files are not in git
go run ./cmd/migrate-records -data-dir ../data -write   # apply
```

Exit codes: `0` nothing pending, `3` a dry run found work to do, `1` an error.

It only backfills what is provably already on disk, and names anything it cannot
know rather than inventing it. Every record is read and validated before
anything is written, so a corrupt or newer-than-expected record stops the run
with nothing changed.

## Version-specific notes

### Unreleased — Master Data became local-only

All Master Data is now gitignored, not just `data/profile.yaml`:
`data/experience/`, `data/projects/` and `data/cover-letter-snippets/` joined it,
and the stubs moved to `data/examples/`
([ADR-0038](https://github.com/gio-del/sumisura/blob/main/docs/adr/0038-all-master-data-is-local-only.md)).

Your files are read from the same paths as before, so **nothing moves on your
disk and nothing needs regenerating**. After pulling, git simply stops tracking
them. Two things to know:

- If your own Master Data was committed to a fork or a private remote, the
  history is still there. Removing it takes a history rewrite, not this upgrade.
- `data/` is now entirely yours to back up. A private repo or a synced folder
  covers it.

### 0.1.0 — first public release

Nothing to migrate. If you were running the project before it was renamed from
CV Reporter, see the rename notes in the release: environment variables changed
from `CV_REPORTER_MODEL_*` to `SUMISURA_MODEL_*`, the LAN header became
`X-Sumisura-Token`, and the plugin is now installed as
`sumisura@sumisura-local` and invoked as `/sumisura:tailor-cv`.
