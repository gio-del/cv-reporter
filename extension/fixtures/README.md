# Fixtures

`indeed-job-view.html` is a **synthetic** approximation of an Indeed
job-view page's DOM shape (the `data-testid`/id selectors `content-indeed.js`
targets), built from Indeed's commonly-documented markup rather than a real
captured page — this environment had no live browser access to take an
actual snapshot when this fixture was written.

`linkedin-job-view.html` is **synthetic** in exactly the same way: it is
built to the DOM shape `content.js` targets (the `"<Job Title> | <Company> |
LinkedIn"` document title, the `a[href*="/company/"]` link wrapping the logo
`<img>`, two `[data-testid="expandable-text-box"]` blocks of differing
lengths each carrying their own "…see more" toggle, and a short salary pill
outside the description), not from a captured LinkedIn page. A green test
certifies that the extraction logic behaves correctly against that shape —
it can never certify that LinkedIn still serves it.

Before relying on either board's capture in production, re-verify its
content script's selectors against a real, currently-served job-view page
and replace the fixture with a genuine (anonymized) snapshot, updating the
fixture and its test together if they've drifted — the same fragility
already accepted for reading any third party's DOM, just made visible here
instead of failing silently.

Variants are produced by mutating the loaded fixture inside a test (removing
the heading node, clearing the logo `src`, dropping the salary pill, passing
a different page URL), not by committing more fixture files — one canonical
fixture per board.
