# Taille du binaire — note de mesure (2026-09-03)

Question posée : « peut-on compresser mieux ou gagner de la taille ? », avec en
tête la police japonaise embarquée. Réponse mesurée : **non, pas à un prix
raisonnable**. Cette note fixe les chiffres pour que la question ne soit pas
rouverte sans nouvelle contrainte (plainte d'utilisateur, limite d'un canal de
distribution, budget mémoire).

Décision : aucun chantier ouvert. Sujet clos.

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
