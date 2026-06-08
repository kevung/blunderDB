// Dev-only generator (NOT a vitest spec — bare .js, excluded by the test glob).
// Builds the parse corpus inputs, runs the CURRENT frontend parsePosition to
// capture its output as the golden spec, and writes testdata/parse_corpus.json.
// Run via the throwaway _parseGoldenWriter.test.js harness, then delete both.
import { parsePosition } from '../services/importService.js';

// ── input builders ────────────────────────────────────────────────
const pad = (s, n) => s.padEnd(n);

function checkerMoveBlock(n, depth, move, eq, err, pWin, pG, pB, oWin, oG, oB) {
    let line = `    ${n}. ${pad(depth, 11)} ${pad(move, 28)} eq:${eq}`;
    if (err) line += ` (${err})`;
    line += `\n      Player:   ${pWin}% (G:${pG}% B:${pB}%)`;
    line += `\n      Opponent: ${oWin}% (G:${oG}% B:${oB}%)`;
    return line;
}

const CASES = [];
const add = (name, input) => CASES.push({ name, input });

// 1. bare XGID line (money Jacoby+Beaver, issue #13 beaver position)
add('bare_xgid', 'XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10');

// 2. real XG doubling-cube text, French, Jacoby+Beaver (issue #13)
add(
    'xg_cube_fr_beaver',
    `XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10

X:Joueur 1   O:Joueur 2
Le score est X:0 O:0. Partie illimitée, Jacoby Beaver
 +13-14-15-16-17-18------19-20-21-22-23-24-+
 | X           O    |   | O              X |
 +12-11-10--9--8--7-------6--5--4--3--2--1-+
Course  X: 162  O: 164 X-O: 0-0
Videau: 1
X: lance ou double

Analysé avec XG Roller++
Chance de gain du joueur: 54.09% (G:20.06% B:0.47%)
Chance de gain de l'adversaire: 45.91% (G:12.14% B:0.58%)

Equités sans videau: Pas de double=+0.160, Double=+0.324

Equités avec videau
       Pas de double: +0.224
       Double/Beaver: -0.021 (-0.245)
       Double/Passe:  +1.000 (+0.776)

Meilleur action du videau: Pas de double / Beaver
Pourcentage de passes incorrectes pour rendre la décision de double correcte: 24.0%

eXtreme Gammon Version: 2.19.211.pre-release`
);

// 3. real XG doubling-cube text, French, too-good / redouble (issue #13)
add(
    'xg_cube_fr_toogood_redouble',
    `XGID=-BaB--C---a-cD-Aabbe-A-A-A:1:-1:-1:00:2:0:0:9:10

X:James MacNaughtan   O:Nicolas Harmand
Le score est X:0 O:2 match en 9 pt(s)
 +13-14-15-16-17-18------19-20-21-22-23-24-+
 | X     X          |   | O        O  X  O |
 +12-11-10--9--8--7-------6--5--4--3--2--1-+
Course  X: 146  O: 162 X-O: 0-2/9
Videau: 2, X a le videau
X: lance ou double

Analysé avec XG Roller++
Chance de gain du joueur: 72,00% (G:31,15% B:2,80%)
Chance de gain de l'adversaire: 28,00% (G:6,88% B:0,24%)

Equités sans videau: Pas de double=+0,751, Double=+1,693

Equités avec videau
       Pas de redouble: +1,068
       Redouble/Prend:  +1,573 (+0,505)
       Redouble/Passe:  +1,000 (-0,068)

Meilleur action du videau: Trop bon pour redoubler / Passe
Pourcentage de prises incorrectes pour rendre la décision de double correcte: 11.8%

eXtreme Gammon Version: 2.10, TEM: Kazaross XG2`
);

// 4. synthetic XG doubling-cube text, English (turn=1 → no swap)
add(
    'xg_cube_en',
    `XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10

X:Player   O:Opponent
Score is X:0 O:0 7 point match
Cube: 1
X on roll, cube decision?

Analyzed in Rollout
Player Winning Chances: 54.00% (G:15.00% B:1.00%)
Opponent Winning Chances: 46.00% (G:12.00% B:0.50%)

Cubeless Equities: No Double=+0.200, Double=+0.350

Cubeful Equities:
       No double:     +0.250 (-0.050)
       Double/Take:   +0.300
       Double/Pass:   +1.000 (+0.700)

Best Cube action: Double / Take
Percentage of wrong pass needed to make the double decision right: 20.0%
Percentage of wrong take needed to make the double decision right: 5.0%

eXtreme Gammon Version: 2.19`
);

// 5. synthetic XG checker-move text, English, player 1 on roll (turn=1)
add(
    'xg_checker_en',
    `XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:7:10

X:Player   O:Opponent
Score is X:0 O:0 7 point match
Cube: 1
X to play 31

${checkerMoveBlock(1, '4-ply', '24/23 13/10', '+0.123', null, '54.32', '12.34', '0.56', '45.68', '10.12', '0.34')}

${checkerMoveBlock(2, '4-ply', '24/20', '+0.100', '-0.023', '53.00', '11.00', '0.50', '47.00', '11.00', '0.40')}

eXtreme Gammon Version: 2.19`
);

