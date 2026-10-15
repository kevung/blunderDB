/**
 * ManagePlayersModal.test.js
 *
 * The merge dialog was the one role="dialog" without the focus trap the other
 * modals use (now Modal.svelte's): Tab walked out of it into the match panel
 * behind. These tests pin the trap — focus lands inside on mount, Tab wraps at
 * the edges — and the dialog's contract with the binding: the names ticked in
 * the list and the canonical name typed in reach MergePlayers as-is, then the
 * parent is told to refresh and close.
 *
 * The selection is a SvelteSet mutated in place. It used to be re-assigned to a
 * copy on every click, which the template — tracking the original instance —
 * never saw: the merge button stayed disabled whatever was ticked. The payload
 * test fails against that version.
 */
import { must } from './helpers/must.js';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const MergePlayers = vi.fn(() => Promise.resolve());
const GetAllPlayerNames = vi.hoisted(() => vi.fn());
const ListAliases = vi.hoisted(() => vi.fn());
const RemoveAlias = vi.hoisted(() => vi.fn());
const MainPlayer = vi.hoisted(() => vi.fn());
const PlanMainPlayer = vi.hoisted(() => vi.fn());
const SetMainPlayer = vi.hoisted(() => vi.fn());
const ClearMainPlayer = vi.hoisted(() => vi.fn());
const PlayerMatchIDs = vi.hoisted(() => vi.fn());
const openMatchListInPanel = vi.hoisted(() => vi.fn());
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: (/** @type {any[]} */ ...args) => GetAllPlayerNames(...args),
    MergePlayers: (/** @type {any[]} */ ...args) => MergePlayers(...args),
    ListAliases: (/** @type {any[]} */ ...args) => ListAliases(...args),
    RemoveAlias: (/** @type {any[]} */ ...args) => RemoveAlias(...args),
    MainPlayer: (/** @type {any[]} */ ...args) => MainPlayer(...args),
    PlanMainPlayer: (/** @type {any[]} */ ...args) => PlanMainPlayer(...args),
    SetMainPlayer: (/** @type {any[]} */ ...args) => SetMainPlayer(...args),
    ClearMainPlayer: (/** @type {any[]} */ ...args) => ClearMainPlayer(...args),
    PlayerMatchIDs: (/** @type {any[]} */ ...args) => PlayerMatchIDs(...args)
}));
vi.mock('../services/positionLoader.js', () => ({
    openMatchListInPanel: (/** @type {any[]} */ ...args) => openMatchListInPanel(...args)
}));

import ManagePlayersModal from '../components/ManagePlayersModal.svelte';

/** @type {ReturnType<typeof vi.fn>} */
let onClose;
/** @type {ReturnType<typeof vi.fn>} */
let onMerged;
/** @type {HTMLButtonElement} */
let outside;

async function mount() {
    const result = render(ManagePlayersModal, { props: { onClose, onMerged } });
    // GetAllPlayerNames resolves on a microtask; two ticks let the list render.
    await tick();
    await tick();
    return result;
}

async function settle() {
    for (let i = 0; i < 6; i++) await tick();
}

function tab(/** @type {any} */ target, shiftKey = false) {
    const event = new KeyboardEvent('keydown', { key: 'Tab', shiftKey, bubbles: true, cancelable: true });
    target.dispatchEvent(event);
    return event;
}

beforeEach(() => {
    onClose = vi.fn();
    onMerged = vi.fn();
    MergePlayers.mockClear();
    GetAllPlayerNames.mockReset();
    GetAllPlayerNames.mockResolvedValue([
        { Name: 'Alice', Count: 3 },
        { Name: 'alice', Count: 1 },
        { Name: 'Bob', Count: 2 }
    ]);
    ListAliases.mockReset();
    ListAliases.mockResolvedValue([]);
    RemoveAlias.mockReset();
    RemoveAlias.mockResolvedValue(true);
    MainPlayer.mockReset();
    MainPlayer.mockResolvedValue('');
    PlanMainPlayer.mockReset();
    PlanMainPlayer.mockResolvedValue({ name: '', swap: [], refused: [] });
    SetMainPlayer.mockReset();
    ClearMainPlayer.mockReset();
    ClearMainPlayer.mockResolvedValue(undefined);
    PlayerMatchIDs.mockReset();
    openMatchListInPanel.mockReset();
    // Something focusable outside the dialog — where Tab used to escape to.
    outside = document.createElement('button');
    outside.id = 'outside';
    document.body.appendChild(outside);
    outside.focus();
});

afterEach(() => {
    cleanup();
    outside.remove();
});

