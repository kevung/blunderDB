# Un ordonnanceur unique et une file par bibliothèque pour les tâches durables

Statut : acceptée.
Voir aussi : ADR-0013 (règle 2 : écrire est une tâche bornée et visible), ADR-0060 (rollout),
ADR-0088 (maintenance : une opération à la fois par tenant), ADR-0093 (budget de cœurs),
ADR-0101 (une tâche de fond reste attachée à sa bibliothèque).

## Contexte

Le bureau lance des tâches longues par des mécanismes séparés : un lot gammonNet, un rollout
(position ou lot), une passe de maintenance, le remplissage de `match_stats` à l'ouverture, des
imports en arrière-plan, des transferts, la génération des tables de bearoff, une
synchronisation. Chacune avait son verrou, son annulation et ses événements ; aucune n'était
listée nulle part. Le créneau de `Database` (`db_jobslot.go`) a une profondeur de 1 : un rollout
demandé pendant une réanalyse était refusé (`BusyError`) au lieu d'attendre. Seul le lot
gammonNet cédait au calcul interactif, en sondant toutes les 50 ms. L'arrêt avant la fermeture
d'une bibliothèque énumérait les tâches à la main.

## Décision

1. **Un registre unique** (`pkg/blunderdb/jobs`, `App.jobs`) liste toute tâche durable du
   bureau : réanalyse gammonNet (manquantes, périmées, d'un match), rollout de position et de
   lot, maintenance, remplissage `match_stats`, import (au premier plan comme en arrière-plan),
   transfert, tables de bearoff, synchronisation. Chaque tâche expose identifiant, type,
   libellé, bibliothèque (groupe), priorité, état (`running`, `waiting`, `paused`, `done`,
   `cancelled`, `error`), progression fait/total, rang dans la file. Les 32 dernières tâches
   finies restent listées jusqu'à leur retrait.
2. **Trois priorités de calcul.** *Visible* : ce qui est à l'écran (évaluation, cartes des jets
   et des scores) ; il n'est pas listé, il entre par `beginVisibleComputation` et en sort par
   la fonction rendue. *Demandé* : un rollout lancé par l'utilisateur. *Fond* : réanalyse,
   maintenance. Une tâche de calcul cède, à ses points de contrôle (`Job.Checkpoint`), à tout
   calcul de priorité supérieure en cours : le lot gammonNet avant chaque position, le rollout
   entre deux paquets de parties. L'attente est réveillée par chaque changement du registre,
   jamais par un sondage à intervalle fixe. La priorité décide qui tourne, pas sur combien de
   cœurs : le budget reste celui de l'ADR-0093.
3. **Une file par bibliothèque, en mémoire.** Une tâche qui écrit sa bibliothèque (lot gammonNet,
   rollout stocké, lot de rollouts, maintenance) y prend son tour (`Job.Turn`) : demandée
   pendant qu'une autre tient la bibliothèque, elle attend au lieu d'être refusée. Elle entre
   devant les tâches de priorité inférieure, après celles de même priorité. La file se
   réordonne, une tâche en attente se met en pause (la suivante passe), se reprend, s'annule ou
   se retire. Une tâche en cours se met en pause si elle a des points de contrôle ; elle garde
   alors la bibliothèque : la pause n'est pas une préemption. Rien n'est persistant : après
   fermeture, la reprise reste celle d'avant (un lot retrouve « ce qui manque », un import son
   journal).
4. **Le créneau de `Database` reste le garde-fou commun aux trois modes** (ADR-0088 §5) : le
   démon et la CLI n'ont pas de registre et gardent leur refus. Au bureau, une tâche qui a son
   tour passe encore par lui, avec `WaitForDatabase` : elle attend un détenteur hors registre
   (échange, compaction) au lieu d'échouer.
5. **Un seul flux d'événements** : `jobs:changed` (l'`Info` d'une tâche inscrite ou changée) et
   `jobs:removed` (`{id}`), dans l'ordre des changements. Les méthodes liées : `ListJobs`,
   `CancelJob`, `PauseJob`, `ResumeJob`, `MoveJob`, `RemoveJob`. Les événements propres à chaque
   tâche (`gammonnet-batch:*`, `rollout:*`, `rollout-batch:*`, `maintenance:*`,
   `import-files:progress`, `bearoff:*`, `sync:status`) restent émis tant que le frontend en
   dépend.
6. **L'arrêt passe par le registre.** `stopGroupJobs` (fermeture ou remplacement d'une
   bibliothèque, arrêt de l'application) annule les tâches de la bibliothèque, en cours ou en
   attente, et les attend dans un délai de grâce partagé. Une tâche liée à aucune bibliothèque
   (tables de bearoff) n'est pas attendue ; une tâche qu'on ne peut pas annuler
   (synchronisation) non plus.
7. **Accord avec les décisions voisines.** ADR-0013 règle 2 : le registre rend chaque tâche
   bornée *et* visible ; il n'introduit aucune tâche résidente. ADR-0088 : rien ne se lance
   seul, la file n'ordonne que ce qui a été demandé ou ce que l'import déclenchait déjà.
   ADR-0060 : un rollout non stocké n'écrit rien et ne prend pas de tour. ADR-0101 §5 : une
   tâche garde le groupe où elle a commencé ; la file est celle de ce groupe.

## Conséquences

- Le démon garde `analyses.queueStatus` et son registre d'imports inchangés ; le registre du
  bureau n'y est pas exposé, faute de contrat commun (un tenant n'est pas un groupe). Suivi
  dans `tasks/BACKLOG.md`.
- Tant que la recherche « au repos » du panneau d'évaluation n'entre pas par
  `beginVisibleComputation`, le lot gammonNet la guette encore par `gnEvalCancel`
  (`waitForInteractiveEvaluation`) ; le sondage disparaît quand elle y entre.
- Une tâche en pause qui tient la bibliothèque bloque la file derrière elle : c'est voulu, la
  reprendre ou l'annuler la libère.

Écarté : persister la file (une file rejouée à l'ouverture lancerait seule des écritures,
contre l'ADR-0088 §4) ; préempter une tâche en cours pour une tâche demandée (un lot gammonNet
interrompu au milieu d'une position perd son travail, le rollout gagnerait quelques secondes) ;
un verrou à profondeur 1 conservé avec un « réessayer » côté frontend (l'attente revenait à
l'utilisateur).

Garde : `pkg/blunderdb/jobs/jobs_test.go` (file, priorités, pause, retrait, arrêt) ;
`internal/gui` `TestRolloutBatchWaitsBehindTheGammonNetBatch`,
`TestStartMaintenance_WaitsBehindTheQueuedBatch`, `shutdown_test.go`.
