# Plan d'ergonomie du gestionnaire de tournoi — 2026-10

Plan de conception, aucun code. Il prolonge [ux.md](ux.md) (budgets KLM, personas D16,
contraintes D17/D18) et part du code tel qu'il est au commit `2bcd82568`. Principe directeur :
**le TD pense « salle et tables »**, pas « épreuve et onglets ». Tout geste courant se fait en
un ou deux clics sur une grosse cible, ou en deux ou trois touches.

Conventions : un fait vérifié porte `fichier:ligne` ; **[H]** marque une hypothèse prise seul,
à confirmer. Les budgets reprennent les opérateurs de [ux.md §1](ux.md) (clic ordinaire
1,3 s ; grosse cible proche 0,66 s ; touche 0,28 s).

## 0. L'existant en une page

| Élément | Où | Ce qu'il fait aujourd'hui |
|---|---|---|
| Page Direction | `frontend/src/components/direction/DirectionView.svelte` (805 l.) | 7 onglets (`:139-146`) : direction, joueurs, arbres, emplacements, classement, historique, réglages ; onglet initial `settings` en préparation, `direction` sinon (`:124-129`) |
| Montage | `frontend/src/App.svelte:439-440` | remplace le plateau dans `.scrollable-content` (`App.svelte:499-510`, `overflow: hidden`) |
| Onglets d'épreuve | `DirectionView.svelte:459-461`, `stores/directionStore.js:436-442` | un onglet par épreuve de la Rencontre ; `switchEpreuve` change `openDirectionIdStore` |
| Grille des tables | `TableGrid.svelte` (234 l.) | case désactivée sans match (`:84`) ; une table d'une épreuve sœur n'affiche que son nom (`:56`, champ `elsewhere`) et n'est pas cliquable ; `@media (max-width: 900px)` sur la fenêtre (`:152`) |
| Fiche de résultat | `ResultCard.svelte` (262 l.) | ←/→, Entrée (`:72-77`) ; noms à 44 px de haut (`:176`) ; changer de table par numéro (`:57`) |
| File des propositions | `ProposalList.svelte`, `services/directionKeys.js` | J/K/↑/↓/Entrée en capture sur `window` |
| Arbres | `BracketsView.svelte` (305 l.), `pkg/blunderdb/database/db_direction_brackets.go:87` | voir §1 |
| Menus contextuels | `components/ContextMenu.svelte` (104 l.) | utilisé par le plateau (`Board.svelte:286`), les onglets (`TabbedPanel.svelte:192`), la transcription (`TranscriptionPanel.svelte:1540`) ; **aucun** dans `direction/` |
| Palette | `components/CommandPalette.svelte`, `services/commandPalette.js` | Ctrl+Maj+P (`services/keyboardService.js:390`) ; sources : commandes, onglets, filtres, matchs (`commandPalette.js:1-8`) ; **Ctrl+K est l'onglet Anki** (`services/tabCatalog.js:20`) |
| API de direction | `stores/directionStore.js` | chaque appel Go prend l'id du tournoi (`:888-892`, `:937-940`) : rien n'oblige à changer d'onglet pour agir sur une autre épreuve |

Écart entre [ux.md](ux.md) et le code : les chiffres `1`–`9` qui ouvrent la fiche de la
table N (ux.md §3) ne sont pas implémentés (`directionKeys.js` ne traite que J/K/↑/↓/Entrée).

## 1. L'arbre des rencontres

### 1.1 Quand il s'affiche aujourd'hui

`Database.Brackets` (`db_direction_brackets.go:87-157`) rend, pour chaque phase, les
`Sections` du moteur Nicomaque ; une phase suisse à vies (`KindSwissLives`) ou une phase sans
section n'a qu'un tableau des vies (`:150-152`). Formats du moteur
(`backgammon-tournoi@v0.4.1/format.go:10-14`) :

| Format | Section en graphe | Vue actuelle dans « Arbres » |
|---|---|---|
| `swiss_lives` | non | tableau des vies (joueur, vies, bilan, adversaires) |
| `gsl` | oui après tirage (blocs) | tableau des vies avant, colonnes après |
| `lives_bracket`, `bracket` (simple, consolante, dernière chance, double élimination) | oui après tirage | colonnes de boutons par tour |
| `round_robin` | oui (poules), barrage sans graphe | colonnes par ronde ; barrage = une phrase |

