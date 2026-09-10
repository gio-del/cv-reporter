# Fixtures

`indeed-job-view.html` is a **synthetic** approximation of an Indeed
job-view page's DOM shape (the `data-testid`/id selectors `content-indeed.js`
targets), built from Indeed's commonly-documented markup rather than a real
captured page — this environment had no live browser access to take an
actual snapshot when this fixture was written.

Before relying on Indeed capture in production, re-verify `content-indeed.js`'s
selectors against a real, currently-served Indeed job-view page and replace
this fixture with a genuine (anonymized) snapshot, updating both together if
they've drifted — the same fragility already accepted for the LinkedIn
script, just made visible here instead of failing silently.
