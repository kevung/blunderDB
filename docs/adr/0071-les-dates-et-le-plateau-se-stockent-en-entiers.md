# ADR-0071 — Les dates et le plateau se stockent en entiers

Statut : acceptée. Fait partie du schéma 2.31.0.
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
   commentaires d'en-tête et de pied, signés de son transcripteur, ou `XG`. L'export le
   copie par l'allow-list `issuance.CarriedMatchCommentColumns`, comme l'auteur d'un
   commentaire de position (ADR-0007).
7. **Les libellés d'action se stockent en codes entiers** : `analysis.best_cube_action`,
   `move.move_type` et `move.cube_action` (`INTEGER` des deux côtés). Un code tient lieu d'une
   chaîne exacte, jamais d'une normalisation : « No Double » et « NoDouble » gardent deux
   codes, et ce que l'on relit est octet pour octet ce qui a été écrit. NULL reste NULL, `""`
   est le code 0. La liste fixe (`domain.ActionLabels`, l'indice est le code) n'est
   qu'allongée. Un libellé qu'elle ignore n'est pas refusé — les importeurs en écrivent de
   libres, BMAB contient `Unknown(-1)` — : la base l'enregistre dans `action_label` sous un
   code ≥ `domain.FirstRegisteredActionCode` (1000). Toute lecture passe par
   `sqlshared.ActionLabelSQL` (un `CASE` des codes fixes, puis `action_label`), toute écriture
   par `sqlshared.ActionCodeFor` ; une requête qui compare à un libellé fixe passe par
   `ActionIsSQL` / `ActionNotInSQL`. Aucune requête n'épelle un code. L'étape SQLite reconstruit
   chaque colonne en `INTEGER` (une colonne déclarée `TEXT` stockerait le code en texte) ;
   PostgreSQL fait de même dans `034_weight_wave.sql`.

   **Sous PostgreSQL, `action_label` est une table de tenant**, pas une table globale : un
   libellé enregistré est du texte qu'un import a apporté (une chaîne libre d'un fichier), donc
   une donnée du tenant. Partagé, il fuirait vers les autres tenants, et la purge d'un tenant
   ne saurait pas s'il peut l'effacer. La table porte donc `tenant_id`, la RLS
   `tenant_isolation`, et figure dans `rlsTables` et `purgeOrder` ; un libellé est unique par
   `(tenant_id, label)`, l'enregistrement et la recherche par libellé filtrent le tenant. Le
   code, lui, vient d'une séquence (`action_label_code_seq`, à partir de 1000) et est unique
   pour toute la base : `ActionLabelSQL`, commune aux deux moteurs, résout un code sans filtre
   de tenant et n'atteint pourtant que la ligne que son propre tenant a enregistrée. Un
   `MAX(code) + 1` ne le permettait pas — sous RLS chaque tenant n'y voit que ses lignes et
   tirerait des codes déjà pris. SQLite n'a qu'un tenant et garde la table sans `tenant_id`.
8. **`analysis_engine` reste du texte.** C'est un libellé libre (nom et version du moteur,
   « XG Roller++ », « gammonNet 1.4 »…) que la recherche (`ae:`) et les statistiques filtrent
   par préfixe ou `LIKE` : un code ne se compare pas par préfixe sans relire tous les libellés.
   Sa cardinalité est faible mais ouverte, et sa colonne n'est pas indexée (décision 5) : le
   gain d'un code y est de quelques octets par analyse, au prix de chaque filtre réécrit.

## Conséquences

- Mesuré sur un échantillon à 2 % de BMAB (312 586 positions) migré puis `VACUUM` :
  292,6 Mo → 247,7 Mo hors `match`/`game` (**−15,3 %**) ; `position` 51,0 → 31,4 Mo,
  `idx_position_match_date` 11,6 → 4,0 Mo, `idx_analysis_creation_date` 8,5 → 4,0 Mo.
- Codes d'action, mesurés à part sur le même échantillon : 249,7 Mo → 245,7 Mo hors
  `match`/`game` (**−1,6 %**, −4,0 Mo) ; les 312 544 meilleures actions et les 332 152 types et
  actions de coup se relisent à l'identique, ligne par ligne. Le gain est sous l'estimation de
  `POIDS.md` § 3.4, qui comptait aussi `state` et `analysis_engine`.
- Une lecture d'action coûte un `CASE` d'une quarantaine de branches ; un libellé enregistré
  ajoute une sous-requête sur `action_label`, que `COALESCE` n'atteint que pour lui.
- Une instance qui ouvre la base en lecture seule (une autre tient le verrou d'écriture) ne
  peut pas la migrer sans écrire (ADR-0007) : si le schéma sur disque n'est pas le sien — le
  détenteur est un blunderDB plus ancien —, elle refuse l'ouverture avec un message qui nomme
  les deux versions, au lieu d'échouer requête par requête sur `action_label` ou `state`.
- Une base déjà en 2.31.0 (versions de développement) ne rejoue pas l'étape ; elle lit
  toujours ses anciennes valeurs, mais garde leur poids.