describe('ManagePlayersModal — focus', () => {
    test('mounts as a dialog and takes the focus away from the page behind', async () => {
        const { container } = await mount();

        const dialog = container.querySelector('[role="dialog"]');
        expect(dialog).not.toBeNull();
        expect(must(dialog).getAttribute('aria-modal')).toBe('true');
        expect(must(dialog).contains(document.activeElement)).toBe(true);
    });

    test('Tab is trapped: it wraps from the last control to the first and back', async () => {
        const { container } = await mount();
        const dialog = container.querySelector('[role="dialog"]');

        // The filter field opens the dialog; the close cross, last in the DOM, ends it.
        const first = must(dialog).querySelector('.filter-input');
        const last = must(dialog).querySelector('.modal-close');
        expect(document.activeElement).toBe(first);

        /** @type {HTMLElement} */ (must(last)).focus();
        const forward = tab(last);
        expect(forward.defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(first);

        const backward = tab(first, true);
        expect(backward.defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(last);
        expect(must(dialog).contains(document.activeElement)).toBe(true);
    });

    test('closing gives the focus back to what had it before', async () => {
        const { unmount } = await mount();
        expect(document.activeElement).not.toBe(outside);
        unmount();
        expect(document.activeElement).toBe(outside);
    });
});

describe('ManagePlayersModal — merging', () => {
    test('the ticked names and the canonical name reach the binding, then the parent refreshes and the window stays open', async () => {
        const { container } = await mount();

        const rows = [...container.querySelectorAll('.player-row')];
        expect(rows.map((r) => must(r.querySelector('.player-name')).textContent)).toEqual(['Alice', 'alice', 'Bob']);

        const mergeButton = container.querySelector('.btn-merge');
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (must(mergeButton)).disabled).toBe(true);

        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        await tick();

        // The first ticked name is proposed as the one to keep.
        const canonical = container.querySelector('#canonical-input');
        expect(/** @type {HTMLInputElement} */ (must(canonical)).value).toBe('Alice');
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (must(mergeButton)).disabled).toBe(false);
        expect(rows[0].classList.contains('selected')).toBe(true);
        expect(rows[1].classList.contains('selected')).toBe(true);
        expect(rows[2].classList.contains('selected')).toBe(false);

        await fireEvent.click(must(mergeButton));
        await tick();
        await tick();

        expect(MergePlayers).toHaveBeenCalledTimes(1);
        expect(MergePlayers).toHaveBeenCalledWith(['Alice', 'alice'], 'Alice');
        expect(onMerged).toHaveBeenCalledTimes(1);
        expect(onClose).not.toHaveBeenCalled();
    });

    test('two merges in a row without closing: list reloaded, selection cleared, feedback shown, focus back on the filter', async () => {
        GetAllPlayerNames.mockResolvedValueOnce([
            { Name: 'Alice', Count: 3 },
            { Name: 'alice', Count: 1 },
            { Name: 'Bob', Count: 2 },
            { Name: 'bob', Count: 1 }
        ]);
        GetAllPlayerNames.mockResolvedValueOnce([
            { Name: 'Alice', Count: 4 },
            { Name: 'Bob', Count: 2 },
            { Name: 'bob', Count: 1 }
        ]);
        GetAllPlayerNames.mockResolvedValueOnce([
            { Name: 'Alice', Count: 4 },
            { Name: 'Bob', Count: 3 }
        ]);
        const { container } = await mount();
        const names = () => [...container.querySelectorAll('.player-name')].map((n) => n.textContent);
        const merge = () => /** @type {HTMLButtonElement} */ (must(container.querySelector('.btn-merge')));

        let rows = [...container.querySelectorAll('.player-row')];
        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        await fireEvent.click(merge());
        await settle();

        expect(MergePlayers).toHaveBeenLastCalledWith(['Alice', 'alice'], 'Alice');
        expect(names()).toEqual(['Alice', 'Bob', 'bob']);
        expect(container.querySelectorAll('.player-row.selected').length).toBe(0);
        expect(/** @type {HTMLInputElement} */ (must(container.querySelector('#canonical-input'))).value).toBe('');
        expect(must(container.querySelector('.merge-done')).textContent).toBe('Alice, alice → Alice');
        expect(merge().disabled).toBe(true);
        expect(document.activeElement).toBe(container.querySelector('.filter-input'));
        expect(onClose).not.toHaveBeenCalled();

        rows = [...container.querySelectorAll('.player-row')];
        await fireEvent.click(rows[1]);
        await fireEvent.click(rows[2]);
        await fireEvent.click(merge());
        await settle();

        expect(MergePlayers).toHaveBeenCalledTimes(2);
        expect(MergePlayers).toHaveBeenLastCalledWith(['Bob', 'bob'], 'Bob');
        expect(names()).toEqual(['Alice', 'Bob']);
        expect(onMerged).toHaveBeenCalledTimes(2);
        expect(onClose).not.toHaveBeenCalled();
    });

    test('the Close button closes the window', async () => {
        const { container } = await mount();
        await fireEvent.click(must(container.querySelector('.btn-close')));
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('unticking a name takes it out of the payload', async () => {
        const { container } = await mount();
        const rows = [...container.querySelectorAll('.player-row')];

        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        await fireEvent.click(rows[2]);
        await fireEvent.click(rows[1]); // untick 'alice'
        await tick();

        await fireEvent.click(must(container.querySelector('.btn-merge')));
        await tick();
        await tick();

        expect(MergePlayers).toHaveBeenCalledWith(['Alice', 'Bob'], 'Alice');
    });

    test('a canonical name typed by the user wins over the proposed one', async () => {
        const { container } = await mount();
        const rows = [...container.querySelectorAll('.player-row')];

        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        const canonical = container.querySelector('#canonical-input');
        await fireEvent.input(must(canonical), { target: { value: 'Alice Martin' } });
        await tick();

        await fireEvent.click(must(container.querySelector('.btn-merge')));
        await tick();
        await tick();

        expect(MergePlayers).toHaveBeenCalledWith(['Alice', 'alice'], 'Alice Martin');
    });

    test('Escape closes without merging', async () => {
        const { container } = await mount();
        const dialog = container.querySelector('[role="dialog"]');

        await fireEvent.keyDown(must(dialog), { key: 'Escape' });

        expect(onClose).toHaveBeenCalledTimes(1);
        expect(MergePlayers).not.toHaveBeenCalled();
        expect(onMerged).not.toHaveBeenCalled();
    });
});

describe('ManagePlayersModal — main player, aliases, counts', () => {
    test('the star shows the swap count, which opens those matches; confirming swaps and refreshes', async () => {
        PlanMainPlayer.mockResolvedValueOnce({ name: 'Bob', swap: [4, 7], refused: [9] });
        SetMainPlayer.mockResolvedValue({ name: 'Bob', swap: [4, 7], refused: [9] });
        const { container } = await mount();
        const bob = /** @type {HTMLElement} */ ([...container.querySelectorAll('.player-row')].find((r) => r.querySelector('.player-name')?.textContent === 'Bob'));
        await fireEvent.click(must(bob.querySelector('[data-testid="main-player-star"]')));
        await settle();

        expect(PlanMainPlayer).toHaveBeenCalledWith('Bob');
        expect(SetMainPlayer).not.toHaveBeenCalled();
        expect(bob.classList.contains('selected')).toBe(false);
        expect(must(container.querySelector('[data-testid="main-player-swap-count"]')).textContent).toBe('2');
        expect(must(container.querySelector('[data-testid="main-player-refused-count"]')).textContent).toBe('1');
        // The star is full as soon as it is clicked, while the band asks for confirmation.
        expect(must(bob.querySelector('[data-testid="main-player-star"]')).textContent).toBe('★');
        expect(container.querySelector('[data-testid="main-player-clear"]')).toBeNull();

        MainPlayer.mockResolvedValue('Bob');
        await fireEvent.click(must(container.querySelector('[data-testid="main-player-apply"]')));
        await settle();
        expect(SetMainPlayer).toHaveBeenCalledWith('Bob');
        expect(onMerged).toHaveBeenCalled();
        expect(container.querySelector('[data-testid="main-player-confirm"]')).toBeNull();
        const stars = [...container.querySelectorAll('[data-testid="main-player-star"]')].map((b) => b.textContent);
        // The Main player heads the list.
        expect(stars).toEqual(['★', '☆', '☆']);
        expect(must(container.querySelector('.player-name')).textContent).toBe('Bob');
    });

    test('the swap count opens those matches in the panel', async () => {
        PlanMainPlayer.mockResolvedValue({ name: 'Bob', swap: [4, 7], refused: [] });
        const { container } = await mount();
        const stars = container.querySelectorAll('[data-testid="main-player-star"]');
        await fireEvent.click(must(stars[2]));
        await settle();
        expect(container.querySelector('[data-testid="main-player-refused-count"]')).toBeNull();
        await fireEvent.click(must(container.querySelector('[data-testid="main-player-swap-count"]')));
        expect(openMatchListInPanel).toHaveBeenCalledWith([4, 7]);
        expect(onClose).toHaveBeenCalled();
    });

    test('the star of the main player with nothing left to swap only forgets them', async () => {
        MainPlayer.mockResolvedValue('Alice');
        const { container } = await mount();
        const star = must(container.querySelector('[data-testid="main-player-star"]'));
        expect(star.textContent).toBe('★');
        await fireEvent.click(star);
        await settle();
        expect(ClearMainPlayer).toHaveBeenCalledTimes(1);
        // Asked once, at load, what is left to swap: nothing.
        expect(PlanMainPlayer).toHaveBeenCalledTimes(1);
        expect(container.querySelector('[data-testid="main-player-confirm"]')).toBeNull();
        expect(SetMainPlayer).not.toHaveBeenCalled();
        expect(star.textContent).toBe('☆');
    });

    test('with matches left to swap, the band comes back with a Remove button; the full star reapplies', async () => {
        MainPlayer.mockResolvedValue('Bob');
        PlanMainPlayer.mockResolvedValue({ name: 'Bob', swap: [5], refused: [] });
        SetMainPlayer.mockResolvedValue({ name: 'Bob', swap: [5], refused: [] });
        const { container } = await mount();
        await settle();
        expect(must(container.querySelector('[data-testid="main-player-swap-count"]')).textContent).toBe('1');

        await fireEvent.click(must(container.querySelector('[data-testid="main-player-cancel"]')));
        await settle();
        expect(container.querySelector('[data-testid="main-player-confirm"]')).toBeNull();
        const star = must(container.querySelector('[data-testid="main-player-star"]'));
        expect(star.textContent).toBe('★');
        await fireEvent.click(star);
        await settle();
        expect(ClearMainPlayer).not.toHaveBeenCalled();
        expect(must(container.querySelector('[data-testid="main-player-swap-count"]')).textContent).toBe('1');

        PlanMainPlayer.mockResolvedValue({ name: 'Bob', swap: [], refused: [] });
        await fireEvent.click(must(container.querySelector('[data-testid="main-player-apply"]')));
        await settle();
        expect(SetMainPlayer).toHaveBeenCalledWith('Bob');
        expect(container.querySelector('[data-testid="main-player-confirm"]')).toBeNull();
    });

    test('the Remove button of the band forgets the Main player', async () => {
        MainPlayer.mockResolvedValue('Bob');
        PlanMainPlayer.mockResolvedValue({ name: 'Bob', swap: [5], refused: [] });
        const { container } = await mount();
        await settle();
        await fireEvent.click(must(container.querySelector('[data-testid="main-player-clear"]')));
        await settle();
        expect(ClearMainPlayer).toHaveBeenCalledTimes(1);
        expect(SetMainPlayer).not.toHaveBeenCalled();
        expect(container.querySelector('[data-testid="main-player-confirm"]')).toBeNull();
        expect([...container.querySelectorAll('[data-testid="main-player-star"]')].map((b) => b.textContent)).toEqual(['☆', '☆', '☆']);
    });

    test('a merge that leaves the Main player matches to swap brings the band back', async () => {
        MainPlayer.mockResolvedValue('Alice');
        const { container } = await mount();
        expect(container.querySelector('[data-testid="main-player-confirm"]')).toBeNull();
        const rows = container.querySelectorAll('.player-row');
        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        PlanMainPlayer.mockResolvedValue({ name: 'Alice', swap: [8], refused: [] });
        await fireEvent.click(must(container.querySelector('.btn-merge')));
        await settle();
        expect(MergePlayers).toHaveBeenCalled();
        expect(must(container.querySelector('[data-testid="main-player-swap-count"]')).textContent).toBe('1');
        expect(container.querySelector('[data-testid="main-player-clear"]')).not.toBeNull();
    });

    test("a player's aliases are listed on its row, and removing one undoes the merge", async () => {
        ListAliases.mockResolvedValue([{ alias: 'A. Martin', canonical: 'Alice' }]);
        const { container } = await mount();
        const chips = container.querySelectorAll('[data-testid="alias-chip"]');
        expect(chips.length).toBe(1);
        expect(must(chips[0].closest('.player-row')?.querySelector('.player-name')).textContent).toBe('Alice');
        await fireEvent.click(chips[0]);
        await settle();
        expect(RemoveAlias).toHaveBeenCalledWith('player', 'A. Martin');
        expect(onMerged).toHaveBeenCalled();
        expect(MergePlayers).not.toHaveBeenCalled();
    });

    test("a player's match count opens their matches", async () => {
        PlayerMatchIDs.mockResolvedValue([1, 2, 3]);
        const { container } = await mount();
        await fireEvent.click(must(container.querySelector('[data-testid="player-match-count"]')));
        await settle();
        expect(PlayerMatchIDs).toHaveBeenCalledWith('Alice');
        expect(openMatchListInPanel).toHaveBeenCalledWith([1, 2, 3]);
    });
});
