# Contributing

## Pull Request Policy

Every behavior change starts from one GitHub issue with observable acceptance
criteria. Keep a Pull Request focused on that issue; split unrelated structural
and behavioral changes into separate Pull Requests.

### Issue completion

When a Pull Request fully completes an issue, its description **must** include
one GitHub closing keyword on its own line:

```text
Closes #123
```

`Fixes #123` and `Resolves #123` are also accepted. GitHub closes the linked
issue when that Pull Request is merged into the default branch.

Do not use a closing keyword for work that only contributes to an issue. Use
`Relates to #123` instead, and leave the issue open until every acceptance
criterion is complete. After merge, confirm that each completed issue is
closed; if GitHub did not close it automatically, close it manually only after
checking the criteria and verification result.

### Pull Request requirements

- State the user-visible change in the summary.
- Link every related issue and use a closing keyword for each fully completed
  one.
- Copy the issue's acceptance criteria into the Pull Request and mark each one
  only when it is satisfied.
- Add or update tests before production behavior changes. Do not weaken,
  skip, or remove tests to make a change pass.
- Update the relevant API, domain, configuration, architecture, or development
  documentation when its contract changes.
- Run `make verify` and report the result before requesting review or merge.
- Do not merge a Pull Request with an unchecked acceptance criterion or failed
  required verification.

### Pull Request description

Use the repository template. In particular, replace the example issue number
with the actual issue number before merging. For example:

```md
## Linked issue

Closes #23

## Verification

- [x] `make verify`
```

### After merge

1. Confirm the Pull Request merged into `main`.
2. Confirm each issue referenced by `Closes`, `Fixes`, or `Resolves` is closed.
3. If the issue remains open because the closing reference was omitted, compare
   the merged change with its acceptance criteria before closing it manually.
