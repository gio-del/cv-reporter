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
  //     it's already the full text regardless of visual line-clamping, so
  //     there's no need to click the "see more" toggle first.
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

  function longestText(selector) {
    let longest = "";
    for (const el of document.querySelectorAll(selector)) {
      const value = el.textContent.trim();
      if (value.length > longest.length) longest = value;
    }
    return longest;
  }

  function captureJobPosting() {
    return {
      title: titleFromDocumentTitle(),
      company: firstNonEmptyText(['a[href*="/company/"]']),
      location: "",
      url: window.location.href.split("?")[0],
      description: longestText('[data-testid="expandable-text-box"]'),
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
