# Balayage du rendu du frontend (2026-10)

Question : ce que coûte le **rendu** de chaque interaction sur une grosse base, backend
exclu. Ce mémo mesure sans rien optimiser. Le démarrage est traité à part
(`latence-demarrage-2026-10.md`).

## Comment mesurer

```bash
cd frontend && npx vite build --sourcemap --outDir dist-measure
node scripts/measure-render.mjs --dist dist-measure --runs 5 --profile 6 [--cpu 4] [--only regex] [--shots dir]
```

- Le build de production est servi par `vite preview` sur le port 4199 (`--port`, jamais 5173),
  et tourne dans Chromium sans tête derrière le mock Wails de l'e2e.
- Le mock est étendu dans la page aux volumes d'une grosse base : 20 000 positions analysées,
  1 000 matchs avec badges (PR, MWC, MWC 7 pts), 60 tournois, 200 collections (dont une de
  2 000 positions), 50 paquets Anki, une recherche de 5 000 résultats et 12 leçons. Il répond en
  temps constant, pour que son propre coût ne compte pas.
- Chaque échantillon part d'une page neuve, prête et préchargée. On mesure de l'action jusqu'à
  la fin de la trame qui suit la dernière mutation du DOM, ou la dernière trame de plus de
  50 ms, une fois la page calme depuis 300 ms. Une trame vide coûte déjà ~17 ms (ligne
  « témoin »).
- « CPU fil principal », script, layout et style viennent des métriques CDP `Performance`
  (TaskDuration, ScriptDuration, LayoutDuration, RecalcStyleDuration) relevées autour du
  scénario. Les long tasks viennent d'un `PerformanceObserver`.
- Les scénarios par pas (j/k, défilement) donnent la médiane et le p95 d'un pas, le rapport
  entre le dernier et le premier quart des pas (1,00 = plat), et le CPU par pas.
