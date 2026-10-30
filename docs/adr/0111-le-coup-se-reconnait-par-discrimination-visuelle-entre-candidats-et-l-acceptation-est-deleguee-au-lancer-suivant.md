# ADR-0111 — Le coup se reconnaît par discrimination visuelle entre candidats, et l'acceptation est déléguée au lancer suivant

Statut : proposée ; acceptation après l'ADR-0110 livrée et sa mesure intra-sujet ; le videau
(règle 7) après le banc 09.2.
Voir aussi : ADR-0108 (cadre de la vision), ADR-0109 (événements et ligne de temps), ADR-0110
(dés et mode assisté), ADR-0044 (transcrire n'est pas jouer), ADR-0092 (aucune frappe avalée),
ADR-0082 (les Repères), ADR-0028 (hachage hors règles de session), ADR-0007 (rien côté
destinataire).

## Contexte

Les ADR-0108 à 0110 lisent les événements et les dés ; reste le coup, et le pas qui fait gagner
le plus de temps : ne plus avoir à valider ce que l'image montre sans ambiguïté. Lire une
position entière échoue (0,99³⁶¹ = 2,7 % de plateaux justes à 99 % par case) ; la liste des
coups légaux est fermée et deux candidats ne diffèrent que sur 2 à 4 points. À 320×180 un pion
fait moins de 19 px : le coup ne se lit qu'en pleine résolution. Un compte à rebours de
validation s'habitue en moins de vingt expositions ; le jet suivant, lui, doit être saisi de
toute façon. L'ADR-0044 règle 1 dit que personne ne décide : une écriture sans frappe doit
dire pourquoi elle la respecte, et laisser une trace.

## Décision

1. **Le coup se reconnaît en discriminant les candidats, à l'image seule.** Des `board_after`
   des candidats (`TranscriptionCandidates`, déduits de leurs `Steps`), seuls les points où ils
   diffèrent sont rectifiés en pleine résolution (≥ 16 px par pion, critère de l'ADR-0109) sur
   les images calmes qui suivent « mains sorties ». Chaque point se lit par un profil 1D le
   long de la pointe et un comptage par pile, contre une longueur occupée L(n) mesurée, pions
   sortis compris. Chaque candidat reçoit un score visuel ; le top-K est classé par ce seul
   score, jamais par le rang du moteur ni l'équité (ADR-0108). L'abstention se règle sur la
   précision ; aucun candidat compatible : état *incertain*, jamais le légal le plus proche.
2. **L'acceptation est déléguée au lancer suivant.** En mode automatique, le top-1 *sûr* est
   écrit quand l'événement « dés au repos » (ADR-0109) du lancer suivant est observé, jamais
   par l'écoulement d'un délai ; *à vérifier* et *incertain* ne le sont jamais. L'acceptation
   s'empile sur l'annulation et `Ctrl+Z` la défait ; elle ne déplace jamais le curseur ; elle
   cède à toute frappe (ADR-0092 règle 1) : une frappe sur l'Action en cours depuis l'arrivée
   de ses candidats (ligne, plateau, dés, `e`) suspend la délégation pour cette Action, et une
   frappe arrivée au même instant l'emporte. L'indicateur permanent (ADR-0092 règle 3) dit
   « Auto : <coup>, accepté au prochain lancer » puis « accepté par la vision ». Le mode est
   coupé par défaut et s'active **par caméra** quand sa courbe risque–couverture locale
   (prototypes de l'ADR-0110) atteint la cible ; une caméra nouvelle repart en assisté.
3. **La trace est une colonne de `move`, `vision_accepted`**, posée sur l'Action du brouillon
   et reportée sur le coup à l'enregistrement comme un Repère (ADR-0082 règle 1) : vrai si le
   coup a été écrit par acceptation déléguée sans frappe, faux sinon (un coup tapé, importé ou
   antérieur l'est en fait). Toute correction par l'utilisateur l'efface. Elle décrit la
   provenance, pas la position : hors du hachage Zobrist, flux de clés inchangé (ADR-0028).
   C'est un changement de schéma au sens de `CLAUDE.md` § Invariants (`DatabaseVersion`,
   `migrate_X_to_Y`, DDL, migration PostgreSQL, suite de contrat des deux backends). Elle
   **voyage à l'export** avec la table `move` : le producteur l'a écrite sur sa transcription,
   elle ne dit rien de qui ouvre le fichier (ADR-0007 règle 3). Le `.mat` ne la porte pas.
4. **L'ADR-0044 règle 1 est réécrite à l'acceptation**, règles 2 et 3 et ADR-0092 intacts :
   > 1. **Nobody decides.** Both sides are facts the user reports. The engine proposes
   >    candidates so the played move is picked faster than typed; it never chooses, rolls or
   >    answers a double. The user may delegate the acceptance of a play to the video reading
   >    (ADR-0111): it reports which candidate the image shows, never weighed by equity, and
   >    writes it only once the next roll is seen on the table — traced on the move, undone by
   >    `Ctrl+Z`, yielding to any keystroke. The user remains the author.

   Ce n'est pas « l'engine chooses » : l'équité n'entre ni dans le score ni dans le classement,
   l'image départage des faits, et l'utilisateur qui a délégué peut défaire chaque acceptation.
5. **Le score de l'overlay se lit en vérification seulement.** Déclenchée par le changement de
   dHash (ADR-0109), la lecture est confrontée à la machine : une divergence est signalée
   comme Incohérence, jamais corrigée ni écrite.
6. **Un coup accepté par la vision est un coup transcrit** : analysé, compté dans le
   Performance Rating, exporté (ADR-0044 règle 3). La trace sert un seul filtre de recherche,
   « matchs avec acceptation déléguée », exposé à la GUI, à la CLI et au serveur, documenté
   avec la fonction.
7. **Le videau reste au clavier sauf banc.** Sa face ne se lit pas : sa valeur se déduit des
   doubles enregistrés. Le geste de double (cube poussé vers l'adversaire) n'est détecté,
   en suggestion non bloquante, que si le banc 09.2 passe (rappel > 90 % **et** < 1 fausse
   alerte par heure) ; sinon `v`. Aucun suivi continu du cube ; le score n'est jamais écrit.

## Conséquences

- Mesures : précision du top-1 visuel en leave-one-stream-out, et séparément sur les coups
  joués qui ne sont pas le premier du moteur (garde de l'ADR-0108 règle 4) — un effondrement
  y signale un score qui s'appuie sur la plausibilité ; top-1 par hauteur de pile (≤ 5, > 5)
  et sur les sorties ; courbe risque–couverture par caméra ; taux de `Ctrl+Z` et de
  corrections après acceptation. Tant que les piles hautes ne sont pas mesurées, l'automatique
  reste coupé sur les coups qui les touchent.
- Un coup faux accepté corrompt la base en silence : le seuil se règle sur la précision et le
  filtre de la règle 6 permet de revoir les matchs concernés.
- Rejeté : validation par délai (vigilance qui s'habitue ; au plus un filet en lecture
  continue, qui reste à trancher par mesure) ; classement ou départage du top-K par le moteur
  ou l'équité (blanchit les erreurs) ; filtre « légal le plus proche » à la DGT (contraire à
  l'ADR-0044 règle 2) ; lecture de la face du videau (aucune donnée, redondante) ; suivi
  continu du cube ; écriture du score ; un mode automatique global, sans caméra (la courbe
  d'une caméra ne vaut pas pour une autre) ; une trace hors schéma, dans le JSON du brouillon
  seul (perdue à l'enregistrement).

## Garde

`TestMigrationSteps_ContinuousChain` et un test de la migration de `vision_accepted` dans
`pkg/blunderdb/database/migration_test.go` ; la colonne relue telle qu'écrite par les deux
backends dans `pkg/blunderdb/storage/storagetest/` ; son passage à l'export dans
`pkg/blunderdb/database/export_test.go` ; une correction qui l'efface dans
`pkg/blunderdb/transcript/` ; en vitest sous `frontend/src/vision/` : aucune acceptation sans
lancer suivant observé, aucune pendant une frappe sur l'Action, `Ctrl+Z` la défait, le
curseur ne bouge pas, une frappe simultanée prime.
