<script>
    // Les erreurs récurrentes : les erreurs du filtre groupées par plan de jeu
    // et par thème, la plus coûteuse d'abord. Le coût est celui du PR, sur les
    // mêmes décisions comptées : la somme des groupes ne dépasse jamais le PR.
    import { loadPositionsFromSelection } from '../../services/positionLoader.js';
    import { quizOnIds, quizOnWorstGroups, deckFromIds, collectionFromIds } from '../../services/recurringStudy.js';
    import { t } from '../../i18n/index.js';

    /** @type {{ data: { NumDecisions: number, Groups: Array<object> }|null, loading?: boolean, error?: string|null }} */
    let { data = null, loading = false, error = null } = $props();

    let groups = $derived(data?.Groups ?? []);
    // Les erreurs qu'aucune règle ne nomme restent hors du classement : la
    // phrase d'explication ne parle qu'à partir de 60 mp, au-dessus du seuil
    // Erreur, et ce reste coifferait sinon un classement qui ne dit rien.
    let unthemed = $derived(data?.Unthemed ?? []);

    function gameTypeLabel(gameType) {
        const key = `stats.gameType_${gameType}`;
        const label = $t(key);
        return label === key ? gameType : label;
    }

    // Les thèmes de videau sont les cases d'erreur de la matrice de direction :
    // mêmes libellés qu'elle, pour qu'un même fait ne porte pas deux noms.
    const CUBE_LABELS = {
        offer_missed: 'stats.cubeOfferMissed',
        offer_premature: 'stats.cubeOfferPremature',
        answer_wrong_pass: 'stats.cubeAnswerWrongPass',
        answer_wrong_take: 'stats.cubeAnswerWrongTake'
    };

    function themeLabel(g) {
        const key = g.Kind === 'cube' && CUBE_LABELS[g.Theme] ? CUBE_LABELS[g.Theme] : `stats.recurringTheme_${g.Theme}`;
        const label = $t(key);
        return label === key ? g.Theme : label;
    }

    function studyName(g) {
        return $t('stats.recurringStudyName', { name: `${gameTypeLabel(g.GameType)} · ${$t(`stats.recurringKind_${g.Kind}`)} · ${themeLabel(g)}` });
    }

    function open(g) {
        if (!g.PositionIDs?.length) return;
        loadPositionsFromSelection(g.PositionIDs);
    }
</script>

<section class="chart-section recurring">
    <h3 class="section-title">{$t('stats.recurringTitle')}</h3>
    <p class="hint">{$t('stats.recurringHint')}</p>
    {#if loading}
        <p class="empty-subsection">{$t('stats.loading')}</p>
    {:else if error}
        <p class="empty-subsection">{error}</p>
    {:else if groups.length === 0 && unthemed.length === 0}
        <p class="empty-subsection">{$t('stats.recurringEmpty')}</p>
    {:else}
        {#if groups.length > 0}
            <button type="button" class="study-btn worst" data-testid="recurring-quiz-worst" onclick={() => quizOnWorstGroups()}>{$t('stats.recurringQuizWorst')}</button>
            <table>
                <thead>
                    <tr>
                        <th>{$t('stats.gameType')}</th>
                        <th>{$t('stats.recurringTheme')}</th>
                        <th class="num">{$t('stats.decisions')}</th>
                        <th class="num">{$t('stats.recurringCost')}</th>
                        <th>{$t('stats.recurringActions')}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each groups as g (g.GameType + '/' + g.Kind + '/' + g.Theme)}
                        <tr class="group-row" title={`${(g.SumErrorMP / 1000).toFixed(3)}`}>
                            <td>{gameTypeLabel(g.GameType)}</td>
                            <td>
                                <button type="button" class="group-link" onclick={() => open(g)}>
                                    {$t(`stats.recurringKind_${g.Kind}`)} · {themeLabel(g)}
                                </button>
                            </td>
                            <td class="num">{g.Count}</td>
                            <td class="num">{g.PRCost.toFixed(2)}</td>
                            <td class="actions">
                                <button type="button" class="study-btn" onclick={() => quizOnIds(g.PositionIDs ?? [])}>{$t('stats.recurringQuiz')}</button>
                                <button type="button" class="study-btn" onclick={() => deckFromIds(studyName(g), g.PositionIDs ?? [])}>{$t('stats.recurringDeck')}</button>
                                <button type="button" class="study-btn" onclick={() => collectionFromIds(studyName(g), g.PositionIDs ?? [])}>{$t('stats.recurringCollection')}</button>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        {/if}
        {#if unthemed.length > 0}
            <ul class="unthemed">
                {#each unthemed as g (g.GameType)}
                    <li>
                        <button type="button" class="group-link" onclick={() => open(g)}>
                            {gameTypeLabel(g.GameType)} — {$t('stats.recurringUnthemed', { n: g.Count, cost: g.PRCost.toFixed(2) })}
                        </button>
                    </li>
                {/each}
            </ul>
        {/if}
    {/if}
</section>

<style>
    .chart-section {
        padding: 12px 16px 0;
    }

    .section-title {
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        margin: 0 0 8px;
    }

    .hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin: 0 0 8px;
    }

    .empty-subsection {
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
        text-align: center;
        padding: 12px 16px;
        margin: 0;
        font-style: italic;
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    th,
    td {
        text-align: left;
        padding: 3px 6px;
        border-bottom: 1px solid var(--color-border);
    }

    .num {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .group-row:hover {
        background: var(--color-surface-alt);
    }

    .unthemed {
        list-style: none;
        margin: 8px 0 0;
        padding: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .actions {
        white-space: nowrap;
    }

    .study-btn {
        font-size: var(--font-size-small);
        padding: 1px 8px;
        margin-right: 4px;
        cursor: pointer;
    }

    .worst {
        margin: 0 0 8px;
    }

    .group-link {
        background: none;
        border: none;
        padding: 0;
        color: var(--color-text);
        cursor: pointer;
        text-align: left;
        text-decoration: underline dotted;
    }
</style>
