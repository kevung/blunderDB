# Générer des diagrammes de backgammon en SVG : architecture, conventions et bibliothèques (Go + JS)

## TL;DR
- **Adoptez un module SVG unique comme source de vérité** (le SVG comme format pivot) : générez-le en Go pour le desktop/serveur, puis dérivez TOUS les autres formats (rendu écran, PNG presse-papier, export fichier, rapport HTML imprimable, client web) de ce même générateur. Pour le client web, servez le SVG depuis Go ou compilez le générateur Go en WASM plutôt que de réimplémenter la logique en JS.
- **Pour la conversion SVG → PNG en Go pur, utilisez `kanrichan/resvg-go`** (moteur resvg via WASM/wazero, sans cgo, gère le texte et les dégradés). **Évitez `srwiley/oksvg`** qui ignore silencieusement les éléments `<text>` — fatal pour un diagramme comportant numéros de points, dés, score et pip count. Si vous dessinez directement (sans fichier SVG intermédiaire), `tdewolff/canvas` (MIT) est excellent.
- **Suivez les conventions Magriel/Robertie** (points numérotés 24→1 du point de vue du joueur au trait, notation `8/5 6/5`, astérisque pour les hits, score « X-away/Y-away », Crawford) ; côté logiciels, XG et GNU Backgammon fixent le standard des IDs (XGID, Position ID/Match ID) et des exports.

## Key Findings

1. **Il existe déjà des générateurs directement réutilisables ou inspirants.** Le plus pertinent pour votre architecture est **BgDiagram** d'Alessandro Scotti — rendu vectoriel (SVG), JavaScript, activement développé en 2025 — décrit par son auteur comme un « highly flexible and configurable vector-based backgammon diagram builder ». **backgammon-js** (canvas, XGID, sans dépendance) et **bgLog** (moteur derrière de nombreux livres publiés et fédérations, avec un mode « print quality ») complètent le paysage.
2. **Les formats d'ID sont parfaitement spécifiés et documentés officiellement.** Le GNU Position ID (14 caractères base64) et le Match ID (12 caractères base64) disposent d'une spécification publiée dans le manuel GNU Backgammon. Le XGID (26 caractères de position + 9 champs séparés par `:`) est documenté par eXtreme Gammon et par plusieurs parseurs open source.
3. **Le principal arbitrage SVG est le texte** : conserver de vrais `<text>` (accessibilité, rendu écran) versus convertir le texte en chemins (rendu déterministe pour export/impression). La solution retenue est de générer deux variantes du même SVG.
4. **L'écriture d'une image PNG dans le presse-papier est désormais fiable** côté navigateur (API async Clipboard, Baseline « newly available » depuis le 30 mars 2025) et côté desktop Go (`golang.design/x/clipboard`, PNG uniquement).
5. **L'accessibilité est réalisable** avec le pattern `role="img"` + `<title>` + `<desc>` + `aria-labelledby`, en générant une description textuelle ordonnée de la position.

## Details

### 1. Générateurs existants

**Services web / outils en ligne**
- **AnkiGammon Tools – Position Converter & Visualizer** (`ankigammon.com/tools/position-converter`) : convertit XGID/GNUID/OGID (détection automatique du format), visualise le plateau, exporte un PNG haute résolution, propose 7 schémas de couleurs et le changement d'orientation. Tourne entièrement dans le navigateur (aucun upload). Récent (2025). L'écosystème AnkiGammon (application desktop `Deinonychus999/AnkiGammon` et paquet PyPI) est sous **GPL-3.0**.
- **apbg.net – Backgammon Diagrams** (`apbg.net/bg_diagrams.php`) : générateur historique s'appuyant sur les scripts de Claes Norreen (adaptés pour BBCode & GNUBG ID), gère XGID et Position/Match ID, les décisions take/pass et l'affichage des scores Crawford.
- **bgLog** (`bglog.org`) : moteur graphique derrière plusieurs livres publiés (Ray Kershaw *Backgammon Funfair*, Simon Hill *Backgammon for Losers*, Chris Bray *Backgammon in the Wind* / *Position of the Day*), l'application FlashBack et la UK Backgammon Federation. Contrôle très fin : 19 champs de couleur indépendants, jusqu'à 4 cubes, styles de numérotation et de score, effets 3D, et un bouton « Snapshot (print) » pour des diagrammes de qualité impression. Accepte les IDs XG et gnuBG. C'est le successeur de GABBI (« Generate A Backgammon Board Interactively », 2006), la version actuelle étant en développement depuis 2014.
- **kinchan.com/bg/board.html** : ancien générateur qui produit du HTML de plateau.

