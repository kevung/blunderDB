// Package parser turns the human-readable position text that blunderDB accepts
// on the clipboard / via file import into a domain.Position (+ analysis). It is
// the single backend home for what used to live only in the frontend
// (frontend/src/services/importService.js `parsePosition`); the GUI now calls it
// over Wails, and the server/CLI reuse it too, so the two implementations can no
// longer drift (see testdata/parse_corpus.json and the dual contract tests).
//
// It handles: a bare XGID line; the XG human-readable export with either a
// doubling-cube or a checker-move analysis block (French / English / Japanese /
// German); blunderDB's own internal export format; and a trailing comment.
//
// The logic is a port of the JS parser. Go's regexp uses Perl-style
// leftmost-first submatching, so the greedy/lazy/fixed-width captures behave
// like the original JS regexes. One quirk of the original was dropped: it
// rewrote every comma of the text into a dot to read "0,491" in a German or
// French export, comments included; the numeric captures now accept either
// separator and pf() reads both, so a comment keeps its commas.
package parser

import (
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Result is the output of ParsePosition. Comment is carried separately because
// domain.PositionAnalysis has no comment field (the GUI stores it via a separate
// path), mirroring the JS `{positionData, parsedAnalysis}` + parsedAnalysis.comment.
type Result struct {
	Position domain.Position          `json:"position"`
	Analysis *domain.PositionAnalysis `json:"analysis"`
	Comment  string                   `json:"comment"`
}

// ErrNoXGID / ErrEmpty mirror the JS throws so callers can map them to 4xx.
var (
	reMu = newRegexCache()
	// leadNum mirrors what JS parseFloat() consumes, exponent notation included
	// (parseFloat("5.55e-17") is 5.55e-17, and JS prints small floats that way).
	leadNum = regexp.MustCompile(`^[+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?`)
)

// ParsePosition is the inverse of the clipboard text builders. It never panics;
// it returns an error for empty input, a missing XGID (as the JS does), and for
// an analysis block it can see but cannot read (ErrUnrecognisedAnalysis).
func ParsePosition(text string) (Result, error) {
	if strings.TrimSpace(text) == "" {
		return Result{}, errEmpty
	}

	// Normalize accented characters to NFC (precomposed). macOS's pasteboard
	// hands us NFD (decomposed, e.g. "e"+combining-acute) for text copied from
	// some sources (observed with XG run under Sikarugir/Wine on Mac); every
	// accent-bearing literal below (e.g. "éq:", "Equités") is written in NFC,
	// so decomposed input silently fails every match and the analysis comes
	// back empty. Linux/Windows clipboards don't exhibit this, which is why it
	// doesn't reproduce there.
	text = norm.NFC.String(text)

	// Normalize newlines + trim, exactly like the JS (importService.js:822-823).
	raw := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"))
	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}

	// Language / format detection runs on the pre-comma content (JS:825-829).
	isFrench := strings.Contains(raw, "Joueur") || strings.Contains(raw, "Adversaire") || strings.Contains(raw, "Videau")
	isJapanese := strings.Contains(raw, "プレーヤー") || strings.Contains(raw, "対戦相手") || strings.Contains(raw, "キューブ")
	isInternalChecker := strings.Contains(raw, "Analysis:\nChecker Move Analysis:")
	isInternalDoubling := strings.Contains(raw, "Analysis:\nDoubling Cube Analysis:")
	isGerman := strings.Contains(raw, "Spieler") || strings.Contains(raw, "Gegner") || strings.Contains(raw, "Dopplerwürfel")

	// The JS parser rewrote every comma into a dot here; the numeric captures
	// below accept both separators instead, so comments keep their commas.
	content := raw

	// XGID line (JS:833-838).
	var xgid string
	for _, l := range lines {
		if after, ok := strings.CutPrefix(l, "XGID="); ok {
			xgid = after
			break
		}
	}
	if xgid == "" {
		return Result{}, errNoXGID
	}

	pos := decodePosition(xgid, lines, isFrench, isJapanese, isGerman, isInternalChecker)

	analysis := &domain.PositionAnalysis{XGID: xgid}
	engineVersion, engineName := parseEngineVersion(content)
	analysis.AnalysisEngineVersion = engineVersion

	switch {
	case isInternalDoubling:
		parseInternalDoubling(content, engineName, analysis)
	case isInternalChecker:
		parseInternalChecker(content, engineName, analysis)
	case reMu.get(`(?m)^ {4}(\d+)\.`).MatchString(content):
		parseCheckerText(content, engineName, pos.PlayerOnRoll, isFrench, isJapanese, isGerman, analysis)
	case hasCubefulBlock(content, isFrench, isJapanese, isGerman):
		parseCubeText(content, engineName, pos.PlayerOnRoll, isFrench, isJapanese, isGerman, analysis)
	}

	if analysis.AnalysisType == "" && looksLikeAnalysis(content) {
		return Result{}, ErrUnrecognisedAnalysis
	}
	if analysis.CheckerAnalysis != nil && !checkerChancesRead(analysis.CheckerAnalysis.Moves) && strings.Contains(content, "(G:") {
		// The move lines were read (XG writes "eq:" in most of its languages)
		// but not one win-chance line was: the Player/Opponent words are in a
		// language the parser does not know. Saving 0 % everywhere would be
		// silently wrong.
		return Result{}, ErrUnrecognisedAnalysis
	}

	comment := extractComment(content, analysis.AnalysisType == "DoublingCube")

	return Result{Position: pos, Analysis: analysis, Comment: comment}, nil
}

