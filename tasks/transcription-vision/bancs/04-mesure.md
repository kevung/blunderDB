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

Il faut **les trois ensemble** pour un même match : la vidéo, ses Repères en base, et les
événements que le volet produit en lisant cette vidéo calibrée (WebKitGTK ; non rejouable hors
du volet).

Marche à suivre (build de développement seulement : l'action n'existe pas dans le binaire publié) :

1. `make dev`, ouvrir le match et son brouillon de transcription avec la vidéo attachée ;
2. calibrer, assistant allumé ;
3. lire la vidéo en entier à 1×, sans saut ;
4. cliquer « Enregistrer les événements (banc) » (sous la vidéo) et choisir `events.json` ;
   le fichier porte `{events, calibration, source}`, `mediaTime` en secondes ;
5. `marks_export.py` puis `mesure.py events.json marks.csv`.

Les événements ne viennent que de la session en cours : recharger la page les perd (le journal
IndexedDB n'est pas relu ici). Les seuils se règlent ensuite en rejouant un journal sous
`createAutomata(overrides)`.

## Première mesure réelle (2026-11-01)

Corpus : `~/bkgm/unger.db` match 285 (Unger – Sekri, 9 pts), vidéo locale `.webm` 640×480,
24 img/s, caméra fixe au-dessus du plateau ; 383 Repères de jet posés à la main avant l'ADR-0109
(vision éteinte, vitesse ×2 environ, voir `05-histogramme.md`). Lecture à 1× dans `make dev` sur
une copie de la base ; une calibration, puis une deuxième à 0 h 53 (contournement de #868). Les
événements cessent à 55 min 12 s (après un `view.lost`, l'analyse ne repart plus malgré la
recalibration) : 1168 événements sur les 55 premières minutes d'un match de 97.

```
type             état     nombre
hands.in         sûr       335      (un toutes les 5 s)
hands.out        sûr       185
dice.thrown      sûr        50
dice.rest        sûr        53
dice.pickup      à vér.     12
frame.uncertain            406      (un toutes les 4 s)
view.lost / view.found   40 / 38
overlay.changed  à vér.     18
```

`mesure.py`, `dice.rest = roll_tick_ms`, ±0,5 s : TP 8, FP 47, FN 287, F1 0,046 ; gain de
navigation du modèle 2,4 %.

Restreint aux tranches où l'analyse tournait (hors `view.lost` → `view.found`) : 1594 s couvertes
sur 3313 (48 %), 75 Repères de jet. Un `dice.rest` sûr tombe dans [Repère − 15 s ; Repère + 1 s]
pour 20 d'entre eux (27 %), écart médian 8 s ; au hasard, avec 53 repos en 1594 s, la chance d'en
avoir un dans 16 s est de 41 % : **le détecteur de dés ne voit pas les jets**. Les mains sont vues
presque partout (73/75), mais leur densité rend ce chiffre non informatif (84 % au hasard).

Verdict : **V1 ne tient pas sur cette vidéo réelle.** Trois causes, par ordre de poids supposé :

1. **Présence** (#868) : la référence est prise pions compris, la vue se « perd » quand la position
   s'éloigne de celle de la calibration ; 52 % du temps sans analyse, et un `view.lost` final dont
   l'analyse ne revient pas même après recalibration.
2. **Témoin hors plateau** : 406 `frame.uncertain` en 55 min ; chacun réinitialise les fonds des
   ROI, ce qui empêche l'automate des dés d'atteindre ses durées minimales. Le témoin tombe
   vraisemblablement là où passent les bras ou le corps des joueurs.
3. **Seuils des automates** réglés sur des journaux synthétiques (`THRESHOLDS` d'`automata.js`).

Suite : le journal de descripteurs de cette lecture est dans l'IndexedDB de la webview de
`make dev` (sessions `<draftId>:Calibration` et `<draftId>:Calibration 2`). Le rendre lisible
hors du volet (export dev-only, `readJournal`) permettrait de régler témoin et seuils par `replay`
avec les Repères comme fenêtres, sans relire la vidéo ; puis corriger #868 et relancer la mesure.
