/**
 * LessonEditorModal.test.js
 *
 * Mounts the lesson editor against mocked bindings: it lists the lessons,
 * opens the targeted one, creates a lesson, appends a step, attaches the
 * board's position beside the collection and reorders the steps.
 */

import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

const api = vi.hoisted(() => ({
    ListLessons: vi.fn(),
    GetLesson: vi.fn(),
    CreateLesson: vi.fn(),
    UpdateLesson: vi.fn(),
    DeleteLesson: vi.fn(),
    AddLessonStep: vi.fn(),
    UpdateLessonStep: vi.fn(),
    RemoveLessonStep: vi.fn(),
    ReorderLessonSteps: vi.fn(),
    GetAllCollections: vi.fn()
}));
vi.mock('../../wailsjs/go/database/Database.js', () => api);
vi.mock('../services/lessonService.js', () => ({ openLesson: vi.fn(), refreshOpenLesson: vi.fn() }));
vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn(async () => true) }));

import LessonEditorModal from '../components/LessonEditorModal.svelte';
import { lessonEditorTargetStore } from '../stores/lessonStore.js';
import { positionStore } from '../stores/positionStore.js';

const lesson = {
    id: 2,
    name: 'Primes',
    description: '',
    steps: [
        { id: 20, title: 'A', text: 'one', collectionId: 5, positionId: 0 },
        { id: 21, title: 'B', text: 'two', collectionId: 0, positionId: 0 }
    ]
};

beforeEach(() => {
    vi.clearAllMocks();
    api.ListLessons.mockResolvedValue([
        { id: 1, name: 'Intro', stepCount: 0 },
        { id: 2, name: 'Primes', stepCount: 2 }
    ]);
    api.GetLesson.mockImplementation(async (id) => (id === 2 ? lesson : { id: 1, name: 'Intro', description: '', steps: [] }));
    api.GetAllCollections.mockResolvedValue([{ id: 5, name: 'Prime collection' }]);
    api.CreateLesson.mockResolvedValue(3);
    lessonEditorTargetStore.set(2);
    positionStore.set({ id: 77 });
});

afterEach(() => cleanup());

describe('LessonEditorModal', () => {
    test('opens on the targeted lesson and shows its steps', async () => {
        const { container } = render(LessonEditorModal, { visible: true });
        await waitFor(() => expect(container.querySelectorAll('.step').length).toBe(2));
        expect(api.GetLesson).toHaveBeenCalledWith(2);
    });

    test('creates a lesson and appends a step', async () => {
        const { container, getByText } = render(LessonEditorModal, { visible: true });
        await waitFor(() => expect(container.querySelectorAll('.step').length).toBe(2));
        const input = container.querySelector('.new input');
        await fireEvent.input(input, { target: { value: 'Nouvelle' } });
        await fireEvent.submit(container.querySelector('.new'));
        await waitFor(() => expect(api.CreateLesson).toHaveBeenCalledWith('Nouvelle', ''));

        await fireEvent.click(getByText('Add a step'));
        await waitFor(() => expect(api.AddLessonStep).toHaveBeenCalled());
    });

    test('attaching the board position keeps the collection, and saving sends both', async () => {
        const { container } = render(LessonEditorModal, { visible: true });
        await waitFor(() => expect(container.querySelectorAll('.step').length).toBe(2));
        const first = container.querySelectorAll('.step')[0];
        const buttons = [...first.querySelectorAll('button')];
        await fireEvent.click(buttons.find((b) => b.textContent.includes('Current position')));
        await fireEvent.click(buttons.find((b) => b.textContent.includes('Save the step')));
        await waitFor(() => expect(api.UpdateLessonStep).toHaveBeenCalledWith(20, 'A', 'one', 5, 77));
    });

    test('moving a step down reorders the lesson', async () => {
        const { container } = render(LessonEditorModal, { visible: true });
        await waitFor(() => expect(container.querySelectorAll('.step').length).toBe(2));
        const down = container.querySelectorAll('.step')[0].querySelectorAll('.row button')[1];
        await fireEvent.click(down);
        await waitFor(() => expect(api.ReorderLessonSteps).toHaveBeenCalledWith(2, [21, 20]));
    });
});
