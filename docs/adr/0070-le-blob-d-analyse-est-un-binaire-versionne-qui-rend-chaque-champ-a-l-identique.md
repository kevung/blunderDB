# Le blob d'analyse est un binaire versionné qui rend chaque champ à l'identique

Statut : acceptée.
Voir aussi : ADR-0030 (zstd avec dictionnaire, le blob nomme son codec).

## Contexte

Le format hérité d'`analysis.data` est le JSON de `domain.PositionAnalysis` compressé par
zstd avec dictionnaire (ADR-0030). Sur la base BMAB (15,6 M blobs, `tasks/plan-grosses-bases-2026-10/POIDS.md`
§2-§3), un blob pèse 284 o en moyenne, dont une bonne part redit à chaque ligne les noms de
champs, le texte décimal des nombres et deux horodatages RFC 3339 à la nanoseconde ; et lire
une analyse coûte surtout `json.Unmarshal` (80 % du décodage). Les blobs font environ 30 % du
fichier.

## Décision

1. **Enveloppe.** Un blob binaire commence par `0xBA` puis l'octet de version (`1`), suivis
   d'une trame zstd **privée de ses quatre octets de magie** (la version les implique ; le
   décodeur les remet). `0xBA` n'est ni `{`, ni le premier octet de la magie zstd, ni un CMF
   zlib (quartet bas 8) : la détection par contenu d'ADR-0030 §2 s'étend sans collision, et
   aucune version de schéma, aucune colonne ne dit le format. Une version inconnue est une
   erreur (`ErrUnknownAnalysisFormat`), jamais une lecture approximative.
2. **Compression.** zstd avec un second dictionnaire embarqué,
   `engine/analysis_bin_dict.bin` (64 Ko), entraîné par `cmd/train-analysis-dict` sur les
   charges binaires du corpus du dépôt (fixtures et base de démo), comme le premier. Niveau 7
   sans somme de contrôle sur tous les chemins d'écriture, compaction comprise : le niveau 19
   coûte vingt fois le CPU pour moins de 2 % des octets. Une trame de niveau 19 (avec somme de
   contrôle) reste lisible ; le drapeau de la trame, au troisième octet du blob, la distingue
   sans décompresser. Le dictionnaire JSON est gelé et ne sert qu'à lire les lignes héritées ; les deux
   vivent dans le même décodeur (chaque trame nomme son `Dictionary_ID`).
3. **Contenu : tous les champs, rien de reconstruit des colonnes.** La charge
   (`engine/analysisbin.go`) écrit chaque champ de `PositionAnalysis` et de ce qu'il contient,
   dans un ordre fixé par la version. Aucun champ n'est omis : ceux qu'on espérait retirer ne se
   reconstruisent pas à l'identique depuis les colonnes —
   - `creationDate` : la colonne `creation_date` est tronquée à la seconde, le blob tient la
     nanoseconde et le décalage horaire ; `lastModifiedDate` n'a pas de colonne ;
   - `player1`/`player2` : `analysis` n'a pas de colonne de noms, et le match d'origine n'est
     ni unique (une position est partagée) ni durable (un match se supprime) ;
   - `positionId` : le chemin d'écriture ne force pas l'égalité avec `analysis.position_id`, et
     une quinzaine de lecteurs décodent un blob sans la colonne sous la main.

   Ils sont donc encodés compactement au lieu d'être retirés : un horodatage tient en secondes,
   nanosecondes et décalage en varints, `lastModifiedDate` et les dates de rollout relativement
   à `creationDate`.
4. **Primitives, toutes exactes.** Entiers en varints (zigzag pour les signés) ; chaînes par
   une table propre au blob, amorcée par une liste figée de libellés (moteurs, profondeurs,
   verdicts de videau, types) qui coûtent un octet ; tranches par un compte `n+1`, `0`
   gardant `nil` distinct de vide ; un flottant prend l'un de quatre modes — `k/100`, `k/1000`,
   la dérivation propre au champ, ou ses 8 octets IEEE-754 — et l'encodeur ne choisit un mode
   que s'il redonne **exactement les mêmes bits** (`-0` a son code). Les dérivations sont
   `opponentWinChance = arrondi₂(100 − playerWinChance)` et
   `equityError = arrondi₃(équité du premier coup − équité du coup)`, vraies dans 99,99 % et
   91 % des cas sur BMAB ; sinon la valeur est écrite. Un horodatage se relit comme
   `encoding/json` le relirait de son texte (décalage 0 → UTC, celui du fuseau local → `Local`,
   sinon zone fixe) : un blob rend la même valeur quel que soit son format.
