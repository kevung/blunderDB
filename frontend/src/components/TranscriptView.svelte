<!--
  TranscriptView — a Transcript: the two columns of a score sheet (CONTEXT.md,
  "Recording a match"): player 1 left, player 2 right, one row per turn, cube
  actions and game ends in the column of whoever acted.

  Presentational, no store: everything comes from one `annotated` object,
  including the Action being typed (drawn dashed in its slot). The Go Replay
  (pkg/blunderdb/transcript) is the only thing that derives; nothing is
  recomputed here. It knows no draft or gesture, so a stored match can mount it
  too: it frames an index and calls back with indices. It owns only two
  fields: a move typed into a cell (ADR-0052, `onEditMove`) and a game's
  score typed in its header (ADR-0053, `onEditScore`).

  The one thing it computes is the LAYOUT: `transcriptRows` follows
  ingest/mat_export.go's `layoutRows`, exported for tests.
-->
<script module>
    /**
     * Le score tapé dans l'en-tête d'une partie : « 3-2 », « 3–2 », « 3 2 ».
     * Rend `[3, 2]`, `null` pour un champ vide — effacer le score annoncé —, et
     * `undefined` pour un texte qui ne dit pas de score.
     *
     * @param {string} text
     * @returns {[number, number] | null | undefined}
     */
    export function parseScore(text) {
        const s = (text ?? '').trim();
        if (!s) return null;
        const m = /^(\d{1,3})\s*(?:[-–—:]|\s)\s*(\d{1,3})$/.exec(s);
        return m ? [Number(m[1]), Number(m[2])] : undefined;
    }

    /**
     * L'Action en cours de saisie, là où elle atterrira, ou `null` — dessinée
     * pour que la cellule montre ce qu'on tape, pas l'Action qu'elle remplace.
     * Tout vient du moteur (`transcript.EntryInfo`).
     *
     * @param {any} annotated
     * @param {any[]} infos
     * @param {any[]} games
     */
    function pendingOf(annotated, infos, games) {
        const e = annotated?.entry;
        if (!e) return null;
        const at = Math.max(0, Math.min(e.at ?? 0, infos.length));
        const replacing = e.replacing === true;
        const typed = (e.dice?.[0] ?? 0) > 0 || (e.dice?.[1] ?? 0) > 0 || !!e.notation;
        // Rien de tapé : montrer l'Action, pas un fantôme d'elle-même.
        if (replacing && (!typed || at >= infos.length)) return null;

        const gameIndex = at < infos.length ? infos[at].game_index : infos.length ? infos[infos.length - 1].game_index : -1;
        if (gameIndex < 0 || gameIndex >= games.length) return null;
        // Partie finie : la saisie ouvrira la suivante, pas encore dessinable.
        if (!replacing && at >= infos.length && games[gameIndex]?.finished) return null;

        return {
            at,
            gameIndex,
            replacing,
            cell: { kind: 'pending', index: at, entry: e, side: e.side ?? 0, opening: e.kind === 'opening', replacing }
        };
    }

    /**
     * @param {any} info
     * @param {string} kind
     */
    function hasFlaw(info, kind) {
        return !!info?.inconsistencies?.some((/** @type {{kind: string}} */ f) => f.kind === kind);
    }

    /**
     * The rows of each game, by `ingest.RenderMAT`'s rule: player 1 opens a
     * row, player 2 joins it or takes one with the left cell blank; an opening
     * spans both columns; a game end sits in the winner's column. The Action
     * being typed takes the slot it will fill (see [pendingOf]).
     *
     * @param {any} annotated - a `transcript.Annotated` as the Go side returns it
     * @returns {{game: any, rows: {left: any, right: any, full: any, numbered: boolean}[]}[]}
     */
    export function transcriptRows(annotated) {
        const infos = annotated?.actions ?? [];
        const games = annotated?.games ?? [];
        const pending = pendingOf(annotated, infos, games);

        return games.map((/** @type {any} */ game, /** @type {number} */ gameIndex) => {
            /** @type {any[]} */
            const rows = [];
            /** @type {any} */
            let row = null;

            const open = (/** @type {boolean} */ numbered) => {
                row = { left: null, right: null, full: null, numbered };
                rows.push(row);
                return row;
            };

            /** Une cellule à sa place — la règle du `.mat`, pour les deux sortes. */
            const place = (/** @type {any} */ cell, /** @type {boolean} */ opening, /** @type {number} */ camp) => {
                if (opening) {
                    open(true).full = cell;
                    row = null;
                    return;
                }
                const side = camp === 1 ? 'right' : 'left';
                if (!row || row[side] || (side === 'left' && row.right)) open(true);
                row[side] = cell;
            };

            for (const info of infos) {
                if (info.game_index !== gameIndex) continue;
                const filling = pending && !pending.replacing && pending.gameIndex === gameIndex && pending.at === info.index;
                // A double turn's missing turn is a cell of its own (ADR-0054),
                // unless the insertion being typed fills it.
                if (!filling && info.kind !== 'opening' && hasFlaw(info, 'double_turn')) {
                    place({ kind: 'hole', index: info.index, side: info.side === 1 ? 0 : 1 }, false, info.side === 1 ? 0 : 1);
                }
                if (pending && pending.gameIndex === gameIndex && pending.at === info.index) {
                    place(pending.cell, pending.cell.opening, pending.cell.side);
                    // Une correction tient la place de l'Action, jamais à côté.
                    if (pending.replacing) continue;
                }
                place({ kind: 'action', index: info.index, info }, info.kind === 'opening', info.side);
            }
            if (pending && pending.gameIndex === gameIndex && pending.at >= infos.length) {
                place(pending.cell, pending.cell.opening, pending.cell.side);
            }

            // " Wins N points": winner's column, unnumbered row.
            if (game.winner === 0 || game.winner === 1) {
                const result = open(false);
                result[game.winner === 1 ? 'right' : 'left'] = { kind: 'result', points: game.points_won };
                row = null;
            }

            return { game, rows };
        });
    }
