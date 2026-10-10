import { get } from 'svelte/store';
import { GetLesson, ListLessons, GetCollectionByID, LessonDoneSteps, SetLessonStepDone } from '../../wailsjs/go/database/Database.js';
import { lessonStore, lessonStepIndexStore, lessonDoneStore, lessonEditorTargetStore } from '../stores/lessonStore.js';
import { openModal, MODAL } from '../stores/uiStore.js';
import { collectionSource, positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { handleOpenCollection } from './modeMachine.js';
import { showImportedPosition } from './importService.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// Lire une Leçon : chaque étape amène sur le plateau la collection ou la position qu'elle
// montre, par les gestes qui existent déjà (ouvrir une collection, aller à une position). La
// barre apporte l'ordre et le texte, pas de nouvelle façon de parcourir.

/** `:le` sans argument : la liste des leçons dans la barre d'état. */
export async function listLessons() {
    try {
        const lessons = (await ListLessons()) || [];
        if (lessons.length === 0) {
            setStatusBarMessage(tMsg('lesson.none'));
            return;
        }
        const names = lessons.map((l) => `${l.id} ${l.name} (${l.stepCount})`).join(' · ');
        setStatusBarMessage(tMsg('lesson.list', { lessons: names }));
    } catch (error) {
        logger.error('could not list lessons:', error);
        setStatusBarMessage(tMsg('lesson.failed'));
    }
}

/** Ouvre la leçon id à sa première étape. */
export async function openLesson(id) {
    try {
        const lesson = await GetLesson(id);
        if (!lesson || !lesson.steps || lesson.steps.length === 0) {
            setStatusBarMessage(tMsg('lesson.empty'));
            return false;
        }
        lessonStore.set(lesson);
        lessonStepIndexStore.set(0);
        await loadDoneSteps(lesson.id);
        await showStep();
        return true;
    } catch (error) {
        logger.error('could not open the lesson:', error);
        setStatusBarMessage(tMsg('lesson.notFound', { id }));
        return false;
    }
}

export async function nextStep() {
    const lesson = get(lessonStore);
    const index = get(lessonStepIndexStore);
    if (!lesson || index + 1 >= lesson.steps.length) return;
    lessonStepIndexStore.set(index + 1);
    await showStep();
}

export async function previousStep() {
    const index = get(lessonStepIndexStore);
    if (index <= 0) return;
    lessonStepIndexStore.set(index - 1);
    await showStep();
}

export function closeLesson() {
    lessonStore.set(null);
    lessonStepIndexStore.set(0);
    lessonDoneStore.set({});
}

/** Lit la progression déjà enregistrée ; une base sans progression n'en a simplement aucune. */
async function loadDoneSteps(/** @type {number} */ lessonId) {
    try {
        lessonDoneStore.set((await LessonDoneSteps(lessonId)) || {});
    } catch (error) {
        logger.error('could not read the lesson progress:', error);
        lessonDoneStore.set({});
    }
}

/**
 * Le geste « étape faite » (ADR-0069) : le seul qui écrive la progression, et seulement dans
 * la base ouverte. Un second clic retire la marque.
 */
export async function toggleStepDone(stepId) {
    const lesson = get(lessonStore);
    if (!lesson) return;
    const done = !!get(lessonDoneStore)[stepId];
    try {
        await SetLessonStepDone(stepId, !done);
        await loadDoneSteps(lesson.id);
    } catch (error) {
        logger.error('could not mark the step:', error);
        setStatusBarMessage(tMsg('lesson.failed'));
    }
}

/** Amène sur le plateau ce que montre l'étape courante ; une étape de texte seul ne bouge rien. */
async function showStep() {
    const lesson = get(lessonStore);
    if (!lesson) return;
    const step = lesson.steps[get(lessonStepIndexStore)];
    if (!step) return;
    try {
        if (step.collectionId) {
            const collection = await GetCollectionByID(step.collectionId);
            await handleOpenCollection(collection, collectionSource(step.collectionId));
            if (step.positionId) {
                const index = await positionsStore.findIndex(step.positionId);
                if (index >= 0) currentPositionIndexStore.set(index);
            }
        } else if (step.positionId) {
            await showImportedPosition(step.positionId);
        }
    } catch (error) {
        logger.error('could not show the lesson step:', error);
        setStatusBarMessage(tMsg('lesson.stepMissing'));
    }
}

/** Ouvre l'éditeur de leçons, sur la leçon id quand elle est donnée. */
export function openLessonEditor(id = 0) {
    lessonEditorTargetStore.set(Number(id) || 0);
    openModal(MODAL.LESSON_EDITOR);
}

/** Après une modification dans l'éditeur, la leçon en lecture reprend son contenu à jour. */
export async function refreshOpenLesson() {
    const lesson = get(lessonStore);
    if (!lesson) return;
    try {
        const fresh = await GetLesson(lesson.id);
        if (!fresh || !fresh.steps || fresh.steps.length === 0) {
            closeLesson();
            return;
        }
        lessonStore.set(fresh);
        lessonStepIndexStore.set(Math.min(get(lessonStepIndexStore), fresh.steps.length - 1));
        await loadDoneSteps(fresh.id);
    } catch {
        closeLesson();
    }
}