// looksLikeAnalysis reports whether the text carries what every XG analysis
// export carries in every language: a numbered move list indented by four
// spaces, or a win-chance breakdown "(G:…% B:…%)". A bare XGID or a board
// diagram has neither and stays a plain position.
func looksLikeAnalysis(content string) bool {
	return strings.Contains(content, "(G:") || reMu.get(`(?m)^ {4}\d+\.\s`).MatchString(content)
}

// checkerChancesRead reports whether at least one move carries a win chance.
// No real position gives every candidate 0 % to both sides.
func checkerChancesRead(moves []domain.CheckerMove) bool {
	for _, m := range moves {
		if m.PlayerWinChance != 0 || m.OpponentWinChance != 0 {
			return true
		}
	}
	return false
}

// ── Position metadata ─────────────────────────────────────────────
// Reuse domain.DecodeXGID for the board/cube/dice/jacoby/beaver decode (shared
// with the server path), then apply the two GUI-specific patches the JS does and
// DecodeXGID does not: decision_type from the analysis TEXT, and the Crawford
// away-score remap (1→0 when field 7 is 0 in match play).
func decodePosition(xgid string, lines []string, isFrench, isJapanese, isGerman, isInternalChecker bool) domain.Position {
	pos, err := domain.DecodeXGID(xgid)
	if err != nil {
		// DecodeXGID is strict (board must be 26 chars); the JS parser is lax. On
		// failure fall back to a best-effort empty board so callers still get the
		// other fields rather than a hard error (the GUI never feeds a bad board).
		pos = domain.Position{}
		for i := range pos.Board.Points {
			pos.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
		}
	}

	fields := strings.Split(xgid, ":")
	field7, has7 := atoi(fields, 7)
	matchLen, hasM := atoi(fields, 8)

	// Crawford away-score remap (JS:870-872): when field 7 == 0, a 1-away score
	// becomes 0. Money games already collapsed to [-1,-1] by DecodeXGID.
	if has7 && field7 == 0 && hasM && matchLen > 0 {
		for i := range pos.Score {
			if pos.Score[i] == 1 {
				pos.Score[i] = 0
			}
		}
	}

	// decision_type from text (JS:893-894), overriding DecodeXGID's dice-based value.
	keyword := "to play"
	if isFrench {
		keyword = "jouer"
	} else if isGerman {
		keyword = "spielen"
	}
	hasDecisionLine := false
	for _, l := range lines {
		if strings.Contains(l, keyword) {
			hasDecisionLine = true
			break
		}
	}
	if hasDecisionLine || isInternalChecker {
		pos.DecisionType = domain.CheckerAction
	} else {
		pos.DecisionType = domain.CubeAction
	}
	return pos
}

// ── Engine version (JS:912-920) ───────────────────────────────────
func parseEngineVersion(content string) (version, engineName string) {
	m := reMu.get(`(?m)eXtreme Gammon Version: (.+?)(?:\. MET: (.+))?$`).FindStringSubmatch(content)
	if m == nil {
		return "", ""
	}
	version = "eXtreme Gammon Version: " + m[1]
	if m[2] != "" {
		version += ", MET: " + m[2]
	}
	return version, "XG"
}

