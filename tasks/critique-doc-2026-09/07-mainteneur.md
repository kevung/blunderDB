# Persona 7 — Le mainteneur

## Qui je suis (3 lignes)

Mainteneur unique de blunderDB, sur mon temps libre, en français, avec huit
catalogues gettext que je ne relis pas tous (ja, fi, ru, el ont été semés par
LLM). Je viens de refondre l'accueil (`792503bca`) : l'historique et l'à-propos
ont leur page, la feuille de route est partie, et la règle tient en une ligne —
la doc décrit la version publiée, au présent, rien d'autre. Ce que je veux
savoir aujourd'hui : ce que chaque page me coûte, ce qui est engendré, et ce qui
sera faux à la prochaine release si je ne fais rien.

## Parcours suivi (fichiers et commandes)

- `CLAUDE.md` (sections Documentation, Release Process, Invariants), `doc/README.txt`,
  `doc/source/conf.py`, `doc/build.py`, `doc/site/index.html`, `doc/Makefile`.
- Volumes : `wc -l doc/source/*.rst` ; `for f in doc/source/locale/en/LC_MESSAGES/*.po; do grep -c '^msgid "' $f; done` ;
  `ls doc/source/locale/*/LC_MESSAGES/*.po | wc -l` ; `cat doc/source/locale/*/LC_MESSAGES/*.po | wc -l`.
- Churn : `for f in doc/source/*.rst; do git log --oneline 0.30.0..HEAD -- $f | wc -l; done` ;
  la même boucle par intervalle de tags (`0.33.0..0.34.0`, … `0.36.0..HEAD`).
- Coût release : `git diff <tag> <tag> --numstat -- doc/source/locale` ;
  `git diff <tag> <tag> -- doc/source/locale/en | grep -c '^+msgid "'`.
- Refonte : `git show --stat 792503bca | tail` ; `git show --numstat --format= 792503bca -- doc/source/locale`.
- Engendré : `cmd/openapi-gen/main.go`, `internal/server/openapigen/openapigen_test.go`,
  `cmd/help-gen/`, `cmd/cli-doc-gen/main.go`, `Makefile` (cibles `help`, `screenshots`).
- Dérive : `sed -n '103,140p' internal/cli/cli.go` contre `grep -oE 'blunderdb <cmd>' doc/source/cli.rst` ;
  `grep -oE '/v1/[a-zA-Z]+\.[a-zA-Z]+' doc/source/{mode_headless,api_reference}.rst openapi.yaml | sort -u` ;
  `git log -1 --format=%ad --date=short -- doc/source/img/*` ;
  `grep -nE "2\.1[0-9]\.[0-9]|0\.3[0-9]\.[0-9]" doc/source/*.rst`.
- Garde-fous : `.github/workflows/build.yml` (jobs `docs-changes`, `docs-i18n-check`, `docs`, `pages`),
  `scripts/doc-po-update.sh`, `scripts/doc-i18n-check.sh`, `scripts/release.sh`,
  `.claude/skills/release-blunderdb/SKILL.md`, `frontend/src/__tests__/helpVocabulary.sync.test.js`.
- Un script Python jeté pour les `:ref:`/`:doc:` cassés et le décompte des liens externes.

## Ce que j'ai trouvé en cinq minutes

1. **La page produit publiée en ce moment a probablement trois boutons morts.**
   Dans `build.yml`, l'étape `Set outputs` du job `docs` pose
   `version=$(git rev-parse --short HEAD)` hors tag, et l'étape *Render the home
   page* fait `sed "s/__VERSION__/${VERSION}/g" doc/site/index.html`. Les trois
   href de `doc/site/index.html:131-133` deviennent alors
   `…/releases/latest/download/blunderDB-windows-<sha>.exe`. Le job `pages` déploie
   ce rendu à chaque poussée sur `main` qui touche `doc/**` — c'est-à-dire
   `792503bca` lui-même.
