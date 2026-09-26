/*
 * Le rendu des codes du moteur (ADR-0047, Nicomaque `codes.go`), qui n'émet aucune phrase :
 * la traduction se fait ici, une fois pour toutes les vues. Un code inconnu s'affiche tel
 * quel, pour que la clé manquante se voie.
 */

/**
 * La valeur du store `$t`, ou son équivalent dans un test.
 *
 * @typedef {(key: string, params?: Record<string, unknown>) => any} Translate
 */
/** @typedef {import('../../stores/directionStore.js').DirectionLabel} DirectionLabel */
/** @typedef {import('../../stores/directionStore.js').ProposalAction} ProposalAction */
/** @typedef {(id: string | undefined) => string | undefined} PlayerName */

/**
 * Rend un libellé structuré (`Label`) dans la langue de l'utilisateur.
 *
 * @param {Translate} t
 * @param {DirectionLabel | null | undefined} label
 * @returns {string}
 */
export function renderLabel(t, label) {
    if (!label || !label.kind) return '';
    /** @type {string} */
    const sub = label.sub ? renderLabel(t, label.sub) : '';
    const key = `direction.label.${label.kind}`;
    /** @type {string} */
    const out = t(key, {
        n: label.n ?? 0,
        losses: label.losses ?? 0,
        match: label.match ?? 0,
        players: label.players ?? 0,
        spots: label.spots ?? 0,
        section: renderSectionName(t, label.section),
        text: label.text ?? '',
        sub
    });
    return out === key ? label.kind : out;
}

/**
 * Rend le nom d'une section, un identifiant du moteur (« main », « conso », « poule:A »).
 *
 * @param {Translate} t
 * @param {string | null | undefined} name
 * @returns {string}
 */
export function renderSectionName(t, name) {
    if (!name) return '';
    const pool = name.startsWith('poule:') ? name.slice('poule:'.length) : null;
    if (pool) return t('direction.section.pool', { letter: pool });
    const barrage = name.startsWith('barrage:') ? name.slice('barrage:'.length) : null;
    if (barrage) return t('direction.section.playoff', { letter: barrage });
    const key = `direction.section.${name}`;
    const out = t(key);
    return out === key ? name : out;
}

/**
 * Rend une note de classement (`Note`).
 *
 * @param {Translate} t
 * @param {{ kind?: string, wins?: number, losses?: number, lives?: number, section?: string, sub?: DirectionLabel } | null | undefined} note
 * @returns {string}
 */
export function renderNote(t, note) {
    if (!note || !note.kind) return '';
    const key = `direction.note.${note.kind}`;
    const out = t(key, {
        wins: note.wins ?? 0,
        losses: note.losses ?? 0,
        lives: note.lives ?? 0,
        section: renderSectionName(t, note.section),
        sub: note.sub ? renderLabel(t, note.sub) : ''
    });
    return out === key ? note.kind : out;
}

/**
 * Rend un avertissement (`Warning`) : ce que le moteur signale sans jamais bloquer.
 *
 * @param {Translate} t
 * @param {{ code?: string, match?: string, section?: string, label?: DirectionLabel, a?: string, b?: string, expected_a?: string, expected_b?: string, length?: number, score_a?: number, score_b?: number } | null | undefined} w
 * @param {PlayerName} [playerName]
 * @returns {string}
 */
export function renderWarning(t, w, playerName = (id) => id) {
    if (!w || !w.code) return '';
    const key = `direction.warning.${w.code}`;
    const out = t(key, {
        match: w.match ?? '',
        section: renderSectionName(t, w.section),
        label: renderLabel(t, w.label),
        a: playerName(w.a),
        b: playerName(w.b),
        expectedA: playerName(w.expected_a),
        expectedB: playerName(w.expected_b),
        length: w.length ?? 0,
        scoreA: w.score_a ?? 0,
        scoreB: w.score_b ?? 0
    });
    return out === key ? w.code : out;
}

