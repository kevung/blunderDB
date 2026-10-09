/**
 * The help's search field takes the focus on a click and keeps it while typing.
 */
import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';

vi.mock('../../wailsjs/go/database/Database', () => ({ GetDatabaseVersion: vi.fn(() => Promise.resolve('2.0.0')) }));

import HelpModal from '../components/HelpModal.svelte';

afterEach(cleanup);

test('a click in the search field focuses it and typing keeps the focus', async () => {
    const user = userEvent.setup();
    render(HelpModal, { visible: true, onClose: vi.fn() });
    await vi.waitFor(() => expect(screen.queryByTestId('help-loading')).toBeNull());

    const input = screen.getByTestId('help-search');
    await user.click(input);
    expect(document.activeElement).toBe(input);

    await user.keyboard('posi');
    expect(input.value).toBe('posi');
    expect(document.activeElement).toBe(input);
    await new Promise((r) => setTimeout(r, 300));
    expect(document.activeElement).toBe(input);
});
