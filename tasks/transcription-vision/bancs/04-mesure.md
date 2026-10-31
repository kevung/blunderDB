# Banc V1.7 — F1 à ±0,5 s contre les Repères, gain de navigation

Verdict : **outil livré et testé sur fixture ; mesure réelle non faite** (corpus manquant, voir
plus bas). Aucun chiffre réel n'est donc donné, et les seuils de `THRESHOLDS`
(`frontend/src/vision/automata.js`) restent ceux réglés sur les journaux synthétiques.

## L'outil

`tools/vision/mesure.py` (Python standard) ; tests `tools/vision/test_mesure.py` ;
`tools/vision/mutations.py` altère chaque convention et vérifie que la suite échoue.

```bash
python3 tools/vision/marks_export.py BASE.db MATCH marks.csv        # Repères (lecture seule)
python3 tools/vision/mesure.py events.json marks.csv [--tol 500] [--avance MS] \
        [--paire dice.rest=roll_tick_ms] [--paire dice.pickup=tick_ms] [--etats sure,check]
python3 -m unittest discover -s tools/vision && python3 tools/vision/mutations.py
```

`events.json` est `replay(records)` (`frontend/src/vision/descriptors.js`) appliqué à un journal
de descripteurs réel et sérialisé : `{type, mediaTime en s, state}`.

Conventions, chacune tenue par un test qu'une mutation fait échouer (17 mutants, 17 tués) :

- tolérance en ms entiers, bornes incluses (500 apparie, 501 non) ;
- Repère = point (Repère − `--avance`) ; l'ADR-0109 dit que le repos précède le Repère du jet,
  l'avance par défaut est 0 et se règle sur l'histogramme `t_lisible − roll_tick_ms` ;
- appariement un à un, glouton par écart croissant (pas par ordre d'arrivée) ; reste = FP / FN ;
- paire type = colonne choisie ; défaut `dice.rest = roll_tick_ms` ; filtre d'états optionnel ;
- gain de navigation, **modèle** : lecture séquentielle des Repères ; sans marqueur la traversée
  r_k − r_(k−1) ; avec marqueurs, un saut (2 s par défaut, à mesurer) plus l'écart résiduel, ou la
  traversée complète si aucun événement n'est apparié. C'est un modèle de coût, pas une mesure
  d'utilisateur : le chiffre de l'ADR demande un essai chronométré.

Fixture (`tools/vision/fixtures/`) : 8 événements, 4 Repères de jet, à la main. Attendu : TP 3,
FP 3, FN 1, F1 0,600, écart médian 200 ms, gain du modèle 58,2 %. Ces chiffres valident l'outil,
ils ne disent rien du détecteur.

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
