---
title: How it works
description: The vocabulary Sumisura uses, and where its no-invented-facts guarantee actually lives.
---

## Master Data, and everything downstream of it

**Master Data** is the complete, untailored superset of everything you have
done: one file per **Entry** (a job, a client engagement, a project), plus
`profile.yaml` for contact details and the sections that are always included
verbatim — education, publications, awards, languages.

Master Data is never shown to an employer as-is. Every generated document is
derived from it:

1. **Selection** picks the Entries worth including for one **Job Description**.
   A one-page CV cannot hold a whole career, so this is a real decision, made
   per job.
2. **Rewrite** re-words the selected bullets toward the job's vocabulary. It may
   change emphasis, order and phrasing. It may **not** add a fact — no new tool,
   metric, scope or responsibility.
3. **Text Review** is yours. Nothing renders until you approve the text.
4. **Render** produces the **Tailored CV** (and a cover letter) with Typst.
5. **Visual Review** is yours too: the PDF must be one page and must survive an
   ATS text extraction.

## Where the guarantee lives

"The AI never invents experience" is not a prompt-level promise. Before you are
asked to approve anything, a **groundedness check** compares every rewritten
bullet against the source bullet it came from and flags anything that introduced
a fact. The same check runs whether you use the web app or the skill — both call
the same Go code (`cvcheck`).

The checks are advisory: they surface a verdict at a checkpoint where a human is
already looking. They never silently rewrite your text, and they never block you
from shipping a document you have read.

## Applications, and what a Generation is

A saved job becomes a **Job Listing** (the posting: company, title, description,
URL, salary range where it can be resolved) with an **Application** attached
(status, notes, contact, history). Each tailoring run recorded against an
Application is a **Generation**: which Entries were used, which model ran, what
it cost, and where the files landed.

Deleting `output/` is safe. The records survive, and the app marks the files as
no longer on disk.

## Two ways in, one set of rules

The **web app** and the **`tailor-cv` skill** read and write the same files
under `data/`, and run the same quality checks. Use the app for browsing,
editing and tracking; use the skill for the tailoring conversation itself.
