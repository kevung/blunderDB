# État d'exécution du plan 2026-09b

Dernière mise à jour : **2026-09-06** (cinquième session — vague 4, lot I : la
vague de schéma 2.19.0 et les trois premières fiches produit). Ce
fichier est le point de reprise : il dit ce qui est fusionné, ce qui attend dans une
branche, et ce qui reste. Le plan lui-même est dans [README.md](README.md) ; les
fiches sont dans les fichiers de lot, chacune avec le numéro de son issue GitHub.

## Où en est-on

**98 des 145 issues du plan sont fermées.** Les étapes 0, 1 et 2 sont livrées,
le lot G compris — cinq de ses fiches fermées, trois livrées à moitié et
documentées comme telles ci-dessous. Hors plan, le générateur de bearoff
(ADR-0027) est exécuté pour moitié : les deux générateurs et la sortie des
tables du binaire sont faits (#305, #306, #307), l'interface et la CLI ne le
sont pas. Restent aussi les trois issues de moteur/amont et la vidéo de démo.

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

| Vague | Contenu | Issues | État |
|---|---|---|---|
| **0 — hygiène** | Faux positif nightly, issues fantômes, ce fichier | #317, #214, #254 | ✅ |
| **1 — release** | | | ✅ **0.36.0** publiée le 2026-09-05 |
| **2 — lot G** | Fusion bloquée reprise, puis les fiches restantes | #233, #236, #238, #240, #241 | ✅ ; #237, #239, #242 restent (voir plus bas) |
| **3 — bearoff** | Les deux générateurs, les empreintes, rien d'embarqué | #305, #306, #307 | ✅ ; #308-#311 restent |
| **4 — lot I** | 34 fiches produit, par paquets thématiques | #257-#290 | en cours : 11 fermées (voir plus bas), 23 restantes |
| **5 — moteur/amont** | Mesure 2-ply contre la table exacte, noyau NEON, décisions amont | #127, #151, #200 | #127 fermée (la mesure existait déjà et était publiée) ; #151, #200 restent |
| **6 — lot J** | Dix chantiers de fond, **chacun une décision produit avant toute ligne de code** | #291-#300 | #300 fermée par **ADR-0037** ; neuf restent, dont plusieurs demandent un arbitrage produit |
| **7 — hors code** | Vidéo de démo : enregistrement humain, pas une tâche d'agent | #102 | à faire |

### La session du 2026-09-06 : dix-neuf issues fermées

| Issue | Fiche | Ce qui a été livré |
|---|---|---|
| #264 | I.8 | **ADR-0035** : la phase de partie, dérivée du plateau seul, jamais modifiable, recalculée par `repair`. Jeton `ph:race`, drapeau `--phase`, route. |
| #263 | I.7 | `comment.origin`, et la clause manquante du prédicat de rétention dans ses **trois** copies : un commentaire écrit par l'utilisateur retient sa position. Étiquette de provenance dans le panneau. |
| #257 | I.1 | Le compte rendu de fin d'import. **La moitié mesurée l'emporte sur la moitié stockée** : PR du lot, positions marquées, positions sans analyse, cinq pires décisions, recalculés à chaque consultation. |
| #288 | I.32 | La vérité sur la synchronisation (un fichier, un écrivain, `serve` pour le multi-poste) ; le geste s'appelle enfin « Fusionner ». |
| #261 | I.5 | La couverture indirecte des plateformes en ligne, **avec ce qui n'a pas été mesuré écrit noir sur blanc**. |
| #281 | I.25 | Publier une base : quatre champs qui existaient déjà, l'annuaire est une catégorie de discussions — conséquence de l'ADR-0007, pas commodité. |
| #127 | — | **La mesure existait déjà et était publiée.** Rejouée : chiffres identiques, en 18,6 s au lieu de 196. |
| #270 | I.14 | `analyze --compare` : ce que gammonNet vaut sur la bibliothèque de l'utilisateur, sans rien écrire. `engine.CanonicalMove` fait passer l'accord de 78,8 % à 93,2 % — quinze points de dialecte. |
| #272 | I.16 | **Mesuré, et le réglage n'est pas offert** : 85× le coût pour 0,0000 d'équité par décision. |
| #285 | I.29 | La corbeille, **instantané et non colonne `deleted_at`** (ADR-0036). Restaurer n'est pas symétrique de supprimer, et c'est écrit. |
| #266 | I.10 | Les mêmes décisions découpées par phase, par étiquette et par score. L'oracle figé des statistiques cesse d'être une contrainte sur l'avenir. |
| #290 | I.34 | **Doublon de #242**, déjà livré par `bd1d0d992`. Vérifié dans le code plutôt que dans la case. |
| #284 | I.28 | L'écran d'accueil. **« Importer mes matchs » enchaîne** base neuve → fichiers → compte rendu (#257) → file d'étude (#259) : la promesse tenue en deux minutes, pas expliquée. |
| #262 | I.6 | `import XGID=…` (le même verbe qu'`import`, avec un argument) et le bouton « enrichir depuis un fichier ». **OGID écarté ici** : sa grammaire doit d'abord être relevée sur des échantillons réels (#260). |
| #259 | I.3 | La file d'étude post-import. Trois raisons ordonnées (ce qui a coûté, ce que le logiciel d'origine avait marqué, les videaux serrés), une position une seule fois, **rien d'enregistré** : ce qu'on en fait EST la trace. |
| #265 | I.9 | Le vocabulaire de tags. Recherche **délimitée** (`#prime` ≠ `#priming`), cumulative (plusieurs tags = « les deux ») ; fenêtre du vocabulaire avec cardinalité, autocomplétion au `#`, `list --type tags`. Zéro schéma, comme la fiche l'annonçait. |
| #258 | I.2 | Le dossier surveillé. **Scrutation, pas `fsnotify`** : le repli devrait exister de toute façon pour les partages réseau, donc il est le seul chemin plutôt que le second. Le Go regarde, l'interface importe — un import surveillé est le même import qu'un glisser-déposer. |
| #268 | I.12 | **Un match importé sans analyse obtient un PR.** La fiche prescrivait `move_analysis` ; rien ne lit cette table pour les statistiques. Le vrai trou était `best_move_equity_error`, laissé à zéro faute de savoir quel coup avait été joué. |
| #267 | I.11 | La matrice du videau : le verdict de la position à tous les scores d'un match de 5, 7 ou 9. Commande `cm`, commande CLI `cubematrix`, route `/v1/gammonnet.cubeMatrix`. |

Sur I.6, l'enrichissement ne cachait aucun mécanisme neuf : réimporter le même
match dans un autre format l'enrichit déjà en place. Le bouton n'apporte que
la trouvabilité, et c'est écrit tel quel dans la doc plutôt que présenté
comme une fonctionnalité. La moitié OGID de la fiche est laissée à #260,
qui exige d'abord des échantillons réels — écrire ce lecteur sans eux serait
exactement ce que la fiche I.4 interdit après le rapport P9.

Sur I.3, la bande de la file n'invente **aucun geste** : trois de ses quatre
boutons ouvrent le panneau où le geste se prend déjà. Les refaire dans la
bande en aurait fait des demi-copies qui dérivent ; la file apporte l'ordre et
le parcours, pas des commandes nouvelles.

Sur I.9, un écart : la fiche disait « panneau », c'est une **fenêtre**. On
consulte un vocabulaire puis on le referme, et cliquer un tag lance une
recherche qui bascule de toute façon sur les résultats ; une fenêtre qui se
ferme au clic est exactement ce comportement. Un onzième onglet aurait coûté
une icône, un raccourci et la place dans la barre, pour une surface qu'on
n'habite pas.

Sur I.2, deux écarts assumés. La fiche proposait `fsnotify` : ce port scrute,
parce qu'un dossier XG vit très souvent sur un partage réseau ou un dossier
synchronisé où inotify ne rapporte rien (ADR-0004), que le repli devrait donc
exister de toute façon, et qu'un `readdir` toutes les dix secondes n'a pas de
coût mesurable — une dépendance et une capacité hôte de moins. Et la fiche
proposait de pointer par défaut sur le dossier d'XG : un chemin inventé
d'après ce qu'XG installe sur le Windows de quelqu'un d'autre est exactement
la sorte d'affirmation non vérifiée que ce projet refuse. Le bouton
« Proposer » ne propose donc un chemin que si `os.Stat` le trouve, et
l'utilisateur l'accepte.

Sur I.12, la fiche se trompait de mécanisme et il fallait le dire : elle
demandait d'écrire une `move_analysis` étiquetée `gammonNet <version>`, mais
aucune statistique ne lit cette table — elle n'est écrite qu'à l'import et
relue qu'à l'export `.mat`. Le PR est une somme de `analysis.best_move_equity_error`,
que le lot gammonNet laissait à zéro : on lui donne une position, et une
position ne se souvient pas de ce qu'on en a fait. La table `move`, elle, le
sait, et elle est écrite à l'import qu'un fichier porte une analyse ou non.
`blunderdb repair` rend donc leur PR aux bases déjà analysées, sans repasser
le moteur. Mesuré sur `testdata/test.mat` : « aucune analyse à noter » →
**PR 5,43 sur 284 décisions**, cinq pires décisions nommées.

Deux écarts assumés sur I.11, écrits ici pour n'avoir pas à les redécouvrir :
la grille est une **fenêtre** et non un onglet du panneau Eval — le panneau est
sans défilement par construction (ADR-0018, ADR-0021) et 9×9 n'y tient pas à
côté du plateau, tandis que la table d'équité de match, exactement la même forme,
est déjà une fenêtre ; et la grille est **post-Crawford d'un bout à l'autre**,
parce qu'une colonne « vous ne pouvez pas doubler » ne dirait rien de la
position. Le balayage 2-ply coûte 1,56 s en 9 points, pas les 5,7 s estimés :
une case de videau ne génère pas de coups.

Fermée aussi : **#300** (J.10, « jouer contre gammonNet »), par **ADR-0037** —
la fiche demandait « une ligne dans une ADR pour ne pas rouvrir la question »,
elle a maintenant sa page.

### Ce qui ne se fera pas par un agent

- **#102** (vidéo de démo) demande un enregistrement d'écran commenté par un
  humain. Rien à automatiser.
- **#151** (noyau NEON arm64) demande une machine arm64 pour être écrit et
  surtout pour être **vérifié** : ADR-0024 exige que tout nouveau chemin
  arithmétique passe `kernel_identity_test.go` bit pour bit sur la cible. Écrire
  du Plan 9 arm64 sans pouvoir l'exécuter serait livrer non mesuré exactement ce
  que cette ADR interdit.
- **#296** (mode club) et **#299** (Ollama) demandent un arbitrage produit que
  le plan attribue explicitement à une session de *grilling*, pas à une fiche.

### La vague de schéma 2.19.0 (2026-09-06)

Les quatre changements de schéma que le lot I demandait sont faits **en un seul
saut de version**, comme le plan le prévoyait, plutôt qu'en quatre montées
successives du même fichier :

- `position.game_phase` — **ADR-0035**. Étiquette dérivée du plateau seul,
  jamais modifiable, recalculée par `blunderdb repair`. Trois de ses quatre
  frontières sont celles de gnubg et sont sourcées (P5) ; la quatrième est une
  constante nommée, `OpeningDisplacementMax`, comme P5 le demande. Le numéro de
  coup n'entre pas dans le calcul, contre ce que la fiche proposait : une
  position se rencontre au coup 3 d'un match et au coup 30 d'un autre.
- `comment.origin` (#263) — et surtout la clause qui manquait au prédicat de
  rétention dans ses **trois** copies : un commentaire écrit par l'utilisateur
  retient sa position à la suppression du match.
- `import_batch` + `match.import_batch_id` (#257) et `trash` (#285, **ADR-0036**)
  — créées dans la même vague pour que le reste du lot n'ait pas à rouvrir le
  schéma. La corbeille est un **instantané**, pas une colonne `deleted_at` :
  aucun des cinquante filtres, aucune des deux implémentations de statistiques,
  aucun prédicat de rétention n'a eu à apprendre son existence. **#285 reste à
  faire** : le socle est là, les gestes ne le sont pas.

Fermées : **#264**, **#263**, **#257**. Trouvé en chemin : 340 lignes de
`commentStore` mortes dans les deux backends.

### Le piège des catalogues, payé deux fois

Les deux premiers commits de la session ont rempli les `.po` avec **polib**, qui
enveloppe les lignes autrement que sphinx-intl : 17 000 lignes réécrites dans
`manuel.po` pour six phrases ajoutées. C'est exactement ce que CLAUDE.md
interdit, et la règle y était déjà. Les catalogues sont repartis de `792503bca`
et refaits ; la vague coûte 5 900 lignes au lieu de 28 000.

`scripts/po-fill.py` existe maintenant pour cela : il modifie le catalogue **en
tant que texte** et ne touche que les entrées qu'on lui demande de remplir.
CLAUDE.md le nomme. **Ne jamais charger un `.po` dans polib pour le sauver.**

### Ce qui est livré à moitié, et pourquoi

Trois fiches sont utiles en l'état et leur reste est écrit dans leur commit ;
l'issue est restée ouverte plutôt que fermée à tort.

- **G.9 (#237)** — la compression gzip des flux NDJSON est livrée et mesurée
  (13,5 % de la taille sur mille lignes). Manque la pagination des familles
  listantes : elle touche le contrat Storage et ses trois implémentations, et
  une limite par défaut côté serveur serait une rupture d'API pour un démon
  qui a déjà des clients.
- **G.14 (#242)** — l'assertion inverse de parité est en place, et les cinq
  capacités qui n'existaient que sur le démon ont leur `Database` et leur CLI.
  Manque leur face GUI (menu contextuel d'une carte, journal dans l'onglet
  Anki, bouton de réparation dans la configuration).
- **G.11 (#239)** — non commencée.

Du lot bearoff, restent l'onglet complet (#308), la CLI `bearoff` (#309), sa
doc (#310) et le lot 2 (#311, l'EPC au-delà du jan). Le socle est là : les
deux générateurs sont identiques à gnubg octet pour octet, vérifiés par
empreinte, et le binaire a perdu 7,33 Mio (−21,2 %).

### Ce que la traversée a appris

- **Un préfixe de route peut déplacer une frontière de sécurité.** Passer
  `vacuum` et `purge` sous `/ops/` les a rendues *publiques*, parce que
  `publicPaths` disait « tout ce qui n'est pas /v1/ ». Les deux appels les
  plus destructeurs du démon seraient devenus les seuls sans tenant. Le test
  l'a dit dans la minute ; la leçon est que les listes écrites en négatif
  changent de sens quand l'ensemble change.
- **Le tenant SQLite était une fiction** : `TenantFilter` valait `1=1`, donc
  deux tenants lisaient les mêmes lignes derrière un en-tête exigé et accepté.
  Trouvé en traitant G.12, sans rapport avec la fiche.
- **Un portage se juge à l'octet.** Les deux générateurs ont été écrits contre
  un test d'identité avec le fichier de gnubg ; c'est ce test, et non la
  relecture, qui a attrapé le complément à un, l'ordre des diagonales et le
  mode qui absorbe l'arrondi.

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
