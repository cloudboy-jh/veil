# Veil — Main Spec

> TUI-first encrypted secret manager for developers. No SaaS, no subscriptions, no bullshit.

---

## Overview

Veil is a terminal-native tool for managing API keys and environment variables across projects and machines. Secrets are encrypted locally with age, synced via GitHub Gist, and injected at runtime — plaintext never touches disk.

Veil is the source of truth for your secrets. It is not a `.env` file editor. It is not a file manager. It is not a vault server.

---

## Stack

| Layer | Tool |
|---|---|
| Language | Go |
| TUI Framework | Bubble Tea |
| TUI Styling | Lip Gloss |
| TUI Components | Bubbles (table, textinput, list, filepicker, spinner, help, textarea, viewport, paginator, key) |
| Encryption | age (filippo.io/age) |
| Key Storage | User choice during init: OS keychain (go-keyring) or local file with strict permissions |
| Data Format | JSON → age encrypted |
| Sync | GitHub Gist API (one gist, all projects as files) |
| Auth | GitHub OAuth device flow + QR code in terminal |
| Token Storage | System credential store / gh CLI token |
| QR Rendering | skip2/go-qrcode |
| Distribution | goreleaser, Homebrew tap, Scoop, go install, curl/irm install scripts |
| Platforms | macOS + Linux + Windows |

---

## Branding

**Name:** Veil

**Logo:** Dotted/stippled key icon (dot matrix style, monochrome)

**Color Palette:**

| Role | Color | Hex |
|---|---|---|
| Primary accent | Violet | `#8B5CF6` |
| Text | Soft white | `#F8FAFC` |
| Muted/secondary | Cool slate | `#94A3B8` |
| Background | Charcoal | `#171717` |
| Warning/reveal | Amber | `#F59E0B` |
| Success/synced | Emerald | `#10B981` |

---

## Data Model

### Store structure

```
~/.veil/
  config.json          # projects list, settings, machine info
  store/
    ld5.json.age       # encrypted secret bundle per project
    porter.json.age
    churn.json.age
```

### Decrypted project JSON schema

```json
{
  "project": "ld5",
  "path": "/Users/jack/code/ld5",
  "secrets": [
    {
      "key": "OPENAI_API_KEY",
      "value": "sk-proj-7fXgKeaZgzty...",
      "group": "API Keys",
      "created_at": "2026-02-14T10:00:00Z",
      "updated_at": "2026-02-14T10:00:00Z"
    }
  ]
}
```

### Project detection

- Auto-detect from cwd (looks for package.json, go.mod, Cargo.toml, etc.)
- Config mapping stores project path ↔ project name
- Optional `.veil` marker file in the project directory
- CLI defaults to auto-detect from cwd, with `-p` flag as fallback

### Secret metadata

- Key, value, group, created_at, updated_at
- No description, notes, or tags in v1

### Groups

- Smart detection from key prefix: `OPENAI_` → "API Keys", `DATABASE_` → "Database", `AWS_` → "AWS", `STRIPE_` → "Payments", etc.
- User can override group assignment
- Groups are optional — defaults to "General" if prefix isn't recognized

---

## TUI

### Pages (3)

**1. Home (dashboard)**
- Veil wordmark/name at top
- Quick action buttons: Add Secret, Import .env, Sync
- Last 3 secrets added (muted, masked): `OPENAI_API_KEY  sk-proj-7f******`
- Sync status: "synced 2m ago"
- Project list with secret counts — select one to enter Project page

**2. Project (secret table)**
- Tab bar across top for switching between projects
- Table with columns: group label, key name, masked value
- Grouped by section (API Keys, Database, etc.) with header rows
- Arrow up/down to navigate rows, selected row highlighted
- `/` to search/filter (built into table)
- Keybinds trigger overlays for actions

**3. Settings**
- Gist connection status
- Machine list (paired devices)
- Shell hook toggle/setup
- Key storage location
- Export default format preference
- GitHub account info

### Overlays (8)

1. **Add secret** — text inputs for key name (with prefix autocomplete) and value, group auto-detected
2. **Edit secret** — same as add but pre-filled with existing values
3. **Delete confirmation** — confirm before removing a secret
4. **Import** — file picker to select `.env` file from filesystem, OR paste area for bulk `.env` content. Shows confirmation list before saving. Handles duplicate detection (overwrite or skip)
5. **Export** — pick format (.env or JSON) and destination path
6. **Reveal warning** — "exposing secret — press again to confirm" before showing plaintext value
7. **Search/filter** — `/` activates filter mode on the table
8. **Init wizard** — first run only (see Init Flow below)

### Key display

- All values masked by default: `sk-proj-7fXg********`
- Navigate to a row, keybind to reveal, warning overlay first
- Copy to clipboard without revealing (no auto-clear)

### Prefix autocomplete

- Built-in common prefixes: `OPENAI_`, `ANTHROPIC_`, `STRIPE_`, `SUPABASE_`, `AWS_`, `DATABASE_`, `GITHUB_`, `NEXT_PUBLIC_`, etc.
- Learns from user's existing keys across all projects
- Progressive: suggests prefix first (`OPENAI_`), then known full names (`OPENAI_API_KEY`, `OPENAI_ORG_ID`)

### Adding secrets — TUI

- **Single:** trigger add overlay, type key name (with autocomplete), paste value
- **Bulk paste:** import overlay, paste a whole `.env` block, parses `KEY=VALUE` lines, confirmation list
- **File import:** import overlay, file picker or type path to `.env` file, parse and confirm

### Bubbles component mapping

