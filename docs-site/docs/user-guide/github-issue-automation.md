# GitHub Issue Automation Guide

k13d can turn a labeled GitHub issue into a local development run, commit, push, pull request, CI wait, and optional preview deployment.

This flow is intentionally gated. By default, automation only runs when the issue author is a member of the repository owner organization and the issue has the configured trigger label.

GitHub repository settings still decide who can open an issue. k13d controls whether that issue is allowed to execute the local automation runner.

## Who Can Trigger Automation

- The issue author must be a member of the GitHub organization that owns the repository.
- The repository must match `github_automation.allowed_repositories`.
- The issue must have the trigger label, usually `codex:auto`.
- Review and merge commands are accepted from `issue_comment` webhooks only when the commenter is an organization member.
- Webhook signatures must pass `X-Hub-Signature-256` verification when a webhook secret is configured.

For organization membership checks, k13d first trusts GitHub webhook `author_association` values of `OWNER` or `MEMBER`. If that value is absent or inconclusive, k13d verifies membership through the GitHub API. Use a token that can read organization membership, for example a fine-grained token with access to the repository and organization metadata, or a classic token with `repo` and `read:org` where appropriate.

When a trusted issue is accepted, k13d can mention organization members in the first automation comment. Built-in issue reviews and PR review wrappers are written in Korean by default. External agent commands also receive `K13D_GHA_REVIEW_LANGUAGE=ko`, so include that variable in your Codex, Claude Code, or Gemini prompt if you want the model-generated review body to stay Korean too.

## Before You Create an Issue

Use automation for small, reviewable changes. A good issue should be scoped enough that the local runner can make one focused PR.

The repository provides friendly GitHub Issue Forms:

- `Codex 개발 요청`: use this when you want issue-driven development automation
- `버그 제보`: use this when something behaves incorrectly
- `기능 제안`: use this when you have an improvement idea but do not need automation yet

Include:

- The problem or goal.
- The expected behavior after the change.
- Files, pages, commands, or screenshots that give useful context.
- Validation expectations such as `go test ./pkg/...`, docs build, or Web UI E2E when relevant.
- Clear constraints, especially when the change must avoid touching CI, deployment, credentials, or public APIs.

Do not include:

- Secrets, kubeconfigs, passwords, API keys, or private tokens.
- Broad requests such as "refactor everything" without acceptance criteria.
- Production-destructive instructions.

## Natural Language Issue Template

```markdown
## Goal
Describe the user-visible outcome.

## Context
Explain the current behavior, error, or relevant page/command.

## Requested Change
- Keep each item small and testable.
- Mention exact files or screens if known.

## Acceptance Criteria
- What should be true when the PR is complete?
- What command or UI flow should pass?

## Validation
- Suggested tests or checks.

## Safety Notes
- Anything the runner must not touch.
```

If you use the `Codex 개발 요청` Issue Form, GitHub asks for the same information with friendlier questions. The form intentionally applies only a triage label. An organization member should review the request first, then add `codex:auto` when it is safe to start automation.

## Trigger Steps

1. Create an issue with the `Codex 개발 요청` form.
2. Confirm the issue is safe, scoped, and does not contain secrets.
3. Apply the `codex:auto` label only when you want k13d to execute it.
4. Confirm k13d assigned the issue author and mentioned the organization reviewers.
5. Wait for k13d to comment with a result.
6. Review the linked PR, requested reviewers, CI result, and preview link before merging.
7. If you want another automated pass, comment `k13d 코드리뷰 해줘` on the issue to run the configured Codex review command again.
8. If `allow_issue_merge` is enabled, comment `k13d merge 해줘` after approval; k13d merges the PR and closes the issue as completed.

If you need to edit the request, remove `codex:auto`, update the issue, then re-apply the label. k13d uses a stable branch name such as `codex/issue-123`, so the next run continues on the same branch and reuses the existing open PR instead of creating another PR.

## Configuration Example

