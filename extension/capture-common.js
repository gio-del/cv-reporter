// Shared, board-agnostic pieces used by every board's content script:
// capture-button/status UI injection, click -> background.js message-passing,
// and generic DOM-reading helpers (Turndown-based HTML-to-Markdown, "first
// non-empty text from a selector list", "longest element matching a
// selector"). Board-specific field extraction (which selectors to try, how
// to build the canonical URL, etc.) stays in each board's own content
// script — this module only holds what genuinely doesn't vary by board.

(function (root) {
  function firstNonEmptyText(doc, selectors) {
    for (const selector of selectors) {
      const el = doc.querySelector(selector);
      const value = el && el.textContent.trim();
      if (value) return value;
    }
    return "";
  }

  function longestElement(doc, selector) {
    let longestEl = null;
    let longestLen = 0;
    for (const el of doc.querySelectorAll(selector)) {
      const len = el.textContent.trim().length;
      if (len > longestLen) {
        longestLen = len;
        longestEl = el;
      }
    }
    return longestEl;
  }

  function htmlToMarkdown(html) {
    // turndown.js is loaded ahead of this module in every content-script
    // entry (see manifest.json), defining the global TurndownService.
    return new TurndownService().turndown(html).trim();
  }

  function descriptionMarkdown(doc, selector, stripSelectors) {
    const el = longestElement(doc, selector);
    if (!el) return "";
    // Job boards commonly nest stray interactive controls ("see more"
    // toggles, apply buttons) inside the description container itself, so
    // their labels leak into the captured text if read as-is. Clone first
    // and strip them out — degrades to the unmodified description (rather
    // than failing capture entirely) if the clone/strip step ever throws.
    let html = el.innerHTML;
    try {
      const clone = el.cloneNode(true);
      (stripSelectors || ["button"]).forEach((sel) => {
        clone.querySelectorAll(sel).forEach((node) => node.remove());
      });
      html = clone.innerHTML;
    } catch (err) {
      console.error("[CVReporter] description clean-up threw, using unmodified description", err);
    }
    return htmlToMarkdown(html);
  }

  function showStatus(statusEl, ok, message) {
    statusEl.textContent = message;
    statusEl.className = "cv-reporter-capture-status " + (ok ? "cv-reporter-capture-status--ok" : "cv-reporter-capture-status--error");
    statusEl.hidden = false;
    root.clearTimeout(showStatus._timer);
    showStatus._timer = root.setTimeout(() => {
      statusEl.hidden = true;
    }, 5000);
  }

  function onCaptureClick(captureJobPosting, button, statusEl) {
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

  function ensureUI(doc, captureJobPosting) {
    if (doc.getElementById("cv-reporter-capture-btn")) return;

    const button = doc.createElement("button");
    button.id = "cv-reporter-capture-btn";
    button.type = "button";
    button.className = "cv-reporter-capture-btn";
    button.textContent = "Save to CV Reporter";

    const statusEl = doc.createElement("div");
    statusEl.id = "cv-reporter-capture-status";
    statusEl.className = "cv-reporter-capture-status";
    statusEl.hidden = true;

    button.addEventListener("click", () => onCaptureClick(captureJobPosting, button, statusEl));

    doc.body.appendChild(button);
    doc.body.appendChild(statusEl);
  }

  function initCaptureUI(captureJobPosting) {
    console.log("[CVReporter] content script loaded", root.location.href);
    ensureUI(document, captureJobPosting);
    // Job boards are typically single-page apps; guard against our injected
    // elements being removed by their own re-renders on client-side
    // navigation between postings.
    new MutationObserver(() => ensureUI(document, captureJobPosting)).observe(document.body, { childList: true, subtree: false });
  }

  const CVReporterCommon = {
    firstNonEmptyText,
    longestElement,
    descriptionMarkdown,
    showStatus,
    onCaptureClick,
    ensureUI,
    initCaptureUI,
  };

  root.CVReporterCommon = CVReporterCommon;
  if (typeof module !== "undefined" && module.exports) {
    module.exports = CVReporterCommon;
  }
})(typeof window !== "undefined" ? window : globalThis);
