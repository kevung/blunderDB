.. _annexe_db_migration:

Annexe: Schéma de la base de données
====================================

Une base de données blunderDB est un simple fichier SQLite (extension *.db*).
En l'absence de blunderDB, elle peut ainsi être ouverte et inspectée avec
n'importe quel éditeur ou navigateur de fichiers SQLite.

Versionnage et migrations
-------------------------

Le schéma de la base de données est **versionné**. La version courante du
schéma est **2.13.0** ; elle est indépendante de la version de l'application et
n'est incrémentée que lorsque la structure interne évolue. La version du schéma
d'une base ouverte est visible dans le panneau **Métadonnées** (commande
``meta``).

.. important::
   Sauvegardez toujours votre fichier *.db* avant d'ouvrir une base créée avec
   une version antérieure de blunderDB.

.. note::
   Depuis la version 0.10.0, **toutes les migrations de schéma sont effectuées
   automatiquement** à l'ouverture d'une base de données. Aucune commande de
   migration manuelle n'est nécessaire. La migration s'effectue sur place :
   une base migrée vers un schéma récent ne peut plus être ouverte par des
   versions plus anciennes de blunderDB, d'où l'importance de la sauvegarde
   préalable.

Principales tables
------------------

Le schéma courant s'articule autour des tables suivantes :

* **Positions et analyses** : ``position`` (les positions, dédupliquées),
  ``analysis`` (les données d'analyse associées) et ``comment`` (les
  commentaires).

* **Matchs** : ``match``, ``game``, ``move`` et ``move_analysis`` stockent les
  matchs importés, leurs parties, leurs coups et l'analyse de chaque coup.

* **Collections** : ``collection`` et ``collection_position`` (table de liaison)
  regroupent des positions choisies manuellement.

* **Tournois** : ``tournament`` regroupe des matchs en tournois.

* **Répétition espacée (Anki)** : ``anki_deck``, ``anki_card`` et
  ``anki_review_log`` gèrent les paquets, les cartes et le journal des
  révisions (algorithme FSRS).

* **Historique et divers** : ``command_history`` et ``search_history``
  conservent l'historique des commandes et des recherches, ``filter_library``
  la bibliothèque de filtres, et ``metadata`` les métadonnées de la base
  (dont la version du schéma).

Principes de conception
-----------------------

À partir du schéma 2.0.0, plusieurs choix de conception accélèrent la recherche
et réduisent la taille du fichier :

* **Déduplication des positions** : chaque position porte un hash de Zobrist et
  un index unique, de sorte qu'une même position rencontrée dans plusieurs
  imports n'est stockée qu'une seule fois.

* **Colonnes de filtrage dénormalisées** : des critères fréquemment recherchés
  (type de décision, dés, différence de course, pions sortis, pions arriérés,
  erreur de coup ou de videau, chances de gain…) sont précalculés en colonnes
  dédiées pour un filtrage rapide.

* **Préfiltre par bitboards** : des colonnes d'occupation et de masques de
  points permettent un préfiltre entier très rapide lors des recherches de
  motifs de structure.

* **Stockage compact** : les positions sont encodées de façon compacte et les
  données d'analyse sont compressées (zlib), ce qui réduit fortement la taille
  du fichier.

* **Journalisation WAL** : le mode WAL et des PRAGMA ajustés améliorent les
  performances en lecture et en écriture.
