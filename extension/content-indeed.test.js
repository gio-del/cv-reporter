const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { JSDOM } = require("jsdom");

// capture-common.js references a bare `TurndownService` identifier
// (matching how it's used in the browser, where turndown.js's top-level
// `var` declaration becomes a global). Requiring the vendored file's
// Node export and assigning it to `global` reproduces that here.
global.TurndownService = require("./turndown.js");
// turndown.js's own HTML-string parser falls back to a bare `document`
// reference when no global `window`/`DOMParser` exists (true in Node) — a
// throwaway document is enough, it's only used to parse the markup string
// turndown is converting, not the fixture document under test.
global.document = new JSDOM("<!doctype html><html><body></body></html>").window.document;

const { captureJobPosting } = require("./content-indeed.js");

function loadFixtureDocument() {
  const html = fs.readFileSync(path.join(__dirname, "fixtures", "indeed-job-view.html"), "utf8");
  const dom = new JSDOM(html, { url: "https://www.indeed.com/viewjob?jk=abc123&from=serp&tk=xyz" });
  return dom.window.document;
}

test("captureJobPosting extracts title, company, location, description, logo, and salary from an Indeed job-view page", () => {
  const doc = loadFixtureDocument();
  const payload = captureJobPosting(doc);

  assert.equal(payload.title, "Senior Backend Engineer");
  assert.equal(payload.company, "Acme Rockets");
  assert.equal(payload.location, "Milan, Italy");
  assert.equal(payload.logoUrl, "https://example.invalid/logos/acme-rockets.png");
  assert.equal(payload.listingSalaryText, "€55,000 - €70,000 a year");
});

test("captureJobPosting builds a canonical URL from the jk query param, dropping tracking params", () => {
  const doc = loadFixtureDocument();
  const payload = captureJobPosting(doc);

  assert.equal(payload.url, "https://www.indeed.com/viewjob?jk=abc123");
});

test("captureJobPosting converts the description to Markdown and strips the embedded Apply button", () => {
  const doc = loadFixtureDocument();
  const payload = captureJobPosting(doc);

  assert.match(payload.description, /\*\*Senior Backend Engineer\*\*/);
  assert.match(payload.description, /Design and operate distributed systems in Go/);
  assert.doesNotMatch(payload.description, /Apply now/);
});

test("captureJobPosting falls back to document.title when the heading selector is missing", () => {
  const doc = loadFixtureDocument();
  doc.querySelector('[data-testid="jobsearch-JobInfoHeader-title"]').remove();

  const payload = captureJobPosting(doc);

  assert.equal(payload.title, "Senior Backend Engineer");
});

test("captureJobPosting yields empty company/description (capture-failure signal) on a page with no job posting markup", () => {
  const dom = new JSDOM("<!doctype html><html><body><h1>Search results</h1></body></html>", {
    url: "https://www.indeed.com/viewjob?jk=none",
  });

  const payload = captureJobPosting(dom.window.document);

  assert.equal(payload.company, "");
  assert.equal(payload.description, "");
});
