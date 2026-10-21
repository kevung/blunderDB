# Latence au démarrage — mesures avant/après (#688)

Question : le délai entre le lancement et une fenêtre utilisable. Réponse mesurée : le temps
part dans le WebView (parse et exécution d'un chunk JS de 990 kB, layout de la pile de polices,
locale appliquée après le premier rendu) et, base ouverte, dans les agrégats de badges des
listes. L'init Go et l'ouverture de la base sont négligeables.

## Comment mesurer

- Frontend : `npm run build`, puis `node scripts/measure-startup.mjs --dist <avant>,<après>`
  (Chromium sans tête derrière le mock Wails de l'e2e, écran d'accueil sans base ; les builds
  alternent, une machine chargée les ralentit également). Jalons `bdb:*` de
  `src/utils/startupMarks.js`, lisibles aussi dans l'inspecteur du WebView (Performance →
  Timings). La référence « avant » est `main` (bf38a6dc5) instrumenté des mêmes jalons.
- Go : `GODEBUG=inittrace=1` sur `cmd/serve` ; appels des bindings chronométrés sur une copie
  d'une base réelle de 66 Mo (286 matchs, 62 tournois, 98 000 analyses).

## Postes côté Go

| Poste | Durée |
|---|---:|
| `init` de tous les paquets (`engine` : tables Zobrist, MET…) | ~5 ms (engine 4,7 ms) |
| `OpenDatabase` de la base de 66 Mo, migration 2.42 → 2.43 comprise | 37 ms |
| `GetDatabaseStatsEstimate` (compteur de la barre d'état) | 0,14–0,19 s |
| `ListMatches` d'une page de 500 (286 matchs), badges compris | 0,61–0,67 s |
| `GetAllTournaments`, badges compris | 0,60–0,65 s |

## Écran d'accueil (sans base), médianes en ms

| Jalon | avant ×1 | après ×1 | avant ×4 CPU | après ×4 CPU |
|---|---:|---:|---:|---:|
| modules évalués | 158 | 202 | 247 | 271 |
| App montée | 158 | 301 | 247 | 476 |
| premier rendu (FCP) | 588 | 448 | 1092 | 864 |
| locale appliquée | 612 | 375 | 1289 | 614 |
| **prêt (accueil utilisable)** | **615** | **379** | **1299** | **624** |
| JS chargé avant « prêt » (kB) | 1662 | 1156 | 1662 | 1156 |

Chunk initial (`index-*.js`) : 991 kB → 135 kB ; le reste du JS avant « prêt » est la locale et
les dépendances partagées. « App montée » recule parce que la locale est désormais attendue
avant le montage (borné à 1 s) : la fenêtre ne peint plus chaque libellé deux fois.
×4 CPU approche le WebKitGTK d'un portable modeste.

## Base ouverte (base de 66 Mo)

| Poste | avant | après |
|---|---:|---:|
| `ListMatches` + `GetAllTournaments` au montage différé de la liste | 3 fois chacun | 1 fois |
| liste et tournois du panneau Matchs | en série, ~1,2 s | en parallèle, ~0,65 s |
| `GetDatabaseStatsEstimate` à l'ouverture | 2 appels | 1 appel |

## Ce qui a changé

- Panneaux et modales rarement ouverts en chunks différés (`lazyComponents.js`), préchargés
  après « prêt » : aucun ne s'ouvre vide au premier clic (`lazyComponents.test.js`).
- Pile de polices courte (`tokens.css`) : onze familles coûtaient ~250 ms de layout forcé
  (fontconfig) au premier rendu.
- La locale est appliquée avant le montage.
- `MatchPanel` charge sa liste une fois à son montage, liste et tournois côte à côte.
- Les rafraîchissements du compteur de bibliothèque d'un même tour partagent une lecture.

- En dev (où tourne l'e2e), tous les panneaux sont demandés dès le chargement de `main.js`,
  sans retenir le montage ; `dismissHomeScreen` attend le montage avant d'écarter l'accueil.

## Suite e2e, branche et `main` lancées au même moment (machine chargée, load ≈ 40)

| | passés | échoués | durée |
|---|---:|---:|---:|
| `main` (bf38a6dc5) | 289 | 0 | 5,9 min |
| branche | 289 | 0 | 5,4 min |

## Non traité

Les badges des matchs et des tournois restent un agrégat complet par liste (~0,6 s chacun sur
la base de 66 Mo) : voir `tasks/BACKLOG.md`, « Badges des matchs et des tournois ».

## Premier clic juste après le démarrage

Avant : « prêt » posé avant le premier pixel, préchargement des 45 chunks d'un seul tenant
(le fil principal occupé 1,5 à 2 s), panneau vide sans signe, chunk en échec vide pour la session.
Après : « prêt » posé après la première peinture (`requestAnimationFrame` puis macrotâche),
préchargement chunk par chunk à chaque trame oisive (onglet actif d'abord), indicateur de
chargement dans l'onglet, message et bouton de rechargement quand un chunk échoue (un nouvel
essai part à la prochaine ouverture ; un `import()` en échec peut rester en échec dans le
navigateur, d'où le rechargement).

Mesure (`measure-startup.mjs`, médianes de 5, builds en alternance, machine chargée), en ms :

| Jalon | avant ×1 | après ×1 | avant ×4 | après ×4 |
|---|---:|---:|---:|---:|
| premier rendu (FCP) | 972 | 912 | 3840 | 3120 |
| prêt | 801 | 924 | 3123 | 3466 |

« Prêt » suit désormais le premier rendu (avant : il le précédait de 0,1 à 0,7 s). La
durée des tâches longues après « prêt » n'a pas été remesurée (`preload.mjs` n'est pas dans le dépôt).


## Le panneau Matchs derrière l'accueil

Sans base, l'onglet actif par défaut (Matchs, le plus gros chunk : 90 kB) était chargé au
montage, avant « prêt », sous un écran d'accueil qui le cache. Il est désormais laissé au
préchargement oisif (où il passe toujours en premier) et demandé dès qu'une base s'ouvre ou
que l'accueil est écarté (`activePanelToLoad`, `TabbedPanel.homeScreen.test.js`).

Mesure (`measure-startup.mjs`, builds en alternance, machine chargée et inégale d'une série à
l'autre), en ms ; l'écart « App montée → prêt » est le seul que le changement touche :

| Série | prêt avant | prêt après | montée → prêt avant | montée → prêt après |
|---|---:|---:|---:|---:|
| ×1, 11 essais | 1429 | 1288 | 288 | 206 |
| ×1, 11 essais, ordre inversé | 621 | 623 | 74 | 64 |
| ×4, 9 essais | 2704 | 2155 | 1144 | 778 |
| ×4, 9 essais, ordre inversé | 989 | 869 | 291 | 192 |

JS chargé avant « prêt » : 1223 → 1098 kB, à chaque essai. Le gain sur « prêt » est dans le
bruit à ×1, de l'ordre de 0,1 s à ×4.
