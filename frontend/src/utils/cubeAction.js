// Interpreting cube action strings ("Double", "Double/Take", "No Double", "Take", …) in one place,
// so the analysis panel and the board agree.

// normalizeCubeAction maps an action to the analysis-row parts it highlights ('nodouble' |
// 'double' | 'take' | 'pass'); a standalone Take/Pass maps onto the combined Double/… row.
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

// isResponseCubeAction: true for a pure take/pass response, false for any doubling decision
// (including "Double/Take", "No Double"). Mirrors Go engine.IsResponseCubeAction.
export function isResponseCubeAction(action) {
    const s = (action || '').toLowerCase().replace(/\s+/g, '');
    if (s.includes('double')) return false; // double, double/take, double/pass, nodouble, redouble
    return s === 'dt' || s === 'dp' || s.includes('take') || s.includes('pass') || s.includes('drop');
}
