# ADR-0088 — La maintenance est un état de santé, une ligne par opération, que rien ne lance seul

Statut : acceptée.
Voir aussi : ADR-0005 (le daemon ne s'authentifie pas), ADR-0007 (rien n'est écrit chez le
destinataire), ADR-0013 (une évaluation comble un trou), ADR-0046 (les réglages de la
bibliothèque), ADR-0070 (le format binaire des analyses), ADR-0085 (la grammaire d'un panneau).

## Contexte

Les passes qui recalculent ce que la base dérive — colonnes d'analyse, phases, sentinelle
Crawford, `match_stats`, `move.error_mp`, réencodage, vacuum, analyse gammonNet — sont
éparpillées. La CLI les a presque toutes, la GUI deux, le daemon les expose en routes isolées.
La plupart tiennent `Database.mu` pendant toute la passe, sans contexte ni progression. Sur une
base de 16 millions de positions, certains comptes coûtent un balayage complet, et l'un d'eux
tourne à chaque ouverture des réglages. Enfin, l'ouverture d'une base remplit `match_stats`,
ce qui écrit dans la base à l'ouverture.

## Décision

1. **Un état de santé, une ligne par opération** : compteur, action, dernière exécution. Le
   paquet `pkg/blunderdb/maintenance` sert la CLI (`blunderdb maintenance`), la GUI (onglet
   Maintenance des réglages) et le daemon (`/v1/maintenance.*`). Il est au-dessus de `storage`
   et n'a pas de SQL propre.
2. **Le compteur, la liste d'ids et l'action d'une ligne partagent un seul prédicat** : le
   compte est la longueur de la liste quand la ligne en a une. Un clic sur un compte charge
   cette liste d'ids (ADR-0085), sans jeton de recherche.
3. **Un compte déclare son coût** (`free`, `seconds`, `scan`, `none`). Seuls les gratuits se
   calculent en lisant l'état ; les autres se demandent, un par un.
4. **Rien ne se lance seul** : ni à l'ouverture de la base, ni à l'import, ni en lisant l'état.
   Le remplissage de `match_stats` quitte l'ouverture : les lectures remplissent ce qui manque,
   et la ligne `stats.matches` le reste.
5. **Une opération à la fois par tenant**, annulable par contexte, avec progression.
   L'emplacement est partagé avec le lot gammonNet et les rollouts : le daemon répond 409,
   le bureau (`Database.acquireJob`) refuse de même. Une passe annulée qui finit son lot
   n'est pas un refus : celle qui la remplace attend qu'elle rende la main. Changer de
   base arrête la passe de maintenance (`StopMaintenance`).
   Le paquet ne prend jamais de verrou ; `Opts.Before` encadre chaque lot que la passe
   valide, et le wrapper de bureau y passe `Database.mu`, qui n'est pas réentrant.
6. **La dernière exécution s'écrit à la fin d'une action demandée**, quel qu'en soit l'issue,
   dans la table des réglages de la bibliothèque (SQLite `metadata`, PostgreSQL
   `library_settings`, par tenant), sous les clés `maintenance.<opération>`. Elle est hors de
   `issuance.CarriedMetadataKeys` : un export ne l'emporte pas. Pas de hausse de
   `DatabaseVersion` : ce sont des lignes, pas un schéma.
7. **Le vacuum reste `/ops/maintenance.vacuum`** et n'existe pas en PostgreSQL (la ligne y est
   `applicable: false`).
8. **« Tout recalculer »** enchaîne colonnes d'analyse → phases → Crawford → erreurs des coups
   → `match_stats`, sans gammonNet (ADR-0013) ni vacuum (destructif, demande de la place).

## Conséquences

- `repair`, `vacuum` et `reencode` deviennent des alias de `maintenance run` à mesure que les
  lignes portent leurs actions.
- La GUI regagne la parité sur phases, Crawford, réencodage et intégrité.
- La purge des positions non retenues réutilisera `positionIsHeldSQL` de chaque backend.
- Écarté — un jeton de recherche par ligne : la grammaire est partagée avec le frontend, et les
  ids donnent l'exactitude à moindre coût.
- Écarté — des comptes mis en cache dans la base : ce serait écrire sur une lecture.
- Le prix de la décision 4 : la première lecture de statistiques après une migration est lente.

## Garde

- `storagetest` : `LibrarySettings/MaintenanceLogRoundTrip` et l'isolation `MaintenanceLog` par
  tenant.
- `internal/server` : 409 pour une seconde opération du même tenant ; `/v1/maintenance.run`
  inscrite dans `streamingCustomPaths`.
- `internal/cli/parity_test.go` : chaque méthode de `Database` a sa sous-commande et sa route.
