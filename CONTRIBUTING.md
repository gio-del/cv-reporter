# Contributing

Thanks for taking a look. This is a small project with one maintainer, so the
process is light — but a few things are deliberate, and this file explains
which.

Read [`CONTEXT.md`](CONTEXT.md) first for the domain vocabulary (Master Data,
Entry, Selection, Rewrite, Tailored CV, Job Listing, Application, …); the rest
of this file assumes it. [`docs/adr/`](docs/adr/) records why the architecture
looks the way it does — check it before proposing to change something
structural.

## Before you start

- **Open an issue first** for anything beyond a small fix, so we agree on the
  shape before you write code.
- Branch from `main`; never commit to `main` directly.
- By contributing you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## Contributor Licence Agreement

First-time contributors are asked to sign a CLA (the bot comments on your first
pull request). It keeps the project's licensing options open — including a
hosted version under different terms — while your contribution stays available
to everyone under the AGPL. If you would rather not sign, open an issue
describing the change and it can be implemented independently.

## Pull request titles

<!-- release-automation:pr-title -->
PR titles are checked in CI once release automation lands (see the release
issue). Until then, write titles in the Conventional Commits style anyway:

```
feat: add an Applications view grouped by Status
fix: keep timestamped Notes when an Application is archived
feat!: rename to Sumisura        # "!" marks a breaking change
build(deps): bump jsdom to 30.0.1
```

Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`,
`chore`, `revert`. Only `feat`, `fix`, `perf` and breaking changes appear in the
changelog. PRs are squash-merged, so **the PR title becomes the commit message
on `main`** — write it for someone reading `git log` a year from now.

## Dev loop

Prerequisites: Go, Node, Docker, the `typst` CLI, and `pdftotext` (poppler) for
the ATS-parsability check.

```
cp .env.example .env          # ANTHROPIC_API_KEY for anything that calls Claude
docker-compose up             # backend 127.0.0.1:8080, frontend 127.0.0.1:5173
```

Backend, from `backend/`:

```
go build ./... && go vet ./... && go test ./...
golangci-lint run ./...
```

Frontend, from `frontend/`:

```
npm run dev      # served by the frontend service above inside Docker
npm run build    # tsc -b + vite build
npm run lint
npm test         # npm run test:watch while working
```

Extension, from `extension/`: `npm test`.

## Tests

Backend tests are Go `testing`-package HTTP integration tests, run with `go test ./...` from `backend/`.

Backend lint: `golangci-lint run ./...` from `backend/`. It reads [`.golangci.yml`](.golangci.yml) at the repo root and is the same command CI runs, so a red lint build is reproducible locally. The enabled set is deliberately small — `gofmt`, `errcheck` (with `check-blank`, so `_ = someCall()` is flagged too), `ineffassign`, `unused`, `govet` — with the rationale for what's left out recorded in the config file itself. A genuinely intentional discard gets a `//nolint:errcheck` with a reason rather than a weaker gate. Install: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2` (the version CI pins).


#### API contract fixtures

`frontend/src/api/types.ts` is hand-written, so a contract test keeps it honest (issue #99, ADR-0035). `backend/internal/api/contract_test.go` drives every JSON-returning route through its real handler and compares the response, normalized (sorted keys; fixed timestamps, version tokens and other clock-derived values), with a golden file in `frontend/src/api/contract/fixtures/`. `frontend/src/api/contract/contract.ts` then type-asserts each fixture against the type `types.ts` declares for that route, so `npm run build` (`tsc -b`) fails with the route and JSON path of any field that is sent but undeclared, declared required but not sent, of the wrong kind, or whose optionality disagrees with the backend (`.populated` fixtures must send every optional field, `.sparse` ones none).

A plain `go test ./...` only verifies, and fails when a response no longer matches its fixture. When a response shape changes on purpose, regenerate the fixtures, review the JSON diff, then update `types.ts` until the frontend build passes:

```
cd backend && UPDATE_CONTRACT_FIXTURES=1 go test ./internal/api -run TestContract
```

Adding a route means adding its fixture to `contractFixtures` and an assertion to `contract.ts` (or an exemption with a reason, for a route with no JSON body); `TestContractRoutesAllCovered` fails otherwise.


### Frontend dev loop

From `frontend/`: `npm run dev` (served by the `frontend` service above inside Docker), `npm run build`, `npm run lint`, `npm test` (`npm run test:watch` while working).

Frontend tests are Vitest + Testing Library, rendering the real page inside a real router and faking only HTTP with Mock Service Worker — the API client, its query-string building and its error handling all run for real, and an unhandled request fails the test. They need no backend, no `ANTHROPIC_API_KEY` and no `typst`. See [`docs/adr/0019-frontend-tests-fake-only-the-network.md`](docs/adr/0019-frontend-tests-fake-only-the-network.md) for why that seam, and `frontend/src/pages/GenerationPage.test.tsx` for the example to copy when adding more.


## Architecture decisions

A change that settles something structural — a new seam, a storage decision, a
rule about what the app may or may not do — gets an ADR in
[`docs/adr/`](docs/adr/), numbered in sequence, in the same pull request as the
code. Superseded ADRs are marked superseded, never rewritten.

## Keeping docs in sync

If a change alters something `README.md`, `CONTEXT.md` or `docs/adr/`
documents (a new or changed API route, a new top-level directory, a new
running-it step, a superseded decision), update that documentation **in the same
pull request** — not in a follow-up.

## Checklist before you open a PR

- [ ] Branched from `main`
- [ ] `go build ./... && go vet ./... && go test ./...` pass in `backend/`
- [ ] `npm run build` and `npm run lint` pass in `frontend/`
- [ ] Contract fixtures regenerated if any response shape changed
- [ ] New architectural decisions recorded in `docs/adr/`
- [ ] `CONTEXT.md` updated if domain vocabulary changed
- [ ] Manually exercised the change in the real app, or said why that wasn't possible