| Screen element | Bubbles component |
|---|---|
| Project tabs | Lip Gloss styled text (custom) |
| Secret table | table |
| Key name input | textinput (with autocomplete) |
| Value input | textinput |
| Bulk paste area | textarea |
| File picker (import) | filepicker |
| Home project list | list |
| Keybind bar | help |
| Scrollable content | viewport |
| Long secret lists | paginator |
| Sync/encrypt indicator | spinner |
| Keybind management | key |
| Toast notifications | Custom component |
| Reveal mask toggle | Custom component |
| QR code | Custom (skip2/go-qrcode) |
| Tab bar | Custom (Lip Gloss) |

---

## CLI Commands

All commands auto-detect project from cwd. Use `-p <project>` to override.

| Command | Description |
|---|---|
| `veil` | Opens TUI (no args) |
| `veil init` | First-time setup wizard |
| `veil set KEY VALUE` | Add or update a single secret |
| `veil get KEY` | Retrieve a single secret value |
| `veil import FILE` | Batch import from `.env` file. Supports `cat .env \| veil import -` for stdin |
| `veil export PROJECT` | Output secrets as `.env` or JSON. Flags: `--format env\|json` |
| `veil run -- COMMAND` | Inject secrets as env vars into subprocess. Plaintext never touches disk |
| `veil sync` | Push/pull encrypted secrets to/from gist |
| `veil list` | Show all projects with secret counts |
| `veil ls PROJECT` | Show keys in a project (masked values) |
| `veil rm KEY` | Delete a secret (with confirmation) |
| `veil link` | Connect to GitHub gist (or create one) |

---

## Lifecycle

### 1. Init (first run)

TUI wizard flow:

1. Run `veil init`
2. Terminal shows QR code + manual URL + device code for GitHub OAuth
3. User scans QR or clicks link, auths on any device (phone, browser)
4. CLI polls for confirmation, receives token
5. Stores token in system credential store
6. Generates age keypair
7. User chooses key storage: OS keychain or local file with permissions
8. If existing gist found → adds machine's public key to recipients
9. If no gist → creates new private gist
10. Prompts to import existing `.env` files — user can type paths or browse to them from any directory
11. Done — Veil is populated and synced

### 2. Add secrets

Everything goes into Veil directly via TUI or CLI. Stored as age-encrypted JSON per project in `~/.veil/store/`. Plaintext never touches disk outside of process memory.

### 3. Use secrets

Two modes:

- **`veil run -- npm run dev`** — injects secrets as env vars into the subprocess. Nothing written to disk. This is the primary workflow.
- **`veil export ld5`** — explicit escape hatch when a file is needed. Shows a warning.

### 4. Sync

- Push/pull encrypted blobs to a private GitHub gist
- One gist with all projects as separate files (`ld5.json.age`, `porter.json.age`, etc.)
- All ciphertext — gist never contains plaintext
- Last write wins on conflicts
- Works offline — caches last synced state, queues changes, syncs when back online
- Sync status visible on home screen ("synced 2m ago")

### 5. Multi-machine

- No pairing command needed
- `veil init` on a second machine → OAuth with same GitHub account → Veil finds existing gist
- Generates new age keypair on new machine
- Adds public key to `recipients.txt` in the gist
- Auto re-encrypts all secrets for all recipients on next sync
- GitHub identity IS your Veil identity — if you can OAuth, you're in

---

## Shell Integration

Auto-suggest on `cd` into a linked project directory. Prints subtle one-liner:

```
🔑 Veil: 12 secrets available for ld5
```

### Supported shells

- Bash
- Zsh
- Fish
- PowerShell

Setup managed via Settings page in TUI or during init.

---

## Error Handling

- **Offline:** Cache last synced state, work offline, sync when back
- **Bad token:** Toast notification in TUI, prompt to re-auth
- **Corrupt store:** Toast notification, suggest `veil sync` to pull fresh from gist
- **Display:** Toast/notification style in TUI (non-blocking)

---

## Distribution

| Channel | Install command |
|---|---|
| Homebrew (macOS/Linux) | `brew install jackhorton/tap/veil` |
| Scoop (Windows) | `scoop bucket add veil https://github.com/jackhorton/scoop-veil && scoop install veil` |
| go install | `go install github.com/jackhorton/veil@latest` |
| curl (Unix) | `curl -sSfL https://raw.githubusercontent.com/jackhorton/veil/main/install.sh \| sh` |
| irm (Windows) | `irm https://raw.githubusercontent.com/jackhorton/veil/main/install.ps1 \| iex` |

All powered by goreleaser via GitHub Actions on tag push. Cross-compiles for macOS (arm64 + amd64), Linux (arm64 + amd64), Windows (amd64).

---

## Public Presence

- GitHub README only (no landing page or docs site for v1)
- Hero visual: screenshot of TUI
- Standard README sections: what it is, install, quick start, CLI reference

---

## v1 Scope

**Everything ships in v1:**

- TUI: 3 pages + 8 overlays
- All 12 CLI commands
- Gist sync (day one)
- Shell hooks (bash/zsh/fish/powershell)
- Prefix autocomplete (built-in + learned)
- Smart group detection
- Import from `.env` (TUI + CLI)
- Export `.env` + JSON
- `veil run` (env var injection)
- OAuth device flow + QR code
- Multi-machine via GitHub identity
- Auto-detect project from cwd
- `/` search/filter in project view

**Post-launch:**

- Landing page / docs site
- Team sharing (multi-user)
- Secret rotation reminders
- VHS demo gif
- npx wrapper

---

## What Veil Is Not

- Not a file manager
- Not a `.env` editor
- Not a vault server
- Not team/org secrets management (solo dev first)
- Not a SaaS product — no accounts, no subscriptions, no corpo bullshit
