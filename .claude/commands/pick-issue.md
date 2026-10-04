---
description: Take the next issue from TASKS.md through implementation and verification in its own worktree, open a small pull request, and stop
---

# Pick an Issue: One Issue Per Invocation

Choose exactly one open issue in `rshade/finfocus-plugin-aws-public`, take it
to a verified pull request, and **stop**. Do not start a second issue.

Adapted from the Spec Kit `pick-issue` command in `rshade/finfocus`. What
differs here:

- **The plan of record is [TASKS.md](../../TASKS.md).** It fixes the order and
  the dependencies. Roadmap labels are context only.
- **One worker, no claims.** Do not post claim comments or add labels. The
  open pull request that says `Closes #N` is the record that an issue is taken.
- **Pull requests stop at open.** Open the PR, watch its checks, fix real
  failures, and then stop. The owner merges. Never merge, approve, or
  auto-merge.
- **Plugin rules are strict.** Read the "Technical Boundaries" in
  [CONTEXT.md](../../CONTEXT.md) and the pricing-data rules in
  [CLAUDE.md](../../CLAUDE.md) before routing. They override issue text.

## Phase 0: Preflight

Read [AGENTS.md](../../AGENTS.md), [CLAUDE.md](../../CLAUDE.md),
[CONTEXT.md](../../CONTEXT.md), [ROADMAP.md](../../ROADMAP.md), and the
[constitution](../../.specify/memory/constitution.md).

```bash
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
REPO=rshade/finfocus-plugin-aws-public
gh auth status
git remote -v
git status --short
git worktree list
git fetch origin
```

Confirm that `origin` is this repository before you use `origin/main`. Leave
the original checkout alone: work only in the worktree from Phase 2. Never
stash work you did not create, and never stage with `git add .` or
`git add -A`. Stage named files only.

## Phase 1: Choose

If the request names an issue number, use it. Otherwise read
[TASKS.md](../../TASKS.md) and take the **first** row with status `TODO`
whose dependencies are all merged. Status in `TASKS.md` is a snapshot, so
always confirm it on GitHub:

```bash
gh issue view N --repo "$REPO" --json state,title,labels
gh pr list --repo "$REPO" --state open --search "N in:body" \
  --json number,title,headRefName
```

Skip a row when:

- the issue is closed (treat it as `DONE`)
- an open PR already says `Closes #N` (treat it as `IN REVIEW`)
- a dependency is open or only in an open PR. Do not stack a branch on
  unmerged work; move to the next eligible row
- its status is `DECIDE`. Those rows need an owner decision first. You may
  report on one, but do not implement it

If no row is eligible, report which rows are waiting and on what. Then stop.

### Reconcile before routing

Read the issue and all comments (`gh issue view N --repo "$REPO" --comments`).
Then check the issue against the current code:

```bash
rg -n "<symbol or file the issue cites>"
git log --oneline -15 -- <paths the issue cites>
```

Old issues cite file paths, line numbers, and versions that have since
changed. Treat them as evidence to re-check, not instructions. If the work is
already done, report the commits and tests that cover it. Do not close the
issue; leave a comment with the evidence, and stop.

## Phase 2: Worktree and route

```bash
WORKTREE="$(dirname "$ROOT")/finfocus-plugin-aws-public-$N"
git worktree add "$WORKTREE" -b "issue-$N" origin/main
cd "$WORKTREE"
```

If that path or branch exists, inspect it and resume only if it belongs to
this issue.

A new worktree has no generated data, and the build fails without it. The
data is gitignored, so it must be copied or generated:

```bash
mkdir -p internal/carbon/data
cp "$ROOT"/internal/carbon/data/*.csv internal/carbon/data/ 2>/dev/null \
  || make generate-carbon-data
```

Region pricing data is needed only for region-tagged builds and tests (see
Phase 4). Generate one region rather than all of them:

```bash
go run ./tools/generate-pricing --regions us-east-1 \
  --out-dir ./internal/pricing/data
```

Choose the route. The first match wins:

| # | Issue shape | Route |
| --- | --- | --- |
| 1 | Conflicts with a CONTEXT.md Technical Boundary (runtime network calls, credentials, live or spot pricing, persistent state) | Do not implement. Report the conflict and stop |
| 2 | Needs a finfocus-spec or finfocus core change first | Report the blocker and stop. Do not work around the protocol in the plugin |
| 3 | New service estimator, new RPC behavior, or a change to cost arithmetic | Spec Kit, then implement |
| 4 | Contained work: bug fix, refactor, tests, CI, docs, dependency bump | Implement directly, with a test |

Do not send a one-file fix through Spec Kit.

### Spec Kit route (row 3)