**Bibliothèques / dépôts (JS, Python, Go, R)**
- **BgDiagram** — `github.com/ascottix/bgdiagram` (démo : `ascottix.github.io/bgdiagram/bgdiagram_demo.html`). « Highly flexible and configurable vector-based backgammon diagram builder », en JavaScript, à rendu SVG. Projet sœur **BgDiagramDb** (`github.com/ascottix/bgdiagramdb`) : base de positions fonctionnant entièrement dans le navigateur, avec « smart review, pip counting, and position tracking ». L'auteur est actif sur le backgammon en 2025 (il a aussi publié `gnubg-core`, un portage web du cœur de GNU Backgammon, annoncé sur la liste `bug-gnubg` le 22 avril 2025). **C'est la meilleure référence à étudier/réutiliser pour un générateur SVG.** Sa licence exacte n'a pas pu être confirmée pendant la recherche (voir Caveats).
- **backgammon-js** — `github.com/gtback/backgammon-js` : bibliothèque JavaScript sans dépendance dessinant un plateau sur un canvas HTML5, rendant une position depuis une chaîne XGID ; thèmes intégrés (Maple, Midnight, Ocean, Slate) via un objet `BoardStyle` réutilisable ; documentation XGID incluse.
- **bgboard** — `github.com/gtzampanakis/bgboard` : plateau JS sans dépendance, format d'entrée = `PositionID:MatchID` GNU, affiche longueur de match/score/Crawford. Utilise des **images bitmap** (dossier `img/`) plutôt qu'un dessin vectoriel — donc moins adapté comme base SVG.
- **xgid2anki** (`github.com/ngvlamis/xgid2anki`) et **AnkiGammon** : convertissent des positions en cartes Anki ; le rendu s'appuie sur **bglog.js** + Chromium headless (Playwright). **GPL-3.0**.
- **lassehjorthmadsen/backgammon** (R) : parseur/convertisseur GNU ID ↔ XGID (fonctions `posid2xgid`, `matchid2xgid`, `gnuid2xgid`), utile comme référence d'algorithme de décodage.
- **kevung/bgfparser** (Go) : parseur de fichiers/IDs backgammon en Go.

**Paquets LaTeX (CTAN)**
- **`bg`** (`ctan.org/pkg/bg`) : « Annotate backgammon matches and positions ». Fournit un paquet + une police (source METAFONT) et deux environnements, `position` (un plateau) et `game` (série de coups, l'état du plateau étant maintenu automatiquement). Les plateaux sont produits par une police METAFONT (pas d'inclusion PostScript), donc affichables par tout previewer DVI. Archive de 119,8 ko. C'est le paquet historique (ex `macros/latex209/contrib/backgammon`). L'ancien *TeX Catalogue* indique une licence « unknown ».
- **`bakoma-games`** (`ctan.org/pkg/bakoma-games`) : modules BaKoMa pour les jeux (chap. 8 du *LaTeX Graphics Companion*). Avertissement de qualité : « For Backgammon the fonts were originally dithered as halftone, making them unsuitable for PDF ». Rendu PDF médiocre.
- **GNU Backgammon** exporte lui-même en **LaTeX (.tex)** (voir §2), option pratique si vous voulez des diagrammes LaTeX sans dépendre d'un paquet tiers.

**Polices de diagramme.** La police METAFONT du paquet `bg` est l'exemple canonique. Historiquement, les magazines (Flint Area Backgammon News, GammOnLine) utilisaient des polices de diagramme dédiées ; ces polices halftone/METAFONT sont aujourd'hui dépassées par le rendu vectoriel (SVG/canvas), qui est la voie à privilégier.

**Formats d'ID (spécifications)**

*GNU Backgammon — Position ID* (source officielle : *GNU Backgammon Manual*, `gnu.org/software/gnubg`). Le manuel précise : « The 10 byte binary format is called the key, and the 14 character ASCII format is the ID. » Construction : pour chaque point, du point-as du joueur au trait jusqu'au point 24 puis la barre, on ajoute autant de `1` que de pions présents, suivis d'un `0` ; on répète pour l'adversaire ; on complète à 80 bits par des `0` (little-endian). L'ID est le Base64 de la clé, « but this padding is omitted » (les deux `=` de padding sont retirés). La position de départ = **`4HPwATDgc/ABMA`**.

