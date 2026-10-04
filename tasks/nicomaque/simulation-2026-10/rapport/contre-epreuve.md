# Contre-épreuve : T1-T3 rejoués après la correction des 25 écarts

Les trois tournois de la simulation ont été rejoués sur `main` une fois fermées les issues des
écarts E1-E25 (#542-#566) : même harnais (vrai front, vrai `*database.Database`, vraie SQLite,
vrai moteur Nicomaque v0.5.0), mêmes personas, mêmes incidents, feuille papier recalculée par
`outils/oracle/oracle.py` sans code de l'application. Feuilles : `../feuilles/T{1,2,3}.json`.

## 1. Résultat

| | Fini | Oracle = app | Avertissements | Particularité |
|---|---|---|---|---|
| **T1** 16 joueurs, suisse 2 vies → tableau | oui | **16/16** | aucun (Direction, page) | Hélène (P4) ouvre le tournoi **au clavier** (Tab + Entrée), sans repli souris |
| **T2** 32 joueurs, incidents | oui | **33/33** | aucun ; `tournament verify` : 164 événements, aucun | SIGKILL en pleine ronde (6 matchs en cours) : 96 événements avant et après, état identique, **Direction rouverte seule** par la session après rechargement |
| **T3** Rencontre A (poules → tableau) + B (suisse → tableau), 8 tables partagées | oui | **A : 16/16 en variante repêchage**, B : 8/8 | aucun | N26 : Léa repêche le suivant de poule depuis la file, comme elle le voulait ; aucune table à deux matchs, aucun match sans table |

La variante `n26:"moteur"` de T3 (place du retiré perdue) n'a plus d'objet : l'application
repêche, et la feuille papier se rejoue désormais jusqu'au classement final.

## 2. Interruptions des joueurs (P5), même modèle qu'au premier passage

| Tournoi | « Je joue où ? » V0 | V1 | « Est-ce que je joue ? » V0 | V1 |
|---|---|---|---|---|
| T1 | 54 | **0** (54/54 lus au mur) | 19 | **0** (4 exemptés : « exempté(e) — entre au tour 2 ») |
| T2 | 124 | **16** | 21 | **1** |
| T3 | 96 (48 matchs) | **0** (48/48, table et épreuve) | 23 | **0** |
| **Total** | 274 | **16** | 63 | **1** |

Premier passage : 58 interruptions en V1 ; contre-épreuve : **17**. Ce qui reste :

- **16 en T2** : les joueurs des 8 premiers matchs, lancés avant que Yanis n'ouvre la page —
  le script du persona la fait découvrir après 8 lancements. Une fois la page ouverte, chaque
  match lancé y figure à sa table (`V1_missing` vide). Ce reste mesure le persona, pas la page.
- **1 en T2** : la retirée elle-même ; le mur ne nomme pas un retiré, qui a quitté la salle
  (choix de `direction/status.go`). Compté par fidélité au modèle du premier passage.
- T3 : le mur nomme les 2 exemptés de B, les 2 éliminés de la suisse, les 7 non-qualifiés des
  poules de A ; les perdants de tableau se lisent dans l'arbre (non comptés en V1, comme avant).

## 3. Écart nouveau

| # | Classe | Constat | Correction |
|---|---|---|---|
| E26 | ergonomie | Après le retrait d'un qualifié de poule, la file propose « Inès Vautrin remplace gaëlle-tessier » : l'identifiant du retiré, pas son nom (la file ne connaissait que les joueurs libres) | commit `fix(direction): la file nomme le qualifié retiré qu'un repêchage remplace` — `ProposalList` reçoit les inscrits ; test `directionRepechageNames.test.js` |

## 4. Défauts de harnais corrigés (pas de l'application)

- T1/P4 visait la cellule de la ligne de tournoi ; le focus est sur la ligne (`tr[tabindex]`).
- T1 comptait un exempté « lu » si son nom figurait n'importe où ; il faut « Nom — exempt ».
- T2 testait la reprise par `isVisible({timeout})`, qui n'attend pas : il cliquait « Ouvrir la
  direction » pendant que la session la rouvrait, et la refermait.
- T2 cherchait le statut dans les 120 caractères autour de la première mention du nom ; il lit
  désormais la ligne du bloc « Est-ce que je joue ? ».
- T2/T3 attendaient l'ancien bouton « Supprimer » ; forfait et retrait se décident à leur nom.
- T3 lisait la salle par épreuve ; elle mêle désormais les épreuves (E24), chaque ligne
  préfixée du nom de la sienne.

## 5. Ce que la contre-épreuve ne prouve pas

Les mêmes limites qu'au premier passage (README § 4) : ni joueur ni directeur réels, pas de vraie
fenêtre Wails, appariements `random` non reproductibles. La décision « pas d'écran personnel »
(ADR-0047, amendée) reste conditionnée : on la rouvre si un tournoi réel compte des interruptions
que le mur ne peut pas éviter.
