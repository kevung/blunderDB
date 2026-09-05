# Synthèse — changements proposés, classés par impact puis par coût

Sept personas, 144 constats. Cette page les regroupe par thème, retient ceux
qui reviennent chez plusieurs lecteurs ou qui bloquent un parcours, et les
classe en cinq lots. Chaque ligne nomme les fichiers touchés et les effets de
bord : `.po` (× 8 langues), PDF, aide intégrée (`make help`), skill de
release, page produit (`doc/site/index.html`, hors Sphinx). Rien n'est
exécuté ici. Les numéros renvoient aux tableaux des fiches (`P1 #3` = persona
1, constat 3).

Légende coût : **T** = technique, sans chaîne à traduire ; **1L** = une seule
langue ; **9L-n** = n chaînes françaises à retraduire en huit langues ;
**page** = une page nouvelle ou supprimée (toctree, neuf catalogues, PDF).

## Lecture croisée : ce que plusieurs personas ont heurté

| Thème | Personas | Verdict |
|---|---|---|
| La page produit est en anglais seul, sans accès aux neuf langues | P1 #1, P5 #1, P6 #20 contre P7 #15 | **Contradiction à trancher** (lot 5) |
| Les tables de bearoff « téléchargées » survivent dans cinq endroits alors que 0.36 les calcule | P1 #16 (FAQ), P2 #7 (crédits), P6 #2 (aide intégrée), P6 #4 (manuel Méthodologie), P6 #11 (historique 0.32) | Fait, lot 1 |
| « depuis la 0.37.0 » dans une doc 0.36.0 | P4 #17, P7 #2 | Fait, lot 1 ; garde-fou lot 4 |
| `/v1/tenant.purge` et `/v1/maintenance.vacuum` restés après le passage sous `/ops/` | P4 #1, P7 #3 | Fait, lot 1 ; garde-fou lot 4 |
| Commentaires des blocs de code en français dans toutes les langues (en 49 lignes, de 56, ja 54) | P2 #15, P5 #5 | Lot 2 |
| EMG jamais défini, PR nommé de deux façons, deux échelles « millipoints » (× 500 et × 1000) | P1 #10 #17, P2 #5 #9 #20, P3 #5 | Lot 2 |
| `SHIFT` contre `MAJ` selon la page | P5 #6, P6 #9 #19 | Lot 1 |
| Balisage RST rendu littéralement (`` `` `` fr ×4, ja ×29 ; `–jobs` demi-cadratin fr/en/de) | P4 #19, P5 #4 #20 | Lot 1 (fr) et lot 3 (ja) |
| La grammaire de recherche est répartie sur quatre pages, `ss` décrite trois fois | P3 #12 #13, P7 #12 #13 | Lot 5 |
| Le manuel ne mentionne jamais le mode serveur ; le poste de travail n'ouvre que des fichiers | P4 #14 #15 | Lot 2 |
| L'aide intégrée n'a aucun lien vers le site ni l'historique, et son onglet Manuel est un résumé indépendant qui a pris du retard | P6 #1 #2 #5 #7 #8 | Lot 1 (prose) ; historique 0.36 à reformuler |

## Lot 1 — Faits faux ou contradictions internes (bloquant, coût faible)

À faire en une seule branche, avant tout le reste : ce sont des erreurs, pas
des choix.

