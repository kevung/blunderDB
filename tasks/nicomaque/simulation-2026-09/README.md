# Simulation de tournois dirigés — plan d'exécution (2026-09-24)

Ce dossier est un **plan à exécuter**, écrit à l'issue d'un entretien de cadrage
(neuf décisions, § 2) pour être repris par une autre session, sans refaire l'entretien.
Il ne contient aucune mesure : les mesures sont ce que l'exécution produit.

**Le but** : dire, chiffres à l'appui, si un directeur de tournoi couvre avec l'interface
de blunderDB tout ce qu'un tournoi réel lui demande — inscrire, choisir un format, annoncer
les rondes, saisir les résultats, publier le classement en cours, absorber les incidents —
et à quel prix en gestes ; puis transformer ce que la mesure révèle en lots d'issues.

Ce que le dépôt sait déjà, et que l'exécution ne redécouvre pas : le chantier D0-D4 est
livré en entier ([plan.md](../plan.md), issues #364-#397 fermées) ; seul le jalon **#380**
(« diriger un vrai tournoi de club ») reste ouvert, et **il le reste** : une simulation ne
remplace pas de vrais joueurs. Les budgets de gestes sont dans [ux.md](../ux.md) § 4, la
spécification dans [fonctionnel.md](../fonctionnel.md), le vocabulaire dans `CONTEXT.md`
§ « Directing a tournament ».

| Document | Contenu |
|---|---|
| [scenarios.md](scenarios.md) | six personas, cinq scénarios, les incidents injectés, ce que chaque scénario doit prouver |
| [mesure.md](mesure.md) | la règle de mesure : compteurs, viewports, grille par opération, format du rapport |
| [outillage.md](outillage.md) | le shim front réel + backend réel, les scénarios Go, les données, ce qui est jetable |
| [lots.md](lots.md) | comment un constat devient une issue ; les manques déjà visibles dans le code, à confirmer par la mesure |

## 1. Ce que l'exécution livre

1. `tasks/nicomaque/simulation-2026-09/rapport/` : un `README.md` index, un fichier par
   scénario (`S1.md` … `S5.md`), un `mesures.md` (la grille comparée aux budgets `ux.md`),
   chacun **sous 500 lignes** (règle du dépôt). Le rapport est aussi publié en Artifact.
2. Les **issues GitHub**, lots **D5 et suivants**, créées avec `gh` par l'exécutant, selon
   [lots.md](lots.md). Chaque issue cite la mesure qui la justifie. Aucune issue sans mesure.
3. Optionnellement, un **test de rejeu** dans `pkg/blunderdb/direction/` par scénario Go,
   s'il tient sous une seconde (voir [outillage.md](outillage.md) § 3).
4. Une petite PR à part, avant la mesure, qui ajoute les `data-testid` manquants — sans
   changement de comportement (décision Q9).

L'exécution **s'arrête là**. Les lots D5+ s'implémentent dans des sessions ultérieures.

## 2. Les neuf décisions de l'entretien

Prises le 2026-09-24 avec l'utilisateur ; l'exécutant les applique et ne les rouvre pas.

| # | Question | Décision |
|---|---|---|
| Q1 | Par quoi passe la simulation « via l'interface » ? | **Voie A** : un shim jetable qui branche le vrai front Svelte sur le vrai `Database` Go (vrai Nicomaque, vraie SQLite), piloté par Playwright ; **complété par des scénarios Go** pour la couverture fonctionnelle et le rejeu. Ni le mock e2e existant (quatre joueurs, aucun appariement), ni un comptage KLM par lecture du code. |
| Q2 | Comment modéliser un week-end à plusieurs épreuves ? | **Une épreuve = un Tournament**, chacun avec sa Direction, comme aujourd'hui. La simulation mesure le va-et-vient réel entre Directions et les tables partagées à la main. Le modèle cible des issues est une **Rencontre** (terme nouveau, § 4) qui regroupe des Tournaments, partage la salle et permet plusieurs Directions ouvertes côte à côte. Écartés : plusieurs Directions par Tournament (casse le 1:1 et le schéma), les épreuves comme sections d'une Direction (impossible sans toucher au moteur). |
| Q3 | Que recouvre « un tournoi sur une semaine » ? | Deux lectures simulées : **le championnat de club en rondes synchrones** (une ronde par soirée, mode `rounds`) et **une épreuve sur cinq jours** (suisse continu, sessions quotidiennes). Le festival d'une semaine est couvert par la Rencontre. |
| Q4 | Personas et scénarios | Les quatre personas de D16 gardés tels quels, **deux ajoutés** (Nadia, Karim) ; cinq scénarios S1-S5 ([scenarios.md](scenarios.md)). |
| Q5 | La règle de mesure | **Compter par geste, pas par seconde** : clics, touches, changements de vue, menus, modales, défilements en crans et en pixels ; le temps KLM calculé à côté pour comparer à `ux.md`. Deux viewports : 1024×768 (référence) et 1366×768. Un dépassement de budget est un constat, pas un échec. Le défilement se mesure en état réel, après une vraie ronde. |
| Q6 | Publication des résultats : jusqu'où vont les issues ? | **La simulation constate, les issues respectent l'ADR-0047** : l'écran des joueurs (téléphone, route du démon) reste fermé jusqu'à #380. Le besoin est mesuré (interruptions « je joue où ? ») et devient **une seule issue de cadrage** pointant l'ADR. Les lots proposent ce qui reste dans le périmètre : page murale par épreuve et par Rencontre, page « ronde N » datée, export fichier, dossier synchronisé. |
| Q7 | Forme des livrables | Rapport sous `tasks/nicomaque/simulation-2026-09/rapport/`, fichiers < 500 lignes, publié en Artifact ; issues D5+ créées par l'exécutant ; #380 reste ouverte ; shim dans le scratchpad, scénarios Go en test s'ils tiennent sous une seconde. |
| Q8 | Où s'arrête la session de cadrage ? | À ce plan. **Une autre session exécute** simulation, rapport et issues, et s'arrête avant d'implémenter les lots. |
| Q9 | Toucher au code du produit pendant la mesure ? | **Lecture seule, deux exceptions** : des `data-testid` dans une PR à part avant la mesure, et les scénarios Go déposés en test. Une simulation qui corrige ce qu'elle mesure ne mesure plus rien : un bug trouvé en route devient une issue, pas un correctif au passage. **Cas limite** : un bug bloquant arrête le scénario, ouvre une issue en priorité haute, et l'exécutant passe au scénario suivant. |

## 3. L'ordre d'exécution

Chaque étape est finie avant la suivante ; les étapes 3 et 4 sont parallélisables.

1. **PR des `data-testid`** (Q9) : lister, en lisant `frontend/src/components/direction/`,
   les prises que les flux de [mesure.md](mesure.md) § 3 exigent et qui manquent ; les
   ajouter ; fusionner. Zéro changement de comportement, `npm test` et
   `npm run test:e2e` verts.
2. **Le shim** ([outillage.md](outillage.md) § 1) : jusqu'à ce que le front réel affiche
   une Direction de démo (celle de `demo.db`) sur une base réelle. Critère : la spec
   `direction-budgets.spec.js` rejouée contre le shim donne les mêmes comptes que contre le
   mock.
3. **Les scénarios Go** ([outillage.md](outillage.md) § 3) : S1 à S5 joués par le moteur
   sur une base réelle, journaux exportés, `blunderdb tournament verify` sans
   avertissement résiduel. Ils produisent aussi les **bases de départ** de l'étape 4 (une
   base « samedi 14 h » avec 25 matchs joués, par exemple).
