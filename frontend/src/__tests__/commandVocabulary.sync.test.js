// Guards the invariant stated at the top of commandVocabulary.js: every command
// offered by autocompletion must actually be handled by commandProcessor.js.
//
// processCommand's if/else chain ends in a trailing `else` that reports
// commands.unknown, but that is itself an observable effect (a status message)
// and would make every vocabulary entry pass trivially if we only checked "did
// something happen". A vocabulary entry whose branch was removed — that is how
// `filter`/`fl` survived long after its panel was folded into the search panel —
// still needs to be caught. So this test drives every entry through
// processCommand and asserts it produces some effect OTHER than commands.unknown:
// a callback access, a modal, a status message with a different key, a log
// line, or a Wails call.

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveComment: vi.fn().mockResolvedValue(undefined),
    ClearCommandHistory: vi.fn().mockResolvedValue(undefined),
    SaveSearchHistory: vi.fn().mockResolvedValue(undefined)
}));

import { COMMANDS } from '../commandVocabulary.js';
import { processCommand, initCommandProcessor } from '../commandProcessor.js';
import { activeModal, statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

// Any property read on the callbacks object means a branch tried to dispatch.
function trackingCallbacks(touched) {
    return new Proxy(
        {},
        {
            get(_target, prop) {
                touched.add(String(prop));
                return vi.fn();
            }
        }
    );
}

const OBSERVED = [activeModal, statusBarTextStore, statusBarModeStore, currentPositionIndexStore];

describe('commandVocabulary ↔ commandProcessor sync', () => {
    beforeEach(() => {
        // A database must look open, or guarded branches take the "no database"
        // path — which is still an observable effect, but not the one we mean.
        // databaseLoadedStore is derived from the path, so set the path.
        databasePathStore.set('/tmp/test.db');
        positionsStore.set([{ id: 1 }]);
        activeModal.set(null);
        statusBarTextStore.set('');
        statusBarModeStore.set('');
        currentPositionIndexStore.set(0);
    });

    test.each(COMMANDS.flatMap((cmd) => [cmd.name, ...cmd.aliases]).map((form) => [form]))('`%s` is handled by processCommand', async (form) => {
        const touched = new Set();
        initCommandProcessor(trackingCallbacks(touched));
        const before = OBSERVED.map((s) => JSON.stringify(get(s)));

        processCommand(form);
        // Branches like `clear` settle their stores in a promise callback, so
        // let the microtask queue drain before looking.
        await new Promise((resolve) => setTimeout(resolve, 0));

        const changed = OBSERVED.some((s, i) => JSON.stringify(get(s)) !== before[i]);
        const statusText = get(statusBarTextStore);
        // A dropped branch now falls through to the trailing `else`, which sets
        // commands.unknown — itself a change to statusBarTextStore, so `changed`
        // alone would pass trivially. Excluding it is what makes this test still
        // catch a vocabulary entry whose handler was removed.
        const fellThroughToUnknown = statusText && typeof statusText === 'object' && statusText.i18nKey === 'commands.unknown';
        expect(
            (touched.size > 0 || changed) && !fellThroughToUnknown,
            `"${form}" is offered by autocomplete but processCommand does nothing with it. ` + `Either add a branch in commandProcessor.js or drop it from commandVocabulary.js.`
        ).toBe(true);
    });

    test('no duplicate command forms across entries', () => {
        const forms = COMMANDS.flatMap((cmd) => [cmd.name, ...cmd.aliases]);
        expect(forms.length).toBe(new Set(forms).size);
    });
});