2. **Deux « depuis la 0.37.0 » dans une doc dont la version publiée est 0.36.0**
   (`conf.py:release = '0.36.0'`, dernier tag `0.36.0`) : `mode_headless.rst:277`
   et `manuel.rst:1320`. Le grep de factualité de la skill ne cherche que des
   tournures, pas des numéros de version : il rend **une** ligne, un libellé Anki.
3. **`mode_headless.rst` se contredit sur deux routes.** Les seules routes `/v1/`
   de la page absentes de l'annexe engendrée sont `/v1/maintenance.vacuum` et
   `/v1/tenant.purge` (lignes 678 et 689) — or les lignes 233-244 de la même page
   disent qu'elles sont passées sous `/ops/`. L'annexe engendrée est l'arbitre.
4. **35 lignes de français coûtent 5 613 lignes de catalogue.** `bd1d0d992`
   ajoute 35 lignes à `manuel.rst` et `+3152 −2461` réparties sur 8 `.po`.

## Coût par page (18 pages)

`lignes` = `wc -l` ; `msgid` = entrées du catalogue anglais (en-tête déduit) ;
`msgstr` = msgid × 8 langues ; `commits` = depuis `0.30.0`.

| Page | lignes | msgid | msgstr | commits | Nature | Mise à jour à une release | Dérive |
|---|---|---|---|---|---|---|---|
| `manuel.rst` | 1700 | 396 | 3168 | 72 | manuel | personne (moi, à la main) | **élevé** — décrit 10 panneaux ; 21 commits sur la seule 0.36.0 |
| `cli.rst` | 1169 | 275 | 2200 | 19 | manuel | personne | **élevé** — 22 sous-commandes sur 23 ; aucun garde-fou vers `internal/cli` |
| `mode_headless.rst` | 837 | 163 | 1304 | 32 | manuel | personne | **élevé** — tags Docker figés, routes contredites par l'annexe engendrée |
| `guide_utilisateur.rst` | 720 | 209 | 1672 | 9 | manuel | personne | **élevé** — 4 tutoriels pas à pas + 13 captures + un tag Docker |
| `historique.rst` | 342 | 223 | 1784 | 1 | manuel | `release.sh --changelog` **et** la skill (phase 2) | faible — figé sauf la section du jour |
| `faq.rst` | 293 | 69 | 552 | 6 | manuel | personne | moyen — répond « pourquoi » plus que « comment » |
| `stats_parity.rst` | 275 | 99 | 792 | 4 | manuel | personne | moyen — seuils chiffrés + lignes du source gnuBG |
| `raccourcis.rst` | 264 | 159 | 1272 | 14 | manuel, **source de l'aide intégrée** | `make help` + `go test ./cmd/help-gen` | moyen — le rendu est verrouillé, le **contenu** ne l'est pas |
| `telecharge_install.rst` | 243 | 70 | 560 | 7 | manuel (URL par substitution `conf.py`) | `release.sh` (indirect, via `release`) | faible — les liens suivent la version |
| `cmd_mode.rst` | 193 | 258 | 2064 | 9 | manuel, **source de l'aide intégrée** | `make help` + `help-gen` + `helpVocabulary.sync.test.js` | **faible** — seule page verrouillée dans les deux sens |
| `glossaire.rst` | 191 | 65 | 520 | 1 | manuel | personne | faible — vocabulaire, pas de version |
| `api_reference.rst` | 183 | 4 | 32 | 3 | **engendrée** (`go run ./cmd/openapi-gen`) | `TestGeneratedFilesAreUpToDate` | **faible** — 132 routes, 4 chaînes traduisibles |
| `annexe_windows_securite.rst` | 141 | 34 | 272 | 0 | manuel + 10 captures | personne | moyen — captures du 2025-01-22, dérive côté Windows, invisible ici |
| `a_propos.rst` | 111 | 30 | 240 | 1 | manuel | personne | faible — seule page autorisée à pointer vers le futur (jalons, Discussions) |
| `annexe_filtres.rst` | 111 | 85 | 680 | 2 | manuel | personne | moyen — délègue à `cmd_mode` mais redit `ss` et la bibliothèque |
| `annexe_db_scheme.rst` | 83 | 21 | 168 | 5 | manuel | personne | moyen — recopie `DatabaseVersion` (2.18.0) à la main |
| `index.rst` | 47 | 6 | 48 | 15 | manuel (sommaire depuis `792503bca`) | personne | faible — trois toctree, plus de contenu |
| `annexe_mac_securite.rst` | 35 | 13 | 104 | 0 | manuel | personne | faible |

