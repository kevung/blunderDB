# État d'exécution du plan 2026-09b

Dernière mise à jour : **2026-09-05** (quatrième session — audit de reprise). Ce
fichier est le point de reprise : il dit ce qui est fusionné, ce qui attend dans une
branche, et ce qui reste. Le plan lui-même est dans [README.md](README.md) ; les
fiches sont dans les fichiers de lot, chacune avec le numéro de son issue GitHub.

## Où en est-on

**93 des 145 issues du plan sont fermées.** Les étapes 0 et 1 sont livrées
en entier. L'étape 2 est livrée **sauf le lot G** (serveur). Restent, hors plan,
sept issues du générateur de bearoff (ADR-0027, conçu et non exécuté), trois
issues de moteur/amont et la vidéo de démo.

Les quatorze recherches externes P5-P18 sont versées sous
[`docs/recherche/`](../../docs/recherche/README.md) ; quatre ont corrigé une fiche
avant qu'un agent n'y travaille (le format OGXM n'existe pas — I.4 et I.5 réécrites ;
l'efficacité du videau est une question de modèle et non un trou de port — C.5 ;
zstd avec dictionnaire — B.12).

### L'audit du 2026-09-05 a corrigé ce fichier

La session précédente l'avait laissé en retard d'une journée de travail. Étaient
annoncées « restantes » huit fiches en réalité fermées et livrées le 2026-09-03 :

| Fiche | Issue | Commit |
|---|---|---|
| B.13 — contexte annulable | #181 | `fdfa179b5` |
| B.14 — duplication entre backends | #182 | `3447d90ef` |
| C.7 — forme close de `levelSolve` | #194 | `46e4e8f0b` → **ADR-0032** : la forme close se décide en amont |
| C.10 — allocations résiduelles | #197 | `04643aa6e` |
| C.11 — surface exportée morte | #198 | `4bdc4f926` |
| C.12 — documentation du moteur | #199 | `d4012a0a9` |
| D.9 — jetons de couleur | #209 | `ffda2996b` → **ADR-0031** |
| D.10 — modules-dieux | #210 | `af121a43e` |

Deux issues étaient restées ouvertes alors que leur travail était sur `main` — le
`Closes` n'avait pas pris à la fusion : **#214** (D.13, `04d5a26fe`) et **#254**
(H.12, `4d86e4af6`). Fermées à la main le 2026-09-05.

**#317, « Nightly failed: 2026-09-04 », est un faux positif** : le run a tourné sur
`aff3b79da`, antérieur de dix heures au correctif `fb61d10f4` qui fait sauter
`TestDSN_PathWithQuestionMarkOpensTheRightFile` sur Windows. Le correctif est sur
`main` depuis. Relance manuelle du workflow pour le prouver, puis fermeture.

**La leçon vaut d'être répétée** : avant d'ouvrir un chantier, lire le code plutôt
que la case. Elle a maintenant coûté deux fois.

## Fusionné sur `main`

| Fiches | Issues | Ce qui a changé |
|---|---|---|
| **Lot A entier** (A.1-A.14) | #155-#168 | Étape 0 : `X-Tenant-ID` non numérique refusé, `metadata` hors `/v1`, PRAGMAs sur toutes les connexions, dépôt et workflows durcis, sept fuites d'erreur serveur, 96 chaînes traduites, base de démo régénérée, notices tierces, conteneur `.dbx` v2, `/healthz` sans stockage, filtre d'erreur déterministe. |
| **Lot B entier** (B.1-B.19) | #169-#187 | Versions majeures numériques et migrations réparables ; Crawford propagé ; import qui refuse plus de quinze pions ; Anki transactionnel ; erreurs SQL cessent d'être avalées ; **schéma 2.18.0** (Jacoby et beaver hors identité Zobrist — ADR-0028 ; `analysis` unique par position ; `CHECK` de plage ; clés étrangères du journal) ; `--format json` sur neuf commandes et `completion` ; zstd à dictionnaire partagé (ADR-0030) ; contexte annulable ; duplication entre backends résorbée ; `find` et `StatsStore.Compute` découpées ; **une seule grammaire de recherche** portée en Go (`pkg/blunderdb/searchquery`) ; gouvernance des versions et dépendances. |
| **Lot C** (C.1-C.6, C.8-C.12) | #188-#193, #195-#199 | Tests moteur manquants et gold régénéré ; une seule règle de verdict de videau ; le lot d'analyse ne retente plus l'inévaluable ; **ADR-0029** (efficacité du videau mesurée par état, lue à la racine) ; **ADR-0032** (l'inversion du videau passe en forme close en amont) ; allocations, surface morte et documentation du moteur. |
| **Lot D entier** (D.1-D.15) | #201-#216 (sauf #212, hors plan) | Six corrections d'ergonomie ; vraie bascule des panneaux ; XGID encode Jacoby/beaver ; parseur de recherche unique ; `Tab` rendu à la navigation clavier ; plafond de warnings Svelte câblé en CI ; **ADR-0031** (une palette, migration progressive) ; modules-dieux découpés ; fiabilité des tests frontend. |
| **Lot E entier** (E.1-E.12) | #217-#228 | `test-os` bloquant, `GOOS=windows go vet`, couverture mesurée juste avec plancher, job `-race` sur le moteur, `t.Skip` silencieux devenus `t.Fatal`, hooks versionnés, `make check` aligné sur la CI, tests Go parallèles et shardés (paquet `database` 159 s → 57 s), crans de complexité remesurés, `nightly.yml`. |
| G.1-G.4, G.6, G.7 | #229-#232, #234, #235 | Compose avec proxy authentifiant, arguments positionnels refusés, rate limit par défaut, tests d'isolation multi-tenant. |
| **Lot H entier** (H.1-H.14) | #243-#256 | Page d'installation, `CONTRIBUTING.md`, code de conduite ; tap Homebrew, Flatpak, `metainfo.xml` ; quatre tutoriels, FAQ élargie, vingt-deux captures régénérables par `make screenshots` ; « Bearoff » → « Eval » en neuf langues ; **aide intégrée engendrée depuis les sources Sphinx** (ADR-0034) ; page d'accueil du site et négociation de langue ; feuille de route publique ; visites guidées ; **binaire Linux arm64** (cinquième runner natif, `.deb`/`.rpm`/AUR bi-architecture). |
| — | #304 | Coquilles de la source française. |
| — | — | `.gitattributes` : `* -text`, sans quoi un checkout Windows convertit les fins de ligne de `met_kazaross_xg2.json` et fait paniquer `engine.init` sur son SHA-256. |

