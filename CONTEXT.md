# CV Reporter

A personal tool that turns one person's complete career history into a tailored, one-page CV (PDF), generated on demand for a specific job application.

## Language

**Master Data**:
The complete, untailored superset of everything the user has ever done, plus the Static Sections. The single source of truth from which every generated CV is derived. Never itself shown to an employer as-is.
_Avoid_: config, profile, source data

**Entry**:
One unit of work experience or project, stored as its own file (YAML frontmatter for dates/Tags, Markdown body for bullets). The unit of Selection during Tailoring.
_Avoid_: item, record, job (ambiguous with Job Description)

**Client Engagement**:
An Entry representing the work done for one client while at a consultancy employer (e.g. Quantyca). Several Client Engagement Entries can share the same employer/title/date-range metadata but are independently selectable — Selection can surface one client's work prominently while omitting another's.
_Avoid_: sub-project, client project

**Tag**:
Metadata on an Entry (tools/technologies used) recorded in its frontmatter. Drives both Selection's relevance-matching and the derived Tech Stack line. Never independently maintained as a separate list.
_Avoid_: skill, keyword, label

**Tech Stack**:
The CV section listing technologies, derived entirely from the Tags of whichever Entries were Selected for a given Generation. Has no data of its own.
_Avoid_: skills list, skills section

**Static Section**:
Content (Awards, Activities, spoken Languages, Publications) that is always included in full on every generated CV. Never subject to Selection or Rewrite.
_Avoid_: extras, misc

**Job Description**:
The pasted text or fetched-URL content describing a role, supplied as input to a Generation to guide Tailoring. Its absence triggers Default Mode.
_Avoid_: posting, job ad

**Generation**:
The end-to-end pipeline that turns Master Data plus an optional Job Description into a Tailored CV: Selection, Rewrite, Text Review, Render, Visual Review.
_Avoid_: pipeline, build, run

**Tailoring**:
The part of a Generation driven by a Job Description: Selection followed by Rewrite, constrained so the result fits one page.
_Avoid_: customization

**Selection**:
The Tailoring step that chooses which Entries, and which of their bullets, to include and in what order, based on relevance to the Job Description.
_Avoid_: filtering, curation

**Rewrite**:
The Tailoring step that adjusts an Entry's bullet phrasing to better match a Job Description's language. Must not introduce facts absent from Master Data.
_Avoid_: rephrasing, editing

**Default Mode**:
A Generation run without a Job Description. Selection falls back to the most recent/representative Entries; Rewrite is skipped since there is no Job Description to tailor phrasing toward.
_Avoid_: generic CV, general CV (that's the output; this is the mode that produces it)

**Text Review**:
The first Human-in-the-Loop checkpoint: the user approves or corrects the Tailoring output (Selection + Rewrite) as text, before Render.
_Avoid_: draft review

**Render**:
The Typst compilation step that turns approved Tailoring output into a PDF.
_Avoid_: compile, build

**Visual Review**:
The second Human-in-the-Loop checkpoint: the user checks the rendered PDF for layout issues (overflow, bad page breaks) after Text Review is approved.
_Avoid_: final check, PDF review

**Tailored CV**:
The final one-page PDF produced by a Generation, for a specific Job Description or from Default Mode. A derived artifact, not versioned in the repo.
_Avoid_: resume, output, generated CV
