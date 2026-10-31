import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

const gui = vi.hoisted(() => ({ SaveBoardImageDialog: vi.fn(), SaveBoardSVG: vi.fn() }));
vi.mock('../../wailsjs/go/gui/App.js', () => gui);

import VisionBenchExport from '../components/VisionBenchExport.svelte';
import { benchFile } from '../services/visionBenchExport.js';

const EVENTS = [
    { type: 'roll', mediaTime: 1.5, state: 'confirmed', payload: { x: 1 } },
    { type: 'move', mediaTime: 4.25, state: 'tentative', payload: {} }
];

afterEach(() => {
    cleanup();
    vi.unstubAllEnvs();
    vi.clearAllMocks();
});

describe('bench export', () => {
    test('the file holds the events in order, in seconds, without payload', () => {
        expect(benchFile(EVENTS, 'cal', 'v.mp4')).toEqual({
            events: [
                { type: 'roll', mediaTime: 1.5, state: 'confirmed' },
                { type: 'move', mediaTime: 4.25, state: 'tentative' }
            ],
            calibration: 'cal',
            source: 'v.mp4'
        });
    });

    test('the action writes that file where the dialog says', async () => {
        gui.SaveBoardImageDialog.mockResolvedValue('/tmp/e.json');
        const { getByRole } = render(VisionBenchExport, { props: { events: EVENTS, calibration: 'cal', source: 'v.mp4' } });
        await fireEvent.click(getByRole('button'));
        await vi.waitFor(() => expect(gui.SaveBoardSVG).toHaveBeenCalled());
        expect(gui.SaveBoardImageDialog).toHaveBeenCalledWith('json', 'events.json');
        const [path, text] = gui.SaveBoardSVG.mock.calls[0];
        expect(path).toBe('/tmp/e.json');
        expect(JSON.parse(text)).toEqual(benchFile(EVENTS, 'cal', 'v.mp4'));
    });

    test('nothing is written when the dialog is cancelled', async () => {
        gui.SaveBoardImageDialog.mockResolvedValue('');
        const { getByRole } = render(VisionBenchExport, { props: { events: EVENTS, calibration: 'c', source: 's' } });
        await fireEvent.click(getByRole('button'));
        await vi.waitFor(() => expect(gui.SaveBoardImageDialog).toHaveBeenCalled());
        expect(gui.SaveBoardSVG).not.toHaveBeenCalled();
    });

    test('absent when DEV is false', () => {
        vi.stubEnv('DEV', false);
        const { queryByRole } = render(VisionBenchExport, { props: { events: EVENTS, calibration: 'c', source: 's' } });
        expect(queryByRole('button')).toBeNull();
    });

    test('absent while no event was received', () => {
        const { queryByRole } = render(VisionBenchExport, { props: { events: [], calibration: 'c', source: 's' } });
        expect(queryByRole('button')).toBeNull();
    });
});
