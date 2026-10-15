# Install the Tool

The tool lives in the deployment directory, with the files it writes. You run it from there as `./streampack-setup`.

## Before You Start

1. Install Docker.
   - On a Mac, install Docker Desktop and start it.
   - On Linux, install Docker Engine and the Compose plugin. Add your user to the `docker` group.
2. Make sure that Docker works:

   ```bash
   docker compose version
   ```

3. Find your processor type:

   ```bash
   uname -m
   ```

   The result is `arm64`, `aarch64` or `x86_64`. Use it to choose a section below.

## Mac with Apple Silicon (`arm64`)

1. Make the deployment directory, and go into it:

   ```bash
   mkdir ~/streampack && cd ~/streampack
   ```

2. Download the tool and its checksums:

   ```bash
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/streampack-setup-darwin-arm64
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/SHA256SUMS
   ```

3. Check the download. The result must be `streampack-setup-darwin-arm64: OK`.

   ```bash
   shasum -a 256 --check --ignore-missing SHA256SUMS
   ```

4. Name the tool, and make it executable:

   ```bash
   mv streampack-setup-darwin-arm64 streampack-setup
   chmod +x streampack-setup
   rm SHA256SUMS
   ```

## Mac with Intel (`x86_64`)

1. Make the deployment directory, and go into it:

   ```bash
   mkdir ~/streampack && cd ~/streampack
   ```

2. Download the tool and its checksums:

   ```bash
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/streampack-setup-darwin-amd64
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/SHA256SUMS
   ```

3. Check the download. The result must be `streampack-setup-darwin-amd64: OK`.

   ```bash
   shasum -a 256 --check --ignore-missing SHA256SUMS
   ```

4. Name the tool, and make it executable:

   ```bash
   mv streampack-setup-darwin-amd64 streampack-setup
   chmod +x streampack-setup
   rm SHA256SUMS
   ```

## Linux on x86-64 (`x86_64`)

1. Make the deployment directory, and go into it:

   ```bash
   mkdir ~/streampack && cd ~/streampack
   ```

2. Download the tool and its checksums:

   ```bash
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/streampack-setup-linux-amd64
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/SHA256SUMS
   ```

3. Check the download. The result must be `streampack-setup-linux-amd64: OK`.

   ```bash
   sha256sum --check --ignore-missing SHA256SUMS
   ```

4. Name the tool, and make it executable:

   ```bash
   mv streampack-setup-linux-amd64 streampack-setup
   chmod +x streampack-setup
   rm SHA256SUMS
   ```

## Linux on ARM (`aarch64`)

1. Make the deployment directory, and go into it:

   ```bash
   mkdir ~/streampack && cd ~/streampack
   ```

2. Download the tool and its checksums:

   ```bash
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/streampack-setup-linux-arm64
   curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/SHA256SUMS
   ```

3. Check the download. The result must be `streampack-setup-linux-arm64: OK`.

   ```bash
   sha256sum --check --ignore-missing SHA256SUMS
   ```

4. Name the tool, and make it executable:

   ```bash
   mv streampack-setup-linux-arm64 streampack-setup
   chmod +x streampack-setup
   rm SHA256SUMS
   ```

## Next

Run the tool from the deployment directory:

```bash
./streampack-setup
```

For a guided first run, follow [Your First Deployment](../tutorials/first-deployment.md).

## Build It Instead

If you have Go 1.26 and `just`, build the tool from a clone of this repository, and copy it into the deployment directory:

```bash
just build
cp streampack-setup ~/streampack/
```