Hors fiches, trouvés en chemin :

- **Les traductions sont complètes pour la première fois depuis longtemps** :
  `scripts/doc-i18n-check.sh` finit sur « all translations complete ». Six chaînes
  de `manuel.rst` étaient servies en français sur les huit sites traduits.
- **`msgfmt -c` refusait `en/manuel.po`** depuis plusieurs versions : `msgmerge`
  marque `python-format` toute entrée dont le texte français contient « 93,4 % ».
  Corrigé, et `scripts/doc-po-update.sh` écrit pour que le faux positif — qui revient
  à chaque régénération — soit traité une fois pour toutes.

## Ce qui reste, et dans quel ordre

L'objectif tenu depuis le 2026-09-05 : **vider le compteur d'issues**, par vagues,
chacune close par une release.

| Vague | Contenu | Issues | Sortie |
|---|---|---|---|
| **0 — hygiène** | Faux positif nightly, issues fantômes, ce fichier | #317, #214, #254 | — |
| **1 — release** | Publier ce que `main` porte déjà | — | **0.36.0** |
| **2 — lot G** | Reprendre la fusion bloquée, puis les cinq fiches restantes | #233, #236-#242 | **0.37.0** |
| **3 — bearoff** | ADR-0027 exécuté : générateur pur, empreintes, rien d'embarqué, onglet, CLI, doc, EPC au-delà du jan | #305-#311 | **0.38.0** |
| **4 — lot I** | 34 fiches produit, par paquets thématiques | #257-#290 | 0.39 → 0.40 |
| **5 — moteur/amont** | Mesure 2-ply contre la table exacte, noyau NEON, décisions amont | #127, #151, #200 | — |
| **6 — lot J** | Dix chantiers de fond, **chacun une décision produit avant toute ligne de code** | #291-#300 | 1.0 ? |
| **7 — hors code** | Vidéo de démo : enregistrement humain, pas une tâche d'agent | #102 | — |

### La branche qui attend

`feat/g8-g10-g13-serveur` (worktree `../blunderDB-g8-g10-g13-serveur`) porte G.8
(contrat d'API), G.10 (observabilité) et G.13 (GUI Go). Elle est **au milieu d'une
fusion** : tous les conflits sont résolus et indexés, le commit de merge n'a jamais
été conclu, et elle a 81 commits de retard sur `main`. À reprendre en ouverture de
la vague 2, dans une passe dédiée — pas au fil d'une autre fiche.

### Ce qui ne se ferme pas par du code

- **Lot J** : #300 (jouer contre gammonNet) est déjà écarté ; #299 (Ollama) et #296
  (mode club) demandent un arbitrage produit avant d'être chiffrés. Une fiche de ce
  lot peut légitimement se fermer sur un ADR « écarté, et pourquoi ».
- **#102** : refaire la vidéo de démo suppose un enregistrement d'écran commenté.

## Deux pièges qui ont coûté cher

1. **`git add` un fichier dès qu'il est résolu**, avant toute boucle sur
   `--diff-filter=U`. Une boucle `git checkout --ours` a écrasé une résolution
   manuelle non ajoutée à l'index et livré une `main` dont `TestDatabaseParity`
   échouait. `po_graft.py` a le même piège en interne.
2. **`go test ./... | grep … | head -N; echo $?` ment** : `$?` est celui de `head`,
   et les lignes de journal poussent le `FAIL` hors des N premières. Écrire
   `go test ./... > log 2>&1; echo $?` et relire le journal.

`internal/cli/parity_test.go` est le fichier que tout le monde percute : ajouter la
ligne en même temps que la méthode évite d'en découvrir le trou à la fusion.
