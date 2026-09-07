/**
 * Le rendu des codes du moteur (ADR-0047).
 *
 * Nicomaque n'émet aucune phrase : un libellé, une note, un avertissement sont un code et ses
 * paramètres. C'est ce qui permet à blunderDB de parler neuf langues d'un moteur qui n'en parle
 * aucune, et ce fichier est le pivot de cette traduction — d'où ce test.
 */
import { describe, test, expect } from 'vitest';
import { renderLabel, renderNote, renderWarning, renderSectionName, proposalLabel, actionKey, renderConfigChange, renderLockReason, isRepair } from '../components/direction/labels.js';
import fr from '../i18n/locales/fr.json';
import en from '../i18n/locales/en.json';

/** Une fonction de traduction sur un catalogue, avec la même interpolation que l'application. */
function tFor(catalog) {
    return (key, params) => {
        const raw = key.split('.').reduce((o, k) => (o == null ? undefined : o[k]), catalog);
        if (raw === undefined) return key;
        if (typeof raw !== 'string' || !params) return raw;
        return raw.replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m));
    };
}

const t = tFor(fr);
const tEn = tFor(en);

describe('les codes du moteur deviennent des phrases', () => {
    test('un libellé simple est rendu avec ses paramètres', () => {
        expect(renderLabel(t, { kind: 'round', n: 3 })).toBe('Ronde 3');
        expect(renderLabel(t, { kind: 'swiss_group', losses: 1, match: 4 })).toBe('1 défaite(s), match 4');
        expect(renderLabel(t, { kind: 'quarter_final' })).toBe('Quart de finale');
    });

    test('un libellé composé inclut son sous-libellé', () => {
        expect(renderLabel(t, { kind: 'main_draw', sub: { kind: 'final' } })).toBe('Principal, Finale');
        expect(renderLabel(t, { kind: 'in_block', n: 2, section: 'B2G1', sub: { kind: 'decider' } })).toContain('Match décisif');
    });

    test('un nom de section est un identifiant côté moteur, lisible ici', () => {
        expect(renderSectionName(t, 'main')).toBe('principal');
        expect(renderSectionName(t, 'poule:A')).toBe('poule A');
        expect(renderSectionName(t, 'barrage:B')).toBe('barrage poule B');
        expect(renderSectionName(t, '')).toBe('');
    });

    test('une note de classement est rendue avec ses nombres', () => {
        expect(renderNote(t, { kind: 'alive', lives: 2 })).toBe('en vie (2 vies)');
        expect(renderNote(t, { kind: 'record', wins: 3, losses: 1 })).toBe('3 victoires, 1 défaites');
        expect(renderNote(t, { kind: 'winner' })).toBe('vainqueur');
    });

    test('un avertissement nomme les joueurs, pas leurs identifiants', () => {
        const w = {
            code: 'bracket_wrong_players',
            match: 'M12',
            section: 'main',
            label: { kind: 'semi_final' },
            a: 'alice',
            b: 'bob',
            expected_a: 'chloe',
            expected_b: 'dan'
        };
        const names = { alice: 'Alice', bob: 'Bob', chloe: 'Chloé', dan: 'Dan' };
        const out = renderWarning(t, w, (id) => names[id] || id);
        expect(out).toContain('Alice');
        expect(out).toContain('Chloé');
        expect(out).toContain('Demi-finale');
        expect(out).not.toContain('alice');
    });

    test('un code inconnu se montre au lieu de laisser un blanc', () => {
        // Une version future du moteur qui ajouterait un libellé doit se VOIR à l'écran :
        // un blanc laisserait le directeur sans rien, et la clé nomme ce qu'il faut traduire.
        expect(renderLabel(t, { kind: 'un_code_a_venir' })).toBe('un_code_a_venir');
        expect(renderNote(t, { kind: 'une_note_a_venir' })).toBe('une_note_a_venir');
        expect(renderWarning(t, { code: 'un_avertissement_a_venir' })).toBe('un_avertissement_a_venir');
    });

    test('un libellé absent ne rend rien', () => {
        expect(renderLabel(t, null)).toBe('');
        expect(renderLabel(t, {})).toBe('');
        expect(renderNote(t, null)).toBe('');
    });

    test('la traduction suit la langue', () => {
        expect(renderLabel(t, { kind: 'final' })).toBe('Finale');
        expect(renderLabel(tEn, { kind: 'final' })).toBe('Final');
        expect(renderNote(tEn, { kind: 'winner' })).toBe('winner');
    });
});

describe('une proposition se lit avant de cliquer', () => {
    const names = { alice: 'Alice', bob: 'Bob' };
    const name = (id) => names[id] || id;

    test('un match nomme ses deux joueurs', () => {
        const out = proposalLabel(t, { kind: 'start_match', a: 'alice', b: 'bob', label: { kind: 'round', n: 2 } }, name);
        expect(out).toContain('Alice');
        expect(out).toContain('Bob');
    });

    test('chaque sorte de proposition a son texte', () => {
        expect(proposalLabel(t, { kind: 'bye', a: 'alice', label: {} }, name)).toContain('Alice');
        expect(proposalLabel(t, { kind: 'finish', label: {} }, name)).toBe('Clore le tournoi');
        expect(proposalLabel(t, { kind: 'draw', label: { kind: 'draw_pools', n: 4 } }, name)).toContain('4 poules');
    });
});

