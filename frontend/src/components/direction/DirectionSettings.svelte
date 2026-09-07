<script>
    /*
     * Les réglages d'une Direction (ADR-0047 §3, tasks/nicomaque/ux.md §2.1, issue #385).
     *
     * Deux contraintes commandent le dessin de cet écran, et elles viennent du cadrage :
     * le coût d'entrée doit être bas — un directeur qui dirige un tournoi par an ne doit rien
     * lire — et la souris est première. D'où : des cartes de format cliquables plutôt qu'un
     * formulaire vide, un défaut qui tient pour un tournoi de club dans chaque champ, une
     * phrase d'infobulle par réglage, et aucun assistant en plusieurs écrans.
     *
     * L'écran sert AUSSI en cours de tournoi : à 22 h un directeur baisse la bascule pour finir
     * plus tôt, le samedi soir il ajoute une consolante qu'il n'avait pas prévue. Deux choses
     * seulement sont alors figées — le format d'une phase ouverte, et le nombre de vies qu'elle
     * a déjà distribué — et elles sont GRISÉES AVEC LEUR RAISON plutôt qu'absentes, pour que le
     * directeur sache que ce n'est pas lui qui a mal cherché.
     *
     * En cours de tournoi, enregistrer n'est pas la sauvegarde d'un formulaire mais une
     * décision : la liste de ce qui va changer s'affiche d'abord, et le directeur confirme.
     * En préparation, rien n'est encore décidé, et la confirmation ne ferait que coûter un clic.
     */
    import { t } from '../../i18n';
    import { namedConfigs } from '../../stores/directionStore';
    import { renderConfigChange, renderLockReason } from './labels.js';

    let {
        config = $bindable(),
        // Le nom compte : une prop nommée `state` ferait lire la rune `$state` de ce fichier
        // comme un abonnement au store `state` — Svelte 5 résout `$x` en abonnement dès qu'un
        // `x` est en portée. L'écran des Réglages a planté ainsi pendant une journée, invisible
        // aux tests unitaires, jusqu'à la première spec de bout en bout (#390).
        directionState = 'draft',
        tournamentName = '',
        onApply = () => {},
        onDelete = null,
        entrantCount = 0,
        onPreview = null,
        locks = [],
        opened = 0,
        outputDir = '',
        onChooseOutput = null,
        onForgetOutput = null,
        onOpenPage = null
    } = $props();

    const isDraft = $derived(directionState === 'draft');

    /* Les cinq formats de phase du moteur. Ce sont des IDENTIFIANTS : leur nom lisible passe
       par direction.format.<kind>, comme partout ailleurs. */
    const phaseKinds = ['swiss_lives', 'lives_bracket', 'gsl', 'bracket', 'round_robin'];

    function lockOf(i) {
        return locks.find((l) => l.phase === i + 1) || null;
    }

    function kindLocked(i) {
        const l = lockOf(i);
        return !!(l && l.locked);
    }

    function kindTitle(i) {
        const l = lockOf(i);
        if (l && l.locked) return $t('direction.settings.kindFrozen', { reason: renderLockReason($t, l.reason) });
        return $t('direction.settings.kindHint');
    }

    /* Une phase OUVERTE a déjà distribué ses vies : changer le nombre ne rattraperait pas les
       joueurs entrés. Le reste — longueurs, bascule, finale — porte sur les matchs à venir. */
    const isOpen = (i) => i < opened;
    const openTitle = $derived($t('direction.settings.phaseOpened'));

    function pickNamed(named) {
        config = named.build(tournamentName || config?.name || '');
        onApply(config);
    }

    function phaseKindLabel(kind) {
        return $t(`direction.format.${kind}`);
    }

    /* Ajouter une phase, c'est l'ajouter APRÈS toutes les autres : une consolante décidée le
       samedi soir n'interrompt pas ce qui se joue. */
    function addPhase() {
        const last = config.phases[config.phases.length - 1];
        config.phases = [...config.phases, { kind: 'bracket', length: last?.length || 7 }];
    }

    function removePhase(i) {
        config.phases = config.phases.filter((_, k) => k !== i);
    }

    /* Les pauses de la journée. Rien n'est bloqué : un match qui finirait dedans porte un
       avertissement, et le directeur décide. */
    function addBreak() {
        // L'heure ronde qui vient de passer : un défaut qu'un directeur corrige d'un geste.
        const now = new Date();
        const start = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours());
        const end = new Date(start.getTime() + 3600000);
        config.breaks = [...(config.breaks || []), { start: start.toISOString(), end: end.toISOString() }];
    }

    function removeBreak(i) {
        config.breaks = (config.breaks || []).filter((_, k) => k !== i);
    }

    /* Un `datetime-local` parle l'heure locale sans fuseau ; le moteur ne connaît que des
       instants. La conversion se fait ici, aux deux bouts, et nulle part ailleurs. */
    function toLocalInput(iso) {
        if (!iso) return '';
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return '';
        const pad = (n) => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    function setBreakBound(i, which, value) {
        const d = new Date(value);
        if (Number.isNaN(d.getTime())) return;
        const next = [...(config.breaks || [])];
        next[i] = { ...next[i], [which]: d.toISOString() };
        config.breaks = next;
    }

    /* Les longueurs tour par tour d'un tableau, DU DERNIER TOUR VERS LE PREMIER : « 15, 13, 11 »
       veut dire finale en 15, demies en 13, quarts en 11. C'est l'ordre dans lequel un
       organisateur annonce son tournoi, et il ne dépend pas de la taille du tableau — la même
       liste sert un tableau de 16 et un de 64, où elle allonge les quatre derniers tours. */
    function lengthsText(phase) {
        return (phase.lengths || []).join(', ');
    }

    function setLengths(phase, text) {
        const parsed = String(text)
            .split(/[,;]/)
            .map((x) => parseInt(x.trim(), 10))
            .filter((n) => Number.isFinite(n) && n > 0);
        phase.lengths = parsed.length ? parsed : undefined;
    }

    /*
     * La dotation (issue #393).
     *
     * Un droit d'entrée, une retenue pour le club, et un barème PAR SECTION : le vainqueur de
     * la consolante n'est pas le finaliste du tournoi, et le payer sur le classement général
     * dirait le contraire.
     *
     * Aucune devise n'est supposée : les montants sont des nombres, écrits avec les séparateurs
     * de la langue de l'utilisateur, et le directeur sait dans quelle monnaie il encaisse.
     */
    const prizeSections = ['all', 'main', 'conso', 'last'];

    function prizes() {
        if (!config.prizes) config.prizes = {};
        if (!config.prizes.retention) config.prizes.retention = {};
        if (!config.prizes.sections) config.prizes.sections = {};
        return config.prizes;
    }

    function scaleText(section) {
        const sc = config.prizes?.sections?.[section];
        if (!sc) return '';
        return (sc.percents || sc.amounts || []).join(', ');
    }

    /* Un barème se saisit en une ligne : « 50, 30, 20 ». Le signe % dit lequel des deux champs
       du moteur est rempli — des pourcentages du distribuable, ou des montants fixes. */
    function setScale(section, text, asPercent) {
        const list = String(text)
            .split(/[,;]/)
            .map((x) => parseFloat(x.trim().replace(',', '.')))
            .filter((n) => Number.isFinite(n) && n > 0);
        const p = prizes();
        if (!list.length) {
            const next = { ...p.sections };
            delete next[section];
            p.sections = next;
            return;
        }
        p.sections = { ...p.sections, [section]: asPercent ? { percents: list } : { amounts: list } };
    }

    function isPercent(section) {
        const sc = config.prizes?.sections?.[section];
        return !sc || !!sc.percents;
    }

    /* Le pool, à côté de l'effectif : il se recalcule à chaque inscription et chaque retrait,
       sans que personne ait à le demander. */
    const pool = $derived((config?.prizes?.entry_fee || 0) * entrantCount);
    const retained = $derived.by(() => {
        const r = config?.prizes?.retention || {};
        return Math.min(pool, (pool * (r.percent || 0)) / 100 + (r.amount || 0));
    });
    const payable = $derived(Math.max(0, pool - retained));

    function money(v) {
        return (v || 0).toLocaleString(undefined, { maximumFractionDigits: 0 });
    }

    /* La liste de contrôle : ce que « Enregistrer » va faire, montré avant de le faire. */
    let pending = $state(null);
    const blocked = $derived(!!pending && pending.refusals && pending.refusals.length > 0);
    const nothing = $derived(!!pending && (pending.changes || []).length === 0 && (pending.refusals || []).length === 0);

    async function askApply() {
        // En préparation, rien n'est encore décidé : la confirmation ne coûterait qu'un clic.
        if (isDraft || !onPreview) {
            onApply(config);
            return;
        }
        const p = await onPreview(config);
        if (!p) {
            onApply(config);
            return;
        }
        pending = p;
    }

    function confirmApply() {
        pending = null;
        onApply(config);
    }

    /* Le nombre d'exemptions qu'un tableau devrait donner au premier tour avec cet effectif :
       une information que le directeur regarde pour choisir son format, pas un réglage. */
    const byesAtFirstRound = $derived.by(() => {
        if (!entrantCount) return 0;
        let size = 1;
        while (size < entrantCount) size *= 2;
        return size - entrantCount;
    });
</script>

<div class="settings">
    {#if isDraft}
        <section class="formats">
            <h3>{$t('direction.settings.format')}</h3>
            <div class="cards">
                {#each namedConfigs as named (named.id)}
                    <button type="button" class="card" class:recommended={named.recommended} onclick={() => pickNamed(named)}>
                        <span class="card-name">{$t(`direction.named.${named.id}`)}</span>
                        <span class="card-hint">{$t(`direction.named.${named.id}Hint`)}</span>
                        {#if named.recommended}
                            <span class="badge">{$t('direction.settings.recommended')}</span>
                        {/if}
                    </button>
                {/each}
            </div>
        </section>
    {/if}

    {#if config}
        <section>
            <h3>{$t('direction.settings.phases')}</h3>
            <ul class="phases">
                {#each config.phases || [] as phase, i (i)}
                    <li>
                        <span class="phase-index">{i + 1}</span>
                        {#if kindLocked(i)}
                            <span class="phase-kind frozen" title={kindTitle(i)}>
                                {phaseKindLabel(phase.kind)}
                                <span class="reason">{renderLockReason($t, lockOf(i)?.reason)}</span>
                            </span>
                        {:else}
                            <select class="phase-kind" bind:value={phase.kind} title={kindTitle(i)}>
                                {#each phaseKinds as kind (kind)}
                                    <option value={kind}>{phaseKindLabel(kind)}</option>
                                {/each}
                            </select>
                        {/if}
                        <label title={$t('direction.settings.lengthHint')}>
                            {$t('direction.settings.length')}
                            <input type="number" min="1" max="99" bind:value={phase.length} />
                        </label>
                        {#if phase.kind === 'swiss_lives'}
                            <label title={isOpen(i) ? openTitle : $t('direction.settings.livesHint')}>
                                {$t('direction.settings.lives')}
                                <input type="number" min="1" max="5" bind:value={phase.lives} disabled={isOpen(i)} />
                            </label>
                            <label title={$t('direction.settings.targetHint')}>
                                {$t('direction.settings.target')}
                                <input type="number" min="0" step="8" bind:value={phase.target} />
                            </label>
                            <label title={$t('direction.settings.batchHint')}>
                                {$t('direction.settings.batch')}
                                <input type="number" min="0" max="120" bind:value={phase.batch_minutes} />
                            </label>
                            <label title={$t('direction.settings.lengthLateHint')}>
                                {$t('direction.settings.lengthLate')}
                                <input type="number" min="0" max="99" bind:value={phase.length_late} />
                            </label>
                            <label title={$t('direction.settings.lateThresholdHint')}>
                                {$t('direction.settings.lateThreshold')}
                                <input type="number" min="0" max="99" bind:value={phase.late_threshold} />
                            </label>
                        {/if}
                        {#if phase.kind === 'bracket' || phase.kind === 'lives_bracket'}
                            <label title={$t('direction.settings.finalLengthHint')}>
                                {$t('direction.settings.finalLength')}
                                <input type="number" min="0" max="99" bind:value={phase.final_length} />
                            </label>
                            <label class="wide" title={$t('direction.settings.lengthsHint')}>
                                {$t('direction.settings.lengths')}
                                <input
                                    type="text"
                                    class="lengths"
                                    value={lengthsText(phase)}
                                    placeholder={$t('direction.settings.lengthsPlaceholder')}
                                    onchange={(e) => setLengths(phase, e.currentTarget.value)}
                                />
                            </label>
                        {/if}
                        {#if !isOpen(i) && (config.phases || []).length > 1}
                            <button type="button" class="link" onclick={() => removePhase(i)} title={$t('direction.settings.removePhaseHint')}>
                                {$t('direction.settings.removePhase')}
                            </button>
                        {/if}
                    </li>
                {/each}
            </ul>
            <button type="button" class="link" onclick={addPhase} title={$t('direction.settings.addPhaseHint')}>
                {$t('direction.settings.addPhase')}
            </button>
        </section>

        <section>
            <h3>{$t('direction.settings.tables')}</h3>
            <label title={$t('direction.settings.tableCountHint')}>
                {$t('direction.settings.tableCount')}
                <input type="number" min="0" max="200" bind:value={config.tables.count} />
            </label>
            {#if entrantCount > 0}
                <p class="facts">
                    {$t('direction.settings.entrants', { n: entrantCount })}
                    {#if byesAtFirstRound > 0}
                        &middot;
                        {$t('direction.settings.byes', { n: byesAtFirstRound })}
                    {/if}
                </p>
            {/if}
        </section>

        <section>
            <h3>{$t('direction.settings.prizes')}</h3>
            <div class="row">
                <label title={$t('direction.settings.entryFeeHint')}>
                    {$t('direction.settings.entryFee')}
                    <input type="number" min="0" step="1" value={config.prizes?.entry_fee || 0} onchange={(e) => (prizes().entry_fee = parseFloat(e.currentTarget.value) || 0)} />
                </label>
                <label title={$t('direction.settings.retentionPercentHint')}>
                    {$t('direction.settings.retentionPercent')}
                    <input
                        type="number"
                        min="0"
                        max="100"
                        value={config.prizes?.retention?.percent || 0}
                        onchange={(e) => (prizes().retention = { ...prizes().retention, percent: parseFloat(e.currentTarget.value) || 0 })}
                    />
                </label>
                <label title={$t('direction.settings.retentionAmountHint')}>
                    {$t('direction.settings.retentionAmount')}
                    <input
                        type="number"
                        min="0"
                        value={config.prizes?.retention?.amount || 0}
                        onchange={(e) => (prizes().retention = { ...prizes().retention, amount: parseFloat(e.currentTarget.value) || 0 })}
                    />
                </label>
            </div>
            {#if pool > 0}
                <p class="facts">
                    {$t('direction.settings.pool', { pool: money(pool), retained: money(retained), payable: money(payable) })}
                </p>
            {/if}
            <ul class="scales">
                {#each prizeSections as section (section)}
                    <li>
                        <span class="scale-name">{$t(`direction.settings.scale_${section}`)}</span>
                        <input
                            type="text"
                            class="scale"
                            value={scaleText(section)}
                            placeholder={$t('direction.settings.scalePlaceholder')}
                            title={$t('direction.settings.scaleHint')}
                            onchange={(e) => setScale(section, e.currentTarget.value, isPercent(section))}
                        />
                        <label title={$t('direction.settings.percentHint')}>
                            <input type="checkbox" checked={isPercent(section)} onchange={(e) => setScale(section, scaleText(section), e.currentTarget.checked)} />
                            {$t('direction.settings.percent')}
                        </label>
                    </li>
                {/each}
            </ul>
        </section>

        <section>
            <h3>{$t('direction.settings.breaks')}</h3>
            <p class="facts">{$t('direction.settings.breaksHint')}</p>
            <ul class="breaks">
                {#each config.breaks || [] as pause, i (i)}
                    <li>
                        <label>
                            {$t('direction.settings.breakStart')}
                            <input type="datetime-local" value={toLocalInput(pause.start)} onchange={(e) => setBreakBound(i, 'start', e.currentTarget.value)} />
                        </label>
                        <label>
                            {$t('direction.settings.breakEnd')}
                            <input type="datetime-local" value={toLocalInput(pause.end)} onchange={(e) => setBreakBound(i, 'end', e.currentTarget.value)} />
                        </label>
                        <button type="button" class="link" onclick={() => removeBreak(i)}>
                            {$t('direction.settings.removeBreak')}
                        </button>
                    </li>
                {/each}
            </ul>
            <button type="button" class="link" onclick={addBreak}>{$t('direction.settings.addBreak')}</button>
        </section>

        <section>
            <h3>{$t('direction.display.title')}</h3>
            <p class="facts">{$t('direction.display.hint')}</p>
            {#if outputDir}
                <p class="facts path">{outputDir}</p>
            {/if}
            <div class="actions">
                {#if onChooseOutput}
                    <button type="button" onclick={onChooseOutput}>
                        {outputDir ? $t('direction.display.changeFolder') : $t('direction.display.chooseFolder')}
                    </button>
                {/if}
                {#if outputDir && onOpenPage}
                    <button type="button" onclick={onOpenPage}>{$t('direction.display.open')}</button>
                {/if}
                {#if outputDir && onForgetOutput}
                    <button type="button" class="link" onclick={onForgetOutput}>{$t('direction.display.forget')}</button>
                {/if}
            </div>
        </section>

        <div class="actions">
            <button type="button" class="primary" onclick={askApply}>
                {$t('direction.settings.apply')}
            </button>
            {#if onDelete}
                <button type="button" class="danger" onclick={onDelete}>
                    {$t('direction.settings.delete')}
                </button>
            {/if}
        </div>
        {#if !isDraft}
            <p class="frozen-note">{$t('direction.settings.frozen')}</p>
        {/if}

        {#if pending}
            <section class="confirm">
                <h3>{$t('direction.change.title')}</h3>
                {#if nothing}
                    <p class="facts">{$t('direction.change.nothing')}</p>
                {:else}
                    <ul class="changes">
                        {#each pending.changes || [] as change, i (change.code + i)}
                            <li>{renderConfigChange($t, change)}</li>
                        {/each}
                        {#each pending.refusals || [] as refusal, i (refusal.code + i)}
                            <li class="refused">
                                {renderConfigChange($t, refusal)}
                                &nbsp;&middot;&nbsp;{renderLockReason($t, refusal.reason)}
                            </li>
                        {/each}
                    </ul>
                {/if}
                {#if blocked}
                    <p class="facts">{$t('direction.change.blocked')}</p>
                {/if}
                <div class="actions">
                    {#if !nothing && !blocked}
                        <button type="button" class="primary" onclick={confirmApply}>
                            {$t('direction.change.confirm')}
                        </button>
                    {/if}
                    <button type="button" onclick={() => (pending = null)}>
                        {nothing || blocked ? $t('direction.change.close') : $t('direction.change.cancel')}
                    </button>
                </div>
            </section>
        {/if}
    {/if}
</div>

<style>
    .settings {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding: 0.75rem;
        overflow-y: auto;
    }

    h3 {
        font-size: var(--font-size-base);
        font-weight: 600;
        margin: 0 0 0.4rem;
        color: var(--color-text);
    }

    .cards {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
        gap: 0.5rem;
    }

    /* Une carte est une cible large : le premier geste d'un directeur occasionnel ne doit pas
       demander de viser. */
    .card {
        display: flex;
        flex-direction: column;
        gap: 0.2rem;
        text-align: left;
        padding: 0.6rem 0.7rem;
        min-height: 64px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .card:hover,
    .card:focus-visible {
        border-color: var(--color-primary);
    }

    .card.recommended {
        border-color: var(--color-primary);
    }

    .card-name {
        font-weight: 600;
    }

    .card-hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .badge {
        align-self: flex-start;
        font-size: var(--font-size-small);
        color: var(--color-primary);
    }

    .phases {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
    }

    .phases li {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.5rem;
    }

    .phase-index {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 1.4rem;
        height: 1.4rem;
        border-radius: 50%;
        background: var(--color-border);
        font-size: var(--font-size-small);
    }

    .phase-kind {
        font-weight: 600;
        min-width: 9rem;
    }

    label {
        display: inline-flex;
        align-items: center;
        gap: 0.3rem;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* Un contrôle n'hérite ni de la taille ni de la famille : sans `font: inherit` il
       retomberait dans la police du navigateur (invariant « une seule échelle de type »). */
    input {
        width: 4.5rem;
        padding: 0.15rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
    }

    input:disabled {
        opacity: 0.55;
        cursor: not-allowed;
    }

    .facts,
    .frozen-note {
        margin: 0.3rem 0 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .actions {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    button.primary,
    button.danger {
        padding: 0.35rem 0.8rem;
        border-radius: var(--radius);
        border: 1px solid var(--color-border);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    button.primary {
        border-color: var(--color-primary);
    }

    button.danger {
        border-color: var(--color-danger);
    }

    select.phase-kind {
        min-width: 9rem;
        padding: 0.15rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
    }

    /* Un format figé reste À SA PLACE, grisé avec sa raison : un contrôle absent se lit comme
       une erreur de recherche, un contrôle grisé se lit comme une règle. */
    .phase-kind.frozen {
        display: inline-flex;
        align-items: baseline;
        gap: 0.35rem;
        opacity: 0.75;
    }

    .reason {
        font-weight: 400;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    button.link {
        border: none;
        background: none;
        padding: 0.2rem 0.1rem;
        color: var(--color-primary);
        cursor: pointer;
        align-self: flex-start;
    }

    .breaks {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
    }

    .breaks li {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.5rem;
    }

    .breaks input {
        width: auto;
    }

    /* La liste de contrôle : elle s'ouvre sous les actions, là où le regard vient de cliquer,
       plutôt que dans une fenêtre qui recouvrirait ce qu'elle décrit. */
    .confirm {
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        padding: 0.6rem 0.7rem;
        background: var(--color-surface-alt);
    }

    .changes {
        list-style: none;
        margin: 0 0 0.5rem;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.2rem;
    }

    .refused {
        color: var(--color-danger);
    }

    /* Un chemin se lit d'un coup d'œil et se coupe où il veut : c'est une adresse, pas une
       phrase. */
    .path {
        word-break: break-all;
        font-family: ui-monospace, monospace;
    }

    /* Une liste de longueurs est plus large qu'un nombre : « 15, 13, 11, 9 » doit se lire d'un
       coup d'œil, sans défilement dans le champ. */
    input.lengths {
        width: 9rem;
    }

    .row {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        flex-wrap: wrap;
    }

    .scales {
        list-style: none;
        margin: 0.3rem 0 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .scales li {
        display: flex;
        align-items: center;
        gap: var(--space-1);
    }

    .scale-name {
        min-width: 9rem;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    input.scale {
        width: 9rem;
    }

    input[type='checkbox'] {
        width: auto;
    }
</style>
