// @ts-nocheck — doublures de test : les signatures typées (Wails, stores) exigent des objets complets.
import { describe, it, expect, vi, beforeEach } from 'vitest';

const lessonHTML = vi.fn(async () => '<html></html>');
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LessonHTML: (...a) => lessonHTML(...a),
    LessonPositionIDs: vi.fn(async () => [3, 3, 4]),
    LoadPositionsByIDs: vi.fn(async (ids) => ids.map((id) => ({ id }))),
    GetLesson: vi.fn(async () => ({ name: 'Primes' }))
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ SaveBoardImageDialog: vi.fn(), SaveBoardSVG: vi.fn() }));
vi.mock('../services/diagramService.js', () => ({ renderPositionSVG: vi.fn((p) => `<svg id="${p.id}"/>`) }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

import { buildLessonHTML } from '../services/lessonHtmlService.js';

describe('buildLessonHTML', () => {
    beforeEach(() => lessonHTML.mockClear());

    it('draws each board once with renderPositionSVG and hands them to the engine', async () => {
        const html = await buildLessonHTML(7);
        expect(html).toBe('<html></html>');
        const [id, , diagrams] = lessonHTML.mock.calls[0];
        expect(id).toBe(7);
        expect(diagrams).toEqual({ 3: '<svg id="3"/>', 4: '<svg id="4"/>' });
    });
});