*(19ᵉ fichier : `stats.rst`, 4 lignes, `:orphan:`, 1 msgid — un moignon de
redirection qui traîne encore 9 catalogues, 224 lignes au total.)*

**Total** : 6 942 lignes de source, **2 181 msgid**, **17 448 msgstr**,
**176 catalogues**, **138 272 lignes de `.po`**. Vingt lignes de catalogue
entretenues pour chaque ligne de français.

## Coût d'une release et de la refonte (chiffres mesurés)

`git diff <a> <b> --numstat -- doc/source/locale` et
`git diff <a> <b> -- doc/source/locale/en | grep -c '^+msgid "'` :

| Intervalle | `.rst` | msgid neufs (en) | Catalogues touchés | Lignes de `.po` ajoutées |
|---|---|---|---|---|
| 0.33.0 → 0.34.0 | +367 / −76 | 87 | 64 | 33 034 |
| 0.34.0 → 0.35.0 | +209 / −6 | 33 | 41 | 9 429 |
| 0.35.0 → 0.36.0 | +2410 / −369 | 614 | 129 | 67 124 |
| 0.36.0 → HEAD | +727 / −761 | 298 | 111 | 24 591 |

Une release ordinaire, c'est donc **30 à 90 msgid** — soit **240 à 720 msgstr à
écrire** — et 10 000 à 33 000 lignes de diff dans les catalogues. La 0.36.0
(glossaire + annexe d'API + refonte des tutoriels) est le cas haut : 614 msgid,
**4 912 msgstr**.

**Coût marginal d'une page** (`git show --numstat --format= 73f0d8c05`, la
création du glossaire) : `glossaire.rst` = **191 lignes → 65 msgid → 8
catalogues de ~515 lignes = 4 097 lignes de `.po`**, plus une entrée dans le
`.pot`, plus 9 rendus HTML et 9 PDF au build. Une page « comparaison avec XG »
de la taille de `stats_parity` (99 msgid) coûterait donc **792 msgstr** et
~6 000 lignes de catalogue, dont **la moitié en ja/fi/ru/el que je ne peux pas
relire**. C'est là le vrai coût : pas les lignes, la confiance.

**La refonte** (`git show --stat 792503bca | tail`) : **115 fichiers,
+17 320 / −36 993**. Côté sources : `a_propos.rst` +111, `historique.rst` +342,
`index.rst` +14/−510, `roadmap.rst` −186. Côté catalogues : **103 fichiers,
+16 696 / −36 156** — dont la purge des entrées obsolètes d'`index.po` et la
suppression de `roadmap.po` en huit langues. Bilan net : **−19 673 lignes**,
et une page de moins qui invitait le futur à chaque release.

## Redites mesurées

