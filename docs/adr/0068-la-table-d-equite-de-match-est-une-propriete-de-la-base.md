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
3. **Chaque analyse porte sa table** : `analysis.met_digest`, `NULL` pour Kazaross-XG2. Une
   analyse gammonNet écrit l'empreinte de la table courante au moment du calcul. Une analyse
   importée (XG, GNUbg, BGBlitz) écrit `NULL` : ces fichiers ne disent pas leur table, et leur
   réglage par défaut est Kazaross-XG2.
4. **« MET différente » est un état lu, jamais écrit.** Une analyse à un score de match dont
   l'empreinte n'est pas celle de la table courante est montrée « MET différente » et sortie
   des comparaisons (moyennes d'erreur, PR, classements, côte à côte de deux joueurs). Une
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
  `analysis.met_digest` ajoutée `NULL` (aucune ligne réécrite), migration
  `033_met_progress_move_error.sql` côté PostgreSQL. Contrat :
  `storage.MatchEquityTableStore` (`Save` idempotent sur l'empreinte, `List`, `Current`,
  `SetCurrent`).
- Le moteur prend la table de la base au lieu d'une constante : `MatchStateFromPosition`
  et la recherche reçoivent la table à utiliser ; la parité avec le C reste mesurée sur
  Kazaross-XG2.
- L'import du `.xml`, l'écriture de `met_digest` par le lot et l'état « MET différente »
  dans l'interface et les statistiques restent à livrer.
