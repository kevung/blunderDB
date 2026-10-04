# Écarts consolidés — simulation 2026-10 (T1, T2, T3)

Une ligne par manque, même vu dans plusieurs tournois. Ids d'origine : `T1-E…` ([T1.md](T1.md)),
`T2-E…` ([T2.md](T2.md)), `T3-E…` / `T3-C…` ([T3.md](T3.md)), `C-H…`
([outils/README.md](../outils/README.md)). Toutes les mesures sont à l'échelle 100 %.
Personas : P1 Sophie, P2 Yanis, P3 Léa, P4 Hélène, P5 les joueurs. Chemins vérifiés dans le
dépôt (grep). Chaque écart porte un **critère** de correction vérifiable. « Issue proche » : issue du label `tournoi` qui traite le même sujet ; une issue
**fermée** veut dire que le cas mesuré ici lui a échappé ou a régressé. Une issue GitHub par écart (titre « Sim T — E<n> … », labels `tournoi` + `frontend`/`backend`) ; table en fin de fichier.

## Bloquant

**E1 — Tab avec le focus sur body quitte la Direction pour la Recherche.**
T1-E2 (et C-H2 qui y mène). P4, P1 dès qu'elle touche Tab.
Voulait repartir par Tab après « Lancer » ; la page bascule sur l'onglet Recherche sans message ;
fait Ctrl+Y, J, 11 Maj+Tab ; 13 frappes au lieu de 1.
Suspects : `frontend/src/services/keyboardService.js` (`isFocusOnBoard` l. 142, Tab l. 411),
`frontend/src/components/direction/ProposalList.svelte`.
Issue proche : #435 (fermée : Tab dans la Direction, mais pas quand le focus est tombé sur body).
**Critère** : avec le focus sur body dans la Direction, Tab garde l'onglet Tournois actif et place le focus sur le premier élément de la Direction ; test vitest de `keyboardService` + e2e « Tab après Lancer ».

**E2 — Un tournoi ne s'ouvre qu'au double-clic, jamais au clavier.**
T1-E1, C-H4 ; vu aussi en T2 (1 hésitation) et T3 (×2). P4 bloquée ; P1, P2, P3 paient le
double-clic sans indice.
Voulait ouvrir la ligne ; les lignes n'ont ni `tabindex` ni touche, Entrée sur ✎ renomme ;
80 Tab en vain puis repli souris.
Suspects : `frontend/src/components/panels/PanelTable.svelte` (`ondblclick`, l. 247),
`frontend/src/components/TournamentPanel.svelte` (`onActivate`).
**Critère** : une ligne de tournoi est atteignable par Tab et Entrée l'ouvre (Entrée sur ✎ continue de renommer) ; un clic simple sélectionne avec un indice « double-clic ou Entrée pour ouvrir » ; e2e clavier seul : créer → ouvrir → Diriger sans souris.

**E3 — Un joueur des deux épreuves d'une Rencontre lancé dans deux matchs à la fois.**
T3-E1 (1er passage sur 3, journal 05:01:26). P3, P5.
Hugo Bastide en quart de A ; la salle propose et lance son quart de B (sans table).
Suspects : `pkg/blunderdb/direction/service/hall.go`, `pkg/blunderdb/direction/room.go`,
`pkg/blunderdb/direction/service/rencontre.go`,
`frontend/src/components/direction/HallView.svelte`.
Issue proche : #446 (fermée : « un joueur en match dans une épreuve n'est pas proposé dans
l'autre »).
**Critère** : service : un Participant en match dans une épreuve n'apparaît dans aucune proposition de la salle pour une épreuve sœur, et `StartMatch` d'une telle proposition est refusé ; test Go rejouant le cas Hugo Bastide (quart de A en cours, quart de B proposé).

