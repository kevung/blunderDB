# Le résultat d'un match monte le plancher de lecteur à la première écriture qui en porte

Statut : acceptée. Applique l'ADR-0113 §6 et §7 au résultat de match.
Voir aussi : ADR-0070 (format et plancher), ADR-0096 (synchronisation par exports incrémentaux).

## Contexte

Le schéma 2.59.0 conserve le résultat d'un match : `final_score_1`, `final_score_2` et `winner`,
calculés à l'import depuis la dernière partie (`domain.MatchResultOf`). La migration 2.58 → 2.59
les calcule une fois pour les matchs déjà en base ; rien ne les recalcule ensuite. Un binaire
2.50 à 2.58 ne connaît pas ces colonnes : avec le plancher à 2.50.0, il ouvre une base 2.59 et y
importe des matchs dont le résultat reste NULL pour toujours, sans que la liste ni la recherche
le disent autrement que « résultat inconnu ».

## Décision

1. **Un plancher propre au résultat.** `domain.MatchResultReaderVersion` vaut 2.59.0.
   `domain.MinReaderVersion` reste 2.50.0 et `domain.CheckerSourcesReaderVersion` 2.58.0.
2. **Le plancher monte par l'écriture.** `min_reader_version` monte à 2.59.0 **dans la
   transaction qui écrit le premier match au résultat connu**
   (`sqlshared.RaiseReaderFloorForMatch`), sur chaque chemin : `Save`, `Reinstate` et
   `ReplaceHeader` du `MatchStore`, dans les deux backends ; l'export SQLite
   (`ingest`, `sqlite.RaiseReaderFloorForMatch`) qui insère ses matchs hors du store. L'import
   d'un fichier, la fusion d'une base `.db`, un changeset et `migrate` vers PostgreSQL passent
   par `Save` ou `ReplaceHeader`. Un match au résultat inconnu ne monte rien ; un
   `ReplaceHeader` sans résultat garde celui qui est stocké et ne monte rien ; l'échange des
   sièges retourne un résultat existant sans en écrire de nouveau.
3. **Jamais à l'ouverture, jamais à la migration.** Le calcul des résultats par la migration
   n'est pas une écriture de match : une base migrée reste lisible par 2.50 à 2.58 tant que ce
   binaire n'y a écrit aucun résultat. Le plancher ne baisse jamais : après une analyse à
   plusieurs sources et un résultat, dans un ordre ou dans l'autre, il vaut 2.59.0.
4. **PostgreSQL : même chemin que l'ADR-0113 §7.** `metadata` est globale, hors RLS ; le rôle du
   démon y a déjà `SELECT`, `INSERT` et `UPDATE`. Le rôle de chargement en masse
   (`blunderdb_loader`), qui n'y avait que `SELECT`, reçoit aussi `INSERT` et `UPDATE`
   (`ApplyLoaderRole`) : il importe des matchs au résultat connu. Sans ces droits, l'enregistrement d'un match
   au résultat connu échoue entier plutôt que d'écrire un résultat que le plancher n'annonce pas.

## Alternatives écartées

- **Recalculer le résultat à la lecture** (ou à chaque ouverture) des matchs restés NULL : une
  lecture qui écrit, une liste qui coûte une requête de plus sur les parties, et un résultat
  qui dépend de la version qui lit. Le résultat se calcule une fois, à l'écriture.
- **Monter le plancher à l'ouverture**, ou en changeant `MinReaderVersion` : contraire à
  l'ADR-0113 §6 — un bureau qui ouvre une base ne la ferme pas aux versions antérieures sans
  qu'elle ait changé de contenu.

## Conséquences

- La première importation d'un match terminé par un binaire 2.59 ferme la base aux binaires
  2.50 à 2.58 ; un export qui porte un match au résultat connu est refusé par un pair 2.58, sans
  perte (le refus précède toute écriture).
- Une base migrée de 2.58 garde ses résultats calculés et son plancher : un binaire 2.58 peut
  encore y importer des matchs, qui restent au résultat inconnu jusqu'à ce qu'un binaire 2.59
  réimporte leur fichier. C'est le prix de la règle 3 ; mettre à jour les binaires qui écrivent
  dans une base partagée avant d'y importer l'évite.
- Une base PostgreSQL 2.59 sert des démons de même version ; un démon 2.58 qui la tenait ouverte
  avant la montée ne la voit pas, comme pour les sources de pions.

## Garde

`storagetest` : `Match/ResultRaisesFloorOnSave`, `Match/ResultRaisesFloorOnReplaceHeader`,
`Match/ResultRaisesFloorOnReinstate`, `Match/ResultFloorAfterSources`,
`Match/ResultFloorBeforeSources` (deux backends, refus d'un lecteur 2.58 par
`storage.CheckReaderFloorAs`) ; `TestMatchResultFloor_PostgresRLS` ;
`TestMigrate_2_58_0_to_2_59_0_MatchResult` (plancher inchangé par la migration) ;
`TestExportRaisesFloorForMatchResult` ; `bulkload` sous le rôle de chargement.