| # | Changement | Fichiers | Coût | Effets de bord | Source |
|---|---|---|---|---|---|
| 1.1 | Les trois boutons de téléchargement de la page produit pointent sur `…-<sha>.exe` hors tag : rendre la page avec le dernier tag (`git describe --tags --abbrev=0`, `fetch-depth: 0` sur le checkout du job `docs`) | `.github/workflows/build.yml` (job `docs`, étapes *Set outputs* / *Render the home page*) | T | page produit | P7 #1 |
| 1.2 | Retirer les deux « depuis la 0.37.0 » : la phrase se dit au présent, sans numéro | `mode_headless.rst:277`, `manuel.rst:1320` | 9L-2 | PDF | P4 #17, P7 #2 |
| 1.3 | Corriger `POST /v1/tenant.purge` et `POST /v1/maintenance.vacuum` en `/ops/` ; écrire en toutes lettres la règle nginx et Caddy de refus de `/ops/` | `mode_headless.rst:678,689` et section *Les routes d'exploitation* | 9L-3 | PDF | P4 #1, P7 #3 |
| 1.4 | Les exemples `serve --addr :8080` et `docker run -p 8080:8080` écoutent sur toutes les interfaces alors que l'encart affirme le contraire : passer à `127.0.0.1:8080` | `mode_headless.rst` (*Le démon serve*, *Déploiement avec Docker*) | 9L-2 | PDF | P4 #5 #6 |
| 1.5 | Bearoff : la FAQ (« base téléchargeable jusqu'à 11 »), les crédits (« générées avec GNU Backgammon »), la Méthodologie du manuel (« TS-06-11 téléchargée ») et la liste des onglets de configuration (« panneau Bearoff ») disent l'ancien monde | `faq.rst`, `a_propos.rst` (Crédits), `manuel.rst` (Méthodologie, Configuration) | 9L-4 | PDF | P1 #16, P2 #7, P6 #3 #4 |
| 1.6 | Aide intégrée : réécrire la prose Eval (plus de téléchargement), ajouter les tournois auto-remplis, aligner Anki 3 sur « Bien », ajouter sous le numéro de version deux liens (documentation, historique) | `frontend/src/i18n/help/prose/*` puis `make help` | 9L-4 (les fragments prose ont leurs propres catalogues) | aide intégrée | P6 #1 #2 #6 #7 #8 |
| 1.7 | L'annexe statistique dit « blunderDB never computes luck: it has no evaluation engine » face à la page produit : « ne *recalcule* pas la chance, l'évaluateur intégré ne sert pas à cela » | `stats_parity.rst` (encart Chance) | 9L-1 | PDF | P2 #1 |
| 1.8 | Le manuel définit le PR comme « qualité de jeu money … indépendant du score », l'annexe le contredit (EMG cubeful, MET) : retirer les deux mentions, renvoyer à l'annexe | `manuel.rst` (Panneau Stats, bascule PR/MWC) | 9L-1 | PDF | P2 #6 |
| 1.9 | Page CLI : « n'importer aucun élément est toujours une erreur » contre « relancer sur un dossier déjà importé reste un succès » : une table à trois lignes des codes de retour d'`import` | `cli.rst` (*import*) | 9L-2 | PDF | P3 #1 |
| 1.10 | Page CLI : l'exemple `--query 's cube p>30 E>50'` est invalidé huit lignes plus bas (jetons de plateau, plateau vide) et la traduction recommandée en drapeaux est interdite par « les combiner est refusé » : remplacer l'exemple, montrer la version tout-drapeaux | `cli.rst` (*search › Le langage de requête*) | 9L-3 | PDF | P3 #2 #3 |
| 1.11 | Suppression d'une position : le guide dit « aucune confirmation », les deux pages de référence « confirmation demandée » : aligner sur le comportement réel | `guide_utilisateur.rst` ou `raccourcis.rst` + `cmd_mode.rst` | 9L-1 | aide intégrée si `raccourcis`/`cmd_mode` bougent | P1 #8 |
| 1.12 | `MAJ` contre `SHIFT` : une seule notation en français (`MAJ`), l'historique 0.36 et le tableau *Base de données* des raccourcis sont les intrus ; en japonais, le manuel dit `MAJ-J`/`ENTREE` en français | `historique.rst`, `raccourcis.rst`, `ja/manuel.po` | 9L-2 + 1L | aide intégrée | P5 #6, P6 #9 #19 |
| 1.13 | Balisage rendu littéralement : « ``/ops/`` » (headless ×1, CLI ×3), `--jobs`/`--match` en demi-cadratin (smartquotes) : mettre en littéral de code | `mode_headless.rst`, `cli.rst`, `manuel.rst:1353` | 9L-5 | PDF | P4 #19, P5 #20 |
| 1.14 | Table des filtres : guillemets typographiques dépareillés (`t’mot1;mot2;…”`) non copiables : cellules en littéral, guillemets droits | `cmd_mode.rst` (*Filtres de recherche*) | 9L-3 | aide intégrée, `helpVocabulary.sync.test.js` | P3 #6 |
| 1.15 | Légende fausse : `s t"Aachen2024"` légendé « position du tournoi » alors que `t` lit les commentaires (`tn1` pour un tournoi) | `annexe_filtres.rst` (Exemples) | 9L-1 | PDF | P3 #11 |
| 1.16 | Le guide dit « GAUCHE/DROITE (ou j/k) », les raccourcis disent `k`/`j` | `guide_utilisateur.rst` (Mon premier import) | 9L-1 | — | P1 #20 |
| 1.17 | La page japonaise d'installation sert le PDF anglais | `ja/telecharge_install.po` (une msgstr) | 1L | — | P5 #3 |
| 1.18 | La FAQ dit du mode serveur qu'il sert « à consulter ou importer des matchs depuis un navigateur » : il n'y a pas d'interface web | `faq.rst` | 9L-1 | — | P4 #16 |

## Lot 2 — Ce qui manque à un parcours (gênant, contenu à écrire, 9 langues)

| # | Changement | Fichiers | Coût | Effets de bord | Source |
|---|---|---|---|---|---|
| 2.1 | Windows : une phrase sous le lien `.exe` (« Windows affichera un avertissement au premier lancement, voici pourquoi ») ; dans l'annexe, dire que « Exécuter quand même » suffit presque toujours et présenter l'exclusion Defender comme dernier recours ; Windows et macOS en tête du tableau « Quel fichier choisir ? » | `telecharge_install.rst`, `annexe_windows_securite.rst` | 9L-6 | PDF | P1 #2 #5 #11 |
| 2.2 | « Mon premier import » : dire que le `.xg` doit avoir été analysé, ce qui se passe sinon, où activer l'analyse automatique ; donner le geste souris (Stats › Top blunders) avant la commande `bl` | `guide_utilisateur.rst` | 9L-4 | — | P1 #3 #4 |
| 2.3 | Glossaire : entrée **EMG** (alias d'équité normalisée) ; une seule dénomination de PR ; nommer distinctement les deux échelles (mpt × 500 du PR, millièmes × 1000 des filtres) et le rappeler dans la table des filtres et sur chaque drapeau CLI d'erreur ; lier chaque sigle vers le glossaire à sa première apparition dans le guide | `glossaire.rst`, `faq.rst`, `stats_parity.rst`, `cmd_mode.rst`, `cli.rst`, `guide_utilisateur.rst` | 9L-12 | aide intégrée, PDF | P1 #9 #10 #17, P2 #5 #9 #20, P3 #5 |
| 2.4 | FAQ pour l'utilisateur XG : « j'ai déjà XG, à quoi sert blunderDB ? » (trois différences), « les rollouts XG sont-ils importés, blunderDB en fait-il ? », « ce qui a été mesuré du moteur et ce qui ne l'a pas été » (course money seulement, verdict au score non mesuré) ; règle d'import « une analyse ne remplit qu'un trou » visible depuis le guide et non seulement sous Configuration | `faq.rst`, `manuel.rst` (Méthodologie), `guide_utilisateur.rst` | 9L-8 | PDF | P1 #13, P2 #2 #3 #4 #13 |
| 2.5 | Commentaires des blocs de code : les faire entrer dans le périmètre traduit, ou les réduire à un mot-clé neutre et des métavariables ASCII (`<path>`) | `cli.rst`, `mode_headless.rst` (tous les `code-block`) | 9L-50 environ, ou T si neutralisés | PDF | P2 #15, P5 #5 |
| 2.6 | CLI : sorties attendues pour `search` (table, json, xgid), `import`, `list`, `info`, `collection`, `verify` (et quel champ tester) ; `--offset` et `--query-help` dans la liste d'options ; `repair`, `bearoff`, `version`, `serve`, `migrate`, `call` dans le tableau des commandes ; forme `--option=false` énoncée une fois ; format d'`import --type position` montré ; note sur l'accès concurrent app ouverte / cron | `cli.rst` | 9L-25 | PDF | P3 #4 #7 #8 #14 #15 #17 #18 #19, P4 #18, P7 #5 |
| 2.7 | Serveur : reproduire `docker-compose.yml` et `Caddyfile` dans la page (avec liens permanents) et une variante qui tire l'image publiée ; compléter le bloc nginx du tutoriel (TLS ou `listen 80` annoncé) ; une étape « plusieurs membres » (`map $remote_user $tenant_id`, effacement de l'en-tête client, refus d'un utilisateur non mappé) ; `--data-dir` au tableau des options et le volume à monter ; section « Mettre à jour un déploiement » (sauvegarde, étiquette de version, `/readyz`, migration sans retour) ; sauvegarde du volume SQLite (WAL) ; `sslmode=disable` annoté ; `/metrics` et `--pprof-addr` dans la liste de ce qu'on n'expose pas ; cycle de vie d'un tenant ; rôle PostgreSQL sans `BYPASSRLS` | `mode_headless.rst`, `guide_utilisateur.rst` (tutoriel serveur) | 9L-30 | PDF | P4 #2 #3 #4 #7 #8 #9 #10 #11 #12 #13 |
| 2.8 | « Le poste de travail et le serveur » : l'application de bureau ouvre des fichiers, pas des URL ; les deux gestes d'aller-retour (`exports.sqlite`, `migrate`) ; pas de lecture inter-tenant, ce qu'on fait à la place (coach) ; un renvoi depuis le manuel | `mode_headless.rst`, `manuel.rst` | 9L-5 | PDF | P4 #14 #15 |
| 2.9 | Libellés d'interface : citer les libellés français (et japonais) de l'interface, l'anglais entre parenthèses la première fois ; captures SmartScreen en anglais signalées comme telles | `guide_utilisateur.rst`, `manuel.rst`, `faq.rst` ; `ja/*.po` | 9L-10 + 1L | — | P1 #7 #14, P5 #13 |
| 2.10 | Historique : introduction pour le lecteur (versions récentes en tête, renvois en fin de version) au lieu de la note de maintenance ; mention « (remplacé en 0.36.0) » sur les puces périmées ; renvoi de chaque puce de schéma vers l'annexe et sa migration sans retour ; reformuler la puce 0.36 sur l'aide intégrée (seuls raccourcis et ligne de commande sont engendrés) | `historique.rst` | 9L-6 | PDF | P6 #5 #11 #12 #15 |
| 2.11 | Page « Après une mise à jour » : ce qu'il faut refaire (réimporter pour la chance et les marques, rattrapage d'analyse, migration de schéma). **Contradiction** : P7 #20 demande de chiffrer toute page nouvelle avant de l'accepter (≈ 30 msgid × 8) ; l'alternative est une section en tête de l'historique | nouvelle page ou `historique.rst` | page ou 9L-8 | toctree, PDF | P6 #13 contre P7 #20 |
| 2.12 | Manuel : section « Import : ce qui est écrit, ce qui ne l'est jamais » ; une phrase sur la pagination de la bibliothèque (0.35) ; retirer le deux-points de « :stats » ; ancien nom `epc` signalé ; la fenêtre d'aide au clavier dans les raccourcis | `manuel.rst`, `cmd_mode.rst`, `raccourcis.rst` | 9L-8 | aide intégrée | P2 #13, P3 #20, P6 #16 #17 #18 |
| 2.13 | Chiffres étayés : version de gnubg dont l'empreinte de bearoff est enregistrée ; échantillon de la parité XG (combien de décisions, quelle version, « typical gap » défini) ; origine du barème de PR ; domaine de la MET (jusqu'à N-away) ; une phrase unique sur quelle MET sert à quoi | `manuel.rst`, `stats_parity.rst`, `faq.rst`, `cmd_mode.rst` | 9L-6 | aide intégrée (si `cmd_mode`), PDF | P2 #8 #11 #12 #18 #19 |

