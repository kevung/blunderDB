# Archived Design Documents

These documents describe completed features and bug fixes.
They are kept for historical reference but do not reflect current code.

- `ANALYSIS_IMPLEMENTATION.md` — XG analysis import (completed)
- `ANALYSIS_STORAGE_OPTIMIZATION.md` — Zlib compression of analysis data (completed)
- `DATABASE_OPTIMIZATION_PLAN.md` — v2.0.0 schema/search optimization plan (shipped; executed via `tasks/00`–`06`, line anchors refer to the pre-split monolithic `db.go`)
- `DISPLAY_FIX_SUMMARY.md` — Player-on-roll encoding + mirroring fix (completed)
- `IMPROVEMENT_PLAN.md` — 2025-04 code-quality audit (shipped: `db.go` split, CI test/lint jobs, `App.svelte` decomposition, Svelte 5 runes migration)
- `MATCH_IMPORT_ARCHITECTURE.md` — Match import schema v1.4.0 (superseded by v2.x)
- `MATCH_MODE_DISPLAY_IMPLEMENTATION.md` — Match position display (completed)
- `PLAYED_MOVE_INDICATOR.md` — Gold star indicator feature (shipped)
- `POSITION_TRACKING_IMPLEMENTATION.md` — Position dedup v1.4.0 (superseded by v2.x)
- `XG_PLAYER_ENCODING_FIX.md` — XG player encoding bug fix (completed)
- `ui-reactivity-audit.md`, `ui-reactivity-benchmark.md`, `ui-reactivity-scenarios.md` — Svelte 5 migration reactivity audit (completed; the store/effect rule it produced lives in `CLAUDE.md`)
