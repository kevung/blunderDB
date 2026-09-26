# ADR-0056 — Une Rencontre regroupe des Tournaments dirigés dans une même salle

Statut : acceptée.
Amende : ADR-0047 §1 (« une seule entité »).
Voir aussi : ADR-0044, ADR-0036, ADR-0005.

## Contexte

Un festival de week-end fait jouer un principal, un speed et des doubles dans la même salle,
par le même directeur. Chaque épreuve est un Tournament avec sa Direction, et chaque moteur
croit avoir la salle à lui. La simulation (S3, `tasks/nicomaque/simulation-2026-09/rapport/S3.md`)
l'a mesuré : un seul tirage du speed pose 7 matchs sur les 7 tables du principal, 28 collisions
sur le week-end ; une joueuse dans deux matchs à la fois ; aucune vue ne croise les épreuves ;
changer d'épreuve ferme l'autre ; trois dossiers de page murale ; les doubles deviennent des
« personnes » « A / B » dans l'annuaire.

## Décision

Une **Rencontre** regroupe des Tournaments dirigés qui partagent une salle. Le Tournament reste
une seule entité, avec au plus une Direction ; il peut appartenir à au plus une Rencontre.
Plusieurs Directions par Tournament restent écartées.

1. **Une entité, pas un regroupement dérivé.** Tables, pauses, page murale et « un joueur n'est
   pas à deux tables » sont des faits sur la salle : il leur faut un propriétaire. Table
   `rencontre` (nom, dates, nombre de tables, dossier de sortie), `tournament.rencontre_id`
   nullable.
2. **Chaque journal reste complet à lui seul.** Un geste de salle (table hors service, pause) se
   déclare une fois et s'écrit en un `EvConfigChanged` dans chaque Direction membre, dans une
   seule transaction. Les tables occupées par les épreuves sœurs ne sont jamais stockées : elles
   sont rejouées à chaque proposition et passées au moteur (Nicomaque N22). Les réservations
   propres à une épreuve restent dans sa configuration.
3. **Le nom est le lien.** Deux Participants de même nom dans deux épreuves d'une Rencontre sont
   la même personne pour la disponibilité, avec la règle qui relie déjà un Participant à ses
   Matchs et l'annuaire à ses lignes. Un joueur qui joue ailleurs n'est pas proposé (N21) ;
   apparié à la main, il est accepté et signalé « joue au principal, table 4 », jamais refusé
   (posture d'ADR-0044).
4. **Une paire est deux personnes.** Une épreuve en doubles inscrit des Participants à deux
   membres (nom, club, cote chacun) ; le libellé « A / B » est dérivé ; la cote d'entrée est la
   moyenne, corrigeable. Le Match porte « A / B » comme Player de chaque côté : une paire décide
   ensemble, ses erreurs sont les siennes. L'annuaire éclate la paire en deux lignes ; la
   disponibilité teste chaque membre.
5. **Ouvrir la Rencontre ouvre toutes ses épreuves.** Un onglet par épreuve dans l'en-tête de la
   Direction, avec son résumé (propositions, matchs en cours, alerte) ; changer d'épreuve est un
   clic, sans confirmation, sans rien fermer ni rejouer. Pas d'écran partagé : 370 px utiles
   n'en portent pas deux. Un tournoi hors Rencontre ne voit aucun changement.
6. **Un dossier de sortie.** La Rencontre écrit `<dossier>/index.html`, la page murale de la
   salle (une ligne par table, quelle que soit l'épreuve ; les rondes annoncées ; un lien vers
   chaque épreuve), et chaque épreuve rattachée écrit dans `<dossier>/<épreuve>/`. Le dossier
   propre de l'épreuve est gardé et ignoré tant qu'elle est rattachée. Un geste dans n'importe
   quelle épreuve régénère la page murale.

## Conséquences

- Cycle de vie : créer une Rencontre puis y créer ou y rattacher des Tournaments dirigés (un
  Tournament sans Direction n'a pas de salle et n'y entre pas). Rattacher aligne les tables de
  l'épreuve par un `EvConfigChanged` montré dans « ce qui va changer » ; permis à tout moment,
  même épreuve lancée. Détacher est permis à tout moment : l'épreuve garde son journal et ses
  tables. Supprimer une Rencontre détache ses épreuves et passe par la corbeille (ADR-0036).
- Schéma 2.25.0, un seul bump pour la Rencontre et les membres de doubles, sur les trois côtés
  (`database`, `storage/sqlite`, `storage/postgres`), migration testée, base de démo régénérée.
- La logique vit sur `Database` et le contrat `Storage` ; `pkg/blunderdb/direction` reste le
  seul paquet qui connaît Nicomaque et calcule tables occupées et disponibilités.
- CLI en lecture seule, comme pour la Direction : `tournament list` nomme la Rencontre,
  `tournament page --rencontre` écrit la page murale. Le démon ne reçoit rien (ADR-0047).
- Vocabulaire : **Rencontre** dans `CONTEXT.md` ; Participant et Directory complétés pour les
  doubles.
- Écartés : un regroupement dérivé (aucun propriétaire pour la salle) ; un journal propre à la
  Rencontre (une épreuve ne se rejouerait plus seule) ; une liste mutable des tables hors
  service (perd l'historique, contredit ADR-0047 §3) ; un lien explicite entre Participants
  (la fiche personne que l'ADR-0047 écarte) ; un Match à quatre noms (refonte du domaine,
  qui a deux joueurs partout) ; un écran partagé.

## Garde

À écrire avec D6.2 : `pkg/blunderdb/direction/` rejoue S3 sans collision ni joueur à deux tables.
