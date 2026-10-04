<script>
    // L'éditeur de Leçons (ADR-0066) : la même chose que `blunderdb lesson create / add-step /
    // edit-step / reorder`, au bureau. Une étape est un texte, et peut montrer une collection,
    // une position, les deux ou aucune ; la position courante du plateau s'y attache d'un clic.
    import Modal from './Modal.svelte';
    import { get } from 'svelte/store';
    import {
        ListLessons,
        GetLesson,
        CreateLesson,
        UpdateLesson,
        DeleteLesson,
        AddLessonStep,
        UpdateLessonStep,
        RemoveLessonStep,
        ReorderLessonSteps,
        GetAllCollections
    } from '../../wailsjs/go/database/Database.js';
    import { lessonEditorTargetStore } from '../stores/lessonStore.js';
    import { positionStore } from '../stores/positionStore.js';
    import { openLesson, refreshOpenLesson } from '../services/lessonService.js';
    import { confirmAction } from '../services/confirmService.js';
    import { logger } from '../utils/logger.js';
    import { t, tMsg } from '../i18n';

    let { visible = false, onClose } = $props();

    /** @type {{id: number, name: string, stepCount: number}[]} */
    let lessons = $state([]);
    /** @type {{id: number, name: string}[]} */
    let collections = $state([]);
    let selectedId = $state(0);
    let name = $state('');
    let description = $state('');
    /** @type {{id: number, title: string, text: string, collectionId: number, positionId: number, dirty: boolean}[]} */
    let steps = $state([]);
    let newName = $state('');
    let error = $state('');

    $effect(() => {
        if (visible) void init();
    });

    async function init() {
        error = '';
        await loadLessons();
        try {
            collections = (await GetAllCollections()) || [];
        } catch (e) {
            logger.error('could not list the collections:', e);
            collections = [];
        }
        const target = get(lessonEditorTargetStore);
        if (target && lessons.some((l) => l.id === target)) await select(target);
        else if (!lessons.some((l) => l.id === selectedId)) await select(lessons[0]?.id ?? 0);
    }

    async function loadLessons() {
        try {
            lessons = (await ListLessons()) || [];
        } catch (e) {
            fail(e);
        }
    }

    function fail(e) {
        logger.error('lesson editor:', e);
        error = tMsg('lessonEditor.failed', { error: String(e?.message ?? e) });
    }

    async function select(id) {
        selectedId = id;
        steps = [];
        if (!id) {
            name = '';
            description = '';
            return;
        }
        try {
            const lesson = await GetLesson(id);
            name = lesson.name;
            description = lesson.description;
            steps = (lesson.steps || []).map((s) => ({ ...s, dirty: false }));
        } catch (e) {
            fail(e);
        }
    }

    /** Toute écriture repasse par ici : relire la liste, la leçon, et la leçon en lecture. */
    async function after(write) {
        error = '';
        try {
            await write();
        } catch (e) {
            fail(e);
        }
        await loadLessons();
        await select(lessons.some((l) => l.id === selectedId) ? selectedId : (lessons[0]?.id ?? 0));
        await refreshOpenLesson();
    }

    function create() {
        const n = newName.trim();
        if (!n) return;
        return after(async () => {
            selectedId = await CreateLesson(n, '');
            newName = '';
        });
    }

    const saveLesson = () => after(() => UpdateLesson(selectedId, name.trim(), description));

    async function remove() {
        const lesson = lessons.find((l) => l.id === selectedId);
        if (!lesson) return;
        if (!(await confirmAction(tMsg('lessonEditor.confirmDelete', { name: lesson.name })))) return;
        await after(() => DeleteLesson(selectedId));
    }

    const addStep = () => after(() => AddLessonStep(selectedId, tMsg('lessonEditor.newStepTitle', { n: steps.length + 1 }), '', 0, 0));

    const saveStep = (s) => after(() => UpdateLessonStep(s.id, s.title, s.text, Number(s.collectionId) || 0, Number(s.positionId) || 0));

    async function removeStep(s) {
        if (!(await confirmAction(tMsg('lessonEditor.confirmRemoveStep', { title: s.title || s.id })))) return;
        await after(() => RemoveLessonStep(s.id));
    }

    function move(index, delta) {
        const ids = steps.map((s) => s.id);
        const j = index + delta;
        if (j < 0 || j >= ids.length) return;
        [ids[index], ids[j]] = [ids[j], ids[index]];
        return after(() => ReorderLessonSteps(selectedId, ids));
    }

    function attachCurrentPosition(s) {
        const id = get(positionStore)?.id;
        if (!id) {
            error = tMsg('lessonEditor.noCurrentPosition');
            return;
        }
        s.positionId = id;
        s.dirty = true;
    }

    function chooseCollection(s, value) {
        s.collectionId = Number(value) || 0;
        s.dirty = true;
    }

    async function read() {
        const id = selectedId;
        onClose?.();
        await openLesson(id);
    }
</script>

