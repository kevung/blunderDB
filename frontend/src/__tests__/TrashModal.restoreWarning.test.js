/**
 * TrashModal.restoreWarning.test.js
 *
 * A restore that succeeded without the match's Direction slot says so in the
 * status bar; a plain restore keeps the plain message.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

const ENTRY = { id: 3, kind: 'match', label: 'Alice – Bob', deletedAt: '2026-10-09T10:00:00Z' };

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => ({
    ...(await importOriginal()),
    ListTrash: vi.fn(() => Promise.resolve([ENTRY])),
    RestoreFromTrash: vi.fn(),
    DiscardFromTrash: vi.fn(() => Promise.resolve()),
    EmptyTrash: vi.fn(() => Promise.resolve(0))
}));

vi.mock('../services/databaseService.js', async (importOriginal) => ({
    ...(await importOriginal()),
    setStatusBarMessage: vi.fn()
}));

vi.mock('../services/positionService.js', async (importOriginal) => ({
    ...(await importOriginal()),
    reloadAllPositions: vi.fn(() => Promise.resolve())
}));

import TrashModal from '../components/TrashModal.svelte';
import { RestoreFromTrash } from '../../wailsjs/go/database/Database.js';
import { setStatusBarMessage } from '../services/databaseService.js';
import { get } from 'svelte/store';
import { t, tMsg } from '../i18n';

async function restoreOnce(result) {
    RestoreFromTrash.mockResolvedValueOnce(result);
    const { findByText } = render(TrashModal, { props: { visible: true, onClose: () => {} } });
    await findByText(ENTRY.label);
    const button = await findByText(get(t)('trash.restore'));
    await fireEvent.click(button);
    await waitFor(() => expect(setStatusBarMessage).toHaveBeenCalled());
    return setStatusBarMessage.mock.calls.at(-1)[0];
}

describe('TrashModal restore warnings', () => {
    beforeEach(() => vi.clearAllMocks());
    afterEach(() => cleanup());

    test('a slot taken since is reported', async () => {
        const msg = await restoreOnce({ id: 7, warnings: [{ code: 'direction_slot_taken', message: 'x' }] });
        expect(msg).toEqual(tMsg('trash.restoredSlotTaken', { what: ENTRY.label }));
    });

    test('a plain restore says restored', async () => {
        const msg = await restoreOnce({ id: 7 });
        expect(msg).toEqual(tMsg('trash.restored', { what: ENTRY.label }));
    });
});
