<!--
  TranscriptView — a Transcript: the two columns of a score sheet.

  The glossary (CONTEXT.md, "Recording a match") gives the word one meaning:
  the RENDERING of a Match or of a Transcription as two columns — player 1
  left, player 2 right, one row per turn, the cube action and the end of the
  game in the column of whoever acted. That is the layout of `testdata/test.mat`
  and of every score sheet, and it is what this component draws.

  It is presentational and it holds NO store. Everything it shows is handed to
  it as one `annotated` object and read from it — the side of an Action, its
  notation, the score a game started on, the Crawford mention, the
  Inconsistencies a Replay found, and the Action being TYPED, which is drawn in
  dashes in the slot it will fill so that what is read and what is recorded
  never say two different things. Nothing here recomputes a derived fact: the
  Go Replay (pkg/blunderdb/transcript) is the only thing that derives, and a
  second opinion written in JavaScript would be a second, drifting one.

  It knows nothing about transcription either — no draft, no gesture, no
  Cursor gesture — so that the Match panel can mount it one day on a stored
  match (tasks/transcription/integration.md §4): it takes an index to frame and
  calls back with the index that was clicked — or right-clicked, `onMenu` —, and
  what those indices mean is the caller's business. The one field it owns is the
  move typed into a cell on a double-click (ADR-0052): a text box and its
  Enter/Escape, handed back as `onEditMove(index, text, pending)` — what the
  text means, and whether it is accepted, is the caller's business too. The
  other is the score of a game, typed in its header on a double-click
  (ADR-0053) and handed back as `onEditScore(opening, score)`.

  That promise used to be false in one place: a `.mat` pane lived here, with a
  clipboard call, an acknowledgement timer and a fold state that belong to a
  DRAFT — a stored match has its own export. It is a modal of the transcription
  panel now (ADR-0048 decision 6), which is also where 62 columns of aligned
  ASCII can be read: the 320 px column this component gets destroyed the
  alignment, which is the only reason to look at a `.mat` at all.

  The one thing it computes is the LAYOUT — which cell of which row an Action
  goes in — because that is a fact about the two columns and about nothing
  else. `transcriptRows` is the same rule as ingest/mat_export.go's
  `layoutRows`, written once here and exported so a test can read it straight.
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
     * L'Action en cours de SAISIE, là où elle atterrira, ou `null`.
     *
     * Pourquoi elle est dessinée. Le Transcript ne montrait que le document, et
     * l'Entry n'en fait pas partie : une correction laissait donc la cellule
     * afficher l'Action enregistrée jusqu'à la validation, et une insertion
     * n'apparaissait nulle part avant elle. L'utilisateur lisait une chose
     * pendant qu'il en tapait une autre — l'écart entre ce qu'on voit et ce qui
     * est enregistré, que le Transcript existe précisément pour fermer.
     *
     * Rien n'est dérivé ici : le moteur dit où l'Entry tombe (`at`), si elle
     * remplace, son camp, ses dés, la notation du coup choisi et la sorte
     * d'Action que sa validation écrirait (`transcript.EntryInfo`).
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
        // Une correction ne se dessine que lorsqu'elle DIT quelque chose : le
        // Cursor posé sur une Action sans rien taper dessus montre l'Action, et
        // non un fantôme d'elle-même.
        if (replacing && (!typed || at >= infos.length)) return null;

        const gameIndex = at < infos.length ? infos[at].game_index : infos.length ? infos[infos.length - 1].game_index : -1;
        if (gameIndex < 0 || gameIndex >= games.length) return null;
        // Une partie finie n'accueille pas la suite : l'Action tapée ouvrira la
        // partie suivante, qui n'existe pas encore et n'a donc aucun tableau où
        // la poser. Elle s'y dessinera dès que son ouverture sera enregistrée.
        if (!replacing && at >= infos.length && games[gameIndex]?.finished) return null;

        return {
            at,
            gameIndex,
            replacing,
            cell: { kind: 'pending', index: at, entry: e, side: e.side ?? 0, opening: e.kind === 'opening', replacing }
        };
    }

    /**
     * The rows of each game: player 1's cell on the left, player 2's on the
     * right, one row per turn.
     *
     * The rule is `ingest.RenderMAT`'s, and it is the one every score sheet
     * follows: an Action of player 1 opens a row, an Action of player 2 joins
     * the row player 1 just opened or takes one of its own with the left cell
     * blank. An opening belongs to neither column — its two dice are one
     * player's each — so it takes a row of its own across both. The end of the
     * game is a cell like any other and belongs to the WINNER's column.
     *
     * The Action being typed follows the same rule, in the slot it will fill:
     * over the cell it replaces, between the two it is inserted between (see
     * [pendingOf]).
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
                // A new row when there is none, when the cell is taken, or
                // when player 1 acts after player 2 answered on this one.
                if (!row || row[side] || (side === 'left' && row.right)) open(true);
                row[side] = cell;
            };

            for (const info of infos) {
                if (info.game_index !== gameIndex) continue;
                if (pending && pending.gameIndex === gameIndex && pending.at === info.index) {
                    place(pending.cell, pending.cell.opening, pending.cell.side);
                    // Une correction TIENT LA PLACE de l'Action : les deux ne se
                    // montrent pas côte à côte, sans quoi le même coup se lirait
                    // deux fois, une fois comme il était et une fois comme il
                    // devient.
                    if (pending.replacing) continue;
                }
                place({ kind: 'action', index: info.index, info }, info.kind === 'opening', info.side);
            }
            if (pending && pending.gameIndex === gameIndex && pending.at >= infos.length) {
                place(pending.cell, pending.cell.opening, pending.cell.side);
            }

            // " Wins N points": the result of a finished game, in the winner's
            // column, on a row that carries no turn and therefore no number.
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
         * Told when a cell is RIGHT-clicked: `(index, {x, y}) => void`, in
         * client pixels. The caller opens whatever menu it wants there; this
         * component knows no gesture and offers none.
         *
         * The native menu is suppressed on those cells, and only there: the
         * right button means something else in the rest of the application,
         * and a Transcript that swallowed it everywhere would take away the
         * browser's own menu from the panel around it (fiche T2.5).
         */
        onMenu = null,
        /**
         * Told when the move typed into a cell is validated with Enter:
         * `(index, text, pending) => boolean` — `pending` for the dashed cell
         * of the Action being typed. A `true` answer closes the field; `false`
         * (a text that says no move) leaves it open. Without it, a double-click
         * edits nothing (ADR-0052).
         */
        onEditMove = null,
        /**
         * Told when the score typed into a game's header is validated with
         * Enter: `(opening, score) => boolean` — `opening` is the index of the
         * game's opening, `score` is `[p1, p2]`, or `null` when the field was
         * emptied (the declared score is cleared, ADR-0053). A `true` answer
         * closes the field. Without it, the score is not editable.
         */
        onEditScore = null
    } = $props();

    let header = $derived(annotated?.document?.header ?? {});
    let at = $derived(cursor ?? annotated?.cursor ?? -1);
    let layout = $derived(transcriptRows(annotated));

    // L'index de la cellule PROVISOIRE effectivement dessinée, ou −1. Elle porte
    // le cadre du Cursor — c'est elle que l'on tape —, et l'Action qu'elle
    // recouvre ne le porte donc pas une seconde fois.
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

    // ── le coup tapé dans sa cellule (ADR-0052) ──────────────────────────
    //
    // Un double-clic sur une cellule de coup la change en champ, pré-rempli de
    // sa notation. On n'y tape QUE le coup : les dés sont ceux de la cellule, et
    // le champ ne les montre pas. Entrée le rend à l'appelant, Échap le ferme
    // sans rien écrire, et le quitter aussi — un champ abandonné n'est pas une
    // validation.
    //
    // Échap passe par escapeService, écouté en capture : c'est le seul moyen
    // qu'il ferme le champ et RIEN d'autre, ni la saisie du panneau, ni le
    // répartiteur global. Les autres touches restent au champ : le panneau et
    // le répartiteur laissent passer tout ce qui est tapé dans un champ
    // (panelKeyGuard, keyboardService).

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
     * Le champ ouvert est-il celui de cette cellule ? Une correction en place
     * TIENT la place de l'Action (voir [pendingOf]) : le champ ouvert sur la
     * cellule reste donc le sien quand le moteur y dessine la saisie.
     *
     * @param {any} c
     */
    function editsCell(c) {
        if (!editing || !c || c.index !== editing.index) return false;
        if (c.kind === 'pending') return editing.pending || c.replacing === true;
        return c.kind === 'action' && !editing.pending;
    }

    // ── le score annoncé d'une partie (ADR-0053) ─────────────────────────
    //
    // Même mécanique que le coup tapé dans sa cellule : un double-clic sur le
    // score de l'en-tête le change en champ, pré-rempli du score affiché ;
    // Entrée le rend à l'appelant, Échap (escapeService) et la perte du focus
    // le ferment sans rien écrire. Un champ vidé puis validé efface le score
    // annoncé : la partie revient au score que donnent les précédentes.

    /** @type {{opening: number, number: number, text: string} | null} */
    let scoring = $state(null);

    $effect(() => {
        if (scoring) return closeOnEscape(() => (scoring = null));
    });

    /**
     * L'index de l'ouverture qui commence la partie, ou −1 quand son score ne
     * se tape pas : en session d'argent (pas de score), ou quand la partie ne
     * commence pas par une ouverture.
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
        // Rien de ce qui est tapé ici n'atteint la machine à touches du
        // panneau ni le répartiteur global, ni le <summary> qui plierait la
        // partie sur une espace.
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
     * Le champ prend la main à son ouverture, texte sélectionné : on retape
     * par-dessus, ou l'on corrige au bout.
     *
     * @param {HTMLInputElement} node
     */
    function focusField(node) {
        node.focus();
        node.select();
    }

    // The dice live on the document's Action (the Replay does not copy them
    // onto its ActionInfo), so the two are read side by side, by index.
    let actions = $derived(annotated?.document?.actions ?? []);

    // The game the Cursor is in — open, while the others start folded.
    let currentGame = $derived.by(() => {
        const infos = annotated?.actions ?? [];
        const here = at >= 0 && at < infos.length ? infos[at] : null;
        if (here) return here.game_number;
        const games = annotated?.games ?? [];
        return games.length ? games[games.length - 1].number : 0;
    });

    // What the user said about a game, when they said anything: a game with no
    // entry here follows the rule (the current one is open, the rest folded).
    const folds = new SvelteMap();
    const isOpen = (/** @type {number} */ number) => (folds.has(number) ? folds.get(number) : number === currentGame);

    // Entering a game clears the fold the user had put on it — otherwise the
    // Cursor would walk into a game whose cells nobody can see. `seen` is a
    // plain variable on purpose: the effect must react to the game changing,
    // never to its own write.
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

    // Le Transcript défile jusqu'à la cellule encadrée. C'est ce qui rend
    // visible le saut du Cursor sur la première Incohérence après un Replay
    // (fonctionnel.md §1.4) : le moteur l'y place, et une correction faite
    // vingt tours plus haut serait autrement encadrée hors de l'écran.
    //
    // `block: 'nearest'` ne bouge rien quand la cellule est déjà visible, ce qui
    // évite de faire sauter la page à chaque touche pendant une saisie normale.
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

    // The keys are written out rather than assembled: a key built at runtime
    // (`'transcript.inconsistency.' + kind`) is invisible to the guard that
    // hunts orphaned translations, and the seven Inconsistencies of
    // fonctionnel.md §1.4 are a closed list anyway.
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
                // The roll is known, the play is not. It is named, never left to
                // look like a dance: the file says nobody wrote the play down, and
                // the cell has to say the same thing.
                return `${dice[0]}${dice[1]}: ${$t('transcript.unrecorded')}`;
            case 'double':
                // The value the cube reaches. `before.cube.value` is the log2
                // exponent everywhere in blunderDB (see the XGID contract), and
                // the offer doubles it — the same arithmetic the board and the
                // cube panel already do to print a cube.
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
     * L'Action en cours de saisie, telle qu'on la tape : les dés au fur et à
     * mesure qu'ils tombent, « · » pour celui qui manque encore, et la notation
     * du coup choisi dès qu'il y en a un.
     *
     * C'est un ÉTAT DE SAISIE et il se voit comme tel (la cellule est en
     * pointillés) : rien n'est écrit dans le document avant la validation.
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
                                    // Le score qui se tape ne plie pas la partie :
                                    // le premier clic du double-clic la replierait.
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
    /* Borné, et c'est ce qui rend le `.scroller` effectif : sans `flex: 1` ni
       `min-height: 0` sur toute la chaîne, un `overflow: auto` n'a rien à faire
       déborder et le conteneur du panneau devient le seul qui défile — d'où le
       `scrollIntoView` du Cursor qui faisait sauter le panneau entier à chaque
       Action validée (ADR-0048 décision 5). */
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

    /* L'Action en cours de saisie : dessinée à sa place, mais en pointillés —
       rien n'est écrit dans le document avant la validation, et la cellule le
       dit d'elle-même plutôt que par une phrase ailleurs. */
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

    /* Le coup tapé dans sa cellule : la cellule encadrée, devenue champ. */
    .move-field {
        border-color: var(--color-primary);
        background: var(--color-surface);
    }
</style>
