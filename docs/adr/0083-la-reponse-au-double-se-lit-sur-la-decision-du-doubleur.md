# ADR-0083 — La réponse au double se lit sur la décision du doubleur

Statut : acceptée.
Voir aussi : ADR-0007 (rien ne s'enregistre chez le destinataire), ADR-0001 (l'identité d'une
position par son hachage Zobrist), ADR-0019 (une seule échelle d'équité sort du moteur).

## Contexte

Une prise ou un refus s'enregistre sur la position d'après le double : le receveur au trait,
le videau tourné et sans propriétaire. C'est ainsi que les imports XG, GnuBG et BGBlitz
l'écrivent, et leur analyse est celle du doubleur, à son videau d'avant le double. Les matchs
transcrits ou joués en duel posaient la réponse sur le videau tourné *possédé par le
receveur*, c'est-à-dire sur la ligne de son propre redouble. gammonNet et le rollout
évaluaient donc le redouble du receveur, pas la prise.

## Décision

1. Une position de réponse (`gammonnet.IsResponsePosition`) s'évalue et se déroule comme la
   décision du doubleur avant le double (`gammonnet.DoublerPosition`). Les équités restent
   celles du doubleur, à l'échelle des imports. Les chances sont tournées vers le receveur au
   trait.
2. La migration 2.41.0 déplace les réponses transcrites vers le videau sans propriétaire.
   Une réponse transcrite se reconnaît à ses lignes, pas à la provenance du match : elle
   suit le Double de l'adversaire dans la même partie. Le repli d'un `.xg` sans segment de
   videau brut pose une prise seule sur la ligne du doubleur, sans Double adverse avant elle ;
   cette ligne ne bouge pas.
3. La migration supprime les verdicts gammonNet déjà stockés sur des positions de réponse.
   Ils jugent la mauvaise décision, et leur étiquette de version ne les distingue pas d'un
   bon verdict. La suppression a lieu une seule fois par bibliothèque : SQLite la garde par
   la clé `answered_doubles_pending`, PostgreSQL par la clé
   `gammonnet_response_analyses_dropped`, écrite dans la même transaction. Un verdict donné
   après la suppression est celui du doubleur et n'est plus jamais effacé. L'import d'une
   base `.db` antérieure à 2.41.0 laisse ces verdicts derrière lui, pour le même motif.

## Conséquences

- La suppression touche une base à son ouverture, y compris une base reçue d'un tiers, ce
  qu'ADR-0007 interdit en principe au destinataire. On l'accepte ici : seul un verdict
  gammonNet est supprimé, et ce verdict est faux. Le destinataire le regagne en relançant
  l'analyse, sans rien perdre de ce que le producteur a écrit. Aucune trace de l'origine ni
  aucun registre n'est touché.
- Le déplacement emporte le coup seul. Les commentaires, les collections, les cartes Anki et
  la Pile qui visaient la position restent sur l'ancienne ligne. Cette ligne est aussi le
  redouble du receveur, la même position au sens de Zobrist. Rien ne permet de savoir si
  l'utilisateur visait la prise ou le redouble, et la ligne est conservée tant qu'un de ces
  liens la tient. Ce choix est assumé.
- Les erreurs des coups déplacés sont recalculées sur la ligne d'arrivée, et les
  statistiques des matchs touchés sont recalculées.
