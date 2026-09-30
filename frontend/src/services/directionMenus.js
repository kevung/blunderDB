/**
 * directionMenus.js — le contenu des menus contextuels de la page Direction : une fonction par
 * objet, qui rend des `MenuItem[]` sans toucher au DOM (les vues les ouvrent, voir
 * contextMenuTrigger.js). Une entrée n'existe que si l'action est possible sur l'objet : le
 * menu ne propose rien qu'il faudrait ensuite refuser.
 *
 * Les gestes qui retirent ou désignent un perdant se confirment ici, avec les mêmes phrases que
 * la fiche de résultat et les boutons de ligne : le menu est un second chemin vers le même
 * geste, jamais un chemin plus court vers un geste non confirmé.
 */

/** @typedef {import('./contextMenuTrigger.js').MenuItem} MenuItem */

/**
 * @typedef {(key: string, params?: Record<string, unknown>) => string} Translate
 *
 * @typedef {{
 *     matchId?: string, a?: string, b?: string, aName?: string, bName?: string,
 *     running?: boolean, done?: boolean, table?: number, noTable?: boolean
 * }} MatchLike
 *
 * Les gestes qu'une vue sait faire ; une clé absente retire l'entrée du menu.
 * @typedef {{
 *     confirm?: (message: string) => boolean,
 *     openResult?: () => void,
 *     openMove?: () => void,
 *     onForfeit?: (matchId: string, winner: string, note: string) => unknown,
 *     onCancel?: (matchId: string) => unknown,
 *     onCorrect?: () => void,
 *     onHistory?: (name: string) => void,
 *     onOpenMatch?: (matchId: string) => void,
 *     onDetach?: () => void,
 *     onTranscribe?: () => void
 * }} MatchActions
 */

/** @param {MatchActions} h @param {string} message */
function ask(h, message) {
    return (h.confirm ?? ((m) => window.confirm(m)))(message);
}

/**
 * Un match, où qu'il s'affiche (case de table, place de l'arbre, emplacement).
 *
 * @param {Translate} t
 * @param {MatchLike} m
 * @param {MatchActions} h
 * @returns {MenuItem[]}
 */
export function matchMenu(t, m, h) {
    /** @type {MenuItem[]} */
    const items = [];
    const id = m.matchId || '';
    const aName = m.aName || m.a || '';
    const bName = m.bName || m.b || '';
    if (m.running && id) {
        if (h.openResult) items.push({ label: t('direction.menu.enterResult'), shortcut: '↵', onClick: h.openResult });
        const forfeit = h.onForfeit;
        if (forfeit) {
            items.push({
                label: t('direction.menu.forfeitOf', { name: aName }),
                onClick: () => {
                    if (ask(h, t('direction.result.forfeitConfirm', { loser: aName, winner: bName }))) forfeit(id, m.b || '', '');
                }
            });
            items.push({
                label: t('direction.menu.forfeitOf', { name: bName }),
                onClick: () => {
                    if (ask(h, t('direction.result.forfeitConfirm', { loser: bName, winner: aName }))) forfeit(id, m.a || '', '');
                }
            });
        }
        if (h.openMove) {
            items.push({ label: t('direction.menu.moveTable'), shortcut: 'M', onClick: h.openMove });
            items.push({ label: t('direction.menu.swapTable'), shortcut: 'X', onClick: h.openMove });
        }
        const cancel = h.onCancel;
        if (cancel) {
            items.push({
                label: t('direction.menu.cancelMatch'),
                onClick: () => {
                    if (ask(h, t('direction.result.cancelConfirm', { a: aName, b: bName }))) cancel(id);
                }
            });
        }
    } else if (m.done && id && h.onCorrect) {
        items.push({ label: t('direction.menu.correctResult'), onClick: h.onCorrect });
    }
    if (h.onHistory && aName) items.push({ label: t('direction.menu.historyOf', { name: aName }), onClick: () => h.onHistory?.(aName) });
    if (h.onHistory && bName) items.push({ label: t('direction.menu.historyOf', { name: bName }), onClick: () => h.onHistory?.(bName) });
    if (h.onOpenMatch && id && !m.running) items.push({ label: t('direction.menu.openMatch'), onClick: () => h.onOpenMatch?.(id) });
    if (h.onDetach) items.push({ label: t('direction.menu.detach'), onClick: h.onDetach });
    if (h.onTranscribe) items.push({ label: t('direction.menu.transcribe'), onClick: h.onTranscribe });
    return items;
}

/**
 * Une case de la grille des tables : le match qui s'y joue, ou la table elle-même quand elle
 * est libre (hors service / remise en service).
 *
 * @param {Translate} t
 * @param {MatchLike & { free?: boolean, unavailable?: boolean, elsewhere?: string }} c
 * @param {MatchActions & { onOutOfService?: (table: number, out: boolean) => void }} h
 * @returns {MenuItem[]}
 */
export function cellMenu(t, c, h) {
    if (c.matchId) return matchMenu(t, { ...c, running: true }, h);
    if (c.elsewhere || !h.onOutOfService || !c.table) return [];
    const table = c.table;
    const toggle = h.onOutOfService;
    if (c.unavailable) return [{ label: t('direction.menu.backInService'), onClick: () => toggle(table, false) }];
    return [{ label: t('direction.menu.outOfService'), onClick: () => toggle(table, true) }];
}