// 6. synthetic XG checker-move text, English, player 2 on roll (turn=-1 → swap)
add(
    'xg_checker_en_p2',
    `XGID=-b----E-C---eE---c-e----B-:0:0:-1:31:0:0:0:7:10

X:Player   O:Opponent
Score is X:0 O:0 7 point match
Cube: 1
O to play 31

${checkerMoveBlock(1, '4-ply', '24/23 13/10', '+0.123', null, '54.32', '12.34', '0.56', '45.68', '10.12', '0.34')}

eXtreme Gammon Version: 2.19`
);

// 7. blunderDB internal Doubling Cube export format
add(
    'internal_doubling',
    `XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10

Position:
Board: {"points":[],"bearoff":[0,0]}
Cube: {"owner":-1,"value":0}
Dice: 0, 0
Score: -1, -1
Player on roll: 0
Decision type: 1

Analysis:
Doubling Cube Analysis:
Analysis Depth: "4-ply"
Player Win Chances: 54%
Player Gammon Chances: 15%
Player Backgammon Chances: 1%
Opponent Win Chances: 46%
Opponent Gammon Chances: 12%
Opponent Backgammon Chances: 0.5%
Cubeless No Double Equity: 0.2
Cubeless Double Equity: 0.35
Cubeful No Double Equity: 0.25
Cubeful No Double Error: 0.05
Cubeful Double Take Equity: 0.3
Cubeful Double Take Error: 0
Cubeful Double Pass Equity: 1
Cubeful Double Pass Error: 0.7
Best Cube Action: Double / Take
Wrong Pass Percentage: 20%
Wrong Take Percentage: 5%
eXtreme Gammon Version: 2.19`
);

// 8. blunderDB internal Checker Move export format
add(
    'internal_checker',
    `XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:7:10

Position:
Board: {"points":[],"bearoff":[0,0]}
Cube: {"owner":-1,"value":0}
Dice: 3, 1
Score: -1, -1
Player on roll: 0
Decision type: 0

Analysis:
Checker Move Analysis:
Move 1: 24/23 13/10
Analysis Depth: "4-ply"
Equity: 0.123
Equity Error: 0
Player Win Chance: 54.32%
Player Gammon Chance: 12.34%
Player Backgammon Chance: 0.56%
Opponent Win Chance: 45.68%
Opponent Gammon Chance: 10.12%
Opponent Backgammon Chance: 0.34%
eXtreme Gammon Version: 2.19`
);

// 9. cube text with a trailing comment block (FR) — exercises comment extraction
add(
    'xg_cube_en_comment',
    `XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10

X:Player   O:Opponent
Score is X:0 O:0 7 point match
Cube: 1
X on roll, cube decision?

Analyzed in Rollout
Player Winning Chances: 54.00% (G:15.00% B:1.00%)
Opponent Winning Chances: 46.00% (G:12.00% B:0.50%)

Cubeless Equities: No Double=+0.200, Double=+0.350

Cubeful Equities:
       No double:     +0.250 (-0.050)
       Double/Take:   +0.300
       Double/Pass:   +1.000 (+0.700)

Best Cube action: Double / Take
Percentage of wrong pass needed to make the double decision right: 20.0%
Percentage of wrong take needed to make the double decision right: 5.0%

This is a note about the position, with 1,234 as a number.

eXtreme Gammon Version: 2.19`
);

// ── normalization: the canonical comparable shape (both Go and JS map to this)
export function normalize(input) {
    const { positionData, parsedAnalysis } = parsePosition(input);
    const a = parsedAnalysis;
    return {
        position: positionData,
        analysis: {
            analysisType: a.analysisType || '',
            xgid: a.xgid || '',
            analysisEngineVersion: a.analysisEngineVersion || '',
            comment: a.comment || '',
            doublingCubeAnalysis: a.analysisType === 'DoublingCube' ? a.doublingCubeAnalysis : null,
            checkerAnalysis: a.analysisType === 'CheckerMove' ? a.checkerAnalysis : []
        }
    };
}

export function buildCorpus() {
    return {
        _comment:
            'Shared parse contract corpus. Asserted by BOTH the Go parser (pkg/blunderdb/parser, parse_contract_test.go) AND the GUI parsePosition (frontend/src/__tests__/parseContract.test.js). Generated from the CURRENT frontend parsePosition (the spec). Conventions: position is the domain.Position JSON shape (snake_case); analysis.doublingCubeAnalysis is the camelCase object or null; analysis.checkerAnalysis is the bare moves array.',
        cases: CASES.map((c) => ({ name: c.name, input: c.input, expected: normalize(c.input) }))
    };
}
