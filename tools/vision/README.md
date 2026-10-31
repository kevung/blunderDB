# tools/vision

Bancs du chantier de transcription par la vidéo. Python 3, bibliothèque
standard ; matplotlib est facultatif (repli en histogramme texte). Hors de
`go build` et `npm run build`.

## Banc V0.5 : t_lisible − roll_tick_ms

1. `python3 marks_export.py base.db <match_id> marks.csv` (base ouverte en
   lecture seule) : colonnes `n, roll_tick_ms, tick_ms, dés`.
2. `python3 prepare_annot.py marks.csv annot.csv [--n 50] [--offset-ms D] [--video v.mkv --clips clips]` :
   choisit 50 coups étalés sur le match (déterministe) et écrit `annot.csv` avec l'instant du
   Repère dans l'horloge du lecteur (`roll_mmss`) ; avec `--video`, imprime une commande ffmpeg
   par coup (extrait de 9 s commençant 5 s avant le Repère ; à lancer avec `sh`). Puis, pour
   chaque ligne : placer la vidéo sur la première image où les dés sont lisibles et noter
   l'instant dans `t_lisible_ms` (ms entières, ou l'horloge `m:ss.mmm` du lecteur ; si l'on
   lit dans un extrait, ajouter son début). Laisser vide un coup illisible.
   (`annot_template.csv` reste le gabarit nu, sans `prepare_annot.py`.)
3. `python3 readable_hist.py marks.csv annot.csv [out.png]` : médiane, p10, p90,
   part hors de [−1 ; +2] s, histogramme.

Test : `python3 -m unittest discover -s tools/vision` depuis la racine.

## Banc V1.7 : F1 des événements et gain de navigation

`python3 mesure.py events.json marks.csv` (voir l'en-tête du script et
`tasks/transcription-vision/bancs/04-mesure.md`). `python3 mutations.py` (aussi pour `readable_hist.py` et `prepare_annot.py`) vérifie que chaque
convention est tenue par un test. Fixture sous `fixtures/`.