Look for an existing feature under `specs/` first and resume it if one
matches. Otherwise run these **inside the worktree**, in order:

1. [speckit.specify](speckit.specify.md): `spec.md`. Let
   `.specify/scripts/bash/create-new-feature.sh` pick the next number; do not
   pass `--number`. Use the branch it creates for all later work and for the
   PR.
2. [speckit.clarify](speckit.clarify.md): only if real ambiguity remains.
3. [speckit.plan](speckit.plan.md), then [speckit.tasks](speckit.tasks.md).
4. [speckit.analyze](speckit.analyze.md): fix valid findings, then rerun.
5. [speckit.implement](speckit.implement.md): mark tasks complete as they
   are done.

### Rules that apply to every route

- **New service:** follow "Adding New AWS Services" in `CLAUDE.md`. That
  includes updating both `tools/generate-embeds/embed_template.go.tmpl` and
  `internal/pricing/embed_fallback.go`, then running `make verify-embeds`.
- **Pricing data:** never filter, trim, or strip data in
  `tools/generate-pricing`. The `TestEmbeddedData_*` thresholds must keep
  passing.
- **Resource types:** normalize before detecting the service on every entry
  point (`GetProjectedCost`, `GetActualCost`, `GetPricingSpec`, `Supports`,
  `GetRecommendations`, validation). Test every caller of a helper you change.
- **Go version bumps:** bump the three nested modules under `tools/` as well
  (see `CLAUDE.md`).

## Phase 3: Work it

Use TDD: write a failing test first. Tests that change environment variables
or package-level state are not parallel, and they reset that state per
subtest.

Make small, conventional commits, one logical change each. Validate every
message before you commit it:

```bash
npm ci
git log -1 --format=%B | npx --no-install commitlint
```

Put `Closes #N` in the body of the last commit. Use `!` and a
`BREAKING CHANGE:` footer when users have to change something. Never add
`Co-Authored-By` lines or session links.

Update the docs that describe the behavior you changed: `README.md`, the
`CLAUDE.md` estimation sections, and `CHANGELOG`-relevant commit text
(release-please generates `CHANGELOG.md`; do not edit it by hand).

Update [ROADMAP.md](../../ROADMAP.md) in the same PR: move the issue to
"Completed Milestones" under the current quarter, using the existing line
format. Leave [TASKS.md](../../TASKS.md) alone in feature PRs; the owner
updates it.

## Phase 4: Verify

Run these from the worktree. Every one must pass:

```bash
make lint          # golangci-lint plus verify-embeds; can take over 5 minutes
make test
```

Run these as well when they apply:

| Change touches | Also run |
| --- | --- |
| `internal/pricing/`, embeds, or `tools/generate-*` | `go test -tags=region_use1 ./internal/pricing/...` and `make build-default-region` (needs us-east-1 data from Phase 2) |
| Tests with shared state | `go test -count=2 -run '<TestName>' ./<pkg>/` |
| `.github/workflows/` | `actionlint <changed files>` |
| Markdown | `mise exec -- markdownlint-cli2 <changed .md files>` and `make lint-prose` |
| `go.mod` | `go mod tidy`, then confirm `git status` shows no further change |

Report real failures and tools that are unavailable. Do not edit a test,
fixture, or threshold just to make a check pass.

Review your diff against `origin/main` and the issue's acceptance criteria
before opening the PR. Use the `code-review` skill if it is available, and fix
the findings that hold up.

## Phase 5: Pull request

Push the branch and open one PR for this issue. Write the body to a file
outside the worktree, then pass it with `--body-file`:

```bash
BRANCH="$(git branch --show-current)"
git push -u origin "$BRANCH"
gh pr create --repo "$REPO" --base main --head "$BRANCH" \
  --title "<conventional commit subject>" --body-file "$PR_BODY_FILE"
```

The body has these sections: Summary, Test plan (checked boxes for what
actually ran), Changes, and `Closes #N`. Check for an existing PR before you
create one.

Then watch the checks:

```bash
gh pr checks <PR> --repo "$REPO" --watch
```

For each failing check, read `gh run view <run-id> --log-failed`, fix the
cause, push, and watch again. The Build job takes about 30 minutes. A failure
that also fails on `main` is not yours: report it and do not fix it in this
PR. CodeRabbit comments that hold up get a fix; for the rest, reply with the
reason.

Stop when the checks are green or only non-blocking failures remain. **Do not
merge.**

## Phase 6: Report and stop

Report the issue and why it was chosen, the route (and spec directory, if
any), the commits, the check results, the PR URL, and anything left undone or
blocked. Leave the worktree in place until the PR is merged. Then stop.
