/**
 * AnalysisPanel.focusOnMount.test.js
 *
 * The analysis panel focuses itself right after it mounts. A match review
 * switches to it only once the move's analysis has loaded, so on a slow
 * machine the mount lands AFTER the user pressed Space: the command line was
 * already open, its input closes itself on blur, and the panel's focus made
 * it vanish (the CI flake of match-navigation / match-exit-keeps-position).
 * The panel still takes the focus when nobody is typing.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({ LoadAnalysis: vi.fn(() => Promise.resolve(null)) }));

const { matchContextStore } = await import('../stores/positionStore.js');
const AnalysisPanel = (await import('../components/AnalysisPanel.svelte')).default;

/** The panel's own deferred focus runs on the next macrotask. */
const nextMacrotask = () => new Promise((resolve) => setTimeout(resolve, 0));

describe('AnalysisPanel: focus on mount', () => {
    afterEach(() => {
        cleanup();
        document.body.innerHTML = '';
        matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
    });

    test('takes the keyboard when nothing else holds it', async () => {
        matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
        render(AnalysisPanel, { props: { onClose: vi.fn() } });
        await nextMacrotask();

        expect(document.activeElement?.id).toBe('analysisPanel');
    });

    test('does not take the caret from the command line opened just before', async () => {
        const commandLine = document.createElement('input');
        commandLine.className = 'command-input';
        document.body.appendChild(commandLine);
        commandLine.focus();

        render(AnalysisPanel, { props: { onClose: vi.fn() } });
        await nextMacrotask();

        expect(document.activeElement).toBe(commandLine);
    });
});
