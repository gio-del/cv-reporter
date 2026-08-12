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
The pasted text or fetched-URL content describing a role, held by a Job Listing, supplied as input to a Generation to guide Tailoring. Its absence triggers Default Mode.
_Avoid_: posting, job ad

**Job Listing**:
A persisted, tracked record of a role the user is considering: source (pasted, browser-extension capture, or ATS feed), company, URL, saved date, its Job Description, and RAL Range. Distinct from Job Description itself, which is just the text/content field it holds.
_Avoid_: posting, job ad, listing

**RAL Range**:
The gross annual salary (Reddito Annuo Lordo) range for a Job Listing, with a source: Stated (found in the Job Description), Estimated (Claude web-researched it for that role/company/location — a guess, not a fact), or N/A (couldn't find anything). Always shown in the FE, source labeled.
_Avoid_: salary, salary range, pay

**Application**:
The tracked record of one attempt to apply to a Job Listing (exactly one Application per Job Listing), created the moment it's saved: its Status, Application Method, Contact (if applicable), and a history of every Generation run for it — the most recent Tailored CV and Cover Letter being what you'd actually send.
_Avoid_: submission

**Status** (of an Application):
Where an Application stands: Saved → Tailoring → Sent → Interviewing → Rejected/Offer.
_Avoid_: state, stage

**Application Method**:
How a Job Listing says to apply, inferred by Claude from its Job Description (portal link, email, LinkedIn Easy Apply, other) and correctable by the user. Determines which guidance the FE surfaces — e.g. a Contact/email draft only for the email method.
_Avoid_: apply type

**Contact**:
The recruiter/hiring-manager name and email for an email-method Application. Entered manually by the user or suggested by Claude via web search and confirmed by the user — never trusted unconfirmed.
_Avoid_: recruiter

**Cover Letter**:
A Generation output alongside the Tailored CV. Selected/adapted from a user-authored library of Cover Letter Snippets in Master Data when one exists; otherwise freshly generated prose grounded in Master Data and the Job Description, under the same no-invented-facts constraint as Rewrite. Reviewed at Text Review like the CV.
_Avoid_: motivation letter

**Cover Letter Snippet**:
Optional Master Data: one reusable cover-letter paragraph (e.g. an opening, a why-this-company, a closing), stored one-per-file like an Entry (YAML frontmatter with a kind and Tags, Markdown body). Selected/lightly rewritten during Cover Letter generation the same way Entries are selected for a CV; if none exist, Cover Letter generation falls back to fresh prose.
_Avoid_: template, boilerplate

**Generation**:
The end-to-end pipeline that turns Master Data plus an optional Job Description into a Tailored CV and Cover Letter: Selection, Rewrite, Text Review, Render, Visual Review.
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
