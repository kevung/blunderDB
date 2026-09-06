<script>
    // Le front web (#295, fiche J.5), dont le périmètre est verrouillé par
    // l'ADR-0039 : consulter, chercher, réviser. Rien d'autre, jamais.
    //
    // Le plateau est dessiné par LE dessinateur — le même que le rapport HTML
    // et l'export d'image, celui que #278 a extrait pour qu'il n'y en ait
    // qu'un. Un second dessinateur en JavaScript nu aurait été plus léger à
    // écrire et impossible à tenir en phase.
    import { call } from './api.js';
    import { renderPositionSVG } from '../services/diagramService.js';

    let query = $state('');
    let positions = $state([]);
    let analyses = $state({});
    let index = $state(0);
    let error = $state('');
    let busy = $state(false);
    let mode = $state('browse');

    // La révision : la carte courante, et si la réponse est dévoilée.
    let decks = $state([]);
    let deckId = $state(0);
    let card = $state(null);
    let revealed = $state(false);

    let current = $derived(positions[index] || null);
    let currentAnalysis = $derived(current ? analyses[current.id] || null : null);
    let boardSVG = $derived(current ? renderPositionSVG(current, { width: 460, height: 340 }) : '');
    let cardSVG = $derived(card ? renderPositionSVG(card.position, { width: 460, height: 340 }) : '');

    async function run(action) {
        busy = true;
        error = '';
        try {
            await action();
        } catch (e) {
            error = String(e.message || e);
        } finally {
            busy = false;
        }
    }

    async function search() {
        await run(async () => {
            // La MÊME grammaire que la ligne de commande : le front web n'a
            // pas de langage de recherche à lui, ce qui est la raison pour
            // laquelle il ne peut pas en avoir un qui dérive.
            const rows = (await call('search.query', { query: query.trim(), limit: 200 })) || [];
            positions = rows.map((row) => row.position || row).filter(Boolean);
            analyses = {};
            for (const row of rows) {
                if (row?.analysis && row?.position?.id) analyses[row.position.id] = row.analysis;
            }
            index = 0;
            if (positions.length === 0) error = 'Aucune position ne correspond.';
        });
    }

    async function loadDecks() {
        await run(async () => {
            decks = (await call('anki.listDecks')) || [];
            if (decks.length > 0 && !deckId) deckId = decks[0].id;
        });
    }

    async function nextCard() {
        await run(async () => {
            revealed = false;
            card = await call('anki.nextCard', { deckId });
        });
    }

    async function grade(rating) {
        await run(async () => {
            await call('anki.reviewCard', { cardId: card.card.id, rating });
            // L'autre moitié d'une décision de videau vient tout de suite
            // quand elle est due (#276) — la même règle que sur le bureau,
            // parce que c'est la même route.
            const linked = await call('anki.linkedCard', { deckId, cardId: card.card.id }).catch(() => null);
            revealed = false;
            card = linked || (await call('anki.nextCard', { deckId }));
        });
    }

    function moveRows(analysis) {
        return analysis?.checkerAnalysis?.moves || [];
    }

    function fmt(value, digits = 3) {
        return typeof value === 'number' ? value.toFixed(digits) : '—';
    }
</script>

<header>
    <h1>blunderDB</h1>
    <nav>
        <button type="button" class:active={mode === 'browse'} onclick={() => (mode = 'browse')}>Consulter</button>
        <button
            type="button"
            class:active={mode === 'review'}
            onclick={() => {
                mode = 'review';
                if (decks.length === 0) loadDecks();
            }}>Réviser</button
        >
    </nav>
</header>