- `--profile N` profile les N scénarios au CPU le plus haut (CDP Profiler, échantillon de
  100 µs), agrégés par fonction source via les sourcemaps. Le bucket natif `(program)` (style,
  layout, peinture, boucle d'événements) domine toujours ; les parts sont aussi données sur le
  seul JavaScript.

Charge machine (`/proc/loadavg`) : 9,84 au début de la passe finale, 4,26 à la fin. Les
autres worktrees lançaient leurs e2e en même temps.

## Tableau complet

Médianes de 5 échantillons, Chromium, CPU ×1, 1280×800, interface en anglais.

| scénario | ms | p95 pas | fin/début | CPU fil principal | script | layout | style | long tasks | ms en long tasks | nœuds DOM |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| filtrer matchs (« Alice », attente de 250 ms comprise) | 320 |  |  | 79 | 28 | 16 | 7,3 | 0 | 0 | 2 405 |
| ouvrir onglet Collections (200 lignes) | 117 |  |  | 181 | 52 | 31 | 14 | 2 | 116 | 3 497 |
| ouvrir onglet Matchs (depuis Analyse) | 74 |  |  | 126 | 37 | 19 | 9,4 | 1 | 63 | 1 078 |
| ouvrir leçon (Espace, `le 1`, Entrée) | 69 |  |  | 84 | 23 | 22 | 3,5 | 0 | 0 | 610 |
| ouvrir une collection (2 000 positions) | 52 |  |  | 56 | 25 | 4,7 | 3,8 | 0 | 0 | 734 |
| ouvrir onglet Anki (50 paquets) | 51 |  |  | 102 | 18 | 18 | 6,7 | 0 | 0 | 1 879 |
| ouvrir onglet Recherche | 45 |  |  | 48 | 18 | 3,3 | 3,4 | 0 | 0 | 618 |
| ouvrir onglet Eval | 45 |  |  | 64 | 24 | 5,7 | 3,9 | 0 | 0 | 519 |
| ouvrir onglet Stats | 42 |  |  | 66 | 21 | 5,7 | 6,6 | 0 | 0 | 650 |
| recherche : 5 000 résultats | 41 |  |  | 55 | 22 | 2,7 | 5,2 | 0 | 0 | 604 |
| changer de thème (→ dark) | 40 |  |  | 68 | 11 | 1,3 | 8,8 | 0 | 0 | 1 157 |
| ouvrir aide (première ouverture) | 40 |  |  | 82 | 11 | 20 | 6,4 | 0 | 0 | 4 068 |
| changer de langue (en → fr) | 37 |  |  | 30 | 13 | 4,4 | 2,1 | 0 | 0 | 1 157 |
| rouvrir aide (seconde ouverture) | 34 |  |  | 54 | 3,3 | 7,0 | 8,5 | 0 | 0 | 4 068 |
| j/k : 50 pas (plateau + analyse), par pas | 33 | 37 | 1,00 | 14 | 7,6 | 0,4 | 0,5 | 0 | 0 | 601→596 |
| recherche : page suivante (PageDown) | 33 |  |  | 18 | 9,3 | 0,3 | 0,5 | 0 | 0 | 596 |
| ouvrir onglet Tournois | 33 |  |  | 67 | 18 | 7,1 | 4,8 | 0 | 0 | 1 289 |
| trier matchs (colonne joueur 1) | 29 |  |  | 37 | 14 | 7,3 | 2,5 | 0 | 0 | 1 079 |
| ouvrir configuration | 28 |  |  | 44 | 6,6 | 14 | 1,9 | 0 | 0 | 1 157 |
| défiler matchs (1 000 lignes), par pas | 21 | 23 | 0,99 | 12 | 3,1 | 2,0 | 0,9 | 0 | 0 | 1 078→868 |
| fermer configuration | 19 |  |  | 11 | 1,7 | 0,1 | 0,2 | 0 | 0 | 1 078 |
| défiler positions de collection (2 000 lignes), par pas | 19 | 21 | 1,00 | 9,5 | 2,3 | 1,8 | 0,8 | 0 | 0 | 736→646 |
| fermer aide | 19 |  |  | 11 | 2,8 | 0,1 | 0,1 | 0 | 0 | 1 078 |
| redimensionner (1280×800 → 1100×700) | 18 |  |  | 29 | 13 | 3,8 | 1,3 | 0 | 0 | 1 078 |
| ouvrir onglet Entraînement | 17 |  |  | 40 | 10 | 1,0 | 2,4 | 0 | 0 | 519 |
| témoin (aucune action) | 17 |  |  | 6,9 | 0,9 | 0 | 0 | 0 | 0 | 1 078 |
| ouvrir onglet Analyse | 16 |  |  | 34 | 8,0 | 1,5 | 1,9 | 0 | 0 | 601 |
| ouvrir onglet Commentaires | 13 |  |  | 29 | 6,6 | 2,3 | 1,9 | 0 | 0 | 499 |

Lecture :
- Rien ne dépasse 120 ms une fois le témoin et l'attente volontaire du filtre (250 ms)
  retirés. Seuls Collections et Matchs produisent des long tasks.
- La virtualisation tient : défilement plat (0,99 et 1,00), DOM stable autour de 650 à
  1 100 nœuds.
- La navigation j/k coûte 14 ms de CPU par pas, sans dérive sur 50 pas. Elle reste dans
  l'ordre des 16,8 ms mesurés dans `optimisation-2026-10.md`, même sur une position analysée.
- Le panneau « positions » est la liste des positions d'une collection, la seule liste de
  positions de l'interface. La liste des matchs défile sur ses 1 000 lignes (le volume
  demandé), celle de la collection sur 2 000.

## Profils (les six scénarios au CPU le plus haut)

### 1. Ouvrir l'onglet Collections : 181 ms de CPU, 2 long tasks
- Où va le temps : 76 % natif (création et layout de 3 500 nœuds). Le premier poste
  JavaScript est `formatDateTime` (`src/utils/format.js:66`) : 25 ms, soit 39 % du JS et 8 %
  du total, appelé par la colonne date de chaque ligne (`CollectionPanel.svelte:753`, via le
  `formatDate` local de la ligne 301).
- Cause : chaque appel construit un `new Intl.DateTimeFormat(...)`, une opération chère
  (résolution de locale, ~0,1 ms) répétée 200 fois. Les 200 lignes ne sont pas virtualisées,
  car `PanelTable` ne virtualise qu'**au-delà** de 200 (`virtualizeAbove = 200`,
  `PanelTable.svelte:137`) : la totalité du tableau est montée.
- Gain estimé : ~25 ms en gardant un formateur par (locale, options). Virtualiser dès 100
  lignes ramène le DOM à ~600 nœuds, soit 60 à 100 ms et la fin des long tasks.