*GNU Backgammon — Match ID* : « The match key is a 9 byte representation… The match ID is the 12 character Base64 encoding of the match key. » Il encode : score du match, longueur du match, valeur du cube (bits 1-4 = log2 de la valeur : « a 8-cube is encoded as 0011 binary (or 3), since 2 to the power of 3 is 8 » ; maximum = 2^15, cube 32768), propriétaire du cube (bits 5-6 : `00` joueur 0, `01` joueur 1, `11` centré), drapeau Crawford, joueur au trait, joueur devant décider, drapeau « doublé », résignation (bits 14-15), dés (bits 16-21), longueur du match (bits 22-36, max 32767). Match ID de départ = **`MIEFAAAAAAAA`**.

*XGID* : `XGID=<26 caractères de position>:c1:c2:c3:c4:c5:c6:c7:c8:c9`.
- **Position (26 caractères)** : `-` = point vide ; minuscules `a`–`p` = les pions d'un joueur (`a`=1, `b`=2, … `o`=15 pions sur le point) ; majuscules `A`–`P` = l'autre joueur. Les 26 caractères = barre + 24 points + barre.
- **Les 9 champs :**
  1. **Valeur du cube**, en exposant de 2 (`0`=1, `1`=2, `2`=4, `3`=8…).
  2. **Position/propriétaire du cube** : `0`=centré, `1`=le joueur du bas possède, `-1`=l'adversaire possède.
  3. **Joueur au trait** : `1`=bas, `-1`=haut, `0`=indéfini.
  4. **Dés** : ex. `63` = 6-3 ; `00` = pas de dés (décision de cube).
  5. **Score joueur 1 (bas)**.
  6. **Score joueur 2 (haut)**.
  7. **Drapeau Crawford** (en match) / Jacoby (en money game).
  8. **Longueur du match** (`0` = money/illimité).
  9. **Cube maximum**, en exposant de 2 (souvent `8` → 256, ou `10` → 1024).
- Exemple (manuel XG2, p. 146) : `XGID=-a-B--E-B-a-dDB--b-bcb----:1:1:-1:63:0:0:0:3:8` = cube à 2 possédé par le joueur du bas, joueur du haut au trait, 6-3 lancé, score 0-0, pas Crawford, match en 3 points, cube max 2^8. **Les champs 1 et 9 sont des exposants, pas la valeur littérale du cube** — c'est la principale source de confusion. À noter aussi : coller un XGID dans un forum peut corrompre les doubles tirets (`--`) transformés en tiret cadratin (utilisez un bloc de code/préformaté).

### 2. Conventions de représentation

**Orientation & numérotation.** La notation de backgammon « is a means for recording backgammon games, developed by Paul Magriel in the 1970s » (*Backgammon*, 1976), reprise par la quasi-totalité des auteurs (Robertie, Woolsey, Kleinman, Trice, Ballard/Weaver, Bagai). Les 24 points sont numérotés **24 → 1 du point de vue du joueur au trait**, le point 1 étant le plus proche de son bear-off tray ; ses pions se déplacent vers son jan intérieur (home board). Quand l'adversaire est au trait, la numérotation s'inverse (son point 24 devient son point 1). Les dés sont notés « 4-2 » ou « 42 ».

**Notation des coups.** `4-2: 8/4 6/4` = avec un 4-2, un pion de 8 vers 4 et un de 6 vers 4. Conventions courantes : `bar/20` pour une entrée depuis la barre, **astérisque** pour un hit (`8/5*`), `(2)` pour un coup répété. C'est la notation algébrique standard « à la Magriel ».

**Score & cube.** Notation « X-away / Y-away » (points restants pour gagner le match) ; money game / illimité (souvent avec règle Jacoby) ; **Crawford game** (jeu unique sans cube quand un joueur atteint 1-away, le cube redevenant utilisable après). Le videau (cube) est représenté sur le bord du plateau : **au centre** s'il n'appartient à personne, **du côté d'un joueur** s'il le possède, avec sa valeur (2, 4, 8…). Le pip count est généralement affiché à côté de chaque camp.

