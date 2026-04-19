# 23 — Clean up stale design documents

**Goal:** Archive or remove 9 completed/stale design documents cluttering the repo root to reduce confusion.

**Depends on:** Nothing.

**Impact:** Low — housekeeping, but prevents contributors from reading outdated docs.

## Context

### Stale documents in repo root

| File | Lines | Reason |
|------|------:|--------|
| `ANALYSIS_IMPLEMENTATION.md` | 264 | Completed implementation note |
| `ANALYSIS_STORAGE_OPTIMIZATION.md` | 91 | Completed — compression implemented |
| `DISPLAY_FIX_SUMMARY.md` | 167 | Completed bug fix note |
| `MATCH_IMPORT_ARCHITECTURE.md` | 416 | References schema v1.4.0 (current is v2.3.0) |
| `MATCH_MODE_DISPLAY_IMPLEMENTATION.md` | 262 | Completed implementation note |
| `PLAYED_MOVE_INDICATOR.md` | 96 | Shipped feature |
| `POSITION_TRACKING_IMPLEMENTATION.md` | 323 | References schema v1.4.0, completed |
| `XG_PLAYER_ENCODING_FIX.md` | 99 | Completed bug fix note |
| `TODO.md` | 1 | Empty (whitespace only) |

### Documents to KEEP in root

| File | Reason |
|------|--------|
| `README.md` | Project README |
| `CLAUDE.md` | AI assistant guidance |
| `CLI_USAGE.md` | CLI reference (active) |
| `CLI_README.md` | CLI quick-start (active) |
| `DATABASE_OPTIMIZATION_PLAN.md` | Active plan, referenced by tasks/ |
| `IMPROVEMENT_PLAN.md` | Active roadmap |
| `LICENSE` | License file |

## Files touched

- **Move:** 8 stale `.md` files → `doc/archive/` (or delete)
- **Delete:** `TODO.md` (empty)
- **New:** `doc/archive/README.md` (index of archived docs)

## Tasks

### 1. Create archive directory

- [ ] Create `doc/archive/` directory
- [ ] Create `doc/archive/README.md`:
  ```markdown
  # Archived Design Documents

  These documents describe completed features and bug fixes.
  They are kept for historical reference but do not reflect current code.

  - `ANALYSIS_IMPLEMENTATION.md` — XG analysis import (completed)
  - `ANALYSIS_STORAGE_OPTIMIZATION.md` — Zlib compression of analysis data (completed)
  - `DISPLAY_FIX_SUMMARY.md` — Player-on-roll encoding + mirroring fix (completed)
  - `MATCH_IMPORT_ARCHITECTURE.md` — Match import schema v1.4.0 (superseded by v2.x)
  - `MATCH_MODE_DISPLAY_IMPLEMENTATION.md` — Match position display (completed)
  - `PLAYED_MOVE_INDICATOR.md` — Gold star indicator feature (shipped)
  - `POSITION_TRACKING_IMPLEMENTATION.md` — Position dedup v1.4.0 (superseded by v2.x)
  - `XG_PLAYER_ENCODING_FIX.md` — XG player encoding bug fix (completed)
  ```

### 2. Move stale documents

- [ ] `git mv ANALYSIS_IMPLEMENTATION.md doc/archive/`
- [ ] `git mv ANALYSIS_STORAGE_OPTIMIZATION.md doc/archive/`
- [ ] `git mv DISPLAY_FIX_SUMMARY.md doc/archive/`
- [ ] `git mv MATCH_IMPORT_ARCHITECTURE.md doc/archive/`
- [ ] `git mv MATCH_MODE_DISPLAY_IMPLEMENTATION.md doc/archive/`
- [ ] `git mv PLAYED_MOVE_INDICATOR.md doc/archive/`
- [ ] `git mv POSITION_TRACKING_IMPLEMENTATION.md doc/archive/`
- [ ] `git mv XG_PLAYER_ENCODING_FIX.md doc/archive/`

### 3. Delete empty TODO.md

- [ ] `git rm TODO.md`

### 4. Update references

- [ ] Check `CLAUDE.md` for references to moved files — update paths or note they're archived
- [ ] Check any other files that might link to the moved documents

### 5. Verify

- [ ] `git status` shows clean moves/deletions
- [ ] No broken links in remaining documentation
- [ ] Repo root is cleaner: only active docs remain

## Acceptance criteria

- [ ] 8 stale documents moved to `doc/archive/`
- [ ] Empty `TODO.md` deleted
- [ ] `doc/archive/README.md` indexes the archived files
- [ ] No broken references in active documents
- [ ] Repo root has ≤ 7 `.md` files

## Rollback

`git revert` — just file moves, easily reversed.