// ── Internal blunderDB formats ────────────────────────────────────
func parseInternalDoubling(content, engineName string, a *domain.PositionAnalysis) {
	re := reMu.get(`Doubling Cube Analysis:\nAnalysis Depth: "(.+)"\nPlayer Win Chances: ([-.\d]+)%\nPlayer Gammon Chances: ([-.\d]+)%\nPlayer Backgammon Chances: ([-.\d]+)%\nOpponent Win Chances: ([-.\d]+)%\nOpponent Gammon Chances: ([-.\d]+)%\nOpponent Backgammon Chances: ([-.\d]+)%\nCubeless No Double Equity: ([-.\d]+)\nCubeless Double Equity: ([-.\d]+)\nCubeful No Double Equity: ([-.\d]+)\nCubeful No Double Error: ([-.\d]+)\nCubeful Double Take Equity: ([-.\d]+)\nCubeful Double Take Error: ([-.\d]+)\nCubeful Double Pass Equity: ([-.\d]+)\nCubeful Double Pass Error: ([-.\d]+)\nBest Cube Action: (.+)\nWrong Pass Percentage: ([-.\d]+)%\nWrong Take Percentage: ([-.\d]+)%`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		return
	}
	a.AnalysisType = "DoublingCube"
	a.DoublingCubeAnalysis = &domain.DoublingCubeAnalysis{
		AnalysisDepth:             strings.TrimSpace(m[1]),
		AnalysisEngine:            engineName,
		PlayerWinChances:          pf(m[2]),
		PlayerGammonChances:       pf(m[3]),
		PlayerBackgammonChances:   pf(m[4]),
		OpponentWinChances:        pf(m[5]),
		OpponentGammonChances:     pf(m[6]),
		OpponentBackgammonChances: pf(m[7]),
		CubelessNoDoubleEquity:    pf(m[8]),
		CubelessDoubleEquity:      pf(m[9]),
		CubefulNoDoubleEquity:     pf(m[10]),
		CubefulNoDoubleError:      pf(m[11]),
		CubefulDoubleTakeEquity:   pf(m[12]),
		CubefulDoubleTakeError:    pf(m[13]),
		CubefulDoublePassEquity:   pf(m[14]),
		CubefulDoublePassError:    pf(m[15]),
		BestCubeAction:            strings.TrimSpace(m[16]),
		WrongPassPercentage:       pf(m[17]),
		WrongTakePercentage:       pf(m[18]),
	}
}

// internalNumber matches a number the way JavaScript prints one, since this
// block is written by the GUI's clipboardService: an optional sign, digits with
// an optional decimal point, and the exponent notation JS falls back to for very
// small magnitudes (an equity subtraction yields 5.55e-17 easily enough).
const internalNumber = `[-+]?[\d.]+(?:[eE][-+]?\d+)?`

// parseInternalChecker reads the checker block blunderDB itself writes to the
// clipboard. Two of its lines are optional, and treating them as mandatory used
// to silently drop the move they belong to:
//   - "Equity Error:" is absent for the best move — its EquityError is nil by
//     design (see ingest.sortCheckerMovesByEquity) and the JSON field is
//     omitempty, so the writer skips the line. Requiring it cost the best move
//     on every copy/paste from one database to another.
//   - the depth may be empty (`Analysis Depth: ""`) for analyses whose import
//     route left it blank.
func parseInternalChecker(content, engineName string, a *domain.PositionAnalysis) {
	re := reMu.get(`(?m)^Move (\d+): (.*)\nAnalysis Depth: "(.*)"\nEquity: (` + internalNumber +
		`)\n(?:Equity Error: (` + internalNumber + `)\n)?Player Win Chance: (` + internalNumber +
		`)%\nPlayer Gammon Chance: (` + internalNumber + `)%\nPlayer Backgammon Chance: (` + internalNumber +
		`)%\nOpponent Win Chance: (` + internalNumber + `)%\nOpponent Gammon Chance: (` + internalNumber +
		`)%\nOpponent Backgammon Chance: (` + internalNumber + `)%`)
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return
	}
	a.AnalysisType = "CheckerMove"
	moves := make([]domain.CheckerMove, 0, len(matches))
	for _, m := range matches {
		// An unmatched optional group is "", which the number pattern can never
		// produce — so an empty group 5 unambiguously means "line absent", i.e.
		// no error to record rather than an error of zero.
		var equityError *float64
		if m[5] != "" {
			equityError = ptr(pf(m[5]))
		}
		moves = append(moves, domain.CheckerMove{
			Index:                    atoiOne(m[1]),
			Move:                     strings.TrimSpace(m[2]),
			AnalysisDepth:            strings.TrimSpace(m[3]),
			AnalysisEngine:           engineName,
			Equity:                   pf(m[4]),
			EquityError:              equityError,
			PlayerWinChance:          pf(m[6]),
			PlayerGammonChance:       pf(m[7]),
			PlayerBackgammonChance:   pf(m[8]),
			OpponentWinChance:        pf(m[9]),
			OpponentGammonChance:     pf(m[10]),
			OpponentBackgammonChance: pf(m[11]),
		})
	}
	a.CheckerAnalysis = &domain.CheckerAnalysis{Moves: moves}
}

