# CV Reporter — LinkedIn Capture

A browser extension that captures the LinkedIn job posting you're currently viewing into your local CV Reporter app as a Job Listing. See `docs/adr/0007-job-sourcing.md` for why this is scoped to reading a page you're already looking at, never scraping.

## How it works

- A "Save to CV Reporter" button appears (bottom-right) on any `linkedin.com/jobs/*` page.
- Clicking it reads the title, company, location, and description already rendered on the page — no request to LinkedIn is made by the extension.
- That content is sent to your local backend (`POST http://localhost:8080/api/job-listings/from-extension`), which saves it as a Job Listing the same way a manually-pasted one is saved (RAL Range looked up, Application Method inferred, Application created at Saved).
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

- LinkedIn ships an atomic/hashed CSS build with no stable semantic class names, no `<h1>`, and no JSON-LD structured data on the job-view page. `content.js` instead reads: the job title from `document.title` (`"<Job Title> | <Company> | LinkedIn"`), the company from the first `a[href*="/company/"]` link, and the description as the longest `[data-testid="expandable-text-box"]` block on the page. If capture starts failing, re-run the diagnostic snippet below in the page console and adjust `content.js` accordingly.
- The backend URL is hardcoded to `http://localhost:8080` in `background.js` — edit it there if your backend runs elsewhere.

### If capture breaks again

Paste into the DevTools console on a LinkedIn job posting page to see what's actually there:

```js
(function(){const og=[...document.querySelectorAll('meta[property^="og:"], meta[name="description"]')].map(el=>({key:el.getAttribute('property')||el.getAttribute('name'),content:el.content}));const dataAttrs=[...document.querySelectorAll('[data-test-id], [data-testid], [data-view-name]')].map(el=>({tag:el.tagName,testId:el.getAttribute('data-test-id')||el.getAttribute('data-testid'),textLen:el.textContent.trim().length})).filter(el=>el.textLen>0);console.log('title:',document.title);console.log('og/meta:',og);console.log('data-attrs:',dataAttrs);})();
```