| Sujet | Écrit à | Mesure |
|---|---|---|
| Commande `ss` (recherche dans les résultats) | `cmd_mode.rst:81` (table), `manuel.rst:488-491`, `annexe_filtres.rst:41-48` | **3 fois**, avec les mêmes exemples `ss nc` / `ss E>40` dans deux des trois |
| Bibliothèque de filtres | `manuel.rst:543-548`, `annexe_filtres.rst:50-60` | **2 fois**, procédure numérotée des deux côtés |
| Panneaux (collections, tournois, stats, matchs, Eval, recherche, import, export) | `guide_utilisateur.rst` (8 sections « Gérer… / Afficher… ») et `manuel.rst` (8 sections « Panneau … ») | **8 sujets décrits deux fois**, ~209 et ~396 msgid |
| Filtres de recherche | `cmd_mode.rst` (table, source unique) ; `cli.rst` **délègue** (`:doc:`cmd_mode`` ligne 56 de sa section `search`) ; `annexe_filtres.rst` **délègue** (`:ref:`cmd_filter``) | pas de redite — le renvoi fonctionne, c'est le bon modèle |
| Statistiques | `manuel.rst` §Panneau Stats et `stats_parity.rst` | pas de redite : l'un décrit l'écran, l'autre le modèle. `stats.rst` est le vestige de l'ancienne troisième copie |
| Pitch produit | `index.rst` (6 msgid, français source, traduit) et `doc/site/index.html` (168 lignes, **anglais seul, hors Sphinx, hors gettext**) | 2 fois, dans deux régimes différents — la page produit est une neuvième langue non gérée |

Aucun bloc d'exemple n'est dupliqué à l'identique entre deux pages (vérifié par
extraction et intersection des lignes indentées) : les redites sont des
reformulations, donc des divergences en puissance, pas des copies.

## Ce qui dérivera à la prochaine release

| Objet | État mesuré | Qui le met à jour | Verdict |
|---|---|---|---|
| Version de l'app dans les liens de téléchargement | substitutions `\|latest_*\|` engendrées par `conf.py` | `release.sh` (une seule ligne, `release = …`) | **sûr** |
| Version dans la page produit | `__VERSION__` × 4 dans `doc/site/index.html`, substitué par `build.yml` | le workflow — mais avec un SHA hors tag | **cassé** (constat 1) |
| Tags Docker | `guide_utilisateur.rst:169` → `0.35.0` ; `mode_headless.rst:561,568,573,578` → `0.34.0` | personne | **déjà dérivé, deux valeurs différentes** |
| `DatabaseVersion` | `annexe_db_scheme.rst:14` et `cli.rst:910` disent 2.18.0 (juste) ; `cli.rst:452`, `stats_parity.rst:112`, `manuel.rst:977` citent 2.15.0 comme borne historique (juste, mais recopié) | personne | **moyen** — un bump de schéma ne touche rien |
| Liste des sous-commandes CLI | 23 dans `handlers()` (`internal/cli/cli.go:103-137`), **22** dans `cli.rst` : `version` manque | personne ; `cli-doc-gen` n'écrit que `CLI_USAGE.md` et n'a **aucun test** | **déjà dérivé** |
| Routes du serveur | 132 dans `openapi.yaml` = 132 dans `api_reference.rst` ; `mode_headless.rst` en cite 7, dont 2 sous un préfixe périmé | `openapi-gen` + `TestGeneratedFilesAreUpToDate` pour l'annexe ; **personne** pour la prose | **annexe sûre, prose dérivée** |
| Raccourcis | `raccourcis.rst` → `make help` → `frontend/src/i18n/help/*.js`, verrouillé par `TestHelpBundlesAreCurrent` | le test | **le rendu est sûr, pas le contenu** : rien ne compare `raccourcis.rst` à `keyboardService.js` (`cmd_mode.rst`, lui, a `helpVocabulary.sync.test.js`) |
| Captures d'écran | 13 panneaux régénérés le 2026-09-03 ; `AnkiPanel.svelte` a changé de 135 lignes le 2026-09-06 (`bd1d0d992`) | `make screenshots`, « verified at release time (skill release-blunderdb) » d'après le `Makefile` — or `grep -i screenshot SKILL.md` **ne rend rien** | **déjà dérivé** (`panel_anki.png`) |
| Captures Windows | 10 fichiers du 2025-01-22 | personne | moyen, invérifiable depuis ici |
| Traductions | `docs-i18n-check --strict` sur toute poussée `main` et tout tag ; `pages` ne déploie que si le job est vert | CI | **sûr** — le meilleur garde-fou du lot |