## Lot 3 — Japonais seul (1 langue, aucun `.rst`)

| # | Changement | Fichiers | Coût | Source |
|---|---|---|---|---|
| 3.1 | 29 lignes de balisage cassé aux bornes CJK (`bl`/`blunders` illisible dans le glossaire et le guide) : échappement `\ ` aux deux bornes, puis relecture du rendu | `ja/glossaire.po`, `ja/guide_utilisateur.po`, `ja/manuel.po`, `ja/historique.po`, `ja/cli.po`, `ja/mode_headless.po` | 1L | P5 #4 |
| 3.2 | Locale japonaise du thème incomplète (`Tip` ×15, `Next`, `Previous`, `Search docs`) alors que l'allemand est complet | `doc/source/locale/ja/LC_MESSAGES/sphinx.po` | 1L | P5 #11 #12 |
| 3.3 | Terminologie : `ポジション` partout (400+ contre 85 `局面`) ; `キューブの決定` = décision, `判定` = verdict, comme le glossaire ; un mot pour erreur (`エラー`) et un pour blunder (`ブランダー`) ; terme anglais entre parenthèses après chaque vedette du glossaire | `ja/cmd_mode.po`, `ja/cli.po`, `ja/historique.po`, `ja/manuel.po`, `ja/glossaire.po` | 1L | P5 #8 #9 #10 #17 |
| 3.4 | Ponctuation et espacement : `：` en prose, `「 」` partout, `？`, espace autour des cadratins, puces terminées par `。`, une règle d'espace latin/japonais appliquée (section « XGからインポート » du guide à reprendre) ; métavariables `t'語1;語2'` traduites | `ja/*.po` | 1L | P5 #7 #14 #15 #16 #19 |
| 3.5 | La puce 0.35 « perdu 5,5 Mo de police japonaise inutile » lue en japonais face au crédit Noto Sans JP : reformuler (« réduite à son sous-ensemble utile ») | `historique.rst` (français aussi) | 9L-1 | P5 #18 |