**Exports d'images des logiciels de référence**
- **eXtreme Gammon (XG)** : « Save the current position in an Image file » avec choix JPG/GIF/BMP et taille (Large/Medium/…) ; « Save the current position in the clipboard » (image) et « Save the current position ID in the clipboard » (XGID, via Ctrl+Shift+C) ; export HTML de tout le match pour publication/impression ; le menu « Layout » configure les éléments visuels et l'Evaluation Meter. Option d'afficher le pip count à l'export HTML.
- **GNU Backgammon** : exporte en **PNG, PostScript (.ps/.eps), PDF, HTML, LaTeX (.tex), texte, Snowie text et SVG** ; « board designs » configurables ; export « position as HTML/BBCode/text » ; rendu ASCII du plateau sur terminal. C'est **GPL** et sa liste d'exports est un bon cahier des charges pour votre propre module.
- **BGBlitz** : inclut le XGID dans tous ses exports ASCII/HTML. **Backgammon Galaxy**, **XG Mobile** : partage/export de positions.

### 3. Le SVG en pratique

**Texte : `<text>` vs chemins (paths).** Les anciens éléments SVG `<font>`/`<glyph>` sont obsolètes et mal supportés (n'existent pas sur le web moderne). Deux stratégies :
- **Vrai `<text>`** : nécessaire pour l'accessibilité (lecteur d'écran, indexation) et le rendu écran. Mais le rendu dépend de la disponibilité de la police chez le lecteur/convertisseur. Piège majeur : **un SVG chargé dans une balise `<img>` n'accède pas au CSS externe de la page ni aux polices chargées par la page** — d'où des polices manquantes/de substitution.
- **Texte converti en chemins** : rendu identique partout, aucune dépendance de police, idéal pour l'export fichier/PNG et l'impression ; coût = perte de l'accessibilité du texte et fichier plus lourd.
- **`@font-face` avec data URI** (WOFF/WOFF2 en base64 dans un bloc `<style>` du SVG) : rend le SVG autonome mais alourdit fortement le fichier (typiquement 20–100 ko par graisse) ; **sous-ensemblez** impérativement la police aux seuls glyphes utilisés.

Pour le backgammon, le texte se limite à des chiffres et quelques symboles. **Recommandation : générez deux variantes du même SVG** : (a) variante « web/écran » avec de vrais `<text>` (police fournie par la page via `@font-face`, ou inline pour l'accessibilité) ; (b) variante « export/impression » avec le texte converti en chemins (ou police sous-ensemblée en data URI) pour un rendu déterministe.

**Impression & résolution.** Travaillez en unités utilisateur avec un `viewBox` fixe et laissez le SVG se mettre à l'échelle. Pour un rapport A4 imprimable : dimensionnez le diagramme en mm (attributs width/height en mm ou CSS d'impression) ; pour la rastérisation en PNG à 300 dpi, taille bitmap = (largeur en pouces) × 300. Utilisez `@media print` et `print-color-adjust: exact` (anciennement `-webkit-print-color-adjust: exact`) pour forcer l'impression des couleurs de fond des points et pions.

**Conversion SVG → PNG en Go pur (sans cgo) — évaluation :**

