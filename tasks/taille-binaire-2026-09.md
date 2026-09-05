# Taille du binaire — note de mesure (2026-09-03)

Question posée : « peut-on compresser mieux ou gagner de la taille ? », avec en
tête la police japonaise embarquée. Réponse mesurée : **non, pas à un prix
raisonnable**. Cette note fixe les chiffres pour que la question ne soit pas
rouverte sans nouvelle contrainte (plainte d'utilisateur, limite d'un canal de
distribution, budget mémoire).

Décision : aucun chantier ouvert. Sujet clos. Complété le 2026-09-03 par la
mesure gammonNet contre la table deux faces et la piste du générateur.

## Ce que pèse la livraison 0.35.0

| Artefact | Taille |
|---|---|
| Linux (UPX `--best`), Windows `.exe` | 19,0 Mo / 17,6 Mo |
| `.deb`, `.rpm`, `.tar.gz`, flatpak | 16,7 à 17,3 Mo |
| macOS `.zip` (binaire universel, sans UPX) | 34,5 Mo |
| Binaire local strippé, sans UPX | 35,4 Mo |

## Ventilation du binaire brut (35,4 Mo, `go tool nm -size` sur un build avec symboles)

| Poste | Brut | Compressé (zstd -19) |
|---|---|---|
| Bearoff deux faces `engine/race/gnubg_ts0.bd` (verdicts de course, ADR-0009) | 6,8 Mo | 5,7 Mo |
| Réseau gammonNet + élagueur (`engine/gammonnet/*.bin`) | 2,1 Mo | 2,0 Mo |
| Frontend compilé (`frontend/dist`, dont 1,7 Mo de JS) | 2,2 Mo | ~0,5 Mo |
| SQLite pur Go (`modernc.org/sqlite/lib` + `libc`) | ~2,0 Mo | code |
| Bearoff une face `engine/gnubg_os6.bd` (EPC) | 1,4 Mo | 0,85 Mo |
| pgx + `crypto/tls` + `net/http` (le daemon `serve`, lié dans le binaire desktop) | ~1,5 Mo | code |
| Base de démo `internal/gui/demo.db.gz` | 0,4 Mo | déjà compressée |
| Police Noto Sans JP (sous-ensemble WOFF2, commit 8393de7b) | 0,18 Mo | déjà compressée |
| Runtime Go, Wails, code blunderDB | le reste | code |

La police japonaise représente 0,5 % du binaire. Elle est déjà réduite au
sous-ensemble utile (5,7 Mo TTF → 178 Ko). La retirer ne se verrait pas ; la
garder coûte rien. Point réglé.

## Leviers évalués et refusés

### UPX LZMA (`-upxflags "--best --lzma"`) — refusé

Wails passe `--best --no-color --no-progress` et **pas** `--lzma`
(`wails/v2/pkg/commands/build/base.go`). Mesure locale, binaire 0.35.0+ :

| Binaire | Taille | Démarrage (`blunderDB version`, moyenne de 3) |
|---|---|---|
| brut, strippé | 35,4 Mo | 43 ms |
| UPX `--best` (CI actuelle) | 20,1 Mo | 130 ms |
| UPX `--best --lzma` | 15,6 Mo | 899 ms |

−22 % de téléchargement contre presque une seconde de décompression **à chaque
lancement**, y compris chaque appel CLI et chaque `call` de script. Refusé.
À noter : l'UPX actuel coûte déjà ~90 ms par lancement pour 15 Mo gagnés ;
si un jour la CLI est appelée en boucle par un outil tiers, c'est UPX
tout entier qu'il faudra remettre en question, pas LZMA.

### Réencodage sans perte du bearoff deux faces — refusé

Les quatre plans uint16 (cubeless, cubeful ×3) sont lisses dans les deux
axes. Mesure (`numpy`, script jetable) :

| Transformée | zstd -19 | xz -9e |
|---|---|---|
| brut | 5,71 Mo | 5,38 Mo |
| plans séparés | 5,70 Mo | 5,46 Mo |
| plans + delta axe « us » | 5,09 Mo | 4,52 Mo |
| plans + delta + séparation octet haut / octet bas | 3,78 Mo | 3,56 Mo |
| plans + prédicteur 2D (gauche + haut − diagonale) + séparation octets | 3,69 Mo | 3,55 Mo |

Gain : 1,8 Mo sur 19 (~10 %). Prix : un format maison, un décodage de 6,8 Mo
en RAM au premier verdict de course (le lecteur actuel fait des lectures de
8 octets sur un `io.ReaderAt` et ne charge jamais le fichier), et une
modification du seul chemin dont l'invariant est « le verdict n'est jamais
estimé ». Les 160 états de `testdata/money_fixtures.json` rendraient la chose
sûre, mais 10 % ne justifient pas un format propriétaire. Refusé.

### Remplacer la table deux faces par gammonNet 2-ply — refusé, mesuré

Question du 2026-09-03 : ne plus embarquer `gnubg_ts0.bd`, le proposer au
téléchargement, et laisser le régime « évalué » (gammonNet) servir de plancher
sur son domaine. Mesure `TestBearoffFloorMeasure`
(`pkg/blunderdb/engine/gammonnet/`, `BLUNDERDB_BEAROFF_FLOOR=1`, N = 4000,
recherche configurée comme le panneau, feuilles valuées avec le videau) :