### 1.2 Pourquoi l'utilisateur ne le voit pas « quand c'est possible »

1. **Il faut aller le chercher.** « Arbres » est le 3ᵉ onglet sur 7 ; rien ne s'y ouvre seul,
   même quand la phase courante bascule sur un tableau (`DirectionView.svelte:124-129` ne
   choisit qu'entre `settings` et `direction`).
2. **Ce n'est pas dessiné comme un arbre.** `BracketsView.svelte:118-138` pose des colonnes de
   boutons (`columns()`, `:44-51`) sans traits de liaison : un tableau à 16 ressemble à une
   liste. Le moteur sait pourtant dessiner le graphe (`render.BracketBoardSVG`,
   `render/render.go:82`), utilisé par la page de l'épreuve (`render.go:594-616`) mais pas
   dans l'application.
3. **Avant le tirage, rien.** Une phase non tirée affiche « pas tiré » (`BracketsView.svelte:73`) :
   le TD d'un suisse → tableau ne voit pas l'arbre à venir, même vide.
4. **Le clic ne fait rien d'utile.** `openBracketMatch` (`DirectionView.svelte:390-393`)
   ignore le match et revient à l'onglet Direction : ni fiche, ni correction.
5. **La page murale de la Rencontre n'a pas d'arbre.** `direction/wallpage.go:47-95` ne
   rend que les tables et un lien par épreuve ; l'arbre n'est que sur la page de l'épreuve.

### 1.3 Proposition

- **Arbre dessiné, interactif** : `BracketsView` dessine en SVG inline (même composant, pas de
  dépendance) les colonnes et les traits vainqueur → place suivante ; consolante à droite du
  principal, poules en grille de résultats croisés (A×B), GSL en blocs de 4. Le suisse garde
  le tableau des vies (pas d'arbre : l'appariement n'est pas un graphe).
- **Squelette avant tirage** [H] : si `Config` de la phase (`BracketPhase.Config`) donne la
  taille, dessiner les places vides grisées. Demande au backend de rendre des sections vides
  pour une phase non tirée (à vérifier côté Nicomaque : issue amont si le moteur ne sait pas).
- **Clic sur une place** = la fiche de résultat (`ResultCard`) si le match court, la fiche de
  correction (`CorrectionPanel.svelte`) s'il est fini ; menu contextuel §2.3 sinon.
  `openBracketMatch` reçoit le match et ouvre la fiche en surimpression sur la place.
- **Ouverture automatique** : quand la phase courante a au moins une section en graphe, la
  pastille de l'onglet « Arbres » s'allume et `Ctrl+Maj+D` bascule Direction ↔ Arbres [H] ;
  dans la vue Salle (§4) l'arbre de l'épreuve est un panneau repliable à droite.
- **Arbre projeté** : la page murale de la Rencontre (`wallpage.go`) ajoute, sous les tables,
  l'arbre de chaque épreuve dont la phase courante est un tableau (`BracketBoardSVG`, déjà
  traduit par `CatalogLabeler`, `direction/labeler.go:108`). Rotation automatique
  tables / arbres toutes les N secondes (réglage de la Rencontre) [H].

Fichiers : `BracketsView.svelte`, `DirectionView.svelte` (`openBracketMatch`),
`db_direction_brackets.go` (squelette), `direction/wallpage.go`, `db_rencontre_page.go`,
locales `direction.bracket.*`. Budget : corriger un résultat d'arbre **2 clics** (place, nom)
contre 4–5 (ux.md §4.3, F8) ; voir l'arbre **0 clic** en phase tableau (pastille + vue Salle).

## 2. Menus contextuels

Un seul composant (`ContextMenu.svelte`) ; ouverture par clic droit, **touche Menu** ou
**Maj+F10** sur l'objet focalisé (aucune de ces touches n'est traitée aujourd'hui ; à ajouter
dans `ContextMenu` ou un `contextMenuService.js` partagé [H]). Chaque entrée porte son
raccourci ; le menu s'ouvre sur l'objet, pas au centre. Les actions rares (forfait, remarque)
quittent le bouton ⋯ de la fiche pour le menu, qui existe partout.

