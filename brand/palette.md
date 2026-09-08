# Palette

| Role | Hex | Rationale |
|---|---|---|
| Primary | `#4F46E5` (indigo) | The document/mark color and the wordmark color — reads as precise and technical without being cold, and is distinct from the frontend's old generic placeholder favicon. |
| Supporting — accent | `#F59E0B` (amber) | The bullseye center in the mark — a warm highlight for "the one role this CV is tailored to," reused for anything that should read as *selected* or *tailored* in the UI. |
| Supporting — ink | `#1E293B` (slate) | Neutral dark for body text/surfaces, chosen for reliable contrast against both light and dark backgrounds. |

No further tints/shades are tracked here beyond `#3730A3`, a one-off darker shade of the primary used for the folded-corner fold in the logo mark itself (see `logo-mark.svg`) — not a separate palette color.

This is intentionally small: a primary plus two supporting colors, not a design-token system. It exists so the UI-rebrand PRD (tracked separately) has one place to read the brand colors from, rather than reverse-engineering them from the logo file.
