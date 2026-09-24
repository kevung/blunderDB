# L'outillage : le shim, Playwright, les scénarios Go, les données

Décisions Q1 et Q9 ([README.md](README.md) § 2). Tout ce qui est décrit ici est **jetable
et vit dans le scratchpad**, sauf les deux exceptions nommées : la PR des `data-testid` et
les tests de rejeu. Rien de ce fichier ne modifie le comportement du produit.

## 1. Le shim : le vrai front sur le vrai `Database`

### 1.1 Pourquoi il suffit

Le front n'appelle jamais `window.go` directement : il importe les modules générés
`frontend/wailsjs/go/database/Database.js` et `frontend/wailsjs/go/gui/App.js`, dont chaque
fonction lit `window['go'][ns][cls][name]` **à l'appel**. C'est ce qui permet au mock e2e
(`frontend/tests/e2e/helpers/wailsMock.js`) de tout remplacer par un `addInitScript`. Le
shim fait la même chose pour le seul espace `database`, en remplaçant les constantes par des
appels HTTP vers un processus Go qui tient un vrai `*database.Database`.

### 1.2 Côté Go (≈ 150 lignes, `scratchpad/uishim/main.go`)

- Ouvre la base : `db := database.NewDatabase(); db.OpenDatabase(path)` — exactement ce que
  fait `main.go` pour le GUI.
- Une route `POST /call/<Méthode>` : le corps est un tableau JSON des arguments, dans
  l'ordre de la signature. Par **réflexion** sur `*database.Database`, le shim convertit
  chaque argument vers le type du paramètre (`json.Unmarshal` dans un `reflect.New(t)`),
  appelle, et renvoie `{"result": …, "error": "…"}`. La convention Wails est la même :
  une méthode `(T, error)` résout `T` ou rejette avec le message de l'erreur ; une méthode
  `error` seule résout `undefined`.
- Une route `GET /methods` qui liste les méthodes exportées, pour vérifier la couverture
  contre `frontend/wailsjs/go/database/Database.d.ts` (260 fonctions ; la Direction en
  utilise une quarantaine, listées en tête de `frontend/src/stores/directionStore.js`).
- CORS ouvert sur l'origine du serveur Vite. Aucune authentification : le shim n'écoute
  que sur `127.0.0.1` et meurt avec la session.
- Le shim sert **une base à la fois** ; on le relance avec une autre `S<n>-<étape>.db`
  (§ 4) entre deux mesures. Pour I5 (coupure) on tue le shim au milieu d'un geste et on le
  relance : le rejeu à l'ouverture est exactement ce que le GUI fait.

Ce que la réflexion ne couvre pas et qu'il faut savoir : les méthodes qui prennent un
`context.Context` ne sont pas liées par Wails non plus (elles sont internes) ; les
`int64` arrivent en nombre JSON, comme sous Wails ; les structures rendues portent les
mêmes balises `json` que celles que Wails sérialise, donc le front les lit à l'identique.

### 1.3 Côté navigateur (`scratchpad/e2e/helpers/shim.js`)

1. `installWailsMock(page, …)` **d'abord**, tel quel : il fournit `window.go.main.Config`
   (langue, thème), `window.go.gui.App` et `window.runtime` (événements, presse-papiers,
   `BrowserOpenURL`), et le front démarre comme dans la suite e2e.
2. Puis un second `addInitScript` remplace `window.go.database.Database` par un `Proxy`
   dont chaque propriété est `(...args) => fetch('http://127.0.0.1:<port>/call/'+name, …)`
   qui résout `result` ou rejette `error`. Le journal `window.__wailsCalls` du mock est
   conservé : il sert à compter les appels par geste.