**E4 — La salle lance un match sans table, absent de la page murale.**
T3-E2 (3 passages sur 3). P3, P5 (4 « je joue où ? » en V1, les seules de T3).
Toutes tables prises : la salle propose B « 7 pts » sans table et « Lancer » le démarre ;
Léa attendait « attendre une table ». `ConfirmAllProposals` écarte déjà ces propositions
(`ReasonWaitingTable`, `pkg/blunderdb/direction/service/acts.go`) ; le « Lancer » unitaire de la
salle, non.
Suspects : `pkg/blunderdb/direction/service/hall.go`, `pkg/blunderdb/direction/service/tables.go`,
`pkg/blunderdb/direction/service/rencontre_page.go`, `HallView.svelte`.
Issue proche : #445 (fermée : une table ne porte qu'un match), #437.
**Critère** : la salle ne propose pas « Lancer » pour un match sans table (il reste en file « attend une table », comme `ConfirmAllProposals`) ; test Go sur `hall.go` : 8 tables prises, aucune proposition lançable de B ; la page murale ne contient aucun match en cours sans table.

**E5 — N26 : aucun moyen de repêcher quand un qualifié de poule se retire.**
T3-E3 (3 passages). P3.
Voulait mettre le 3e de la poule à la place du retiré ; la file ne propose que « Phase
suivante », la place devient une exemption (variante moteur) ; « Apparier à la main » ne touche
pas aux places d'un tableau. Oracle = app seulement en variante moteur.
Suspects : moteur `backgammon-tournoi` (règle N26 ouverte),
`frontend/src/components/direction/PlayersView.svelte`, `ProposalList.svelte`.
Issue proche : aucune (règle N26 non tranchée).
**Critère** : une voie de l'interface remplace le qualifié retiré par le suivant de sa poule avant le tirage (ou le moteur le propose selon la règle N26 tranchée) ; T3 rejoué : oracle = app en variante `repechage`.