### 2.1 Case de table (`TableGrid.svelte`)

| Action | Appel existant (`directionStore.js`) | Remarque |
|---|---|---|
| Saisir le résultat | `enterResult` | ouvre la fiche |
| Forfait de A / de B | `enterForfeit` | |
| Changer de table… | `moveMatchToTable` | |
| Échanger avec la table… | **absent** | deux `TableChangedEvent` en une transaction : nouvelle méthode `SwapTables` sur `Database` [H] |
| Annuler ce match | `cancelMatch` | confirmation |
| Transcrire ce match | `transcribeFromSlot` | |
| Historique de A / de B | `history` + filtre | ouvre Historique filtré |
| Table hors service / remise en service | `SetRencontreTableOutOfService` (`db_rencontre.go:224`) | table libre seulement ; hors Rencontre : `EvConfigChanged` de l'épreuve [H] |
| Lancer ici la proposition sélectionnée | `confirmProposal` | table libre |

Aujourd'hui une case libre ou d'une épreuve sœur est `disabled` (`TableGrid.svelte:84`) : elle
doit devenir focalisable pour porter un menu.

### 2.2 Joueur (Joueurs, file d'attente, fiche, arbre)

Saisir le résultat de son match en cours · aller à sa table · voir son historique ·
marquer absent / présent (`makeParticipantAbsent` / `makeParticipantAvailable`) · retirer
maintenant / après son match (`withdrawParticipant`) · réintégrer (`reinstateParticipant`) ·
corriger nom, club, cote (`updateParticipant`, `updatePair`) · apparier à la main avec…
(`startMatchManually`) · « joue aussi à <épreuve>, table N » : aller là-bas.

### 2.3 Match (arbre, emplacements, classement)

Saisir / corriger le résultat (`enterResult` / `correctResult`) · forfait · annuler ·
changer de table · transcrire · rattacher / détacher un match importé
(`attachMatchToSlot` / `detachMatchFromSlot`) · ouvrir le match rattaché sur le plateau.

### 2.4 Proposition (`ProposalList.svelte`)

