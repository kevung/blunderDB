# Banc V0.5 — histogramme de t_lisible − roll_tick_ms

Verdict : **outils livrés et testés sur fixture ; annotation à la main non faite**. Aucun chiffre
réel n'est donné : la médiane, p10, p90 et la part hors de [−1 ; +2] s restent à produire (section
« Résultat » vide, à remplir après saisie). Issue #847 ouverte jusqu'à cette saisie.

Question : de combien le Repère du jet tapé (`roll_tick_ms`, posé après repos, lecture et frappe)
est-il en retard sur l'instant où les dés sont lisibles à l'image ? Ce décalage fixe la fenêtre de
l'ADR-0109 règle 5 et dit si les Repères valent comme fenêtres d'apprentissage faible.

## Conventions (chacune tenue par un test qu'une mutation fait échouer)

- **Signe** : `delta = t_lisible − roll_tick_ms`. Négatif : dés lisibles *avant* le Repère (attendu
  si le Repère suit la lecture) ; positif : après.
- **Unité** : `t_lisible_ms` et `roll_tick_ms` en millisecondes ; l'affichage est en secondes. La
  saisie accepte aussi l'horloge du lecteur `m:ss.mmm`.
- **Quantiles** : interpolation linéaire entre rangs voisins, rang `(n−1)·p/100`, sur valeurs triées.
- **Hors fenêtre** : strictement hors de [−1000 ; +2000] ms (bornes incluses dans la fenêtre).
- **Classes** de l'histogramme : 500 ms, plancher. Coup à cellule vide : ignoré.
- **Sélection** des coups : `n` coups étalés régulièrement (premier et dernier inclus), sans tirage.
- `roll_tick_ms` est un temps de lecture de la vidéo depuis le début du média : `roll_mmss` se lit directement dans le lecteur. L'extrait facultatif commence 5 s avant, dure 9 s (réglables).

Outils : `tools/vision/{marks_export,prepare_annot,readable_hist}.py`, tests `test_vision.py` et
`test_annot.py`, mutations `python3 tools/vision/mutations.py` (49 mutants, 49 tués).

## Procédure d'annotation (Kevin)

Deux vidéos, 25 coups chacune (ou 50 sur une), dont les Repères existent en base.

1. `python3 tools/vision/marks_export.py BASE.db MATCH marks.csv`
2. `python3 tools/vision/prepare_annot.py marks.csv annot.csv --n 50 --video V.mkv --clips clips > clips.sh`
   puis `sh clips.sh` (facultatif : on peut lire dans la vidéo entière à `roll_mmss`).
3. Pour chaque ligne : aller à `roll_mmss − 5 s`, avancer image par image jusqu'à la **première
   image où les deux dés sont lisibles** (valeurs identifiables à l'œil), noter l'instant du
   lecteur dans `t_lisible_ms` (`m:ss.mmm` ou ms). Dans un extrait, ajouter le début de l'extrait
   (`roll_mmss − 5 s`). Coup où les dés ne sont jamais lisibles : laisser vide.
4. `python3 tools/vision/readable_hist.py marks.csv annot.csv histo.png`, recopier ici la sortie
   (nombre de coups, médiane, p10, p90, part hors fenêtre) et conclure sur la borne de 60 s.

## Corpus : ce qui manque

- `~/src/lazyBG/rawvid/*.mkv` : vidéos sans Repères (pas de base blunderDB associée).
- `/home/unger/src/blunderDB/a.db` : match 1 (Kévin Unger – James MacNaughtan) a 242 Repères
  `roll_tick_ms`, mais la vidéo correspondante n'est pas sur le disque ; les autres matchs de
  cette base n'ont aucun Repère. `~/a.db` n'en a aucun.
- Il faut donc soit la vidéo du match 1, soit des Repères posés sur un match de `rawvid`
  (importer le match, taper les jets en regardant la vidéo).

## Condition sur les Repères du corpus

Les Repères doivent avoir été posés avec l'assistant de vision **éteint**. Depuis l'ADR-0109
règle 5 (`TranscriptionPanel.svelte`), un Repère posé pendant un repos détecté sûr prend le
`mediaTime` de ce repos : delta tendrait vers 0 et la mesure ne vaudrait rien.

## Résultat

À remplir après annotation.
