# Release the Tool

For maintainers. Releases are made by tagging; CI builds them.

1. Make sure `main` is green in CI.
2. Tag the commit and push the tag:

   ```bash
   git tag v1.2.0
   git push origin v1.2.0
   ```

CI tests the tag, builds static binaries for linux and darwin, amd64 and arm64, with the version stamped in (`streampack-setup version`), and attaches them to a GitHub release with `SHA256SUMS`.

The streampack docs link to `releases/latest/download/…`, so a new release is what they download from then on.
