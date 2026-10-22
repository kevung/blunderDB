# Un transfert entre bibliothèques est un export fusionné par l'import natif, tiré par celui qui reçoit

Status: accepted.
See also: ADR-0007 (rien n'est enregistré côté destinataire), ADR-0063 et ADR-0065 (lire à
travers les tenants, n'écrire que chez soi), ADR-0096 (synchronisation incrémentale),
`pkg/blunderdb/ingest/transfer.go`.

## Context

Le chantier « plusieurs bibliothèques ouvertes » veut copier une position, un match, une
collection ou une leçon d'une bibliothèque à une autre, dans les deux sens, depuis le bureau,
la CLI et le serveur. Deux outils existent déjà et chacun tient une moitié du problème :

- l'export (`ingest.ExportSQLite`) sait **quoi** emporter : une `ingest.Selection` lue par
  `ingest.OptionsFor` depuis les options de l'export, avec ses fermetures (les membres d'une
  collection, les positions d'un match, les matchs d'un tournoi, le contenu d'une leçon) ;
- l'import natif (`ingest.DBImporter`, qui partage `MergeMatches`, `MergeCollections`,
  `MergeLessons` et `MergeDecks` avec l'import du bureau) sait **comment** fusionner : une
  position se reconnaît à son hachage Zobrist, analyses et commentaires se fusionnent, un
  match, une collection, une leçon ou un paquet déjà présent n'est pas dupliqué.

Une troisième marche, ligne à ligne de `Storage` à `Storage`, serait une troisième copie de
ces règles, et les copies de l'export ont déjà dérivé par le passé.

## Decision

1. **Un transfert est l'export de la sélection suivi de l'import natif de son résultat**,
   enchaînés par `ingest.Transfer(ctx, from, to TransferEnd, opts domain.ExportOptions, …)`
   à travers un fichier SQLite temporaire que l'utilisateur ne voit pas. Les options sont
   celles de l'export, lues par le même `OptionsFor` : le bureau, la CLI et le serveur
   copient donc la même sélection (au point 5 près) et fusionnent exactement comme un
   import de cet export. A → B puis B → A ne crée aucun doublon (testé à chaque niveau).
2. **Rien n'est écrit qui dise d'où vient la copie** (ADR-0007) : ni filigrane, ni lot
   d'import (`ImportBatches`, qui porterait le nom de la source), ni métadonnée de la source.
   Le fichier intermédiaire ne porte que `issuance.CarriedMetadataKeys`, et l'import natif
   n'en lit rien : la destination garde les siennes. `Watermark`, `Password`, `Metadata` et
   `Incremental` sont refusés (`ErrInvalid`) — l'incrémental avance un repère dans la source,
   et un transfert n'écrit rien dans sa source.
3. **Trois expositions, une seule fonction.** `Database.TransferTo(ctx, dst, opts)` pour le
   bureau et la CLI (`blunderdb transfer --from --to` avec les sélecteurs de
   `export --type database`) ; un verrou de paquet sérialise les transferts, qui seuls
   tiennent deux verrous `Database.mu` à la fois. Le serveur expose `across.transferPull`.
4. **Au serveur, un transfert est tiré, jamais poussé** (ADR-0065) : le tenant source doit
   figurer dans `X-Read-Tenants`, la destination est toujours `X-Tenant-ID`. C'est la seule
   route `across.*` qui écrit, et elle n'écrit que chez l'appelant ; chaque côté est lu ou
   écrit sous un contexte limité à son tenant (RLS). Le transfert compte comme un import du
   tenant qui reçoit (quotas, annulation à l'arrêt). Comme les autres `across.*`, la route
   est propre au serveur : `call` et le bureau n'ont qu'un tenant.
5. **Une sélection nommée est une cueillette, pas une bibliothèque.** Un export de toute la
   bibliothèque garde toutes ses positions même quand ses familles sont restreintes, pour
   rester une bibliothèque ; un transfert qui nomme des IDs doit livrer ce qui a été choisi
   et rien de plus, surtout pas les commentaires privés des autres positions.
   `ingest.Transfer` ajuste donc le plan de l'export : dès qu'une liste d'IDs est donnée,
   les positions sont celles qu'atteignent les familles retenues ; une collection part
   toujours avec ses membres ; un match nommé emporte la fiche de son tournoi, sans les
   autres matchs du tournoi. Un ID que la source ne contient pas est refusé (`ErrInvalid`)
   au lieu d'être sauté. La CLI refuse une source à un autre schéma, sondée en lecture
   seule avant de l'ouvrir : l'ouvrir migrerait une base ancienne en place.
6. **Ce que l'import natif ne prend pas, le transfert ne le prend pas** : la bibliothèque de
   filtres et l'historique de révision des paquets Anki (un paquet du même nom déjà présent
   garde son calendrier). La CLI n'offre donc ni `--filters` ni `--anki-history`.

## Consequences

- Une règle de fusion corrigée dans l'import l'est aussi pour le transfert ; une famille
  ajoutée à l'export l'est aussi, dès que l'import natif la lit.
- Le coût est un aller-retour par un fichier temporaire (`os.TempDir`) : acceptable pour des
  sélections de l'ordre de la bibliothèque, et c'est le prix de n'avoir qu'une règle.
- Le glisser-déposer entre groupes d'onglets, le lot et la protection par groupe appellent
  `Database.TransferTo` sans règle propre (voir l'amendement).

## Amendement — le poste d'échange du bureau

Le bureau échange depuis un seul endroit, le panneau « Autres bases » : pour la position
affichée, une ligne par autre base ouverte (présente, absente, présente avec une autre
analyse) et deux gestes, *Copier vers* et *Récupérer depuis*, qui sont le même transfert dans
les deux sens. Le panier de la barre d'onglets disparaît au profit d'un **lot** intégré au
panneau, à base d'arrivée explicite ; le menu contextuel (groupe *Envoyer*) y mène aussi.

- `TransferBatch` porte `TournamentIDs` : un tournoi part avec tous ses matchs, un match seul
  n'emporte que la fiche de son tournoi. La règle reste celle de `ingest.Transfer` ; un lot qui
  nomme à la fois des tournois et d'autres lignes part en deux transferts, pour qu'un match nommé
  seul y garde la fiche de son tournoi.
- Les copies de position annulables forment l'**historique de la session** (reçus de
  `CopyPosition`), repris du plus récent au plus ancien jusqu'à la fermeture ; dans une même
  base, une copie plus récente doit être reprise d'abord (`ErrLaterCopyFirst`). Chaque reprise
  garde la règle sûre de `TakeBackPositionCopy` : refus si la position ou la base a changé.
- Les copies de matchs, de tournois, de collections et de lots entrent au même historique : l'import
  natif dit ce que sa transaction a créé (`Summary.CreatedMatches`, `CreatedTournaments`,
  `CreatedCollections`, `AddedMembers`), `SealCopy` en fait un reçu à empreintes, `TakeBackCopy`
  vérifie toutes les empreintes puis retire, dans une transaction, les membres ajoutés, les
  collections, les matchs (parties d'abord, pour que la purge d'orphelins ne touche aucune position
  préexistante), les tournois, puis les positions créées. Ce que la base tenait avant reste. Une
  copie qui a créé une leçon n'a pas de reçu : retirer ses positions couperait ses étapes.
- La présence d'un match, d'un tournoi, d'une collection ou d'une leçon dans les autres bases est
  `storage.ElsewhereObject`, sous la clé où la copie fusionne (empreinte du match, nom sinon) ; le
  bureau, `blunderdb elsewhere --match/--tournament/--collection/--lesson` et
  `across.objectElsewhere` la lisent.
- Le verrou devient « Protéger des copies » et se mémorise par chemin de fichier dans les
  préférences du poste, jamais dans la base (ADR-0007).
- Le serveur n'a pas de *Copier vers* : un tenant n'écrit que chez lui (ADR-0065) ;
  `across.transferPull` couvre *Récupérer depuis*.
