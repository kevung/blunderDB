# Un ordonnanceur unique et une file par bibliothèque pour les tâches durables

Statut : acceptée. Amende ADR-0088 §5 au bureau seulement : une opération demandée pendant
qu'une autre tient la bibliothèque y attend son tour ; le refus reste la règle du démon et de la
CLI.
Voir aussi : ADR-0013 (règle 2 : écrire est une tâche bornée et visible), ADR-0060 (rollout),
ADR-0093 (budget de cœurs), ADR-0101 (une tâche de fond reste attachée à sa bibliothèque).

## Contexte

Le bureau lance des tâches longues par des mécanismes séparés : un lot gammonNet, un rollout
(position ou lot), une passe de maintenance, le remplissage de `match_stats` à l'ouverture, des
imports, des transferts, une réparation de match, la génération des tables de bearoff, une
synchronisation. Chacune avait son verrou, son annulation et ses événements ; aucune n'était
listée nulle part. Le créneau de `Database` (`db_jobslot.go`) a une profondeur de 1 : un rollout
demandé pendant une réanalyse était refusé (`BusyError`) au lieu d'attendre. L'arrêt avant la
fermeture d'une bibliothèque énumérait les tâches à la main.

## Décision

1. **Un registre unique** (`pkg/blunderdb/jobs`, `App.jobs`) liste toute tâche durable du
   bureau : réanalyse gammonNet (manquantes, périmées, d'un match), rollout de position et de
   lot, maintenance, remplissage `match_stats` (une tâche par remplissage), import (fichiers au
   premier plan et en arrière-plan, et les imports que le frontend appelle directement sur la
   poignée liée — `.xg`, gnubg, BGF, base `.db` — par `database.SetTaskTracker`), réparation de
   match, transfert (une entrée, quel que soit le nombre de bibliothèques lues), tables de
   bearoff, synchronisation. Chaque tâche expose identifiant, type, libellé, bibliothèque
   (groupe), priorité, état (`running`, `waiting`, `paused`, `done`, `cancelled`, `error`),
   progression fait/total, rang dans la file. Toute tâche finit dans un état final sur tous
   les chemins, panique comprise : sa bibliothèque passe alors à la suivante. Les 32 dernières
   tâches finies restent listées jusqu'à leur retrait.
2. **Trois priorités de calcul.** *Visible* : ce qui est à l'écran — l'évaluation au repos
   du panneau, la carte des jets, la carte des scores (qui tient lieu de matrice de videau).
   Chacun entre par `beginVisibleComputation` en prenant la case des recherches interactives
   (`beginInteractiveSearch`) et en sort dans tous les chemins, fin, annulation, remplacement,
   erreur ou panique ; un calcul visible n'en préempte jamais un autre, le plus récent
   remplace seulement le précédent. *Demandé* : un
   rollout, un import, un transfert. *Fond* : réanalyse, maintenance, remplissage. Une tâche de
   calcul cède, à ses points de contrôle (`Job.Checkpoint`), à un calcul de priorité supérieure
   en cours et non en pause : le lot gammonNet avant chaque position, le rollout entre deux
   paquets de parties. L'attente est réveillée par chaque changement du registre, jamais par un
   sondage. La priorité décide qui tourne, pas sur combien de cœurs : le budget reste celui de
   l'ADR-0093.
3. **Une file par bibliothèque, en mémoire.** Une tâche qui écrit sa bibliothèque (lot gammonNet,
   rollout stocké, lot de rollouts, maintenance) y prend son tour (`Job.Turn`) : demandée
   pendant qu'une autre la tient, elle attend au lieu d'être refusée. Elle entre devant les
   tâches de priorité inférieure, après celles de même priorité. La file se réordonne ; une
   tâche en attente se met en pause (la suivante passe), se reprend, s'annule ou se retire. Une
   tâche en cours se met en pause si elle a des points de contrôle, et garde alors la
   bibliothèque. Rien n'est persistant : après fermeture, la reprise reste celle d'avant (un lot
   retrouve « ce qui manque », un import son journal).
4. **Une tâche demandée préempte le lot gammonNet.** Le lot est la seule tâche préemptible
   (`Spec.Preemptible`) : son travail reprend de lui-même, il recalcule ce qui manque. Quand une
   tâche de priorité supérieure, non en pause, attend sa bibliothèque, son point de contrôle
   rend `ErrPreempted` : la position en cours s'interrompt et reste sans analyse, le lot rend la
   bibliothèque (`Job.Requeue`), se replace en tête des tâches de fond et repart sur ce qui
   manque à son tour. Un rollout stocké n'attend donc plus la fin d'une réanalyse entière. La
   maintenance n'est pas préemptée : une passe interrompue n'a pas de reprise propre.
5. **Une tâche en file reste liée à la bibliothèque où elle a été demandée** (ADR-0101 §5). Le
   lot gammonNet et la maintenance tournent sur une poignée empruntée à leur bibliothèque dès la
   demande ; un rollout, qui montre son résultat au plateau, tient le groupe à l'écran
   (`libraryBusy`) dès qu'il est demandé, en attente compris.
6. **Le créneau de `Database` reste le garde-fou des trois modes.** Le démon et la CLI n'ont pas
   de registre et refusent toujours (ADR-0088 §5). Au bureau, une tâche qui a son tour passe
   encore par lui avec `WaitForDatabase` (lot, rollout, maintenance par
   `RunMaintenanceContext`) : elle attend un détenteur hors registre (échange, compaction) au
   lieu d'échouer.
7. **Un seul flux d'événements** : `jobs:changed` (l'`Info` d'une tâche inscrite ou changée) et
   `jobs:removed` (`{id}`), dans l'ordre des changements. Méthodes liées : `ListJobs`,
   `CancelJob`, `PauseJob`, `ResumeJob`, `MoveJob`, `RemoveJob`. Les événements propres à chaque
   tâche (`gammonnet-batch:*`, `rollout:*`, `rollout-batch:*`, `maintenance:*`,
   `import-files:progress`, `bearoff:*`, `sync:status`) restent émis : les panneaux de ces
   tâches les lisent. Aucun panneau ne lit encore le flux `jobs:*`.
8. **L'arrêt passe par le registre.** `stopGroupJobs` (fermeture ou remplacement d'une
   bibliothèque, arrêt de l'application) annule les tâches de la bibliothèque, en cours ou en
   attente, et les attend dans un délai de grâce partagé. Une tâche liée à aucune bibliothèque
   (tables de bearoff) n'est pas attendue ; une tâche qu'on ne peut pas annuler
   (synchronisation) non plus.
9. **Accord avec les décisions voisines.** ADR-0013 règle 2 : le registre rend chaque tâche
   bornée *et* visible ; il n'introduit aucune tâche résidente. ADR-0088 §4 : rien ne se lance
   seul, la file n'ordonne que ce qui a été demandé ou ce que l'import déclenchait déjà ; §5 est
   amendé au bureau comme dit en tête. ADR-0060 : un rollout non stocké n'écrit rien et ne prend
   pas de tour.

## Conséquences

- Le démon garde `analyses.queueStatus` et son registre d'imports inchangés ; le registre du
  bureau n'y est pas exposé, faute de contrat commun (un tenant n'est pas un groupe). Suivi
  dans `tasks/BACKLOG.md`.
