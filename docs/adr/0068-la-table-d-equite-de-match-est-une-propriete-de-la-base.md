# ADR-0068 — La table d'équité de match est une propriété de la base

Statut : acceptée. Amende ADR-0016 (règle 6, « pas de réglage »).
Voir aussi : ADR-0007 (le receveur n'écrit rien, liste blanche `issuance.Carried`), ADR-0013
(les analyses importées sont intouchables), ADR-0016 (le référentiel est une propriété de la
position), ADR-0019 (une seule échelle d'équité sort du moteur) ; issue #271, fiche I.15.

## Contexte

ADR-0016 refuse tout réglage de la table d'équité de match (MET) : un interrupteur global
rendrait deux analyses d'une même position incomparables selon une case cochée au moment du
lot. Le refus tient pour un réglage *global* ; il ne répond pas au joueur qui étudie avec une
autre table que Kazaross-XG2 (une table de club, Rockwell-Kazaross, une table maison
calculée par gnubg) et dont toutes les analyses à un score de match sont alors fausses pour
lui. La comparabilité qu'ADR-0016 protège n'exige pas une seule table : elle exige que chaque
nombre dise avec quelle table il a été calculé.

## Décision

1. **La MET est choisie par base, pas par l'application.** Une base (un tenant côté
   PostgreSQL) tient zéro, une ou plusieurs tables importées (`match_equity_table`) et au
   plus une *courante* (`is_current`). Aucune courante : la table intégrée Kazaross-XG2.
   Changer d'application, de machine ou de base ne change pas la table d'une base.
2. **Une table s'importe depuis un `.xml` gnubg** et s'identifie par l'empreinte de ses
   *valeurs* (`digest`), écrites sous une forme canonique fixée une fois par l'importeur :
   deux fichiers qui ne diffèrent que par la mise en page, les commentaires ou le nom
   désignent la même table, et deux bases qui la tiennent s'accordent sur ce qui est
   comparable. Le texte du fichier est gardé (`source`) pour être relu et réexporté. Une
   table importée dont l'empreinte est celle de Kazaross-XG2 se résout en la table intégrée.
3. **Chaque analyse porte sa table** : `analysis.met_id`, l'identifiant de la ligne
   `match_equity_table` de la base, `NULL` pour Kazaross-XG2. Une analyse gammonNet écrit la
   table courante au début du lot qui la calcule. Une analyse
   importée (XG, GNUbg, BGBlitz) écrit `NULL` : ces fichiers ne disent pas leur table, et leur
   réglage par défaut est Kazaross-XG2.
4. **« MET différente » est un état lu, jamais écrit.** Une analyse à un score de match dont
   l'empreinte n'est pas celle de la table courante est montrée « MET différente » et sortie
   des comparaisons (moyennes d'erreur, PR, classements, côte à côte de deux joueurs) :
   décision par décision, et match entier pour les agrégats par match (`match_stats`), qui
   ne retiennent un match que si toutes ses analyses ont la table courante. Une
   analyse en money n'est jamais différente : la MET n'y intervient pas (ADR-0016, règle 3).
   Changer la table courante ne réécrit aucune analyse (ADR-0013) ; il change seulement ce
   que les comparaisons retiennent.
5. **Une table voyage avec les analyses qui la citent.** Un export qui emporte une analyse
   d'empreinte non nulle emporte la ligne `match_equity_table` correspondante, sans son
   drapeau courant : à l'import, les tables sont fusionnées par empreinte et arrivent non
   courantes. La table courante n'est pas une métadonnée portée (`issuance.Carried`) :
   ouvrir ou importer un fichier ne change pas la table du receveur (ADR-0007).
6. **Ce qui reste d'ADR-0016, règle 6** : il n'y a toujours pas d'interrupteur global ni de
   réglage au moment du lot. La table d'un calcul est celle de la base qui le reçoit, et
   elle est écrite dans l'analyse.

## Conséquences

- Schéma 2.31.0 : table `match_equity_table` (empreinte unique par tenant), colonne
  `analysis.met_id` ajoutée `NULL` (aucune ligne réécrite), index partiel `idx_analysis_met`
  sur les seules analyses étiquetées. Contrat `storage.MatchEquityTableStore` (`Save`
  idempotent sur l'empreinte, `List`, `Current`, `SetCurrent`, `TagAnalyses`, `OfAnalysis`) ;
  le paquet `mets` le sert aux trois modes (GUI : onglet *gammonNet* et badge du panneau
  d'analyse ; CLI : `met` ; démon : routes `met.*`).
- Le moteur prend la table de la base au lieu d'une constante : `engine.MET` (nil =
  Kazaross-XG2, bit pour bit), portée par `gammonnet.MatchState` jusqu'à `metAfter`. Seules
  les tables gnubg explicites sont lues ; au-delà de leur longueur, la table intégrée
  répond. La parité avec le C reste mesurée sur Kazaross-XG2.
- L'empreinte est un SHA-256 d'une forme canonique des valeurs (`blunderdb-met/1`) ; le
  `Kazaross-XG2.xml` de gnubg a celle de la table intégrée.
- Règle 5 : l'export SQLite, l'import d'une base (`ingest.DBImporter`) et la fusion de base
  du GUI/CLI (`CommitImportDatabase`) copient chaque table citée par `mets.Carrier`, qui la
  range par `Save` (dédoublonnée par empreinte, jamais courante) et remappe le `met_id`. À
  la fusion d'une analyse, le verdict gardé garde la table de son côté (`mets.AfterMerge`).
  L'export NDJSON écrit chaque table citée une fois, sur une ligne à elle
  (`matchEquityTable`) avant la première analyse qui la cite (`met`, son id dans le flux) ;
  l'import la passe au même `Carrier`, et refuse une analyse qui cite une table absente du
  flux plutôt que de la lire sous Kazaross-XG2. La migration SQLite → PostgreSQL copie
  toutes les tables, la courante et le `met_id` de chaque analyse ; la corbeille garde le
  `met_id` d'une position supprimée et le rend à la restauration.
- Le `met_id` suit le verdict : l'upsert d'une analyse le remet à `NULL` dès que le verdict
  des colonnes n'est plus celui de gammonNet (XG, GNUbg, rollout), et un calcul gammonNet
  écrit l'analyse et sa table dans une seule transaction (`rollouts.SaveValuedAnalysis`,
  `Database.saveAnalysisAt`) : un échec n'en laisse aucune des deux.
- L'évaluation en direct (panneau, `gammonnet.evaluate`, grille de videau) est valorisée
  avec la table courante de la base, comme le lot : le panneau ne montre jamais un nombre
  d'une autre table que l'analyse qu'il remplacerait. La commande `cubematrix` de la CLI,
  qui n'ouvre pas de base, reste sur Kazaross-XG2.
- Les rollouts et les conversions MWC des importeurs restent sur Kazaross-XG2.
