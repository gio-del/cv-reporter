# Changelog

## 0.1.0 (2026-09-15)

The first public release of **Sumisura** — self-hosted CV tailoring that keeps
every rewritten bullet grounded in what you actually did.

The project was previously called CV Reporter and was never released. This
release is the point at which it becomes something another person can install:
it has a licence, a name, documentation and a version.

### What you get

* **Tailoring.** `/sumisura:tailor-cv` walks Selection → Rewrite → Text Review
  (you approve) → Render → Visual Review (you approve), and produces a one-page
  CV and a matching cover letter for one specific job.
* **A groundedness check** that compares every rewritten bullet against the
  Entry it came from, before you are asked to approve anything — plus page
  count, ATS-parsability and language checks on the rendered PDF.
* **Application tracking**: Job Listings and Applications with status, notes,
  contacts, salary ranges and the Generations produced for each, fed by paste,
  by Greenhouse/Lever/Ashby boards, or by the browser extension on LinkedIn and
  Indeed.
* **Documentation** at https://gio-del.github.io/sumisura/ — quickstart, Master
  Data, tailoring, configuration, LAN mode, upgrading, troubleshooting.
* **AGPL-3.0**, with the name and logo kept as trademarks, and an explicit
  statement that the CVs and cover letters you generate are yours.

### ⚠ BREAKING CHANGES

* **rename to Sumisura** ([#145](https://github.com/gio-del/sumisura/pull/145)).
  Only affects an installation that predates this release: environment
  variables moved from `CV_REPORTER_MODEL_*` to `SUMISURA_MODEL_*`, the LAN
  header is now `X-Sumisura-Token`, and the skill is installed as
  `sumisura@sumisura-local` and invoked as `/sumisura:tailor-cv`. No
  compatibility fallbacks — see
  [ADR-0036](https://github.com/gio-del/sumisura/blob/main/docs/adr/0036-renamed-to-sumisura.md).

### Features

* add the landing page and docs site (Astro + Starlight) ([#147](https://github.com/gio-del/sumisura/issues/147)) ([953fe52](https://github.com/gio-del/sumisura/commit/953fe52e071dbb39b98d4d1e83a85631c93b1557))
* license under AGPL-3.0 and add the community health files ([#144](https://github.com/gio-del/sumisura/issues/144)) ([f638231](https://github.com/gio-del/sumisura/commit/f638231d83881bb2f79597e5f234fba535c42899)), closes [#135](https://github.com/gio-del/sumisura/issues/135)
* redesign the brand identity for Sumisura ([#143](https://github.com/gio-del/sumisura/issues/143)) ([5a0fb9d](https://github.com/gio-del/sumisura/commit/5a0fb9dd5d690cb6a0df422294597cce2bc370db)), closes [#134](https://github.com/gio-del/sumisura/issues/134)
* rename to Sumisura ([#145](https://github.com/gio-del/sumisura/issues/145)) ([bffdd59](https://github.com/gio-del/sumisura/commit/bffdd593f28d61cb22867f914cc91f5a6e8fb8bf)), closes [#136](https://github.com/gio-del/sumisura/issues/136)

### Install

See the [quickstart](https://gio-del.github.io/sumisura/quickstart). You need
Docker, an Anthropic API key, Claude Code, and the `typst` CLI. The browser
extension is attached to this release as a zip; load it unpacked.
