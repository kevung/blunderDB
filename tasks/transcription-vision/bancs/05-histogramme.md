# Banc V0.5 — histogramme de t_lisible − roll_tick_ms

Verdict : **mesuré sur 96 coups de deux vidéos** (2026-11-01) : les dés sont lisibles 2,5 s
(médiane) avant le Repère, presque jamais après ; la fenêtre [Repère − 7 s ; Repère + 1 s] en
contient 95 %.

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

Corpus : `~/bkgm/unger.db` match 285 (Unger – Sekri, 9 pts, BMAB 2026-10-10, ronde 2), vidéo
locale `.webm` 640×480 à 24 img/s, 383 Repères posés à la main pendant la transcription, avant
l'ADR-0109 (vision éteinte). 50 coups étalés par `prepare_annot.py`, saisis dans mpv au ralenti
×0,5 avec `annoter.lua` (Entrée à la première image où les deux dés sont lisibles ; temps de
réaction ≈ 0,1 s de média, qui décale légèrement vers le tard). Une seule vidéo, un seul
annotateur : le chiffre vaut pour ce poste et cette façon de transcrire.

```
coups annotés : 47 (3 illisibles)
médiane : -2.99 s
p10     : -5.55 s
p90     : -1.37 s
hors [-1 ; +2] s : 94 %
extrêmes : -10.9 s et -0.02 s ; aucun delta positif
```

| Fenêtre autour du Repère | Part des coups |
|---|---|
| [−1 ; +2] s | 6 % |
| [−6 ; −1] s | 85 % |
| [−6,5 ; −0,5] s | 94 % |
| [−7 ; 0] s | 98 % |

Conclusions :

- **Le Repère n'est pas l'instant du jet** : il le suit de 1,4 à 5,5 s (p10–p90), le temps de
  lire et de taper. Il ne peut donc pas servir de centre à la fenêtre [−1 ; +2] s de la règle 1
  (6 % dedans).
- **Comme fenêtre d'apprentissage faible (ADR-0110), le Repère vaut** à condition d'être décalé :
  [Repère − 6,5 s ; Repère − 0,5 s] couvre 94 % des instants lisibles, [Repère − 7 s ; Repère]
  98 %. Les images calmes du corpus se cherchent dans cette fenêtre, pas autour du Repère.
- **Borne de 60 s de la règle 5** : le repos détecté précède la frappe de 11 s au pire ; 60 s
  est très au-delà du besoin. (Corrigé plus bas : ce chiffre dépend de la vitesse de lecture.) Elle pourrait descendre, ce qui
  réduit le risque de caler sur un repos ancien ; à décider avec la mesure V1.7 (#858), qui dit
  combien de repos sont détectés sûrs.
- La fenêtre du jet de `mesure.py` (de l'action précédente au Repère, ±0,5 s) contient bien
  l'instant lisible : aucun delta n'est positif.

### Deuxième vidéo et réunion

Corpus : brouillon de transcription 5 de `~/bkgm/unger.db` (Unger – Guerbaa, 9 pts, même tournoi,
ronde 4, non terminé), vidéo locale `.mp4` en portrait 480×640 à 24 img/s, 271 Repères posés les
12 et 13 octobre (vision éteinte), lus dans le document du brouillon (aucun `move` en base). Même
annotateur, même outil. Même tournoi, donc même matériel et même salle : la deuxième vidéo
éprouve la stabilité du délai chez un transcripteur, pas la diversité des sources.

```
                 coups  médiane   p10     p90     extrêmes          delta > 0
vidéo 1 (285)      47   -2.99 s  -5.55 s -1.37 s  -10.9 / -0.02 s       0
vidéo 2 (brouil.)  49   -1.96 s  -4.49 s -0.94 s  -12.0 / +1.89 s       2
réunion            96   -2.48 s  -5.60 s -1.09 s  -12.0 / +1.89 s       2
```

| Fenêtre autour du Repère | Vidéo 2 | Réunion |
|---|---|---|
| [−6 ; −1] s | 80 % | 82 % |
| [−6,5 ; −0,5] s | 88 % | 91 % |
| [−7 ; 0] s | 90 % | 94 % |
| [−7 ; +1] s | — | 95 % |
| [−8 ; +2] s | — | 97 % |

Ce que la deuxième vidéo change :

- Le délai tient en ordre de grandeur (médiane 2 s contre 3 s) ; la seconde frappe est un peu plus
  rapide. Deux coups ont un Repère posé *avant* que les dés soient lisibles (+0,04 s et +1,89 s) :
  la frappe a devancé l'image, ou les dés ont roulé encore. La fenêtre d'apprentissage faible doit
  donc déborder un peu après le Repère : **[Repère − 7 s ; Repère + 1 s]** (95 % des 96 coups)
  remplace [− 6,5 ; − 0,5] s.
- La borne de 60 s de la règle 5 reste très au-delà du besoin (pire cas −12 s) à ×2 ; voir la section sur la vitesse de lecture avant de la réduire.
- Une vraie deuxième source (autre salle, autre matériel) reste à mesurer avant d'en faire une
  constante : Marseille, Jussieu ou les Tuileries, une fois transcrits avec Repères.

### Le délai est en secondes de média : il dépend de la vitesse de lecture

Les deux vidéos ont été transcrites à ×2 environ, de mémoire de l'annotateur ; la vitesse n'est
enregistrée nulle part (ni en base ni en `localStorage`). Si la vidéo continue de tourner pendant
la lecture des dés et la frappe, le délai mesuré vaut temps de réaction × vitesse : 2,5 s de média
à ×2, c'est 1,25 s d'horloge ; le même transcripteur à ×1 donnerait ~1,25 s, à ×4 ~5 s. S'il met
en pause pour taper, le délai ne dépend plus de la vitesse. Les chiffres ci-dessus ne valent donc
que pour cette habitude (×2, sans pause) et ne sont pas des constantes :

- **Fenêtre d'apprentissage faible** : ne pas la figer à [−7 ; +1] s. L'estimer par transcription,
  à partir de la médiane de `roll_tick_ms − mediaTime` des repos de dés détectés sûrs et appariés
  (la mesure V1.7 donne ces paires), puis élargir autour ; [−7 ; +1] s reste le défaut quand la
  vision n'a rien détecté.
- **Borne de la règle 5** : elle s'applique pendant la saisie, quand la vitesse est connue ; la
  borner en temps d'horloge (≈ 8 s × vitesse, en média) plutôt qu'en temps de média fixe. 60 s
  couvre déjà ×4 (pire cas mesuré 6 s d'horloge → 24 s de média) ; une borne fixe à 15 s ne le
  couvrirait pas.

Habitude déclarée par le transcripteur : vidéo laissée à ×2 pendant la frappe, mise en pause
parfois sur un coup complexe ou dans une course de sortie rapide. Le délai mesuré mélange donc deux
régimes : le gros de la distribution suit le temps de réaction × 2, les pauses tirent vers 0 (et
peuvent expliquer les deux Repères à peine postérieurs à l'image lisible). C'est une raison de plus
pour estimer la fenêtre par transcription, et pour la garder large d'un côté et de l'autre.
