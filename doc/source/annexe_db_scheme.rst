.. _annexe_db_migration:

Annexe: Schéma de la base de données
====================================

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