5. **Mêmes valeurs que le JSON (ADR-0019).** La charge porte exactement ce que portait le
   JSON : les probabilités en pourcentage et les équités à l'échelle déjà stockée, arrondies à
   l'import par `RoundAnalysisForStorage`. Aucune échelle interne du moteur n'y entre ; le
   format ne transforme aucune valeur, il la code.
6. **Écriture binaire, lecture de tout.** `EncodeAnalysisForStorage` écrit toujours le binaire,
   sur les deux backends (PostgreSQL stocke le même blob dans son `BYTEA`).
   `DecodeAnalysisFromStorage` lit le binaire et les trois formats hérités. Pas de bump de
   `DatabaseVersion` : la colonne ne change ni de type ni de sens (ADR-0030 §2).
7. **Conversion explicite et reprenable, jamais à l'ouverture.** Une ligne héritée se lit pour
   toujours et se convertit quand elle est réécrite : fusion d'import, import de `.db` natif
   (`RecompressAnalysisData`), compaction de `vacuum` (`compactAnalyses` de `sqlite.Storage.Vacuum`, via
   `engine.RecompressAnalysesConcurrently`, qui laisse un blob déjà binaire tel quel). S'y ajoute `AnalysisStore.ReencodeAnalyses(after, limit)`,
   même forme que `MatchStore.ScoreMoves` : lots en ordre d'id, une transaction par lot, le
   premier octet comparé en SQL (`substr(data, 1, 1) <> x'BA'` sur les deux dialectes) pour
   qu'une reprise ne relise pas ce qui est converti ; un blob illisible est laissé et journalisé.
   `storage.ReencodeAllAnalyses` boucle la passe pour les trois modes :
   `Database.ReencodeAnalyses` (GUI, CLI `reencode`) et la route `/v1/maintenance.reencode`,
   limitée au tenant de l'appelant — coûteuse mais pas transverse, donc hors `/ops/` comme
   `gammonnet.sweepStale`.
8. **Décodeur borné.** Entrée hostile : jamais de panique ; chaque tranche est bornée par les
   octets restants et le total des éléments et chaînes par `maxBinaryElements` (65 536), si bien
   que les allocations suivent la taille de la charge, elle-même plafonnée par
   `MaxAnalysisBytes` à la décompression.

## Conséquences

- Mesuré sur 78 117 blobs BMAB (1 sur 200, copie de la base) : **284,4 → 158,8 o par blob
  (−44 %)** au niveau 7, 156,1 o au niveau 19 ; décodage **39 → 2,9 µs** par blob ;
  aller-retour exact sur chacun (JSON identique octet pour octet à celui de l'ancien blob). Un
  dictionnaire entraîné sur BMAB même ne fait pas mieux (≈ 157 o) : celui du dépôt suffit.
  POIDS.md estimait −39 % pour un binaire complet et −51 % sans `positionId`, noms ni dates ;
  la table de chaînes, les dérivations et les dates en varints rattrapent une partie de
  l'écart sans rien retirer.
- Un champ ajouté à `PositionAnalysis` doit être écrit par le codec **sous une nouvelle
  version** : `TestBinaryFormatCoversEveryField` épingle la forme du struct, et le format des
  lignes déjà écrites ne change jamais. Modifier la table d'amorce est aussi une nouvelle
  version.
- `DecompressAnalysisData` rend toujours du JSON (celui d'un blob hérité tel quel, celui d'un
  blob binaire remarshallé) ; `CompressAnalysisData` prend du JSON et écrit du binaire.
- Le dictionnaire JSON reste embarqué (32 Ko) tant qu'une base peut contenir une ligne héritée,
  c'est-à-dire pour toujours.
- Rejeté : retirer `positionId`, noms et dates du blob (variante « E » de POIDS.md) — aucun ne se
  reconstruit exactement, et l'exactitude prime sur sept points de taille.
- Rejeté : CBOR/MessagePack — ils gardent les noms de champs et le texte des petits flottants
  que ce format supprime, sans exactitude de `nil`/vide ni dérivations.
- Rejeté : convertir dans une étape de migration — réécrire 15 M lignes à l'ouverture, pour un
  gain que `vacuum` et `reencode` apportent quand l'utilisateur s'attend à attendre.

## Garde

`engine/analysisbin_test.go` (aller-retour exact de chaque analyse de la base de démo et de
valeurs limites, forme du struct, version inconnue, bombe binaire, montée des trois formats
hérités), `FuzzUnmarshalAnalysisBinary` et la graine binaire de
`FuzzDecodeAnalysisFromStorage` (`fuzz.yml`), `storagetest.CheckReencodeUpgradesLegacy` et le
cas de contrat `Analysis/ReencodeSkipsBinary` sur SQLite et PostgreSQL.
