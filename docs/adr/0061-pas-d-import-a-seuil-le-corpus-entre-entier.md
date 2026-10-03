# ADR-0061 — Pas d'import à seuil : un corpus entre entier

Statut : acceptée.
Voir aussi : ADR-0001, ADR-0028 ; `tasks/plan-grosses-bases-2026-10/README.md` (GB0.3),
`tasks/plan-grosses-bases-2026-10/MESURES.md`.

## Contexte

Le plan « grosses bases » estimait un corpus BMAB de 21 Go à 40 à 70 M de positions uniques et
une base de 30 à 55 Go. Ne garder à l'import que les décisions fautives (au-dessus d'un seuil,
de videau seulement, d'un seul joueur) diviserait ce volume par 5 à 10. Trois options étaient
ouvertes avant le pipeline d'import (GB2.1), où un filtre s'insérerait :

- A : pas d'import à seuil ; le plan vise le volume complet ;
- B : import à seuil en option explicite, les stats refusant ou signalant une base partielle ;
- C : tout importer, puis une commande de purge des positions sans erreur, marque, commentaire
  ni collection.

Les mesures de GB0.2 (`MESURES.md`) ont déplacé le problème : le corpus est livré en cinq
copies identiques (chaque match figure dans les cinq dossiers régionaux, même numéro, même
contenu). 166 713 fichiers ne font que 33 370 matchs distincts, 4,9 Go et non 21. Le volume
attendu tombe à ≈ 16,6 M de positions brutes, ≈ 16 M uniques, ≈ 13 Go de base au codec
d'avant le lot GB1.

## Décision

**Option A.** Un import écrit toutes les positions d'un match ; aucun filtre d'import, aucune
purge des positions d'un match.

1. **B est rejetée** parce qu'elle fausse ce que la base affirme. La PR et les stats de match
   comptent toutes les décisions ; une base filtrée rend des nombres faux sans qu'aucune requête
   ne puisse le savoir, sauf à faire porter un drapeau « partielle » à chaque calcul. Et une
   position est identifiée par son hash (ADR-0001, ADR-0028) pour être retrouvée quelle que soit
   la porte par laquelle elle est entrée : décider à l'import, sur un critère de provenance
   (le joueur, l'erreur de ce jour-là), qu'elle n'existe pas, c'est faire de la provenance une
   condition d'existence — ce que ces deux ADR refusent pour `individually_imported`,
   `has_jacoby` et `has_beaver`. Une position écartée ne revient que par un réimport.
2. **C est rejetée** parce qu'elle contredit le prédicat de rétention : une position est tenue
   par le coup d'un match qui la référence (`positionIsHeldSQL`, écrit trois fois). Purger les
   positions sans erreur d'un match revient à supprimer des lignes `move`, donc à mutiler le
   match : sa relecture, son export et ses stats tombent. La purge qui existe — celle des
   positions orphelines quand un match est supprimé — reste la seule.
3. **Le volume se traite par le format, pas par l'oubli** : déduplication des fichiers à
   l'import (empreinte SHA-256, lot 4), codec (lot 1), pipeline (lot 2), GUI paginée (lot 3).
   Les ordres de grandeur mesurés tiennent dans SQLite sans filtrer.

## Conséquences

- Les estimations du README du plan (§ 2) sont corrigées par `MESURES.md` ; les objectifs de
  débit et de taille portent sur ≈ 33 000 matchs, ≈ 16,6 M de positions brutes.
- Qui ne veut qu'une partie d'un corpus l'obtient par la recherche et les collections, ou par
  un export filtré, jamais par une base dont les stats mentent.
- À rouvrir seulement si un corpus réel, une fois dédupliqué, dépasse ce que le lot GB1 et le
  lot GB2 font tenir sur un poste ordinaire ; la question serait alors celle d'un format, pas
  d'un filtre.
