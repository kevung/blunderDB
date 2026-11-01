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

### Rejouer le journal sans relire la vidéo

Le journal des descripteurs de la lecture reste dans l'IndexedDB de la webview (base
`blunderdb-vision-journal`, sessions `<brouillon>:<calibration>`) ; rien ne l'efface dans
l'application (`clearJournal` n'a pas d'appelant d'interface), mais l'IndexedDB est liée à
l'origine de `wails dev` : ne pas en changer le port.

1. Sous `make dev`, volet transcription du même brouillon : « Enregistrer le journal (banc) »
   écrit `journal.json` (`{ draftId, sessions: { "<brouillon>:<calibration>": [relevés…] },
   calibrations: { "<session>": { name, clicks, modelToImage, overlay? } } }`, toutes les
   sessions du brouillon).
2. `node tools/vision/replay.mjs journal.json [--session S] [--overrides '{"seuil":…}'] > events.json`
   (mêmes événements que l'export du volet ; les seuils : `createAutomata(overrides)`).
3. `python3 tools/vision/mesure.py events.json marks.csv`.


## Réglage par rejeu (2026-11-01)

Journal de la lecture ci-dessus rejoué hors de l'application (`~/bkgm/bancs/v17-match285/`,
données inchangées). Le rejeu au défaut reproduit le live à l'unité.

**Protocole.** Seuls comptent les Repères dont `roll_tick_ms` tombe dans le temps couvert par le
journal (`mesure.py --plages journal.json` : 1675 s, 83 Repères ; ailleurs l'analyse ne tournait
pas, #868). 43 de ces 83 Repères n'ont pas d'action précédente horodatée : `--avance 7` leur donne
la fenêtre [Repère − 7 s ; Repère] de `05-histogramme.md` au lieu de les sortir du rappel. Le temps
couvert est coupé en deux moitiés égales à 1136,7 s : A (41 Repères) pour régler, B (42) pour
valider, puis l'inverse pour contrôle. Sous `--plages`, le gain de navigation ne compte que le
temps couvert.

**Le hasard est haut.** Les fenêtres couvrent 44 % du temps couvert : 50 instants tirés au
hasard donnent F1 0,35 sur A et 0,28 sur B, gain de navigation 0,27–0,28. Tout chiffre ci-dessous
se lit contre cette ligne.

### Diagnostic

- **La bande des dés est vide la moitié du temps.** Les dés sont l'excès entre `E_hi` et
  `handsIn`. Sur cette vidéo (webm 640×480, compressée), le plancher de bruit d'une ROI va de
  0,001 à 0,09 (déciles) et `E_hi` a pour médiane 0,029 : il dépasse `handsIn` = 0,03 sur 49 %
  des relevés, et pendant la majorité du temps dans 61 des 83 fenêtres de Repère. Les mains
  font un excès de 0,1 à 0,2 ; un jet propre (Repère à 1133 s, plancher 0,002) un excès de 0,01
  à 0,03 — et là l'automate marche (lancer et repos sûrs 1,5 s avant le Repère).
- **Le témoin suit le plateau.** `w` a pour médiane 0,00 quand la plus grande distance census des
  ROI est sous 0,005, 0,44 quand elle dépasse 0,1, et dépasse 60 % sur 17 % des images à mains
  franches : `u = 1` dans 62 des 83 fenêtres. Chaque image incertaine remet les automates et le
  plancher à zéro au milieu d'une main ; le plancher réappris sur les images de mouvement qui
  suivent porte `E_hi` à 0,1–0,2 pendant plusieurs secondes. Exposition automatique, débit du
  codeur ou bras au-dessus du témoin (48×24 px en haut à droite, hors plateau) : le relevé ne
  permet pas de trancher.
- **Mais le témoin sert de béquille.** Le plancher ne s'apprend que sur des images calmes ; sans
  les remises du témoin, il reste celui d'une scène plus calme, et le repos (5 images sous
  `E_lo`) n'est plus atteint : avec `handsIn` 0,15 et le témoin coupé, F1 0,24 (A) / 0,33 (B).
- **Durées minimales, `maxRoll`, `throwGap`** : peu ou pas d'effet à elles seules ; `restFrames`
  3 aide sur A, pas sur B.
