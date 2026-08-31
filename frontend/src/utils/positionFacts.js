// Converts mover/opponent-relative position facts (win, gammon, backgammon
// as fractions, cubeless as an equity from the mover's point of view — the
// convention gammonNet and imported cube analyses both use) into the
// board's own bottom/top sides — the physical convention PositionFactsTable
// renders (CONTEXT.md's "Position fact" is per player, meaning per board
// side, not per mover; see ADR-0017).
//
// onRoll is domain.Black (0, bottom) or domain.White (1, top). mover is
// null when no fact is available yet — the caller passes through nulls
// rather than deciding not to call this, so the facts table always gets a
// {bottom, top} shape and renders its usual blank cells (ADR-0017 rule 3:
// structure never depends on the state of the calculation).
export function moverFactsToSides(mover, opponent, onRoll) {
    if (!mover) return { bottom: null, top: null };
    const moverSide = { win: mover.win, gammon: mover.gammon, backgammon: mover.backgammon, cubeless: mover.cubeless };
    const opponentSide = {
        win: opponent?.win ?? null,
        gammon: opponent?.gammon ?? null,
        backgammon: opponent?.backgammon ?? null,
        cubeless: mover.cubeless == null ? null : -mover.cubeless
    };
    return onRoll === 1 ? { bottom: opponentSide, top: moverSide } : { bottom: moverSide, top: opponentSide };
}
