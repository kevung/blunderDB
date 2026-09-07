/*
 * Le rendu des CODES du moteur (ADR-0047, Nicomaque `codes.go`).
 *
 * Le moteur n'émet aucune phrase : un libellé de match, une note de classement, un
 * avertissement ou une raison d'attente sont un code et ses paramètres. C'est ce qui permet à
 * blunderDB de parler neuf langues d'un moteur qui n'en parle aucune, et c'est ici que la
 * traduction se fait — une seule fois, pour toutes les vues.
 *
 * Un code inconnu n'est jamais masqué : il s'affiche tel quel. Une version future du moteur qui
 * ajouterait un libellé se verra donc à l'écran plutôt que de laisser un blanc, et la clé
 * manquante nomme elle-même ce qu'il faut traduire.
 */

/** Rend un libellé structuré (`Label`) dans la langue de l'utilisateur. */
export function renderLabel(t, label) {
    if (!label || !label.kind) return '';
    const sub = label.sub ? renderLabel(t, label.sub) : '';
    const key = `direction.label.${label.kind}`;
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
 * Rend le nom d'une section. Un nom de section est un IDENTIFIANT côté moteur — « main »,
 * « conso », « poule:A » — jamais un libellé ; c'est ici qu'il devient lisible.
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

/** Rend une note de classement (`Note`). */
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

/** Rend un avertissement (`Warning`) : ce que le moteur signale sans jamais bloquer. */
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
 * Le texte d'une proposition, dans la langue de l'utilisateur : ce que le directeur lit avant
 * de cliquer. Le libellé du moteur (« quart de finale », « 1 défaite, match 3 ») situe le
 * match ; les deux noms disent qui joue.
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
            return t('direction.proposals.cancel', { where });
        default:
            return where || a.kind;
    }
}

/**
 * Une clé stable pour une proposition, afin que Svelte réutilise la ligne plutôt que de la
 * recréer — et pour que « ignorer pour l'instant » désigne bien la même proposition au
 * prochain appel, puisque le moteur est déterministe.
 */
export function actionKey(a) {
    return [a.kind, a.phase, a.section || '', a.key || '', a.a || '', a.b || ''].join('|');
}

/**
 * Rend une VALEUR de configuration : un nombre reste un nombre, un booléen devient oui/non, un
 * type de phase passe par son nom de format. Une valeur vide se dit « aucune », sans quoi la
 * liste des changements comporterait des trous que personne ne sait lire.
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
 * Rend une ligne de la liste « voici ce qui va changer » (issue #385).
 *
 * Appliquer une configuration en cours de tournoi n'est pas l'enregistrement d'un formulaire,
 * c'est une décision : elle se montre avant d'être prise, réglage par réglage, avec la valeur
 * d'avant et celle d'après.
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

/** Rend la raison pour laquelle le format d'une phase ne change plus. */
export function renderLockReason(t, reason) {
    if (!reason) return '';
    const key = `direction.lock.${reason}`;
    const out = t(key);
    return out === key ? reason : out;
}