| | 0-ply | 1-ply | 2-ply |
|---|---|---|---|
| \|ΔP gain\| moyenne / p95 / max | 1,7 / 5,6 / 20,3 % | 1,4 / 5,1 / 17,2 % | 0,85 / 3,1 / 8,4 % |
| accord double / pas double, videau centré | 94,0 % | 95,8 % | 94,8 % |
| accord double / pas double, videau possédé | 95,5 % | 97,1 % | 96,3 % |
| coût moyen en équité exacte (centré) | 0,0095 | 0,0051 | 0,0065 |
| décisions coûtant ≥ 0,08 (centré) | 3,65 % | 2,50 % | 3,00 % |
| coût maximal | 0,500 | 0,336 | 0,446 |
| accord take / pass | 95-96 % | 96 % | 96 % |

La probabilité de gain converge avec le pli ; le verdict, non. Les désaccords
sont presque tous des « double/take » exacts que gammonNet refuse (2-ply
centré : 189 DT→ND sur 206 désaccords), sur des positions à un ou deux
lancers de la fin : le modèle de videau vivant (efficacité, vigueur du
redouble) y sous-évalue le double de 0,3 à 0,45 point alors que le videau est
mort. C'est précisément le domaine que la table couvre. À titre de
comparaison, l'estimateur par convolution est à σ = 0,05 % sur la
probabilité de gain, trente fois plus près que le réseau à 2 plis. Refusé ;
ADR-0009 et ADR-0012 tiennent.

### Régénérer les tables au lieu de les embarquer — FAIT (ADR-0027, 0.36.0)

La table est une donnée dérivée : `makebearoff -t 6x6` la produit en 24 s, et
un prototype Go d'induction arrière (port de `generate_ts` / `BearOff2` /
`CubeEquity`, arithmétique `short int` reproduite, parcours par diagonales
d'index `us+them` croissant) la reproduit **octet pour octet** en 2,3 s sur
16 cœurs, 5,5 s mono-cœur sans optimisation. C'est le seul levier qui sort
6,8 Mo du binaire (un tiers de la livraison compressée) sans toucher à
l'invariant « le verdict n'est jamais estimé » : exact reste exact, calculé
au lieu de lu. Prix : ~150 lignes de générateur, un test d'identité contre le
fichier gnubg gardé en fixture, un cache dans le répertoire de données XDG
(comme le TS-06-11 téléchargé) ou un calcul en mémoire au premier verdict.

**Fait en 0.36.0**, et davantage : la table une face de l'EPC (`gnubg_os6.bd`,
8,2 Mo) est partie avec, ainsi que l'asset de téléchargement de 1,2 Go. Le
paquet `pkg/blunderdb/engine/bearoffgen` porte les deux générateurs, chacun
identique octet pour octet à `makebearoff` et vérifié contre une empreinte
SHA-256 enregistrée.

Mesure sur le binaire Linux `-tags webkit2_41`, entre 0.35.0 et le retrait des
deux `go:embed` :

| | 0.35.0 | après | écart |
|---|---:|---:|---:|
| brut, strippé | 34,6 Mo | 27,3 Mo | −7,33 Mo (−21,2 %) |

C'est le plus gros levier de toute cette note, et le seul qui n'ait rien coûté
à l'utilisateur : les deux tables par défaut se refont au premier lancement en
arrière-plan, six secondes d'un cœur, et le domaine étendu se calcule sur
demande au lieu de se télécharger. Le générateur deux faces est parallèle par
diagonale (TS-06-09 : 78,9 s → 9,8 s sur seize fils) et reprend après une
pause.

### Exclure pgx du binaire desktop (build tag) — refusé

~1 Mo compressé. Le poids de pgx est de la réflexion (`pgtype` enregistre tous
ses types), pas du code appelé, donc seul le retrait du paquet entier compte.
Contredit la règle « un exécutable, cinq modes » de `CLAUDE.md` ; `cmd/serve`
existe déjà pour le sens inverse (daemon sans Wails). Refusé.

### macOS : un zip par architecture — non fait

Le binaire universel double le téléchargement (34,5 Mo). Deux artefacts
arm64/amd64 le diviseraient par deux, au prix d'un choix demandé à
l'utilisateur et d'une matrice CI plus large. Aucune plainte : non fait.
C'est le seul levier à rouvrir si une plainte macOS arrive.

### Recoder à la main plutôt qu'appeler une bibliothèque — non

L'éditeur de liens Go élimine déjà toute fonction non appelée. Ce qui reste
d'une bibliothèque est ce qu'on utilise réellement (SQLite : le moteur
lui-même) ou ce que la réflexion rend atteignable (pgx). Réécrire gagne des
dizaines de kilo-octets contre des bugs ; seul le retrait d'un sous-arbre de
dépendances entier se voit, et il n'y a qu'un candidat, traité ci-dessus.

## Comment refaire la mesure

```bash
go build -tags webkit2_41 -o /tmp/bdb-sym .          # build avec symboles
go tool nm -size -sort size /tmp/bdb-sym | head -50   # gros symboles
size build/bin/blunderDB                              # text / data / bss
cp build/bin/blunderDB /tmp/b && upx --best --lzma /tmp/b && ls -l /tmp/b
gh release view --json assets -q '.assets[] | "\(.size) \(.name)"'
```
