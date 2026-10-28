# Contributing and releasing

## Verification

CI runs on every pull request and push to `main`. The stable required status check
is **Go verification** (workflow **CI**). It checks formatting, the release validator,
`golangci-lint`, `go vet ./...`, `go test -race ./...`, example compilation, and
`govulncheck ./...`. No path filters allow documentation-only PRs to skip the check.
Tests include fuzz seed cases; extended fuzzing remains available as described in
README. A failure in any step fails the required job.

The same workflow is called by Release on the tagged commit. Its check is nested
under **Release verification**; select the ordinary PR check **Go verification**
for branch protection, not the tag-only jobs **Validate release** or
**Publish GitHub Release**. Keep the PR job name stable when editing workflows.

Go is selected from the `go` directive in `go.mod`, using the latest patch of that
minor release. `GOTOOLCHAIN=local` prevents tools from silently selecting a newer
compiler. The minimum supported version is Go 1.26.0; older Go versions are no
longer supported. Tool pins are `golangci-lint v2.11.4` and `govulncheck v1.9.0`.
The linter uses the v2 configuration schema and checks ignored errors, ineffective
assignments, static analysis, and unused code. Formatting and vet run separately.
Tool installations do not add dependencies to this library's `go.mod`.

Use the latest available Go 1.26 patch for verification and consuming applications.
A vulnerability finding in reachable standard-library code can block this workflow;
update the toolchain and compatible tool pins rather than suppressing findings or
making the check optional. Findings depend on the Go patch, vulnerability database,
and platform. See the [Go release policy](https://go.dev/doc/devel/release#policy).

To reproduce checks locally with the Go version from `go.mod`:

```sh
# Keep tool installation out of the library's dependency list.
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
# Ensure $(go env GOPATH)/bin is on PATH.
gofmt -l . # Must print no files; CI fails rather than modifying them.
python3 -B -m unittest discover -s .github/scripts -p 'test_*.py' -v
golangci-lint run ./...
go vet ./...
go test -race ./...
go build -o /tmp/shorturl-basic ./examples/basic
govulncheck ./...
```

An installed newer Go compiler can be incompatible with older analysis tools.
Use the declared compiler, or update Go and the tool pins together. The reference
CI environment is `ubuntu-24.04` with CGO enabled for race detection. Example
executables are temporary compilation outputs only; they are not release assets.

## Reviews and protecting main

`.github/CODEOWNERS` names `@glennsteven`, identified from the configured GitHub
origin `git@github.com:glennsteven/shortened-url.git`. It covers every file,
including CODEOWNERS and the workflows. Confirm that this account is the intended
maintainer and retains write access. Add other eligible maintainers on that same
`*` line as needed. GitHub uses CODEOWNERS from the PR's base branch, so merge the
file to `main` before relying on automatic review requests.

CODEOWNERS automatically requests review on ready PRs; **it does not enforce
approval by itself**. A repository administrator must configure protection.
No repository settings are changed by the files in this change.

In **Settings → Branches → Branch protection rules → Add classic branch protection rule**, set:

1. **Branch name pattern:** `main`.
2. Enable **Require a pull request before merging**.
3. Enable **Require approvals**, with **Required number of approvals before merging: 1**.
4. Enable **Dismiss stale pull request approvals when new commits are pushed**.
5. Enable **Require review from Code Owners**.
6. Enable **Require status checks to pass before merging** and add
   **Go verification**, selecting **GitHub Actions** as the expected source when
   offered. Enable **Require branches to be up to date before merging**.
7. Enable **Require conversation resolution before merging**.
8. Enable **Do not allow bypassing the above settings** so protection also applies
   to administrators and bypass roles. Leave any PR bypass list empty.
9. Leave **Allow force pushes** and **Allow deletions** disabled, then save.

Run CI once after the workflow is present on GitHub so its status name is available
in the selector. Check the PR Checks tab if GitHub displays a workflow prefix.
Do not select release-only checks, which do not run on PRs. See GitHub's
[branch protection documentation](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
and [CODEOWNERS documentation](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners).

**PR authors cannot approve their own PRs.** A solo maintainer needs another
eligible reviewer to meet the approval rule. To satisfy Code Owner approval on a
PR authored by `@glennsteven`, another reviewer must also have write access and be
listed as a Code Owner for the affected files. Another generic approval alone
will not meet that requirement. Arrange this before enabling enforcement; do not
expect CODEOWNERS or an automated review bot to provide a self-approval.

## Versioning and publication prerequisites

Versions are Git tags such as `v0.1.0` and `v1.0.0`; there is no separate version
file. Tags may have valid SemVer prerelease identifiers, such as `v1.0.0-rc.1`.
The workflow marks these as GitHub prereleases. Leading zeroes in numeric version
components/identifiers and build metadata (`+...`) are rejected by this release
policy. The workflow supports this repository's single root GitHub module.

The confirmed root module path is `github.com/glennsteven/shortened-url`, matching
origin `git@github.com:glennsteven/shortened-url.git`. The intended first version is
`v0.1.0`; no version tag existed at the preparation check. Recheck tags immediately
before creating one. The module path and imports are ready locally, but no public
release or installation is claimed. The release validator requires the module to
match `github.com/$GITHUB_REPOSITORY` and its version suffix; a fork must deliberately
adopt its own module path before its release workflow can publish.

For v0 and v1, use the unsuffixed module path. For v2 and later, Go modules generally
require `/v2`, `/v3`, etc. in both `go.mod` and consumer imports; e.g. a `v2.0.0` tag
requires `github.com/glennsteven/shortened-url/v2`. The release validator rejects a
tag/module-major mismatch, including a v1 tag against a `/v2` module. This project
does not use the special `gopkg.in` versioning convention. See the
[Go major-version rules](https://go.dev/ref/mod#major-version-suffixes).

Confirm ownership and publication rights and select a license before releasing;
no license is assumed or added here. Confirm that the repository is public and
accessible to consumers. Installation does not require a GitHub Release: Go resolves
versions from an accessible repository, valid module path, and Git version tag.
A tag can be fetched before the release-note workflow finishes, so a release
workflow is a verification gate for GitHub Release publication, not for creation
of the already-pushed Go module version.

**Published versions are immutable.** Never move, delete/recreate, force-push, or
overwrite a published version tag or release. Go proxies and checksums may already
have cached it. Fix mistakes with a new patch version. Tag deletion is not a rollback
for consumers. The workflow rejects tag updates and refuses to overwrite existing
GitHub Releases; it also checks that the tag still points at the verified commit
immediately before publication. These checks do not replace GitHub tag protection.

In **Settings → Rules → Rulesets**, add an **Active tag ruleset** targeting `v*`.
Enable **Restrict updates** and **Restrict deletions**, with no routine bypasses.
Restrict creation to release maintainers if your repository's available ruleset
settings allow it. Protecting tag immutability helps prevent races that a workflow
alone cannot eliminate.

## First release, when ready

`v0.1.0` below is an **example planned version, not a claim of an existing release**.
Check remote tags first and choose a new version if it already exists.

1. Choose a license and confirm publication rights. Authenticate to GitHub with
   `gh auth login`, confirm repository visibility remains public, and inspect the
   actual settings below. Confirm eligible Code Owners, branch protection, and a
   successful CI run on `main`. Do not weaken existing protection.
2. Check out an up-to-date `main` with a clean working tree and rerun local checks.
3. Create and push one annotated tag at that verified commit:

   ```sh
   git switch main
   git pull --ff-only origin main
   git status --short # Must be empty before proceeding.
   git ls-remote --tags origin 'refs/tags/v*'
   # After local verification and confirming v0.1.0 is unused:
   python3 -B .github/scripts/validate_release.py v0.1.0 \
     "$(go list -m -f '{{.Path}}')" glennsteven/shortened-url
   git tag -a v0.1.0 -m 'Release v0.1.0'
   git push origin refs/tags/v0.1.0
   ```

4. Watch **Actions → Release**. It validates tag syntax, new-tag creation, module
   path/major, and ancestry on `main`; calls the full CI workflow on the tagged
   commit; then creates a GitHub Release with generated notes. It builds no
   distributable binaries and attaches no artifacts. Review the generated notes.
5. After successful publication, run the clean consumer verification below. Do not
   create the GitHub Release manually before the workflow; publishing refuses an
   existing one. Public installation is verified only after that separate build
   succeeds without a replace directive.

If a transient failure occurs after a tag is pushed, rerun the failed workflow on
that same commit. Do not move the tag to fix failing code; merge the fix and choose
a new version. Only the publishing job has `contents: write`, using GitHub's
short-lived built-in token. No personal access token or other secret is required.
The publishing job neither checks out nor executes repository code.

## Workflow maintenance

Actions are pinned to full commit SHAs with release-version comments. Dependabot
checks GitHub Actions and Go modules weekly. Review its updates, including changed
SHAs and the corresponding upstream release. It does not update `go install` tool
pins in shell commands: maintain those pins alongside the Go version and
`.golangci.yml` schema. Recheck tool minimum Go requirements before upgrading.

Outdated PR runs are canceled; main and release runs are not canceled by this
setting. Release concurrency groups are separate from verification groups to
avoid a reusable workflow canceling or deadlocking its caller. Untrusted PR code
runs under `pull_request` with read-only contents permission and no inherited
secrets; this setup does not use `pull_request_target` or an external review service.

## Local consumer integration

Before publication, use a separate temporary module and an explicit local replace.
This checks the public import path and API against the checkout only. It is not
proof that any version can be downloaded publicly. Use Go 1.26 or newer:

```sh
shorturl_checkout="$(pwd)" # Run from this repository checkout.
consumer_dir="$(mktemp -d)"
(
  cd "$consumer_dir"
  export GOWORK=off
  go mod init example.com/shorturl-local-check
  go mod edit -require=github.com/glennsteven/shortened-url@v0.1.0
  go mod edit "-replace=github.com/glennsteven/shortened-url=$shorturl_checkout"
  cp "$shorturl_checkout/examples/basic/main.go" main.go
  go mod tidy
  go build ./...
  go run .
  go list -m -json github.com/glennsteven/shortened-url
)
```

The `v0.1.0` requirement here is a placeholder for the intended version; replace
satisfies it locally even when that tag does not exist. Expect `go list` to show
`Replace` pointing to the checkout. Keep this integration check separate from
REL-09 in the acceptance checklist.

## Source review and authorized publication handoff

The module identity is confirmed. Before tagging, the owner still needs to choose
a license and confirm publication rights, arrange an eligible second Code Owner
for owner-authored PRs, and verify actual GitHub protection and CI. None of those
settings are changed by local files. GitHub CLI authentication was unavailable at
the preparation check; authenticate without sharing credentials in reports:

```sh
gh auth login
gh api repos/glennsteven/shortened-url --jq '{full_name, visibility, default_branch}'
gh api repos/glennsteven/shortened-url/collaborators/glennsteven/permission
gh api repos/glennsteven/shortened-url/branches/main/protection
gh api repos/glennsteven/shortened-url/rulesets
# Inspect every returned ruleset's details, including bypasses and required checks:
# gh api repos/glennsteven/shortened-url/rulesets/ACTUAL_RULESET_ID
```

Use the exact branch/tag settings in the earlier sections. If an endpoint returns
403/404, obtain an authorized administrator's settings evidence; do not interpret
lack of access as proof that protection is absent. Confirm Code Owner eligibility
and review requests on a representative PR, and observe that a failed or missing
`Go verification` check blocks merging.

Review all working-tree changes, including untracked configuration. The following
commands are a handoff for the owner, not actions performed during preparation:

```sh
git status --short
git diff --check
git diff
# Use a feature branch; if this name exists, choose another unused branch name.
git switch -c prepare-public-v0.1.0
# Inspect the untracked files too before staging them.
git add go.mod examples/basic/main.go README.md CONTRIBUTING.md docs \
  .github .golangci.yml .gitignore validation.go
# Stage the owner-chosen LICENSE separately once publication rights are confirmed.
git diff --cached --check
git diff --cached
git commit -m 'Prepare public Go module and CI release verification'
# Push the reviewed branch, then open/review/merge a PR under main's protections.
git push -u origin HEAD
gh pr create --base main --fill
```

After approval and merge, follow **First release, when ready**: update `main`,
ensure its working tree is clean and its required checks pass, rerun the local
verification commands on that commit, validate `v0.1.0` against the module path,
check the remote tag is unused, then create/push the annotated tag. Do not use
`--force`. If `v0.1.0` appears before you tag, stop and choose a new reviewed,
unused version rather than moving it. Watch the Release workflow and record its
commit SHA and run URL in the checklist. The workflow's read-only validation and
CI jobs both check out the push event's `github.sha`; publishing depends on both
jobs and rechecks the remote tag's resolved commit. Main is used for an ancestry
check, not as the source for tagged verification.

## Post-release public consumer verification

**Run only after the authorized release exists.** This has not been performed
as proof of public availability during preparation. The clean module disables Go
workspaces and uses a fresh module download cache; no replace directive is added.
The unversioned `go get` checks public discovery; the second command pins the
intended first release so the build verifies that specific version:

```sh
consumer_dir="$(mktemp -d)"
(
  cd "$consumer_dir"
  export GOWORK=off
  export GOMODCACHE="$consumer_dir/module-cache"
  go mod init example.com/shorturl-public-check
  cat > main.go <<'GO'
package main

import (
    "context"
    "log"

    shorturl "github.com/glennsteven/shortened-url"
)

func main() {
    service, err := shorturl.New(&shorturl.MemoryStore{}, shorturl.Config{})
    if err != nil {
        log.Fatal(err)
    }
    ctx := context.Background()
    link, err := service.Shorten(ctx, "https://example.com/docs", 0)
    if err != nil {
        log.Fatal(err)
    }
    destination, err := service.Resolve(ctx, link.Code)
    if err != nil {
        log.Fatal(err)
    }
    if destination != "https://example.com/docs" {
        log.Fatal("unexpected destination")
    }
}
GO
  go get github.com/glennsteven/shortened-url
  go get github.com/glennsteven/shortened-url@v0.1.0
  test "$(go list -m -f '{{.Version}}' github.com/glennsteven/shortened-url)" = v0.1.0
  test -z "$(go list -m -f '{{if .Replace}}REPLACED{{end}}' github.com/glennsteven/shortened-url)"
  go list -m -json github.com/glennsteven/shortened-url
  go build ./...
  go run .
)
```

Only after successful retrieval, version/no-replace checks, and compilation should
REL-09 become PASS. Record the actual command output, version, date, and tagged
commit. If another version was selected for publication, replace the explicit
version and expected-version assertion together. Never change a published tag to
make this check succeed.
