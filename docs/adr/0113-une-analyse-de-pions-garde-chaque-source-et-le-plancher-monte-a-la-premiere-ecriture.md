# Une analyse de pions garde chaque source, et le plancher de lecteur monte à la première écriture qui en porte

Statut : acceptée. Amende l'ADR-0070 (règles 1 et 6).
Voir aussi : ADR-0013 (fusion d'une base importée), ADR-0019 (une seule échelle d'équité),
ADR-0096 (synchronisation par exports incrémentaux).

## Contexte

La fusion des analyses de pions garde, coup par coup, l'évaluation la plus profonde
(`engine.MergeCheckerMoves`) : quand XG et GNUbg, ou un rollout et un 2-ply, ont évalué le même
coup, l'autre évaluation est jetée. Le videau garde déjà une entrée par moteur
(`AllCubeAnalyses`). Un joueur qui importe le même match depuis XG et depuis GNUbg veut comparer
les deux analyses ; gammonGo le demande pour son sélecteur de source.

Garder les sources change le blob d'analyse : un décodeur 2.57 ne lit pas le nouveau champ.
L'ADR-0070 posait une version de blob unique et aucun bump de schéma pour le format.

## Décision

1. **Une source est un moteur à une profondeur**, un rollout étant une profondeur à part
   (`domain.CheckerSourceAnalysis`). Sa profondeur est le libellé de son coup le plus profond.
   `PositionAnalysis.AllCheckerAnalyses` garde une entrée par source quand **au moins deux**
   sources ont évalué la position, triées comme le videau (`engine.SortCheckerSources`). Une seule
   source n'a pas d'ensemble : la vue fusionnée est cette source.
2. **La vue fusionnée ne change pas.** `CheckerAnalysis` est calculée comme avant ; les sources
   l'accompagnent et ne la nourrissent jamais. `TestMergedViewUnchangedBySources` (corpus
   charlot, XG puis GNUbg et l'inverse, chaque import deux fois) et
   `TestRolloutAgainstTwoPlyKeepsBothSources` le tiennent.
3. **Une seule règle de source, dans `engine`** (`MergeCheckerSources`). Une source qui arrive
   remplace sa propre entrée ; une source absente est ajoutée, plus courte comprise ; une liste
   déjà présente dans la vue fusionnée (un lecteur qui réenregistre ce qu'il a chargé) n'est pas
   une source nouvelle. L'import d'un match (`ingest.mergeAnalysis`, `deepenAnalysis`),
   l'enregistrement d'une analyse (`storage.MergeSavedAnalysis`) et gammonNet
   (`gammonnet.Supersede`) l'appellent sans condition à elles. La fusion d'une base `.db`
   importée (`MergeImportedAnalysis`, ADR-0013) reste telle quelle : elle garde l'analyse
   principale du destinataire, sources comprises.
4. **Une analyse antérieure à 2.58 n'est pas réécrite.** Celle d'un seul moteur est sa propre
   source, telle quelle, étiquetée comme ses colonnes de provenance (`analysis_engine`,
   `analysis_depth`), même quand ses coups portent plusieurs profondeurs (XG déroule ses meilleurs
   coups et laisse les autres moins profonds) ; rien n'est reclassé ni recalculé. Une vue qui mêle
   plusieurs moteurs ne donne **aucune** source : ses rangs et ses erreurs ont été mesurés depuis
   le meilleur coup d'un autre moteur, et ce qu'a dit chaque moteur seul est perdu. Un client le
   reconnaît : `allCheckerAnalyses` absent et plusieurs `analysisEngine` parmi les coups de
   `checkerAnalysis` (`api_reference.rst`).
5. **Amende l'ADR-0070 §1 : deux versions de blob.** La version 2 est la charge de la version 1
   suivie des sources, signalée par le drapeau `binFlagCheckerSources`. Un blob sans sources
   s'écrit toujours en version 1, octet pour octet ; un ensemble vide s'encode comme aucun.
   Seule une position évaluée par plusieurs sources demande un lecteur de version 2.
6. **Amende l'ADR-0070 §6 : bump de schéma, plancher monté par l'écriture.** `DatabaseVersion`
   passe à 2.58.0, par une étape de migration vide (aucune ligne n'est relue).
   `domain.MinReaderVersion` reste 2.50.0 ; `domain.CheckerSourcesReaderVersion` vaut 2.58.0 et
   le plancher `min_reader_version` y monte **dans la transaction qui écrit le premier blob de
   version 2** (`sqlshared.RaiseReaderFloorForAnalysis`), sur chaque chemin d'écriture : `write`
   et `SaveExportedAnalysis` en SQLite, `write`, `SaveMany` et `SaveFresh` en PostgreSQL. Il ne
   monte jamais à l'ouverture. Une bibliothèque qui n'a qu'une source par position reste lisible
   par 2.50 à 2.57 ; un bureau qui ouvre une base ne la ferme pas aux versions antérieures sans
   qu'elle ait changé de contenu. Une montée à l'ouverture était l'autre choix : elle refuse les
   anciens lecteurs sur des bases qu'ils liraient sans perte.
7. **Le rôle PostgreSQL du démon écrit dans `metadata`.** La table est globale, hors RLS
   (`rlsTables`) ; la montée du plancher y fait un `INSERT … ON CONFLICT DO UPDATE` : le rôle
   doit y avoir `SELECT`, `INSERT` et `UPDATE`. Sans ces droits, la première écriture d'une
   analyse à plusieurs sources échoue entière plutôt que d'écrire un blob que le plancher
   n'annonce pas. `TestCheckerSources_PostgresRLS` le tient sous un rôle restreint.

## Conséquences

- **Taille**, mesurée sur le corpus charlot importé depuis XG puis GNUbg (285 analyses, dont
  180 à deux sources) : un blob à plusieurs sources pèse en moyenne 643 octets contre 455 sans
  ses sources (+41 %), au pire 4 413 contre 3 044 (+45 %). La migration ne lit ni n'écrit
  aucune ligne : sa durée ne dépend pas de la base.
- **Synchronisation (ADR-0096) et pairs 2.57.** Le plancher se vérifie à l'ouverture. Un export
  incrémental qui porte un blob de version 2 a son plancher monté : un pair 2.57 refuse de
  l'importer, sans perte. Une analyse qu'un pair 2.57 écrit ne porte que sa vue fusionnée ;
  arrivée par synchronisation sur une position que le destinataire n'avait pas, elle n'a pas de
  sources, et si elle mêle des moteurs, aucune ne se dérive. Un processus 2.57 qui tenait la
  base ouverte avant la montée du plancher (un démon PostgreSQL pendant une mise à jour par
  étapes) ne la voit pas : s'il réécrit une analyse par `Save`, il remplace le blob par sa vue
  fusionnée et **les sources sont perdues**. Mettre à jour tous les démons d'une base avant
  d'importer une seconde source l'évite.
- `TestBinaryFormatCoversEveryField` compte les champs de la version 2 ; un nouveau champ
  demande une version 3.

## Garde

`storagetest` : `Analysis/KeepsCheckerSources`, `Analysis/CheckerSourcesRaiseFloorOnMerge`,
`Analysis/CheckerSourcesRaiseFloorOnSaveMany` (deux backends), `CheckerSources` dans
l'isolation des tenants ; `TestCheckerSources_PostgresRLS` ; `TestMergedViewUnchangedBySources`
(`testdata/merged_view.golden`) ; `TestMigrate_2_57_0_to_2_58_0_KeepsAnalysesAsTheirSingleSource`.
