# ADR-0105 — Les règles de tournoi sont des fonctions pures de Nicomaque, partagées avec gammonGo

Statut : acceptée.
Voir aussi : ADR-0047 (Direction, Nicomaque consommé par tag), ADR-0056 (Rencontre), ADR-0058
(propriétés des tables), ADR-0057 (exposition au démon), ADR-0019.

## Contexte

gammonGo organise lui aussi des tournois, sur ses propres lignes PostgreSQL : inscriptions
publiques, résultats saisis par les joueurs, tables en ligne, suisse Baku et Monrad, Buchholz,
circuits. Il lui manque ce que Nicomaque sait de la conduite de salle : vies et bascule
Σvies = 2^k, GSL, dotation, prévision de fin, feuille d'appariements. Deux voies sont fermées :
appeler les Directions de blunderDB (une Direction est le journal d'un organisateur seul : ce
serait l'état en double) et réécrire ces règles dans gammonGo (les règles en double).

Or Nicomaque (`github.com/PileOfCells/backgammon-tournoi`, `go 1.22`, aucune dépendance) ne
calcule rien hors de son `State`, que `Replay` reconstruit depuis un journal d'événements :
`Propose`, `ClockAt`, `SectionRanking` sont des méthodes de `State` ; `sim.Forecast` prend un
`Journal` ; `render.PairingSheet` prend un `*State`. Seul `tournoi.Prizes` est déjà pur. Et
blunderDB porte lui-même une part de la prévision de fin : temps de jeu (union des intervalles
de match), jour de jeu, fin estimée (médiane de `sim.Forecast` reportée au-delà des pauses),
dans `pkg/blunderdb/direction/clock.go`.

## Décision

1. **Un package de fonctions pures, `rules`, dans le module Nicomaque.** Sans état, sans E/S,
   sans horloge (l'instant et la graine sont des arguments), bibliothèque standard seule. Ses
   types d'entrée et de sortie sont les siens : ni `State`, ni `Event`, ni identifiant de base.
   Un joueur y est une référence opaque (`Ref`, une chaîne) que l'appelant choisit et retrouve.
   Il couvre cinq règles :
   - **Appariements** : `Pair(format, instantané, graine)` → paires, longueurs, exemptions, pour
     chaque format de Nicomaque (suisse à vies, tableau à vies et à exemptions, élimination
     avec consolante, GSL, poules) ;
   - **Classement et départages** : `Rank(format, résultats, ordre des départages)` → rangs et
     notes codées ;
   - **Dotation** : droit d'entrée, retenue, barème par section, partage des places ex aequo ;
   - **Prévision de fin** : matchs restants, rythme observé, temps de jeu, jour de jeu, fin
     estimée après les pauses ;
   - **Feuille d'appariements** : les lignes d'une ronde, en codes, sans phrase ni HTML.
   Le package émet des codes, jamais des phrases (ADR-0047) : chaque hôte traduit.
2. **Le dépôt hôte est celui de Nicomaque**, dans le même module que le moteur, et non un module
   à part ni un dépôt blunderDB. Le moteur reste à son auteur, consommé par tag, jamais bifurqué
   (ADR-0047) ; la demande s'ouvre en issue de son dépôt. Les méthodes de `State`, `sim.Forecast`
   et `render.PairingSheet` y deviennent des **enveloppes** de `rules` : il n'existe qu'une
   implémentation de chaque règle, et la parité entre le moteur à journal et le package pur est
   l'affaire du dépôt qui les porte tous deux.
3. **Chacun garde son état.** blunderDB garde la Direction comme source de vérité : journal
   append-only, état dérivé rejoué, jamais stocké, tirages matérialisés dans `EvDraw`. gammonGo
   garde ses lignes. Ni l'un ni l'autre ne stocke une sortie de `rules`. Aucun appel réseau ne
   relie les deux pour une règle.
4. **Ce qui sort de blunderDB** : la prévision de fin écrite dans `direction/clock.go`
   (`PlayingTime`, `DayOfPlay`, `AfterBreaks`, médiane de la prévision). Elle passe dans `rules`
   et `direction` l'appelle.
   **Ce qui reste** : la Direction, son `Record`, son `Store` et le schéma ; la Rencontre, la
   salle et la fusion des tables des épreuves sœurs passées au moteur en `External`
   (ADR-0056) ; les propriétés des tables (ADR-0058) ; Participant, paires de doubles,
   annuaire ; le lien Slot ↔ Match ; le classement de saison par PR (il lit des analyses) ; la
   traduction des codes (`labeler.go`, catalogues) ; les pages murales, la feuille rendue en HTML
   et l'arbre en SVG ; l'exposition `serve`/`call` (ADR-0057). Un format ou un départage que
   gammonGo verse dans `rules` (Buchholz, Monrad, demi-bye) n'apparaît dans blunderDB que si une
   configuration de Direction le demande.
5. **Versionnement : les tags semver du module**, `v0.x` jusqu'à stabilisation, puis `v1`. Une
   fonction ajoutée sans rien changer est un correctif ou une mineure ; **une règle dont la
   sortie change à entrée égale** (un tirage, un rang, une part de dotation, une fin estimée) est
   une mineure et s'inscrit au changelog des règles. Le dépôt versionne des **fichiers dorés**
   (entrée et sortie JSON de chaque fonction) que ses propres tests rejouent et qu'un client
   rejoue contre son adaptateur. Un journal existant reste rejouable : les tirages sont dans
   `EvDraw`, et la version de journal suit sa propre règle (ADR-0047).
6. **Compatibilité Go : la directive `go` du module reste ≤ `1.26.0`**, le plancher de
   blunderDB. gammonGo embarque `pkg/blunderdb/server` et hérite de ce plancher : un module qui
   le dépasserait forcerait la montée de blunderDB, et `CLAUDE.md` l'interdit sans dépendance qui
   l'exige. La directive suit la plus petite version que le code de `rules` exige, et `rules`
   n'importe aucun module tiers.
7. **Une seule version dans un binaire.** gammonGo construit blunderDB et Nicomaque dans le même
   graphe de modules : la sélection de version minimale y retient la plus haute des deux
   exigences. gammonGo ne peut donc pas épingler une version de Nicomaque plus ancienne que celle
   de blunderDB ; chacun monte à son rythme, mais seulement vers le haut, et le changelog des
   règles lui dit ce qu'il reçoit en montant blunderDB.
8. **`EngineVersion` égale le `require` de `go.mod`.** La constante de `direction` est stockée
   avec chaque Direction pour tracer un rejeu au moteur qui l'a écrit : elle nomme le tag
   réellement compilé, ce qu'un test vérifie.

## Ordre de migration

1. **Issue au dépôt Nicomaque** : le package `rules` et ses cinq règles, ce document en
   référence.
2. **Nicomaque, premier lot, sans changement de règle** : `Prizes`, la prévision (y compris
   l'arithmétique reprise de `direction/clock.go`) et la feuille, pures ; `State`, `sim` et
   `render` en enveloppes ; fichiers dorés ; tag mineur.
3. **blunderDB** : `go.mod` monte au tag ; `direction/clock.go` délègue à `rules` et perd son
   arithmétique ; `EngineVersion` alignée et gardée par un test. Ni schéma, ni `DatabaseVersion`,
   ni format de journal, ni base de démo ne changent ; les tests de `direction` et la
   contre-épreuve de simulation restent verts sans retouche.
4. **Nicomaque, second lot** : `Rank` et `Pair` pour chaque format, `Propose` et `SectionRanking`
   en enveloppes, parité par les configurations de `sim_test.go` et les fichiers dorés ; tag
   mineur. blunderDB monte le tag comme à l'étape 3, sans code.
5. **gammonGo** adopte à son rythme : dotation, feuille et prévision après l'étape 2, vies et GSL
   après l'étape 4. Ses règles sans équivalent y migrent de préférence, pour qu'une règle n'ait
   qu'un propriétaire.

## Conséquences

- blunderDB ne change de comportement à aucune étape : `rules` est atteint par les enveloppes
  du moteur, et seul `direction` l'importe directement.
- Une correction de règle se fait une fois, au dépôt Nicomaque, et parvient aux deux
  applications par un tag.
- Le crédit à Nicolas Harmand (ADR-0047) couvre `rules` sans ajout : c'est son moteur.
- Écartés : appeler les Directions de blunderDB depuis gammonGo (état en double, une Direction
  est le journal d'un seul organisateur) ; porter les règles dans gammonGo (règles en double) ;
  un module ou un dépôt séparé pour `rules` (deux tags à accorder, parité entre deux dépôts) ;
  héberger `rules` dans blunderDB (gammonGo dépendrait de blunderDB pour une règle, et le moteur
  serait bifurqué) ; un `State` partagé (impose le journal à gammonGo).

## Garde

Au dépôt Nicomaque, les fichiers dorés de `rules` et `sim_test.go`, rejoués par les
enveloppes. Dans blunderDB, les tests de `pkg/blunderdb/direction` inchangés à chaque montée de
tag, et le test qui lie `EngineVersion` au `require` de `go.mod`.
