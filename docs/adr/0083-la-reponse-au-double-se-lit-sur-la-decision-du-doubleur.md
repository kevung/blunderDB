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
   bon verdict. La suppression aboutit une seule fois par bibliothèque. SQLite la garde par
   la clé `answered_doubles_pending`, que l'ouverture abaisse après le déplacement.
   PostgreSQL la garde par la clé `gammonnet_response_analyses_dropped`, écrite dans la
   transaction même de la suppression, hors de la génération des passes Go. Elle ne s'écrit
   que si la transaction voit les lignes de tous les tenants. Cette passe vient en tête des
   passes Go.
4. L'import d'une base `.db` antérieure à 2.41.0 laisse ces verdicts derrière lui, pour le
   même motif. La règle est écrite une fois (`ingest.StaleResponseVerdict`) et sert à
   l'import du bureau comme à celui du daemon.
5. Une prise ou un refus se convertit en MWC au videau d'avant le double. Sur le videau
   tourné sans propriétaire, la conversion descend donc d'un cran. Une prise laissée par le
   repli `.xg` sur la ligne du doubleur est déjà à ce videau et garde le sien. La migration
   vide les statistiques des matchs qui en portent une, pour les recalculer.

## Conséquences

- Ce qui est garanti : une fois la suppression aboutie, un verdict gammonNet écrit ensuite
  n'est plus effacé, quelle que soit la génération des passes à venir. Avant qu'elle
  aboutisse, un verdict écrit sur une position de réponse peut encore lui revenir. C'est le
  cas sous PostgreSQL si le démarrage a buté sur un verrou ou sur un rôle qui ne voit pas
  tout. Il suffit alors de réanalyser.
- La suppression touche une base à son ouverture, y compris une base reçue d'un tiers, ce
  qu'ADR-0007 interdit en principe au destinataire. On l'accepte ici : seul un verdict
  gammonNet est supprimé, et ce verdict est faux. Le destinataire en obtient un juste en
  relançant l'analyse. Aucune trace de l'origine ni aucun registre n'est touché ; ADR-0007
  cite l'exception à sa règle 3.
- Jusqu'à « analyser les manquants », les prises et refus dont le verdict a été supprimé
  sortent des statistiques, comme toute décision non analysée.
- Le déplacement emporte le coup seul. Les commentaires, les collections, les cartes Anki et
  la Pile qui visaient la position restent sur l'ancienne ligne. Cette ligne est aussi le
  redouble du receveur, la même position au sens de Zobrist. Rien ne permet de savoir si
  l'utilisateur visait la prise ou le redouble, et la ligne est conservée tant qu'un de ces
  liens la tient. Ce choix est assumé. La ligne d'arrivée est marquée comme réponse
  (`is_cube_response`), pour le filtre prise/refus de la recherche.
- Les erreurs des coups déplacés sont recalculées sur la ligne d'arrivée, et les
  statistiques des matchs touchés sont recalculées.
- Rejeté : reconnaître une réponse transcrite à la provenance du match (sans fichier
  source). Une transcription corrige aussi un match importé et garde son fichier.
- Rejeté : comparer le joueur du coup au joueur au trait. Dans les deux formes, c'est le même.
- Rejeté : supprimer les verdicts périmés d'après leur étiquette de version gammonNet. Une
  même version a produit les deux lectures, l'ancienne et la bonne.
- Rejeté : supprimer d'après la date de création du verdict. Une réanalyse qui remplace un
  verdict en garde la date.
- Rejeté : garder la suppression par la seule génération des passes PostgreSQL. Une
  génération suivante la relancerait sur des verdicts justes.
- Garde : `TestResponseRolloutIsTheDoublersDecision` (rollout) ;
  `testMatchReanchorAnsweredDoubles` (contrat des deux backends : déplacement, repli `.xg`
  immobile, drapeau de réponse) ; `TestMigrate_2_40_0_to_2_41_0_AnsweredDoubles` et
  `TestAnsweredDoubles_ResumeAndClean` (SQLite : migration, reprise, base propre, verdict
  postérieur conservé) ; `TestMigrate_AnsweredDoublesUnderRLS`,
  `TestMigrate_ResponseAnalysesDroppedOnce` et
  `TestMigrate_ResponseAnalysesDropWaitsForAFullView` (PostgreSQL) ;
  `TestImportDatabase_StaleResponseVerdicts` et `TestDBImportStaleResponseVerdicts`
  (imports du bureau et du daemon) ; `TestCubeMultiplierOfAnAnswer` (conversion MWC).