// ── XG checker-move TEXT (JS:971-1035) ────────────────────────────
func parseCheckerText(content, engineName string, playerOnRoll int, isFrench, isJapanese, isGerman bool, a *domain.PositionAnalysis) {
	eq := "eq:"
	if isFrench {
		eq = "éq:"
	}
	moveRe := reMu.get(`(?m)^ {4}(\d+)\.\s(.{11})\s(.{28})\s` + eq + `(.{5,7})\s(?:\((-?[-.,\d]{5,7})\))?`)
	playerRe := reMu.get(playerPctPattern(isFrench, isJapanese, isGerman, true))
	opponentRe := reMu.get(playerPctPattern(isFrench, isJapanese, isGerman, false))

	locs := moveRe.FindAllStringSubmatchIndex(content, -1)
	if len(locs) == 0 {
		return
	}
	a.AnalysisType = "CheckerMove"
	moves := make([]domain.CheckerMove, 0, len(locs))
	for _, loc := range locs {
		grp := func(i int) string {
			if loc[2*i] < 0 {
				return ""
			}
			return content[loc[2*i]:loc[2*i+1]]
		}
		mv := domain.CheckerMove{
			Index:          atoiOne(grp(1)),
			AnalysisDepth:  strings.TrimSpace(grp(2)),
			AnalysisEngine: engineName,
			Move:           strings.TrimSpace(grp(3)),
			Equity:         pf(grp(4)),
			EquityError:    ptr(pfOrZero(grp(5))),
		}
		remaining := content[loc[1]:]
		if pm := playerRe.FindStringSubmatch(remaining); pm != nil {
			mv.PlayerWinChance = pf(pm[1])
			mv.PlayerGammonChance = pf(pm[2])
			mv.PlayerBackgammonChance = pf(pm[3])
		}
		if om := opponentRe.FindStringSubmatch(remaining); om != nil {
			mv.OpponentWinChance = pf(om[1])
			mv.OpponentGammonChance = pf(om[2])
			mv.OpponentBackgammonChance = pf(om[3])
		}
		moves = append(moves, mv)
	}
	if playerOnRoll == 1 {
		for i := range moves {
			moves[i].PlayerWinChance, moves[i].OpponentWinChance = moves[i].OpponentWinChance, moves[i].PlayerWinChance
			moves[i].PlayerGammonChance, moves[i].OpponentGammonChance = moves[i].OpponentGammonChance, moves[i].PlayerGammonChance
			moves[i].PlayerBackgammonChance, moves[i].OpponentBackgammonChance = moves[i].OpponentBackgammonChance, moves[i].PlayerBackgammonChance
		}
	}
	a.CheckerAnalysis = &domain.CheckerAnalysis{Moves: moves}
}

func playerPctPattern(isFrench, isJapanese, isGerman, player bool) string {
	switch {
	case isFrench:
		if player {
			return `Joueur:\s*` + pctTriple
		}
		return `Adversaire:\s*` + pctTriple
	case isJapanese:
		if player {
			return `プレーヤー:\s*` + pctTriple
		}
		return `対戦相手:\s*` + pctTriple
	case isGerman:
		if player {
			return `Spieler:\s*` + pctTriple
		}
		return `Gegner:\s*` + pctTriple
	default:
		if player {
			return `Player:\s*` + pctTriple
		}
		return `Opponent:\s*` + pctTriple
	}
}

// pct is a percentage as XG prints it, with a dot or a comma for the decimal
// separator depending on the locale.
const (
	pct       = `(\d+[.,]\d+)`
	pctTriple = pct + `%.*\(G:` + pct + `% B:` + pct + `%\)`
)