describe('la clé d’une proposition est stable', () => {
    test('la même proposition donne la même clé', () => {
        const a = { kind: 'start_match', phase: 0, a: 'alice', b: 'bob' };
        const b = { kind: 'start_match', phase: 0, a: 'alice', b: 'bob' };
        expect(actionKey(a)).toBe(actionKey(b));
    });

    test('deux propositions différentes donnent des clés différentes', () => {
        const a = { kind: 'start_match', phase: 0, a: 'alice', b: 'bob' };
        const b = { kind: 'start_match', phase: 0, a: 'alice', b: 'chloe' };
        expect(actionKey(a)).not.toBe(actionKey(b));
    });

    test('la clé ignore la table, qui change sans que la proposition change', () => {
        // « Ignorer pour l'instant » doit désigner la même proposition au prochain appel ;
        // la table est réattribuée à chaque appel et ne fait pas partie de son identité.
        const a = { kind: 'start_match', phase: 0, a: 'alice', b: 'bob', table: 3 };
        const b = { kind: 'start_match', phase: 0, a: 'alice', b: 'bob', table: 7 };
        expect(actionKey(a)).toBe(actionKey(b));
    });
});

describe('la liste de ce qui va changer (#385)', () => {
    test('un changement nomme le réglage, sa valeur d’avant et celle d’après', () => {
        expect(renderConfigChange(t, { code: 'tableCount', phase: 0, from: '8', to: '12' })).toBe('Nombre de tables : 8 → 12');
        expect(renderConfigChange(t, { code: 'target', phase: 1, from: '16', to: '8' })).toBe('Phase 1 — bascule : 16 → 8');
        expect(renderConfigChange(tEn, { code: 'target', phase: 1, from: '16', to: '8' })).toBe('Phase 1 — switch: 16 → 8');
    });

    test('un format se dit par son nom, pas par son identifiant', () => {
        expect(renderConfigChange(t, { code: 'phaseAdded', phase: 3, to: 'bracket' })).toBe('Phase 3 ajoutée : Tableau');
        expect(renderConfigChange(t, { code: 'kind', phase: 2, from: 'bracket', to: 'round_robin' })).toBe('Phase 2 — format : Tableau → Poules');
    });

    test('un booléen se dit oui ou non, une valeur absente se dit aucun', () => {
        expect(renderConfigChange(t, { code: 'consolation', phase: 1, from: 'false', to: 'true' })).toBe('Phase 1 — consolante : non → oui');
        expect(renderConfigChange(t, { code: 'phaseRemoved', phase: 2, from: 'gsl' })).toBe('Phase 2 retirée : Blocs GSL');
        expect(renderConfigChange(t, { code: 'name', phase: 0, from: '', to: 'Open de Lyon' })).toBe('Nom : aucun → Open de Lyon');
    });

    test('un code inconnu s’affiche tel quel plutôt que de laisser un blanc', () => {
        expect(renderConfigChange(t, { code: 'somethingNew', phase: 1 })).toBe('somethingNew');
        expect(renderConfigChange(t, null)).toBe('');
    });

    test('la raison d’un format figé est un code, rendue dans la langue de l’utilisateur', () => {
        expect(renderLockReason(t, 'started')).toBe('matchs lancés');
        expect(renderLockReason(t, 'drawn')).toBe('tirage fait');
        expect(renderLockReason(tEn, 'finished')).toBe('phase over');
        expect(renderLockReason(t, 'quelquechose')).toBe('quelquechose');
        expect(renderLockReason(t, '')).toBe('');
    });
});

describe('la réparation d’un tableau (#389)', () => {
    const name = (id) => ({ a1: 'Hugo', b2: 'Léa' })[id] || id;

    test('une annulation se lit comme une réparation, avec ceux qui ont joué là', () => {
        const act = { kind: 'cancel_match', a: 'a1', b: 'b2', label: { kind: 'semi_final' } };
        expect(proposalLabel(t, act, name)).toBe('Annuler Demi-finale : Hugo – Léa');
        expect(proposalLabel(tEn, act, name)).toBe('Cancel Semi-final: Hugo – Léa');
    });

    /* Le moteur ne propose une annulation que pour remettre un graphe d'accord avec les
       résultats : c'est le seul cas, et c'est ce qui distingue ces lignes sans inventer de
       marqueur. */
    test('seule une annulation est une réparation', () => {
        expect(isRepair({ kind: 'cancel_match' })).toBe(true);
        expect(isRepair({ kind: 'start_match' })).toBe(false);
        expect(isRepair({ kind: 'wait' })).toBe(false);
        expect(isRepair(null)).toBe(false);
    });
});
