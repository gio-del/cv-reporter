---
title: Contributing
description: Where the contributor documentation lives and how decisions get recorded.
---

Contributor docs live in the repository, next to the code they describe:

- **[`CONTRIBUTING.md`](https://github.com/gio-del/sumisura/blob/main/CONTRIBUTING.md)**
  — dev loop, test and lint commands, the API contract fixtures, PR title format,
  the CLA.
- **[`CONTEXT.md`](https://github.com/gio-del/sumisura/blob/main/CONTEXT.md)** —
  the domain vocabulary. Read it before naming anything.
- **[`docs/adr/`](https://github.com/gio-del/sumisura/tree/main/docs/adr)** —
  architecture decision records: why the app is localhost-only, why rendering
  shells out to Typst, why the frontend types are held to the backend by
  fixtures, and so on. Read the relevant one before proposing to change
  something structural.
- **[`SECURITY.md`](https://github.com/gio-del/sumisura/blob/main/SECURITY.md)**
  — how to report a vulnerability, and which documented designs are not
  vulnerabilities.

Nothing is duplicated here on purpose: documentation that lives beside the code
is the documentation that gets updated with the code.

## The short version

- Open an issue before anything large.
- Branch from `main`; PR titles are Conventional Commits (CI checks this).
- A structural decision gets an ADR in the same PR.
- User-facing behaviour changes update these docs in the same PR.
- Sign the CLA on your first PR — it keeps the project's licensing options open
  while your work stays available to everyone under the AGPL.
