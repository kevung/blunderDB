# Étape 3 — les scénarios Go (S1-S5) : ce qui a été joué

Exécuté le 2026-09-24 (ancre de la dernière génération : 2026-09-24T03:49:58+02:00, sur main eebb62b0f ; résultats identiques à la génération précédente sur 8855b6b04). Tout ce qui suit a été
**exécuté** ; les chiffres viennent de `scenarios-go.json` (même dossier), des journaux exportés
et de `open-timings.txt`. Rien n'a été modifié dans `pkg/`, `internal/`, `frontend/`.

## 0. Comment c'est fait

- **Code** : `scratchpad/scenarios/` (module Go jetable `simscen`, `replace` vers une worktree
  jetable `sim/scenarios`). `sim.go` = horloge commune + directeur simulé ; `s1.go`…`s5.go` = les
  scénarios ; `incidents.go` = les gestes I1-I6 ; `fixtures.go` = la base gabarit.
- **Moteur et Store réels** : chaque Direction est jouée par `direction.Create/Enter/Apply/
  EventFor/ProposeAt/Finish/Reopen` sur `db.DirectionStore()` d'une vraie SQLite
  (`database.NewDatabase()` + `OpenDatabase`). Résultats tirés par **`sim.PGain`** (cotes), durées
  par **`sim.Duree`** (8 min/pt), cotes par **`sim.Champ(P, 6, 2, 2, 10)`** : les trois fonctions
  existent sous ces noms dans `backgammon-tournoi@v0.2.1/sim`.
- **Le directeur simulé** fait ce que `ConfirmAllProposals` fait (tout ce qui est proposé, à UN
  instant, sauf `finish` et `waiting_table`), avec quatre retenues explicites : rien n'est lancé
  (ni tiré) hors des heures de salle ; rien pendant une pause, ni un match dont la fin attendue
  dépasse de plus de 20 min le début d'une pause ; rien qui implique un joueur « retenu » (I6) ;
  en S3, rien sur une table ou avec un joueur occupés dans une autre épreuve.
- **Ancre** (outillage § 3.2) : chaque base est jouée deux fois — une passe à blanc sur un Store
  mémoire donne l'instant de l'étape (ou, pour la fin, du dernier événement) ; la vraie passe est
  décalée de `ancre − cet instant`, `Config.Breaks` compris. L'étape tombe donc 2 min avant la
  génération ; pour une base de fin, c'est le dernier événement. Les bases intermédiaires sont
  des **rejeux arrêtés** à l'étape, jamais des copies.
- **Écart d'API à connaître** : `direction.SetConfig` et `Database.SetDirectionConfig` horodatent
  à `time.Now()` (Enter/Finish/Reopen prennent un `now`). Pour garder une chronologie simulée, le
  générateur fait ce que fait `SetConfig` à l'heure du scénario : `cfg.Validate()` puis
  `Apply(ConfigChangedEvent(cfg, now))`. Même raison pour les autres méthodes de `Database`
  (toutes à `time.Now()`) : chaque geste est joué par l'événement que la méthode écrit, et le
  rapport nomme la méthode.
