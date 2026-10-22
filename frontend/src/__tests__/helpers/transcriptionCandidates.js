/**
 * The answer of the TranscriptionCandidates binding (transcription.Candidates),
 * built from a test's fixtures: `plays` in the generator's order (LegalMoves'
 * shape, `{notation, steps}`) and `ranked`, the engine's moves best first
 * (`{move, equity, ...}`), joined by notation as the Go side does. A play the
 * engine left out follows the ranked ones, rank 0; no ranked play recognised
 * (or `ranked` null) is the unranked answer, generator order.
 *
 * @param {Array<{notation: string, steps?: any[]}>} plays
 * @param {Array<any>|null} ranked
 * @param {{side?: number, dice?: number[]}} [roll]
 */
export function candidatesAnswer(plays, ranked, roll = {}) {
    const base = { side: roll.side ?? 0, dice: roll.dice ?? [0, 0] };
    const index = new Map();
    plays.forEach((p, i) => {
        if (!index.has(p.notation)) index.set(p.notation, i);
    });
    const list = [];
    for (const m of ranked ?? []) {
        const i = index.get(m.move);
        if (i === undefined) continue;
        list.push({ index: i, notation: plays[i].notation, steps: plays[i].steps ?? [], rank: list.length + 1, analysis: m });
    }
    if (list.length === 0) {
        return { ...base, unranked: true, candidates: plays.map((p, i) => ({ index: i, notation: p.notation, steps: p.steps ?? [], rank: 0 })) };
    }
    const taken = new Set(list.map((c) => c.index));
    plays.forEach((p, i) => {
        if (!taken.has(i)) list.push({ index: i, notation: p.notation, steps: p.steps ?? [], rank: 0 });
    });
    return { ...base, unranked: false, candidates: list };
}