```yaml
github_automation:
  enabled: true
  webhook_secret: ${K13D_GITHUB_AUTOMATION_WEBHOOK_SECRET}
  personal_access_token: ${GITHUB_TOKEN}
  allowed_repositories:
    - cloudbro-kube-ai/k13d
  require_author_org_member: true
  mention_org_members: true
  mention_max_members: 20
  review_language: ko
  trigger_label: codex:auto
  repo_path: /absolute/path/to/k13d
  worktree_root: ~/.cache/k13d/github-automation
  development_command: ./scripts/run-agent-dev.sh
  review_command: ./scripts/run-agent-review.sh
  wait_for_ci: true
  auto_commit: true
  auto_push: true
  auto_create_pr: true
  allow_issue_merge: true
  merge_method: squash
  auto_deploy_preview: true
  deploy_preview_command: ./scripts/deploy-preview.sh
  preview_url_base: https://fingerscore.net
  preview_path_prefix: /previews
```

`require_author_org_member` should stay enabled for public or semi-public repositories. If it is disabled, any user who can create or label issues in an allowed repository can trigger the local automation runner.

`mention_org_members` needs a token that can list organization members. `mention_max_members` caps the number of `@mentions` in one comment so a large organization does not get noisy.

When a trusted issue is accepted, k13d assigns the issue author to the issue. After a PR exists, k13d requests organization members as PR reviewers, capped by `mention_max_members`. This keeps responsibility clear: the author owns the issue, while the organization reviews the generated code.

When `review_command` is set, k13d runs it after development and posts the output as a PR Review. The repository includes `scripts/run-agent-review.sh`, which wraps `codex exec review` and asks Codex to write a Korean review focused on bugs, regressions, security, concurrency, and missing tests. Organization members can also re-run that review from the issue by commenting `k13d 코드리뷰 해줘`, `k13d 리뷰해줘`, or `k13d review`.

If `allow_issue_merge` is enabled, an organization member can complete the flow from the issue by commenting a natural-language merge request such as `k13d merge 해줘`, `k13d main에 merge 해줘`, or `k13d 병합해줘`. k13d finds the stable issue branch PR, asks GitHub to merge it using `merge_method`, closes the issue as completed after a successful merge, and posts a Korean success or failure comment. Branch protection, required reviews, and CI rules still apply on GitHub.

GitHub tokens are kept server-side. k13d does not pass `GITHUB_TOKEN`, `GH_TOKEN`, `K13D_GITHUB_AUTOMATION_TOKEN`, or similar GitHub token env vars to development, review, or preview deployment commands. Captured command output and admin status payloads also redact configured GitHub token and webhook secret values.

When preview deployment is enabled, the deploy command should print `K13D_PREVIEW_TARGET=http://127.0.0.1:<port>` after the branch build is running locally. k13d exposes that branch through `preview_url_base + preview_path_prefix`, for example `https://fingerscore.net/previews/codex-issue-123/`, and includes that human verification link in both the final issue comment and the generated PR comment after CI/CD finishes.

## Troubleshooting

| Symptom | What To Check |
|---------|---------------|
| Issue is ignored | Confirm `codex:auto` is present and the author is an organization member |
| Membership verification fails | Confirm `personal_access_token` can read organization membership |
| Organization members are not mentioned | Confirm `mention_org_members` is enabled and the token can list org members |
| Assignee or reviewer request fails | Confirm the token has issue write and pull request write permissions |
| No PR is created | Confirm `auto_push`, `auto_create_pr`, and `personal_access_token` are configured |
| Multiple attempts create confusion | Confirm the issue still maps to the stable `codex/issue-<number>` branch and that any older manual PRs use that branch |
| Review command is ignored | Confirm `review_command` is configured, the comment contains `k13d` and a review phrase, and the commenter is an organization member |
| Review command fails | Confirm Codex CLI is installed/authenticated on the k13d host and that `scripts/run-agent-review.sh` can run inside the issue worktree |
| Merge command is ignored | Confirm `allow_issue_merge: true`, the comment contains `k13d` and a merge phrase, and the commenter is an organization member |
| Merge command fails | Check branch protection, required reviews, CI status, and whether the token has pull request write permissions |
| PR merged but issue stayed open | Confirm the token has issue write permission; k13d reports this warning in the merge completion comment |
| No preview link appears | Confirm `auto_deploy_preview` is enabled and `deploy_preview_command` prints `K13D_PREVIEW_TARGET=...` |
| Preview link is missing from the PR | Confirm the token can comment on pull requests; k13d posts the same verification path on the PR after CI/CD completes |
| CI never completes | Check GitHub Actions on the generated branch and `ci_wait_timeout_seconds` |
| The request is too broad | Split the issue into smaller issues before applying `codex:auto` |
