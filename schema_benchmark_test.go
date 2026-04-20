package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestSchemaBenchmark_CrossVersion is a comprehensive benchmark comparing
// v1.9.0 (pre-denormalization), v2.1.0 (denorm + full JSON state),
// v2.2.0 (denorm + compact board encoding), and v2.3.0 (compact board +
// compressed analysis) across import speed, search latency, disk storage,
// position loading, and export.
//
// Run with:
//
//	go test -v -run TestSchemaBenchmark_CrossVersion -timeout 30m
func TestSchemaBenchmark_CrossVersion(t *testing.T) {
	t.Skip("skipping cross-version benchmark: takes 10+ min, run manually when needed")
	if testing.Short() {
		t.Skip("skipping cross-version benchmark in short mode")
	}
	tournoisDir := filepath.Join("testdata", "tournois")
	if _, err := os.Stat(tournoisDir); os.IsNotExist(err) {
		t.Skip("testdata/tournois/ not found — skipping cross-version benchmark")
	}

	// Collect fixture files
	var files []string
	var totalInputBytes int64
	filepath.Walk(tournoisDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".xg" || ext == ".sgf" || ext == ".mat" || ext == ".bgf" {
			files = append(files, path)
			totalInputBytes += info.Size()
		}
		return nil
	})
	t.Logf("Fixture: %d files, %.1f MB input", len(files), float64(totalInputBytes)/(1024*1024))

	tmpDir := t.TempDir()

	// ─────────────────────────────────────────────────────────────────────
	// Phase 1: Import into v2.3.0 (current schema) and measure time
	// ─────────────────────────────────────────────────────────────────────
	dbPath230 := filepath.Join(tmpDir, "v230.db")
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 1: Import into v2.3.0 (compact board + compressed analysis)")
	t.Log("══════════════════════════════════════════════════════════")

	importStart := time.Now()
	db230 := NewDatabase()
	if err := db230.SetupDatabase(dbPath230); err != nil {
		t.Fatalf("setup v2.3.0: %v", err)
	}
	importCount := 0
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		var importErr error
		switch ext {
		case ".xg":
			_, importErr = db230.ImportXGMatch(f)
		case ".sgf", ".mat":
			_, importErr = db230.ImportGnuBGMatch(f)
		case ".bgf":
			_, importErr = db230.ImportBGFMatch(f)
		}
		if importErr == nil {
			importCount++
		}
	}
	importDuration := time.Since(importStart)

	var posCount230, analysisCount, matchCount, moveCount int
	db230.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount230)
	db230.db.QueryRow(`SELECT COUNT(*) FROM analysis`).Scan(&analysisCount)
	db230.db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matchCount)
	db230.db.QueryRow(`SELECT COUNT(*) FROM move`).Scan(&moveCount)

	t.Logf("  Import: %d files → %d matches, %d positions, %d analyses, %d moves",
		importCount, matchCount, posCount230, analysisCount, moveCount)
	t.Logf("  Import time: %v (%.0f positions/sec)", importDuration,
		float64(posCount230)/importDuration.Seconds())

	// Close and measure disk size
	db230.db.Exec(`VACUUM`)
	db230.db.Close()
	size230 := fileSize(t, dbPath230)
	t.Logf("  Disk size (vacuumed): %s", humanBytes(size230))

	// ─────────────────────────────────────────────────────────────────────
	// Phase 2: Create v2.2.0 clone (decompress analysis data → raw JSON)
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 2: Create v2.2.0 clone (raw JSON analysis)")
	t.Log("══════════════════════════════════════════════════════════")

	dbPath220 := filepath.Join(tmpDir, "v220.db")
	copyFile(t, dbPath230, dbPath220)

	sqlDB220, err := sql.Open("sqlite", dbPath220)
	if err != nil {
		t.Fatalf("open v2.2.0 clone: %v", err)
	}
	sqlDB220.Exec(`PRAGMA journal_mode = WAL`)
	sqlDB220.Exec(`PRAGMA synchronous = NORMAL`)
	sqlDB220.Exec(`PRAGMA cache_size = -65536`)

	// Decompress all analysis data back to raw JSON
	decompressStart := time.Now()
	analysisRows220, err := sqlDB220.Query(`SELECT id, data FROM analysis WHERE data IS NOT NULL`)
	if err != nil {
		t.Fatalf("query v2.2.0 analysis: %v", err)
	}
	updateStmt220, _ := sqlDB220.Prepare(`UPDATE analysis SET data = ? WHERE id = ?`)
	decompCount := 0
	for analysisRows220.Next() {
		var id int64
		var data []byte
		analysisRows220.Scan(&id, &data)
		if len(data) > 0 && data[0] != '{' {
			decompressed, err := decompressAnalysisData(data)
			if err == nil {
				updateStmt220.Exec(string(decompressed), id)
				decompCount++
			}
		}
	}
	analysisRows220.Close()
	updateStmt220.Close()
	sqlDB220.Exec(`UPDATE metadata SET value='2.2.0' WHERE key='database_version'`)
	sqlDB220.Exec(`VACUUM`)
	sqlDB220.Close()
	decompressDuration := time.Since(decompressStart)

	size220 := fileSize(t, dbPath220)
	t.Logf("  Decompressed %d analysis rows: %v", decompCount, decompressDuration)
	t.Logf("  Disk size (vacuumed): %s", humanBytes(size220))

	// ─────────────────────────────────────────────────────────────────────
	// Phase 3: Create v2.1.0 clone (revert compact state → full JSON)
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 3: Create v2.1.0 clone (full JSON state)")
	t.Log("══════════════════════════════════════════════════════════")

	dbPath210 := filepath.Join(tmpDir, "v210.db")
	copyFile(t, dbPath220, dbPath210)

	sqlDB210, err := sql.Open("sqlite", dbPath210)
	if err != nil {
		t.Fatalf("open v2.1.0 clone: %v", err)
	}
	sqlDB210.Exec(`PRAGMA journal_mode = WAL`)
	sqlDB210.Exec(`PRAGMA synchronous = NORMAL`)
	sqlDB210.Exec(`PRAGMA cache_size = -65536`)

	// Revert compact states to full JSON
	revertStart := time.Now()
	rows210, err := sqlDB210.Query(`
		SELECT id, state, decision_type, player_on_roll, dice_1, dice_2,
		       cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver
		FROM position`)
	if err != nil {
		t.Fatalf("query v2.1.0 positions: %v", err)
	}

	type posRow struct {
		id    int64
		state string
		pos   Position
	}
	var posRows []posRow
	for rows210.Next() {
		var r posRow
		var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
		if err := rows210.Scan(&r.id, &r.state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if isCompactState(r.state) {
			r.pos = reconstructPosition(r.id, r.state,
				int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
				int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
				int(hj.Int64), int(hb.Int64))
		} else {
			json.Unmarshal([]byte(r.state), &r.pos)
			r.pos.ID = r.id
		}
		posRows = append(posRows, r)
	}
	rows210.Close()

	updateStmt210, _ := sqlDB210.Prepare(`UPDATE position SET state = ? WHERE id = ?`)
	for _, r := range posRows {
		fullJSON, _ := json.Marshal(r.pos)
		updateStmt210.Exec(string(fullJSON), r.id)
	}
	updateStmt210.Close()
	sqlDB210.Exec(`UPDATE metadata SET value='2.1.0' WHERE key='database_version'`)
	sqlDB210.Exec(`VACUUM`)
	sqlDB210.Close()
	revertDuration := time.Since(revertStart)

	size210 := fileSize(t, dbPath210)
	t.Logf("  Revert to full JSON: %v (%d positions)", revertDuration, len(posRows))
	t.Logf("  Disk size (vacuumed): %s", humanBytes(size210))

	// ─────────────────────────────────────────────────────────────────────
	// Phase 4: Create v1.9.0 clone (drop denorm columns + indexes)
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 4: Create v1.9.0 clone (no denorm columns, no indexes)")
	t.Log("══════════════════════════════════════════════════════════")

	dbPath190 := filepath.Join(tmpDir, "v190.db")
	copyFile(t, dbPath210, dbPath190) // start from v2.1.0 (full JSON)

	sqlDB190, err := sql.Open("sqlite", dbPath190)
	if err != nil {
		t.Fatalf("open v1.9.0 clone: %v", err)
	}
	sqlDB190.Exec(`PRAGMA journal_mode = WAL`)

	// Drop all v2.0.0+ indexes
	v2Indexes := []string{
		"idx_position_zobrist", "idx_position_decision_type", "idx_position_dice",
		"idx_position_cube_value", "idx_position_cube_owner", "idx_position_score",
		"idx_position_pip_diff", "idx_position_pip_1", "idx_position_pip_2",
		"idx_position_off_1", "idx_position_off_2", "idx_position_no_contact",
		"idx_position_back_checkers_1", "idx_position_back_checkers_2",
		"idx_position_occupancy_1", "idx_position_occupancy_2",
		"idx_position_point_mask_1", "idx_position_point_mask_2",
		"idx_analysis_cube_error", "idx_analysis_best_move_equity_error",
		"idx_analysis_player1_win_rate",
	}
	for _, idx := range v2Indexes {
		sqlDB190.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", idx))
	}

	// Recreate position table without denorm columns
	sqlDB190.Exec(`CREATE TABLE position_old AS SELECT id, state FROM position`)
	sqlDB190.Exec(`DROP TABLE position`)
	sqlDB190.Exec(`CREATE TABLE position (id INTEGER PRIMARY KEY AUTOINCREMENT, state TEXT)`)
	sqlDB190.Exec(`INSERT INTO position (id, state) SELECT id, state FROM position_old`)
	sqlDB190.Exec(`DROP TABLE position_old`)

	// Similarly strip analysis denorm columns
	sqlDB190.Exec(`CREATE TABLE analysis_old AS SELECT id, position_id, data FROM analysis`)
	sqlDB190.Exec(`DROP TABLE analysis`)
	sqlDB190.Exec(`CREATE TABLE analysis (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		position_id INTEGER,
		data JSON,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
	)`)
	sqlDB190.Exec(`INSERT INTO analysis (id, position_id, data) SELECT id, position_id, data FROM analysis_old`)
	sqlDB190.Exec(`DROP TABLE analysis_old`)

	sqlDB190.Exec(`UPDATE metadata SET value='1.9.0' WHERE key='database_version'`)
	sqlDB190.Exec(`VACUUM`)
	sqlDB190.Close()

	size190 := fileSize(t, dbPath190)
	t.Logf("  Disk size (vacuumed): %s", humanBytes(size190))

	// ─────────────────────────────────────────────────────────────────────
	// Phase 5: Size breakdown per table
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 5: Per-table size breakdown")
	t.Log("══════════════════════════════════════════════════════════")

	for _, variant := range []struct {
		name, path string
	}{
		{"v1.9.0", dbPath190},
		{"v2.1.0", dbPath210},
		{"v2.2.0", dbPath220},
		{"v2.3.0", dbPath230},
	} {
		db, _ := sql.Open("sqlite", variant.path)
		t.Logf("\n  --- %s ---", variant.name)

		// Measure state column total size
		var stateTotal int64
		db.QueryRow(`SELECT COALESCE(SUM(LENGTH(state)),0) FROM position`).Scan(&stateTotal)
		t.Logf("  position.state total bytes: %s", humanBytes(stateTotal))

		// Avg state size
		var avgState float64
		db.QueryRow(`SELECT COALESCE(AVG(LENGTH(state)),0) FROM position`).Scan(&avgState)
		t.Logf("  position.state avg bytes:   %.0f", avgState)

		// Analysis data total size
		var analysisTotal int64
		db.QueryRow(`SELECT COALESCE(SUM(LENGTH(data)),0) FROM analysis`).Scan(&analysisTotal)
		t.Logf("  analysis.data total bytes:  %s", humanBytes(analysisTotal))

		// Page count and page size
		var pageCount, pageSize int64
		db.QueryRow(`PRAGMA page_count`).Scan(&pageCount)
		db.QueryRow(`PRAGMA page_size`).Scan(&pageSize)
		t.Logf("  pages: %d × %d = %s", pageCount, pageSize, humanBytes(pageCount*pageSize))

		// Per-table page usage (dbstat virtual table if available)
		rows, err := db.Query(`
			SELECT name, SUM(pgsize) as total_size
			FROM dbstat
			GROUP BY name
			ORDER BY total_size DESC
			LIMIT 15
		`)
		if err == nil {
			t.Log("  Per-table/index page usage:")
			for rows.Next() {
				var name string
				var sz int64
				rows.Scan(&name, &sz)
				t.Logf("    %-40s %s", name, humanBytes(sz))
			}
			rows.Close()
		}

		db.Close()
	}

	// ─────────────────────────────────────────────────────────────────────
	// Phase 6: Search benchmarks (v2.3.0 indexed search)
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 6: Search latency — v2.3.0 (indexed + compact + compressed)")
	t.Log("══════════════════════════════════════════════════════════")

	// Re-open v2.3.0 as Database object
	db230obj := NewDatabase()
	if err := db230obj.OpenDatabase(dbPath230); err != nil {
		t.Fatalf("open v2.3.0 DB: %v", err)
	}

	type searchCase struct {
		name   string
		filter Position
		args   searchArgs
	}

	toSearchFilters := func(sc searchCase) SearchFilters {
		return SearchFilters{
			Filter:                        sc.filter,
			IncludeCube:                   sc.args.includeCube,
			IncludeScore:                  sc.args.includeScore,
			PipCountFilter:                sc.args.pipCountFilter,
			WinRateFilter:                 sc.args.winRateFilter,
			GammonRateFilter:              sc.args.gammonRateFilter,
			BackgammonRateFilter:          sc.args.backgammonRateFilter,
			Player2WinRateFilter:          sc.args.player2WinRateFilter,
			Player2GammonRateFilter:       sc.args.player2GammonRateFilter,
			Player2BackgammonRateFilter:   sc.args.player2BackgammonRateFilter,
			Player1CheckerOffFilter:       sc.args.player1CheckerOffFilter,
			Player2CheckerOffFilter:       sc.args.player2CheckerOffFilter,
			Player1BackCheckerFilter:      sc.args.player1BackCheckerFilter,
			Player2BackCheckerFilter:      sc.args.player2BackCheckerFilter,
			Player1CheckerInZoneFilter:    sc.args.player1CheckerInZoneFilter,
			Player2CheckerInZoneFilter:    sc.args.player2CheckerInZoneFilter,
			SearchText:                    sc.args.searchText,
			Player1AbsolutePipCountFilter: sc.args.player1AbsolutePipCountFilter,
			EquityFilter:                  sc.args.equityFilter,
			DecisionTypeFilter:            sc.args.decisionTypeFilter,
			DiceRollFilter:                sc.args.diceRollFilter,
			MovePatternFilter:             sc.args.movePatternFilter,
			DateFilter:                    sc.args.dateFilter,
			Player1OutfieldBlotFilter:     sc.args.player1OutfieldBlotFilter,
			Player2OutfieldBlotFilter:     sc.args.player2OutfieldBlotFilter,
			Player1JanBlotFilter:          sc.args.player1JanBlotFilter,
			Player2JanBlotFilter:          sc.args.player2JanBlotFilter,
			NoContactFilter:               sc.args.noContactFilter,
			MirrorFilter:                  sc.args.mirrorFilter,
			MoveErrorFilter:               sc.args.moveErrorFilter,
			MatchIDsFilter:                sc.args.matchIDsFilter,
			TournamentIDsFilter:           sc.args.tournamentIDsFilter,
			RestrictToPositionIDs:         sc.args.restrictToPositionIDs,
		}
	}

	cases := []searchCase{
		{
			name: "DecisionCube",
			filter: func() Position {
				f := Position{}
				f.DecisionType = CubeAction
				f.PlayerOnRoll = 0
				return f
			}(),
			args: searchArgs{decisionTypeFilter: true},
		},
		{
			name:   "MoveError>0.1",
			filter: Position{},
			args:   searchArgs{moveErrorFilter: "E>100"},
		},
		{
			name:   "PipDiff[-2,2]",
			filter: Position{},
			args:   searchArgs{pipCountFilter: "p-2,2"},
		},
		{
			name:   "Win>55%+Gammon>20%",
			filter: Position{},
			args:   searchArgs{winRateFilter: "w>0.55", gammonRateFilter: "g>0.2"},
		},
		{
			name: "Score6-4",
			filter: func() Position {
				f := Position{}
				f.Score = [2]int{6, 4}
				return f
			}(),
			args: searchArgs{includeScore: true},
		},
		{
			name: "Dice65+Checker",
			filter: func() Position {
				f := Position{}
				f.Dice = [2]int{6, 5}
				f.PlayerOnRoll = 0
				f.DecisionType = CheckerAction
				return f
			}(),
			args: searchArgs{diceRollFilter: true},
		},
		{
			name: "20ptAnchor",
			filter: func() Position {
				f := Position{}
				f.Board.Points[20] = Point{Checkers: 2, Color: Black}
				return f
			}(),
		},
		{
			name: "5-Prime(4-8)",
			filter: func() Position {
				f := Position{}
				for _, pt := range []int{4, 5, 6, 7, 8} {
					f.Board.Points[pt] = Point{Checkers: 2, Color: Black}
				}
				return f
			}(),
		},
		{
			name:   "NoContact",
			filter: Position{},
			args:   searchArgs{noContactFilter: true},
		},
		{
			name:   "Mirror",
			filter: Position{},
			args:   searchArgs{mirrorFilter: true},
		},
	}

	const searchIters = 20 // repeat each search N times for stable timing

	for _, sc := range cases {
		// v2.2.0
		start := time.Now()
		var resultCount int
		for j := 0; j < searchIters; j++ {
			results, _ := db230obj.LoadPositionsByFilters(toSearchFilters(sc))
			resultCount = len(results)
		}
		elapsed := time.Since(start)
		avgMs := float64(elapsed.Microseconds()) / float64(searchIters) / 1000.0
		t.Logf("  %-25s %6d results   %8.2f ms/search", sc.name, resultCount, avgMs)
	}

	// v1.9.0-style: full table scan + JSON unmarshal + Go-side filter
	// We simulate the two simplest filters to show the order-of-magnitude difference.
	t.Log("\n  --- v1.9.0-style (full scan + JSON unmarshal) ---")

	sqlDB190s, _ := sql.Open("sqlite", dbPath190)
	sqlDB190s.Exec(`PRAGMA cache_size = -65536`)
	sqlDB190s.Exec(`PRAGMA mmap_size = 268435456`)

	v190Search := func(label string, match func(*Position) bool) {
		start := time.Now()
		var count int
		for j := 0; j < searchIters; j++ {
			rows, _ := sqlDB190s.Query(`SELECT id, state FROM position`)
			count = 0
			for rows.Next() {
				var id int64
				var state string
				rows.Scan(&id, &state)
				var pos Position
				json.Unmarshal([]byte(state), &pos)
				pos.ID = id
				if match(&pos) {
					count++
				}
			}
			rows.Close()
		}
		elapsed := time.Since(start)
		avgMs := float64(elapsed.Microseconds()) / float64(searchIters) / 1000.0
		t.Logf("  %-25s %6d results   %8.2f ms/search", label, count, avgMs)
	}

	v190Search("DecisionCube(v1.9)", func(p *Position) bool {
		return p.DecisionType == CubeAction && p.PlayerOnRoll == 0
	})
	v190Search("PipDiff[-2,2](v1.9)", func(p *Position) bool {
		pip1, pip2 := p.ComputePipCounts()
		diff := pip1 - pip2
		return diff >= -2 && diff <= 2
	})
	v190Search("NoContact(v1.9)", func(p *Position) bool {
		return p.MatchesNoContact()
	})

	sqlDB190s.Close()

	// v2.2.0-style: same indexed search but with raw JSON analysis
	t.Log("\n  --- v2.2.0 (indexed search, raw JSON analysis) ---")
	db220obj := NewDatabase()
	if err := db220obj.OpenDatabase(dbPath220); err != nil {
		t.Fatalf("open v2.2.0 DB: %v", err)
	}

	for _, sc := range cases {
		start := time.Now()
		var resultCount int
		for j := 0; j < searchIters; j++ {
			results, _ := db220obj.LoadPositionsByFilters(toSearchFilters(sc))
			resultCount = len(results)
		}
		elapsed := time.Since(start)
		avgMs := float64(elapsed.Microseconds()) / float64(searchIters) / 1000.0
		t.Logf("  %-25s %6d results   %8.2f ms/search", sc.name+"(v2.2)", resultCount, avgMs)
	}
	db220obj.db.Close()

	// ─────────────────────────────────────────────────────────────────────
	// Phase 6b: Analysis load latency (v2.3.0 compressed vs v2.2.0 raw)
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 6b: Analysis load latency (compressed vs raw)")
	t.Log("══════════════════════════════════════════════════════════")

	const analysisIters = 5
	// Sample 1000 analysis IDs spread across the range
	var sampleAnalysisIDs []int64
	func() {
		db, _ := sql.Open("sqlite", dbPath230)
		defer db.Close()
		rows, _ := db.Query(`SELECT position_id FROM analysis ORDER BY id LIMIT 1000`)
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			sampleAnalysisIDs = append(sampleAnalysisIDs, id)
		}
		rows.Close()
	}()

	// v2.3.0: read + decompress
	func() {
		db, _ := sql.Open("sqlite", dbPath230)
		db.Exec(`PRAGMA cache_size = -65536`)
		db.Exec(`PRAGMA mmap_size = 268435456`)
		defer db.Close()

		start := time.Now()
		for i := 0; i < analysisIters; i++ {
			for _, pid := range sampleAnalysisIDs {
				var data []byte
				db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, pid).Scan(&data)
				if len(data) > 0 {
					decodeAnalysisFromStorage(data)
				}
			}
		}
		elapsed := time.Since(start)
		t.Logf("  v2.3.0 (compressed): %d loads × %d iters in %v (%.1f µs/load)",
			len(sampleAnalysisIDs), analysisIters, elapsed,
			float64(elapsed.Microseconds())/float64(len(sampleAnalysisIDs)*analysisIters))
	}()

	// v2.2.0: read raw JSON
	func() {
		db, _ := sql.Open("sqlite", dbPath220)
		db.Exec(`PRAGMA cache_size = -65536`)
		db.Exec(`PRAGMA mmap_size = 268435456`)
		defer db.Close()

		start := time.Now()
		for i := 0; i < analysisIters; i++ {
			for _, pid := range sampleAnalysisIDs {
				var data string
				db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, pid).Scan(&data)
				if data != "" {
					var ana PositionAnalysis
					json.Unmarshal([]byte(data), &ana)
					_ = ana
				}
			}
		}
		elapsed := time.Since(start)
		t.Logf("  v2.2.0 (raw JSON):   %d loads × %d iters in %v (%.1f µs/load)",
			len(sampleAnalysisIDs), analysisIters, elapsed,
			float64(elapsed.Microseconds())/float64(len(sampleAnalysisIDs)*analysisIters))
	}()

	// ─────────────────────────────────────────────────────────────────────
	// Phase 7: LoadAllPositions benchmark
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 7: LoadAllPositions (full scan + deserialize)")
	t.Log("══════════════════════════════════════════════════════════")

	const loadIters = 5
	start := time.Now()
	for i := 0; i < loadIters; i++ {
		positions, err := db230obj.LoadAllPositions()
		if err != nil {
			t.Fatalf("LoadAllPositions: %v", err)
		}
		if i == 0 {
			t.Logf("  v2.3.0 LoadAllPositions: %d positions", len(positions))
		}
	}
	elapsed230Load := time.Since(start)
	t.Logf("  v2.3.0 avg: %.2f ms", float64(elapsed230Load.Microseconds())/float64(loadIters)/1000.0)

	// Compare with v1.9.0-style full JSON scan (simulate by reading state+unmarshal)
	sqlDB190b, _ := sql.Open("sqlite", dbPath190)
	sqlDB190b.Exec(`PRAGMA cache_size = -65536`)
	sqlDB190b.Exec(`PRAGMA mmap_size = 268435456`)

	start = time.Now()
	for i := 0; i < loadIters; i++ {
		rows, _ := sqlDB190b.Query(`SELECT id, state FROM position`)
		count := 0
		for rows.Next() {
			var id int64
			var state string
			rows.Scan(&id, &state)
			var pos Position
			json.Unmarshal([]byte(state), &pos)
			pos.ID = id
			count++
		}
		rows.Close()
		if i == 0 {
			t.Logf("  v1.9.0 LoadAllPositions (full JSON): %d positions", count)
		}
	}
	elapsed190Load := time.Since(start)
	t.Logf("  v1.9.0 avg: %.2f ms", float64(elapsed190Load.Microseconds())/float64(loadIters)/1000.0)
	sqlDB190b.Close()

	// v2.1.0-style (full JSON in state column, but has denorm columns)
	sqlDB210b, _ := sql.Open("sqlite", dbPath210)
	sqlDB210b.Exec(`PRAGMA cache_size = -65536`)
	sqlDB210b.Exec(`PRAGMA mmap_size = 268435456`)

	start = time.Now()
	for i := 0; i < loadIters; i++ {
		rows, _ := sqlDB210b.Query(`SELECT id, state FROM position`)
		count := 0
		for rows.Next() {
			var id int64
			var state string
			rows.Scan(&id, &state)
			var pos Position
			json.Unmarshal([]byte(state), &pos)
			pos.ID = id
			count++
		}
		rows.Close()
		if i == 0 {
			t.Logf("  v2.1.0 LoadAllPositions (full JSON): %d positions", count)
		}
	}
	elapsed210Load := time.Since(start)
	t.Logf("  v2.1.0 avg: %.2f ms", float64(elapsed210Load.Microseconds())/float64(loadIters)/1000.0)
	sqlDB210b.Close()

	// ─────────────────────────────────────────────────────────────────────
	// Phase 8: Single position load benchmark
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 8: Single LoadPosition by ID")
	t.Log("══════════════════════════════════════════════════════════")

	const singleIters = 1000
	// Pick position IDs spread across the range
	testIDs := []int{1, posCount230 / 4, posCount230 / 2, posCount230 * 3 / 4, posCount230}

	start = time.Now()
	for i := 0; i < singleIters; i++ {
		for _, id := range testIDs {
			db230obj.LoadPosition(id)
		}
	}
	elapsed230Single := time.Since(start)
	t.Logf("  v2.3.0: %d loads in %v (%.1f µs/load)",
		singleIters*len(testIDs), elapsed230Single,
		float64(elapsed230Single.Microseconds())/float64(singleIters*len(testIDs)))

	// ─────────────────────────────────────────────────────────────────────
	// Phase 9: Import speed (cold import measurement)
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 9: Cold import speed (subset)")
	t.Log("══════════════════════════════════════════════════════════")

	// Use first 10 .xg files for a repeatable import speed test
	var xgFiles []string
	for _, f := range files {
		if strings.ToLower(filepath.Ext(f)) == ".xg" {
			xgFiles = append(xgFiles, f)
		}
		if len(xgFiles) >= 10 {
			break
		}
	}

	const importIters = 3
	var importTimes []time.Duration
	for i := 0; i < importIters; i++ {
		dbPathTmp := filepath.Join(tmpDir, fmt.Sprintf("import_bench_%d.db", i))
		dbTmp := NewDatabase()
		dbTmp.SetupDatabase(dbPathTmp)

		start := time.Now()
		for _, f := range xgFiles {
			dbTmp.ImportXGMatch(f)
		}
		importTimes = append(importTimes, time.Since(start))

		var cnt int
		dbTmp.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&cnt)
		dbTmp.db.Close()
		os.Remove(dbPathTmp)

		if i == 0 {
			t.Logf("  %d XG files → %d positions", len(xgFiles), cnt)
		}
	}
	var totalImport time.Duration
	for _, d := range importTimes {
		totalImport += d
	}
	t.Logf("  Avg import time: %v (over %d runs)", totalImport/time.Duration(importIters), importIters)

	// ─────────────────────────────────────────────────────────────────────
	// Phase 10: Export benchmark
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("Phase 10: ExportDatabase speed")
	t.Log("══════════════════════════════════════════════════════════")

	exportPath := filepath.Join(tmpDir, "export_test.db")
	allPositions, _ := db230obj.LoadAllPositions()
	start = time.Now()
	db230obj.ExportDatabase(ExportOptions{
		ExportPath:      exportPath,
		Positions:       allPositions,
		Metadata:        map[string]string{},
		IncludeAnalysis: true,
		IncludeComments: true,
		IncludePlayedMoves: true,
		IncludeMatches:  true,
	})
	exportDuration := time.Since(start)
	exportSize := fileSize(t, exportPath)
	t.Logf("  Export: %v, disk size: %s", exportDuration, humanBytes(exportSize))
	os.Remove(exportPath)

	// ─────────────────────────────────────────────────────────────────────
	// Final summary
	// ─────────────────────────────────────────────────────────────────────
	t.Log("\n══════════════════════════════════════════════════════════")
	t.Log("SUMMARY")
	t.Log("══════════════════════════════════════════════════════════")
	t.Logf("Dataset: %d files, %d positions, %d analyses, %d moves",
		len(files), posCount230, analysisCount, moveCount)
	t.Log("")
	t.Logf("Disk size:")
	t.Logf("  v1.9.0 (full JSON, no indexes):          %s", humanBytes(size190))
	t.Logf("  v2.1.0 (full JSON + denorm cols):         %s", humanBytes(size210))
	t.Logf("  v2.2.0 (compact board + raw analysis):    %s", humanBytes(size220))
	t.Logf("  v2.3.0 (compact board + compressed anal): %s", humanBytes(size230))
	if size190 > 0 {
		t.Logf("  v2.1.0 vs v1.9.0: %+.1f%%", (float64(size210)-float64(size190))/float64(size190)*100)
		t.Logf("  v2.2.0 vs v1.9.0: %+.1f%%", (float64(size220)-float64(size190))/float64(size190)*100)
		t.Logf("  v2.3.0 vs v1.9.0: %+.1f%%", (float64(size230)-float64(size190))/float64(size190)*100)
		t.Logf("  v2.3.0 vs v2.2.0: %+.1f%%", (float64(size230)-float64(size220))/float64(size220)*100)
	}
	t.Log("")
	t.Logf("LoadAllPositions (avg):")
	t.Logf("  v1.9.0: %.2f ms", float64(elapsed190Load.Microseconds())/float64(loadIters)/1000.0)
	t.Logf("  v2.1.0: %.2f ms", float64(elapsed210Load.Microseconds())/float64(loadIters)/1000.0)
	t.Logf("  v2.3.0: %.2f ms", float64(elapsed230Load.Microseconds())/float64(loadIters)/1000.0)

	db230obj.db.Close()
}

// searchArgs bundles the many filter parameters for LoadPositionsByFilters
// so test cases are readable.
type searchArgs struct {
	includeCube                   bool
	includeScore                  bool
	pipCountFilter                string
	winRateFilter                 string
	gammonRateFilter              string
	backgammonRateFilter          string
	player2WinRateFilter          string
	player2GammonRateFilter       string
	player2BackgammonRateFilter   string
	player1CheckerOffFilter       string
	player2CheckerOffFilter       string
	player1BackCheckerFilter      string
	player2BackCheckerFilter      string
	player1CheckerInZoneFilter    string
	player2CheckerInZoneFilter    string
	searchText                    string
	player1AbsolutePipCountFilter string
	equityFilter                  string
	decisionTypeFilter            bool
	diceRollFilter                bool
	movePatternFilter             string
	dateFilter                    string
	player1OutfieldBlotFilter     string
	player2OutfieldBlotFilter     string
	player1JanBlotFilter          string
	player2JanBlotFilter          string
	noContactFilter               bool
	mirrorFilter                  bool
	moveErrorFilter               string
	matchIDsFilter                string
	tournamentIDsFilter           string
	restrictToPositionIDs         string
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMG"[exp])
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("copy %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}