/**
 * Le texte d'une proposition : le libellé du moteur situe le match, les deux noms disent qui joue.
 *
 * @param {Translate} t
 * @param {{ kind: string, a?: string, b?: string, label?: DirectionLabel }} a
 * @param {PlayerName} [playerName]
 * @returns {string}
 */
export function proposalLabel(t, a, playerName = (id) => id) {
    const where = renderLabel(t, a.label);
    switch (a.kind) {
        case 'start_match':
            return t('direction.proposals.match', {
                a: playerName(a.a),
                b: playerName(a.b),
                where
            });
        case 'bye':
            return t('direction.proposals.bye', { player: playerName(a.a), where });
        case 'draw':
            return t('direction.proposals.draw', { where });
        case 'next_phase':
            return t('direction.proposals.nextPhase', { where });
        case 'finish':
            return t('direction.proposals.finish');
        case 'cancel_match':
            // Une annulation n'est proposée que pour réparer (Nicomaque reparation.go).
            return t('direction.proposals.repair', {
                where,
                a: playerName(a.a),
                b: playerName(a.b)
            });
        default:
            return where || a.kind;
    }
}

/**
 * Clé stable d'une proposition : Svelte réutilise la ligne, et « ignorer pour l'instant »
 * désigne la même au prochain appel (le moteur est déterministe).
 *
 * @param {ProposalAction} a
 */
export function actionKey(a) {
    return [a.kind, a.phase, a.section || '', a.key || '', a.a || '', a.b || ''].join('|');
}

/**
 * Rend une valeur de configuration ; une valeur vide se dit « aucune », jamais un trou.
 *
 * @param {Translate} t
 * @param {string} code
 * @param {string | null | undefined} value
 * @returns {string}
 */
function renderConfigValue(t, code, value) {
    if (value === undefined || value === null || value === '') return t('direction.change.none');
    if (value === 'true') return t('direction.change.yes');
    if (value === 'false') return t('direction.change.no');
    if (code === 'kind' || code === 'phaseAdded' || code === 'phaseRemoved') {
        const key = `direction.format.${value}`;
        const out = t(key);
        return out === key ? value : out;
    }
    return value;
}

/**
 * Rend une ligne de la liste « voici ce qui va changer », valeur d'avant et d'après.
 *
 * @param {Translate} t
 * @param {{ code?: string, phase?: number, from?: string, to?: string } | null | undefined} change
 * @returns {string}
 */
export function renderConfigChange(t, change) {
    if (!change || !change.code) return '';
    const key = `direction.change.${change.code}`;
    const out = t(key, {
        phase: change.phase ?? 0,
        from: renderConfigValue(t, change.code, change.from),
        to: renderConfigValue(t, change.code, change.to)
    });
    return out === key ? change.code : out;
}

/**
 * Rend la raison pour laquelle le format d'une phase ne change plus.
 *
 * @param {Translate} t
 * @param {string | null | undefined} reason
 * @returns {string}
 */
export function renderLockReason(t, reason) {
    if (!reason) return '';
    const key = `direction.lock.${reason}`;
    const out = t(key);
    return out === key ? reason : out;
}

/**
 * Vrai quand la proposition fait partie d'une réparation : le moteur ne propose une
 * annulation que dans ce cas.
 *
 * @param {{ kind?: string } | null | undefined} a
 */
export function isRepair(a) {
    return !!a && a.kind === 'cancel_match';
}

/**
 * Nom proposé pour un CSV : `<tournoi>-<mot>-<AAAA-MM-JJ>.csv`, mot traduit ; caractères
 * interdits et espaces → tiret, accents gardés.
 *
 * @param {string} tournament
 * @param {string} word
 * @param {Date} [date]
 */
export function csvFilename(tournament, word, date = new Date()) {
    const pad = (/** @type {number} */ n) => String(n).padStart(2, '0');
    const day = `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
    const slug = (/** @type {string} */ s) =>
        (s || '')
            .replace(/[\s\\/:*?"<>|]+/g, '-')
            .replace(/-+/g, '-')
            .replace(/^-|-$/g, '');
    return [slug(tournament), slug(word), day].filter(Boolean).join('-') + '.csv';
}