Lancer · lancer à la table… · changer la longueur · ignorer pour l'instant (ux.md §7 H2) ·
apparier autrement (ouvre l'appariement manuel pré-rempli) · imprimer la feuille de la ronde.

### 2.5 Ligne d'historique (`HistoryView.svelte`, props `:21`)

Corriger (`onCorrect`) · annuler (`onCancel`) · ajouter une remarque (`onNote`) · filtrer sur
ce joueur / ce match · aller à la table ou à la place dans l'arbre.

### 2.6 Écarts du lot E2 livré

Livré sur des méthodes `Database` existantes : « lancer ici la proposition sélectionnée » (case
libre ; passe par `startMatchManually` avec la table de la case, `ConfirmProposal` n'acceptant pas
de table imposée), « apparier à la main avec… » (joueur libre), « joue aussi à <épreuve>, aller
là-bas » (onglet de l'épreuve sœur), « rattacher » (emplacement sans match, un match importé
attendu par entrée), « lancer à la table… » et « changer la longueur… » (l'appariement à la main
s'ouvre pré-rempli, curseur sur le champ), « imprimer la feuille » (quand une ronde existe).
Les entrées qui agissent sont grisées pendant `busy`, comme les boutons.

Non livré, faute de méthode ou de lot [H] — aucun contournement côté frontend :

- **« Transcrire ce match » sur une case de table** : `transcribeFromSlot` prend un emplacement ;
  une `TableCell` ne porte pas le sien. Demande un lien match en cours → emplacement côté Go.
- **Hors service hors Rencontre** : seule `SetRencontreTableOutOfService` existe ; une épreuve
  isolée exige l'événement `EvConfigChanged` [H]. L'entrée n'est offerte que dans une Rencontre.
- **« Rattacher » depuis l'arbre ou le classement** : seules les places des emplacements ont un
  `slotId` ; l'arbre n'en porte pas.
- **§2.5 « aller à la table / à la place dans l'arbre »** : une ligne d'historique de match
  terminé n'a plus de table, et l'arbre n'a pas de révélation d'une place par match [H].
- **« Échanger avec la table… »** retirée : `MoveMatchToTable` échange déjà quand la table visée
  est occupée, l'entrée doublait « Changer de table… » (même `openMove`). Le lot E5 se réduit au
  glisser-déposer pointeur et à la parité CLI de l'échange.

Fichiers : `ContextMenu.svelte`, nouveau `services/directionMenus.js` (une fonction par objet
qui rend `MenuItem[]`, testable sans DOM), les cinq vues. Budget : forfait **3 → 2 clics**
(clic droit, entrée) ; changer de table 3 clics + 2 K inchangé mais sans ouvrir la fiche.

## 3. Recherche rapide

**Réutiliser la palette** (`Ctrl+Maj+P`) plutôt qu'une seconde barre : `buildPaletteItems`
(`commandPalette.js:56`) prend déjà des sources ; on ajoute une source `direction` active
quand une Direction est ouverte :

- **joueur** (toutes les épreuves de la Rencontre) → entrée « Alice — table 4 (principal),
  en cours » ; Entrée = fiche de son match en cours ; sinon sa liste de résultats éditables ;
- **n° de table** (`t4`, `4`) → fiche de la table, quelle que soit l'épreuve ;
- **épreuve** → bascule d'onglet ;
- **actions de TD** (lancer tout, imprimer la ronde, pause…) comme commandes.

Raccourci : **`/`** sur la page Direction ouvre la palette pré-filtrée sur la direction
(`/` n'est pas lié dans `keyboardService.js` [H : à revérifier contre la ligne de commande]).
**Ctrl+K reste Anki** (`tabCatalog.js:20`) : ne pas le reprendre. Classement : correspondance
floue existante (`utils/fuzzy.js`), puis joueurs en cours avant joueurs libres.

Données : une méthode Go `RencontreSearchIndex(rencontreId)` qui rend joueurs × épreuve ×
table × match en cours [H], pour ne pas faire N appels `participants()` depuis le front.

Budget : résultat d'un joueur dont on ne sait pas la table : **`/`, 3 K, Entrée, → , Entrée
≈ 2 s** contre ≈ 8 s aujourd'hui (onglet d'épreuve, Joueurs, filtre, retour Direction, case).

## 4. La vue « Salle »

### 4.1 Multi-épreuves

Une vue **Salle**, onglet à gauche de ceux des épreuves quand une Rencontre est ouverte :
**une case par table physique**, quelle que soit l'épreuve, couleur par épreuve (pastille +
nom court), et **la fiche de résultat fonctionne sur toutes** puisque l'API prend l'id de
l'épreuve (`directionStore.js:888-892`). Sous la grille : les propositions de toutes les
épreuves, groupées, chacune « Lancer ». Le résumé par épreuve (déjà dans les onglets,
ADR-0056 §5) reste en tête.

Données : un `RencontreTableGrid(rencontreId)` Go qui fusionne les `TableGrid` des membres
avec l'id d'épreuve par case [H] ; la page murale a déjà cette fusion
(`db_rencontre_page.go:130-178`), à factoriser.

Écart avec ADR-0056 §5 (« pas d'écran partagé : 370 px utiles n'en portent pas deux ») : la
Salle n'est pas deux épreuves côte à côte, c'est **une** grille de salle ; elle suppose la
pleine largeur (§4.3). **Tranché** : ADR-0056 §5 admet la vue Salle.

### 4.2 Placer et échanger

- **Glisser-déposer** une case occupée sur une case libre = changer de table ; sur une case
  occupée = échanger (`SwapTables`, §2.1) après une ligne « Table 3 ↔ Table 7 ? [Échanger] ».
- **Glisser un joueur** de la file d'attente sur une case libre = appariement manuel
  commencé (le second joueur se choisit ensuite) ; une proposition sur une case = la lancer là.
- **Clavier** : sur une case focalisée, `M` (déplacer) puis flèches ou numéro, Entrée ; `X`
  (échanger) idem. Flèches entre cases (grille ARIA `role="grid"`), Entrée = fiche.
  Chiffres `1`–`9` / deux chiffres = table N (dette d'ux.md §3, à livrer ici).
- Accessibilité : glisser-déposer par pointeur (`pointerdown`/`pointermove`), pas l'API HTML5
  DnD, qui interfère avec le drag-drop de fichiers de Wails (`DisableWebViewDrop` reste
  `false`, `internal/gui/run.go`) [H à éprouver].

### 4.3 Mise en page

- **Pleine largeur** : en mode TD, le panneau ancré se replie en bande (titre + boutons), la
  vue prend toute la zone. Un **défilement par onglet** (chaque vue a son `overflow: auto`
  propre et garde sa position au changement d'onglet), au lieu du défilement unique de la
  page.
- **Container queries** : remplacer `@media (max-width: 900px)` (`TableGrid.svelte:152`) par
  `container-type: inline-size` sur la vue ; la grille passe de lignes à 3, 4, 6 colonnes
  selon la largeur de la **zone**, pas de la fenêtre (le dock latéral la réduit).
- **Jetons « mode TD »** dans `style.css` : `--td-target: 44px` (cible minimale, ≥ 40 px),
  `--td-gap`, `--td-cell-min: 140px` ; actifs sous `.direction-view` seulement. Les cases
  passent de `min-height: 62px` (`TableGrid.svelte:168`) à 2 × `--td-target`. Typographie
  inchangée (ADR-0008 ; ux.md §7 H7 écartée : c'est la cible qui grossit).

### 4.4 Question ouverte : plein écran dédié

Aujourd'hui la Direction vit dans la zone du plateau (ADR-0047, conséquences ; `App.svelte:439`).
Option A — **garder la zone**, panneau replié (moins de code, cohérent avec le reste).
Option B — **mode plein écran TD** (`F11` ou bouton) : barre d'outils et panneau masqués,
`WindowFullscreen` de Wails, sortie par Échap long / F11 [H]. B sert la salle (écran portable
de 13 pouces, 1366 px) ; A suffit sur un poste de bureau. Recommandation : **A d'abord** (lot
E4), B mesuré au premier tournoi réel (#380) avant d'être construit.

## 5. Budgets de clics avant / après

| Flux | Avant (gestes) | Avant | Après (gestes) | Après |
|---|---|---|---|---|
| Résultat, table de l'épreuve ouverte | case, nom | 2 clics | idem, ou `4` `→` Entrée | 2 clics / 3 K |
| Résultat, table d'une épreuve sœur | onglet épreuve, case, nom | 3 clics + attente de rejeu | Salle : case, nom | **2 clics** |
| Résultat d'un joueur, table inconnue | onglet, Joueurs, filtre (3 K), lire la table, Direction, case, nom | 5 clics + 3 K ≈ 8 s | `/`, 3 K, Entrée, `→`, Entrée | **0 clic, 7 K ≈ 2 s** |
| Forfait | case, ⋯, forfait de X | 3 clics | clic droit, forfait de X | **2 clics** |
| Échanger deux tables | 2 × (case, ⋯, changer, N, Entrée) — et une table occupée entre-temps | 6 clics + 4 K | glisser 3 sur 7, [Échanger] | **1 glisser + 1 clic** |
| Déplacer vers une table libre | case, ⋯, changer, N, Entrée | 3 clics + 2 K | glisser | **1 glisser** |
| Corriger un résultat vu dans l'arbre | Arbres, place (→ onglet Direction, rien), Historique, ligne, ⋯, corriger, nom | 6–7 clics | Arbres, place, nom | **3 clics** (2 si Salle) |
| Marquer un joueur absent | Joueurs, filtre, ligne, bouton | 3 clics + K | clic droit sur le nom (partout), absent | **2 clics** |
| Voir l'historique d'un joueur | Historique, filtre (K) | 1 clic + K | clic droit, historique | 2 clics |
| Voir l'arbre en phase tableau | onglet Arbres | 1 clic | panneau de la Salle | **0** |

Les specs Playwright de ux.md §6 reçoivent ces nouveaux plafonds.

## 6. Lots ordonnés

Chaque lot est livrable seul, avec sa documentation (`doc/source/manuel.rst`,
`raccourcis.rst` et leurs huit `.po`, CLAUDE.md du dépôt) et ses specs de budget.

| Lot | Contenu | Fichiers principaux | Dépend de |
|---|---|---|---|
| **E1** Arbre utile | clic sur une place = fiche / correction ; pastille de l'onglet Arbres | `DirectionView.svelte:390`, `BracketsView.svelte`, `CorrectionPanel.svelte` | — |
| **E2** Menus contextuels | `directionMenus.js`, Menu / Maj+F10 dans `ContextMenu`, cases libres focalisables, §2.1–2.5 | `ContextMenu.svelte`, `TableGrid.svelte`, `ProposalList.svelte`, `HistoryView.svelte`, `PlayersView.svelte` | — |
| **E3** Clavier de la grille | chiffres = table N, flèches entre cases, `M`/`X` | `directionKeys.js`, `TableGrid.svelte` | E2 |
| **E4** Mise en page TD | jetons, container queries, un défilement par onglet, panneau replié | `style.css`, `DirectionView.svelte`, `TableGrid.svelte`, `App.svelte` | — |
| **E5** Échange de tables | glisser-déposer pointeur + parité CLI de l'échange (le menu n'y a plus part, §2.6) | `db_direction_result.go`, `TableGrid.svelte` | E2, E4 |
| **E6** Recherche | source `direction` de la palette, `/`, `RencontreSearchIndex` | `commandPalette.js`, `CommandPalette.svelte`, `keyboardService.js`, `db_rencontre.go` | E2 |
| **E7** Vue Salle | `RencontreTableGrid`, grille fusionnée, propositions groupées ; ADR-0056 §5 amendé | `DirectionView.svelte`, nouveau `HallView.svelte`, `db_rencontre_page.go` | E4, E5 |
| **E8** Arbre dessiné | SVG inline avec traits, poules en croisé, squelette avant tirage | `BracketsView.svelte`, `db_direction_brackets.go` | E1 |
| **E9** Arbre au mur | arbres des épreuves en phase tableau sur la page de la Rencontre, rotation | `direction/wallpage.go`, `db_rencontre_page.go` | E8 |
| **E10** Plein écran (option B) | seulement si #380 le demande | `App.svelte`, `internal/gui` | E4, mesure |

Ordre : E1 et E2 d'abord (gain immédiat, aucun schéma), puis E4 (préalable visuel de la
Salle), E3, E5, E6, E7 ; l'arbre dessiné (E8, E9) en parallèle dès E1. Aucun lot ne touche le
schéma ni le hash Zobrist ; E5 ajoute des événements du moteur existant, pas un type nouveau.

## 7. Parité et invariants

- Toute action nouvelle côté Go (`SwapTables`, `RencontreTableGrid`, `RencontreSearchIndex`)
  vit sur `Database` et, pour les écritures, a sa sous-commande CLI si la CLI en reçoit
  (aujourd'hui la CLI de direction est en lecture seule, ADR-0056 conséquences) ; l'exposition
  serveur est traitée par [../plan-headless-transcription-direction-2026-10.md](../plan-headless-transcription-direction-2026-10.md).
- Rien n'est stocké de l'état dérivé (ADR-0047 §3) : la Salle et l'index de recherche sont
  rejoués.
- Un échange de tables sur une table occupée par une épreuve sœur est accepté et signalé,
  jamais refusé (posture d'ADR-0044, ADR-0056 §3).

## 8. Hypothèses à confirmer

1. Maj+F10 et la touche Menu ne sont captés par aucun autre service (à vérifier dans
   `keyboardService.js`).
2. `/` est libre sur la page Direction (la ligne de commande s'ouvre par `:`, ux.md §3).
3. Le moteur sait rendre les sections d'une phase non tirée, ou l'accepte comme issue amont.
4. Le glisser-déposer par pointeur ne réveille pas le drag-drop de fichiers de Wails.
5. Un échange = deux `TableChangedEvent` suffit ; sinon un événement `TablesSwapped` à demander
   à Nicomaque.
6. ~~La vue Salle amende ADR-0056 §5~~ — tranché : ADR-0056 §5 réécrit, une grille de salle
   est admise.
