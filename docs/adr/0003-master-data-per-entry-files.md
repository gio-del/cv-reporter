# Master Data as one file per Entry, not a single config file

The original project stored everything in a single JS config file. Master Data is instead one file per Entry (YAML frontmatter for dates/Tags, Markdown body for bullets), split across experience/ and projects/ directories. This scales better as Entries accumulate over years, keeps git diffs small and reviewable when adding or editing a single Entry, and gives the Selection step clean per-Entry metadata to reason about without parsing a growing monolith.