4. **La mesure Playwright** ([mesure.md](mesure.md)) : scénario par scénario, persona par
   persona, opération par opération, aux deux viewports. Captures d'écran des états où un
   défilement ou une modale apparaît.
5. **Le rapport**, puis **les issues** ([lots.md](lots.md)). Les issues ne s'écrivent
   qu'une fois le rapport relu en entier : un même manque vu dans trois scénarios est une
   issue, pas trois.

## 4. Le vocabulaire que ce plan introduit

**Rencontre** : plusieurs Tournaments dirigés dans la même salle, aux mêmes dates, par le
même directeur — un festival de week-end avec son principal, son speed et ses doubles. Elle
partage les tables, les pauses et la page murale ; chaque épreuve reste un Tournament avec sa
Direction, ses Matchs et son classement. Un joueur inscrit à deux épreuves est deux
Participants, et c'est la Rencontre qui sait qu'il ne peut pas être à deux tables.
_Éviter_ : festival (c'est un nom d'événement, pas un objet), meeting, réunion.

Le terme **n'entre pas encore dans `CONTEXT.md`** : il n'existe dans aucun code, et
l'ADR-0047 dit « une seule entité ». Il y entre avec l'ADR du lot qui le construit
([lots.md](lots.md) § 3, lot « Rencontre »), qui amende 0047 sur ce point.

## 5. Ce que l'exécutant ne fait pas

- Fermer #380, ou écrire dans le rapport qu'un tournoi réel a été dirigé.
- Proposer l'écran des joueurs autrement que par l'issue de cadrage de Q6.
- Modifier le moteur : un défaut de Nicomaque se corrige chez son auteur (issue dans
  `PileOfCells/backgammon-tournoi`), pas en contournement ici — c'est ce que le chantier
  a appris ([decisions.md](../decisions.md), « Nicomaque v0.2.1 »).
- Committer le shim, la configuration Playwright jetable ou les bases de simulation.
- Écrire dans le rapport une mesure qu'il n'a pas faite : un flux non mesuré est marqué
  « non mesuré, raison », jamais estimé.