**E7 — « Clore le tournoi » depuis la file laisse l'état stocké à `running`.**
Constat T2, devenu T2-E6 ; même chose dans la base de T3 (les deux épreuves). P1, P2, P3.
Vérifié : `direction.state = running` après clôture ; la vue dit `finished: true` ; l'en-tête
affiche « en cours » au-dessus de « Rouvrir ». Cause : la proposition passe par
`confirmAt` → `Direction.Apply(EvFinished)`, qui n'écrit pas `rec.State` ; seul
`Direction.Finish` (bouton du Classement) le fait.
Suspects : `pkg/blunderdb/direction/service/acts.go` (`confirmAt`),
`pkg/blunderdb/direction/direction.go` (`Apply`, `Finish`),
`frontend/src/components/direction/DirectionView.svelte` (`direction.state.*`, l. 709).
Classé bloquant : l'état stocké est faux (`tournament list`, la fiche du tournoi et l'en-tête
disent « en cours » d'un tournoi clos), ce n'est pas une gêne d'usage.
**Critère** : après « Clore le tournoi » confirmé depuis la file, `direction.state = finished` en base, `ListDirections` et `GetDirection` rendent `finished`, l'en-tête n'affiche plus « en cours » ; test Go dans `pkg/blunderdb/direction/service` (ConfirmProposal d'un `ActFinish`), et la même chose après `Reopen` puis nouvelle clôture.

## Ergonomie

**E6 — Focus perdu sur body après chaque Lancer, chaque résultat, la création d'un tournoi.**
C-H2, T1-E3, T3-E8, T2 (mesure). P1, P2, P3, P4. T1 : 38 ; T2 : 127 ; T3 : 108 (100/100 dans la
salle). Au clavier, le focus reste sur la case après un résultat (bien).
Suspects : `ProposalList.svelte`, `ResultCard.svelte`, `HallView.svelte`, `TableGrid.svelte`,
`TournamentPanel.svelte` (`panel-new`).
**Critère** : après « Lancer », « Tout lancer », la saisie d'un vainqueur et la création d'un tournoi, `document.activeElement` est dans la Direction (proposition suivante, case de table ou ligne créée), jamais body ; le mètre de T1/T2/T3 compte 0 `focusLost`.

**E8 — La page murale ne se trouve pas : « Ouvrir dans le navigateur », « dossier de sortie ».**
T1-E7, T2-E3. P1, P2, P5. Rien ne dit « page murale » ; aucun message du fichier écrit ni de son
chemin. T2 : Matchs → Arbres → Réglages, 6 clics, 2 hésitations, 16 interruptions évitables ;
un directeur qui ne la trouve pas reste en V0 (T1 : 73 au lieu de 4).
Suspects : `DirectionView.svelte` (`openPageFromHeader`),
`frontend/src/components/direction/DirectionSettings.svelte`.
Issue proche : #386, #448 (fermées : la page existe ; c'est sa découverte qui manque).
**Critère** : le bouton s'appelle « Page murale » (ou équivalent) dans l'en-tête de la Direction et de la salle ; après écriture, un message donne le chemin du fichier ; P2 la trouve en 1 clic sans hésitation dans T2 rejoué.

**E9 — La page murale ne dit ni « exempt », ni « éliminé », ni « qualifié » en fin de phase.**
T1-E6, T2-E5, T3-E10. P5. « Est-ce que je joue ? » non évités en V1 : T1 4, T2 21, T3 13.
Suspects : `pkg/blunderdb/direction/wallpage.go`, `pkg/blunderdb/direction/page.go`,
`pkg/blunderdb/direction/service/rencontre_page.go`.
Issue proche : #457 (la décision « écran des joueurs » en dépend).
**Critère** : la page murale écrit « exempté(e) — entre au tour N », « éliminé(e) » et « qualifié(e) » en fin de poule ou de suisse ; test Go sur le HTML de `wallpage.go` et `rencontre_page.go` ; V1 « est-ce que je joue ? » = 0 dans T1-T3 rejoués.

**E10 — Forfait et retrait confirmés par un bouton rouge « Supprimer ».**
T2-E1, T3-E7. P2, P3. Le retrait est réversible, le forfait ne supprime rien ; le forfait et le
retrait sont deux parcours (T2 : 10 clics, 9 frappes).
Suspects : `frontend/src/components/direction/ResultCard.svelte` (l. 92),
`PlayersView.svelte` (l. 157) — `confirmAction` sans libellé ;
`frontend/src/components/WarningModal.svelte` (l. 44, `confirmLabel || common.delete`).
**Critère** : le forfait et le retrait se confirment par un bouton à leur nom (« Déclarer forfait », « Retirer »), non rouge pour le retrait réversible ; le forfait propose de retirer aussi le joueur dans la même fiche ; test vitest des libellés de `confirmAction`.

**E11 — Corriger un ancien résultat : deux écrans pour rien, puis une ligne sans le perdant.**
T2-E4, T2-E2. P2. Le « Corriger » du bandeau ne vise que le dernier match, le Classement est en
lecture seule ; l'Historique écrit « X gagne » sans adversaire, il faut filtrer. 8 clics,
6 frappes, 2 hésitations.
Suspects : `frontend/src/components/direction/HistoryView.svelte` (`resultNoScore`, l. 122),
bandeau `direction-last-correct`.
Issue proche : #436 (fermée : « Corriger » dans l'Historique, fait ; la découverte manque).
**Critère** : chaque ligne de résultat de l'Historique nomme les deux joueurs ; le Classement (ou la fiche joueur) mène à la correction d'un résultat ; T2 rejoué : correction d'un ancien résultat en ≤ 4 clics, 0 hésitation.

**E12 — Lancer ou ouvrir une table au clavier : 62 Tab ; K/J et ← → sans indice à l'écran.**
T1-E5. P4 (62 Tab, 62 encore à 150 %), P1 cherche les raccourcis.
Suspects : `DirectionView.svelte` (ordre DOM), `ProposalList.svelte`, `TableGrid.svelte`.
**Critère** : depuis l'onglet Direction, la tête de file est atteinte en ≤ 3 Tab, et les raccourcis K/J et ← → sont affichés à l'écran ; e2e clavier P4.

**E13 — Une nouvelle Direction s'ouvre sur l'onglet de la précédente (Classement, « Clore » à côté).**
T1-E4. P1, P4. 1 clic ou 12 Maj+Tab de plus.
Suspect : `DirectionView.svelte` (onglet mémorisé hors tournoi).
**Critère** : une Direction ouverte pour la première fois s'ouvre sur Joueurs (préparation) ou Direction (en cours), jamais sur l'onglet d'un autre tournoi ; test vitest de `DirectionView`.

**E14 — Rechargement ou relance : la Direction est fermée et tout l'état perdu.**
T3-E4, T2 (reprise après SIGKILL). P3, P2. 6 clics (+3 crans) par rechargement en T3 ; 5 clics
après relance en T2.
Suspects : `frontend/src/components/TournamentPanel.svelte`,
`frontend/src/stores/directionStore.js` (`openDirectionIdStore`, l. 135), `DirectionView.svelte`.
**Critère** : après rechargement ou relance, la dernière Direction ouverte se rouvre sur la même épreuve et le même onglet ; e2e : reload → 0 clic pour revenir.

**E15 — Changer d'onglet de l'app fait quitter la salle et ferme la fiche ouverte.**
T3-E5. P3. Retour sur « Suisse B / Direction » ; 2 clics + 1 frappe.
Suspects : `DirectionView.svelte` (`hallOpen` local, l. 393), `directionStore.js`
(`epreuveTabsStore`).
**Critère** : changer d'onglet de l'app puis revenir rend la salle (ou l'épreuve) et l'onglet quittés ; e2e T3 « onglet Stats puis Tournois » → salle.

**E16 — Préréglage suisse_tableau : la suisse est sautée sans un mot quand Σ vies ≤ cible.**
C-H5, T3-E6. P3. À 8 joueurs, la file propose d'emblée « Phase suivante » ; il faut deviner
que « Bascule » est la cible. Non reproduit à 16 joueurs (T1).
Suspects : `directionStore.js` (préréglage, `target`), `DirectionSettings.svelte`.
**Critère** : si Σ vies initiales ≤ cible de bascule, les Réglages préviennent (« la suisse serait sautée ») avant le premier lancement, et le libellé dit ce qu'est la cible ; test vitest du préréglage à 8 joueurs.

**E17 — Le panneau Rencontre est au fond des Réglages d'une épreuve, hors écran et en cibles de 22 px.**
T3-E9, T3-C1. P3. Hors écran aux deux viewports (10 à 28 crans, 104 au total sur T3) ; rien
depuis la salle.
Suspects : `frontend/src/components/direction/RencontrePanel.svelte`, `DirectionSettings.svelte`.
**Critère** : le panneau Rencontre est atteignable depuis la salle et visible sans défilement à 1366×768 ; ses cibles mesurent ≥ 24 px ; T3 rejoué : 0 cran automatique pour la Rencontre.

**E18 — Cibles encore sous 24 px à l'échelle réelle.**
T1-E9, T2, T3-C1 (reste de C-H1). P1, P2, P3, P4. « Diriger » 103×20, « Ouvrir la direction »
106×20, « ← » 20×12, « Corriger » d'Historique 65×23 (panneau Rencontre : E17).
Suspects : `TournamentPanel.svelte` (`.direction-btn`), `HistoryView.svelte`.
**Critère** : toutes les cibles cliquées par T1-T3 mesurent ≥ 24 px de haut à 100 % (`meter.js`, aucun problème « cible … < 24 »).

**E19 — À 150 %, « Lancer » masqué par le dock.**
T1-E11 (passage préparatoire, non rejoué). P4.
Suspects : `DirectionView.svelte` (hauteur du panneau), mise en page du dock.
Issue proche : #440 (fermée : grille à 768 px de haut, pas à 150 %).
**Critère** : à 150 % (911×512), « Lancer » et les cases de table ne sont masqués par aucun élément (`elementFromPoint` au centre) ; e2e T1 P4@150 avec un lancement.

**E25 — « Ouvrir la direction » referme la Direction qu'un double geste vient d'ouvrir.**
C-H3 (relevé en écrivant le harnais, non rejoué par les specs). P2 (novice qui double-clique).
« Diriger » ouvre la Direction ; le bouton voisin la referme si on clique avant que la vue,
chargée à la demande, n'apparaisse.
Suspect : `frontend/src/components/TournamentPanel.svelte` (`toggleDirection`).
**Critère** : un second clic pendant le chargement de la Direction ne la referme pas (« Diriger » et « Ouvrir la direction » ignorent le clic tant que la vue charge, ou fusionnent en un bouton) ; e2e : double-clic sur « Diriger » → Direction ouverte.

## Confort

**E20 — Aucun retour après inscription, retardataire, Phase suivante, Tirage, Clore, page murale.**
C-H6, T1-E8, T2, T3-C2. P1, P2, P3, P4. Inscriptions sans retour : 16/16 + 8/8 (T1), 32/32
(T2), 24/24 (T3) ; le retardataire de T2 n'apprend rien de son entrée ; « Lancer » de la salle
sans retour 35/54.
Suspects : `PlayersView.svelte`, `ProposalList.svelte`, `HallView.svelte`.
**Critère** : inscription, retardataire, Phase suivante, Tirage, Clore et écriture de la page murale produisent un retour visible et `aria-live` (« Sophie Martin inscrite — 16 inscrits ») ; le mètre compte 0 « sans retour » pour ces actions.

**E21 — CSV du classement : colonne « Phase » vide.**
T1-E10. P1.
Suspect : `pkg/blunderdb/direction/service/standings.go` (`StandingsCSV`, l. 143).
**Critère** : la colonne « Phase » du CSV porte la phase atteinte par chaque joueur ; test Go sur `StandingsCSV`.

**E22 — « Enregistrer » des Réglages masqué en son centre par un champ à 1920×1080.**
T3-C3 (2/2, sonde `elementFromPoint` : `input.input[tab-content]`). P3. À vérifier à l'œil :
la sonde bascule de viewport en cours d'action.
Suspect : `DirectionSettings.svelte`.
**Critère** : à 1920×1080, `elementFromPoint` au centre d'« Enregistrer » des Réglages rend ce bouton ; vérifié à l'œil et par la sonde.

**E23 — Le filtre des Joueurs d'une épreuve persiste sans indice.**
T3-C4. P3.
Suspect : `PlayersView.svelte`.
**Critère** : un filtre actif des Joueurs est signalé (puce « filtre : … » avec ×) ou remis à zéro au changement d'épreuve ; test vitest de `PlayersView`.

**E24 — La salle liste toutes les propositions de A avant celles de B.**
T3-C5. P3. B attend derrière les poules.
Suspects : `HallView.svelte`, `pkg/blunderdb/direction/service/hall.go`.
**Critère** : la salle intercale les propositions des épreuves (par ancienneté d'attente ou par ronde), B n'attend pas la fin des poules de A ; test Go sur l'ordre de `hall.go`.

## Synthèse par persona

Gestes = clics + frappes + crans de défilement (dont imposés), toutes actions du persona.

| Persona | Total de gestes | Pire friction | Blocages |
|---|---|---|---|
| **P1 Sophie** (T1) | 448 (82 clics, 366 frappes dont ≈ 350 de saisie, 0 cran) | le silence : 25 actions sans retour (E20), focus perdu 38 fois (E6) | E7 : le tournoi clos par la file reste « en cours » |
| **P2 Yanis** (T2) | 878 (234 clics, 644 frappes dont 592 d'inscription, 0 cran) | forfait + retrait : 19 gestes, deux écrans, deux « Supprimer » (E10) | E7 |
| **P3 Léa** (T3) | 839 (235 clics, 497 frappes, 107 crans dont 104 imposés) | rechargement : Direction fermée, 6 clics + 3 crans pour revenir (E14) | E3, E4, E5, E7 |
| **P4 Hélène** (T1, clavier) | 532 (2 clics forcés, 530 frappes dont 292 Tab) | 80 Tab en vain pour ouvrir un tournoi, 62 pour « Lancer » (E2, E12) | E1, E2 (contournés hors clavier ou par raccourci) |
| **P5 joueurs** (T1-T3) | 335 interruptions sans page murale, 58 avec | « est-ce que je joue ? » : 38 des 58 restantes (E9) | E4 : match lancé sans table, introuvable au mur (E3 au 1er passage) |

## Harnais (pas des écarts du produit)

- **H1 — UIScale.** `outils/e2e/realBackend.js` rendait `UIScale: 1` au lieu d'un pourcentage :
  interface à 50 %, tailles de C-H1 et premières mesures de T2/T3 fausses. Corrigé
  (`UIScale: 100`), T2 et T3 rejoués (T1-H1).
- **H2 — `meter.tabTo`** ne simule pas le retour au début de page après le dernier élément
  (Chromium sans interface laisse le focus sur body) ; contourné par Maj+Tab dans T1 (T1-H2).
- **H3 — `tournament verify` de T2** : `outils/t2-verify` renommé `_t2-verify` pendant le
  passage à 100 %, la spec a reçu un statut `null`. Rejoué à part sur la base conservée :
  164 événements, aucun avertissement. Réglé depuis : la spec construit l'outil par
  `buildTool('t2-verify')` (`shimProcess.js`) au lieu de supposer un binaire déjà là.
- **H4 — `repechageOffered` de T3** : l'expression `/rep[êe]ch|suivant|remplac/` trouve
  « suivant » dans « Phase suivante » : faux positif, aucun repêchage n'est proposé.
- **H5 — Coupure du shim** : la page reste vivante sans message pendant le SIGKILL ; dans
  l'application la fenêtre mourrait avec le processus.
- **H6 — `pairing: random`** : les appariements suisses ne sont pas reproductibles depuis le
  front ; la graine ne fixe que les vainqueurs (noms des incidents de T2 différents d'un passage
  à l'autre).
- **H7 — MCP** : `realBackend.js` fige `ConfigureMCPHost` ; le front logue « Error starting the
  MCP server » au démarrage, sans effet sur la Direction.

## Issues

| Écart | Issue | Classe | Label |
|---|---|---|---|
| E1 | #542 | bloquant | frontend |
| E2 | #543 | bloquant | frontend |
| E3 | #544 | bloquant | backend |
| E4 | #545 | bloquant | backend |
| E5 | #546 | bloquant | backend |
| E6 | #548 | ergonomie | frontend |
| E7 | #547 | bloquant | backend |
| E8 | #549 | ergonomie | frontend |
| E9 | #550 | ergonomie | backend |
| E10 | #551 | ergonomie | frontend |
| E11 | #552 | ergonomie | frontend |
| E12 | #553 | ergonomie | frontend |
| E13 | #554 | ergonomie | frontend |
| E14 | #555 | ergonomie | frontend |
| E15 | #556 | ergonomie | frontend |
| E16 | #557 | ergonomie | frontend |
| E17 | #558 | ergonomie | frontend |
| E18 | #559 | ergonomie | frontend |
| E19 | #560 | ergonomie | frontend |
| E20 | #562 | confort | frontend |
| E21 | #563 | confort | backend |
| E22 | #564 | confort | frontend |
| E23 | #565 | confort | frontend |
| E24 | #566 | confort | backend |
| E25 | #561 | ergonomie | frontend |

E5 relève du moteur `backgammon-tournoi` : l'issue est tenue dans blunderDB, et le dit. Le `null` de `tournament verify` (H3) est réglé par `buildTool` : pas d'issue.