- Effort : S pour le cache, S-M pour le seuil (à vérifier sur les autres tables).
- Fichiers : `src/utils/format.js` (`formatDate`, `formatDateTime`, `formatNumber`,
  `formatPercent`) ; `src/components/panels/PanelTable.svelte`.
- Neuf.

### 2. Ouvrir l'onglet Matchs depuis Analyse : 126 ms de CPU, 1 long task de 63 ms
- Où va le temps : 82 % natif. Côté JS, le démontage du panneau quitté (`remove`, sous
  `TabbedPanel.svelte:380`) prend 11 ms, soit 39 % du JS. `formatNumber`
  (`format.js:73`, appelé par `fmtMwc7` et `mwc7Tooltip` dans `src/utils/mwc7.js`) prend 7 %
  du JS.
- Cause : chaque bascule d'onglet démonte un panneau et en remonte un autre
  (`TabbedPanel` remonte son enfant à chaque fois), puis le tableau des matchs reconstruit
  ses ~29 lignes visibles. Chaque ligne crée trois `Intl.NumberFormat` (deux badges MWC 7 pts
  et leur infobulle).
- Gain estimé : 5 à 10 ms avec le cache de formateurs. Garder les panneaux montés et masqués
  supprimerait le démontage et le remontage (~60 ms), au prix de la mémoire et des effets
  vivants des panneaux cachés.
- Effort : S pour le cache, L pour garder les panneaux montés (contrat de montage de
  chaque panneau, voir `screenshot-panels.spec.js`).
- Fichiers : `src/utils/format.js`, `src/utils/mwc7.js`, `src/components/TabbedPanel.svelte`.
- Le cache de formateurs est neuf. Le remontage à chaque bascule est connu (il est décrit
  dans les specs), son coût jamais chiffré.
- Suite : le panneau Matchs reste monté, caché, une fois visité (`KEEP_MOUNTED` de
  `TabbedPanel.svelte`) ; seul lui, car ses effets globaux s'arrêtent tous sur `openPanels`
  (Stats recalcule à chaque invalidation, Collections recharge à chaque position). Scénario
  « bascule analysis → matches (déjà visité) », médianes de 9, machine chargée (loadavg 12 à
  30) : script 11 → 5,6 ms au calme, 23 → 4,8 et 54 → 16 ms sous charge ; CPU fil principal
  −25 à −45 %. Le retour vers Analyse ne change pas. Le coût : ~830 nœuds DOM gardés.

### 3. Ouvrir l'onglet Anki : 102 ms de CPU
- Où va le temps : 90 % natif. Le JS est éparpillé : le snippet `icon`
  (`AnkiPanel.svelte:569`) et les cellules (`AnkiPanel.svelte:850`) dépassent à peine
  2 ms chacun.
- Cause : chaque paquet porte quatre boutons à icône SVG, plus des `CountLink`, ce qui fait
  ~1 900 nœuds pour 50 lignes. Le coût est la création et la mise en page de ces SVG, pas le
  script.
- Gain estimé : 20 à 40 ms en sortant les icônes en sprite (`<use href>`) ou en ne montrant
  les actions qu'au survol ou sur la ligne sélectionnée.
- Effort : M.
- Fichiers : `src/components/AnkiPanel.svelte` ; le même motif `icon` vit dans d'autres
  panneaux.
- Neuf.

### 4. Ouvrir une leçon (Espace, `le 1`, Entrée) : 84 ms de CPU
- Où va le temps : 47 % du JS est un seul `focus()` (21 ms) dans l'effet qui ouvre la ligne
  de commande (`StatusBar.svelte:141`).
- Cause : `focus()` force un recalcul de style et un layout synchrones de toute la page (la
  liste des matchs ouverte, le plateau) au moment où la barre change de forme. La leçon
  elle-même (plateau, `LessonBar`) coûte peu.
- Gain estimé : ~20 ms à chaque ouverture de la ligne de commande (Espace), pas seulement
  pour les leçons. Par exemple, focaliser avec `preventScroll` après la trame
  (`requestAnimationFrame`) plutôt qu'au milieu du flush.
- Effort : S.
- Fichiers : `src/components/StatusBar.svelte`.
- Neuf.

