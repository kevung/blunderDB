/**
 * Le panneau Calculs et sa puce : trois groupes, les actions de chaque ligne, la puce unique
 * de la barre d'état. Monté pour de bon : un composant qui plante au montage échoue ici.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

const mocks = vi.hoisted(() => ({
    handlers: /** @type {Record<string, (p: any) => void>} */ ({}),
    ListJobs: vi.fn(() => Promise.resolve([])),
    CancelJob: vi.fn(() => Promise.resolve()),
    PauseJob: vi.fn(() => Promise.resolve()),
    ResumeJob: vi.fn(() => Promise.resolve()),
    MoveJob: vi.fn(() => Promise.resolve()),
    RemoveJob: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    ListJobs: mocks.ListJobs,
    CancelJob: mocks.CancelJob,
    PauseJob: mocks.PauseJob,
    ResumeJob: mocks.ResumeJob,
    MoveJob: mocks.MoveJob,
    RemoveJob: mocks.RemoveJob
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((name, cb) => {
        mocks.handlers[name] = cb;
        return () => {};
    })
}));

import { jobsStore, startJobsFeed, runningJobs, waitingJobs, endedJobs, jobsProgress } from '../stores/jobsStore.js';
import JobsModal from '../components/JobsModal.svelte';

const job = (over) => ({ id: 1, kind: 'rollout', label: 'Position 3', library: 1, priority: 'requested', state: 'running', done: 2, total: 8, position: 0, pausable: true, cancelable: true, ...over });

const JOBS = [
    job({ id: 1 }),
    job({ id: 2, kind: 'gammonnet-missing', label: 'x', state: 'waiting', done: 0, total: 10, position: 1, pausable: false }),
    job({ id: 3, kind: 'gammonnet-missing', label: 'y', state: 'waiting', done: 0, total: 5, position: 2, pausable: false }),
    job({ id: 4, state: 'done', done: 8, total: 8, pausable: false, cancelable: false })
];

beforeEach(() => {
    vi.clearAllMocks();
    jobsStore.set(JOBS);
});
afterEach(() => {
    cleanup();
    jobsStore.set([]);
});

describe('jobsStore', () => {
    test('les événements jobs:* alimentent le registre', async () => {
        jobsStore.set([]);
        mocks.ListJobs.mockResolvedValueOnce([job({ id: 9 })]);
        const stop = startJobsFeed();
        await waitFor(() => expect(get(jobsStore).map((j) => j.id)).toEqual([9]));
        mocks.handlers['jobs:changed'](job({ id: 9, done: 5 }));
        mocks.handlers['jobs:changed'](job({ id: 10 }));
        expect(get(jobsStore).find((j) => j.id === 9).done).toBe(5);
        mocks.handlers['jobs:removed']({ id: 9 });
        expect(get(jobsStore).map((j) => j.id)).toEqual([10]);
        stop();
    });

    test('les groupes et la progression globale', () => {
        expect(get(runningJobs).map((j) => j.id)).toEqual([1]);
        expect(get(waitingJobs).map((j) => j.id)).toEqual([2, 3]);
        expect(get(endedJobs).map((j) => j.id)).toEqual([4]);
        // Les tâches terminées ne comptent pas : (2 + 0 + 0) / (8 + 10 + 5).
        expect(get(jobsProgress)).toBeCloseTo(2 / 23);
    });
});

describe('JobsModal', () => {
    test('monte les trois groupes avec leurs lignes', async () => {
        const { findByTestId, getAllByTestId } = render(JobsModal, { props: { visible: true, onClose: vi.fn() } });
        await findByTestId('jobs-running');
        expect(getAllByTestId('job-row')).toHaveLength(4);
        await findByTestId('jobs-waiting');
        await findByTestId('jobs-ended');
    });

    test('pastille de bibliothèque et compteur fait / total', async () => {
        const { findAllByTestId } = render(JobsModal, { props: { visible: true, onClose: vi.fn() } });
        const [chip] = await findAllByTestId('job-library');
        expect(chip.getAttribute('style')).toContain('--group-color-1');
        const [count] = await findAllByTestId('job-count');
        expect(count.textContent.trim()).toBe('2 / 8');
    });

    test('une tâche en cours se met en pause et s’annule', async () => {
        const { findByTestId } = render(JobsModal, { props: { visible: true, onClose: vi.fn() } });
        await fireEvent.click(await findByTestId('job-pause'));
        expect(mocks.PauseJob).toHaveBeenCalledWith(1);
        await fireEvent.click(await findByTestId('job-cancel'));
        expect(mocks.CancelJob).toHaveBeenCalledWith(1);
    });

    test('une tâche en attente monte, descend et se retire', async () => {
        const { findAllByTestId } = render(JobsModal, { props: { visible: true, onClose: vi.fn() } });
        const ups = await findAllByTestId('job-up');
        const downs = await findAllByTestId('job-down');
        expect(ups[0].disabled).toBe(true);
        expect(downs[1].disabled).toBe(true);
        await fireEvent.click(ups[1]);
        expect(mocks.MoveJob).toHaveBeenCalledWith(3, 0);
        await fireEvent.click(downs[0]);
        expect(mocks.MoveJob).toHaveBeenCalledWith(2, 1);
        const removes = await findAllByTestId('job-remove');
        await fireEvent.click(removes[0]);
        expect(mocks.RemoveJob).toHaveBeenCalledWith(2);
    });

    test('Tout en pause arrête ce qui court et ce qui attend', async () => {
        const { findByTestId } = render(JobsModal, { props: { visible: true, onClose: vi.fn() } });
        await fireEvent.click(await findByTestId('jobs-pause-all'));
        await waitFor(() => expect(mocks.PauseJob).toHaveBeenCalledTimes(3));
    });

    test('le compteur ouvre la bibliothèque de la tâche et ferme le panneau', async () => {
        const onClose = vi.fn();
        const { findAllByTestId } = render(JobsModal, { props: { visible: true, onClose } });
        await fireEvent.click((await findAllByTestId('job-count'))[0]);
        await waitFor(() => expect(onClose).toHaveBeenCalled());
    });

    test('sans tâche, une ligne vide', async () => {
        jobsStore.set([]);
        const { findByTestId } = render(JobsModal, { props: { visible: true, onClose: vi.fn() } });
        await findByTestId('jobs-empty');
    });
});
