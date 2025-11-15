# Monitor CLI Usage Guide

**📚 Navigation:** [🏠 Main README](../README.md) | [🚀 Production Deployment](production-deployment.md) | [🗄️ PostgreSQL Setup](postgresql-setup.md) | [🌐 Nginx Setup](nginx-setup.md)

The monitor CLI (`cmd/monitor`) is a standalone terminal UI for inspecting local or remote system metrics. Perfect for server administration and troubleshooting.

## 1. Build a Linux Binary
1. Ensure Docker Desktop/Engine is running.
2. From the repository root run:
   ```bash
   ./scripts/build-monitor-cli-linux.sh
   ```
   - A reusable `golang:1.24-bullseye` container (`build-go-linux`) downloads dependencies and cross-compiles `./cmd/monitor` for `linux/amd64` with CGO enabled.
   - The resulting binary is written to `bin/monitor-cli-linux` on your host.
   - Re-run the script any time you need a fresh build; it reuses the same container for faster builds. Remove it with `docker rm -f build-go-linux` if desired.

## 2. Install System-Wide on a Server
1. Copy the binary to your server:
   ```bash
   scp bin/monitor-cli-linux user@server:/tmp/monitor-cli
   ```
2. SSH into the server and install it into a directory on `$PATH` (e.g., `/usr/local/bin`):
   ```bash
   ssh user@server
   sudo mv /tmp/monitor-cli /usr/local/bin/monitor-cli
   sudo chown root:root /usr/local/bin/monitor-cli
   sudo chmod 0755 /usr/local/bin/monitor-cli
   ```
3. Verify that the command is available everywhere on the server:
   ```bash
   which monitor-cli
   monitor-cli --refresh 5s            # runs in local mode
   monitor-cli --url http://127.0.0.1:3500/monitoring --token <JWT>
   ```
   Use `Ctrl+C` to exit the UI. Add `--log-level info` if you need verbose logs (stderr) while troubleshooting.

## 3. Updating the CLI
1. Rebuild on your workstation (`./scripts/build-monitor-cli-linux.sh`).
2. Copy the new binary to the server with a temporary name (e.g., `/tmp/monitor-cli.new`).
3. Replace the existing binary atomically:
   ```bash
   sudo install -m 0755 /tmp/monitor-cli.new /usr/local/bin/monitor-cli
   ```
4. Re-run `monitor-cli --version` (or `--help`) to confirm the new build is active.

Keeping the binary in `/usr/local/bin` (or another PATH directory) makes it universally accessible to all users and automation scripts on the host without touching shell profiles.
