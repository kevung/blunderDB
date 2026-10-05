# ADR-0058 — Une table porte ses propriétés, et une Rencontre se partage en salles

Statut : acceptée.
Voir aussi : ADR-0044, ADR-0047, ADR-0056 (§1, §5), ADR-0057.

## Contexte

Un open occupe deux salles : la salle A porte les tables 1 à 20, la salle B les tables 21 à 32.
Le principal joue dans les deux, le DMP et les doubles en B, le speed en A. Une table est
filmée : le directeur la renomme « Stream » et y place lui-même le match vedette ; un joueur
qui filme ses parties demande à jouer toujours à la même table. Un tournoi seul, hors de toute
Rencontre, a les mêmes besoins pour son stream. Le moteur (Nicomaque) ne connaît que des
tables numérotées, leur nombre, leurs pannes et des réservations par section ou par phase ; il
ne connaît ni salle, ni nom, ni joueur attitré, et il appartient à un autre dépôt.

## Décision

Vu de haut, il n'y a que des tables numérotées. Une table peut recevoir des **propriétés** ; la
salle en est une, au même titre que son nom et sa réservation.

1. **Un seul modèle : les propriétés d'une table.** Une ligne par table qui en a : numéro, nom,
   salle, réservée, attitrée à des personnes. Le numéro reste l'identifiant : la CLI, l'API et
   les joueurs disent « table 23 », un numéro est unique dans la Rencontre (numérotation
   continue d'une salle à l'autre). Le nom s'affiche à côté du numéro partout où une table est
   nommée : grille, propositions, page murale. Une table sans ligne est une table ordinaire.
2. **La salle est un libellé, pas une entité.** Une salle est l'ensemble des tables qui portent
   le même libellé de salle ; elle n'a pas de ligne à elle, ni d'identifiant, ni de nombre de
   tables propre. Une Rencontre où aucune table ne porte de salle a implicitement une seule
   salle qui les couvre toutes : les Rencontres existantes sont inchangées.
3. **Où vivent les propriétés.** Elles appartiennent à qui possède les tables : la Rencontre
   pour ses épreuves, le Tournament dirigé quand il joue seul. Une table de propriétés dont le
   propriétaire est l'un ou l'autre, jamais les deux. Une épreuve rattachée lit celles de sa
   Rencontre ; les siennes sont gardées et ignorées tant qu'elle est rattachée, comme son
   dossier de sortie (ADR-0056 §6), et reviennent quand elle est détachée.
4. **Les propriétés ne vont pas au journal.** Une table hors service change ce que le moteur
   rejoue, elle reste un `EvConfigChanged` dans chaque Direction (ADR-0056 §2). Nom, salle,
   réservation et attribution ne changent que ce que l'hôte **propose** : elles sont, comme les
   tables des épreuves sœurs, passées au moteur en `tournoi.External.BusyTables` au moment de
   la proposition, jamais rejouées. Un match lancé écrit son numéro de table dans le journal ;
   aucune décision passée n'en dépend, chaque Direction se rejoue seule, et changer un nom ne
   réécrit aucun historique. Le format du journal de Nicomaque n'a pas de champ pour elles :
   les y mettre exigerait un changement du moteur pour un fait qu'il ne lit pas.
5. **Les salles d'une épreuve sont un fait de la Rencontre.** La restriction « le DMP joue en
   B » se dit en noms de salle, qui n'ont de sens que dans la Rencontre ; elle est rangée avec
   l'appartenance (`tournament.rencontre_rooms`, à côté de `rencontre_id`) et s'efface au
   détachement. Dans le journal, elle obligerait à relire la Rencontre pour rejouer la
   Direction. Liste vide : l'épreuve joue sur toutes les tables.
6. **Propositions.** Le moteur ne propose une table que dans les salles de l'épreuve. Sont
   passées comme occupées : les tables des épreuves sœurs en cours, les tables hors des salles
   de l'épreuve, les tables réservées, et les tables attitrées sauf pour leurs titulaires.
7. **Table réservée** : jamais proposée ; le directeur y place un match à la main (lancement
   manuel, déplacement, glisser-déposer), et ce geste est accepté.
