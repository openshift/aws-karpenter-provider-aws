# Contributing to Karpenter AWS Provider (OpenShift Downstream)

This document covers contribution guidelines specific to the OpenShift downstream fork of the [Karpenter AWS Provider](https://github.com/aws/karpenter-provider-aws). For upstream contribution guidelines, see the [upstream documentation](https://karpenter.sh/docs/contributing/).

## Related Resources

| Resource | Link |
|----------|------|
| Upstream repo | [aws/karpenter-provider-aws](https://github.com/aws/karpenter-provider-aws) |
| Core Karpenter repo | [openshift/kubernetes-sigs-karpenter](https://github.com/openshift/kubernetes-sigs-karpenter) |
| Operator repo | [openshift/karpenter-operator](https://github.com/openshift/karpenter-operator) |
| CI configuration | [openshift/release/.../aws-karpenter-provider-aws/](https://github.com/openshift/release/tree/master/ci-operator/config/openshift/aws-karpenter-provider-aws) |
| AI guidance | [AGENTS.md](AGENTS.md) |
| OpenShift docs | [Karpenter documentation](https://docs.openshift.com/) |

## Review and Approval Policy

Every change in every pull request must be understood and approved by two humans. This can be the PR author and a reviewer, or — if the author used an AI tool and does not fully understand the contents of the PR — two human reviewers.

**Exception:** PRs authored by deterministic automation tools that are part of our CI and related systems (whose code has been reviewed by the OpenShift engineering org) can be merged with a single human review.

Every change should be closely scrutinized for bugs. Our software is complex with many interdependencies. Review changes from multiple angles:

- **Product architecture**: Does this fit the intended design of Karpenter and OpenShift?
- **Security**: Are there new attack surfaces, credential handling issues, or privilege escalations? Karpenter provisions nodes and manages AWS resources — IAM permissions and EC2 operations must be carefully reviewed.
- **Thread safety**: The provider uses concurrent AWS API calls and caching — are shared resources properly synchronized?
- **Regressions**: Could this break existing instance provisioning, interruption handling, or pricing/capacity queries?
- **Effects on other components**: How does this impact the Karpenter operator, the core Karpenter controllers, or node lifecycle management?

## Upstream Commit Convention

This is a downstream fork. All non-upstream commits must use one of the following prefixes to ensure changes are not lost during the next upstream rebase:

- `UPSTREAM: <carry>:` -- A change that should be kept (carried) indefinitely, or as long as it makes sense to do so
- `UPSTREAM: <drop>:` -- A change that should be discarded during the next rebase cycle
- `UPSTREAM: 1234:` -- A change carried until the rebase includes upstream PR #1234

Examples:
```text
UPSTREAM: <carry>: Add OpenShift-specific RHEL AMI family support
UPSTREAM: <drop>: Pin AWS SDK version until upstream compatibility fix
UPSTREAM: 5678: Backport fix for EC2NodeClass validation race
```

Commit prefixes are for individual commits, not PR titles.

## Upstream-First Policy

New feature work should be directed to the [upstream Karpenter AWS provider project](https://github.com/aws/karpenter-provider-aws). Downstream-only features are discouraged due to the ongoing cost of maintaining them through each rebase cycle. If a downstream-only change is necessary, use the `UPSTREAM: <carry>:` prefix and include a comment in the PR explaining why it cannot go upstream.

## PR Title Convention

PR titles should be prefixed with a Jira ticket reference:
```text
AUTOSCALE-123: Fix the whatsit in the thingamajig
OCPBUGS-456: Correct nil pointer in instance provider shutdown
NO-JIRA: Update Go module dependencies
```

The Jira prefix goes in the **PR title**. The upstream commit prefix goes in the **commit message**.

## PR Workflow

This repo uses [OpenShift CI (Prow)](https://docs.ci.openshift.org/) for continuous integration. GitHub Actions workflows in this repo are from upstream and are **not used** for our CI. PRs are automatically merged once all required tests pass and the correct labels are present.

### Required labels for merge

- `lgtm` — Added by a reviewer via the `/lgtm` command. Any developer from the OpenShift org can add this after reviewing the PR.
- `approved` — Added by an approver listed in the [OWNERS](OWNERS) file via the `/approve` command.
- `verified` — Added by anyone in the OpenShift org, but typically by the PR author.

### Useful commands

Comment these on the PR:

| Command | Effect |
|---------|--------|
| `/lgtm` | Add the `lgtm` label after reviewing. In repos using [LGTM mode](https://docs.ci.openshift.org/how-tos/creating-a-pipeline/#the-pipeline-required-command), this also triggers E2E and other second-stage tests. |
| `/lgtm cancel` | Remove the `lgtm` label |
| `/approve` | Add the `approved` label (OWNERS approvers only) |
| `/pipeline required` | Manually trigger all required second-stage tests (e.g., E2Es) without waiting for `/lgtm` |
| `/retest` | Re-run all failed required tests |
| `/retest-required` | Re-run only the failed required tests |
| `/test <test-name>` | Run a specific test, e.g. `/test e2e-hypershift` |
| `/hold` | Prevent the PR from being merged |
| `/hold cancel` | Remove the hold and allow merging |
| `/verified` | Mark the PR as verified |
| `/cherry-pick release-4.18` | Create a cherry-pick PR to a release branch |

### LGTM mode and E2E tests

Repos enrolled in [LGTM mode](https://docs.ci.openshift.org/how-tos/creating-a-pipeline/#the-pipeline-required-command) defer second-stage tests (such as E2Es) until the `/lgtm` label is applied. This avoids wasting CI resources on PRs that haven't been reviewed yet. If you need to run E2Es before getting `/lgtm` (e.g., to validate before requesting review), use `/pipeline required`.

### Preventing premature merges

- Add the `WIP:` prefix to the PR title (e.g., `WIP: AUTOSCALE-123: Work in progress`). Prow adds the `do-not-merge/work-in-progress` label automatically.
- Use `/hold` to temporarily block merging while awaiting additional review or testing.

## Test Expectations

PRs should include tests to verify correctness and prevent future regressions:

- **Unit tests**: Required for new logic, bug fixes, and behavior changes. Run with `make test`.
- **E2E tests**: Expected for new features or significant behavior changes. The OpenShift CI runs E2E tests against real AWS infrastructure.

## Verified Label

Use `/verified` to indicate changes have been verified. Examples:
```text
/verified
/verified by e2e-aws-ovn
/verified by unit tests
/verified by E2Es
/verified later
```

## Generated Code

The following files are generated and should never be hand-edited:

| File(s) | Generator | Regenerate with |
|---------|-----------|-----------------|
| `pkg/apis/v1/zz_generated.deepcopy.go` | controller-gen | `make codegen` |
| `charts/karpenter/crds/*.yaml` | controller-gen | upstream task, generally not modified downstream |

After modifying API types, regenerate and commit the results in the same PR.

## Development Quick Reference

| Task | Command |
|------|---------|
| Build controller binary | `make binary` |
| Build controller image | `make image` |
| Run unit tests | `make test` |
| Run tests with race detection | `TEST_FLAGS="-race" make test` |
| Run linter | `make verify` |
| Format code | `go fmt ./...` |
| Generate deepcopy code | `make codegen` |
| Deploy to local cluster | `make apply` |
| Run vulnerability checks | `make vulncheck` |
| Verify licenses | `make licenses` |
| Full presubmit checks | `make presubmit` |

## Pre-Submit Checklist

Before requesting review:

1. `make binary` — Verify the code compiles
2. `make test` — Run unit tests
3. `make verify` — Run linters and checks
4. `make codegen && git diff --exit-code` — Ensure generated files are up to date
5. Review your diff for secrets, credentials, or debug code
6. Address any [CodeRabbit](https://coderabbit.ai/) review feedback — as a courtesy to the human reviewer who follows. Responding with an explanation of why you're not acting on a suggestion is fine; the goal is to resolve straightforward issues so human reviewers can focus on the substantive aspects.

## Code Style

- Run `go fmt ./...` before committing
- Follow Go conventions for error strings: lowercase, no trailing punctuation, wrap with `fmt.Errorf("context: %w", err)`
- Use structured logging with the klog/logr interface: constant messages, key-value pairs in lowerCamelCase
- Import ordering: stdlib, external packages, internal packages (separated by blank lines)

## AI Code Review

Our repos use CodeRabbit for automated AI code review. CodeRabbit will post review comments on your PR automatically.

As a courtesy to the human reviewer who follows, please address CodeRabbit's feedback before requesting human review. You do not need to accept every suggestion — responding with an explanation of why you are not taking action on a comment is perfectly acceptable. The goal is to resolve straightforward issues so that human reviewers can focus on the substantive aspects of the change.
