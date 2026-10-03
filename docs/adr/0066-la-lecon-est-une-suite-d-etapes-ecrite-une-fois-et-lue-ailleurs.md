# ADR-0066 — La Leçon est une suite d'étapes écrite une fois et lue ailleurs

Statut : acceptée.
Voir aussi : ADR-0001 (le hachage identifie la position), ADR-0007 (le receveur n'écrit rien),
ADR-0046 (la file d'étude), ADR-0065 (le coach écrit chez lui) ; `tasks/plan-2026-10b.md`, I.21.

## Contexte

Un coach veut remettre à son élève un parcours : des étapes ordonnées, chacune dite en
quelques lignes et montrant une Collection, une Position, les deux ou rien. Deux objets
existants s'en approchent sans convenir. La **file d'étude** (ADR-0046) est calculée : elle
liste les positions dont l'erreur passe un seuil, dans l'ordre que le calcul donne, sans texte
et sans auteur. Une **Collection** est un ensemble ordonné de positions, sans texte d'étape
ni suite de plusieurs ensembles.

## Décision

1. **Un objet à part : la Leçon.** Une Leçon porte un nom, une description et des Étapes dans
   un ordre explicite ; une Étape porte un titre, un texte et, au choix, une `CollectionID`
   et une `PositionID` (0 = aucune). On ne réutilise ni la file d'étude (calculée, sans texte)
   ni la Collection (pas de texte par étape, pas de séquence d'ensembles) : les étendre
   mêlerait trois sens dans un seul mot. Le mot « parcours » reste libre pour le vocabulaire
   courant ; le terme du domaine est Leçon.
2. **Elle voyage dans un `.dbx`, par la sélection d'export.** L'export choisit les Leçons
   (`Selection.LessonIDs` / `AllLessons`) et, sous `LessonContents`, ajoute ce que leurs
   Étapes montrent (les Collections et les Positions) : un fichier qui contient une Leçon
   contient de quoi la lire. Les métadonnées de la base passent toujours par
   `issuance.Carried` (liste d'autorisation, jamais d'exclusion) ; la Leçon, elle, n'entre que
   par la sélection, jamais par défaut dans un export partiel.
3. **À l'import, le nom décide.** `ingest.MergeLessons`, la même règle pour les deux formes
   de `.db` natif : une Leçon dont le nom existe déjà chez le receveur est laissée telle
   quelle (réimporter ne change rien, une Leçon que le receveur a retouchée n'est jamais
   écrasée) ; toute autre est créée avec ses Étapes. La position d'une Étape est
   retrouvée par son hachage (ADR-0001), sa Collection par son nom.
4. **Une Étape retient sa position.** `lesson_step.position_id` rejoint les motifs de
   `positionIsHeldSQL` (écrit trois fois, identique au dialecte près : `database/db_match.go`,
   `sqlite/matches_sqlite.go`, `postgres/matches_postgres.go`) : supprimer le match d'où
   venait une position ne purge pas celle qu'une Leçon montre. À l'inverse, supprimer une
   Collection ou une Position que la Leçon montre laisse l'Étape et son texte : le texte du
   coach reste lisible, l'écran dit que l'objet manque.
5. **Le receveur n'écrit rien.** Lire une Leçon n'enregistre ni la progression, ni l'étape
   atteinte, ni l'ouverture (ADR-0007). La barre de lecture du bureau tient l'étape courante
   en mémoire de session seulement.
6. **La suppression est définitive.** Il n'y a pas de corbeille pour les Leçons : `delete`
   retire la Leçon et ses Étapes, jamais les Collections ni les Positions. Une corbeille
   existerait pour une seule entité ; l'autre barrière est le fichier `.dbx` déjà remis, qui
   est une copie. Écart assumé par rapport aux Collections, à rouvrir si un utilisateur perd
   une Leçon qu'il n'avait pas exportée.
7. **Une surface, trois modes.** Le domaine, le stockage (SQLite et PostgreSQL, suite de
   contrat commune), les routes `/v1/lessons.*`, la CLI `lesson`, les outils MCP
   `list_lessons` / `lesson` et la commande `:le [N]` du bureau lisent les mêmes méthodes de
   `storage.LessonStore`.

## Conséquences

- Schéma 2.29.0 : tables `lesson` et `lesson_step`, migration `031_lesson.sql` côté PostgreSQL.
- L'éditeur graphique et l'export de Leçons depuis le dialogue d'export du bureau ne sont pas
  livrés : l'écriture passe par la CLI et les routes, l'export par `lesson export`.
