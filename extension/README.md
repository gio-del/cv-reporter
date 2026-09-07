# CV Reporter — LinkedIn Capture

A browser extension that captures the LinkedIn job posting you're currently viewing into your local CV Reporter app as a Job Listing. See `docs/adr/0007-job-sourcing.md` for why this is scoped to reading a page you're already looking at, never scraping.

## How it works

- A "Save to CV Reporter" button appears (bottom-right) on any `linkedin.com/jobs/*` page.
- Clicking it reads the Job Title, company, location, Company Logo, and Job Description (as Markdown) already rendered on the page — no request to LinkedIn is made by the extension.
- That content is sent to your local backend (`POST http://localhost:8080/api/job-listings/from-extension`), which saves it as a Job Listing the same way a manually-pasted one is saved (Company Logo downloaded, RAL Range looked up, Application Method inferred, Application created at Saved).
- The button shows a success/failure message after each attempt.
- Nothing happens automatically in the background — only an explicit click triggers a capture.

Requires the CV Reporter backend running locally (`docker-compose up` from the repo root; see the root `README.md`).

## Loading it (unpacked, for local personal use — not published to any store)

**Chrome / Chromium-based:**

1. Open `chrome://extensions`.
2. Enable **Developer mode** (top-right toggle).
3. Click **Load unpacked** and select this `extension/` directory.
4. Visit any LinkedIn job posting page — the "Save to CV Reporter" button should appear.

**Firefox:**

1. Open `about:debugging#/runtime/this-firefox`.
2. Click **Load Temporary Add-on…** and select `extension/manifest.json` (the manifest file itself, not the folder).
3. Visit any LinkedIn job posting page — the "Save to CV Reporter" button should appear.

Note: Firefox unloads temporary add-ons when the browser restarts — you'll need to reload it each session. `manifest.json` declares both `background.service_worker` (Chrome) and `background.scripts` (Firefox) so the same extension works unmodified in both.

## Notes

- LinkedIn ships an atomic/hashed CSS build with no stable semantic class names, no `<h1>`, and no JSON-LD structured data on the job-view page. `content.js` instead reads: the job title from `document.title` (`"<Job Title> | <Company> | LinkedIn"`), the company from the first `a[href*="/company/"]` link, the Company Logo from that same link's `<img>` (its `src`, or `data-delayed-url` if LinkedIn hasn't lazy-loaded it yet), and the description as the longest `[data-testid="expandable-text-box"]` block on the page — its own "…altro"/"…more" toggle `<button>` stripped out first, so its label doesn't leak into the captured text — converted to Markdown by the vendored Turndown library (`turndown.js`) so paragraphs, line breaks, lists, bold/italic, and links survive. If capture starts failing, re-run the diagnostic snippet below in the page console and adjust `content.js` accordingly.
- LinkedIn's salary-insight pill (e.g. "63,2K € /yr - 70,8K € /yr") has no stable selector either — `content.js` scans short page elements outside the description for currency-plus-number-shaped text instead and sends the first match as `listingSalaryText`, a separate field from `description`. RAL Range resolution uses it alongside the Job Description text (ADR-0014); a missing or unparsed match changes nothing.
- The Company Logo is downloaded by the backend at save time and stored alongside the Job Listing record (ADR-0013) — a failed download never blocks the save, the Job Listing just ends up with no logo.
- The captured URL: on a direct `/jobs/view/<id>/` page, it's the stripped `window.location.href`. On the search-results split-pane view, clicking between postings only changes the `currentJobId` query param — `window.location` itself stays on the generic search page — so `content.js` reads `currentJobId` and builds `https://www.linkedin.com/jobs/view/<id>/` instead.
- The backend URL is hardcoded to `http://localhost:8080` in `background.js` — edit it there if your backend runs elsewhere.
- `turndown.js` is [Turndown](https://github.com/mixmark-io/turndown) vendored as a plain browser-global script (no npm/build step) and loaded as a `content_scripts` entry ahead of `content.js`, which uses the `TurndownService` global it defines.

### If capture breaks again

Paste into the DevTools console on a LinkedIn job posting page to see what's actually there:

```js
(function(){const og=[...document.querySelectorAll('meta[property^="og:"], meta[name="description"]')].map(el=>({key:el.getAttribute('property')||el.getAttribute('name'),content:el.content}));const dataAttrs=[...document.querySelectorAll('[data-test-id], [data-testid], [data-view-name]')].map(el=>({tag:el.tagName,testId:el.getAttribute('data-test-id')||el.getAttribute('data-testid'),textLen:el.textContent.trim().length})).filter(el=>el.textLen>0);console.log('title:',document.title);console.log('og/meta:',og);console.log('data-attrs:',dataAttrs);})();
```
