# RAL Range can be Stated from a listing's own salary field, not just the Job Description text

`ParseStatedRAL` now also runs against an optional salary-field text captured separately from the Job Description — LinkedIn shows a computed salary-insight badge next to the job title that's often absent from the description prose entirely, so relying on Job Description text alone under-detects Stated RAL for listings that only expose it as a separate field. When the description and the listing field both state a figure and their ranges don't overlap at all, `RALRange` reports a new `conflict` source with both figures labeled, rather than picking one silently; overlapping or matching figures resolve to the Job-Description-stated value.

## Considered Options

- Fold the captured salary text into the Job Description string sent to the backend (no schema change) — rejected because it repeats the "fold captured signals into the description blob" pattern this repo's own Job Title work (PR #20) just moved away from, and stretches Job Description's definition ("the pasted text or fetched-URL content describing a role") to include something that was never part of that text.
- Always prefer one source over the other when they disagree — rejected because RAL Range has no manual-correction UI (unlike Application Method), so a silently wrong pick can't be fixed by the user afterward; surfacing both figures lets them judge which is right for that listing.

## Consequences

- `RALRange` gains a `conflict` source and two optional labeled sub-figures (populated only in that state); every consumer of `RALRange` (FE `RALBadge`, YAML persistence, generation output) needs to handle the new case.
- The dedicated salary-field text is only ever populated by the browser-extension capture path today — manual-paste and ATS-board intake have no equivalent structured field to read from, so RAL resolution for those paths is unchanged.
- The Go parser's numeric shorthand (`numberToken`) needs a companion fix to handle fractional-K notation ("63,2K" / "63.2K" meaning 63,200) — LinkedIn's own salary badge uses this format, and the existing regex silently mis-parses it as 63 rather than failing loudly.