<Modal open={visible} onclose={onClose} size="large" label={$t('lessonEditor.title')}>
    <h2 class="modal-title">{$t('lessonEditor.title')}</h2>
    <div class="editor">
        <aside class="lessons">
            <ul>
                {#each lessons as lesson (lesson.id)}
                    <li>
                        <button type="button" class:selected={lesson.id === selectedId} onclick={() => select(lesson.id)}>
                            {lesson.name} <span class="muted">({lesson.stepCount})</span>
                        </button>
                    </li>
                {/each}
            </ul>
            {#if lessons.length === 0}<p class="muted">{$t('lesson.none')}</p>{/if}
            <form
                class="new"
                onsubmit={(e) => {
                    e.preventDefault();
                    create();
                }}
            >
                <input type="text" bind:value={newName} placeholder={$t('lessonEditor.newLessonName')} aria-label={$t('lessonEditor.newLessonName')} />
                <button type="submit" disabled={!newName.trim()}>{$t('lessonEditor.create')}</button>
            </form>
        </aside>

        {#if selectedId}
            <section class="lesson">
                <label>{$t('lessonEditor.name')} <input type="text" bind:value={name} /></label>
                <label>{$t('lessonEditor.description')} <textarea rows="2" bind:value={description}></textarea></label>
                <div class="row">
                    <button type="button" onclick={saveLesson} disabled={!name.trim()}>{$t('lessonEditor.save')}</button>
                    <button type="button" onclick={read} disabled={steps.length === 0}>{$t('lessonEditor.read')}</button>
                    <button type="button" class="danger" onclick={remove}>{$t('lessonEditor.delete')}</button>
                </div>

                <ol class="steps">
                    {#each steps as step, i (step.id)}
                        <li class="step">
                            <div class="row">
                                <input type="text" bind:value={step.title} oninput={() => (step.dirty = true)} aria-label={$t('lessonEditor.stepTitle')} placeholder={$t('lessonEditor.stepTitle')} />
                                <button type="button" onclick={() => move(i, -1)} disabled={i === 0} title={$t('lessonEditor.up')} aria-label={$t('lessonEditor.up')}>↑</button>
                                <button type="button" onclick={() => move(i, 1)} disabled={i + 1 === steps.length} title={$t('lessonEditor.down')} aria-label={$t('lessonEditor.down')}>↓</button>
                            </div>
                            <textarea rows="3" bind:value={step.text} oninput={() => (step.dirty = true)} aria-label={$t('lessonEditor.stepText')} placeholder={$t('lessonEditor.stepText')}></textarea>
                            <div class="row">
                                <select value={step.collectionId || 0} onchange={(e) => chooseCollection(step, e.currentTarget.value)} aria-label={$t('lessonEditor.collection')}>
                                    <option value={0}>{$t('lessonEditor.noCollection')}</option>
                                    {#each collections as c (c.id)}
                                        <option value={c.id}>{c.name}</option>
                                    {/each}
                                </select>
                                <span class="muted">{step.positionId ? $t('lessonEditor.position', { id: step.positionId }) : ''}</span>
                                <button type="button" onclick={() => attachCurrentPosition(step)}>{$t('lessonEditor.useCurrentPosition')}</button>
                                {#if step.positionId}
                                    <button
                                        type="button"
                                        onclick={() => {
                                            step.positionId = 0;
                                            step.dirty = true;
                                        }}>{$t('lessonEditor.detachPosition')}</button
                                    >
                                {/if}
                            </div>
                            <div class="row">
                                <button type="button" onclick={() => saveStep(step)} disabled={!step.dirty}>{$t('lessonEditor.saveStep')}</button>
                                <button type="button" class="danger" onclick={() => removeStep(step)}>{$t('lessonEditor.removeStep')}</button>
                            </div>
                        </li>
                    {/each}
                </ol>
                <button type="button" onclick={addStep}>{$t('lessonEditor.addStep')}</button>
            </section>
        {/if}
    </div>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
</Modal>

<style>
    .editor {
        display: flex;
        gap: 1em;
        min-width: 40em;
        max-height: 70vh;
    }

    .lessons {
        flex: 0 0 14em;
        overflow-y: auto;
    }

    .lessons ul {
        list-style: none;
        margin: 0;
        padding: 0;
    }

    .lessons li button {
        width: 100%;
        text-align: left;
        border: none;
        background: none;
        padding: 0.2em 0.3em;
        cursor: pointer;
    }

    .lessons li button.selected {
        background: var(--color-surface-alt);
        font-weight: 600;
    }

    .new {
        display: flex;
        gap: 0.3em;
        margin-top: 0.6em;
    }

    .new input {
        flex: 1;
        min-width: 0;
    }

    .lesson {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.4em;
        overflow-y: auto;
    }

    .lesson label {
        display: flex;
        flex-direction: column;
        gap: 0.15em;
    }

    .row {
        display: flex;
        gap: 0.4em;
        align-items: center;
        flex-wrap: wrap;
    }

    .row input[type='text'] {
        flex: 1;
    }

    .steps {
        margin: 0.4em 0;
        padding-left: 1.5em;
        display: flex;
        flex-direction: column;
        gap: 0.6em;
    }

    .step {
        display: flex;
        flex-direction: column;
        gap: 0.3em;
        border-bottom: 1px solid var(--color-border);
        padding-bottom: 0.5em;
    }

    .muted {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .danger {
        color: var(--color-danger);
    }

    .error {
        color: var(--color-danger);
        margin: 0.5em 0 0;
    }
</style>