- **Les dés atterrissent-ils dans les ROI ?** Les ROI sont les deux demi-champs entiers ; les dés
  y tombent (image de calibration), mais deux dés font environ 0,4 % des bits d'une moitié : leur
  signal est de l'ordre du bruit de compression dès que celui-ci monte.

Le témoin est rejouable : `w` est au relevé, et `u = w > seuil` rejoue fidèlement les automates
de mains et de dés (la distance à l'image précédente ne dépend pas des fonds) ; seule la distance
au fond, donc l'automate de partie, garde les remises faites à 60 % pendant la lecture.

### Résultats (dice.rest = roll_tick_ms, ±0,5 s, TP/FP/FN)

| réglage | A : F1 (P, R) | B : F1 (P, R) | gain A / B |
|---|---|---|---|
| hasard, ~50 instants | 0,35 | 0,28 | 0,27 / 0,28 |
| défaut | 0,197 (0,23, 0,17) 7/23/34 | 0,149 (0,20, 0,12) 5/20/37 | 0,07 / 0,15 |
| **`handsIn` 0,15, `handsOut` 0,10** | 0,547 (0,48, 0,63) 26/28/15 | **0,374** (0,35, 0,40) 17/32/25 | 0,44 / 0,39 |
| recherche à 13 paramètres, choisie sur A | 0,766 | 0,351 | 0,69 / 0,32 |
| même recherche, choisie sur B (contrôle) | 0,653 | 0,525 | 0,63 / 0,52 |

La recherche (1600 réglages tirés : seuils, facteurs, planchers, durées, fenêtre du bruit, témoin
à 0,6 / 0,8 / 0,9 / désactivé) surapprend : choisie sur A, elle fait moins bien sur B que le seul
`handsIn`, et ses meilleurs réglages sur A exploitent le témoin à 60 %, qui n'aide pas sur B.
`loFactor` 3 améliore les deux moitiés, mais rend `E_lo` égal à `E_hi` : l'hystérésis de
l'ADR-0109 disparaît, il n'est pas retenu.

Avec `handsIn` 0,15, 44 des 103 repos détectés tombent 0 à 6 s avant un Repère, le délai de lecture
mesuré par `05-histogramme.md` ; au défaut, la distribution est plate (8 sur 55).

### Verdict

`handsIn` 0,15 / `handsOut` 0,10 est le seul réglage que le diagnostic justifie (rendre non vide
la bande des dés sous un bruit de compression réel) et qui tient en validation. Il fait passer le
détecteur de **pire que le hasard** à un peu mieux : F1 de validation 0,15 → 0,37, gain de
navigation 0,15 → 0,39, contre 0,28 au hasard.

**Il n'est pas reporté dans `THRESHOLDS`.** Les scènes synthétiques qui tiennent les seuils
(`vision.automata.replay.test.js`) dessinent une main de 44×36 px à texture lisse, dont l'excès
census plafonne vers 0,1 : sous `handsIn` 0,15, elle passe pour des dés et 13 gardes tombent.
L'agrandir ne suffit pas (64×52 px : 0,15 au plus), c'est la texture qui manque. Adopter le
réglage demande donc de refaire la main des scènes sur le modèle d'une vraie, et cela vaut d'être
fait sur un deuxième journal réel, après #868, plutôt que sur 83 Repères d'une seule vidéo. D'ici
là, le réglage se passe en `--overrides` au rejeu.

**Cela ne suffirait pas** : F1 de validation sous 0,5, précision 0,35. La limite tient à la
méthode, pas aux seuils. Une distance census sur un demi-champ entier ne mesure qu'une quantité de
changement : elle ne distingue pas deux dés qui roulent d'une main qui frôle le plateau, d'un pion
qu'on déplace ou d'un regain de bruit du codeur, et un repos d'un instant où rien ne bouge par
hasard. Le plancher de bruit, appris sur les seules images calmes, ne suit pas une vidéo dont le
bruit varie avec le mouvement, et le témoin qui devait signaler les sauts d'image réagit à la
scène. Il faudrait voir les dés eux-mêmes (une ROI qui les suit, un détecteur de dés appris) pour
dépasser ce plafond.

Reproduire :

```
node tools/vision/replay.mjs journal.json --overrides '{"handsIn":0.15,"handsOut":0.1}' > ev.json
python3 tools/vision/mesure.py ev.json marks.csv --plages journal.json --avance 7
```
