default:
    @just --list

# Run the tests, vet, and the gofmt check, as CI does
test:
    #!/usr/bin/env bash
    set -euo pipefail
    unformatted="$(gofmt -l .)"
    if [[ -n "$unformatted" ]]; then
      printf 'Not gofmt-formatted:\n%s\n' "$unformatted" >&2
      exit 1
    fi
    go vet ./...
    go test -race ./...

# Build ./streampack-setup for this machine, stamped with VERSION
build:
    go build -trimpath -ldflags "-X main.version=$(cat VERSION)" -o streampack-setup ./cmd/streampack-setup

# Bumps VERSION (patch, minor or major; an X.Y.Z-SNAPSHOT releases as X.Y.Z), commits, tags vX.Y.Z,
# and pushes main and the tag together. CI builds the binaries from the tag and attaches them to a
# GitHub release.
# Release: bump, commit, tag and push
release level="patch": _on-main
    #!/usr/bin/env bash
    set -euo pipefail
    level="{{level}}"
    files=(VERSION)
    case "$level" in
      patch|minor|major) ;;
      *) echo "release level must be one of: patch, minor, major" >&2; exit 1 ;;
    esac

    if [[ -n "$(git status --porcelain)" ]]; then
      echo "The working tree has uncommitted changes (a failed release?). Commit or restore them first." >&2
      exit 1
    fi
    git fetch -q --tags origin

    current="$(tr -d '[:space:]' < VERSION)"
    if [[ "$current" == *-SNAPSHOT ]]; then
      next="${current%-SNAPSHOT}"
    else
      IFS=. read -r major minor patch <<<"$current"
      if [[ -z "${major:-}" || -z "${minor:-}" || -z "${patch:-}" ]]; then
        printf 'current version is not semantic: <%s>\n' "$current" >&2
        exit 1
      fi
      case "$level" in
        patch) next="${major}.${minor}.$((patch + 1))" ;;
        minor) next="${major}.$((minor + 1)).0" ;;
        major) next="$((major + 1)).0.0" ;;
      esac
    fi
    if git rev-parse -q --verify "refs/tags/v$next" >/dev/null; then
      echo "v$next is already tagged." >&2
      exit 1
    fi

    just test >/dev/null

    echo "Releasing $current -> $next"
    trap 'status=$?; if [[ $status -ne 0 && -z "${committed:-}" ]]; then git checkout -- "${files[@]}"; echo "Release failed; VERSION restored to $current." >&2; fi' EXIT
    printf '%s\n' "$next" > VERSION
    git commit -q -m "updating release version" -- "${files[@]}"
    committed=1
    git tag -a "v$next" -m "streampack-setup $next"
    if ! git push -q --atomic origin main "v$next"; then
      echo "Push failed; main has the release commit and v$next locally. Push both with: git push --atomic origin main v$next" >&2
      exit 1
    fi
    echo "Released $next: pushed main and v$next. CI builds the binaries:"
    echo "  https://github.com/streampack-dev/setup/actions"

# Releases come from main, up to date with origin's: never from a feature branch.
_on-main:
    #!/usr/bin/env bash
    set -euo pipefail
    branch="$(git branch --show-current)"
    if [[ "$branch" != "main" ]]; then
      echo "Release from main, not ${branch:-a detached HEAD}." >&2
      exit 1
    fi
    git fetch -q origin main
    if [[ -n "$(git rev-list HEAD..origin/main)" ]]; then
      echo "main is behind origin/main. Pull first." >&2
      exit 1
    fi

# Here just release already commits, tags and pushes, and CI publishes.
# The whole release, under the name the other projects use
full-release level="patch":
    just release {{level}}
