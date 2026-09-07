/**
 * transcriptionKeys.turn.test.js — le tableau d'ux.md §3, ligne par ligne.
 *
 * Un tour de pions est le geste le plus fréquent du document (~250 fois par
 * match) et le cœur du budget KLM. Ce fichier tient les deux : une transition
 * par test, y compris les cas limites nommés par la fiche T1.3 — dernier coup
 * d'une partie, danse, correction du jet avant validation — et un compte de
 * touches qui échoue au-delà du budget d'ux.md §4.1.
 *
 * L'ouverture, qui est la même saisie de deux dés avec une validation
 * différente, est dans transcriptionKeys.opening.test.js.
 */

import { describe, test, expect } from 'vitest';
import { PHASE, COMMAND, initialKeyState, pressKey, applyCandidates, selectCandidate, selectionDelta } from '../services/transcriptionKeys.js';

const CHECKER = { expects: 'checker' };

/**
 * Une frappe. Les chiffres sont positionnels (`event.code`), les lettres se
 * lisent au caractère produit (`event.key`) : le pilote pose les deux, comme le
 * navigateur le fait.
 */
function key(code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

/**
 * Un pilote qui compte les touches, exactement comme un utilisateur les presse.
 * `candidates` est ce que le moteur répond à chaque jet — c'est le seul point
 * où la machine attend quelqu'un d'autre.
 */
function driver({ expects = 'checker', candidates = 5 } = {}) {
    let state = initialKeyState();
    const commands = [];
    let presses = 0;

    const settle = () => {
        if (!state.awaitingCandidates) return;
        const result = applyCandidates(state, candidates);
        state = result.state;
        commands.push(...result.commands);
    };

    return {
        press(code, extra) {
            presses += 1;
            const result = pressKey(state, key(code, extra), { expects });
            state = result.state;
            commands.push(...result.commands);
            settle();
            return result;
        },
        answer(count) {
            candidates = count;
            settle();
        },
        get state() {
            return state;
        },
        get commands() {
            return commands;
        },
        get presses() {
            return presses;
        },
        kinds() {
            return commands.map((c) => c.kind);
        }
    };
}

describe('les transitions d’ux.md §3', () => {
    test('dés attendus + un chiffre → premier dé', () => {
        const d = driver();
        d.press('Digit3');
        expect(d.state.phase).toBe(PHASE.DIE1);
        expect(d.state.dice).toEqual([3, 0]);
        expect(d.commands).toEqual([{ kind: COMMAND.DIE, value: 3 }]);
    });

    test('un dé + un chiffre → second dé, liste classée, premier présélectionné', () => {
        const d = driver({ candidates: 17 });
        d.press('Digit3');
        d.press('Digit1');
        expect(d.state.phase).toBe(PHASE.ROLL);
        expect(d.state.dice).toEqual([3, 1]);
        expect(d.state.selected).toBe(0);
        expect(d.state.candidateCount).toBe(17);
        expect(d.kinds()).toEqual([COMMAND.DIE, COMMAND.DIE, COMMAND.SELECT]);
        expect(d.commands.at(-1)).toEqual({ kind: COMMAND.SELECT, index: 0 });
    });

    // « jet corrigeable » : c'est ce que veut dire ce nom.
    test('jet corrigeable + un chiffre → le jet recommence, rien n’est validé', () => {
        const d = driver();
        d.press('Digit3');
        d.press('Digit1');
        d.press('Digit4');
        expect(d.state.phase).toBe(PHASE.DIE1);
        expect(d.state.dice).toEqual([4, 0]);
        expect(d.kinds()).not.toContain(COMMAND.VALIDATE);
    });

    test('jet corrigeable + j/k → la sélection bouge et l’on sort du jet corrigeable', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        expect(d.state.phase).toBe(PHASE.CANDIDATE);
        expect(d.state.selected).toBe(1);
        expect(d.commands.at(-1)).toEqual({ kind: COMMAND.SELECT, index: 1 });
    });

    test('les flèches font ce que font j et k', () => {
        expect(selectionDelta(key('ArrowDown'))).toBe(1);
        expect(selectionDelta(key('ArrowUp'))).toBe(-1);
        const d = driver({ candidates: 4 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('ArrowDown');
        d.press('ArrowDown');
        d.press('ArrowUp');
        expect(d.state.selected).toBe(1);
    });

    test('la sélection est bornée : elle ne boucle pas', () => {
        const d = driver({ candidates: 2 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyK'); // déjà en haut
        expect(d.state.selected).toBe(0);
        // Mais on a bien quitté le jet corrigeable : la liste a été touchée.
        expect(d.state.phase).toBe(PHASE.CANDIDATE);
        d.press('KeyJ');
        d.press('KeyJ');
        expect(d.state.selected).toBe(1);
    });

    test('candidat choisi + j/k → la sélection bouge, on y reste', () => {
        const d = driver({ candidates: 6 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        d.press('KeyJ');
        expect(d.state.phase).toBe(PHASE.CANDIDATE);
        expect(d.state.selected).toBe(2);
    });

    test('candidat choisi + un chiffre → valide, puis premier dé du tour suivant', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        d.press('Digit6');
        expect(d.kinds().slice(-2)).toEqual([COMMAND.VALIDATE, COMMAND.DIE]);
        expect(d.state.phase).toBe(PHASE.DIE1);
        expect(d.state.dice).toEqual([6, 0]);
        expect(d.state.selected).toBe(0);
    });

    // Le cas limite que la règle « un chiffre valide » laisse ouvert : il n'y a
    // pas de tour suivant pour porter la validation.
    test('Entrée valide seule — le dernier coup d’une partie', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('Enter');
        expect(d.kinds().at(-1)).toBe(COMMAND.VALIDATE);
        expect(d.state.phase).toBe(PHASE.DICE);
        expect(d.state.dice).toEqual([0, 0]);
    });

    test('Entrée valide aussi depuis « candidat choisi »', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        d.press('Enter');
        expect(d.kinds().at(-1)).toBe(COMMAND.VALIDATE);
        expect(d.state.phase).toBe(PHASE.DICE);
    });

    test('Entrée sans jet saisi ne valide rien', () => {
        const result = pressKey(initialKeyState(), key('Enter'), CHECKER);
        expect(result.handled).toBe(false);
    });

    // La correction du jet AVANT validation : Retour arrière efface les deux
    // dés, l'Action n'existe pas encore.
    test('Retour arrière efface les deux dés depuis le jet corrigeable', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('Backspace');
        expect(d.kinds().at(-1)).toBe(COMMAND.CLEAR);
        expect(d.state.phase).toBe(PHASE.DICE);
        expect(d.state.dice).toEqual([0, 0]);
        expect(d.kinds()).not.toContain(COMMAND.VALIDATE);
    });

    test('Retour arrière efface aussi depuis « candidat choisi »', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        d.press('Backspace');
        expect(d.kinds().at(-1)).toBe(COMMAND.CLEAR);
        expect(d.state.phase).toBe(PHASE.DICE);
    });

    test('un clic sur une ligne choisit ce candidat', () => {
        const d = driver({ candidates: 9 });
        d.press('Digit3');
        d.press('Digit1');
        const picked = selectCandidate(d.state, 4);
        expect(picked.state.phase).toBe(PHASE.CANDIDATE);
        expect(picked.state.selected).toBe(4);
        expect(picked.commands).toEqual([{ kind: COMMAND.SELECT, index: 4 }]);
    });
});

describe('la danse', () => {
    // « dès le second dé, si la liste est vide, l'Action `dance` est créée et
    // l'on passe à "dés attendus" pour l'autre camp — zéro touche de plus. »
    test('un jet sans aucun coup légal crée la danse, sans une touche de plus', () => {
        const d = driver({ candidates: 0 });
        d.press('Digit6');
        d.press('Digit6');
        expect(d.presses).toBe(2);
        expect(d.kinds()).toEqual([COMMAND.DIE, COMMAND.DIE, COMMAND.DANCE]);
        expect(d.state.phase).toBe(PHASE.DICE);
        expect(d.state.dice).toEqual([0, 0]);
        expect(d.state.awaitingCandidates).toBe(false);
    });

    test('la danse ne présélectionne rien', () => {
        const after = applyCandidates({ ...initialKeyState(), awaitingCandidates: true }, 0);
        expect(after.commands).toEqual([{ kind: COMMAND.DANCE }]);
        expect(after.state.candidateCount).toBe(0);
    });

    // Tant que le moteur n'a pas répondu, j/k n'a rien à déplacer — mais la
    // touche reste au panneau, sans quoi elle irait déplacer le plateau.
    test('j avant la réponse du moteur est pris et n’émet rien', () => {
        let state = initialKeyState();
        for (const code of ['Digit3', 'Digit1']) {
            state = pressKey(state, key(code), CHECKER).state;
        }
        const result = pressKey(state, key('KeyJ'), CHECKER);
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([]);
    });
});

describe('le budget d’ux.md §4.1', () => {
    // DIVERGENCE MESURÉE, et signalée plutôt que tranchée en silence.
    //
    // ux.md §3 et ux.md §4.1 ne disent pas la même chose de la même touche. §3 :
    // « jet corrigeable | 1–6 | recommence : premier dé » — un chiffre re-saisit
    // le jet, ce qui EST la correction chiffrée à 2 K par §4.3 (« dé mal lu, vu
    // aussitôt : 4 1 »). §4.1 : « meilleur coup joué | 3 1 (le chiffre suivant
    // valide) | 2 K ». Les deux décrivent la même frappe depuis le même état, et
    // s'excluent : un chiffre ne peut pas à la fois recommencer le jet et
    // valider le coup présélectionné.
    //
    // C'est §3 qui est implémenté — la fiche T1.3 dit « exactement le tableau
    // §3 », et §3 est le seul des deux qui garde une correction à deux touches.
    // Le coût mesuré du meilleur coup joué est donc de TROIS touches, `3` `1`
    // Entrée, et non deux ; le n-ième coup tient son budget, sa validation étant
    // portée par le premier chiffre du tour suivant. À l'auteur de trancher :
    // faire valider un chiffre depuis « jet corrigeable » ramène le meilleur
    // coup à 2 K et pousse la correction de §4.3 à 3 K (Retour, `4`, `1`).

    // « n-ième coup, n ≤ 5 : 3 1 j×(n−1) — (n+1) K » : tenu.
    test('le n-ième coup coûte 2 + (n−1) touches', () => {
        for (let n = 2; n <= 5; n += 1) {
            const d = driver({ candidates: 17 });
            d.press('Digit3');
            d.press('Digit1');
            for (let i = 1; i < n; i += 1) d.press('KeyJ');
            expect(d.state.selected).toBe(n - 1);
            expect(d.presses).toBe(2 + (n - 1));
            // Et le coup est choisi : sa validation est portée par la première
            // touche du tour suivant, qui n'appartient plus à ce tour-ci.
            expect(d.state.phase).toBe(PHASE.CANDIDATE);
        }
    });

    test('la validation du n-ième coup est portée par le tour suivant', () => {
        const d = driver({ candidates: 17 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        const before = d.presses;
        d.press('Digit6'); // premier dé du tour suivant
        expect(d.kinds()).toContain(COMMAND.VALIDATE);
        expect(d.presses).toBe(before + 1);
    });

    // Le meilleur coup joué : deux touches pour le désigner, une troisième pour
    // l'enregistrer tant que §3 fait recommencer le jet sur un chiffre.
    test('le meilleur coup est désigné en deux touches', () => {
        const d = driver({ candidates: 17 });
        d.press('Digit3');
        d.press('Digit1');
        expect(d.presses).toBe(2);
        expect(d.state.selected).toBe(0);
        expect(d.commands.at(-1)).toEqual({ kind: COMMAND.SELECT, index: 0 });
    });

    test('et enregistré en trois — la divergence mesurée, budget §4.1 : 2', () => {
        const d = driver({ candidates: 17 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('Enter');
        expect(d.kinds()).toContain(COMMAND.VALIDATE);
        expect(d.presses).toBe(3);
    });

    // Le budget est une propriété du dessin, pas d'un tour choisi : aucun rang
    // ne coûte plus que son 2 + (n−1), validation exclue.
    test('aucun tour ne dépasse son budget', () => {
        for (const rank of [1, 2, 3, 8, 12]) {
            const d = driver({ candidates: 20 });
            d.press('Digit5');
            d.press('Digit3');
            for (let i = 1; i < rank; i += 1) d.press('KeyJ');
            expect(d.presses).toBeLessThanOrEqual(2 + (rank - 1));
        }
    });
});
