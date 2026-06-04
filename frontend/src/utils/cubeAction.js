// Shared helpers for interpreting cube action strings (as stored in
// move.cube_action / analysis.playedCubeActions: "Double", "Double/Take",
// "Double/Pass", "No Double", "Take", "Pass", …). Kept in one place so the
// analysis panel and the board agree on what each action means.

// normalizeCubeAction maps a cube action to the canonical analysis-row parts it
// highlights: combinations of 'nodouble' | 'double' | 'take' | 'pass'. A
// standalone Take/Pass (the opponent's response) maps onto the combined
// Double/Take or Double/Pass row.
export function normalizeCubeAction(action) {
    const s = (action || '').toLowerCase().replace(/\s+/g, '');
    if (s === 'double/take' || s === 'doubletake') return ['double', 'take'];
    if (s === 'double/pass' || s === 'doublepass') return ['double', 'pass'];
    if (s === 'nodouble' || s === 'nodoubleorredouble' || s === 'noredouble') return ['nodouble'];
    if (s === 'redouble') return ['double'];
    if (s === 'take') return ['double', 'take'];
    if (s === 'pass' || s === 'drop') return ['double', 'pass'];
    return [s]; // "double", etc.
}

// isResponseCubeAction is true for a pure take/pass response (the cube was
// offered to this player), and false for a doubling decision — including the
// doubler's combined actions "Double/Take"/"Double/Pass" and "No Double".
// Mirrors Go engine.IsResponseCubeAction. Used to render the offered cube in
// the middle of the board for take/pass decisions.
export function isResponseCubeAction(action) {
    const s = (action || '').toLowerCase().replace(/\s+/g, '');
    if (s.includes('double')) return false; // double, double/take, double/pass, nodouble, redouble
    return s === 'dt' || s === 'dp' || s.includes('take') || s.includes('pass') || s.includes('drop');
}
