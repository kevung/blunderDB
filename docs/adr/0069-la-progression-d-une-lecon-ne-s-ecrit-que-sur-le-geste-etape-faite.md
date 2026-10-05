# ADR-0069 — La progression d'une Leçon ne s'écrit que sur le geste « étape faite »

Statut : acceptée.
Voir aussi : ADR-0007 (le receveur n'écrit rien, liste blanche `issuance.Carried`), ADR-0065
(le coach lit à travers les tenants et n'écrit que chez lui), ADR-0066 règle 5 (la Leçon) ;
fiche I.21.

## Contexte

La fiche I.21 demande des « parcours pédagogiques » : une suite ordonnée d'étapes, chacune un
texte puis une Collection ou une Position, remise dans un `.dbx` filigrané. C'est la Leçon
d'ADR-0066 ; « parcours » reste un mot de l'interface et du
vocabulaire courant, pas un second objet du domaine. Reste la progression :
la lecture n'enregistre ni l'étape atteinte ni l'ouverture (ADR-0066 règle 5, ADR-0007), et
l'élève qui reprend une Leçon de trente étapes une semaine plus tard doit savoir où il en est.

## Décision

1. **La progression est l'ensemble des Étapes que l'élève a marquées faites**, avec la date
   du geste, dans la base de l'élève (`lesson_progress`, une ligne par Étape faite).
2. **Seul le geste explicite « étape faite » l'écrit**, et son retrait l'efface
   (`LessonStore.SetStepDone`). Lire une Leçon, l'ouvrir, passer d'une Étape à l'autre,
   l'importer ou l'exporter n'écrivent rien : la règle 5 d'ADR-0066 tient pour tout ce qui
   n'est pas ce geste. ADR-0007 n'est pas entamée : le geste est un acte de l'élève dans sa
   propre base, comme un commentaire, pas une trace de réception. Marquer deux fois garde
   la première date.
3. **La progression ne voyage jamais.** Elle vit dans une table qu'aucun exporteur ne lit
   (l'export copie `lesson` et `lesson_step`) et n'est pas une métadonnée portée
   (`issuance.Carried`) : un `.dbx` remis à quelqu'un d'autre ne dit pas où en est son
   auteur. Un export complet ne l'emporte pas non plus ; la sauvegarde d'une base est la
   copie de son fichier.
4. **Elle suit les Étapes, pas leur ordre.** Supprimer une Étape supprime sa progression ;
   réordonner les Étapes, retoucher leur texte ou réimporter la même Leçon (laissée telle
   quelle par `MergeLessons`) la garde.
5. **Une surface, trois modes**, comme la Leçon : `SetStepDone` et `DoneSteps` sur
   `storage.LessonStore`, exposés au bureau, à la CLI et au démon par les mêmes méthodes.

## Conséquences

- Schéma 2.31.0 : table `lesson_progress` ; côté PostgreSQL, clé `(tenant_id, id)` sur
  `lesson_step` pour la clé étrangère composite, migration `033_met_progress_move_error.sql`.
- Le glossaire (`CONTEXT.md`, *Step done*) dit que la Leçon est suivie par le seul geste de
  l'élève.
- Le geste est le bouton « étape faite » de la barre de lecture (`LessonBar.svelte`), la
  commande `lesson done` (avec `lesson progress` pour la lire) et la route
  `/v1/lessons.setStepDone`.