| Bibliothèque | Texte `<text>` | Dégradés / clip / transform | cgo | Maintenance | Licence | Verdict |
|---|---|---|---|---|---|---|
| **srwiley/oksvg + rasterx** | ❌ ne gère pas `<text>` ; defs/gradients partiels (l'auteur note « many elements like defs, gradients, or animations have not been added ») | paths SVG 2.0 complets | non | dépôt peu actif (miroirs `kb-sp`, `qiniu`) | BSD-like | **À éviter** pour des diagrammes contenant du texte |
| **tdewolff/canvas** | ✅ excellent (formatage, embarque/sous-ensemble TTF/OTF/WOFF/WOFF2, ou conversion en outlines) | ✅ complet | non (Go pur) | **actif** (~1,8k★) | **MIT** | **Idéal si on dessine directement** ; le *chargement* de fichiers SVG est encore WIP |
| **kanrichan/resvg-go** | ✅ (resvg via WASM/wazero : `LoadFontData`, `ConvertText`, `Render`) | ✅ (moteur resvg complet) | non (WASM wazero) | **actif** (maj janv. 2026) | à vérifier (moteur resvg = MPL-2.0) | **Meilleur choix pour rendre un fichier SVG existant** en PNG en Go pur |
| **ajstarks/svgo** | génération seulement | n/a | non | stable | — | Génération de SVG, pas de rastérisation |
| **fogleman/gg** | ✅ (dessin, pas de parsing SVG) | dessin 2D | non | peu actif | MIT | Alternative pour dessiner le PNG directement |
| **golang.org/x/image/vector** | bas niveau | rastériseur seul | non | actif | BSD | Brique bas niveau (utilisée par rasterx) |

Pièges : **oksvg ignore silencieusement `<text>`** (et une partie de defs/gradients) sans émettre d'erreur — fatal pour un diagramme avec numéros de points, dés, score et pip count. **resvg-go** embarque le moteur resvg (Rust) compilé en WASM et exécuté par wazero ; il faut charger explicitement les polices (`LoadFontData`/`LoadSystemFonts`), appeler `ConvertText`, et **les workers ne sont pas goroutine-safe** (à isoler par requête ou via un pool sérialisé).

**Alternative : dessiner le PNG directement en Go** avec `tdewolff/canvas` (ou `fogleman/gg`) plutôt que de passer par un fichier SVG. Avantage : un seul moteur, contrôle total du texte, pas de parseur SVG à maintenir ; `canvas` peut d'ailleurs **sortir SVG *et* PNG (et PDF/EPS) depuis la même description**. Inconvénient : vous ne réutilisez plus un « fichier SVG source » ; la logique de dessin doit être écrite dans l'API canvas. **C'est en réalité une excellente option pour un projet Go** : elle unifie SVG + PNG + PDF en une seule description vectorielle.

**Dans le navigateur.** Rasteriser un SVG en canvas : créer une `Image` depuis un `Blob`/data URL du SVG, la dessiner dans un `<canvas>` (ou `OffscreenCanvas`), puis `canvas.toBlob('image/png')`. Pièges : le SVG chargé dans `<img>` n'accède pas au CSS/polices de la page (embarquer la police en data URI ou convertir en paths) ; risque de **tainted canvas** si le SVG référence des ressources cross-origin, ce qui interdit alors `toBlob`. Alternative sans canvas : **resvg-wasm** (`@resvg/resvg-wasm`, **MPL-2.0**, v2.6.2) ou `svg2png-wasm` pour un rendu PNG déterministe côté client, **avec les mêmes polices que le serveur** — utile pour garantir un rendu identique écran/export/serveur.

**Écrire l'image dans le presse-papier (navigateur).** `navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })])`. L'API async Clipboard (avec `ClipboardItem.supports()`) est **Baseline « newly available » depuis le 30 mars 2025** (Chrome, Edge, Firefox, Safari). Contraintes : **PNG toujours supporté** (text/plain, text/html, image/png sont garantis) ; **contexte sécurisé (HTTPS) requis** ; geste utilisateur ; la fenêtre doit avoir le focus. **`image/svg+xml` n'est PAS garanti** (support variable) — d'où le PNG comme format de presse-papier universel. Pour copier un SVG, on peut sinon copier son *code source* en text/plain.

**Presse-papier en desktop Go.** **`golang.design/x/clipboard`** (**MIT**, Changkun Ou / golang.design Initiative) : `clipboard.Init()` puis `clipboard.Write(clipboard.FmtImage, pngBytes)` — les octets image doivent être **PNG** (l'implémentation transcode PNG ↔ format natif de la plateforme). Multi-plateforme macOS/Linux(X11)/Windows/Android/iOS ; **images « Desktop-only »** (mobile = texte seulement). Sous Linux, nécessite X11 (libx11-dev, donc cgo pour X11) ; sous Windows, pas de cgo ni de dépendance ; **Wayland tombe en repli via XWayland** (GNOME < 49 non supporté nativement). Sur X11, la donnée disparaît à la fin du processus s'il n'y a pas de gestionnaire de presse-papier. `atotto/clipboard` ne gère que le texte et ne convient donc pas pour l'image.

### 4. Accessibilité

**Pattern SVG accessible recommandé.** D'après les tests inter-navigateurs de Deque, le pattern le plus fiable est un SVG inline avec `role="img"` + `<title>` + `<desc>` + `aria-labelledby="idTitle idDesc"`, et `focusable="false"` (pour éviter que d'anciens lecteurs tabulent dedans). Masquez les éléments purement décoratifs avec `aria-hidden="true"`. **SVG ne supporte pas `alt`** : utilisez `<title>` (nom accessible court) et `<desc>` (description longue). Cela couvre **WCAG 2.2 – 1.1.1 Non-text Content**.

**Texte alternatif d'une position — structure déterministe recommandée :** joueur au trait et couleur ; puis, pour chaque point de 24 à 1, le nombre de pions et leur couleur ; pions sur la barre de chaque joueur ; pions sortis (borne off) ; dés ; cube (valeur + propriétaire/centre) ; score (« X-away / Y-away », Crawford ou non) ; pip count des deux joueurs. Gardez `<title>` en résumé et mettez le détail dans `<desc>` ; pour une image complexe, encadrez le SVG d'un `figure/figcaption` HTML, ou proposez une description longue dépliable via `<details>/<summary>`.

**Transposable des échecs.** Les échecs offrent des précédents utiles : la notation FEN comme équivalent textuel d'une position, et les efforts d'accessibilité de lichess/chess.com (annonce des positions, navigation clavier). L'équivalent backgammon est le couple **XGID / GNU ID** (données machine) + une **description en langage naturel** (accessibilité humaine).

**Contraste & daltonisme.** Respectez **WCAG 1.4.3** (contraste du texte : numéros, score, pip) et **1.4.11** (contraste des éléments non textuels : bord des pions, dés, cube). **Ne distinguez jamais les deux camps par la seule teinte** (rouge/vert problématique) : combinez clair/foncé + contour marqué + éventuellement un motif, et assurez un ratio de contraste suffisant entre pions et points. Prévoyez un thème monochrome/haute lisibilité (comme le thème « Monochrome » d'AnkiGammon).

## Recommendations

**Architecture — un générateur SVG comme source unique de vérité.**

1. **Cœur de rendu = une fonction pure `position → SVG`.** Entrée : une structure normalisée (24 points + 2 barres + 2 bear-off, cube, dés, scores, longueur de match, Crawford, pip count, flèches de coup optionnelles), obtenue en parsant XGID *ou* GNU Position/Match ID (réutilisez les algorithmes de `lassehjorthmadsen/backgammon` ou `kevung/bgfparser`). Sortie : une chaîne SVG avec `viewBox` fixe, groupes nommés (`<g id="point-13">`, `checkers`, `dice`, `cube`, `score`, `pip`, `arrows`) et accessibilité intégrée (`role="img"`, `<title>`, `<desc>`, `aria-labelledby`).

2. **Dérivés à partir de ce générateur :**
   - **Écran (aujourd'hui two.js)** : à terme, injectez le SVG inline natif (supprime la dépendance two.js et donne l'accessibilité gratuitement) ; conservez de vrais `<text>` + police via `@font-face` de la page.
   - **PNG presse-papier (desktop Go)** : variante SVG « texte→paths » → **`kanrichan/resvg-go`** → PNG → **`golang.design/x/clipboard`** `FmtImage`.
   - **PNG presse-papier (web)** : SVG → `<canvas>`/`OffscreenCanvas` → `toBlob('image/png')` → `navigator.clipboard.write`, **ou** `@resvg/resvg-wasm` pour un rendu identique au serveur.
   - **Export fichier** : SVG natif (vectoriel, éditable) + PNG (resvg-go) + PDF (via `tdewolff/canvas`).
   - **Rapport HTML imprimable** : SVG inline (accessible) + CSS `@media print` + `print-color-adjust: exact`, dimensions en mm, un diagramme par `figure` avec `figcaption`.
   - **Client web futur** : voir le partage de code ci-dessous.

3. **Partage de code Go ↔ JS — trois options, par ordre de préférence :**
   - **(A) Générer le SVG côté Go et le servir au client** (endpoint HTTP `image/svg+xml`, ou SVG inline dans le HTML). Zéro duplication, une seule implémentation. Idéal pour le rapport et l'affichage statique/serveur. Limite : interactivité côté client réduite.
   - **(B) Compiler le générateur Go en WASM** et l'appeler depuis le navigateur (`syscall/js`). Une seule base de code, rendu strictement identique partout, interactivité possible. Coût : taille du binaire WASM et complexité de build.
   - **(C) Réimplémenter en JS** (comme BgDiagram / backgammon-js). Meilleure ergonomie et performance web, mais duplication de logique à maintenir en parallèle (risque de divergence de rendu écran/serveur).
   - **Décision recommandée :** commencez par **(A)** pour le rapport HTML et l'écran serveur ; adoptez **(B) WASM** dès que le client web a besoin d'interactivité, en réutilisant le générateur Go tel quel. Ne réservez **(C)** qu'aux cas où les contraintes de poids/perf l'imposent réellement. Si vous choisissez de dessiner directement en Go avec `tdewolff/canvas`, notez qu'il génère nativement SVG + PNG + PDF, ce qui simplifie encore l'architecture (un seul moteur de dessin, plusieurs sorties).

4. **Bibliothèques retenues (avec licences) :**
   - Go, SVG → PNG (rendu d'un fichier SVG) : **`kanrichan/resvg-go`** (WASM/wazero, sans cgo ; moteur resvg = MPL-2.0 — vérifier la licence du wrapper).
   - Go, dessin vectoriel unifié SVG/PNG/PDF (option forte) : **`tdewolff/canvas`** — **MIT**.
   - Go, presse-papier image : **`golang.design/x/clipboard`** — **MIT**.
   - Web, SVG → PNG déterministe (option) : **`@resvg/resvg-wasm`** — **MPL-2.0** (v2.6.2).
   - Inspiration / réutilisation JS : **BgDiagram** (ascottix, licence à confirmer), **backgammon-js** (gtback).
   - Parsing d'IDs : **`lassehjorthmadsen/backgammon`** (R, référence d'algorithme) ou **`kevung/bgfparser`** (Go).

**Benchmarks / seuils qui feraient changer ces choix :**
- Si le rendu par resvg-go dépasse un budget de latence acceptable sur un rapport batch (50+ positions), basculez sur un dessin direct `tdewolff/canvas` (pas de parsing SVG à chaque diagramme).
- Si le client web n'a besoin que d'affichage (pas d'édition interactive), restez sur l'option (A) serveur et n'introduisez pas de WASM.
- Si l'accessibilité impose du vrai texte *dans le PNG* exporté (cas rare), conservez `<text>` + police embarquée en data URI plutôt que la conversion en paths.

## Caveats
- **Licence de BgDiagram non confirmée** : le dépôt `github.com/ascottix/bgdiagram` n'a pas pu être ouvert pendant la recherche. Vérifiez directement `…/blob/main/LICENSE` avant toute réutilisation (MIT est plausible au vu des autres projets de l'auteur, mais non prouvé). De même, vérifiez la licence exacte du wrapper `kanrichan/resvg-go` sur son dépôt (le moteur resvg sous-jacent est MPL-2.0).
- **Mapping exact des 26 caractères du XGID** (barre en tête vs en queue, et affectation minuscule/majuscule à « bas »/« haut ») reconstruit à partir de sources concordantes (parseurs, exemples de forum) plutôt que d'une phrase unique de spécification officielle XG. **Testez votre parseur avec des positions connues** (barre pleine, pions sortis) pour lever toute ambiguïté.
- **Champs XGID 1 et 9 = exposants de 2**, pas la valeur littérale du cube — erreur classique à éviter dans le parseur.
- **oksvg/rasterx** : ne pas l'utiliser pour un diagramme comportant du texte ; le piège est qu'il **n'émet pas d'erreur**, il ignore simplement les `<text>`.
- **Presse-papier web** : `image/svg+xml` n'est pas un type garanti ; ne comptez pas dessus — copiez du PNG. HTTPS + geste utilisateur + focus fenêtre obligatoires.
- **Presse-papier Go sous Linux** : dépend de X11 (libx11-dev, donc cgo côté Linux — attention à votre exigence « sans cgo », qui ne vaut alors que pour Windows/macOS) ; Wayland via XWayland ; données perdues à la sortie du processus sans gestionnaire de presse-papier.
- **`bakoma-games`** rend le backgammon en polices halftone « unsuitable for PDF » ; le paquet LaTeX **`bg`** a une licence historiquement « unknown » — à clarifier avant redistribution. Pour du LaTeX, l'export `.tex` de GNU Backgammon (GPL) est une alternative plus sûre.
- **two.js** peut rendre en SVG, canvas2d ou WebGL : si vous conservez two.js pour l'écran, faites-le produire du SVG afin de rester cohérent avec le format pivot ; sinon, préférez du SVG inline natif.