3. Trois liaisons `gui`/`runtime` à figer pour la Direction : `OpenDirectionOutputDialog`
   rend un dossier du scratchpad (la page murale s'y écrit pour de vrai, par le shim) ;
   `BrowserOpenURL` ne fait rien mais est compté ; `ClipboardSetText` capture le texte
   (c'est ainsi qu'on lit le CSV du classement et de l'annuaire).
4. `LoadPositionsByIDs` et le reste du catalogue du mock ne servent pas : la base est
   réelle, le shim répond.

### 1.4 Le critère d'acceptation du shim

`frontend/tests/e2e/direction-budgets.spec.js` rejoué **contre le shim** sur une base où la
Direction de démo (`internal/gui/demo.db.gz`, décompressée dans le scratchpad) est ouverte,
donne **les mêmes comptes** que contre le mock pour les flux qu'elle couvre. Si un compte
diffère, c'est soit le shim, soit une différence réelle entre le mock et le produit — les
deux cas s'écrivent dans le rapport avant d'aller plus loin.

## 2. Playwright, localement

- Configuration **jetable** dans le scratchpad, copiée de `frontend/playwright.config.js`,
  avec `launchOptions.executablePath: '/usr/bin/chromium'` (les navigateurs Playwright ne
  sont pas installés ici) et `BLUNDERDB_E2E_PORT` sur un port libre : **5173 est squatté**
  par gammonGo, et un port squatté teste l'autre application en silence.
- `testDir` pointe sur le scratchpad ; les helpers du dépôt (`gestureCount.js`,
  `wailsMock.js`, `fixtures.js`) sont importés par chemin absolu, jamais copiés.
- Deux `describe` par spec, un par viewport (§ [mesure.md](mesure.md) § 2), comme
  `eval-panel-no-scroll.spec.js` le fait à 1024×768.
- Le compteur de gestes existant compte clics et touches ; l'exécutant l'**étend dans le
  scratchpad** (pas dans le dépôt) pour compter vues, menus, modales et défilements
  ([mesure.md](mesure.md) § 1). Un défilement synthétique = `page.mouse.wheel(0, 100)` =
  un cran ; les pixels sont lus sur le conteneur avant et après.
- Une spec par scénario, un `test` par opération, dans l'ordre du déroulé ; la spec **ne
  s'arrête pas** au premier écart : elle enregistre l'écart et continue (le budget est un
  constat, décision Q5).

## 3. Les scénarios Go : jouer le tournoi avant de le regarder

### 3.1 Ce qu'ils font

Pour chaque scénario S1-S5, un programme Go (scratchpad, ou test si § 3.3) joue le tournoi
entier sur **une base SQLite réelle**, avec le vrai moteur, et laisse derrière lui les bases
de départ de la mesure (§ 4) et le journal exporté. Deux usages :

- **Couverture fonctionnelle** : chaque incident I1-I6 est joué par l'API, et le programme
  dit si l'API l'accepte, ce que `State.Warnings` contient après, et si
  `blunderdb tournament verify` passe à la fin.
- **Rejeu** : le journal final rejoué donne le même classement (`Ranking()` du paquet
  `direction`, `Standings` de `Database`) et le temps de `Open` est mesuré (`BenchmarkOpen` existe déjà, sur 64 joueurs / 318
  événements ; S5 devrait en produire ~400).

### 3.2 Comment ils tiennent le temps

`Database` horodate tout à `time.Now()` (`db_direction_*.go`) : on ne peut pas jouer un
week-end en dix secondes par cette couche. Les scénarios passent donc **par le paquet
`direction`** sur le `Store` réel : `direction.Create(ctx, db.DirectionStore(), id, cfg,
seed, t0)`, `dir.Enter(ctx, player, when)` pour les inscriptions, puis `dir.ProposeAt(when)`
/ `dir.EventFor(action, when)` / `dir.Apply(ctx, ev)` et `dir.Finish(ctx, when)`, avec un
`when` que le scénario avance (c'est ce que `db_direction_test.go` fait déjà). Les
résultats sont tirés avec `sim.PGain(a, b, n)` du moteur (probabilité de gain par cote) et
les durées avec `sim.Duree(n, minPerPoint, rng)`, pour que les matchs lents et la fin
estimée aient un sens.

Deux conséquences à ne pas oublier :

- La **bande d'horloge** et la fin estimée se lisent au `time.Now()` réel. Le générateur
  prend donc une **ancre** : il décale toute la chronologie du scénario pour que son
  dernier événement tombe quelques minutes avant l'instant de la mesure (« samedi 14 h »
  devient « il y a six minutes »). Sans cela, un tournoi joué « en janvier » afficherait
  huit mois de temps écoulé.
- Les gestes faits ensuite **à l'écran** sont horodatés au vrai `time.Now()`, ce qui est
  cohérent avec l'ancre.

`Config.Breaks` porte des instants absolus : le générateur les décale avec le reste.

### 3.3 En faire un test, ou pas

Un scénario devient un test dans `pkg/blunderdb/direction/` (décision Q7) **si** il tourne
sous une seconde en `-short`, n'écrit que dans un `t.TempDir()`, et vérifie quelque chose
d'énonçable en une phrase (« un journal de 50 joueurs sur deux jours avec les six incidents
se rejoue sans avertissement résiduel et sous 50 ms »). Sinon il reste dans le scratchpad et
le rapport cite ses sorties. Le `-race` sur le paquet `database` est lent, pas bloqué : le
test va dans `direction`, pas dans `database`.

## 4. Les données

- **Noms fictifs uniquement** (#162) : un générateur de 200 noms à partir de deux listes
  (prénoms, patronymes) fixées par une graine ; clubs fictifs (« BC Ourcq », « Cercle du
  Rhône »…) ; cotes tirées avec `sim.Champ(P, 6, 2, 2, 10, rng)` comme l'étude du moteur.
- Une **base de départ** par scénario et par étape de mesure, nommée `S<n>-<étape>.db`
  (`S2-samedi-14h.db`, `S3-samedi-20h-speed.db`, `S4-lundi-3.db`…), produite par § 3,
  jamais committée. Chaque base contient aussi ~300 Players et une centaine de Matchs
  importés (fixtures de `testdata/`) pour que l'autocomplétion, l'annuaire et le
  rattachement (O2, O3, O20) soient mesurés dans un état réel.
- Les journaux exportés et les captures vont dans `scratchpad/rapport-sources/`, cités
  par le rapport ; seules les captures utiles au rapport sont copiées à côté de lui, au
  format PNG, nommées `S<n>-O<m>-<viewport>.png`.

## 5. La PR des `data-testid` (avant tout le reste)

Lire `frontend/src/components/direction/*.svelte` et `TournamentPanel.svelte` en cochant
chaque cible que [mesure.md](mesure.md) § 3 doit atteindre : onglets de la Direction (déjà
`direction-tab-*`), file et boutons (`.proposals .queue li`, `.go`, `.all` : déjà là),
cases de la grille des tables **par numéro**, lignes de la vue Joueurs par identifiant, menu
⋯ et ses entrées, bandeau `LastDecision`, boutons Clore/Rouvrir, champs de Réglages par nom,
liste « ce qui va changer », panneau annuaire. Ajouter ce qui manque, **rien d'autre** ;
`npm run lint`, `npm run format:check`, `npm test`, `npm run test:e2e` verts ; fusionner
avant la première mesure pour que les specs jetables ne dépendent d'aucun sélecteur fragile.
