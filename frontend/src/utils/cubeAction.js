// Interpreting cube action strings ("Double", "Double/Take", "No Double", "Take", …) in one place,
// so the analysis panel and the board agree.

// normalizeCubeAction maps an action to the analysis-row parts it highlights ('nodouble' |
// 'double' | 'take' | 'pass'); a standalone Take/Pass maps onto the combined Double/… row.
// The separators are dropped like engine.CanonicalCubeAction does ("Double, Take" is how the
// BGF import writes it), and "too good to double" is a no-double: the position is played on.
function squash(action) {
    return (action || '').toLowerCase().replace(/[\s/,._-]+/g, '');
}

export function normalizeCubeAction(action) {
    const s = squash(action);
    if (s.includes('toogood')) return ['nodouble'];
    if (s.startsWith('nodouble') || s.startsWith('noredouble')) return ['nodouble'];
    if (s === 'doubletake') return ['double', 'take'];
    if (s === 'doublepass') return ['double', 'pass'];
    if (s === 'redouble') return ['double'];
    if (s === 'take') return ['double', 'take'];
    if (s === 'pass' || s === 'drop') return ['double', 'pass'];
    return [s]; // "double", etc.
}

// isResponseCubeAction: true for a pure take/pass response, false for any doubling decision
// (including "Double/Take", "No Double"). Mirrors Go engine.IsResponseCubeAction.
export function isResponseCubeAction(action) {
    const s = squash(action);
    if (s.includes('double') || s.includes('toogood')) return false; // double, double/take, double/pass, nodouble, redouble
    return s === 'dt' || s === 'dp' || s.includes('take') || s.includes('pass') || s.includes('drop');
}