## Lot 4 — Garde-fous (technique, aucune chaîne)

| # | Changement | Fichiers | Source |
|---|---|---|---|
| 4.1 | Refuser tout numéro de version cité dans un `.rst` supérieur à `conf.py:release` (test Go ou étape de `release-check`) | test, `scripts/` | P7 #2 |
| 4.2 | Comparer `cli.CommandNames()` aux sections de `cli.rst` (modèle `helpVocabulary.sync.test.js`) | `internal/cli` test | P7 #5 |
| 4.3 | Extraire les routes `/v1/…` et `/ops/…` de `mode_headless.rst` et les comparer à `api_reference.rst` | test `cmd/` ou `internal/server` | P7 #3 |
| 4.4 | `sphinx-build -b linkcheck` dans `nightly.yml`, en avertissement (49 liens externes jamais vérifiés) | `.github/workflows/nightly.yml` | P7 #10 |
| 4.5 | Raccourcis : test entre `raccourcis.rst` et `keyboardService.js`/`tabToggles.js` | frontend test | P7 #11 |
| 4.6 | Skill de release : remplacer le bloc `make gettext` / `sphinx-intl update` (phase 1 étape 4) par `scripts/doc-po-update.sh` ; remplacer `stats.rst` par `stats_parity.rst` ; ajouter une phase « captures » (`make screenshots`, revue du diff) ou faire dire la vérité au `Makefile` ; contrôle `DatabaseVersion` contre `annexe_db_scheme.rst` | `.claude/skills/release-blunderdb/SKILL.md`, `Makefile` | P7 #6 #7 #8 #9 |
| 4.7 | Régénérer `panel_anki.png` (capture du 2026-09-03, composant modifié le 2026-09-06) | `doc/source/img/` | P7 #7 |
| 4.8 | Supprimer `stats.rst` (moignon `:orphan:` de 4 lignes, 9 catalogues) après vérification qu'aucune URL publiée `stats.html` ne perd un lecteur | `doc/source/stats.rst` + 9 `.po` | P7 #16 |

