# CI and Release Acceptance Checklist

## How to use

Use this file to verify the GitHub CI and release implementation. Keep it at `docs/ci-release-checklist.md`.

Evaluate every criterion after implementation and update its status and evidence. This checklist is not executable configuration and does not automatically run checks.

Allowed statuses:

- PASS: supported by an actual command result or inspected GitHub evidence.
- FAIL: verification ran and the criterion was not met.
- NOT VERIFIED: verification has not run, evidence is unavailable, or manual configuration remains.

Do not infer successful GitHub workflow execution from local YAML. Do not mark remote repository settings as PASS based on documentation describing how to configure them. Record evidence separately for local behavior and remote enforcement. Never include credentials or tokens.

## Verification record

- Verified commit SHA: NOT VERIFIED as a committed artifact. Tests covered the uncommitted working tree based on HEAD `f13b997faa243e3f6abbcda4ab4c8edefa8b5c45`; that base SHA alone does not identify the tested changes.
- Verification date: 2026-10-28 (UTC).
- Go version: `go1.26.10 darwin/arm64`, selected with `GOTOOLCHAIN=go1.26.10`; `go.mod` declares minimum `go 1.26`.
- Repository URL: `https://github.com/glennsteven/shortened-url`, derived from local origin `git@github.com:glennsteven/shortened-url.git`. Remote metadata verified via unauthenticated GitHub API: HTTP 200, public visibility, default branch main. Protection/eligibility remain NOT VERIFIED.
- Pull request / workflow run URLs: NOT VERIFIED. Fresh `gh api user --jq .login` exited 4 (unauthenticated). Public Actions workflows/runs endpoints returned HTTP 200 with empty lists; no PR or hosted run URL is available, and none was created.
- Verifier: Codex, local execution and configuration inspection only.
- Tested working-tree snapshot SHA-256: `73457d5919e8e2e4b5208be0fc66c61fe81db111acc17b3206fbabe85140aede`. Sorted relative paths plus NUL, file bytes plus NUL, for root Go files, example Go files, `go.mod`, `.gitignore`, `.golangci.yml`, README, CONTRIBUTING, design docs, and `.github` files excluding bytecode. This checklist is excluded to avoid self-reference.

All Go commands below used `GOTOOLCHAIN=go1.26.10`. The installed linter is
`golangci-lint v2.11.4` (built with Go 1.26.1); the vulnerability scanner is
`govulncheck v1.9.0`. PASS for inspected configuration means local configuration
only; it does not assert a hosted workflow ran. These files remain uncommitted.
After committing or changing them, repeat affected checks and record the actual
commit/run URLs. This pass changes module identity, the example import, a generic invalid-module test fixture, and documentation. Service API/behavior and CI/release workflow code are unchanged. All affected local checks were rerun.

Documentation-only refresh: removed references to the private instruction file
and refreshed the snapshot. Go source, module identity, tests, and workflow
configuration are unchanged from the verification above. Existing runtime results
are retained; documentation references and `git diff --check` were checked again.

### Remote evidence from read-only requests

The following requests used no credentials and changed no repository state:

| Request | Observed result |
| --- | --- |
| `gh api user --jq .login` | Exit 4; GitHub CLI has no authenticated session. |
| `GET https://api.github.com/repos/glennsteven/shortened-url` | HTTP 200; full_name glennsteven/shortened-url, visibility public, default_branch main. |
| `GET https://api.github.com/repos/glennsteven/shortened-url/actions/workflows` | HTTP 200; workflows list empty. |
| `GET https://api.github.com/repos/glennsteven/shortened-url/actions/runs?per_page=5` | HTTP 200; workflow_runs list empty. |
| `GET https://api.github.com/repos/glennsteven/shortened-url/rulesets` | HTTP 200; empty list. Does not establish classic branch protection. |
| `GET https://api.github.com/repos/glennsteven/shortened-url/branches/main/protection` | HTTP 401; unavailable without authentication. |
| `GET https://api.github.com/repos/glennsteven/shortened-url/collaborators/glennsteven/permission` | HTTP 401; eligibility not established. |
| `git ls-remote https://github.com/glennsteven/shortened-url.git HEAD refs/heads/main 'refs/tags/v*'` | Exit 0; HEAD and main point to d8328541de573b23c632f443f0e2b89719cff85a; no version-tag refs returned. |

### Readiness distinctions

