<script>
    /*
     * Les réglages d'une Direction (tasks/nicomaque/fonctionnel.md §3, tasks/nicomaque/ux.md §2.1).
     *
     * Coût d'entrée bas, souris d'abord : cartes de format, un défaut de club par champ, une
     * infobulle par réglage. Sert aussi en cours de tournoi, où seuls le format d'une phase
     * ouverte et ses vies distribuées sont figés — grisés avec leur raison, pas absents — et
     * où enregistrer montre d'abord la liste des changements à confirmer.
     */
    import { t } from '../../i18n';
    import { namedConfigs } from '../../stores/directionStore';
    import { renderConfigChange, renderLockReason } from './labels.js';

    /** @typedef {import('../../stores/directionStore.js').DirectionConfig} DirectionConfig */
    /** @typedef {import('../../../wailsjs/go/models').tournoi.PhaseConfig} PhaseConfig */
    /** @typedef {import('../../../wailsjs/go/models').database.PhaseLock} PhaseLock */
    /** @typedef {import('../../../wailsjs/go/models').database.ConfigChange} ConfigChange */
    /**
     * Ce que rend l'aperçu d'une configuration : les changements, et ceux que le moteur refuse.
     *
     * @typedef {{ changes?: ConfigChange[], refusals?: ConfigChange[] }} ConfigCheck
     */

    /**
     * @type {{
     *     config?: DirectionConfig | null,
     *     directionState?: string,
     *     tournamentName?: string,
     *     onApply?: (config: DirectionConfig) => void,
     *     onDelete?: (() => void) | null,
     *     entrantCount?: number,
     *     onPreview?: ((config: DirectionConfig) => Promise<ConfigCheck | null>) | null,
     *     locks?: PhaseLock[],
     *     opened?: number,
     *     outputDir?: string,
     *     onChooseOutput?: (() => void) | null,
     *     onForgetOutput?: (() => void) | null,
     *     onOpenPage?: (() => void) | null
     * }}
     */
    let {
        config = $bindable(),
        // Pas `state` : Svelte 5 lirait la rune `$state` comme un abonnement au store `state`.
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

    /* Identifiants du moteur ; libellés via direction.format.<kind>. */
    const phaseKinds = ['swiss_lives', 'lives_bracket', 'gsl', 'bracket', 'round_robin'];

    /** @param {number} i */
    function lockOf(i) {
        return locks.find((l) => l.phase === i + 1) || null;
    }

    /** @param {number} i */
    function kindLocked(i) {
        const l = lockOf(i);
        return !!(l && l.locked);
    }

    /** @param {number} i */
    function kindTitle(i) {
        const l = lockOf(i);
        if (l && l.locked) return $t('direction.settings.kindFrozen', { reason: renderLockReason($t, l.reason) });
        return $t('direction.settings.kindHint');
    }

    /* Phase ouverte : ses vies sont distribuées, le nombre est figé ; le reste vaut pour la suite. */
    const isOpen = (/** @type {number} */ i) => i < opened;
    const openTitle = $derived($t('direction.settings.phaseOpened'));

    /** @param {number} i */
    function bracketFrozenTitle(i) {
        return $t('direction.settings.bracketFrozen', { reason: renderLockReason($t, lockOf(i)?.reason) });
    }

    /* Sans consolante, ni réconciliation ni recharge. */
    /** @param {PhaseConfig} phase @param {boolean} on */
    function setConsolation(phase, on) {
        phase.consolation = on;
        if (!on) setReconciliation(phase, false);
    }

    /** @param {PhaseConfig} phase @param {boolean} on */
    function setReconciliation(phase, on) {
        phase.reconciliation = on;
        if (!on) phase.recharge = false;
    }

    /** @param {(typeof namedConfigs)[number]} named */
    function pickNamed(named) {
        config = named.build(tournamentName || config?.name || '');
        onApply(config);
    }

    /** @param {string} kind */
    function phaseKindLabel(kind) {
        return $t(`direction.format.${kind}`);
    }

    /* Ajoutée après toutes les autres : n'interrompt pas ce qui se joue. */
    function addPhase() {
        if (!config) return;
        const last = config.phases[config.phases.length - 1];
        config.phases = [...config.phases, { kind: 'bracket', length: last?.length || 7 }];
    }

    /** @param {number} i */
    function removePhase(i) {
        if (!config) return;
        config.phases = config.phases.filter((_, k) => k !== i);
    }

    /* Pauses : rien n'est bloqué, un match qui y déborde porte un avertissement. */
    function addBreak() {
        if (!config) return;
        // Défaut : l'heure ronde qui vient de passer.
        const now = new Date();
        const start = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours());
        const end = new Date(start.getTime() + 3600000);
        config.breaks = [...(config.breaks || []), { start: start.toISOString(), end: end.toISOString() }];
    }

    /** @param {number} i */
    function removeBreak(i) {
        if (!config) return;
        config.breaks = (config.breaks || []).filter((_, k) => k !== i);
    }

    /* `datetime-local` (local, sans fuseau) ↔ instant du moteur : conversion ici seulement. */
    /** @param {string} iso */
    function toLocalInput(iso) {
        if (!iso) return '';
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return '';
        const pad = (/** @type {number} */ n) => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    /**
     * @param {number} i
     * @param {'start' | 'end'} which
     * @param {string} value
     */
    function setBreakBound(i, which, value) {
        if (!config) return;
        const d = new Date(value);
        if (Number.isNaN(d.getTime())) return;
        const next = [...(config.breaks || [])];
        next[i] = { ...next[i], [which]: d.toISOString() };
        config.breaks = next;
    }

    /* Longueurs du dernier tour vers le premier (« 15, 13, 11 » = finale, demies, quarts),
       indépendantes de la taille du tableau. */
    /** @param {PhaseConfig} phase */
    function lengthsText(phase) {
        return (phase.lengths || []).join(', ');
    }

    /**
     * @param {PhaseConfig} phase
     * @param {string} text
     */
    function setLengths(phase, text) {
        const parsed = String(text)
            .split(/[,;]/)
            .map((x) => parseInt(x.trim(), 10))
            .filter((n) => Number.isFinite(n) && n > 0);
        phase.lengths = parsed.length ? parsed : undefined;
    }

    /* Tables hors service, sautées à l'attribution (baisser le nombre retirerait la
       dernière, pas la cassée). Numéros séparés par des virgules. */
    function unavailableText() {
        return (config?.tables?.unavailable || []).join(', ');
    }

    /** @param {string} text */
    function setUnavailable(text) {
        if (!config) return;
        const parsed = [
            ...new Set(
                String(text)
                    .split(/[,;\s]+/)
                    .map((x) => parseInt(x.trim(), 10))
                    .filter((n) => Number.isFinite(n) && n > 0)
            )
        ].sort((a, b) => a - b);
        config.tables.unavailable = parsed.length ? parsed : undefined;
    }

    /* La dotation : barème par section (la consolante n'est pas le classement général),
       sans devise supposée. */
    const prizeSections = ['all', 'main', 'conso', 'last'];

    function prizes() {
        // Seul le formulaire l'appelle, et il n'existe que lorsqu'une configuration est là.
        const c = /** @type {DirectionConfig} */ (config);
        if (!c.prizes) c.prizes = {};
        if (!c.prizes.retention) c.prizes.retention = {};
        if (!c.prizes.sections) c.prizes.sections = {};
        return c.prizes;
    }

    /** @param {string} section */
    function scaleText(section) {
        const sc = config?.prizes?.sections?.[section];
        if (!sc) return '';
        return (sc.percents || sc.amounts || []).join(', ');
    }

    /* « 50, 30, 20 » ; avec % ce sont des pourcentages du distribuable, sinon des montants. */
    /**
     * @param {string} section
     * @param {string} text
     * @param {boolean} asPercent
     */
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

    /** @param {string} section */
    function isPercent(section) {
        const sc = config?.prizes?.sections?.[section];
        return !sc || !!sc.percents;
    }

    /* Le pool se recalcule avec l'effectif. */
    const pool = $derived((config?.prizes?.entry_fee || 0) * entrantCount);
    const retained = $derived.by(() => {
        const r = config?.prizes?.retention || {};
        return Math.min(pool, (pool * (r.percent || 0)) / 100 + (r.amount || 0));
    });
    const payable = $derived(Math.max(0, pool - retained));

    /** @param {number | undefined} v */
    function money(v) {
        return (v || 0).toLocaleString(undefined, { maximumFractionDigits: 0 });
    }

    /* La liste de contrôle : ce que « Enregistrer » va faire, montré avant de le faire. */
    let pending = $state(/** @type {ConfigCheck | null} */ (null));
    const blocked = $derived(!!pending && pending.refusals && pending.refusals.length > 0);
    const nothing = $derived(!!pending && (pending.changes || []).length === 0 && (pending.refusals || []).length === 0);

    async function askApply() {
        // En préparation, rien n'est encore décidé : la confirmation ne coûterait qu'un clic.
        if (!config) return;
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
        if (!config) return;
        onApply(config);
    }

    /* Exemptions du premier tour avec cet effectif : une information, pas un réglage. */
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
                    <button type="button" class="card" data-testid="direction-format-{named.id}" class:recommended={named.recommended} onclick={() => pickNamed(named)}>
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
                            <!-- Têtes de série éteintes par défaut, selon l'étude du moteur. -->
                            <label title={$t('direction.settings.seedingHint')}>
                                <input type="checkbox" checked={phase.seeding === 'rating'} onchange={(e) => (phase.seeding = e.currentTarget.checked ? 'rating' : undefined)} />
                                {$t('direction.settings.seeding')}
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
                        {#if phase.kind === 'bracket'}
                            <!-- Consolante et dépendances : grisées après le tirage, que le
                                 moteur accepte sans rien créer (PileOfCells/backgammon-tournoi#16). -->
                            <label title={kindLocked(i) ? bracketFrozenTitle(i) : $t('direction.settings.consolationHint')}>
                                <input
                                    type="checkbox"
                                    data-testid="direction-settings-consolation-{i + 1}"
                                    checked={!!phase.consolation}
                                    disabled={kindLocked(i)}
                                    onchange={(e) => setConsolation(phase, e.currentTarget.checked)}
                                />
                                {$t('direction.settings.consolation')}
                            </label>
                            {#if phase.consolation}
                                <label title={kindLocked(i) ? bracketFrozenTitle(i) : $t('direction.settings.reconciliationHint')}>
                                    <input
                                        type="checkbox"
                                        data-testid="direction-settings-reconciliation-{i + 1}"
                                        checked={!!phase.reconciliation}
                                        disabled={kindLocked(i)}
                                        onchange={(e) => setReconciliation(phase, e.currentTarget.checked)}
                                    />
                                    {$t('direction.settings.reconciliation')}
                                </label>
                                {#if phase.reconciliation}
                                    <label title={kindLocked(i) ? bracketFrozenTitle(i) : $t('direction.settings.rechargeHint')}>
                                        <input type="checkbox" data-testid="direction-settings-recharge-{i + 1}" bind:checked={phase.recharge} disabled={kindLocked(i)} />
                                        {$t('direction.settings.recharge')}
                                    </label>
                                {/if}
                            {/if}
                            {#if kindLocked(i)}
                                <span class="reason" data-testid="direction-settings-consolation-{i + 1}-frozen">{bracketFrozenTitle(i)}</span>
                            {/if}
                            {#if phase.consolation && !scaleText('conso')}
                                <p class="facts wide" data-testid="direction-settings-conso-scale-hint">{$t('direction.settings.consoScaleHint')}</p>
                            {/if}
                        {/if}
                        {#if phase.kind === 'round_robin'}
                            <label title={kindLocked(i) ? kindTitle(i) : $t('direction.settings.groupSizeHint')}>
                                {$t('direction.settings.groupSize')}
                                <input type="number" min="2" max="16" data-testid="direction-settings-group-size-{i + 1}" bind:value={phase.group_size} disabled={kindLocked(i)} />
                            </label>
                            <label title={kindLocked(i) ? kindTitle(i) : $t('direction.settings.qualifiersHint')}>
                                {$t('direction.settings.qualifiers')}
                                <input type="number" min="1" max="16" data-testid="direction-settings-qualifiers-{i + 1}" bind:value={phase.qualifiers} disabled={kindLocked(i)} />
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
            <button type="button" class="link" data-testid="direction-settings-add-phase" onclick={addPhase} title={$t('direction.settings.addPhaseHint')}>
                {$t('direction.settings.addPhase')}
            </button>
        </section>

        <section>
            <h3>{$t('direction.settings.tables')}</h3>
            <label title={$t('direction.settings.tableCountHint')}>
                {$t('direction.settings.tableCount')}
                <input type="number" data-testid="direction-settings-tables" min="0" max="200" bind:value={config.tables.count} />
            </label>
            <label title={$t('direction.settings.minPerPointHint')}>
                {$t('direction.settings.minPerPoint')}
                <input type="number" min="1" max="30" step="0.5" data-testid="direction-settings-min-per-point" placeholder="8" bind:value={config.min_per_point} />
            </label>
            <label title={$t('direction.settings.unavailableHint')}>
                {$t('direction.settings.unavailable')}
                <input
                    type="text"
                    class="lengths"
                    data-testid="direction-settings-unavailable"
                    value={unavailableText()}
                    placeholder={$t('direction.settings.unavailablePlaceholder')}
                    onchange={(e) => setUnavailable(e.currentTarget.value)}
                />
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
            <button type="button" class="link" data-testid="direction-settings-add-break" onclick={addBreak}>{$t('direction.settings.addBreak')}</button>
        </section>

        <section>
            <h3>{$t('direction.display.title')}</h3>
            <p class="facts">{$t('direction.display.hint')}</p>
            {#if outputDir}
                <p class="facts path">{outputDir}</p>
            {/if}
            <div class="actions">
                {#if onChooseOutput}
                    <button type="button" data-testid="direction-settings-output" onclick={onChooseOutput}>
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
            <button type="button" class="primary" data-testid="direction-settings-apply" onclick={askApply}>
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
            <section class="confirm" data-testid="direction-settings-changes">
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
                        <button type="button" class="primary" data-testid="direction-settings-confirm" onclick={confirmApply}>
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

    /* Une cible large. */
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

    /* `font: inherit` : un contrôle n'hérite ni taille ni famille. */
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

    /* Un format figé reste en place, grisé : absent, il se lirait comme une erreur. */
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

    /* Sous les actions, pas dans une fenêtre qui recouvrirait ce qu'elle décrit. */
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

    /* Un chemin se coupe n'importe où. */
    .path {
        word-break: break-all;
        font-family: ui-monospace, monospace;
    }

    /* Assez large pour « 15, 13, 11, 9 » sans défiler. */
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
