.. _annexe_db_migration:

Annexe: Schéma de la base de données
====================================


.. important:: 
    Sauvegardez toujours votre fichier .db avant d'effectuer des migrations de base de données.

Version 1.0.0
-------------

La version 1.0.0 de la base de données contient les tables suivantes :

- **position** : Stocke les positions avec les colonnes `id` (clé primaire) et `state` (état de la position en format JSON).
- **analysis** : Stocke les analyses des positions avec les colonnes `id` (clé primaire), `position_id` (clé étrangère vers `position`), et `data` (données de l'analyse en format JSON).
- **comment** : Stocke les commentaires associés aux positions avec les colonnes `id` (clé primaire), `position_id` (clé étrangère vers `position`), et `text` (texte du commentaire).
- **metadata** : Stocke les métadonnées de la base de données avec les colonnes `key` (clé primaire) et `value` (valeur associée à la clé).

Version 1.1.0
-------------

La version 1.1.0 de la base de données ajoute la table suivante :

- **command_history** : Stocke l'historique des commandes avec les colonnes `id` (clé primaire), `command` (texte de la commande), et `timestamp` (date et heure de l'exécution de la commande).

Les autres tables restent inchangées par rapport à la version 1.0.0.

Pour migrer la base de données de la version 1.0.0 à la version 1.1.0, exécutez la commande ``migrate_from_1_0_to_1_1`` dans blunderDB.

Version 1.2.0
-------------

La version 1.2.0 de la base de données ajoute la table suivante :

- **filter_library** : Stocke les filtres de recherche avec les colonnes `id` (clé primaire), `name` (nom du filtre), `command` (commande associée au filtre), et `edit_position` (position éditée lors de l'enregistrement du filtre). 

Les autres tables restent inchangées par rapport à la version 1.1.0.

Pour migrer la base de données de la version 1.1.0 à la version 1.2.0, exécutez la commande ``migrate_from_1_1_to_1_2`` dans blunderDB.

Version 1.3.0
-------------

La version 1.3.0 de la base de données ajoute la table suivante :

- **search_history** : Stocke l'historique des recherches de positions avec les colonnes `id` (clé primaire), `command` (texte de la commande de recherche), `position` (état de la position au moment de la recherche), et `timestamp` (date et heure de la recherche).

Les autres tables restent inchangées par rapport à la version 1.2.0.

Pour migrer la base de données de la version 1.2.0 à la version 1.3.0, exécutez la commande ``migrate_from_1_2_to_1_3`` dans blunderDB.

Version 1.4.0
-------------

La version 1.4.0 de la base de données ajoute les tables suivantes pour la gestion des matchs :

- **match** : Stocke les matchs importés avec les colonnes `id` (clé primaire), `player1_name`, `player2_name`, `event`, `location`, `round`, `match_length`, `match_date`, `import_date`, `file_path`, `game_count`, et `match_hash` (hash pour la détection de doublons).
- **game** : Stocke les parties d'un match avec les colonnes `id` (clé primaire), `match_id` (clé étrangère vers `match`), `game_number`, `initial_score_1`, `initial_score_2`, `winner`, `points_won`, et `move_count`.
- **move** : Stocke les coups d'une partie avec les colonnes `id` (clé primaire), `game_id` (clé étrangère vers `game`), `move_number`, `move_type`, `position_id` (clé étrangère vers `position`), `player`, `dice_1`, `dice_2`, `checker_move`, et `cube_action`.
- **move_analysis** : Stocke l'analyse de chaque coup avec les colonnes `id` (clé primaire), `move_id` (clé étrangère vers `move`), `analysis_type`, `depth`, `equity`, `equity_error`, `win_rate`, `gammon_rate`, `backgammon_rate`, `opponent_win_rate`, `opponent_gammon_rate`, et `opponent_backgammon_rate`.

La migration de 1.3.0 à 1.4.0 est automatique à l'ouverture de la base de données.

Version 1.5.0
-------------

La version 1.5.0 de la base de données ajoute les tables suivantes pour la gestion des collections :

- **collection** : Stocke les collections de positions avec les colonnes `id` (clé primaire), `name`, `description`, `sort_order`, `created_at`, et `updated_at`.
- **collection_position** : Table de liaison qui associe des positions aux collections avec les colonnes `id` (clé primaire), `collection_id` (clé étrangère vers `collection`), `position_id` (clé étrangère vers `position`), `sort_order`, et `added_at`. La paire (`collection_id`, `position_id`) est unique.

La migration de 1.4.0 à 1.5.0 est automatique à l'ouverture de la base de données.

Version 1.6.0
-------------

La version 1.6.0 de la base de données ajoute la table suivante pour la gestion des tournois :

- **tournament** : Stocke les tournois avec les colonnes `id` (clé primaire), `name`, `date`, `location`, `sort_order`, `created_at`, et `updated_at`.
- Ajout de la colonne `tournament_id` (clé étrangère vers `tournament`) dans la table `match` pour assigner un match à un tournoi.

La migration de 1.5.0 à 1.6.0 est automatique à l'ouverture de la base de données.

Version 1.7.0
-------------

La version 1.7.0 de la base de données ajoute la colonne suivante :

- Ajout de la colonne `last_visited_position` dans la table `match` pour mémoriser la dernière position visitée dans chaque match.

La migration de 1.6.0 à 1.7.0 est automatique à l'ouverture de la base de données.

.. note:: Depuis la version 0.10.0, toutes les migrations de bases de données
   sont effectuées automatiquement lors de l'ouverture d'un fichier de base de
   données.

