# Diriger un tournoi — plan de développement

**Publié le 2026-09-07** : quarante-huit issues, [#364-#397](https://github.com/kevung/blunderDB/issues?q=label%3Atournoi) sur blunderDB et
[#1-#14](https://github.com/PileOfCells/backgammon-tournoi/issues) sur Nicomaque.

Cinq lots, chacun livrable et utilisable seul, découpés en issues **verticales** : une issue
traverse Go → binding Wails → Svelte → tests → doc/`.po` et laisse l'application utilisable.
Efforts : S ≤ ½ j, M ≤ 2-3 j, L = chantier. Une issue = une branche = un worktree = une PR
(`CLAUDE.md`). Les issues préfixées **N** sont dans le dépôt **Nicomaque**
(`github.com/PileOfCells/backgammon-tournoi`) ; les issues **D** dans blunderDB. Une issue D
qui consomme une issue N nomme le tag attendu.

## Lot 0 — les fondations (rien de visible)

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) | Codes structurés (libellés, notes, avertissements, raisons) et `Event.Version` | M | — |
| [N2](https://github.com/PileOfCells/backgammon-tournoi/issues/2) | Tables : indisponibles, réservées, `table_changed` | S | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [N3](https://github.com/PileOfCells/backgammon-tournoi/issues/3) | Résultat : score libre, forfait d'un match, remarque, retrait différé | S | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [D0.1](https://github.com/kevung/blunderDB/issues/364) | ADR-0047, glossaire (Direction, Participant, Directory, Slot), amendement 0037 | S | — (cette branche) |
| [D0.2](https://github.com/kevung/blunderDB/issues/367) | Paquet `direction` : `Open`/`Propose`/`Apply`/`State`, traduction des codes, tests sur journaux simulés | L | N1, D0.3 |
| [D0.3](https://github.com/kevung/blunderDB/issues/365) | Schéma 2.23.0 : `direction`, `direction_event`, `match.direction_match_id` ; trois copies, migration 024, `demo.db` | M | — |
| [D0.4](https://github.com/kevung/blunderDB/issues/366) | Prototype jetable de la page Direction (file, grille, fiche) sur données `sim.Run` | S | — |

**Livrable** : `go test ./pkg/blunderdb/direction/...` vert ; un journal simulé de 64 joueurs
rejoué sans avertissement ; la base à 2.23.0 ; un dessin validé à la main.

## Lot 1 — diriger un tournoi de club

Le chemin complet : suisse 2 vies continu + tableau à exemptions, 16 à 32 joueurs.

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [D1.1](https://github.com/kevung/blunderDB/issues/368) | Créer une Direction depuis le panneau Tournoi ; états ; configurations nommées ; suppression | M | D0.2, D0.3 |
| [D1.2](https://github.com/kevung/blunderDB/issues/369) | Vue Joueurs : inscrire (autocomplétion Players + PR), corriger, retirer (immédiat/différé), effectif et pool | M | [D1.1](https://github.com/kevung/blunderDB/issues/368) |
| [D1.3](https://github.com/kevung/blunderDB/issues/370) | Page Direction : file des propositions, [Lancer], [Tout lancer], apparier à la main | M | [D1.1](https://github.com/kevung/blunderDB/issues/368) |
| [D1.4](https://github.com/kevung/blunderDB/issues/371) | Grille des tables et fiche de résultat : vainqueur en un clic, score libre, forfait, remarque | M | D1.3, N2, N3 |
| [D1.5](https://github.com/kevung/blunderDB/issues/372) | Corriger et annuler : dernier résultat avec [Corriger], `Ctrl+Z`, fiche rouverte, événements de correction | M | [D1.4](https://github.com/kevung/blunderDB/issues/371) |
| [D1.6](https://github.com/kevung/blunderDB/issues/373) | Vue Arbres : sections SVG, poules, blocs, clic sur un match | M | D1.3, N11 |
| [D1.7](https://github.com/kevung/blunderDB/issues/374) | Vue Classement et clôture : classement, prix, CSV | S | [D1.3](https://github.com/kevung/blunderDB/issues/370) |
| [D1.8](https://github.com/kevung/blunderDB/issues/375) | Vue Historique : événements filtrables, corriger/annuler depuis là | S | [D1.5](https://github.com/kevung/blunderDB/issues/372) |
| [D1.9](https://github.com/kevung/blunderDB/issues/376) | Cohabitation : zone principale, badge d'onglet, barre d'état, `Ctrl+Maj+D`, commandes | M | [D1.3](https://github.com/kevung/blunderDB/issues/370) |
| [D1.10](https://github.com/kevung/blunderDB/issues/377) | Bande d'horloge : temps, matchs lents, fin estimée | S | [D1.3](https://github.com/kevung/blunderDB/issues/370) |
| [D1.11](https://github.com/kevung/blunderDB/issues/379) | Documentation : `manuel.rst`, `guide_utilisateur.rst`, `raccourcis.rst`, `cmd_mode.rst`, `a_propos.rst`, huit `.po`, `make help` | M | [D1.9](https://github.com/kevung/blunderDB/issues/376) |
| [D1.12](https://github.com/kevung/blunderDB/issues/378) | Crédit : bouton info, fenêtre, liens, version du module | S | [D1.1](https://github.com/kevung/blunderDB/issues/368) |
| [D1.13](https://github.com/kevung/blunderDB/issues/380) | Jalon : diriger un vrai tournoi de club | M | D1.11 |

**Livrable** : un tournoi de club de 16 à 32 joueurs dirigé de bout en bout, en doublant sur
papier ; budgets 4.1 et 4.2 de [ux.md](ux.md) tenus à la main. **La documentation fait partie
du lot.** Ce qui a manqué au directeur devient des issues.

## Lot 2 — les Matchs

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [N4](https://github.com/PileOfCells/backgammon-tournoi/issues/4) | `EvConfigChanged`, `EvReopened` | M | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [D2.1](https://github.com/kevung/blunderDB/issues/381) | Transcrire depuis un Slot : en-tête hérité, Slot réservé au brouillon, retour | M | [D1.4](https://github.com/kevung/blunderDB/issues/371) |
| [D2.2](https://github.com/kevung/blunderDB/issues/382) | Rattacher un Match existant : picker, suggestions par noms égaux, détacher | M | [D1.4](https://github.com/kevung/blunderDB/issues/371) |
| [D2.3](https://github.com/kevung/blunderDB/issues/383) | Bandeau des matchs importés non rattachés ; écart de score signalé | S | [D2.2](https://github.com/kevung/blunderDB/issues/382) |
| [D2.4](https://github.com/kevung/blunderDB/issues/384) | Le Slot dans le panneau Match ; ouvrir un Match depuis son Slot | S | [D2.2](https://github.com/kevung/blunderDB/issues/382) |
| [D2.5](https://github.com/kevung/blunderDB/issues/385) | Changer la configuration en cours ; réouvrir un tournoi clos | M | D1.7, N4 |

**Livrable** : un tournoi du BMAB dont les 28 matchs sont transcrits ou importés et rangés
par ronde et par joueur.

## Lot 3 — la salle

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [N7](https://github.com/PileOfCells/backgammon-tournoi/issues/7) | Micro-rondes (`BatchMinutes`) et pauses (`Breaks`) | M | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [N8](https://github.com/PileOfCells/backgammon-tournoi/issues/8) | Réparation proposée après correction dans un graphe | M | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [N11](https://github.com/PileOfCells/backgammon-tournoi/issues/11) | `render/` : `Labeler`, CSS injecté, page autonome, feuille d'appariements, grille des tables | M | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [D3.1](https://github.com/kevung/blunderDB/issues/386) | Sorties : dossier, page HTML réécrite à chaque événement, ouverture dans le navigateur | M | D1.6, N11 |
| [D3.2](https://github.com/kevung/blunderDB/issues/387) | Feuille d'appariements imprimable | S | [D3.1](https://github.com/kevung/blunderDB/issues/386) |
| [D3.3](https://github.com/kevung/blunderDB/issues/388) | Micro-rondes : file d'attente, compte à rebours, badge ; pauses et leur avertissement | M | D1.10, N7 |
| [D3.4](https://github.com/kevung/blunderDB/issues/389) | Réparation proposée : les propositions d'annulation et de relance dans la file | S | D1.5, N8 |
| [D3.5](https://github.com/kevung/blunderDB/issues/390) | Specs Playwright : budgets de clics F6, F7, F28, F10, F12, F13 ; coût d'entrée F1+F3+F5 | M | [D1.9](https://github.com/kevung/blunderDB/issues/376) |

**Livrable** : un tournoi de 48 à 64 joueurs sur deux jours, mur et feuilles comprises ;
budgets vérifiés par la CI.

## Lot 4 — l'entour

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [N5](https://github.com/PileOfCells/backgammon-tournoi/issues/5) | Longueurs par tour ; fin de suisse allongée | S | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [N6](https://github.com/PileOfCells/backgammon-tournoi/issues/6) | Retardataires après tirage (`Slot` sur `player_added`) | M | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [N9](https://github.com/PileOfCells/backgammon-tournoi/issues/9) | Dotation par section, retenue, pourcentages ; classement par section | M | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [N10](https://github.com/PileOfCells/backgammon-tournoi/issues/10) | Têtes de série en option | S | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [N12](https://github.com/PileOfCells/backgammon-tournoi/issues/12) | `players.ToCSV` | S | — |
| [N13](https://github.com/PileOfCells/backgammon-tournoi/issues/13) | Site GitHub Pages en neuf langues (documents introductifs, étude, spécification) | L | — |
| [N14](https://github.com/PileOfCells/backgammon-tournoi/issues/14) | Fuzzing d'`Apply`, journaux `Version: 0`, bascule Σvies en mode `rounds` | M | [N1](https://github.com/PileOfCells/backgammon-tournoi/issues/1) |
| [D4.1](https://github.com/kevung/blunderDB/issues/391) | Annuaire : vue dérivée, reprendre les inscrits, import/export CSV | M | D1.2, N12 |
| [D4.2](https://github.com/kevung/blunderDB/issues/392) | Longueurs par tour et retardataires dans les Réglages et la vue Joueurs | S | D1.2, N5, N6 |
| [D4.3](https://github.com/kevung/blunderDB/issues/393) | Dotation : réglage, pool, prix par section, CSV | M | D1.7, N9 |
| [D4.4](https://github.com/kevung/blunderDB/issues/394) | Têtes de série : option de tirage désactivée par défaut | S | D1.1, N10 |
| [D4.5](https://github.com/kevung/blunderDB/issues/395) | CLI `tournament verify|standings|page|export|list` ; `CLI_USAGE.md`, `cli.rst`, `help` | M | [D3.1](https://github.com/kevung/blunderDB/issues/386) |
| [D4.6](https://github.com/kevung/blunderDB/issues/396) | Export de base : la Direction voyage avec son Tournament | S | [D0.3](https://github.com/kevung/blunderDB/issues/365) |
| [D4.7](https://github.com/kevung/blunderDB/issues/397) | Direction d'exemple dans `demo.db` | S | D0.3, D1.3 |

**Livrable** : tout `RESTE_A_FAIRE.md` traité (D5), le moteur documenté en neuf langues.

## Ordre et parallélisme

- **N1 d'abord, seul** : tout en dépend, et il change le format du journal — il doit être
  fait pendant qu'aucun journal réel n'existe.
- D0.3 (schéma) et D0.4 (prototype) sont parallèles à N1.
- Le lot 1 est séquentiel sur D1.1 → D1.3 → D1.4 ; D1.6, D1.7, D1.8, D1.10, D1.12 sont
  parallèles ensuite.
- Les lots 2, 3 et 4 sont parallélisables entre eux une fois le lot 1 livré, sauf les
  dépendances nommées.
- **Le tournoi réel de la fin du lot 1 est un jalon, pas une formalité** : ce qu'il révèle
  se planifie avant d'attaquer le lot 3.

## Risques

| Risque | Parade |
|---|---|
| Deux dépôts qui avancent ensemble | chaque issue D nomme l'issue N et le tag attendu ; le lot 0 fige N1 |
| Le premier tournoi révèle un manque de fond | il est placé à la fin du lot 1, pas à la fin du chantier |
| La zone principale sans plateau déroute | D0.4 (prototype) avant le lot 1 ; le plateau revient à tout autre onglet |
| Le coût d'entrée dérive | D3.5 mesure F1+F3+F5 en CI ; le budget est dans ux.md |
| Les neuf langues du site Nicomaque | N13 réutilise la chaîne de blunderDB ; c'est un lot L, pas une case à cocher |