// ── XG doubling-cube TEXT (JS:1036-1242) ──────────────────────────
func hasCubefulBlock(content string, isFrench, isJapanese, isGerman bool) bool {
	switch {
	case isFrench:
		return strings.Contains(content, "Equités sans videau") || strings.Contains(content, "Equités avec videau")
	case isJapanese, isGerman:
		// JS uses the EN markers for JP, and German markers for DE.
		if isGerman {
			return strings.Contains(content, "Equities ohne Dopplerwürfel") || strings.Contains(content, "Equities mit Dopplerwürfel")
		}
		return strings.Contains(content, "Cubeless Equities") || strings.Contains(content, "Cubeful Equities")
	default:
		return strings.Contains(content, "Cubeless Equities") || strings.Contains(content, "Cubeful Equities")
	}
}

func parseCubeText(content, engineName string, playerOnRoll int, isFrench, isJapanese, isGerman bool, a *domain.PositionAnalysis) {
	a.AnalysisType = "DoublingCube"
	d := &domain.DoublingCubeAnalysis{AnalysisEngine: engineName}

	pick := func(fr, jp, de, en string) string {
		switch {
		case isFrench:
			return fr
		case isJapanese:
			return jp
		case isGerman:
			return de
		default:
			return en
		}
	}
	find := func(pattern string) []string {
		return reMu.get(pattern).FindStringSubmatch(content)
	}

	if m := find(pick(`Analysé avec\s+([^\n]*)`, `Analyzed in\s+([^\n]*)`, `Analysiert in\s+([^\n]*)`, `Analyzed in\s+([^\n]*)`)); m != nil {
		d.AnalysisDepth = strings.TrimSpace(m[1])
	}
	if m := find(pick(
		`Chance de gain du joueur:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`,
		`Player Winning Chances:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`,
		`Spieler Gewinnchancen:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`,
		`Player Winning Chances:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`)); m != nil {
		d.PlayerWinChances = pf(m[1])
		d.PlayerGammonChances = pf(m[2])
		d.PlayerBackgammonChances = pf(m[3])
	}
	if m := find(pick(
		`Chance de gain de l'adversaire:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`,
		`Opponent Winning Chances:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`,
		`Gewinnchancen des Gegners:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`,
		`Opponent Winning Chances:\s+(\d+[.,]\d+)% \(G:(\d+[.,]\d+)% B:(\d+[.,]\d+)%\)`)); m != nil {
		d.OpponentWinChances = pf(m[1])
		d.OpponentGammonChances = pf(m[2])
		d.OpponentBackgammonChances = pf(m[3])
	}
	if m := find(pick(
		`Equités sans videau\s*:\s*Pas de double=([+\-]?\d+(?:[.,]\d+)?).\s*Double=([+\-]?\d+(?:[.,]\d+)?)`,
		`Cubeless Equities:\s*No Double=([+\-]?\d+(?:[.,]\d+)?).\s*Double=([+\-]?\d+(?:[.,]\d+)?).`,
		`Equities ohne Dopplerwürfel\s*:\s*Nicht Doppeln=([+\-]?\d+(?:[.,]\d+)?).\s*Doppeln=([+\-]?\d+(?:[.,]\d+)?)`,
		`Cubeless Equities:\s*No Double=([+\-]?\d+(?:[.,]\d+)?).\s*Double=([+\-]?\d+(?:[.,]\d+)?)`)); m != nil {
		d.CubelessNoDoubleEquity = pf(m[1])
		d.CubelessDoubleEquity = pf(m[2])
	}
	if m := find(pick(
		`Pas de double\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`ノーダブル\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Nicht Doppeln\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`No double\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`)); m != nil {
		d.CubefulNoDoubleEquity = pf(m[1])
		d.CubefulNoDoubleError = pfOrZero(m[2])
	}
	if m := find(pick(
		`Double/Prend:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`ダブル/テイク:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Doppeln/Annehmen:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Double/Take:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`)); m != nil {
		d.CubefulDoubleTakeEquity = pf(m[1])
		d.CubefulDoubleTakeError = pfOrZero(m[2])
	}
	if m := find(pick(
		`Double/Passe:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`ダブル/パス:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Doppeln/Ablehnen:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Double/Pass:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`)); m != nil {
		d.CubefulDoublePassEquity = pf(m[1])
		d.CubefulDoublePassError = pfOrZero(m[2])
	}
	if m := find(pick(
		`Pas de redouble\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`ノーリダブル\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Nicht Redoppeln\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`No redouble\s*:\s*([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`)); m != nil {
		d.CubefulNoDoubleEquity = pf(m[1])
		d.CubefulNoDoubleError = pfOrZero(m[2])
	}
	if m := find(pick(
		`Redouble/Prend:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`リダブル/テイク:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Redoppeln/Annehmen:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Redouble/Take:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`)); m != nil {
		d.CubefulDoubleTakeEquity = pf(m[1])
		d.CubefulDoubleTakeError = pfOrZero(m[2])
	}
	if m := find(pick(
		`Redouble/Passe:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`リダブル/パス:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Redoppeln/Ablehnen:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Redouble/Pass:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`)); m != nil {
		d.CubefulDoublePassEquity = pf(m[1])
		d.CubefulDoublePassError = pfOrZero(m[2])
	}
	if m := find(pick(
		`Double/Beaver:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`ダブル/ビーバー:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Doppeln/Beaver:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`,
		`Double/Beaver:\s+([+\-]?\d+(?:[.,]\d+)?)(?: \(([+\-]?\d+(?:[.,]\d+)?)\))?`)); m != nil {
		d.CubefulDoubleTakeEquity = pf(m[1])
		d.CubefulDoubleTakeError = pfOrZero(m[2])
	}
	if m := find(pick(`Meilleur action du videau:\s*(.*)`, `ベストキューブアクション：\s*(.*)`, `Beste Dopplerwürfel Aktion\s*(.*)`, `Best Cube action:\s*(.*)`)); m != nil {
		d.BestCubeAction = strings.TrimSpace(m[1])
	}
	if m := find(pick(
		`Pourcentage de passes incorrectes pour rendre la décision de double correcte:\s*(\d+[.,]\d+)%`,
		`ダブルを正当化するのに必要な相手がパスする確率:\s*(\d+[.,]\d+)%`,
		`Prozent von falschen Ablehnen gebraucht damit Doppelentscheidung richtig wäre.:\s*(\d+[.,]\d+)%`,
		`Percentage of wrong pass needed to make the double decision right:\s*(\d+[.,]\d+)%`)); m != nil {
		d.WrongPassPercentage = pf(m[1])
	}
	if m := find(pick(
		`Pourcentage de prises incorrectes pour rendre la décision de double correcte:\s*(\d+[.,]\d+)%`,
		`ダブルを正当化するのに必要な相手がテイクする確率:\s*(\d+[.,]\d+)%`,
		`Prozent von falschen Annehmen gebraucht damit Doppelentscheidung richtig wäre.:\s*(\d+[.,]\d+)%`,
		`Percentage of wrong take needed to make the double decision right:\s*(\d+[.,]\d+)%`)); m != nil {
		d.WrongTakePercentage = pf(m[1])
	}

	if playerOnRoll == 1 {
		d.PlayerWinChances, d.OpponentWinChances = d.OpponentWinChances, d.PlayerWinChances
		d.PlayerGammonChances, d.OpponentGammonChances = d.OpponentGammonChances, d.PlayerGammonChances
		d.PlayerBackgammonChances, d.OpponentBackgammonChances = d.OpponentBackgammonChances, d.PlayerBackgammonChances
	}
	a.DoublingCubeAnalysis = d
}

// ── Comment extraction (JS:1250-1292) ─────────────────────────────
func extractComment(content string, isDoublingCube bool) string {
	if isDoublingCube {
		m := reMu.get(`(?:Best Cube action: .+|Meilleur action du videau: .+|Percentage of wrong .+|Pourcentage de passes incorrectes .+%)\n\n([\s\S]+?)\n\neXtreme Gammon Version:`).FindStringSubmatch(content)
		if m != nil {
			return strings.TrimSpace(m[1])
		}
		return ""
	}
	lines := strings.Split(content, "\n")
	lastOpponent := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "Opponent") || strings.Contains(lines[i], "Adversaire") {
			lastOpponent = i
			break
		}
	}
	if lastOpponent == -1 {
		return ""
	}
	blank := 0
	start := -1
	for i := lastOpponent + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			blank++
		} else {
			blank = 0
		}
		if blank == 2 {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := -1
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "eXtreme Gammon Version:") {
			end = i
			break
		}
	}
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

// ── small helpers ─────────────────────────────────────────────────
func pf(s string) float64 {
	// XG writes "0,491" in its German, French and Russian locales.
	m := leadNum.FindString(strings.ReplaceAll(strings.TrimSpace(s), ",", "."))
	if m == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(m, 64)
	return f
}

// pfOrZero matches JS `match ? parseFloat(match) : 0` for optional captures.
func pfOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	return pf(s)
}

func ptr(f float64) *float64 { return &f }

func atoi(fields []string, idx int) (int, bool) {
	if idx >= len(fields) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(fields[idx]))
	if err != nil {
		return 0, false
	}
	return n, true
}

func atoiOne(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
