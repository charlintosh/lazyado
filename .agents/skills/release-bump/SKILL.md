# release-bump — Skill

## Summary

This skill documents the procedure used to perform a patch version bump: updating
`CHANGELOG.md` and `README.md`, creating an annotated git tag, and optionally
pushing the changes to the remote.

## Prerequisites

- `git` installed and configured.
- Work on the branch you want to release from (e.g. `main`).

## Quick Steps

1. Get the latest tag: `git describe --tags --abbrev=0` (fallback to `v0.0.0` if none).
2. Compute the next patch version (increment Z in `vX.Y.Z`).
3. List commits since the last tag and format them as bullets for the changelog.
4. Edit `CHANGELOG.md`: under `## [Unreleased]` insert a `## [vX.Y.Z] - YYYY-MM-DD` section with the bullets.
5. Update the version badge in `README.md` (replace the `version-...` segment).
6. `git add` modified files and `git commit -m "chore(release): vX.Y.Z"`.
7. Create an annotated tag: `git tag -a vX.Y.Z -m "release vX.Y.Z"`.
8. (Optional) Push the commit and tag: `git push origin BRANCH && git push origin vX.Y.Z`.

## Useful commands (portable examples)

Note: prefer editing `CHANGELOG.md` with your editor for best control. The commands
below are examples for convenience and include cross-platform alternatives.

POSIX shell (Linux/macOS with bash/zsh):

```sh
# Determine repository root (works anywhere inside the repo)
repo=$(git rev-parse --show-toplevel)

# Determine latest tag (fallback to v0.0.0) and compute next patch version
latest=$(git -C "$repo" describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
IFS=. read -r maj min pat <<< "${latest#v}"
new=v$maj.$min.$((pat+1))
echo "Latest: $latest  Next: $new"

# List commits since the last tag (copy into CHANGELOG)
git -C "$repo" log "$latest..HEAD" --no-merges --pretty=format:'- %s (%h)'

# Update README badge (example):
# macOS: sed -i '' -E "s/version-[0-9]+\.[0-9]+\.[0-9]+/version-${new#v}/" "$repo/README.md"
# GNU sed (Linux): sed -i -E "s/version-[0-9]+\.[0-9]+\.[0-9]+/version-${new#v}/" "$repo/README.md"

# Commit and create annotated tag
git -C "$repo" add CHANGELOG.md README.md
git -C "$repo" commit -m "chore(release): ${new#v}"
git -C "$repo" tag -a "$new" -m "release $new"

# Push (optional) -- replace BRANCH with your release branch (e.g. main)
git -C "$repo" push origin BRANCH
git -C "$repo" push origin "$new"
```

PowerShell (Windows) examples:

```powershell
# From repo root or any subfolder
$repo = git rev-parse --show-toplevel

# Latest tag and next version (simple parse)
$latest = (git -C $repo describe --tags --abbrev=0 2>$null) -or 'v0.0.0'
$parts = $latest.TrimStart('v').Split('.')
$new = "v{0}.{1}.{2}" -f $parts[0], $parts[1], ([int]$parts[2] + 1)
Write-Output "Latest: $latest  Next: $new"

# List commits since last tag
git -C $repo log "$latest..HEAD" --no-merges --pretty=format:'- %s (%h)'

# Update README (PowerShell replace)
# (Get-Content "$repo/README.md") -replace 'version-\d+\.\d+\.\d+', "version-$($new.TrimStart('v'))" | Set-Content "$repo/README.md"

# Commit and tag
git -C $repo add CHANGELOG.md README.md
git -C $repo commit -m "chore(release): $($new.TrimStart('v'))"
git -C $repo tag -a $new -m "release $new"

# Push (optional)
git -C $repo push origin BRANCH
git -C $repo push origin $new
```

## Automation tips

- Use a small script (bash, PowerShell, or Node/Python) if you perform releases often.
- Prefer manual verification of `CHANGELOG.md` before committing to avoid formatting mistakes.
- Use annotated tags (`git tag -a`) to keep release messages.

End of portable examples.

## Notes & Best Practices

- Ensure `CHANGELOG.md` follows the project's Keep a Changelog format.
- Run quick tests or builds if needed before releasing.
- Use annotated tags (`-a`) to preserve message and metadata.
- If the remote branch is protected, open a PR instead of pushing directly.

Example of the content we added in the recent bump (reference):

```
## [v0.1.3] - 2026-02-19

### Added

- refactor: details and filter panels to use viewport for improved rendering (eab0bab)
- feat: add scripts to sync .agents/skills/ to supported folders via symlinks (37e0f68)
...
```
