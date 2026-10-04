# ADR-0070 — Les dates et le plateau se stockent en entiers

Statut : acceptée. Complète le schéma 2.31.0 (non publié) sans nouveau numéro.
Voir aussi : ADR-0001 (identité par Zobrist, inchangée), ADR-0068 (table d'équité de match) ;
`tasks/plan-grosses-bases-2026-10/POIDS.md` § 3.4 et § 5, fiche `SCHEMA-2-31.md`.

## Contexte

Sur une bibliothèque BMAB (15,6 M positions), la 2.30 stockait `position.match_date` en texte
`AAAA-MM-JJ hh:mm:ss +0000 UTC` (29 o, dans la ligne et dans le plus gros index de la base),
`analysis.creation_date` en texte de 19 o, le plateau `position.state` en tableau JSON de
≈ 63 o, et indexait `analysis_engine` et `analysis_depth` sans qu'aucun plan de requête ne
lise ces index.

## Décision

1. **Une date stockée par blunderDB est un instant en secondes Unix, UTC.** Cela vaut pour
   `position.match_date` et `analysis.creation_date` en SQLite (colonnes `INTEGER`).
   PostgreSQL garde `TIMESTAMPTZ`, qui est déjà un entier sur 8 octets compté en UTC : même
   convention, aucun gain à convertir. Une borne de jour d'une recherche (`md:`) est minuit
   UTC ; `sqlshared.Dialect.InstantArg` lie l'instant sous la forme de chaque moteur.
2. `match.match_date` reste le texte que le pilote écrit pour un `time.Time` (la table
   `match` pèse 10 Mo). `sqlite.UnixFromMatchDateSQL` le convertit en SQL, décalage de fuseau
   compris, là où la date de la position en dérive.
3. **Le plateau est 28 octets signés** (`engine.EncodeBoardState`) : les 28 valeurs du tableau
   compact, un `int8` chacune. Aucune ne vaut `[` ni `{`, si bien que le premier octet sépare
   les trois formes, et `engine.DecodeBoardCompact` les lit toutes. SQLite garde la colonne
   déclarée `TEXT` (l'affinité TEXT stocke un BLOB tel quel ; base migrée et base neuve ont
   ainsi une seule déclaration) ; PostgreSQL passe en `BYTEA`. Le hash Zobrist ne lit pas
   cette colonne.
4. `analysis.met_digest TEXT` devient `analysis.met_id`, référence entière à
   `match_equity_table`. NULL reste la table intégrée.
5. `idx_analysis_engine` et `idx_analysis_depth` sont retirés ; la passe de provenance lit
   l'index partiel `idx_analysis_provenance_pending` (`WHERE analysis_engine IS NULL`), vide
   une fois la passe finie.
6. Le commentaire d'un match est signé : `match.comment_author`. Un fichier XG donne ses
   commentaires d'en-tête et de pied, signés de son transcripteur, ou `XG`.

## Conséquences

- Mesuré sur un échantillon à 2 % de BMAB (312 586 positions) migré puis `VACUUM` :
  292,6 Mo → 247,7 Mo hors `match`/`game` (**−15,3 %**) ; `position` 51,0 → 31,4 Mo,
  `idx_position_match_date` 11,6 → 4,0 Mo, `idx_analysis_creation_date` 8,5 → 4,0 Mo.
- Les énumérations texte (`best_cube_action`, `move.move_type`, `move.cube_action`,
  `analysis_engine`) restent du texte : `analysis_engine` est un libellé libre filtré par
  préfixe, et les trois autres sont lues par une centaine de requêtes partagées. Leur passage
  en codes entiers est un chantier séparé.
- Une base déjà en 2.31.0 (versions de développement) ne rejoue pas l'étape ; elle lit
  toujours ses anciennes valeurs, mais garde leur poids.