/**
 * Un joueur. `state` est celui de la ligne des Joueurs : free, playing, absent, withdrawn.
 *
 * @param {Translate} t
 * @param {{ id: string, name: string, state?: string, table?: number }} p
 * @param {{
 *     confirm?: (message: string) => boolean,
 *     onGoTable?: (table: number, open: boolean) => void,
 *     onHistory?: (name: string) => void,
 *     onAbsent?: (id: string) => void,
 *     onReturn?: (id: string) => void,
 *     onWithdraw?: (id: string, afterCurrent: boolean) => unknown,
 *     onReinstate?: (id: string) => void,
 *     onEdit?: (id: string) => void
 * }} h
 * @returns {MenuItem[]}
 */
export function playerMenu(t, p, h) {
    /** @type {MenuItem[]} */
    const items = [];
    const playing = p.state === 'playing' && !!p.table;
    if (playing && h.onGoTable) {
        const table = /** @type {number} */ (p.table);
        items.push({ label: t('direction.menu.enterResult'), onClick: () => h.onGoTable?.(table, true) });
        items.push({ label: t('direction.menu.goToTable', { n: table }), onClick: () => h.onGoTable?.(table, false) });
    }
    if (h.onHistory) items.push({ label: t('direction.menu.historyOf', { name: p.name }), onClick: () => h.onHistory?.(p.name) });
    if (p.state === 'free' && h.onAbsent) items.push({ label: t('direction.menu.absent'), onClick: () => h.onAbsent?.(p.id) });
    if (p.state === 'absent' && h.onReturn) items.push({ label: t('direction.menu.present'), onClick: () => h.onReturn?.(p.id) });
    const withdraw = h.onWithdraw;
    if (withdraw && p.state !== 'withdrawn') {
        items.push({
            label: t('direction.menu.withdrawNow'),
            onClick: () => {
                if (ask(h, t('direction.players.withdrawNowConfirm', { name: p.name }))) withdraw(p.id, false);
            }
        });
        if (p.state === 'playing') {
            items.push({
                label: t('direction.menu.withdrawLater'),
                onClick: () => {
                    if (ask(h, t('direction.players.withdrawLaterConfirm', { name: p.name }))) withdraw(p.id, true);
                }
            });
        }
    }
    if (p.state === 'withdrawn' && h.onReinstate) items.push({ label: t('direction.menu.reinstate'), onClick: () => h.onReinstate?.(p.id) });
    if (h.onEdit) items.push({ label: t('direction.menu.correctPlayer'), onClick: () => h.onEdit?.(p.id) });
    return items;
}

/**
 * Une proposition de la file.
 *
 * @param {Translate} t
 * @param {{ kind: string, a?: string, b?: string }} a
 * @param {{ busy?: boolean, onLaunch?: () => void, onIgnore?: () => void, onArrange?: () => void }} h
 * @returns {MenuItem[]}
 */
export function proposalMenu(t, a, h) {
    /** @type {MenuItem[]} */
    const items = [];
    if (h.onLaunch) items.push({ label: t('direction.menu.launch'), shortcut: '↵', disabled: !!h.busy, onClick: h.onLaunch });
    if (h.onArrange && a.kind === 'start_match' && a.a && a.b) items.push({ label: t('direction.menu.arrangeOther'), onClick: h.onArrange });
    if (h.onIgnore) items.push({ label: t('direction.menu.ignore'), onClick: h.onIgnore });
    return items;
}

/**
 * Une ligne de l'historique.
 *
 * @param {Translate} t
 * @param {{ correctable?: boolean, cancellable?: boolean, matchId?: string, aName?: string, bName?: string, a?: string, b?: string }} e
 * @param {{
 *     confirm?: (message: string) => boolean,
 *     onCorrect?: () => void,
 *     onCancel?: (matchId: string) => unknown,
 *     onNote?: () => void,
 *     onFilter?: (name: string) => void
 * }} h
 * @returns {MenuItem[]}
 */
export function historyMenu(t, e, h) {
    /** @type {MenuItem[]} */
    const items = [];
    const aName = e.aName || e.a || '';
    const bName = e.bName || e.b || '';
    if (e.correctable && h.onCorrect) items.push({ label: t('direction.menu.correctResult'), onClick: h.onCorrect });
    const cancel = h.onCancel;
    if (e.cancellable && !e.correctable && cancel) {
        items.push({
            label: t('direction.menu.cancelMatch'),
            onClick: () => {
                if (ask(h, t('direction.result.cancelConfirm', { a: aName, b: bName }))) cancel(e.matchId || '');
            }
        });
    }
    if (h.onNote) items.push({ label: t('direction.menu.addNote'), onClick: h.onNote });
    if (h.onFilter && aName) items.push({ label: t('direction.menu.filterOn', { name: aName }), onClick: () => h.onFilter?.(aName) });
    if (h.onFilter && bName) items.push({ label: t('direction.menu.filterOn', { name: bName }), onClick: () => h.onFilter?.(bName) });
    return items;
}