## Factualité : le grep élargi

Le grep de la skill (`SKILL.md` phase 1 étape 3) rend **une** ligne :

| # | Occurrence | Page › ligne | Verdict |
|---|---|---|---|
| 1 | « revoir **bientôt** » | `raccourcis.rst:228` | vocabulaire — libellé de la note Anki *À revoir* |

Élargi à `sera|seront|prochainement|bientôt|pour l'instant|actuellement|en cours|`
`à venir|en préparation|pas encore|prévu|futur|ultérieur|prochaine version|va être|vont être` :
**42 lignes** (`grep -cE … doc/source/*.rst`), résumées par famille — aucune n'est une promesse :

| Famille | Lignes | Pages | Verdict |
|---|---|---|---|
| « actuellement » = *à l'instant présent* (positions filtrées, base ouverte, connexions) | 10 | `cmd_mode:81`, `glossaire:38`, `annexe_filtres:42`, `guide_utilisateur:288,309`, `manuel:489,537,1577`, `mode_headless:447,708` | vocabulaire |
| « en cours » = *en train de tourner* (import, recherche, rattrapage, session shell) | 15 | `cli:1099,1108`, `guide_utilisateur:567`, `historique:56`, `raccourcis:249`, `manuel:145,450,1134`, `mode_headless:292,404,446,450`, … | vocabulaire — dont 2 faux positifs lexicaux, « en **course** » (`manuel:1351`, `historique`) |
| « seront / sera » = conséquence d'une action de l'utilisateur | 5 | `guide_utilisateur:294,297,298`, `faq:76,77` | vocabulaire (futur procédural ; à basculer au présent par confort de traduction, pas par factualité) |
| « prévu / futur » | 6 | `faq:29,51`, `guide_utilisateur:334`, `a_propos:46`, `manuel:1374`, `mode_headless:151` | vocabulaire — `mode_headless:151` est « im**prévu** » ; `a_propos:46` (« soutenir les développements passés et futurs ») est le seul cas limite, et c'est la page à qui la règle laisse le droit de regarder devant |
| « pas encore » = état d'une donnée, pas d'une fonctionnalité | 4 | `cli:591`, `manuel:232,653,1490` | vocabulaire |
| « bientôt » | 1 | `raccourcis:228` | vocabulaire (le seul hit du grep de la skill) |
| « ultérieur » | 1 | `historique:7` | vocabulaire |

**Ce que ni l'un ni l'autre n'attrape** : `manuel.rst:1320` et
`mode_headless.rst:277` écrivent « **depuis la 0.37.0** » alors que la version
publiée est 0.36.0. La règle n'est pas violée par une tournure mais par un
**numéro** ; le motif manquant est une comparaison entre les versions citées
dans les `.rst` et `conf.py:release`.

## Garde-fous présents et manquants

**Présents**

- `docs-i18n-check` en CI (`build.yml:1480`), **strict sur toute poussée** et sur
  tout tag ; `pages` refuse de déployer s'il n'est pas vert. C'est ce qui a
  fermé le #161.
- `docs-changes` (paths-filter) évite 10 minutes et 1 Go de LaTeX sur les PR qui
  ne touchent pas `doc/**` ; contourné sur tag et dispatch.
- `TestGeneratedFilesAreUpToDate` (`internal/server/openapigen/openapigen_test.go:37`)
  verrouille `openapi.yaml` **et** `doc/source/api_reference.rst`.
- `TestHelpBundlesAreCurrent`, `TestEveryLanguageIsFullyTranslated`,
  `TestTranslationIsNeverSilentlyFrench` (`cmd/help-gen/help_gen_test.go`) :
  l'aide intégrée ne peut ni retarder sur la doc ni retomber en français.