| State | Status | Evidence / next step |
| --- | --- | --- |
| Local source ready | PASS | Public module identity, tag compatibility, final checks and separate local replace integration pass. This is technical readiness for review; publication rights/license still need the owner. |
| GitHub CI and review enforcement verified | NOT VERIFIED | No authenticated session; public workflows/runs lists are empty, rulesets list is empty, protection/eligibility endpoints require authentication. Follow CONTRIBUTING; do not weaken protections. |
| Version published | NOT VERIFIED | v0.1.0 is intended, not published; no version tags were returned by the remote check. No tag/release action was performed. |
| Public installation verified | NOT VERIFIED | No published version retrieved and built without replace. Run the documented fresh-consumer procedure only after authorized release. |

If the source changes after verification, identify affected results and verify them again. A completed checklist is evidence for a specific commit, not a guarantee for future changes.

## 1. Local checks and workflow configuration

| ID | Acceptance criterion | Status | Evidence / remaining action |
| --- | --- | --- | --- |
| CI-01 | CI triggers on pull requests and pushes to main. | PASS | Inspected `.github/workflows/ci.yml`: `pull_request`, push branches `[main]`, and `workflow_call`; no path filters. Hosted trigger execution NOT VERIFIED. |
| CI-02 | CI selects the intended Go version from go.mod and uses compatible pinned tool versions. | PASS | `go.mod`: `go 1.26`; both workflows use `setup-go` with `go-version-file: go.mod` and `check-latest: true`. Local commands ran on Go 1.26.10; tool pins v2.11.4/v1.9.0 both executed successfully. |
| CI-03 | Formatting check fails when tracked Go source needs gofmt; CI does not silently format and pass. | PASS | Reran the exact formatting step with Bash fail-fast flags on tracked repository Go files: exit 0. In a temporary Git repo, a tracked unformatted fixture exits 1 and remains byte-for-byte unchanged; after gofmt it exits 0. Actual source formatting check is clean. |
| CI-04 | golangci-lint runs with the committed practical configuration. | PASS | `golangci-lint run ./...`, v2.11.4, exit 0: `0 issues.` Used working-tree `.golangci.yml` enabling errcheck, ineffassign, staticcheck, unused; pending commit, not a hosted run. |
| CI-05 | go vet ./... passes. | PASS | `go vet ./...` exit 0. |
| CI-06 | go test -race ./... passes, including meaningful existing unit tests. | PASS | Fresh `go test -count=1 ./...` and `go test -race -count=1 ./...` both exit 0 under the confirmed public module path. Existing URL/code, collision, storage/context, expiration, revocation, and concurrency tests passed. |
| CI-07 | govulncheck ./... runs successfully with no reported actionable vulnerabilities under the configured policy. | PASS | `/tmp/shorturl-ci-tools/bin/govulncheck ./...`, scanner v1.9.0, Go 1.26.10, official `https://vuln.go.dev` database, exit 0: `No vulnerabilities found.` Local darwin/arm64 result; hosted Ubuntu result NOT VERIFIED. No findings suppressed. |
| CI-08 | All runnable examples compile. | PASS | `go build -o /tmp/shorturl-ci-tools/example-build/ ./examples/...` exit 0; covers the runnable `examples/basic` program. Output remains temporary and is not a release asset. |
| CI-09 | Failed commands fail their jobs; required checks do not use continue-on-error or shell logic that masks failures. | PASS | Inspected run steps: no continue-on-error or error-masking suffixes; failed steps stop the job. Exact formatting step demonstrated exit 1 for a bad tracked fixture under Bash fail-fast flags. Required-check enforcement on GitHub remains REV-05 NOT VERIFIED. |
| CI-10 | Workflow syntax and referenced configuration files are valid. | PASS | actionlint v1.7.7 validates both workflows, exit 0. Ruby YAML parsing of lint/Dependabot configuration exits 0; Python release-validator suite (3 test methods with multiple cases) passes. Actual go.mod plus v0.1.0 passes CLI validation. `go mod tidy` and `go mod tidy -diff` exit 0, no added dependencies or go.sum. |

## 2. Code review and remote enforcement

