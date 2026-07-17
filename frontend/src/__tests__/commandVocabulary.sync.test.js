// Guards the invariant stated at the top of commandVocabulary.js: every command
// offered by autocompletion must actually be handled by commandProcessor.js.
//
// processCommand's if/else chain has no trailing `else`, so an unhandled command
// is a silent no-op — autocomplete keeps offering it and typing it does nothing.
// That is how `filter`/`fl` survived in the vocabulary long after its panel was
// folded into the search panel. Rather than grep the source for each name, drive
// every entry through processCommand and assert it produces *some* observable
// effect: a callback access, a modal, a status message, a log line, or a Wails call.

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveComment: vi.fn().mockResolvedValue(undefined),
    Migrate_1_0_0_to_1_1_0: vi.fn().mockResolvedValue(undefined),
    Migrate_1_1_0_to_1_2_0: vi.fn().mockResolvedValue(undefined),
    Migrate_1_2_0_to_1_3_0: vi.fn().mockResolvedValue(undefined),
    ClearCommandHistory: vi.fn().mockResolvedValue(undefined),
    SaveSearchHistory: vi.fn().mockResolvedValue(undefined)
}));

import { COMMANDS } from '../commandVocabulary.js';
import { processCommand, initCommandProcessor } from '../commandProcessor.js';
import { activeModal, statusBarTextStore, statusBarModeStore, logEntriesStore, currentPositionIndexStore } from '../stores/uiStore.js';
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

const OBSERVED = [activeModal, statusBarTextStore, statusBarModeStore, logEntriesStore, currentPositionIndexStore];

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
        logEntriesStore.set([]);
        currentPositionIndexStore.set(0);
    });

    test.each(COMMANDS.flatMap((cmd) => [cmd.name, ...cmd.aliases]).map((form) => [form]))('`%s` is handled by processCommand', async (form) => {
        const touched = new Set();
        initCommandProcessor(trackingCallbacks(touched));
        const before = OBSERVED.map((s) => JSON.stringify(get(s)));

        processCommand(form);
        // Branches like `clear` and the migrations settle their stores in a
        // promise callback, so let the microtask queue drain before looking.
        await new Promise((resolve) => setTimeout(resolve, 0));

        const changed = OBSERVED.some((s, i) => JSON.stringify(get(s)) !== before[i]);
        expect(
            touched.size > 0 || changed,
            `"${form}" is offered by autocomplete but processCommand does nothing with it. ` + `Either add a branch in commandProcessor.js or drop it from commandVocabulary.js.`
        ).toBe(true);
    });

    test('no duplicate command forms across entries', () => {
        const forms = COMMANDS.flatMap((cmd) => [cmd.name, ...cmd.aliases]);
        expect(forms.length).toBe(new Set(forms).size);
    });
});