- `helpVocabulary.sync.test.js` : `cmd_mode.rst` ⟷ `commandVocabulary.js` dans
  les deux sens. La seule page de doc verrouillée **au contenu**.
- `scripts/doc-po-update.sh` : chemin gettext relatif, `python-format` normalisé,
  jamais de `msgcat`. `doc/README.txt` documente les trois pièges et la méthode
  Babel bloc-par-bloc.
- `release.sh --check` (cible `make release-check`) : `conf.py` / `metaStore.js` /
  `wails.json` d'accord.

**Manquants**

- **Aucun `sphinx-build -b linkcheck`**, nulle part : 49 liens externes distincts
  dans les `.rst`, jamais vérifiés (gnuBG, Flathub, Discord, PayPal, Discussions).
  Les `:ref:`/`:doc:` internes, eux, sont sains (78 étiquettes, 30 références,
  0 cassée — vérifié par script).
- **Aucun garde-fou `cli.rst` ⟷ `internal/cli`** : `cli-doc-gen` n'a pas de test
  et n'écrit que `CLI_USAGE.md`. `version` manque déjà.
- **Aucun garde-fou `raccourcis.rst` ⟷ `keyboardService.js`** — la page a
  159 msgid et 14 commits depuis 0.30.0.
- **Aucune vérification des numéros de version cités dans les `.rst`** contre
  `conf.py:release` et `domain.DatabaseVersion` (deux « 0.37.0 » et cinq
  recopies de version de schéma).
- **Aucune vérification de fraîcheur des captures** : `make screenshots` écrit
  des fichiers suivis, le `Makefile` dit qu'il est vérifié à la release, la skill
  ne le mentionne pas.
- **La page produit est hors de tous les filets** : anglais seul, hors gettext,
  hors `docs-i18n-check`, et son unique variable est substituée par un SHA hors tag.
- **`SKILL.md` est en dérive sur son propre outillage** : phase 1 étape 4 décrit
  `cd doc && make gettext ; sphinx-intl update -l fr -l en …`, exactement ce que
  `CLAUDE.md` et `doc/README.txt` interdisent, alors que la phase 2 étape 3
  utilise le bon `scripts/doc-po-update.sh`.

## Constats

