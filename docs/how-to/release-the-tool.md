# Release the Tool

For maintainers. The version is in `VERSION`. Releases are made from `main` with `just`, and CI builds them.

1. Make sure `main` is green in CI, and that your checkout is on `main` and up to date.
2. Release:

   ```bash
   just release          # patch: 0.1.3 -> 0.1.4
   just release minor    # 0.1.3 -> 0.2.0
   just release major    # 0.1.3 -> 1.0.0
   ```

   An `X.Y.Z-SNAPSHOT` in `VERSION` releases as `X.Y.Z`, whatever the level. `just full-release` is the same recipe, under the name the other streampack projects use.

`just release` refuses to run from a branch other than `main`, from a `main` behind `origin/main`, with uncommitted changes, or when the tag already exists. It runs `just test`, writes the new version to `VERSION`, commits it ("updating release version"), tags `vX.Y.Z`, and pushes `main` and the tag together. If the push fails, it says how to push both by hand.

CI then tests the tag, checks it against `VERSION`, builds static binaries for linux and darwin, amd64 and arm64, with the version stamped in (`streampack-setup version`), and attaches them to a GitHub release with `SHA256SUMS`. Watch it at the repository's Actions page.

The streampack docs link to `releases/latest/download/…`, so a new release is what they download from then on.

## Building Locally

```bash
just build
./streampack-setup version
```

`just build` stamps the binary with `VERSION` as it stands, `-SNAPSHOT` included.
