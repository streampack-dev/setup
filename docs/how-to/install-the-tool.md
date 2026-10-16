# Install the Tool

The tool lives in the deployment directory, with the files it writes. You run it from there as `./streampack-setup`.

## Get a Container Engine

The tool runs the server in containers. It needs a container engine and Compose. Docker and Podman both work. The tool finds the one that is installed and running, and uses it. Use the section for your system.

### Mac with Docker Desktop

1. Install Docker Desktop from <https://www.docker.com/products/docker-desktop/>.
2. Start Docker Desktop.
3. Make sure that Docker works:

   ```bash
   docker compose version
   docker run --rm hello-world
   ```

### Mac with Podman

1. Install Podman and podman-compose with Homebrew:

   ```bash
   brew install podman podman-compose
   ```

2. Make and start the Podman virtual machine:

   ```bash
   podman machine init
   podman machine start
   ```

3. Make sure that Podman works:

   ```bash
   podman-compose version
   podman run --rm hello-world
   ```

### Linux with Docker Engine

1. Install Docker Engine and the Compose plugin. Follow the steps for your distribution at <https://docs.docker.com/engine/install/>.
2. Add your user to the `docker` group:

   ```bash
   sudo usermod -aG docker "$USER"
   ```

3. Log out, and log in again. The group change starts at your next login.
4. Make sure that Docker works:

   ```bash
   docker compose version
   docker run --rm hello-world
   ```

### Linux with Podman (Rocky Linux, RHEL, Fedora)

Podman runs containers as your own user. You do not change any group.

1. Install Podman:

   ```bash
   sudo dnf install podman
   ```

2. Install podman-compose. On Rocky Linux and RHEL, it is in EPEL:

   ```bash
   sudo dnf install epel-release
   sudo dnf install podman-compose
   ```

   On Fedora, do only the second command.

3. Check the podman-compose version. It must be 1.2.0 or later:

   ```bash
   podman-compose version
   ```

   If the version is older, remove the package and install podman-compose with pip. The tool does not start with an older version, because an older version gives the server the quotes in `.env` as part of each value.

   ```bash
   sudo dnf remove podman-compose
   pip3 install --user podman-compose
   ```

   pip puts the command in `~/.local/bin`. That directory must be on your `PATH`.

4. Let your containers start again after the computer restarts:

   ```bash
   loginctl enable-linger "$USER"
   systemctl --user enable --now podman-restart.service
   ```

5. Make sure that Podman works:

   ```bash
   podman run --rm hello-world
   ```

## Before You Start

1. Get a container engine. See [Get a Container Engine](#get-a-container-engine).
2. If you changed your groups or your `PATH`, open a new terminal.
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

5. Check the version. Run:

   ```bash
   ./streampack-setup --version
   ```

   The result is the name of the tool and its version, for example `streampack-setup 0.1.3`. The tool also writes this line first each time it runs.

   On a normal run, the next line tells which container engine the tool uses, for example `Containers: podman, with podman-compose`.

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

5. Check the version. Run:

   ```bash
   ./streampack-setup --version
   ```

   The result is the name of the tool and its version, for example `streampack-setup 0.1.3`. The tool also writes this line first each time it runs.

   On a normal run, the next line tells which container engine the tool uses, for example `Containers: podman, with podman-compose`.

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

5. Check the version. Run:

   ```bash
   ./streampack-setup --version
   ```

   The result is the name of the tool and its version, for example `streampack-setup 0.1.3`. The tool also writes this line first each time it runs.

   On a normal run, the next line tells which container engine the tool uses, for example `Containers: podman, with podman-compose`.

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

5. Check the version. Run:

   ```bash
   ./streampack-setup --version
   ```

   The result is the name of the tool and its version, for example `streampack-setup 0.1.3`. The tool also writes this line first each time it runs.

   On a normal run, the next line tells which container engine the tool uses, for example `Containers: podman, with podman-compose`.

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
