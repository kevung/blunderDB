# ADR-0067 — Les métadonnées source d'un match voyagent avec lui, et ses règles de session font foi

Statut : acceptée.
Voir aussi : ADR-0007 (le receveur n'écrit rien, liste blanche `issuance.CarriedMetadataKeys`), ADR-0028
(Jacoby et Beaver hors du hachage), ADR-0045 (la transcription réécrit un match) ;
`tasks/plan-grosses-bases-2026-10/`.

## Contexte

Un fichier eXtreme Gammon dit des joueurs et de la session plus que l'en-tête que blunderDB
retenait : un Elo et une expérience par joueur, le transcripteur, les règles Jacoby et Beaver,
la version du format. Le schéma 2.30.0 a les colonnes sur `match` (`player1_elo`,
`player2_elo`, `player1_experience`, `player2_experience`, `transcriber`, `has_jacoby`,
`has_beaver`, `engine_version`). Restaient trois questions : où vit la règle de session quand
la position a déjà ses propres `has_jacoby`/`has_beaver` ; ce qu'emporte un export natif ;
ce que devient un match importé avant que ces colonnes existent.

## Décision

1. **La règle de session est celle du match.** `match.has_jacoby`/`has_beaver` font foi :
   c'est la session qui se joue sous ces règles. Les colonnes de même nom sur `position` en
   sont la copie pour la recherche de positions, écrite à l'import pour une partie libre
   seulement (`ingest.copySessionRules`) — à un score de match, aucune des deux règles ne
   s'applique et la position n'en dit rien. Aucune n'entre dans le hachage ; le flux de clés
   d'`engine.init` est inchangé (ADR-0028).
2. **Inconnu n'est pas zéro.** Les champs sont des pointeurs dans `domain.Match` et NULL en
   base : un fichier qui ne dit rien (MAT, gnubg) laisse tout à NULL ; un Elo de 0 et un
   joueur sans classement ne sont pas le même fait. XG écrit un Elo et une expérience pour
   chacun, classé ou non : on les garde tels quels.
3. **L'export emporte les colonnes par liste blanche.** `issuance.Carried` filtre des clés de
   métadonnées, pas des colonnes ; les colonnes de `match` ont leur propre liste,
   `exportedMatchColumns`, et `notExportedMatchColumns` nomme les autres avec leur raison
   (`id`, `import_batch_id`, `dice_hash`, `direction_match_id`). Un test échoue tant qu'une
   colonne nouvelle n'est rangée dans aucune des deux : jamais d'export par exclusion. Les
   métadonnées source sont dans la liste blanche : c'est ce que le fichier disait du match,
   elles le suivent. Le receveur les relit en ouvrant l'export, par le même
   `MatchStore.Get` ; l'import d'une base dans une autre ne porte pas de matchs et n'a donc
   rien à relire.
4. **Réimporter complète sans écraser.** Un doublon exact ou un enrichissement inter-format
   donne au match stocké chaque champ qu'il ignore et que le fichier dit
   (`domain.FillSourceMetadata`), sans toucher à ce qu'il porte. Un corpus importé avant
   2.30.0 gagne ses Elo en réimportant ses fichiers. Une transcription qui corrige un match
   importé garde les métadonnées du fichier.
5. **Ce qui n'a pas de colonne n'est pas importé** : commentaires d'en-tête et de pied de
   match, horloge, table d'équité. La version du programme est celle du format de fichier
   (« eXtreme Gammon, file format 30 ») : un `.xg` ne nomme pas la version d'eXtreme Gammon
   qui l'a écrit, et les chaînes de son en-tête GDF sont le titre du match.

## Conséquences

- GUI (onglet Infos de la fiche du match), CLI (`match --format text|summary`, et `json`) et
  serveur lisent les mêmes champs, par le contrat de stockage.
- `MatchStore.ReplaceHeader` réécrit aussi les métadonnées source ; ses appelants qui ne
  veulent pas les perdre les recopient (transcription, complément à l'import).
