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
  Inconsistencies a Replay found. Nothing here recomputes a derived fact: the
  Go Replay (pkg/blunderdb/transcript) is the only thing that derives, and a
  second opinion written in JavaScript would be a second, drifting one.

  It knows nothing about transcription either — no draft, no gesture, no
  Cursor gesture — so that the Match panel can mount it one day on a stored
  match (tasks/transcription/integration.md §4): it takes an index to frame and
  calls back with the index that was clicked — or right-clicked, `onMenu` —, and
  what those indices mean is the caller's business.

  The one thing it computes is the LAYOUT — which cell of which row an Action
  goes in — because that is a fact about the two columns and about nothing
  else. `transcriptRows` is the same rule as ingest/mat_export.go's
  `layoutRows`, written once here and exported so a test can read it straight.
-->
<script module>
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
     * @param {any} annotated - a `transcript.Annotated` as the Go side returns it
     * @returns {{game: any, rows: {left: any, right: any, full: any, numbered: boolean}[]}[]}
     */
    export function transcriptRows(annotated) {
        const infos = annotated?.actions ?? [];
        const games = annotated?.games ?? [];

        return games.map((game, gameIndex) => {
            /** @type {any[]} */
            const rows = [];
            let row = null;

            const open = (numbered) => {
                row = { left: null, right: null, full: null, numbered };
                rows.push(row);
                return row;
            };

            for (const info of infos) {
                if (info.game_index !== gameIndex) continue;
                const cell = { kind: 'action', index: info.index, info };

                if (info.kind === 'opening') {
                    open(true).full = cell;
                    row = null;
                    continue;
                }

                const side = info.side === 1 ? 'right' : 'left';
                // A new row when there is none, when the cell is taken, or
                // when player 1 acts after player 2 answered on this one.
                if (!row || row[side] || (side === 'left' && row.right)) open(true);
                row[side] = cell;
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
    import { onDestroy } from 'svelte';
    import { SvelteMap } from 'svelte/reactivity';
    import { t } from '../i18n';
    import { writeTextToClipboard } from '../services/clipboardService.js';

    let {
        /** A `transcript.Annotated`, verbatim. */
        annotated = null,
        /** The Action to frame, as an index into `annotated.actions`. */
        cursor = null,
        /** `[player 1, player 2]`; the header's names are used when absent. */
        players = null,
        /** The exact `.mat` text of the same document, for the pane below. */
        matText = '',
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
        /** Told when the `.mat` pane is folded or unfolded: `(open) => void`. */
        onMatToggle = null,
        /** Overrides the copy button's action; the clipboard by default. */
        onCopy = null
    } = $props();

    let header = $derived(annotated?.document?.header ?? {});
    let at = $derived(cursor ?? annotated?.cursor ?? -1);
    let layout = $derived(transcriptRows(annotated));

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
    const isOpen = (number) => (folds.has(number) ? folds.get(number) : number === currentGame);

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
    let scroller = $state(null);
    $effect(() => {
        void at;
        void layout;
        const cell = scroller?.querySelector('.cell.cursor');
        if (typeof cell?.scrollIntoView === 'function') cell.scrollIntoView({ block: 'nearest', inline: 'nearest' });
    });

    function playerName(side) {
        const given = players?.[side];
        if (given) return given;
        const named = side === 0 ? header.player1 : header.player2;
        return named || (side === 0 ? $t('transcript.player1') : $t('transcript.player2'));
    }

    // The keys are written out rather than assembled: a key built at runtime
    // (`'transcript.inconsistency.' + kind`) is invisible to the guard that
    // hunts orphaned translations, and the six Inconsistencies of
    // fonctionnel.md §1.4 are a closed list anyway.
    const FLAW_KEY = {
        illegal_move: 'transcript.inconsistency.illegal_move',
        double_turn: 'transcript.inconsistency.double_turn',
        impossible_cube: 'transcript.inconsistency.impossible_cube',
        past_end: 'transcript.inconsistency.past_end',
        inconsistent_dice: 'transcript.inconsistency.inconsistent_dice',
        unrecorded_move: 'transcript.inconsistency.unrecorded_move'
    };
    const RESIGN_KEY = { 1: 'transcript.resignSingle', 2: 'transcript.resignGammon', 3: 'transcript.resignBackgammon' };

    /** "31: 8/5 6/5", "Doubles => 2", "Wins 2 points" — one cell's text. */
    function cellText(c) {
        if (!c) return '';
        if (c.kind === 'result') {
            return c.points === 1 ? $t('transcript.winsOne') : $t('transcript.wins', { n: c.points });
        }
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

    /** The Inconsistencies of a cell, NAMED — never a bare warning sign. */
    function flawsOf(c) {
        return c?.kind === 'action' ? (c.info.inconsistencies ?? []) : [];
    }

    function flawTitle(flaws) {
        return $t('transcript.flawed', { names: flaws.map((f) => (FLAW_KEY[f.kind] ? $t(FLAW_KEY[f.kind]) : f.kind)).join(' · ') });
    }

    let copied = $state(false);
    let copyTimer = null;

    async function copy() {
        if (onCopy) onCopy(matText);
        else await writeTextToClipboard(matText);
        copied = true;
        clearTimeout(copyTimer);
        copyTimer = setTimeout(() => (copied = false), 1500);
    }

    onDestroy(() => clearTimeout(copyTimer));
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
                        <span class="game-score">{group.game.initial_score[0]}–{group.game.initial_score[1]}</span>
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

    {#if matText !== null}
        <details class="mat" ontoggle={(e) => onMatToggle?.(e.currentTarget.open)}>
            <summary class="mat-header">{$t('transcript.matPane')}</summary>
            <div class="mat-body">
                <button class="copy-btn" type="button" onclick={copy}>{copied ? $t('transcript.copied') : $t('transcript.copy')}</button>
                <pre class="mat-text">{matText}</pre>
            </div>
        </details>
    {/if}
</div>

<!--
  One cell. It is a button when the caller listens, a span when it does not:
  a Transcript nobody can walk is still a Transcript.
-->
{#snippet cellBlock(c)}
    {#if !c}
        <span class="cell empty-cell"></span>
    {:else}
        {@const flaws = flawsOf(c)}
        {@const framed = c.kind === 'action' && c.index === at}
        {#if onSelect && c.kind === 'action'}
            <button
                type="button"
                class="cell"
                class:cursor={framed}
                class:flawed={flaws.length > 0}
                data-index={c.index}
                data-inconsistency={flaws.length ? flaws.map((f) => f.kind).join(' ') : undefined}
                aria-current={framed ? 'true' : undefined}
                title={flaws.length ? flawTitle(flaws) : undefined}
                onclick={() => onSelect(c.index)}
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
                class:flawed={flaws.length > 0}
                class:result={c.kind === 'result'}
                data-index={c.kind === 'action' ? c.index : undefined}
                data-inconsistency={flaws.length ? flaws.map((f) => f.kind).join(' ') : undefined}
                aria-current={framed ? 'true' : undefined}
                title={flaws.length ? flawTitle(flaws) : undefined}
            >
                {cellText(c)}{#if flaws.length}<span class="flaw-mark" aria-hidden="true">⚠</span>{/if}
            </span>
        {/if}
    {/if}
{/snippet}

<style>
    .transcript-view {
        display: flex;
        flex-direction: column;
        min-height: 0;
        gap: var(--space-1);
    }

    /* The wide content scrolls in ITS OWN box; the page never scrolls sideways. */
    .scroller {
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

    .mat {
        border-top: 1px solid var(--color-border);
    }

    .mat-header {
        padding: var(--space-1) var(--space-2);
        color: var(--color-text-muted);
        cursor: pointer;
    }

    .mat-body {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-1) var(--space-2);
    }

    .copy-btn {
        align-self: flex-start;
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .copy-btn:hover {
        background: var(--color-surface-alt);
    }

    .mat-text {
        margin: 0;
        max-height: 16em;
        overflow: auto;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
    }
</style>
