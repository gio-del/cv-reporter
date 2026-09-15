# ADR-0036: Renamed to Sumisura; "CV" stays the domain term

- Status: accepted
- Date: 2026-09-15

## Context

The project shipped for a year as `cv-reporter` / "CV Reporter". The name was a
working title: descriptive, unmemorable, and inaccurate — the tool does not
*report* on CVs, it tailors them. The repo was also public with no licence, no
description, no topics, no releases and no site, which made it read as a private
scratch project rather than something anyone could run.

Two positioning decisions came with the rename (grilling session, 2026-09-14):

1. The project is **self-hostable by other people** — someone can clone it, drop
   in their own Master Data and run it — rather than "a personal tool, not a
   product pitched at other users", which is what the README and `CONTEXT.md`
   used to say. A hosted version is a possible future, not a commitment; nothing
   here changes ADR-0004's localhost-first, no-accounts design.
2. The differentiator is **groundedness**: every rewritten bullet is checked
   against the Entry it came from, and no draft reaches a PDF without human
   approval. "AI resume builder" is a crowded category whose main complaint is
   invented experience.

The new name is **Sumisura**, from the Italian *su misura*, "made to measure" —
the tailoring metaphor the product is built on, and the source of the
tape-measure mark in `brand/`.

## Decision

**Rename everything that carries the old name, as a clean break, before the
first release.** The module path, the `SUMISURA_MODEL_*` environment variables,
the `X-Sumisura-Token` LAN header, the plugin (`/sumisura:tailor-cv`), the
extension and its injected CSS classes, and all prose. No compatibility
fallbacks: no dual-read of old environment variable names, no acceptance of the
old header. There are no releases and no installs other than the maintainer's,
so the cost is one `.env` edit and one plugin reinstall; after a release the
same rename would be a breaking change requiring migration notes.

**Keep "CV" as the domain term.** *Tailored CV*, `template/cv.typ`,
`cmd/cvcheck` and the `tailor-cv` skill keep their names, and `CONTEXT.md`'s
vocabulary is unchanged. "Resume" is used only on marketing surfaces — the repo
description, topics, and the landing page — because that is the word people
search for, particularly in the US.

**Do not rewrite historical ADRs.** ADRs 0001–0035 keep the old names and paths
they were written with; they are a record of decisions at a point in time, not
current documentation. This ADR is the pointer that explains the discrepancy.

## Consequences

- Anyone with a checkout re-points the remote (GitHub redirects the old URL),
  renames their `CV_REPORTER_MODEL_*` keys to `SUMISURA_MODEL_*`, reinstalls the
  plugin from the renamed marketplace, and reloads the extension.
- Historical ADRs mention `plugins/cv-reporter-skills/`, `cv-reporter-local` and
  `X-CV-Reporter-Token`. That is expected: read them against this ADR.
- The extension's Gecko id changed, so Firefox treats it as a new add-on. It
  stores nothing, so nothing is lost; the id will be fixed for good before the
  first AMO submission.
- A test (`TestLANAuth_AcceptsRenamedHeaderOnly`) pins the clean break, so the
  old header cannot quietly come back as a fallback.
- The repository moved to `github.com/gio-del/sumisura`. A `sumisura`
  organisation and the `sumisura.io` domain are deliberately deferred; moving
  there later changes the module path again, which is internal churn only —
  nothing outside this repo imports the backend module.
