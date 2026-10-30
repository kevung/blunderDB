# tools/vision

Bancs du chantier de transcription par la vidéo. Python 3, bibliothèque
standard ; matplotlib est facultatif (repli en histogramme texte). Hors de
`go build` et `npm run build`.

## Banc V0.5 : t_lisible − roll_tick_ms

1. `python3 marks_export.py base.db <match_id> marks.csv` (base ouverte en
   lecture seule) : colonnes `n, roll_tick_ms, tick_ms, dés`.
2. `cp annot_template.csv annot.csv`, puis pour chaque coup `n` de `marks.csv`
   (50 lignes ; ajouter ou retirer des lignes au besoin) : dans le lecteur,
   placer la vidéo sur la première image où les dés sont lisibles et noter
   l'instant en millisecondes dans `t_lisible_ms`.
   Lire l'instant : le temps du lecteur (mm:ss.ms) ; ms = (60·mm + ss)·1000 + ms.
   Laisser vide un coup illisible.
3. `python3 readable_hist.py marks.csv annot.csv [out.png]` : médiane, p10, p90,
   part hors de [−1 ; +2] s, histogramme.

Test : `python3 -m unittest discover -s tools/vision` depuis la racine.