| ID | Acceptance criterion | Status | Evidence / remaining action |
| --- | --- | --- | --- |
| REV-01 | .github/CODEOWNERS contains the actual eligible maintainer account or team, with no unresolved placeholder. | NOT VERIFIED | CODEOWNERS has `* @glennsteven`, matching confirmed repository identity, no placeholder. Permission endpoint returned HTTP 401 to an unauthenticated request, and gh api user exited 4. Actual owner eligibility and base-branch CODEOWNERS are not verified. |
| REV-02 | A representative PR automatically requests the expected Code Owner review. | NOT VERIFIED | No representative PR/review-request evidence. Public workflows and workflow-runs endpoints both returned empty lists; no PR created. Observe automatic Code Owner review requests after the files are merged. |
| REV-03 | A ruleset or branch protection rule targets main and requires pull requests, at least one approval, and Code Owner approval. | NOT VERIFIED | Public rulesets endpoint returned HTTP 200 with an empty list. Main protection endpoint returned HTTP 401, so classic branch protection, required PRs/approvals and Code Owner approval remain unknown. Configure/inspect actual settings using CONTRIBUTING. |
| REV-04 | Stale approvals are dismissed after new commits as configured, and the bypass policy is documented. | NOT VERIFIED | Public repository rulesets list is empty; classic protection and bypass settings require authenticated access (HTTP 401). Documentation specifies stale-approval dismissal and no bypass but does not prove enforcement. |
| REV-05 | Required status checks match the workflow's stable job names and block merging when unsuccessful or missing. | NOT VERIFIED | Local job name remains Go verification. Public workflows and workflow-runs endpoints returned empty lists; branch protection requires authenticated access. No required-check selection or blocked PR has been observed. |
| REV-06 | There is an eligible reviewer other than the PR author when approval is required. | NOT VERIFIED | Only @glennsteven is listed locally. No other eligible reviewer has been confirmed. Before enforcing Code Owner approval on a PR authored by this owner, add another owner with write access. CONTRIBUTING explicitly prohibits relying on author self-approval. |

CODEOWNERS requests reviews; it does not enforce approval by itself. Local repository changes do not configure remote protection settings.

## 3. Workflow security

| ID | Acceptance criterion | Status | Evidence / remaining action |
| --- | --- | --- | --- |
| SEC-01 | Default token permissions are minimal; contents: write is granted only to the release publishing job. | PASS | Both workflows default to `contents: read`; reusable verification explicitly read-only. Only release.yml job `publish` grants `contents: write`; that job neither checks out nor executes repository code. |
| SEC-02 | Contributor code is not executed through pull_request_target or another privileged untrusted-code path. | PASS | Inspected triggers and checkout: contributor code uses `pull_request`, checkout ref is github.sha, credentials not persisted, no inherited secrets; no pull_request_target or privileged contributor-code job. |
| SEC-03 | Third-party actions use verified full commit SHAs with version comments. | PASS | Fresh upstream `git ls-remote` checks: checkout v6.0.2 = `de0fac2e4500dabe0009e67214ff5f5447ce83dd`; setup-go v6.3.0 = `4b73464bb391d4059bd26b0524d20df3927bd417`. Every external action use has the matching full SHA and version comment. |
| SEC-04 | Linter and vulnerability tools use explicit compatible versions rather than latest. | PASS | ci.yml and CONTRIBUTING pin golangci-lint v2.11.4 using the /v2 module path and govulncheck v1.9.0; both ran successfully with Go 1.26.10. No latest tool pin. |
| SEC-05 | Dependabot covers GitHub Actions and Go modules. | PASS | Parsed `.github/dependabot.yml`: version 2, github-actions and gomod at `/`, weekly schedules. Configuration only; actual Dependabot activation/runs NOT VERIFIED. |
| SEC-06 | Outdated PR runs are cancelled; an active release is not cancelled by a newer run. | PASS | Configuration inspection: ci group includes PR number or ref; cancel-in-progress only for pull_request. Release uses a distinct release-ref group with cancel-in-progress false. Actual cancellation behavior NOT VERIFIED without hosted runs. |
| SEC-07 | No unnecessary secrets or external services are required. | PASS | Only GH_TOKEN backed by the built-in github.token is used by publishing. No added secret, PAT, AI review, deployment service, database, or infrastructure dependency. Verification downloads Go tools and queries the official vulnerability database. |

## 4. Versioning and releases

