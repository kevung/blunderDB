import { describe, it, expect } from 'vitest';
import { boardZones, pointZone, barZone, trayZone, checkerCenter, initialPositionGhosts, FIELD_WIDTH, FIELD_HEIGHT, BOARD_RECT, CALIBRATION_CORNERS } from './boardModel.js';

const overlaps = (a, b) => a.x0 < b.x1 && b.x0 < a.x1 && a.y0 < b.y1 && b.y0 < a.y1;
const inside = (r, [u, v]) => u > r.x0 && u < r.x1 && v > r.y0 && v < r.y1;

describe('board model in checker diameters', () => {
    it('has 28 disjoint zones inside the rectified frame', () => {
        const zones = boardZones();
        expect(zones).toHaveLength(28);
        expect(new Set(zones.map((z) => z.id)).size).toBe(28);
        for (let i = 0; i < zones.length; i++) {
            const r = zones[i].rect;
            expect(r.x0 >= BOARD_RECT.x0 && r.x1 <= BOARD_RECT.x1 && r.y0 >= BOARD_RECT.y0 && r.y1 <= BOARD_RECT.y1).toBe(true);
            for (let j = i + 1; j < zones.length; j++) expect(overlaps(r, zones[j].rect), `${zones[i].id} / ${zones[j].id}`).toBe(false);
        }
    });

    it('is about 15 × 13 d with the trays', () => {
        expect(FIELD_WIDTH).toBe(13);
        expect(FIELD_HEIGHT).toBe(13);
        expect(BOARD_RECT.x1 - BOARD_RECT.x0).toBeGreaterThanOrEqual(13);
        expect(BOARD_RECT.x1 - BOARD_RECT.x0).toBeLessThanOrEqual(15);
    });

    it('places the clicked corners at the 1-, 12-, 13- and 24-points', () => {
        const corner = (n) => CALIBRATION_CORNERS.find(([u, v]) => Math.abs(u - pointZone(n).base[0]) <= 0.5 && v === pointZone(n).base[1]);
        expect(corner(1)).toEqual(CALIBRATION_CORNERS[0]);
        expect(corner(12)).toEqual(CALIBRATION_CORNERS[1]);
        expect(corner(13)).toEqual(CALIBRATION_CORNERS[2]);
        expect(corner(24)).toEqual(CALIBRATION_CORNERS[3]);
    });

    it('puts checker k at (k + 0.5)·d from the base edge', () => {
        expect(checkerCenter(pointZone(1), 0)).toEqual([0.5, 0.5]);
        expect(checkerCenter(pointZone(1), 4)).toEqual([0.5, 4.5]);
        expect(checkerCenter(pointZone(24), 0)).toEqual([0.5, 12.5]);
        expect(checkerCenter(pointZone(7), 0)).toEqual([7.5, 0.5]);
        expect(checkerCenter(pointZone(12), 0)).toEqual([12.5, 0.5]);
        expect(checkerCenter(pointZone(13), 2)).toEqual([12.5, 10.5]);
        expect(barZone('near').rect.x0).toBe(6);
        expect(trayZone('far').rect.x1).toBe(0);
        expect(() => pointZone(0)).toThrow(RangeError);
        expect(() => pointZone(25)).toThrow(RangeError);
    });

    it('gives 15 ghost checkers per side, each on its own spot inside a point', () => {
        const ghosts = initialPositionGhosts();
        expect(ghosts.filter((g) => g.side === 'near')).toHaveLength(15);
        expect(ghosts.filter((g) => g.side === 'far')).toHaveLength(15);
        const points = boardZones().filter((z) => z.kind === 'point');
        for (const g of ghosts) expect(points.some((z) => inside(z.rect, g.center))).toBe(true);
        expect(new Set(ghosts.map((g) => g.center.join(','))).size).toBe(30);
    });
});
