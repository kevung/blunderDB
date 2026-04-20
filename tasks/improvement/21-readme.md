# 21 — Improve README.md

**Goal:** Replace the 7-line stub README with a proper project README that helps new users and contributors understand blunderDB.

**Depends on:** Nothing.

**Impact:** High — the README is the first thing visitors see on GitHub.

## Context

### Current README.md (7 lines total)
- 1 documentation badge (shields.io)
- 1 section linking to GitHub Pages docs

### Missing
- Project description
- Screenshot / demo GIF
- Feature list
- Download / install instructions
- Build instructions (Wails)
- Quick-start usage (GUI + CLI)
- Supported file formats
- Tech stack
- License reference
- CI badge

## Files touched

- **Edit:** `README.md`

## Tasks

### 1. Write project header

- [x] Project name + one-line description:
  > **blunderDB** — A backgammon blunder analysis tool. Desktop app (GUI + CLI) for importing, storing, searching, and studying positions from eXtreme Gammon, GnuBG, and BGBlitz.
- [x] Add badges: CI status, license, documentation link

### 2. Add screenshot

- [ ] Take a screenshot of the GUI with a position loaded and analysis visible (TODO: manual step)
- [ ] Save as `doc/source/_static/screenshot.png` (or similar)
- [x] Embed in README: placeholder comment added, ready to uncomment when screenshot exists

### 3. Write Features section

- [x] Key features:
  - Import matches from XG (`.xg`, `.xgp`), GnuBG (`.sgf`), BGBlitz (`.bgf`), Jellyfish (`.mat`)
  - Search positions by checker structure, pip count, equity, error threshold, dice, etc.
  - Spaced repetition (FSRS/Anki-style) flash cards for position review
  - EPC (Effective Pip Count) calculator with embedded bearoff database
  - Collections and tournaments for organizing positions
  - Match Equity Tables display (Kazaross, Rockwell, etc.)
  - CLI for scripted analysis workflows
  - Cross-platform: Linux, macOS, Windows

### 4. Write Installation section

- [x] Download section: link to GitHub Releases
- [x] Build from source:
  ```bash
  # Prerequisites: Go 1.23+, Node.js 23+, Wails v2
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  git clone https://github.com/kevung/blunderDB.git
  cd blunderDB
  wails build
  # Binary at build/bin/blunderdb
  ```
- [x] Linux note: requires webkit2gtk-4.1 (`wails build -tags webkit2_41`)

### 5. Write Usage section

- [x] GUI quick start: launch binary, open/create DB, import a match file
- [x] CLI quick start: link to `CLI_USAGE.md`
- [x] Example commands:
  ```bash
  ./blunderdb                                              # GUI mode
  ./blunderdb import --db my.db --type match --file game.xg  # Import
  ./blunderdb search --db my.db --error ">0.05"              # Search
  ./blunderdb list --db my.db --type stats                   # Stats
  ```

### 6. Write Tech Stack section

- [x] Brief mention:
  - **Backend:** Go + pure-Go SQLite (`modernc.org/sqlite`)
  - **Frontend:** Svelte 5 + Vite + two.js (board rendering)
  - **Framework:** Wails v2 (Go ↔ WebView bridge)
  - **Docs:** Sphinx (French + English)

### 7. Write Documentation / Contributing / License sections

- [x] Documentation: link to GitHub Pages
- [x] Contributing: brief note (open issues, PRs welcome, run `go test ./...` before submitting)
- [x] License: reference LICENSE file

### 8. Keep it concise

- [x] Target 80-120 lines total (95 lines)
- [x] Don't duplicate CLI_USAGE.md — link to it

## Acceptance criteria

- [x] README has: description, screenshot (placeholder), features, install, usage, tech stack, license
- [x] ≤ 150 lines (95 lines)
- [x] All links work (badges, docs, CLI_USAGE.md)
- [x] Someone unfamiliar with the project can understand what it does and how to use it

## Rollback

`git revert` — single file edit.