{#if error}
    <p class="error">{error}</p>
{/if}

{#if mode === 'browse'}
    <form
        class="search"
        onsubmit={(e) => {
            e.preventDefault();
            search();
        }}
    >
        <input type="text" bind:value={query} placeholder="E&gt;100 gt:holding" aria-label="Recherche" />
        <button type="submit" disabled={busy}>Chercher</button>
    </form>

    {#if current}
        <!-- Le SVG vient de NOTRE dessinateur, jamais d'une saisie : two.js
             l'assemble à partir de nombres et de couleurs, et une position ne
             porte aucun texte libre qui serait rendu ici. C'est la raison —
             pas l'habitude — pour laquelle la règle est levée sur cette ligne
             et seulement sur elle. -->
        <!-- eslint-disable-next-line svelte/no-at-html-tags -->
        <div class="board">{@html boardSVG}</div>
        <p class="nav">
            <button type="button" disabled={index === 0} onclick={() => (index -= 1)}>◀</button>
            <span>{index + 1} / {positions.length}</span>
            <button type="button" disabled={index >= positions.length - 1} onclick={() => (index += 1)}>▶</button>
        </p>
        {#if currentAnalysis}
            <table class="moves">
                <thead>
                    <tr><th>Coup</th><th>Équité</th><th>Erreur</th></tr>
                </thead>
                <tbody>
                    {#each moveRows(currentAnalysis) as move (move.move)}
                        <tr>
                            <td>{move.move}</td>
                            <td class="num">{fmt(move.equity)}</td>
                            <td class="num">{fmt(move.equityError)}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
            {#if currentAnalysis.doublingCubeAnalysis}
                <p class="verdict">{currentAnalysis.doublingCubeAnalysis.bestCubeAction}</p>
            {/if}
        {:else}
            <p class="note">Cette position n'a pas d'analyse enregistrée.</p>
        {/if}
    {/if}
{:else}
    <div class="review">
        <label>
            Paquet
            <select bind:value={deckId} onchange={() => (card = null)}>
                {#each decks as deck (deck.id)}
                    <option value={deck.id}>{deck.name}</option>
                {/each}
            </select>
        </label>
        <button type="button" disabled={!deckId || busy} onclick={nextCard}>Commencer</button>
    </div>

    {#if card}
        <!-- Même raison qu'au-dessus : le SVG est produit ici, pas reçu. -->
        <!-- eslint-disable-next-line svelte/no-at-html-tags -->
        <div class="board">{@html cardSVG}</div>
        {#if revealed}
            <table class="moves">
                <thead>
                    <tr><th>Coup</th><th>Équité</th><th>Erreur</th></tr>
                </thead>
                <tbody>
                    {#each moveRows(card.analysis) as move (move.move)}
                        <tr>
                            <td>{move.move}</td>
                            <td class="num">{fmt(move.equity)}</td>
                            <td class="num">{fmt(move.equityError)}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
            <p class="grades">
                <button type="button" onclick={() => grade(1)}>À revoir</button>
                <button type="button" onclick={() => grade(2)}>Difficile</button>
                <button type="button" onclick={() => grade(3)}>Bien</button>
                <button type="button" onclick={() => grade(4)}>Facile</button>
            </p>
        {:else}
            <p class="grades"><button type="button" onclick={() => (revealed = true)}>Voir la réponse</button></p>
        {/if}
    {/if}
{/if}

<style>
    :global(body) {
        margin: 0;
        padding: 0.6rem;
        font-family: var(--font-family-ui);
        font-size: var(--font-size-base);
        color: var(--color-text);
        background: var(--color-surface);
    }

    header {
        display: flex;
        align-items: baseline;
        gap: 1rem;
        flex-wrap: wrap;
    }

    h1 {
        margin: 0;
        font-size: var(--font-size-title);
    }

    nav button,
    .search button,
    .grades button,
    .nav button,
    .review button {
        cursor: pointer;
    }

    nav button.active {
        font-weight: 700;
    }

    .search,
    .review {
        display: flex;
        gap: 0.4rem;
        align-items: center;
        margin: 0.6rem 0;
        flex-wrap: wrap;
    }

    .search input {
        flex: 1;
        min-width: 12rem;
    }

    .board {
        max-width: 100%;
        overflow-x: auto;
    }

    .moves {
        border-collapse: collapse;
        width: 100%;
        max-width: 40rem;
    }

    .moves th,
    .moves td {
        padding: 2px 6px;
        text-align: left;
        border-bottom: 1px solid var(--color-border);
    }

    .num {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .nav,
    .grades {
        display: flex;
        gap: 0.4rem;
        align-items: center;
    }

    .error {
        color: var(--color-danger);
    }

    .note,
    .verdict {
        color: var(--color-text-muted);
    }
</style>
