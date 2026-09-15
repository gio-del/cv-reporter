---
title: Tracking applications
description: Job Listings, Applications, statuses, notes and generations — the part of Sumisura that is not about writing.
---

Tailoring a CV for a job you never record is how job searches get lost. Every
job you save becomes a **Job Listing** with an **Application** attached.

## Saving a job

- **Paste** a URL or the description text into the web app.
- **Pull from an ATS board** — Greenhouse, Lever and Ashby expose public job
  boards; track a board and browse its listings in the app.
- **Capture what you are reading** with the [browser extension](./extension.md)
  on LinkedIn or Indeed.

On save, Sumisura does some best-effort work: it resolves a salary range where
the posting or public sources allow, infers how one applies (ATS form, email,
referral), and downloads the company logo. Any of these can come back
`unresolved` without blocking the save, and you can retry later.

## The Application

Each Application carries:

- a **status** through a defined pipeline: saved → tailoring → sent →
  interviewing → offer, plus rejected and withdrawn
- **timestamped notes** you can edit and delete
- a **contact**, which can be suggested for you and corrected by hand
- every **Generation** produced for the job, with the model used and its cost
- a **freshness** check: is the posting still live?

The Applications view groups everything by status, and floats stale
applications — nothing has moved for two weeks — to the top of their group.

## Archiving

Archive a Job Listing to get it out of the default view without deleting
anything. Archived listings still count in your statistics and can be brought
back.

## Your records are yours

`data/jobs/` and `data/applications/` are gitignored flat files: they are about
your job search, not your career history. Use **Export** in the app to download
a zip of both — that is the backup path, since git is deliberately not carrying
them.
