## Summary

<!-- What does this PR change, and why? Reference the PRD/issue it implements. -->

## Checklist

- [ ] Branched from `main` (not committing directly to `main`)
- [ ] New architectural decisions recorded in `docs/adr/` (see `CONTEXT.md`)
- [ ] `CONTEXT.md` updated if this PR introduces or renames domain vocabulary

## Test plan

- [ ] `go build ./...`, `go vet ./...`, `go test ./...` pass in `backend/`
- [ ] `npx tsc -b` and `npx oxlint .` pass in `frontend/`
- [ ] Manually exercised the change (dev server / real app), or noted why that wasn't possible

<!-- Add specifics: which flows you clicked through, which live APIs you hit, etc. -->
