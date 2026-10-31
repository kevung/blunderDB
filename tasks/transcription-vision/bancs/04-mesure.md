# Banc V1.7 — F1 à ±0,5 s contre les Repères, gain de navigation

Verdict : **outil livré et testé sur fixture ; mesure réelle non faite** (corpus manquant, voir
plus bas). Aucun chiffre réel n'est donc donné, et les seuils de `THRESHOLDS`
(`frontend/src/vision/automata.js`) restent ceux réglés sur les journaux synthétiques.

## L'outil

`tools/vision/mesure.py` (Python standard) ; tests `tools/vision/test_mesure.py` ;
`tools/vision/mutations.py` altère chaque convention et vérifie que la suite échoue.

```bash
python3 tools/vision/marks_export.py BASE.db MATCH marks.csv        # Repères (lecture seule)
python3 tools/vision/mesure.py events.json marks.csv [--tol 500] \
        [--paire dice.rest=roll_tick_ms] [--paire dice.pickup=tick_ms] [--etats sure,check]
python3 -m unittest discover -s tools/vision && python3 tools/vision/mutations.py
```

`events.json` est `replay(records)` (`frontend/src/vision/descriptors.js`) appliqué à un journal
de descripteurs réel et sérialisé : `{type, mediaTime en s, state}`.

Conventions, chacune tenue par un test qu'une mutation fait échouer (25 mutants, 25 tués) :

- un Repère est une **fenêtre** (`02-couverture-sql.md` § 4) : celle du jet va du `tick_ms` de
  l'action précédente de la partie (colonne `prev_tick_ms` de `marks_export.py`) à
  `roll_tick_ms` ; celle de l'action va de `roll_tick_ms` à `tick_ms` ; il n'y a plus d'avance
  réglable ;
- un événement du type voulu est un vrai positif s'il tombe dans la fenêtre élargie de
  `--tol` ms (500) de chaque côté ; bornes et tolérance incluses ; chacune des deux bornes a sa
  mutation ;
- une fenêtre sans borne de début (premier jet d'une partie, action précédente sans Repère) ou
  à rebours n'est pas évaluable : elle sort du rappel, est comptée à part (`sans_fenetre`) et la
  précision est alors un minorant ;
- appariement un à un, glouton par distance croissante à la fenêtre ; reste = FP / FN ;
- paire type = colonne choisie ; défaut `dice.rest = roll_tick_ms` ; filtre d'états optionnel ;
- gain de navigation, **modèle** : les Repères de toutes les paires forment une seule suite lue
  dans l'ordre du temps. Sans marqueur, la traversée depuis le Repère précédent ; avec
  marqueurs, un saut (2 s par défaut, à mesurer) plus la distance de l'événement à sa fenêtre,
  ou la traversée si la fenêtre n'a pas d'événement, et **un saut de plus par faux positif**.
  C'est un modèle de coût, pas une mesure d'utilisateur : le chiffre de l'ADR demande un essai
  chronométré.

Fixture (`tools/vision/fixtures/`) : 9 événements, 4 coups, à la main. Attendu pour
`dice.rest = roll_tick_ms` : TP 3, FP 4, FN 0, une fenêtre non évaluable, F1 0,600, gain du modèle
39,0 %. Ces chiffres valident l'outil, ils ne disent rien du détecteur.

## Ce qui manque pour la mesure réelle

Trouvé sur le disque (2026-10-31) :

- 5 vidéos `~/src/lazyBG/rawvid/*.mkv` avec transcriptions `.mat` : aucun Repère (les `.mat` n'ont
  pas d'instants), et aucune base blunderDB ne les rattache ;
- `~/src/blunderDB/a.db` : 242 Repères (match 1, Unger – MacNaughtan, 9 pts), mais pas la vidéo ;
- `~/dwn/unger.db` (matchs 284-286, banc V0.2) : absente du disque.

Il faut **les trois ensemble** pour un même match : la vidéo, ses Repères en base, et le journal
de descripteurs que le volet produit en lisant cette vidéo calibrée (WebKitGTK ; non rejouable
hors du volet). Marche à suivre : ouvrir le match dans le volet, calibrer, lire en entier,
exporter le journal, `replay` → `events.json`, puis `marks_export.py` et `mesure.py`. Les seuils se
règlent ensuite en rejouant le journal sous `createAutomata(overrides)`.