| ID | Acceptance criterion | Status | Evidence / remaining action |
| --- | --- | --- | --- |
| REL-01 | go.mod declares the intended public repository module path. | PASS | User confirmed github.com/glennsteven/shortened-url. go.mod now declares exactly that path, examples/docs import it, tidy adds no dependencies, and the actual module identity passes v0.1.0 validation. HTTPS origin/tag query succeeded; no version tags exist. |
| REL-02 | Release workflow triggers on intended version tags and validates supported semantic-version syntax. | PASS | Inspected push tags v* trigger and reran semantic-version cases: valid stable/prerelease versions accepted; malformed versions and build metadata rejected. Fresh remote tag query returned no version tags, so v0.1.0 is the intended unused version. No tag created. |
| REL-03 | Tag major version matches the Go module path; v2+ uses the appropriate major-version suffix for this module layout. | PASS | Reran generic major-path boundary tests and validated v0.1.0 against the actual go.mod public module. v0/v1 use no suffix; v2+ require matching suffix. Tests reject local-module, wrong repository, wrong major, missing suffix, /v1 and /v02. |
| REL-04 | Required verification runs against the tagged commit before publishing; it does not accidentally check main instead. | NOT VERIFIED | Inspected both checkouts at github.sha and publish dependencies on validate plus reusable CI; main is used only for ancestry. Local actionlint passes. Public hosted workflows/runs lists are empty, so no real tagged-commit verification run exists. |
| REL-05 | A GitHub Release is created only after verification succeeds, with generated release notes. | NOT VERIFIED | Publishing statically depends on validate and verify, rechecks remote tag commit, and uses --verify-tag --generate-notes. Public API reports no workflows/runs and remote has no version tags. No publication was authorized or performed. |
| REL-06 | No application binaries or redundant version file are introduced. | PASS | Inspected publish commands and repository files: no application binary build/upload, release assets, or redundant version file. Example binaries only compile into temporary storage during verification. Actual remote release assets not inspected. |
| REL-07 | README documents installation using the real module path and labels unreleased example versions clearly. | PASS | README now uses the confirmed module in the go get command and minimal example, labels v0.1.0 intended/unpublished, and distinguishes replace integration from public installation. The README example built and ran in a separate temporary local consumer with replace. Public availability remains REL-09 NOT VERIFIED. |
| REL-08 | README explains major-version imports, immutable published tags, and manual release commands. | PASS | README explains major suffixes and immutable tags and links CONTRIBUTING for exact review/commit/PR/tag handoff, branch protection, local replace integration, and a clean post-release public consumer script. Both explain that GitHub Release notes are optional for Go module resolution. |
| REL-09 | A consumer can retrieve and build against an actually published version. | NOT VERIFIED | No published version was fetched/built without replace. Separate temporary consumer successfully used require at intended v0.1.0 plus explicit replace to the checkout, with GOPROXY=off and GOWORK=off; README and post-release program both built and ran. This is local integration only. CONTRIBUTING supplies clean no-replace public verification for after authorized publication. |

Git tags provide Go module versions; a GitHub Release adds release notes but is not required for module resolution. Do not create or push tags or publish a release merely to complete this checklist without user authorization. Leave runtime release checks NOT VERIFIED until an authorized release exists.

## 5. Completion report

After implementation, replace the entries below with actual results:

- Files changed in this preparation pass: go.mod, examples/basic/main.go, .github/scripts/test_validate_release.py, README.md, CONTRIBUTING.md, docs/design.md, docs/ci-release-checklist.md. Preserved other existing working-tree changes. No workflow scope or service API was expanded.
- Local checks passed: formatting and controlled formatting rejection; go mod tidy and tidy -diff; uncached unit/race tests; vet; lint (0 issues); govulncheck v1.9.0 (no vulnerabilities); all example builds; actionlint and YAML parsing; release-validator tests and actual v0.1.0/module compatibility. URL fuzzing: 10 seconds, 590135 executions, PASS. Code fuzzing: 10 seconds, 1128506 executions, PASS. Separate temporary consumer built/ran the README and post-release example programs with explicit local replace and downloads disabled. git diff --check passed.
- Checks failed: none of the local acceptance criteria. The previous module identity and installation-documentation failures are fixed; this does not establish public installation availability.
- Checks not run and reasons: authenticated review eligibility and main protection checks unavailable (gh exit 4; public endpoints HTTP 401). No representative PR, hosted CI/release run, or published-version consumer retrieval. Public API returned no workflows/runs and no rulesets; the remote has no version tags. No remote changes were attempted.
- GitHub settings still requiring configuration or authenticated confirmation: main requires PR, at least one approval, Code Owner approval, stale approval dismissal, required Go verification from GitHub Actions, up-to-date branches, conversation resolution, no bypass, no force push/deletion; v* tag rules prohibit updates/deletions. Exact UI/API checks and handoff commands are in CONTRIBUTING. Confirm CODEOWNERS on the base branch and demonstrate blocked merging for a failed/missing check.
- Maintainer values still requiring confirmation: actual owner/reviewer eligibility; another eligible Code Owner for owner-authored PRs; chosen license/publication rights. Module path is now explicitly confirmed. No credential or token is recorded.
- Release readiness and blocking items: local source technically ready for review; GitHub enforcement NOT VERIFIED; v0.1.0 not published; public installation NOT VERIFIED. Files remain uncommitted. Owner must decide license/rights, authenticate/configure/verify protection and reviewers, review/commit/push/merge the source with green hosted CI, then explicitly authorize tagging/publication. CONTRIBUTING includes exact commands and a clean no-replace post-release go get/build check.

Report implementation complete separately from GitHub enforcement verified and release verified. Do not mark the entire task verified while required items are FAIL or NOT VERIFIED.
