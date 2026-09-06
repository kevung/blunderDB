import { TranslateIntent } from '../../wailsjs/go/database/Database.js';
import { commandTextStore, statusBarModeStore } from '../stores/uiStore.js';
import { positionStore } from '../stores/positionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// La grammaire d'intentions (#283, fiche I.27), côté interface.
//
// Le résultat n'est PAS une recherche : c'est une ligne de commande, écrite
// dans la barre, que l'utilisateur relit et lance. C'est tout l'intérêt de
// l'option 3 de #38 — les jetons sont VISIBLES, donc apprenables, et une
// traduction fausse se voit avant de renvoyer des résultats faux plutôt
// qu'après.
//
// La traduction elle-même est en Go : l'interface, la ligne de commande et le
// démon doivent comprendre la même phrase de la même façon, et une seconde
// table de vocabulaire ici aurait été un second sens pour « blunder ».

/**
 * Traduit une phrase et prépare la recherche : les jetons vont dans la barre
 * de commande, les deux intentions qui décrivent le PLATEAU de recherche y
 * sont posées, et ce qui n'a pas été compris est dit.
 * @param {string} phrase
 */
export async function askIntent(phrase) {
    const text = (phrase || '').trim();
    if (!text) {
        setStatusBarMessage(tMsg('intent.empty'));
        return null;
    }
    let intent;
    try {
        intent = await TranslateIntent(text);
    } catch (error) {
        logger.error('could not translate the intent:', error);
        setStatusBarMessage(tMsg('intent.failed'));
        return null;
    }
    const tokens = intent?.tokens || [];
    const ignored = intent?.ignored || [];
    if (tokens.length === 0 && !intent?.board?.decision && !intent?.board?.score) {
        setStatusBarMessage(tMsg('intent.understoodNothing', { words: ignored.join(' ') }));
        return intent;
    }

    applyBoardHint(intent.board);
    // La ligne est PRÉPARÉE, pas lancée : l'utilisateur la relit. Une couche
    // qui devine ne doit pas aussi décider.
    commandTextStore.set(`s ${tokens.join(' ')}`.trim());
    setStatusBarMessage(
        ignored.length > 0 ? tMsg('intent.partly', { matched: (intent.matched || []).join(', '), words: ignored.join(' ') }) : tMsg('intent.understood', { matched: (intent.matched || []).join(', ') })
    );
    return intent;
}

/**
 * Pose sur le plateau de recherche les deux intentions qui ne sont pas des
 * jetons. Le type de décision se règle par les dés — une décision de videau
 * est une position sans dés posés — et le score par le passage money/match.
 * @param {{decision?: string, score?: string}} board
 */
export function applyBoardHint(board) {
    if (!board || (!board.decision && !board.score)) return;
    statusBarModeStore.set('EDIT');
    positionStore.update((pos) => {
        if (!pos) return pos;
        if (board.decision === 'cube') {
            pos.decision_type = 1;
            pos.dice = [0, 0];
        } else if (board.decision === 'checker') {
            pos.decision_type = 0;
            if (pos.dice[0] === 0 && pos.dice[1] === 0) pos.dice = [3, 1];
        }
        if (board.score === 'money') {
            pos.score = [-1, -1];
        } else if (board.score === 'match' && (pos.score[0] === -1 || pos.score[1] === -1)) {
            // Un score de match sans plus de précision : 3-away/3-away, le
            // score le plus banal qui ne soit ni Crawford ni money.
            pos.score = [3, 3];
        }
        return pos;
    });
}