8. **Table attitrée** à une ou plusieurs personnes, par leur nom (le nom est le lien,
   ADR-0056 §3 ; une paire est concernée dès qu'un de ses membres l'est). Le match proposé
   d'un titulaire va sur sa table si elle est libre et dans les salles de l'épreuve ; sinon il
   reçoit une table ordinaire : l'attribution est un confort, elle ne fait jamais attendre un
   match. Hors des matchs de ses titulaires, elle se comporte comme une table réservée : la
   prêter ferait attendre le titulaire au moment où il se libère, et la caméra y est déjà
   installée ; le directeur peut toujours y placer un match à la main.
9. **Deux titulaires de tables différentes se rencontrent** : le match va sur la plus petite
   des tables attitrées libres. La règle ne dépend ni du côté ni de l'ordre des joueurs, et
   elle se prévoit ; l'autre table reste réservée, et le directeur déplace s'il préfère.
10. **Gestes.** Déplacer un match, l'échanger avec une épreuve sœur ou lancer un match à la
    main sur une table hors des salles de l'épreuve est refusé (`ErrRefused`), pour chacun des
    deux matchs d'un échange. Une table réservée ou attitrée accepte un geste explicite.
11. **Changer les salles en cours de route.** On ne peut pas retirer à une épreuve une salle
    qui porte un de ses matchs en cours, ni changer la salle d'une table qui porte un match en
    cours d'une épreuve qui n'aurait plus le droit d'y jouer : refusé, en nommant la table.
    Ajouter une salle, renommer une table, réserver ou attitrer est permis à tout moment.
12. **Les vues groupent par salle.** La grille de la Rencontre et la page murale restent
    uniques par Rencontre (ADR-0056 §5, §6) et groupent leurs tables par salle quand des
    salles existent. La vue d'interface de la grille s'appelle **Toutes les
    tables** : « salle » désigne une partie de la Rencontre, et la grille les montre
    toutes ; « Événement » désigne déjà le panneau qui la gère.
13. **L'interface dit « événement ».** La Rencontre s'appelle **Événement** dans l'interface,
    l'aide et la documentation (en : *Event*) : un événement regroupe des épreuves, et les deux
    mots ne se remplacent jamais l'un l'autre. Le nom technique reste `rencontre` dans le code,
    le schéma, la CLI (`tournament … --rencontre`) et l'API (`/v1/rencontres.*`), parce que
    `event` y est déjà pris : le flux SSE `/v1/events`, les événements du journal, `HallEvent`.

## Conséquences

- Schéma 2.28 : table `table_setting` (propriétaire Rencontre ou Tournament, numéro, nom,
  salle, réservée, personnes attitrées) et colonne `tournament.rencontre_rooms`, sur les trois
  côtés, migration testée, base de démo régénérée. La corbeille garde les propriétés d'une
  Rencontre et les salles de ses épreuves, et les rend à la restauration.
- Service : `SetRencontreTables`, `SetDirectionTables`, `SetEventRooms` ; les vues de Direction,
  de Rencontre, de la grille et de la page murale portent nom et salle de chaque table.
  Routes `/v1/` d'écriture sous `serve --direction`, avec `If-Match` (ADR-0057 règle 4).
- CLI : `tournament tables` imprime les propriétés et les salles ; `tournament hall` groupe par
  salle. Les écritures passent par `call`, comme tout geste de Direction (ADR-0057).
- Écartés : une table `room` avec ses bornes de tables (une entité de plus pour un libellé, et
  deux sources du nombre de tables) ; les propriétés dans le journal (le moteur ne les lit pas,
  et une épreuve rattachée aurait à recopier celles de la Rencontre à chaque geste) ; les
  salles d'une épreuve dans son journal (sa relecture exigerait la Rencontre) ; une épreuve
  dans plusieurs Rencontres ; prêter une table attitrée libre (le titulaire attendrait).

## Garde

`pkg/blunderdb/direction/service` : une épreuve restreinte à une salle ne reçoit aucune
proposition hors de ses tables ; un déplacement hors de ses salles est refusé ; retirer une
salle qui porte un match en cours est refusé ; une table réservée n'est jamais proposée ; le
match d'un titulaire va sur sa table libre, et sur la plus petite quand deux titulaires se
rencontrent.
