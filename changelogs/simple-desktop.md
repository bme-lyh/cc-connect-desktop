## Simple desktop derivative — 2026-09-01

### Features

- Zero-configuration startup with a loopback-only management service and no synthetic project.
- Five-step first-run wizard for agent health, workspace, safe permissions, platform credentials, connection setup, and controlled restart.
- Bot-card home page, full registered-platform setup catalog, and existing projects/providers/skills/cron retained under Advanced Settings.
- Windows Credential Manager-backed secret references, API/config redaction, atomic configuration writes, previous-config recovery, and per-bot failure isolation.
- Windows desktop launcher, per-user installer, Task Scheduler integration, portable ZIP/SHA-256 packaging, and GitHub Actions workflow.

### Compatibility

- Existing TOML projects and the `cc-connect`, `cc-connect web`, and `cc-connect daemon` command families remain supported.
- Existing plaintext platform credentials remain readable and are migrated to the system credential store when a simple bot is saved.
- This derivative preserves the upstream module/binary names, MIT license, and third-party copyright notices.
