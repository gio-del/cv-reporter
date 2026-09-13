# The Claude model is chosen per call site, with env overrides

The backend's Claude calls (ADR-0005) no longer share one pinned model. `internal/claude/models.go` maps each *call site* — one kind of request, e.g. `ral_extraction` — to a default model, and every request takes its model from that map. The key is a call site, not the usage `CallType` string, because `EstimateRAL` and `SuggestContact` each make two requests (research, then extraction — ADR-0011) recorded under one `CallType`, and those halves warrant different models. `CallType` strings are persisted on every `GenerationRecord`, so they stay as they are.

Defaults follow one rule: a call whose output is judgment the user reviews, or that has to weigh sources, keeps the model it had (Sonnet 5); a call whose output is mechanically derived from an earlier call's notes, or is one click from correction, runs on Haiku 4.5.

| Call site | Default |
|---|---|
| `selection_rewrite`, `selection_preview`, `cover_letter` | Sonnet 5 |
| `ral_research`, `contact_research` | Sonnet 5 |
| `ral_extraction`, `contact_extraction` | Haiku 4.5 |
| `application_method_inference` | Haiku 4.5 |

`CV_REPORTER_MODEL_<CALL_SITE>` overrides one call site, `CV_REPORTER_MODEL_DEFAULT` all of them; per-call-site wins over blanket, which wins over the built-in default, and an empty value counts as unset.

## Considered Options

- Keep one model for everything and let the user override it globally — rejected because the calls are very different work (an eight-thousand-token Rewrite under a no-invention constraint vs. a four-way enum), and a single knob can't express "try Opus on Rewrite alone" or "everything on Haiku except Rewrite".
- Downgrade Selection/Rewrite/Cover Letter too — rejected because those are the outputs the user sends to employers; changing quality and cost in the same step makes any regression unattributable. The override exists so that experiment happens one call site at a time.
- Downgrade the two web-research calls — rejected because they must judge whether a salary or contact source is credible, and their flat $10/1,000 web-search fee means a cheaper model saves proportionally little.
- Key models on the existing `CallType` strings — rejected because it cannot tell the research half of a two-call method from its extraction half, and re-keying the strings would reinterpret existing Generation records.
- Validate override values against a local allowlist at startup — rejected because a localhost tool shouldn't refuse to boot over a model name, and the API's own error is clearer than a stale local list.

## Consequences

- `internal/claude/pricing.go` must price every model a call site can reach. Before this, one priced model and a silent `$0` for anything else was merely latent; with three models it would under-report the cost figures from issue #39. The table now holds Sonnet 5, Haiku 4.5 and Opus 5, resolves dated snapshot ids (`<alias>-YYYYMMDD`) to their alias, and logs a warning once per unpriced model. `TestPricingTable_CoversEveryReachableModel` fails if a default names an unpriced model.
- The call-site tests assert on the model id in the outgoing request body, via a request-capturing fake Anthropic server, rather than on the map's contents, so a call site that bypasses the map is caught.
- The Selection preview's call, previously unrecorded, now records usage as `selection_preview`, logged to the standalone usage log. The app's lifetime cost total steps up as a correction, not an increase.
- The `tailor-cv` skill is unaffected: it runs under whatever model the user's Claude Code session uses.