## Lot 5 — Structure : décisions à prendre (contradictions laissées visibles)

| # | Question | Pour | Contre | Coût de chaque issue |
|---|---|---|---|---|
| 5.1 | **La page produit et les langues.** Trois lecteurs (club, japonais, retour) butent sur une racine en anglais seul, sans indice que neuf langues existent, avec un sélecteur de langue enterré en bas de la barre latérale de la doc en codes latins. | P1 #1, P5 #1 #2, P6 #20 : traduire l'accroche ou au moins afficher une grille de langues en noms natifs sur la page produit, et remonter le sélecteur de la doc sous le logo | P7 #15 : la page produit est une neuvième source hors gettext, ne pas l'internationaliser | Grille de langues en noms natifs sur la page produit + sélecteur remonté dans `_templates/versions.html` : **T**, sans traduction, et répond aux trois lecteurs. Traduire l'accroche : neuf copies à tenir à la main. Recommandation : la grille, pas la traduction. |
| 5.2 | **Une référence unique pour la grammaire de recherche.** Quatre pages la disent (liste des commandes = table sans exemples, annexe filtres = exemples sans table, CLI = drapeaux, manuel = libellés), aucune ne se déclare la référence, alors que 0.36 annonce « une seule grammaire ». | P3 #12 #13, P7 #12 #13 : nommer `cmd_mode` référence (elle est verrouillée par le test), y rapatrier deux exemples et une colonne « équivalent CLI », réduire ou supprimer `annexe_filtres` | — | Fusion de l'annexe dans le manuel : suppression d'un catalogue en huit langues (comme `roadmap`), un `:ref:` entrant à rediriger, toctree. Colonne CLI dans la table : 9L-20 environ, aide intégrée régénérée. |
| 5.3 | **Guide contre manuel.** Huit panneaux décrits deux fois. | P7 #14 : ne pas fusionner, mais imposer la règle « le guide montre une tâche, le manuel décrit un écran » et transformer les descriptions d'écran du guide en `:ref:` | Coût d'une refonte : plusieurs dizaines de chaînes × 8 | À faire progressivement, page par page, jamais en un lot. |
| 5.4 | **Place de l'historique.** Décision amont : dernière entrée des Annexes. | P6 #14 : une utilisatrice de retour le cherche en tête et le trouve en avant-dernière position sur seize ; proposer de le remonter sous « Prise en main » ou de le citer dans le corps de l'accueil | Décision amont prise pour garder l'accueil sur un écran | Une phrase dans l'accueil (« Ce qui a changé : historique des versions ») coûte 9L-1 et règle le cas sans rouvrir la décision. |
| 5.5 | **Page produit : promesses à nuancer.** « judges any position offline » (le manuel a deux verdicts d'abstention), « five modes » dont trois nommés, trois formats d'import quand la doc en liste quatre. | P1 #18, P2 #16 | — | T (anglais seul). |
| 5.6 | **Coût marginal d'une page.** Le mainteneur mesure 20 lignes de catalogue par ligne de français (138 272 lignes de `.po` pour 6 942 de `.rst`), moitié dans quatre langues non relues ; une page de 191 lignes vaut 4 097 lignes de catalogue. | P7 #20 : chiffrer toute page nouvelle (msgid × 8) avant de l'accepter | Les personas 1, 2, 4 et 6 demandent chacun une page ou une section nouvelle | Convention à écrire dans `CLAUDE.md` : une page nouvelle est proposée avec son nombre de msgid. |

## Ce que les personas ont trouvé bon (à ne pas casser)

- L'avertissement d'absence d'authentification apparaît quatre fois, en encart,
  juste après les premiers exemples (P4).
- Les onglets Raccourcis et Commandes de l'aide intégrée sont identiques ligne
  pour ligne au site : la génération tient sa promesse là où elle s'applique (P6).
- L'annexe statistique cite ses sources dans gnubg et publie une mesure
  défavorable (61,1 % près du point de prise) au lieu de la taire (P2).
- La barre latérale de l'historique déplie toutes les versions datées (P6).
- La page CLI est complète sur tout ce qu'annoncent 0.35 et 0.36 (P6).
- La refonte a retiré 19 673 lignes nettes du dépôt, dont 3 413 entrées
  obsolètes de catalogues (P7).

## Ordre proposé

1. Lot 1 en une branche, tout de suite : dix-huit corrections, environ 40
   chaînes à retraduire, un changement de workflow.
2. Lot 4 en une branche technique : les garde-fous empêchent 1.2, 1.3 et
   2.6 de revenir.
3. Lot 3 par un relecteur japonais ou une passe outillée sur `ja/*.po`,
   indépendante du reste.
4. Lot 2 découpé par page (installation et guide ; CLI ; serveur ; glossaire
   et unités), chaque branche avec ses huit `.po`.
5. Lot 5 : quatre décisions à prendre avant d'ouvrir une branche.
