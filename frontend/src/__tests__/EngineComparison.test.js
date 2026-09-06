/**
 * EngineComparison.test.js — la comparaison inter-moteurs (#269, fiche I.13).
 *
 * Ce qui est vérifié : la bande ne s'affiche pas quand il n'y a rien à
 * comparer, elle nomme le désaccord quand il existe, et — le point qui compte
 * — le « meilleur coup » de chaque moteur est le meilleur DE CE MOTEUR, et non
 * le premier de la liste commune, qui est triée tous moteurs confondus.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';

import EngineComparison from '../components/EngineComparison.svelte';

afterEach(cleanup);

describe('la comparaison inter-moteurs', () => {
    test("ne s'affiche pas avec un seul moteur", () => {
        const { container } = render(EngineComparison, {
            props: {
                kind: 'cube',
                analysis: { doublingCubeAnalysis: { analysisEngine: 'XG', bestCubeAction: 'Double, Take' } }
            }
        });
        expect(container.querySelector('.engine-comparison')).toBeNull();
    });

    test('nomme le désaccord sur le videau', () => {
        render(EngineComparison, {
            props: {
                kind: 'cube',
                analysis: {
                    allCubeAnalyses: [
                        { analysisEngine: 'XG', analysisDepth: '4-ply', bestCubeAction: 'Double, Take' },
                        { analysisEngine: 'gammonNet', analysisDepth: '2-ply', bestCubeAction: 'No Double' }
                    ]
                }
            }
        });
        expect(screen.getByText('The engines disagree:')).toBeTruthy();
        expect(screen.getByText('Double, Take')).toBeTruthy();
        expect(screen.getByText('No Double')).toBeTruthy();
    });

    test("dit l'accord sans le crier", () => {
        render(EngineComparison, {
            props: {
                kind: 'cube',
                analysis: {
                    allCubeAnalyses: [
                        { analysisEngine: 'XG', bestCubeAction: 'No Double' },
                        { analysisEngine: 'GNUbg', bestCubeAction: 'No Double' }
                    ]
                }
            }
        });
        expect(screen.getByText('The engines agree:')).toBeTruthy();
    });

    // Le point qui compte. La liste des coups est triée par équité, TOUS
    // moteurs confondus : son premier élément n'est le meilleur coup d'aucun
    // moteur en particulier. Chaque ligne doit porter le meilleur de son
    // propre moteur.
    test('le meilleur coup de chaque moteur est le sien, pas le premier de la liste', () => {
        render(EngineComparison, {
            props: {
                kind: 'checker',
                analysis: {
                    checkerAnalysis: {
                        moves: [
                            { move: '13/7 8/7', equity: 0.21, analysisEngine: 'XG' },
                            { move: '24/18 13/12', equity: 0.19, analysisEngine: 'gammonNet' },
                            { move: '13/7 24/23', equity: 0.15, analysisEngine: 'XG' },
                            { move: '13/7 8/7', equity: 0.11, analysisEngine: 'gammonNet' }
                        ]
                    }
                }
            }
        });
        expect(screen.getByText('The engines disagree:')).toBeTruthy();
        // XG : 13/7 8/7 (0,21) ; gammonNet : 24/18 13/12 (0,19), et surtout
        // pas 13/7 8/7, qui est en tête de la liste commune.
        expect(screen.getByText('24/18 13/12')).toBeTruthy();
        expect(screen.getByText('13/7 8/7')).toBeTruthy();
    });
});
