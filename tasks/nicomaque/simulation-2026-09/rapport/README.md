# Simulation de tournois dirigés — le rapport (2026-09-24)

**#380 reste ouverte : cette simulation n'a mis aucun joueur réel devant l'interface.**

Exécution du plan [../README.md](../README.md), étapes 1 à 5 : les `data-testid` (main 1462a1058),
le shim (vrai front Svelte sur vrai `Database`, vrai Nicomaque v0.2.1, vraie SQLite), les
scénarios S1-S5 joués par le moteur ([scenarios-go.md](scenarios-go.md)), la mesure Playwright aux
deux viewports, ce rapport, puis les issues.

| Fichier | Contenu |
|---|---|
| [S1.md](S1.md) | soir de club, 20 joueurs — Marc puis Yanis |
| [S2.md](S2.md) | week-end, 51 joueurs, 14 tables — Sophie, Léa |
| [S3.md](S3.md) | festival, trois épreuves, salle partagée — Karim |
| [S4.md](S4.md) | championnat en rondes, six lundis — Nadia |
| [S5.md](S5.md) | cinq jours, micro-rondes — Sophie |
| [mesures.md](mesures.md) | la grille agrégée, comparée aux budgets `ux.md` |
| [scenarios-go.md](scenarios-go.md) | l'étape 3 : ce que le moteur a joué, `verify`, `Open`, hypothèses H* |

## 1. Les lots créés

| Lot | Issues | Ce qu'il regroupe |
|---|---|---|
| **D5 — L'échelle** | #434 D5.1 molette · #435 D5.2 Tab · #436 D5.3 corriger depuis l'Historique · #437 D5.4 match manuel sans table · #438 D5.5 table hors service · #439 D5.6 fiche d'un retiré · #440 D5.7 la grille à 768 px · #441 D5.8 clore/rouvrir · #442 D5.9 import CSV · #443 D5.10 nettoyage | les bugs trouvés en route, et ce qui tient à 20 joueurs mais pas à 51 |
| **D6 — La Rencontre** | #444 D6.1 ADR · #445 D6.2 salle partagée · #446 D6.3 qui est libre · #447 D6.4 deux Directions · #448 D6.5 page murale · #449 D6.6 doubles | S3 |
| **D7 — Le championnat** | #450 D7.1 absence · #451 D7.2 feuille d'une ronde à venir · #452 D7.3 ronde plus grande que la salle · #453 D7.4 classement du mardi | S4, S5 |
| **D8 — Les sorties** | #454 D8.1 fichier · #455 D8.2 réglages de phase · #456 D8.3 bande d'horloge | S1, S2, S4, S5 |
| **D9 — Écran joueur** | #457 D9.1 cadrage, bloqué par #380 | Q6 |
| **Moteur** (`PileOfCells/backgammon-tournoi`, lot:5) | N15 #15 table occupée · N16 #16 consolante après tirage · N17 #17 ronde > salle · N18 #18 bye seul · N19 #19 correction qui élimine · N20 #20 rang d'un retiré · N21 #21 joueur indisponible · N22 #22 tables occupées ailleurs | ce qui se corrige chez l'auteur du moteur, jamais en contournement ici |

## 2. Ce que la mesure dit, en cinq phrases

1. Les gestes unitaires tiennent leurs budgets **quand leur cible est visible** : résultat 2, forfait
   3, corriger la dernière 2, reprendre un tournoi 3, rattacher 28 matchs 29.
2. **La hauteur casse tout à l'échelle** : 370 px utiles à 768 px, dock ouvert ; à 24 propositions
   la grille de 14 tables est entièrement sous l'écran (3-5 crans par geste), et **la molette ne
   défile pas** (E1). 1366×768 ne change rien.
3. **Six besoins n'ont aucun chemin** : corriger un résultat ancien, déclarer une table hors
   service, ajouter une consolante, absenter un joueur, savoir qui est libre pour une autre
   épreuve, annoncer une ronde avant de la lancer.
4. **Trois épreuves dans une salle collisionnent dès le premier tirage** : 7 matchs du speed sur
   les 7 tables du principal.
5. Le moteur est rapide (`Open` ≤ 7 ms pour 355 événements) et `verify` ne laisse aucun
   avertissement sur 19 journaux ; ses défauts sont de règle (rondes, corrections, retraits), pas
   de vitesse.

## 3. Les défauts trouvés en route

Écran (vus par la mesure) :

| # | Défaut | Issue |
|---|---|---|
| E1 | la molette au-dessus de la Direction change la position de la bibliothèque cachée (757 → 753) et ne défile pas | #434 |
| E2 | « Corriger » dans l'Historique ne fait que basculer sur l'onglet Direction : aucune fiche | #436 |
| E3 | Tab dans un champ de la Direction ouvre l'onglet Recherche | #435 |
| C1 | un match apparié à la main sans table n'occupe aucune case (écart mock / produit) | #437 |
| E6 | l'import CSV inscrit une ligne sans séparateur et les doublons, sans erreur | #442 |
| H7 | « Clore » en un clic, sans confirmation, matchs en cours | #441 |

API et moteur (vus par les scénarios Go, [scenarios-go.md](scenarios-go.md) § 9, où ils sont
nommés G1-G7) : G1 aperçu vide pour une table indisponible (#438), G2 deux matchs sur une table
(N15), G3 consolante après tirage sans effet (N16), G4 corriger un retiré le réinscrit (#439),
G5 ronde plus grande que la salle (N17), G6 bye seul (N18), G7 correction qui élimine un joueur en
match (N19).

**Hypothèses de [../lots.md](../lots.md) § 4** : H1, H2, H3, H4, H5, H6, H7, H8 (aggravée en E2),
H9, H10, H11, H12, H13 (plus large : aucune fin estimée du tout), H14, H15, H16 **confirmées** ;
aucune réfutée. H16 a un contournement accepté par le moteur (passer 3 → 2 vies en cours).

## 4. Ce qui n'a pas été mesuré, et pourquoi

- Le **nombre d'interruptions « je joue où ? »** : pas de joueurs réels (D9.1, #380).
- La **coupure en plein geste** (`kill -9`) : le shim sert une base par processus ; la coupure
  jouée est fermeture + réouverture (rejeu identique partout).
- **Deux postes** (assistante + directrice) : il n'y en a qu'un ; S2 mesure un seul portable.
- **Le temps réel** d'un geste : les secondes données sont KLM (`ux.md` § 1), calculées depuis
  les comptes, pas chronométrées.
- Le décalage entre mock et shim n'a été vu que sur « apparier à la main » (C1) ; les six autres
  flux de `direction-budgets.spec.js` rendent les mêmes comptes contre le vrai moteur.

## 5. Reproduire

Tout l'outillage est jetable et n'est pas committé (décision Q7) : le shim (`uishim/`), les specs
(`e2e/S1.spec.js` … `S5.spec.js`, `helpers/measure.js`, `ops.js`) et le générateur des bases
(`scenarios/regen.sh`) vivent dans le scratchpad de la session du 2026-09-24. Le test de rejeu
optionnel n'a pas été déposé : il vérifie des journaux produits par le scratchpad, et sans ce
générateur il n'aurait rien à rejouer.
