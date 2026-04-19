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

- [ ] Project name + one-line description:
  > **blunderDB** — A backgammon blunder analysis tool. Desktop app (GUI + CLI) for importing, storing, searching, and studying positions from eXtreme Gammon, GnuBG, and BGBlitz.
- [ ] Add badges: CI status, license, documentation link

### 2. Add screenshot

- [ ] Take a screenshot of the GUI with a position loaded and analysis visible
- [ ] Save as `doc/source/_static/screenshot.png` (or similar)
- [ ] Embed in README: `![blunderDB screenshot](doc/source/_static/screenshot.png)`

### 3. Write Features section

- [ ] Key features:
  - Import matches from XG (`.xg`, `.xgp`), GnuBG (`.sgf`), BGBlitz (`.bgf`), Jellyfish (`.mat`)
  - Search positions by checker structure, pip count, equity, error threshold, dice, etc.
  - Spaced repetition (FSRS/Anki-style) flash cards for position review
  - EPC (Effective Pip Count) calculator with embedded bearoff database
  - Collections and tournaments for organizing positions
  - Match Equity Tables display (Kazaross, Rockwell, etc.)
  - CLI for scripted analysis workflows
  - Cross-platform: Linux, macOS, Windows

### 4. Write Installation section

- [ ] Download section: link to GitHub Releases
- [ ] Build from source:
  ```bash
  # Prerequisites: Go 1.23+, Node.js 23+, Wails v2
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  git clone https://github.com/kevung/blunderDB.git
  cd blunderDB
  wails build
  # Binary at build/bin/blunderdb
  ```
- [ ] Linux note: requires webkit2gtk-4.1 (`wails build -tags webkit2_41`)

### 5. Write Usage section

- [ ] GUI quick start: launch binary, open/create DB, import a match file
- [ ] CLI quick start: link to `CLI_USAGE.md`
- [ ] Example commands:
  ```bash
  ./blunderdb                                              # GUI mode
  ./blunderdb import --db my.db --type match --file game.xg  # Import
  ./blunderdb search --db my.db --error ">0.05"              # Search
  ./blunderdb list --db my.db --type stats                   # Stats
  ```

### 6. Write Tech Stack section

- [ ] Brief mention:
  - **Backend:** Go + pure-Go SQLite (`modernc.org/sqlite`)
  - **Frontend:** Svelte 5 + Vite + two.js (board rendering)
  - **Framework:** Wails v2 (Go ↔ WebView bridge)
  - **Docs:** Sphinx (French + English)

### 7. Write Documentation / Contributing / License sections

- [ ] Documentation: link to GitHub Pages
- [ ] Contributing: brief note (open issues, PRs welcome, run `go test ./...` before submitting)
- [ ] License: reference LICENSE file

### 8. Keep it concise

- [ ] Target 80-120 lines total
- [ ] Don't duplicate CLI_USAGE.md — link to it

## Acceptance criteria

- [ ] README has: description, screenshot, features, install, usage, tech stack, license
- [ ] ≤ 150 lines
- [ ] All links work (badges, docs, CLI_USAGE.md)
- [ ] Someone unfamiliar with the project can understand what it does and how to use it

## Rollback

`git revert` — single file edit.