</script>

<script>
    import { SvelteMap } from 'svelte/reactivity';
    import { t } from '../i18n';
    import { closeOnEscape } from '../services/escapeService.js';

    let {
        /** A `transcript.Annotated`, verbatim. */
        annotated = null,
        /** The Action to frame, as an index into `annotated.actions`. */
        cursor = null,
        /** `[player 1, player 2]`; the header's names are used when absent. */
        players = null,
        /** Told when a cell is clicked: `(index) => void`. */
        onSelect = null,
        /**
         * Told when the hole of a double turn is clicked: `(index) => void`,
         * `index` being the Action the hole stands before (ADR-0054).
         */
        onHole = null,
        /**
         * Told when a cell is right-clicked: `(index, {x, y}) => void`, client
         * pixels. The native menu is suppressed on cells only.
         */
        onMenu = null,
        /**
         * `(index, text, pending) => boolean` on Enter in a cell's move field;
         * `true` closes it. Absent: cells are not editable (ADR-0052).
         */
        onEditMove = null,
        /**
         * `(opening, score) => boolean` on Enter in a game's score field;
         * `score` is `[p1, p2]` or `null` to clear it (ADR-0053); `true` closes it.
         */
        onEditScore = null
    } = $props();

    let header = $derived(annotated?.document?.header ?? {});
    let at = $derived(cursor ?? annotated?.cursor ?? -1);
    let layout = $derived(transcriptRows(annotated));

    // Index de la cellule provisoire dessinée, ou −1 ; elle porte seule le cadre du Cursor.
    let pendingIndex = $derived.by(() => {
        for (const group of layout) {
            for (const row of group.rows) {
                for (const c of [row.full, row.left, row.right]) {
                    if (c?.kind === 'pending') return c.index;
                }
            }
        }
        return -1;
    });

    // Coup tapé dans sa cellule (ADR-0052) : double-clic, seul le coup s'y tape.
    // Entrée valide ; Échap et la perte du focus ferment sans écrire. Échap passe
    // par escapeService (capture) pour ne fermer que le champ.

    /** @type {{index: number, pending: boolean, text: string} | null} */
    let editing = $state(null);

    $effect(() => {
        if (editing) return closeOnEscape(() => (editing = null));
    });

    /**
     * La cellule se tape-t-elle ? Un coup de pions, une danse, un coup non
     * consigné — et la saisie en cours quand ses deux dés sont là.
     *
     * @param {any} c
     */
    function typable(c) {
        if (!onEditMove || !c) return false;
        if (c.kind === 'pending') return !c.opening && (c.entry?.dice?.[0] ?? 0) > 0 && (c.entry?.dice?.[1] ?? 0) > 0;
        return c.kind === 'action' && (c.info?.kind === 'checker' || c.info?.kind === 'dance' || c.info?.kind === 'unrecorded');
    }

    /**
     * Le champ ouvert est-il celui de cette cellule (y compris sa saisie en place) ?
     *
     * @param {any} c
     */
    function editsCell(c) {
        if (!editing || !c || c.index !== editing.index) return false;
        if (c.kind === 'pending') return editing.pending || c.replacing === true;
        return c.kind === 'action' && !editing.pending;
    }

    // Score annoncé (ADR-0053), même mécanique ; vidé puis validé, il s'efface.

    /** @type {{opening: number, number: number, text: string} | null} */
    let scoring = $state(null);

    $effect(() => {
        if (scoring) return closeOnEscape(() => (scoring = null));
    });

    /**
     * Index de l'ouverture de la partie, ou −1 si son score ne se tape pas
     * (argent, ou pas d'ouverture).
     *
     * @param {any} game
     */
    function scoreOpening(game) {
        if (!onEditScore || !((header.match_length ?? 0) > 0)) return -1;
        const first = game?.first ?? -1;
        return first >= 0 && annotated?.actions?.[first]?.kind === 'opening' ? first : -1;
    }

    /** @param {any} game */
    function startScore(game) {
        const opening = scoreOpening(game);
        if (opening < 0) return;
        const [a, b] = game.initial_score ?? [0, 0];
        scoring = { opening, number: game.number, text: `${a}-${b}` };
    }

    /** @param {KeyboardEvent} event */
    function scoreKeyDown(event) {
        // Rien n'atteint le panneau, le répartiteur ni le <summary> (plié par espace).
        event.stopPropagation();
        if (event.key !== 'Enter' || !scoring) return;
        event.preventDefault();
        const score = parseScore(scoring.text);
        if (score === undefined) return;
        if (onEditScore?.(scoring.opening, score)) scoring = null;
    }

    /**
     * Le score diffère-t-il de celui que donnent les parties précédentes ?
     * C'est le moteur qui le dit (GameInfo.declared/derived_score).
     *
     * @param {any} game
     */
    function scoreDiffers(game) {
        const d = game?.derived_score;
        const s = game?.initial_score;
        return game?.declared === true && !!d && !!s && (d[0] !== s[0] || d[1] !== s[1]);
    }

    /** @param {any} c */
    function startEdit(c) {
        if (!typable(c)) return;
        const text = c.kind === 'pending' ? (c.entry?.notation ?? '') : c.info?.kind === 'checker' ? (c.info.notation ?? '') : '';
        editing = { index: c.index, pending: c.kind === 'pending', text };
    }

    /** @param {KeyboardEvent} event */
    function editKeyDown(event) {
        if (event.key !== 'Enter' || !editing) return;
        event.preventDefault();
        event.stopPropagation();
        const { index, pending, text } = editing;
        if (onEditMove?.(index, text, pending)) editing = null;
    }

    /**
     * Focus à l'ouverture, texte sélectionné.
     *
     * @param {HTMLInputElement} node
     */
    function focusField(node) {
        node.focus();
        node.select();
    }

    // Dice live on the document's Action, not on ActionInfo: read both by index.
    let actions = $derived(annotated?.document?.actions ?? []);

    // The game the Cursor is in — open, while the others start folded.
    let currentGame = $derived.by(() => {
        const infos = annotated?.actions ?? [];
        const here = at >= 0 && at < infos.length ? infos[at] : null;
        if (here) return here.game_number;
        const games = annotated?.games ?? [];
        return games.length ? games[games.length - 1].number : 0;
    });

    // User folds; absent = current game open, others folded.
    const folds = new SvelteMap();
    const isOpen = (/** @type {number} */ number) => (folds.has(number) ? folds.get(number) : number === currentGame);

    // Entering a game unfolds it. `seen` is plain so the effect reacts to the
    // game, never to its own write.
    let seen = -1;
    $effect(() => {
        const number = currentGame;
        if (number === seen) return;
        seen = number;
        folds.delete(number);
    });

    /**
     * @param {number} number
     * @param {boolean} open
     */
    function toggleGame(number, open) {
        folds.set(number, open);
    }

    // Défile jusqu'à la cellule encadrée (le saut sur une Incohérence,
    // fonctionnel.md §1.4) ; `block: 'nearest'` ne bouge rien si elle est visible.
    let scroller = $state(/** @type {HTMLDivElement | null} */ (null));
    $effect(() => {
        void at;
        void layout;
        const cell = scroller?.querySelector('.cell.cursor');
        if (typeof cell?.scrollIntoView === 'function') cell.scrollIntoView({ block: 'nearest', inline: 'nearest' });
    });

    /** @param {number} side */
    function playerName(side) {
        const given = players?.[side];
        if (given) return given;
        const named = side === 0 ? header.player1 : header.player2;
        return named || (side === 0 ? $t('transcript.player1') : $t('transcript.player2'));
    }

    // Keys written out: runtime-built keys escape the orphan-translation guard.
    /** @type {Record<string, string>} */
    const FLAW_KEY = {
        illegal_move: 'transcript.inconsistency.illegal_move',
        double_turn: 'transcript.inconsistency.double_turn',
        impossible_cube: 'transcript.inconsistency.impossible_cube',
        past_end: 'transcript.inconsistency.past_end',
        inconsistent_dice: 'transcript.inconsistency.inconsistent_dice',
        unrecorded_move: 'transcript.inconsistency.unrecorded_move',
        score_mismatch: 'transcript.inconsistency.score_mismatch'
    };
    /** @type {Record<number, string>} */
    const RESIGN_KEY = { 1: 'transcript.resignSingle', 2: 'transcript.resignGammon', 3: 'transcript.resignBackgammon' };

    /**
     * "31: 8/5 6/5", "Doubles => 2", "Wins 2 points" — one cell's text.
     *
     * @param {any} c
     */
    function cellText(c) {
        if (!c) return '';
        if (c.kind === 'result') {
            return c.points === 1 ? $t('transcript.winsOne') : $t('transcript.wins', { n: c.points });
        }
        if (c.kind === 'pending') return pendingText(c.entry);
        const info = c.info;
        const action = actions[c.index] ?? {};
        const dice = action.dice ?? [0, 0];
        switch (info.kind) {
            case 'opening':
                return dice[0] === dice[1] ? $t('transcript.openingTie', { a: dice[0], b: dice[1] }) : $t('transcript.opening', { a: dice[0], b: dice[1] });
            case 'checker':
                return `${dice[0]}${dice[1]}: ${info.notation ?? ''}`.trim();
            case 'dance':
                return `${dice[0]}${dice[1]}: ${$t('transcript.dance')}`;
            case 'unrecorded':
                // Roll known, play unrecorded: named, never shown as a dance.
                return `${dice[0]}${dice[1]}: ${$t('transcript.unrecorded')}`;
            case 'double':
                // `before.cube.value` is the log2 exponent; the offer doubles it.
                return $t('transcript.doubles', { n: 2 ** ((info.before?.cube?.value ?? 0) + 1) });
            case 'take':
                return $t('transcript.takes');
            case 'pass':
                return $t('transcript.drops');
            case 'resign':
                return $t('transcript.resigns', { level: $t(RESIGN_KEY[action.level] ?? RESIGN_KEY[1]) });
            default:
                return info.notation ?? '';
        }
    }

    /**
     * L'Action en cours de saisie : dés tapés (« · » pour un manquant) et coup choisi.
     *
     * @param {any} e - `transcript.EntryInfo`
     */
    function pendingText(e) {
        const a = e.dice?.[0] || '·';
        const b = e.dice?.[1] || '·';
        if (e.kind === 'opening') return $t('transcript.opening', { a, b });
        return e.notation ? `${a}${b}: ${e.notation}` : `${a}${b}`;
    }

    /**
     * The Inconsistencies of a cell, NAMED — never a bare warning sign.
     *
     * @param {any} c
     * @returns {{kind: string}[]}
     */
    function flawsOf(c) {
        return c?.kind === 'action' ? (c.info.inconsistencies ?? []) : [];
    }

    /** @param {{kind: string}[]} flaws */
    function flawTitle(flaws) {
        return $t('transcript.flawed', { names: flaws.map((f) => (FLAW_KEY[f.kind] ? $t(FLAW_KEY[f.kind]) : f.kind)).join(' · ') });
    }
