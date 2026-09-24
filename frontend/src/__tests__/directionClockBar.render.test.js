/**
 * La bande d'horloge (#456, D8.3).
 *
 * S5 mercredi 9 h lisait « 48 h 45 · … · pause à HH:MM » : les nuits comptées comme du jeu, une
 * pause sans jour, et aucune fin estimée. Une Direction close annonçait encore une pause.
 */
import { describe, test, expect, afterEach, beforeEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import ClockBar from '../components/direction/ClockBar.svelte';
import { language } from '../i18n';

const WEDNESDAY_9H = new Date(2026, 9, 7, 9, 0);
const iso = (/** @type {Date} */ d) => d.toISOString();

beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(WEDNESDAY_9H);
    language.set('en');
});
afterEach(() => {
    cleanup();
    vi.useRealTimers();
});

const BASE = { elapsedSeconds: 0, playingSeconds: 0, day: 0, played: 0, running: 0, minutesPerPoint: 0, plannedPerPoint: 8, slowMatches: 0, warnings: 0, finished: false };

describe("la bande d'horloge", () => {
    test('sur plusieurs jours, elle dit le jour et le temps de jeu, pas le temps écoulé', () => {
        const clock = { ...BASE, elapsedSeconds: 48 * 3600 + 45 * 60, playingSeconds: 21 * 3600 + 10 * 60, day: 3 };
        const { container } = render(ClockBar, { props: { clock } });
        const text = container.textContent || '';
        expect(text).toContain('day 3');
        expect(text).toContain('21 h 10');
        expect(text).not.toContain('48 h 45');
    });

    test('sur une seule journée, le temps écoulé reste', () => {
        const clock = { ...BASE, elapsedSeconds: 2 * 3600 + 5 * 60, playingSeconds: 3600, day: 1 };
        const { container } = render(ClockBar, { props: { clock } });
        expect(container.textContent).toContain('2 h 05');
        expect(container.textContent).not.toContain('day 1');
    });

    test('la fin estimée est affichée, avec son jour quand ce n’est pas aujourd’hui', () => {
        const clock = { ...BASE, day: 3, estimatedEnd: iso(new Date(2026, 9, 9, 2, 0)) };
        const { getByTestId } = render(ClockBar, { props: { clock } });
        const end = getByTestId('direction-clock-end').textContent || '';
        expect(end).toContain('Fri');
        expect(end).toContain('02:00');
    });

    test('une fin ou une pause du jour même n’a pas de jour', () => {
        const clock = { ...BASE, day: 3, estimatedEnd: iso(new Date(2026, 9, 7, 17, 30)), nextBreak: iso(new Date(2026, 9, 7, 12, 30)) };
        const { getByTestId } = render(ClockBar, { props: { clock } });
        expect(getByTestId('direction-clock-end').textContent).not.toContain('Wed');
        expect(getByTestId('direction-clock-break').textContent).not.toContain('Wed');
        expect(getByTestId('direction-clock-break').textContent).toContain('12:30');
    });

    test('la pause d’un autre jour porte son jour', () => {
        const clock = { ...BASE, nextBreak: iso(new Date(2026, 9, 8, 12, 30)) };
        const { getByTestId } = render(ClockBar, { props: { clock } });
        expect(getByTestId('direction-clock-break').textContent).toContain('Thu');
    });

    test('après la clôture, la bande ne dit plus rien', () => {
        const clock = { ...BASE, finished: true, nextBreak: iso(new Date(2026, 9, 7, 12, 30)) };
        const { container } = render(ClockBar, { props: { clock } });
        expect(container.querySelector('.clock')).toBeNull();
    });
});