- **Données** : 166 fichiers `.xg` de `testdata/` importés par `Database.ImportXGMatch`
  (153 importés, 13 doublons refusés « duplicate match »), puis **dénommés** : joueurs
  (`UpdateMatch`), événement, lieu, fichier source, commentaires, `import_batch.source`
  (`RawConn`). 300 Players fictifs (40 habitués, 260 occasionnels), noms générés par graine
  (prénoms × patronymes forgés), aucun recoupement avec les 20+ noms réels des fixtures (vérifié à
  l'exécution). Clubs fictifs (« BC Ourcq », « Cercle du Rhône »…). Import : 4 min 22 s, une fois
  (`bases/_fixtures.db`, réutilisée).

**Régénérer tout** (bases + journaux + verify), à relancer juste avant une mesure puisque
l'ancre est l'instant de la commande :

    bash /tmp/claude-1000/-home-unger-src-blunderDB/11a70712-b3d4-43bb-9be3-caf70f994c3a/scratchpad/scenarios/regen.sh

(recrée la worktree jetable si besoin, compile `scratchpad/blunderdb`, joue S1-S5 en ~10 s,
lance `tournament verify` et `tournament export` sur chaque base, écrit
`rapport-sources/verify/verify.txt`, `journaux/*.json`, `S4-*-standings.csv`.)

## 1. Les bases produites (`scratchpad/bases/`)

Toutes contiennent la base gabarit : **153 Matchs importés, 300 Players** (265 dans les bases
S2, voir S2 ci-dessous), 31 Tournaments importés, plus les Tournaments du scénario.

| Base | Heure du scénario (= ancre) | Directions : état, inscrits, événements, joués / en cours / propositions |
|---|---|---|
| S1-19h55.db | ven 19:55 | août : clos, 16, 74, 27/0/0 · septembre : en cours, 21, 30, 0/8/2 |
| S1-21h.db | ven 21:00 | août idem · septembre : en cours, 21, 54, 12/8/1 |
| S1-fin.db | sam 02:01 | août idem · septembre : clos, 21, 104, 38/0/0 |
| S2-samedi-14h.db | sam 13:59 | en cours, 51, 130, **39/0/24** (fin du repas, file pleine) |
| S2-samedi-soir.db | sam 18:02 | suisse fini, 51, 225, 86/0/1 (`next_phase` proposé, consolante pas encore ajoutée) |
| S2-fin.db | dim 17:21 | clos, 51, 280, 110/0/0 ; 28 Matchs importés rattachables |
| S3-samedi-20h.db | sam 19:59 | principal : en cours, 65, 287, 105/7/0 · speed : préparation, 32, 33, tirage proposé · doubles : préparation, 16 paires, 17 |
| S3-fin.db | dim 23:59 (dernier év. dim 17:15) | principal clos 332 év. · speed clos 121 · doubles clos 51 |
| S4-lundi-3.db | lun 3, 19:59 | en cours, 25, 81, 24/0/13 (ronde 3 proposée, absents revenus) |
| S4-fin.db | mar 15/09 12:00 | clos, 25, 181, 69 matchs |
| S5-mercredi-matin.db | mer 08:59 | en cours, 51, **311**, 129/0/8 (dernier événement mardi soir) |
| S5-fin.db | clos mer 23:35 | clos, 51, 355, 148 matchs |

- **S1** contient le tournoi du « mois dernier » (« Soirée du club — août », 16 inscrits, joué
  en entier, date = ancre − 28 j) : `DirectorySources` le liste, `DirectoryEntrants` rend ses 16
  inscrits dans l'ordre.
- **S2** : 28 Matchs importés portent les noms de 28 paires qui ont joué et sont rangés dans le
  Tournament (`UpdateMatch` + `SetMatchTournamentByName`) : `UnattachedMatches` en rend 28, dont
  28 avec un Slot suggéré (O20 mesurable). Renommer ces matchs fait tomber les Players à 265.
- `bases/_fixtures.db` = la base gabarit ; `bases/demo.db` n'est pas de cette étape.

## 2. `tournament verify`, journaux, Open

`blunderdb tournament verify` sur **chaque Direction de chaque base** (19 journaux) :
**« no warning » partout, code de sortie 0** (`rapport-sources/verify/verify.txt`). Journaux :
`rapport-sources/journaux/<base>-t<id>.json` (`tournament export`).

Temps d'`Open` (rejeu complet), mesuré à part, 200 fois par journal, charge 0,9
(`open-timings.txt`) :

| Journal | Événements | SQLite médiane / p95 / max | Mémoire médiane |
|---|---|---|---|
| S2 fin | 280 | 3,0 / 3,3 / 4,1 ms | 2,4 ms |
| S3 principal (double élim. 64) | 332 | 7,0 / 7,7 / 8,4 ms | 6,4 ms |
| S4 fin | 181 | 1,2 / 1,8 / 2,6 ms | 0,9 ms |
| S5 fin | 355 | 3,6 / 3,8 / 4,0 ms | 3,0 ms |
| réplique de `BenchmarkOpen` | 318 | — | 2,9 ms |

`go test -bench BenchmarkOpen ./pkg/blunderdb/direction/` sur la même machine : 3,45-3,73 ms/op.
**Budget 50 ms tenu partout, d'un facteur 7 au moins.** Le principal S3 coûte 2,4× le benchmark
à taille égale : le profil (`open-S3.prof`) met `PhaseState.resolve` en tête (24 % du temps).
Les mesures prises *pendant* la génération (`scenarios-go.json`, champ Timings) donnaient 16-34 ms
de médiane et 61 ms de max sur S3 : la machine était alors chargée (charge ≈ 9), elles ne
servent pas de référence. Réouverture complète (Close + OpenDatabase + rejeu) mesurée pendant
les I5 : 7-38 ms.

Test de rejeu (§ 3.3) : `scratchpad/replay_tests/` — quatre journaux (S2, S3 principal, S4, S5)
rejoués sans avertissement, à l'identique deux fois, classement final = classement rejoué,
Open < 50 ms ; **0,04 s en tout** (`go test -short` idem). Laissé dans le scratchpad.

## 3. S1 — le soir de club (Marc, puis Yanis)

**Déroulé joué.** 20 joueurs (16 repris du mois dernier par l'annuaire + 2 Players connus + 2
inconnus), 8 tables, préréglage `suisse_tableau` aux longueurs du scénario (suisse 5, tableau 5,
finale 7, bascule 16). 19:45 : 8 matchs lancés d'un coup. Bascule vers le tableau à **22:41**,
clôture à **02:01** (salle prévue jusqu'à 23:30) : avec 20 joueurs, la bascule à 16 garde un
tableau de 16 places, 38 matchs en tout. Le mois dernier (16 joueurs, même format) avait fini à
01:20. `EntrySuggestions` rend 300 Players ; 2 des 4 « nouveaux » y sont (PR pré-rempli).
`DirectorySources` place en tête le tournoi **en préparation** (0 inscrit) avant « août ».

| Incident | Voulu | Geste API (méthode à l'écran) | Accepté | Après |
|---|---|---|---|---|
| I2 19:50 | retardataire | `PlayerAddedEvent` (AddParticipant) | oui | 8 matchs lancés, aucune place d'exemption (suisse) : entre avec ses 2 vies, `Infos` vide |
| I1 20:40→20:59 | joueur parti sans prévenir | `PlayerWithdrawnEvent` immédiat (WithdrawParticipant) | oui | le moteur l'apparie à 20:58 (M19) ; retrait → M19 perdu par forfait, rang 21 `forfeit` |
| I6 21:05 | absent 30 min, sans quitter | **aucun événement** : le TD ne confirme pas ce qui l'implique | — | le moteur continue de l'apparier ; « Tout lancer » inutilisable pendant 30 min |
| I3 21:20 | Yanis corrige un vainqueur inversé par Marc à 20:20 | `CorrectionEvent` (CorrectResult) | oui | 12 résultats depuis ; la vraie gagnante, saisie perdante, était **éliminée depuis une heure** (D2, 0 vie) et revient à 1 vie, sans avertissement ni information ; 0 warning |
| I4 21:40 | table 5 cassée | `ConfigChangedEvent(unavailable+=5)` (SetDirectionConfig) puis `TableChangedEvent(M23, 1)` (MoveMatchToTable) | oui, oui | `PreviewDirectionConfig` rend **changes = null** (bug G1) |
| I5 22:00 | coupure | Close + OpenDatabase + `direction.Open` | oui | rejeu 12 ms ; classement, propositions, avertissements identiques |
| clôture 02:01 | Yanis clôt | `Finish` | oui | `verify` : no warning |

## 4. S2 — le week-end à une épreuve (Sophie, Léa)

**Déroulé joué.** 50 joueurs + 1 retardataire, 14 tables, suisse 2 vies en 7 → tableau 16 en 9
(finale 11), pauses sam 12:30-14:00 et 19:30-20:30, dim 12:30-14:00, salle sam 10:00-23:00 et
dim 10:00-18:30, dotation 30 €/inscrit, retenue 10 %, 40/25/15/10 %. Samedi 13:59 : 39 matchs
joués, 0 en cours, **24 propositions de match dans la file** (repas respecté ; 7 matchs lancés
avec l'avertissement `ends_in_break` sur tout le week-end). Suisse fini à **samedi 18:02**
(avant le dîner) ; consolante ajoutée alors, puis passage de phase et tirage. Clôture **dimanche
17:21** (prévue 18:00), 111 matchs, 280 événements. `Standings` : pool 1 530, retenue 153,
distribuable 1 377 ; **une seule section** (générale, 4 primés) : la consolante n'a pas de
classement propre faute de barème à son nom.

| Incident | Voulu | Geste API | Accepté | Après |
|---|---|---|---|---|
| I2 sam 11:15 | retardataire | `PlayerAddedEvent` | oui | 28 matchs déjà lancés ; entre au suisse avec 2 vies |
| I4 sam 16:00 | table 7 cassée | `ConfigChangedEvent(unavailable+=7)` | oui | aperçu vide (B1) ; déplacer M76 : **aucune table libre** (14/14 occupées) → le match reste sur la table cassée, rien ne le signale |
| sonde 16:00 | déplacer un match sur une table occupée | `TableChangedEvent(M59 → 11)` alors que M64 y joue | **oui** | aucun avertissement (bug G2) |
| I5 sam 17:00 | coupure | rejeu | oui | 13 ms, identique |
| F15 sam 18:02 | ajouter la consolante avant le tirage | `ConfigChangedEvent(phases[1].consolation=true)` | oui | aperçu : `consolation false→true` ; consolante construite au tirage |
| sonde 18:02 | la même chose **après** le tirage | idem, sur copie en mémoire | **oui** | aucune consolante créée, Config dit `consolation=true`, aucun refus (bug G3) |
| I6 sam 21:00 | un joueur du tableau ne revient que dim 14:00 | aucun événement (TD retient) | — | sa proposition reste dans la file ; « Tout lancer » l'aurait lancée |
| I1 sam 22:00 | départ en plein tableau 16 | `PlayerWithdrawnEvent` immédiat | oui | Valentine Chalvignac, **demi-finaliste** : M100 perdu par forfait ; placée comme perdante dans la consolante, sa place s'y résout sans match ; **rang final 14/51 `forfeit`** |
| sonde 22:00 | corriger le club de la retirée | `PlayerAddedEvent` même ID (= UpdateParticipant) | oui | **elle n'est plus retirée**, rang 1 `running` (bug G4) |
| I3 dim | vainqueur inversé en finale du principal (M105, 11:26), corrigé à 16:00 alors que le vrai vainqueur, saisi perdant, avait commencé la finale de consolante M110 | `CorrectionEvent` | oui | 1 warning `bracket_wrong_players` sur M110 **et 1 annulation proposée** ; confirmée → M110 annulé, M111 relancé avec la bonne joueuse ; 0 warning ensuite |

H11 (sonde sam 21:00, tableau 16 tiré) et H12 : § 8.

## 5. S3 — le festival à trois épreuves (Karim)

**Déroulé joué.** Trois Tournaments / trois Directions : principal 64 (double élimination :
`bracket` 7 pts, consolante + réconciliation + recharge, finale 11), speed 32 (`bracket` 3 pts,
sam 20:00-23:45), doubles 16 paires (`bracket` 5 pts, préparé samedi matin, lancé dim 10:00).
14 tables partagées. Speed = **28 joueurs du principal + 4 extérieurs** (le plan dit « 40 des 64
jouent aussi le speed », incompatible avec un speed de 32). Doubles : 20 joueurs du principal +
12 autres ; une paire = un Participant « A / B », cote = moyenne des deux, calculée à la main.
Clôtures : speed **dim 00:02**, doubles dim 15:13, principal **dim 17:15** (finales prévues 15 h).

**La salle partagée, mesurée** (le moteur de chaque Direction croit avoir 14 tables à lui) :
- **28 collisions** : un match proposé par une Direction sur une table occupée par une autre ;
  toutes résolues par un lancement puis un déplacement (30 `TableChangedEvent`, dont 2 induits par
  un déplacement précédent dans la même file) ;
- 3 appariements distincts retenus faute de table libre dans la salle ; **12 appariements
  distincts retenus parce qu'un joueur jouait dans une autre épreuve** — rien ne le dit à
  l'écran : il faut ouvrir l'autre Direction ;
- à 19:59, **6 des 32 inscrits au speed jouent au principal** ;
- changements de Direction entre deux lancements consécutifs : sam 20 h : 5, 21 h : 4, 22 h : 6 ;
  dim 10 h : 1, 11 h : 4, 14 h : 3 (plancher : les résultats saisis n'y sont pas comptés).

| Incident | Voulu | Geste API | Accepté | Après |
|---|---|---|---|---|
| I2 sam 10:20 | retardataire au principal (tableau 64 tiré) | `PlayerAddedEvent` | oui | 0 place libre ; `Infos = no_entry` : inscrit, n'entrera nulle part |
| I3 sam 15:30 | vainqueur inversé à 14:24, vu 1 h plus tard | `CorrectionEvent` | oui | warning `bracket_wrong_players` sur M59 + 1 annulation proposée, confirmée, relance ; 0 warning ensuite |
| I4 sam 21:00 | deux tables cassées (3 au principal, 12 au speed) | 4 `ConfigChangedEvent` : 3 et 12 dans **chacune des trois** Directions (3+12 principal, 12+3 speed, 3,12 doubles) + `TableChangedEvent(M115, 3→1)` | oui ×5 | aperçu vide (B1) ; une table cassée se déclare autant de fois qu'il y a d'épreuves |
| I1 sam 21:00 | un joueur des deux épreuves part | 2 × `PlayerWithdrawnEvent` (une par Direction) | oui ×2 | principal : pas de match en cours, rang 64 ; speed : M13 perdu par forfait |
| I5 sam 22:00 | coupure, deux épreuves en cours | rejeu des Directions | oui | 38 ms, identique |
| I6 sam 23:00 | un joueur du principal n'arrive dimanche qu'à 11:00 | aucun événement | — | proposition retenue jusqu'à 11:00 |

H10 : § 8. Annuaire après le festival : 85 lignes dont **16 « A / B »** (une paire y devient une
personne). Identifiant d'une paire : `jasmine-jambardel-lison-olivarès-pell`.

## 6. S4 — le championnat de club en rondes (Nadia)

**Déroulé joué.** 24 joueurs (+1 en semaine 2), 6 lundis, 6 tables, 7 pts, `swiss_lives`
`mode=rounds`, **6 vies** (choix de l'exécutant, le plan ne dit pas le nombre : avec 2 vies, la
moitié du club serait éliminée à la ronde 3). Lancement à 20:00 chaque lundi seulement.

- **Sonde lundi 1 (bug G5)** : la ronde 1 propose 6 matchs avec table et 6 `waiting_table` ;
  lancer les 6 premiers (ce que fait « Tout lancer ») → la file devient `wait matches_running` :
  les 6 autres appariements **disparaissent** ; quand la 1re vague finit, le moteur propose
  **12 matchs de ronde 2** à tout le monde, alors que 12 joueurs n'ont pas joué la ronde 1.
  Contournement retenu : lancer les 12 d'un coup, dont 6 sans table (`ConfirmProposal` d'une
  proposition `waiting_table`) ; leur durée enregistrée couvre deux vagues → la bande d'horloge
  affiche **63,5 min/pt observées** (prévu 8) à la base S4-lundi-3.
- **Match en retard** : 2 matchs reportés le lundi 1, saisis jeudi 18:00. Lundi 23:59, file =
  `wait matches_running` : **la ronde 2 n'est pas proposée tant qu'ils ne sont pas saisis.**
- **Annoncer la ronde 2 le vendredi** (H2) : 12 appariements proposés, non lancés ;
  `PairingSheet(0)` imprime le dernier lot **lancé** (Rounds() = 1, la ronde 1). Aucune feuille de
  la ronde de lundi sans la lancer (ce qui horodaterait ses matchs au vendredi).
- **Sonde bye seul (bug G6)** lundi 2 : ronde 2 = 12 matchs + bye(Octave) ; confirmer le seul bye
  puis redemander → le moteur propose **12 matchs de ronde 3 + un bye de ronde 3**, l'exempté
  réapparié. `ConfirmAllProposals` n'y est pas exposé (une liste, un instant) ; la confirmation
  ligne par ligne l'est.
- Byes : la ronde 4 en a donné 3 (25 joueurs, un appariement retenu, revanches évitées), la 6 : 3.
- Clôture mardi 15/09 : le moteur proposait encore la **ronde 7** ; il n'a pas de nombre de
  rondes, `Finish` est un geste manuel (accepté).

| Incident | Voulu | Geste API | Accepté | Après |
|---|---|---|---|---|
| I5 chaque lundi 19:00 | réouverture après six jours | rejeu | oui ×5 | 7-8 ms, identique |
| I2 lun 2 19:45 | nouveau joueur | `PlayerAddedEvent` | oui | entre avec 6 vies, une ronde de retard |
| I1 lun 2 20:20 | absent non prévu | `ForfeitEvent` (EnterForfeit) | oui | l'absent perd une vie (D1) |
| I6 ven 21/08 | trois absents annoncés pour lundi 3 | `PlayerWithdrawnAfterCurrentEvent` ×3 | oui | pas de match en cours → retrait **immédiat** ; au classement du mardi : rangs 23-25 `forfeit` |
| I6 lun 3 19:30 | leur retour | `PlayerAddedEvent` même ID ×3 (à l'écran : UpdateParticipant ; AddParticipant créerait un nouvel identifiant) | oui | victoires, défaites, vies, byes **conservés** (ex. V2 D0 6 vies → identique), rang 1, 7, 8 |
| I3 lun 4 19:40 | résultat de la ronde 2 faux | `CorrectionEvent` | oui | 22 résultats depuis, 0 warning, 0 réparation (suisse) |
| I6 lun 4 | absent prévu, rien fait | aucun événement, match non lancé | — | après la ronde, la proposition devient un match de **ronde 5** ; il ne rattrape pas la ronde 4 |
| I6 lun 5 | absent prévu, forfait | `ForfeitEvent` | oui | lui coûte une vie |
| I4 | — | sans objet | — | — |
| F21 mar 15/09 | clore après 6 rondes | `direction.Finish` (CloseDirection) | oui | propositions juste avant : ronde 7 |

Classement du mardi sans le portable : `blunderdb tournament standings --db S4-lundi-3.db --id 32`
fonctionne (CSV `;`, en français : « en vie (6 vies) ») — si le fichier de base est ailleurs que
sur le portable. Le CSV ne dit que les vies, pas les victoires.

## 7. S5 — une épreuve sur cinq jours (Sophie)

**Déroulé joué.** 50 joueurs (+1 mardi), 10 tables, 11 pts, sessions 9:00-23:00, pauses
12:30-14:00 et 19:00-20:30 chaque jour, `swiss_lives` continu **3 vies**, `batch_minutes = 20`.
Chaque soir 23:30 : base fermée puis rouverte (I5 « normal »). **Le suisse s'épuise en deux jours
et demi** : 129 matchs joués mardi soir (base S5-mercredi-matin : 311 événements au lieu des
~200 prévus), 8 joueurs en vie mercredi 12:00, clôture **mercredi 23:35**. Les incidents du jeudi
et du vendredi sont **sans objet** (tournoi clos) : retour de l'absent jeudi, table 4 jeudi 11:00.

| Incident | Voulu | Geste API | Accepté | Après |
|---|---|---|---|---|
| I6 lun 22:00 | absent mar-mer, « je reviens jeudi » | `PlayerWithdrawnAfterCurrentEvent` | oui | retrait immédiat (pas de match), rang 50 `forfeit` ; tournoi clos avant jeudi : **finit `forfeit`** |
| I5 ×4 | fermeture du soir | rejeu | oui | 29-35 ms (sous charge), identique |
| I2 mar 10:00 | retardataire | `PlayerAddedEvent` | oui | 78 matchs lancés ; entre avec **3 vies** au milieu de joueurs qui en ont perdu |
| I3 mer 10:00 | résultat de mardi 15:05 faux | `CorrectionEvent` | oui | 40 résultats depuis ; la nouvelle perdante tombe à **0 vie alors qu'elle joue M130** ; aucun warning, aucune réparation ; elle gagne M130 et son adversaire perd une vie contre une joueuse éliminée (bug G7) |
| F15 mer 12:00 | tableau final décidé en cours | `ConfigChangedEvent(lives 3→2, target 8, + phase lives_bracket 11/13)` | **oui** | aucun refus ; la somme des vies restantes reste 11 (les vies d'entrée ne changent pas) ; bascule quand elle tombe à 8 : passage et tirage **mer 16:07** |
| I1 mer 16:00 | départ définitif du meilleur (8 victoires, rang 1) | `PlayerWithdrawnAfterCurrentEvent` (il jouait) | oui | finit son match, puis retiré ; **rang final 50/51 `forfeit`** (H15) |

Bande d'horloge (`Database.Clock`) à la fin : `elapsedSeconds = 225 464` → ClockBar affiche
**« 62 h 37 »** ; la Direction close annonce encore une `nextBreak` (la pause du jeudi, placée
dans le futur par l'ancre). À la base S5-mercredi-matin, `nextBreak` = mercredi 12:30, affichée
« pause à HH:MM » sans jour.

## 8. Hypothèses de lots.md § 4

| # | Verdict | Observation exacte |
|---|---|---|
| H2 | **CONFIRMÉE** | `tournoi.Config` n'a ni jour, ni session, ni date, ni nombre de rondes (name, phases, min_per_point, tables, prizes, breaks). S4 vendredi : 12 appariements de la ronde 2 proposés ; `PairingSheet(0)` imprime la ronde 1 (Rounds() = 1). Clore après 6 rondes : geste manuel, le moteur proposait la ronde 7. |
| H3 | **CONFIRMÉE** | Aucun événement d'absence. Trois chemins joués en S4 : retrait (différé = immédiat sans match) puis `PlayerAddedEvent` même ID — acquis conservés, mais l'absent est `forfeit` au classement pendant l'absence ; ne rien faire — son appariement glisse à la ronde suivante ; forfait — une vie perdue. |
| H9 | **CONFIRMÉE** | `blunderdb tournament --help` : list, verify, standings, page, export ; aucun sous-commande n'écrit d'événement (`page --out` écrit seulement le dossier de sortie dans la base). Un résultat reçu par message ne se saisit qu'à l'écran. `standings` marche sur S4-lundi-3.db. |
| H10 | **CONFIRMÉE** | S3 : 16 paires = 16 Participants « A / B » ; `Player{ID,Name,Club,Rating}` sans champ de paire ; cote = moyenne calculée à la main ; `EntrySuggestions` ne propose aucune paire (0) ; l'annuaire compte 16 lignes « A / B » sur 85. |
| H11 | **CONFIRMÉE** | S2 sam 21:00, tableau 16 tiré : `FreeSlots` = 0 ; inscription acceptée ; `Infos = [no_entry]`. S3 : tableau 64 plein à 10:20 → `no_entry`. Au suisse (S1, S2, S5), le retardataire entre avec toutes ses vies. |
| H12 | **CONFIRMÉE** | S2 sam 22:00, demi-finaliste retirée : match en cours perdu par forfait ; placée comme perdante dans la consolante (`conso/conso.3.0`), place résolue sans match ; rang final 14/51 `forfeit`. |
| H13 | **CONFIRMÉE, et plus large** | Il n'existe **aucune fin estimée** : ni `ClockView` ni `ClockBar.svelte` n'en calculent (`sim.Forecast` n'est appelé nulle part). Le temps écoulé s'affiche en heures seules (« 62 h 37 » en S5, 336 h à S4-lundi-3) ; la pause suivante en « HH:MM » sans jour (S3 sam 20 h : la pause de dimanche 12:30) ; une Direction close annonce encore une pause. |
| H15 | **CONFIRMÉE** | S5 : 8 victoires, rang 1 au départ mercredi 16:00, **rang final 50/51 `forfeit`** (ex æquo avec l'absente qui n'a pas pu revenir). Même chose en S2 : demi-finaliste → 14/51. |
| H16 | **CONFIRMÉE** (avec un contournement accepté) | `Validate()` de `swiss_lives lives=3 target=16` : « phase 0 : la bascule vers un tableau à vies suppose 2 vies ». Mais sur le suisse **commencé**, `lives 3→2 + target 8` est accepté sans refus (les vies d'entrée restent à 3) et la bascule a lieu. |

## 9. Bugs et constats (aucun corrigé ; reproductions minimales)

- **G1 — « Ce qui va changer » vide pour une table cassée** (S1, S2, S3). `PreviewDirectionConfig`
  (config courante + `tables.unavailable=[5]`) rend `changes: null` : `diffConfig`
  (`db_direction_config.go`) compare `tables.count`, ni `unavailable` ni `reserved`.
- **G2 — deux matchs en cours sur une table** (S2, moteur). `Apply(TableChangedEvent(M59, 11))`
  alors que M64 joue table 11 → accepté, aucun warning ; `MoveMatchToTable` écrit cet événement.
- **G3 — consolante cochée après le tirage : acceptée, sans effet** (S2, moteur). Rejeu jusqu'au
  `draw` du tableau 16 ; `ConfigChangedEvent(phases[1].consolation=true)` → accepté,
  `Config.Phases[1].Consolation=true`, `Sections` reste `[main]` ; aucun refus ni warning.
- **G4 — corriger la fiche d'un retiré le réinscrit** (S2). `UpdateParticipant` écrit
  `PlayerAddedEvent` même ID ; le moteur fait `delete(Withdrawn, id)` → la retirée redevient
  rang 1 `running`, sans confirmation. (C'est aussi le seul chemin de retour d'un absent, H3.)
- **G5 — mode rounds, ronde plus grande que la salle** (S4, moteur). 24 joueurs, 6 tables :
  lancer les 6 matchs avec table → les 6 autres appariements disparaissent
  (`proposeSwissRound` rend nil tant qu'un match tourne) ; ensuite, ronde 2 pour tous.
- **G6 — mode rounds, bye confirmé seul** (S4, moteur). Confirmer le bye avant les matchs → la
  proposition suivante est une ronde N+1 où l'exempté rejoue.
- **G7 — correction au suisse qui élimine un joueur en cours de match** (S5, moteur). Après
  `CorrectionEvent(M97)`, Salomé Yvergnaux a 0 vie mais joue M130 ; aucun warning ni réparation
  (le moteur n'en propose que pour les graphes) ; M130 compte, son adversaire perd une vie.
  Symétrique en S1 : la fausse perdante, éliminée une heure à tort, revient sans que rien ne le
  dise.
- **C1** — `direction.SetConfig` / `SetDirectionConfig` n'ont pas de paramètre `now` (Enter,
  Finish, Reopen en ont) : un scénario ne peut pas passer par eux.
- **C2** — `BenchmarkOpen` appelle `b.ReportMetric(…, "events")` avant `b.ResetTimer()`, qui efface
  les métriques : la ligne n'affiche jamais « 318 events » (réplique : 318 événements, 125 matchs).
- **C3** — S1 : `DirectorySources` place le tournoi en préparation (0 inscrit) avant « le mois
  dernier ».
- **C4** — formats trop courts pour l'horaire : S1 finit à 02:01 (salle 23:30) ; S5 finit
  mercredi soir (prévu vendredi) ; S3 principal à 17:15 (finales prévues 15 h). Aucune fin
  estimée ne l'aurait annoncé (H13).
- **C5** — I4 S2 : 14/14 tables occupées, le match de la table cassée ne peut aller nulle part ;
  rien ne le propose ni ne le signale ensuite.
- **C6** — S2 : le classement « par section » ne montre que la générale tant que la consolante n'a
  pas de barème à son nom.

## 10. Écarts au plan, et pourquoi

- S1 et S4 ne listaient pas les six incidents ; S1 a reçu I4 (21:40) et I6 (21:05) ; S4 I4 est
  « sans objet ». L'I1 de S1 est joué comme « parti sans prévenir, découvert quand le moteur
  l'apparie » (une deuxième défaite à 2 vies l'aurait éliminé, sans geste).
- S3 : 28 + 4 au speed au lieu de « 40 des 64 » (incompatible avec 32). Les doubles sont préparés
  dès samedi matin pour que la base de 20 h ait les trois Tournaments.
- S4 : 6 vies (voir § 6). S5 : les incidents de jeudi-vendredi sans objet (tournoi clos mercredi).
- I5 : la coupure est une fermeture puis réouverture du fichier (Close + OpenDatabase). Un
  `kill -9` au milieu d'un geste n'a pas été joué (non mesuré : le verrou de fichier interdit deux
  ouvertures dans le même processus ; cela demande un processus fils).
- Les compteurs « retenus » de `Stats` comptent des appariements distincts, gonflés par le suisse
  continu qui réapparie à chaque événement : ils ne sont pas cités comme mesures.
