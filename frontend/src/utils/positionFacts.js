// Converts mover/opponent-relative position facts (fractions, cubeless equity from the mover's
// view) into board sides bottom/top, as PositionFactsTable renders them (ADR-0017). onRoll is
// domain.Black (0, bottom) or White (1, top). A null mover passes nulls through, so the table
// always gets {bottom, top} (ADR-0017 rule 3).
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