- Le lot gammonNet et le rollout cèdent aux calculs visibles par leur seul point de contrôle :
  plus de sondage de la recherche du panneau, ni de lot réduit en cœurs à son démarrage.
- Une tâche en pause qui tient la bibliothèque bloque la file derrière elle, sauf préemption
  (§4) : la reprendre ou l'annuler la libère.

Écarté : persister la file (une file rejouée à l'ouverture lancerait seule des écritures,
contre l'ADR-0088 §4) ; préempter toute tâche de fond (une passe de maintenance interrompue n'a
pas de reprise) ; garder le verrou à profondeur 1 avec un « réessayer » côté frontend (l'attente
revenait à l'utilisateur).

Garde : `pkg/blunderdb/jobs/jobs_test.go` (file, priorités, pause, préemption, retrait, arrêt) ;
`internal/gui` `TestRequestedTask_PreemptsTheBackgroundBatch`,
`TestPanickedTask_GivesItsLibraryToTheNext`,
`TestStartMaintenance_WaitingPassKeepsItsLibraryAcrossASwitch`,
`TestMatchStatsFill_EachFillIsItsOwnTaskEndedOnEveryPath`,
`TestRolloutBatchWaitsBehindTheGammonNetBatch`, `shutdown_test.go` ;
`database` `TestTracked_EndsOnceOnEveryPath`.
