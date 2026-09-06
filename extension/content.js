// Runs only on linkedin.com/jobs/* pages the user is already viewing, and
// only reads the page's own DOM — no requests to LinkedIn are made by this
// script (story 2). It acts only when the injected button is clicked
// (story 6); the network call itself happens in background.js.

(function () {
  // LinkedIn's job-view page ships an atomic/hashed CSS build (class names
  // like "b2cfd878") with no stable semantic classes, no <h1>, and no
  // schema.org JSON-LD to read structured data from. These three signals
  // held up under inspection instead:
  //   - document.title follows "<Job Title> | <Company> | LinkedIn"
  //   - the company name is the first `a[href*="/company/"]` link's text
  //   - the job description is the longest `[data-testid="expandable-text-box"]`
  //     block (a shorter one is typically an "about the company" blurb) —
  //     it's already the full content regardless of visual line-clamping, so
  //     there's no need to click the "see more" toggle first. Its innerHTML
  //     (not textContent) is run through Turndown so paragraphs, line
  //     breaks, lists, and bold/italic/links survive as Markdown.
  // If LinkedIn changes any of this, capture will start failing — that's a
  // known fragility of reading a third party's DOM (see extension/README.md).

  function firstNonEmptyText(selectors) {
    for (const selector of selectors) {
      const el = document.querySelector(selector);
      const value = el && el.textContent.trim();
      if (value) return value;
    }
    return "";
  }

  function titleFromDocumentTitle() {
    // "Senior Machine Learning Engineer | Prima | LinkedIn" -> the first part
    const [jobTitle] = document.title.split("|").map((part) => part.trim());
    return jobTitle || "";
  }

  function captureUrl() {
    // The search-results split-pane view (.../jobs/search-results/?currentJobId=123...)
    // never changes window.location itself when you click between postings —
    // only the currentJobId query param does — so the stripped href is the
    // generic search page, not the posting. A direct /jobs/view/<id>/ page has
    // no currentJobId param, so the existing stripped-URL behavior still
    // applies there unchanged.
    const currentJobId = new URLSearchParams(window.location.search).get("currentJobId");
    if (currentJobId) {
      return `https://www.linkedin.com/jobs/view/${currentJobId}/`;
    }
    return window.location.href.split("?")[0];
  }

  function companyLogoUrl() {
    // The company logo is an <img> inside the same a[href*="/company/"]
    // link the company name comes from. LinkedIn lazy-loads some images
    // via a data-delayed-url attribute before src is populated, so fall
    // back to that when src is still empty/placeholder.
    const link = document.querySelector('a[href*="/company/"]');
    const img = link && link.querySelector("img");
    if (!img) return "";
    return img.src || img.getAttribute("data-delayed-url") || "";
  }

  function longestElement(selector) {
    let longestEl = null;
    let longestLen = 0;
    for (const el of document.querySelectorAll(selector)) {
      const len = el.textContent.trim().length;
      if (len > longestLen) {
        longestLen = len;
        longestEl = el;
      }
    }
    return longestEl;
  }

  function descriptionMarkdown(selector) {
    const el = longestElement(selector);
    if (!el) return "";
    // turndown.js is loaded as a content script ahead of this one (see
    // manifest.json), defining the global TurndownService — preserves
    // paragraphs, line breaks, lists, bold/italic, and links as Markdown
    // instead of flattening the description into one run of text.
    return new TurndownService().turndown(el.innerHTML).trim();
  }

  function captureJobPosting() {
    return {
      title: titleFromDocumentTitle(),
      company: firstNonEmptyText(['a[href*="/company/"]']),
      location: "",
      url: captureUrl(),
      description: descriptionMarkdown('[data-testid="expandable-text-box"]'),
      logoUrl: companyLogoUrl(),
    };
  }

  function showStatus(statusEl, ok, message) {
    statusEl.textContent = message;
    statusEl.className = "cv-reporter-capture-status " + (ok ? "cv-reporter-capture-status--ok" : "cv-reporter-capture-status--error");
    statusEl.hidden = false;
    window.clearTimeout(showStatus._timer);
    showStatus._timer = window.setTimeout(() => {
      statusEl.hidden = true;
    }, 5000);
  }

  function onCaptureClick(button, statusEl) {
    console.log("[CVReporter] button clicked");
    let payload;
    try {
      payload = captureJobPosting();
    } catch (err) {
      console.error("[CVReporter] captureJobPosting threw", err);
      showStatus(statusEl, false, "Capture failed: " + err.message);
      return;
    }
    console.log("[CVReporter] captured payload", payload);

    if (!payload.company || !payload.description) {
      showStatus(statusEl, false, "Couldn't find a job posting on this page — open a specific listing and try again.");
      return;
    }

    button.disabled = true;
    button.textContent = "Saving…";

    console.log("[CVReporter] sending message to background");
    chrome.runtime.sendMessage({ type: "CV_REPORTER_CAPTURE", payload }, (response) => {
      console.log("[CVReporter] got response", response, "lastError:", chrome.runtime.lastError);
      button.disabled = false;
      button.textContent = "Save to CV Reporter";

      if (chrome.runtime.lastError) {
        showStatus(statusEl, false, chrome.runtime.lastError.message);
        return;
      }
      if (response && response.ok) {
        showStatus(statusEl, true, "Saved to CV Reporter.");
      } else {
        showStatus(statusEl, false, (response && response.error) || "Failed to save.");
      }
    });
  }

  function ensureUI() {
    if (document.getElementById("cv-reporter-capture-btn")) return;

    const button = document.createElement("button");
    button.id = "cv-reporter-capture-btn";
    button.type = "button";
    button.className = "cv-reporter-capture-btn";
    button.textContent = "Save to CV Reporter";

    const statusEl = document.createElement("div");
    statusEl.id = "cv-reporter-capture-status";
    statusEl.className = "cv-reporter-capture-status";
    statusEl.hidden = true;

    button.addEventListener("click", () => onCaptureClick(button, statusEl));

    document.body.appendChild(button);
    document.body.appendChild(statusEl);
  }

  console.log("[CVReporter] content script loaded", window.location.href);
  ensureUI();

  // LinkedIn is a single-page app; guard against our injected elements
  // being removed by its own re-renders on client-side navigation between
  // job postings.
  new MutationObserver(() => ensureUI()).observe(document.body, { childList: true, subtree: false });
})();