</script>

<div class="transcript-view" role="group" aria-label={$t('transcript.title')}>
    <div class="scroller" bind:this={scroller}>
        {#if !layout.length}
            <p class="empty">{$t('transcript.empty')}</p>
        {:else}
            {#each layout as group (group.game.number)}
                {@const open = isOpen(group.game.number)}
                <details class="game" {open} ontoggle={(e) => toggleGame(group.game.number, e.currentTarget.open)}>
                    <summary class="game-header">
                        <span class="game-title">{$t('transcript.game', { n: group.game.number })}</span>
                        {#if scoring && scoring.number === group.game.number}
                            <input
                                class="score-field"
                                type="text"
                                size="7"
                                aria-label={$t('transcript.scoreField')}
                                placeholder={$t('transcript.scoreFieldPlaceholder')}
                                bind:value={scoring.text}
                                onkeydown={scoreKeyDown}
                                onkeyup={(e) => e.stopPropagation()}
                                onclick={(e) => e.preventDefault()}
                                onblur={() => (scoring = null)}
                                use:focusField
                            />
                        {:else}
                            {@const editable = scoreOpening(group.game) >= 0}
                            {@const differs = scoreDiffers(group.game)}
                            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
                            <span
                                class="game-score"
                                class:editable
                                class:flawed={differs}
                                data-declared={group.game.declared ? 'true' : undefined}
                                title={differs ? $t('transcript.derivedScore', { a: group.game.derived_score[0], b: group.game.derived_score[1] }) : editable ? $t('transcript.editScore') : undefined}
                                onclick={(e) => {
                                    // Le premier clic du double-clic ne plie pas la partie.
                                    if (editable) e.preventDefault();
                                }}
                                ondblclick={() => startScore(group.game)}
                                >{group.game.initial_score[0]}–{group.game.initial_score[1]}{#if differs}<span class="flaw-mark" aria-hidden="true">⚠</span>{/if}</span
                            >
                        {/if}
                        {#if group.game.crawford}
                            <span class="tag">{$t('transcript.crawford')}</span>
                        {/if}
                    </summary>
                    {#if open}
                        <table class="turns">
                            <thead>
                                <tr>
                                    <th class="num" aria-label={$t('transcript.turn')}></th>
                                    <th class="side">{playerName(0)}</th>
                                    <th class="side">{playerName(1)}</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each group.rows as row, i (i)}
                                    <tr>
                                        <td class="num">{row.numbered ? i + 1 : ''}</td>
                                        {#if row.full}
                                            <td class="side" colspan="2">{@render cellBlock(row.full)}</td>
                                        {:else}
                                            <td class="side">{@render cellBlock(row.left)}</td>
                                            <td class="side">{@render cellBlock(row.right)}</td>
                                        {/if}
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    {/if}
                </details>
            {/each}
        {/if}
    </div>
</div>

<!--
  One cell. It is a button when the caller listens, a span when it does not:
  a Transcript nobody can walk is still a Transcript.
-->
{#snippet cellBlock(/** @type {any} */ c)}
    {#if !c}
        <span class="cell empty-cell"></span>
    {:else if c.kind === 'hole'}
        {#if onHole}
            <button type="button" class="cell hole" data-hole={c.index} title={$t('transcript.hole')} aria-label={$t('transcript.hole')} onclick={() => onHole(c.index)}>&nbsp;</button>
        {:else}
            <span class="cell hole" data-hole={c.index}>&nbsp;</span>
        {/if}
    {:else if editing && editsCell(c)}
        <input
            class="cell move-field"
            type="text"
            data-index={c.index}
            aria-label={$t('transcript.moveField')}
            placeholder={$t('transcript.moveFieldPlaceholder')}
            bind:value={editing.text}
            onkeydown={editKeyDown}
            onblur={() => (editing = null)}
            use:focusField
        />
    {:else}
        {@const flaws = flawsOf(c)}
        {@const framed = c.kind === 'pending' || (c.kind === 'action' && c.index === at && pendingIndex !== c.index)}
        {#if onSelect && c.kind !== 'result'}
            <button
                type="button"
                class="cell"
                class:cursor={framed}
                class:pending={c.kind === 'pending'}
                class:flawed={flaws.length > 0}
                data-index={c.index}
                data-pending={c.kind === 'pending' ? 'true' : undefined}
                data-inconsistency={flaws.length ? flaws.map((/** @type {{kind: string}} */ f) => f.kind).join(' ') : undefined}
                aria-current={framed ? 'true' : undefined}
                title={flaws.length ? flawTitle(flaws) : undefined}
                onclick={() => onSelect(c.index)}
                ondblclick={() => startEdit(c)}
                oncontextmenu={(event) => {
                    if (!onMenu) return;
                    event.preventDefault();
                    onMenu(c.index, { x: event.clientX, y: event.clientY });
                }}
            >
                {cellText(c)}{#if flaws.length}<span class="flaw-mark" aria-hidden="true">⚠</span>{/if}
            </button>
        {:else}
            <span
                class="cell"
                class:cursor={framed}
                class:pending={c.kind === 'pending'}
                class:flawed={flaws.length > 0}
                class:result={c.kind === 'result'}
                data-index={c.kind === 'result' ? undefined : c.index}
                data-pending={c.kind === 'pending' ? 'true' : undefined}
                data-inconsistency={flaws.length ? flaws.map((/** @type {{kind: string}} */ f) => f.kind).join(' ') : undefined}
                aria-current={framed ? 'true' : undefined}
                title={flaws.length ? flawTitle(flaws) : undefined}
            >
                {cellText(c)}{#if flaws.length}<span class="flaw-mark" aria-hidden="true">⚠</span>{/if}
            </span>
        {/if}
    {/if}
{/snippet}

<style>
    /* Borné pour que `.scroller` défile, pas le panneau (ADR-0048 décision 5). */
    .transcript-view {
        display: flex;
        flex: 1;
        flex-direction: column;
        min-height: 0;
        gap: var(--space-1);
    }

    /* The wide content scrolls in ITS OWN box; the page never scrolls sideways. */
    .scroller {
        flex: 1;
        min-height: 0;
        overflow: auto;
    }

    .empty {
        margin: var(--space-2);
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .game {
        border-bottom: 1px solid var(--color-border);
    }

    .game-header {
        display: flex;
        align-items: baseline;
        gap: var(--space-2);
        padding: var(--space-1) var(--space-2);
        cursor: pointer;
        list-style: none;
    }

    .game-header::-webkit-details-marker {
        display: none;
    }

    .game-header::before {
        content: '▸';
        color: var(--color-text-muted);
    }

    .game[open] > .game-header::before {
        content: '▾';
    }

    .game-title {
        font-weight: 600;
    }

    .game-score,
    .tag {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .game-score.editable {
        cursor: text;
    }

    .game-score.flawed {
        color: var(--color-danger);
    }

    /* Le score tapé dans l'en-tête : le score, devenu champ. */
    .score-field {
        padding: 0 var(--space-1);
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        font-size: var(--font-size-small);
    }

    .turns {
        width: 100%;
        border-collapse: collapse;
    }

    .turns th {
        position: sticky;
        top: 0;
        padding: var(--space-1) var(--space-2);
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface);
        color: var(--color-text-muted);
        font-weight: 600;
        text-align: left;
    }

    .turns td {
        padding: 0 var(--space-1);
        vertical-align: top;
    }

    .num {
        width: 2.5em;
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
        text-align: right;
    }

    .side {
        width: 50%;
    }

    .cell {
        display: block;
        width: 100%;
        padding: var(--space-1);
        border: 1px solid transparent;
        border-radius: var(--radius);
        background: none;
        color: var(--color-text);
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        text-align: left;
        white-space: nowrap;
    }

    button.cell {
        cursor: pointer;
    }

    button.cell:hover {
        background: var(--color-surface-alt);
    }

    /* The Cursor is a CELL, and it is framed (ux.md §5). */
    .cell.cursor {
        border-color: var(--color-primary);
        font-weight: 600;
    }

    /* Saisie en cours : pointillés, rien n'est encore écrit. */
    .cell.pending {
        border-style: dashed;
        border-color: var(--color-primary);
        color: var(--color-text-muted);
        font-style: italic;
    }

    .cell.flawed {
        color: var(--color-danger);
    }

    .flaw-mark {
        margin-left: var(--space-1);
    }

    .cell.result {
        color: var(--color-text-muted);
        font-style: italic;
    }

    .empty-cell {
        border-color: transparent;
    }

    /* Tour perdu d'un double trait (ADR-0054), marqué comme Incohérence. */
    .cell.hole {
        border-style: dashed;
        border-color: var(--color-danger);
    }

    /* Le coup tapé dans sa cellule : la cellule encadrée, devenue champ. */
    .move-field {
        border-color: var(--color-primary);
        background: var(--color-surface);
    }
</style>
