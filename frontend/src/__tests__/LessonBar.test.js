/**
 * LessonBar.test.js
 *
 * Mounts the reading bar of a Lesson: nothing without a Lesson, the step's
 * text and position once one is open, and the buttons call the reading
 * service. The service is mocked so no Wails backend is needed.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const service = vi.hoisted(() => ({
    nextStep: vi.fn(),
    previousStep: vi.fn(),
    closeLesson: vi.fn(),
    toggleStepDone: vi.fn(),
    openLessonEditor: vi.fn()
}));
vi.mock('../services/lessonService.js', () => service);

import LessonBar from '../components/LessonBar.svelte';
import { lessonStore, lessonStepIndexStore, lessonDoneStore } from '../stores/lessonStore.js';

const lesson = {
    id: 1,
    name: 'Prime vs prime',
    steps: [
        { id: 10, title: 'Timing', text: 'Count the pips.', collectionId: 3, positionId: 0 },
        { id: 11, title: 'Escape', text: '', collectionId: 0, positionId: 7 }
    ]
};

beforeEach(() => {
    lessonStore.set(null);
    lessonStepIndexStore.set(0);
    lessonDoneStore.set({});
    vi.clearAllMocks();
});

afterEach(() => cleanup());

describe('LessonBar', () => {
    test('renders nothing without a lesson', () => {
        const { container } = render(LessonBar);
        expect(container.querySelector('.lesson-bar')).toBeNull();
    });

    test('shows the current step and follows the index', async () => {
        lessonStore.set(lesson);
        const { container, getByText } = render(LessonBar);
        expect(getByText('Prime vs prime')).toBeTruthy();
        expect(getByText('Timing')).toBeTruthy();
        expect(getByText('Count the pips.')).toBeTruthy();

        lessonStepIndexStore.set(1);
        await tick();
        expect(getByText('Escape')).toBeTruthy();
        // A step without text draws no paragraph.
        expect(container.querySelector('.text')).toBeNull();
    });

    test('previous is disabled on the first step, next on the last', async () => {
        lessonStore.set(lesson);
        const { container } = render(LessonBar);
        const [previous, next] = container.querySelectorAll('.actions button');
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (previous).disabled).toBe(true);
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (next).disabled).toBe(false);

        lessonStepIndexStore.set(1);
        await tick();
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (previous).disabled).toBe(false);
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (next).disabled).toBe(true);
    });

    test('the buttons call the reading service', async () => {
        lessonStore.set(lesson);
        const { container } = render(LessonBar);
        const [, next, edit, close] = container.querySelectorAll('.actions button');
        await fireEvent.click(next);
        await fireEvent.click(edit);
        await fireEvent.click(close);
        expect(service.nextStep).toHaveBeenCalledTimes(1);
        expect(service.openLessonEditor).toHaveBeenCalledWith(1);
        expect(service.closeLesson).toHaveBeenCalledTimes(1);
    });

    test('"step done" reflects the recorded progress and is the only writer', async () => {
        lessonStore.set(lesson);
        lessonDoneStore.set({ 10: '2026-10-04' });
        const { container } = render(LessonBar);
        const box = container.querySelector('.done input');
        expect(/** @type {HTMLInputElement} */ (must(box)).checked).toBe(true);
        expect(must(container.querySelector('.done-count')).textContent).toContain('1');

        lessonStepIndexStore.set(1);
        await tick();
        expect(/** @type {HTMLInputElement} */ (must(box)).checked).toBe(false);
        // Moving between steps writes nothing.
        expect(service.toggleStepDone).not.toHaveBeenCalled();
        await fireEvent.click(box);
        expect(service.toggleStepDone).toHaveBeenCalledWith(11);
    });
});
