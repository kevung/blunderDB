/**
 * ExportDatabaseModal.focus.test.js
 *
 * Regression: ticking "mark this file with its origin" revealed the fields, but they could
 * not be typed into — the caret would not stay in them. The modal mirrors the options it is
 * given into local $state (the parent passes a plain object out of a store, which Svelte 5
 * does not track); if that mirror is re-seeded while the user types, the inputs are
 * recreated on every keystroke and focus is lost.
 *
 * These tests type into each of the fields added for issuance and assert that the value
 * accumulates, the element keeps focus, and the change reaches the object the export service
 * reads.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import { writable, get } from 'svelte/store';

import ExportDatabaseModal from '../components/ExportDatabaseModal.svelte';
import ExportModalHost from './fixtures/ExportModalHost.svelte';

afterEach(cleanup);

function baseOptions() {
    return {
        includeAnalysis: true,
        includeComments: true,
        includeFilterLibrary: false,
        includePlayedMoves: true,
        includeMatches: false,
        matchIDs: [],
        includeTournaments: false,
        includeTournamentIDs: [],
        includeCollections: false,
        collectionIDs: [],
        watermarkEnabled: false,
        watermark: '',
        watermarkNote: '',
        passwordEnabled: false,
        password: ''
    };
}

function mount(exportOptions) {
    return render(ExportDatabaseModal, {
        props: {
            visible: true,
            mode: 'metadata',
            positionCount: 3,
            metadata: { user: '', description: '', dateOfCreation: '' },
            exportOptions,
            matches: [],
            onCancel: () => {},
            onExport: () => {},
            onClose: () => {}
        }
    });
}

// Typing a word one character at a time is the whole point: a mirror that re-seeds on each
// keystroke keeps only the last character, and the element loses focus every time.
async function typeInto(input, text) {
    for (const char of text) {
        await fireEvent.input(input, { target: { value: input.value + char } });
        await tick();
    }
}

// The real parent binds the options to a store (App.svelte). Mounting the modal directly
// with a plain prop hides the interaction that broke typing, so this second suite drives it
// exactly as the application does.
function mountThroughStore(options) {
    const optionsStore = writable(options);
    const metadataStore = writable({ user: '', description: '', dateOfCreation: '' });
    return { ...render(ExportModalHost, { props: { optionsStore, metadataStore } }), optionsStore };
}

describe('issuance fields, mounted the way the application mounts them', () => {
    test('the origin field keeps focus while typing', async () => {
        const options = baseOptions();
        options.watermarkEnabled = true;
        const { container, optionsStore } = mountThroughStore(options);
        await tick();

        const input = container.querySelector('#export-origin');
        expect(input).not.toBeNull();
        input.focus();
        await typeInto(input, 'Cours de Jean Dupont');
        await tick();

        expect(container.querySelector('#export-origin')).toBe(input);
        expect(input.value).toBe('Cours de Jean Dupont');
        expect(document.activeElement).toBe(input);
        expect(get(optionsStore).watermark).toBe('Cours de Jean Dupont');
    });

    test('the password field keeps focus while typing', async () => {
        const options = baseOptions();
        options.passwordEnabled = true;
        const { container, optionsStore } = mountThroughStore(options);
        await tick();

        const input = container.querySelector('#export-password');
        input.focus();
        await typeInto(input, 'mot-de-passe');
        await tick();

        expect(container.querySelector('#export-password')).toBe(input);
        expect(input.value).toBe('mot-de-passe');
        expect(document.activeElement).toBe(input);
        expect(get(optionsStore).password).toBe('mot-de-passe');
    });
});

describe('issuance fields in the export modal', () => {
    test('the origin field accepts typing and keeps focus', async () => {
        const options = baseOptions();
        options.watermarkEnabled = true;
        const { container } = mount(options);
        await tick();

        const input = container.querySelector('#export-origin');
        expect(input).not.toBeNull();

        input.focus();
        await typeInto(input, 'Cours de Jean');
        await tick();

        expect(container.querySelector('#export-origin')).toBe(input);
        expect(input.value).toBe('Cours de Jean');
        expect(document.activeElement).toBe(input);
        expect(options.watermark).toBe('Cours de Jean');
    });

    test('the note field accepts typing', async () => {
        const options = baseOptions();
        options.watermarkEnabled = true;
        const { container } = mount(options);
        await tick();

        const input = container.querySelector('#export-watermark-note');
        input.focus();
        await typeInto(input, 'Ne pas rediffuser');
        await tick();

        expect(input.value).toBe('Ne pas rediffuser');
        expect(document.activeElement).toBe(input);
        expect(options.watermarkNote).toBe('Ne pas rediffuser');
    });

    test('the password field accepts typing', async () => {
        const options = baseOptions();
        options.passwordEnabled = true;
        const { container } = mount(options);
        await tick();

        const input = container.querySelector('#export-password');
        input.focus();
        await typeInto(input, 's3cret');
        await tick();

        expect(input.value).toBe('s3cret');
        expect(document.activeElement).toBe(input);
        expect(options.password).toBe('s3cret');
    });

    test('ticking a box reveals its fields', async () => {
        const options = baseOptions();
        const { container } = mount(options);
        await tick();

        expect(container.querySelector('#export-origin')).toBeNull();
        await fireEvent.click(container.querySelector('#export-watermark'));
        await tick();
        expect(container.querySelector('#export-origin')).not.toBeNull();
        expect(options.watermarkEnabled).toBe(true);
    });
});