| # | Constat | Fichier › section | Gravité | Proposition | Effets de bord |
|---|---|---|---|---|---|
| 1 | Hors tag, `version` = SHA court ; les trois boutons de téléchargement de la page produit deviennent `…/blunderDB-windows-<sha>.exe` (404), et `pages` déploie ce rendu à chaque merge touchant `doc/**` | `.github/workflows/build.yml` › job `docs`, étapes *Set outputs* et *Render the home page* ; `doc/site/index.html:131-133` | **bloquant** | Rendre la page produit avec le **dernier tag** dans tous les cas (`git describe --tags --abbrev=0`), en gardant le SHA pour le seul nommage d'artefacts ; exige `fetch-depth: 0` (ou `fetch-tags`) sur le checkout du job `docs` | page produit ; aucun `.po`, aucun PDF, aucune aide intégrée |
| 2 | « depuis la 0.37.0 » dans une doc dont la version publiée est 0.36.0 | `mode_headless.rst:277` ; `manuel.rst:1320` | **bloquant** | Retirer les deux mentions (la doc décrit la version publiée : la phrase se dit au présent sans numéro), puis ajouter au filet un contrôle qui refuse tout numéro de version supérieur à `conf.py:release` dans les `.rst` | 2 × 8 `.po` à reprendre ; PDF ; pas d'aide intégrée |
| 3 | La même page dit `/ops/maintenance.vacuum` (l. 233-244) et `POST /v1/maintenance.vacuum` (l. 689) ; idem pour `tenant.purge` (l. 238 vs 678). Ce sont les deux seules routes de la page absentes de l'annexe engendrée | `mode_headless.rst` › *Opérations privilégiées* vs *Décommissionner un tenant* | **bloquant** | Corriger les deux occurrences tardives ; ajouter au filet une extraction `grep -oE '/v1/[a-z]+\.[a-z]+'` de `mode_headless.rst` comparée à `api_reference.rst` | 2 × 8 `.po` ; PDF |
| 4 | Tags Docker figés à deux versions périmées et différentes | `guide_utilisateur.rst:169` (0.35.0) ; `mode_headless.rst:561,568,573,578` (0.34.0) | gênant | Remplacer par `ghcr.io/kevung/blunderdb-serve:latest` dans les exemples et ne garder le numéro que dans la phrase qui explique le choix `latest` / version figée | 5 chaînes × 8 `.po` ; PDF |
| 5 | `blunderdb version` n'est pas documenté (23 sous-commandes dans `handlers()`, 22 dans la page) | `cli.rst` › *Commandes disponibles* | gênant | Ajouter la section ; puis un test Go qui compare `cli.CommandNames()` aux titres `<cmd> — …` de `cli.rst`, sur le modèle de `helpVocabulary.sync.test.js` | 1 section × 8 `.po` ; PDF ; pas d'aide intégrée |
| 6 | `DatabaseVersion` et ses bornes historiques recopiés à la main en cinq points | `annexe_db_scheme.rst:14`, `cli.rst:452,910`, `stats_parity.rst:112`, `manuel.rst:977` | gênant | Étendre `release-check` (ou un test Go) à un contrôle de cohérence entre `domain.DatabaseVersion` et l'unique occurrence « courante » (`annexe_db_scheme.rst`) ; les quatre bornes historiques restent, elles sont datées par nature | aucun tant que rien ne bouge ; à un bump : 1 chaîne × 8 `.po` |
| 7 | `panel_anki.png` est périmée : captures du 2026-09-03, `AnkiPanel.svelte` +135 lignes le 2026-09-06 ; et la vérification annoncée par le `Makefile` n'existe pas dans la skill | `doc/source/img/panel_anki.png` ; `Makefile` › cible `screenshots` ; `SKILL.md` | gênant | Ajouter une phase « captures » à la skill (`make screenshots`, revue du diff) et faire dire au `Makefile` la vérité si la phase n'est pas ajoutée | images du site et des 9 PDF ; skill ; aucun `.po` |
| 8 | La skill contredit `CLAUDE.md` et `doc/README.txt` sur la régénération des catalogues | `SKILL.md` › phase 1, étape 4 | gênant | Remplacer le bloc `make gettext` / `sphinx-intl update` par `scripts/doc-po-update.sh` (déjà correct en phase 2 étape 3) | skill seule |
| 9 | La skill demande encore de documenter les nouveautés dans `stats.rst`, moignon `:orphan:` de 4 lignes | `SKILL.md` › phase 1, étape 2 | gênant | Remplacer par `stats_parity.rst` et la section *Panneau Stats* de `manuel.rst` | skill seule |
| 10 | Aucun `linkcheck` : 49 liens externes jamais vérifiés | `.github/workflows/build.yml` › jobs `docs` / `nightly.yml` | gênant | Ajouter `sphinx-build -b linkcheck doc/source doc/build/linkcheck` **dans `nightly.yml`** (pas sur chaque PR : réseau lent et faux positifs), en avertissement, pas en échec | aucun sur le contenu ; un job de plus dans le nocturne |
| 11 | `raccourcis.rst` est la seule source des raccourcis et n'a aucun garde-fou vers le code | `raccourcis.rst` ; `frontend/src/services/keyboardService.js` | gênant | Étendre `helpVocabulary.sync.test.js` (ou un frère) aux touches déclarées dans `keyboardService.js`/`tabToggles.js` | test seul ; révélera peut-être des lignes à corriger, donc `.po` + aide intégrée |
| 12 | La commande `ss` est décrite trois fois, avec les mêmes exemples dans deux d'entre elles | `cmd_mode.rst:81` ; `manuel.rst:488-491` ; `annexe_filtres.rst:41-48` | gênant | Garder la table de `cmd_mode` (verrouillée par le test) et la phrase du manuel ; `annexe_filtres` renvoie, comme elle le fait déjà pour la liste des filtres | −1 chaîne × 8 `.po` ; aide intégrée inchangée (`cmd_mode` n'est pas touché) |
| 13 | `annexe_filtres.rst` : 111 lignes, 85 msgid, 680 msgstr entretenus pour une page qui délègue l'essentiel et redit le reste | `annexe_filtres.rst` | gênant | Fusionner *Recherche en ligne de commande* et *Recherche dans les résultats courants* dans `manuel.rst` › *Panneau Recherche*, ne garder l'annexe que pour la bibliothèque de filtres et les exemples — ou la supprimer et déplacer les exemples | suppression de catalogue en 8 langues (comme `roadmap`) ; toctree d'`index.rst` ; l'unique renvoi entrant (`manuel.rst:550`) à rediriger ; PDF |
| 14 | Huit panneaux décrits deux fois (guide pas-à-pas / manuel de référence) | `guide_utilisateur.rst` (8 sections) ; `manuel.rst` (8 sections *Panneau …*) | gênant | Ne pas fusionner (les deux registres sont légitimes) mais imposer une règle : le guide **montre une tâche**, le manuel **décrit un écran** ; tout écran décrit dans le guide devient un renvoi `:ref:` | à la refonte : plusieurs dizaines de chaînes × 8 ; PDF |
| 15 | La page produit est une neuvième langue non gérée : anglais seul, hors gettext, hors `docs-i18n-check` | `doc/site/index.html` | mineur | L'assumer par écrit (un commentaire en tête du fichier le dit déjà à moitié) et n'y mettre **que** ce qui ne se traduit pas : capture, boutons, liens. Ne pas l'internationaliser : ce serait une neuvième source à tenir | aucun |
| 16 | `stats.rst` : moignon `:orphan:` de 4 lignes, 9 catalogues, 224 lignes | `doc/source/stats.rst` et ses `.po` | mineur | Supprimer le fichier et ses neuf catalogues (le `:ref:`stats`` du manuel existe déjà) ; vérifier qu'aucune URL publiée `…/stats.html` ne perd un lecteur, sinon garder la page une release de plus | suppression de 9 catalogues ; une URL du site disparaît |
| 17 | Numéros de ligne du source gnuBG cités dans une annexe | `stats_parity.rst:275` (`gnubg/eval.c:5088–5100`) | mineur | Citer le nom du prédicat (`isCloseCubedecision`) et la version de gnuBG lue, pas les lignes | 1 chaîne × 8 `.po` |
| 18 | Cinq futurs procéduraux (« les positions … seront mises à jour ») là où le présent dirait la même chose | `guide_utilisateur.rst:294,297,298` ; `faq.rst:76,77` | mineur | Basculer au présent — pas au nom de la factualité (ce n'est pas une promesse) mais parce que le présent se traduit mieux et aligne le ton du reste | 5 chaînes × 8 `.po` |
| 19 | `historique.rst` grossit de ~6 puces × 9 langues par release ; 223 msgid déjà figés | `historique.rst` | mineur | Ne rien changer maintenant : la puce unitaire est le bon grain (une correction n'invalide qu'une chaîne). Réexaminer au-delà de ~500 lignes, en archivant les versions < 0.10 dans une section repliée | aucun ; à l'archivage : `.po` + PDF |
| 20 | Le budget global est de 20 lignes de catalogue par ligne de français (138 272 `.po` pour 6 942 `.rst`), dont la moitié dans quatre langues que je ne relis pas | `doc/source/locale/` | mineur | En faire un critère explicite avant d'accepter une page : toute page nouvelle est chiffrée (msgid × 8) dans la discussion qui la propose, comme le glossaire l'a été a posteriori (65 msgid → 4 097 lignes) | aucun ; convention de projet |