### 5. Ouvrir l'aide : première ouverture 82 ms de CPU, seconde 54 ms
- Où va le temps : 89 % natif. Côté JS, `cachedHtml`/`mount` (`HelpModal.svelte:39`, 44 % du
  JS) insère le manuel analysé, puis vient `src/i18n/help/index.js:32` (13 %). À la seconde
  ouverture, le cache DOM retire presque tout le script (11 → 3,3 ms). Le layout baisse
  (20 → 7 ms), mais le style monte (6,4 → 8,5 ms) : les 4 000 nœuds réinsérés sont
  restylés.
- Cause : le manuel entier est dans le DOM, pas seulement la section affichée.
- Gain estimé : 30 à 50 ms par ouverture en ne montant que la section visible ou
  `content-visibility: auto` sur les sections. À évaluer contre la recherche dans l'aide,
  qui parcourt le DOM.
- Effort : M.
- Fichiers : `src/components/HelpModal.svelte`, `src/i18n/help/index.js`.
- Connu en partie : le cache DOM par langue et par onglet existe déjà et produit ce gain de
  script. Le coût du style à la réouverture est neuf.

### 6. Filtrer les matchs (« Alice ») : 79 ms de CPU, 320 ms de mur dont 250 d'attente volontaire
- Où va le temps : `formatNumber` (`format.js:73`) pèse 28 % du JS, puis les cellules de
  `MatchPanel.svelte:1232-1305`.
- Cause : le résultat (~100 lignes) passe sous le seuil de virtualisation, donc toutes les
  lignes sont montées (2 400 nœuds), et chacune refait ses `Intl.NumberFormat`. Les 250 ms
  sont le debounce de saisie (`MatchPanel.svelte:394`), voulu.
- Gain estimé : ~7 ms avec le cache de formateurs. Le reste suit le seuil de virtualisation
  (constat 1).
- Effort : S.
- Fichiers : `src/utils/format.js`, `src/utils/mwc7.js`, `PanelTable.svelte`.
- Neuf.

## Ce qui recoupe les mémos existants

- `optimisation-2026-10.md`, « Front, listes longues » : virtualisation, tri (13 ms) et
  navigation (16,8 ms). Ces chiffres sont confirmés avec une analyse affichée et des badges :
  tri 29 ms de mur (37 ms de CPU), j/k 14 ms de CPU par pas, défilement plat. La ligne
  « listes longues quasi constantes » reste vraie.
- La piste A9 du même mémo (petits re-rendus du plateau) n'apparaît dans aucun profil, ce qui
  est cohérent avec « marginal ».
- `latence-demarrage-2026-10.md` : les chunks différés sont tous préchargés avant chaque
  mesure. Aucun constat ne vient du chargement de code.
- Neufs : le formateur `Intl` recréé à chaque appel (constats 1, 2 et 6), le seuil
  `virtualizeAbove` qui laisse monter 200 lignes et tout résultat filtré, le `focus()`
  synchrone de la ligne de commande, les icônes SVG d'Anki, le style à la réouverture de
  l'aide.

## Chromium n'est pas WebKitGTK

L'app réelle tourne dans WebKitGTK (JavaScriptCore, moteur de rendu WebKit), plus lent que
Chromium de bureau. `--cpu 4` approche un portable modeste, sans reproduire le moteur.

- **Formateurs `Intl`** (constats 1, 2 et 6) : sous JSC, `Intl` s'appuie aussi sur ICU, et la
  construction d'un formateur y est au moins aussi chère. Le gain du cache devrait tenir ou
  grandir. À vérifier avec `jsc`, comme pour les dates (voir la mémoire `webkit_date_parse`).
- **DOM, style et layout** (constats 1, 3 et 5, et le démontage du constat 2) : WebKit
  restyle et met en page plus lentement que Blink, ce qui alourdit le poids natif (75 à 90 %
  ici). Les gains de réduction du DOM (virtualisation, sprite d'icônes, aide partielle) sont
  probablement plus grands dans l'app.
- **`focus()` synchrone** (constat 4) : le layout forcé existe dans les deux moteurs ; sa
  durée suit le coût du layout de WebKit.
- **Long tasks** : WebKitGTK n'expose pas l'API Long Tasks. Les comptes ci-dessus valent pour
  Chromium seul. Dans l'inspecteur du WebView, il faut lire la timeline.
- Les temps absolus de ce mémo ne sont pas ceux de l'app. Le classement relatif est ce qui
  doit tenir. À confirmer par une passe `--cpu 4` et un essai à la main dans l'inspecteur de
  WebKitGTK sur Collections et sur l'aide.
