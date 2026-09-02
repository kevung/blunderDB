# Compression de blobs JSON répétitifs dans SQLite/PostgreSQL pour blunderDB : recommandation chiffrée, protocole de mesure et plan de migration

## TL;DR
- **Recommandation tranchée** : passez de zlib niveau 9 à **zstd niveau 19 AVEC dictionnaire partagé** via la bibliothèque Go pure `github.com/klauspost/compress/zstd`. La man page officielle zstd chiffre le gain d'un dictionnaire sur les petits objets : *« Typical gains range from ~10% (at 64KB) to x5 better (at <1KB) »* ; le benchmark Facebook sur des « small JSON records » de ~300 o passe de ×1,33 à ×5,86–6,83 (**~×4,7**). Sur vos blobs de 2–20 ko, cela devrait réduire la table `analysis` (donc le fichier) de **~50 % à 80 %** vs zlib-9. L'extension `sqlite-zstd` de phiresky, qui applique exactement cette technique, annonce *« reduce the size of your database by 50 to 95% »* et montre des lignes de ~24 ko compressées ~×77 dans ses logs.
- **Le dictionnaire est le levier n°1**, pas le format de sérialisation : après compression, l'avantage d'un format binaire (CBOR, MessagePack) sur le JSON tombe à ~10–20 % et peut même s'inverser. Le dictionnaire, lui, multiplie le ratio. Gardez le JSON (ou passez à CBOR pour le gain CPU seulement), et investissez dans un dictionnaire zstd entraîné, versionné par `Dictionary_ID`, stocké dans une table de la base.
- **Migration** : préfixez chaque blob d'un octet de version/format (magic byte), détectez le format à la lecture (zlib = `0x78 0x9C`, zstd = `0x28 B5 2F FD`), recompressez en tâche de fond par batchs, faites un `VACUUM` final, protégez la compatibilité descendante avec `PRAGMA user_version`/`application_id`. Posez impérativement des bornes anti-bombe de décompression car un fichier blunderDB est échangeable entre utilisateurs — c'est un vecteur d'attaque.

## Key Findings

1. **Un dictionnaire partagé est décisif sur des petits JSON.** La man page zstd chiffre **×2 à ×5** pour les fichiers < 1 ko et **~10 %** à 64 ko. Le benchmark historique Facebook : small JSON records (~300 o) ×1,33→×5,86–6,83 (~×4,7) ; Mercurial events (~1,5 ko) ~×1,5 ; large JSON docs (~6 ko) ×3,8–4,0→×8,9–13,4 (~×2,8). Pour des blobs de 2–20 ko, on est dans la zone où le dictionnaire aide encore nettement mais avec un gain décroissant avec la taille.

2. **Il existe désormais un entraîneur de dictionnaire en Go pur (2026), mais expérimental.** Historiquement, `klauspost/compress` ne savait qu'*utiliser* un dictionnaire, pas l'*entraîner* (issue #140, close par PR #281 qui n'a ajouté que l'encodage avec dict). Depuis, le sous-paquet **`github.com/klauspost/compress/dict`** (`BuildZstdDict`, CLI `builddict`) permet de **construire un dictionnaire zstd en Go pur sans cgo**. MAIS le README le décrit comme *« experimental »*, il utilise un algorithme **maison** (pas COVER/fastCover — *« This diverges from the Zstandard dictionary builder »*) et a une très faible adoption. La voie mûre reste : **entraîner hors-ligne** avec le CLI C `zstd --train` (ou le cgo `valyala/gozstd` `BuildDict`, qui appelle `C.ZDICT_trainFromBuffer`), **livrer le dictionnaire comme asset statique**, puis le charger au runtime avec le zstd Go pur (`WithEncoderDict`/`WithDecoderDicts`). L'entraînement ne tourne que sur la machine du développeur : **la contrainte « pas de cgo » ne s'applique qu'au runtime**, ce qui lève l'obstacle.

3. **Le format binaire compact est un quasi-leurre après compression.** Plusieurs benchmarks publiés (msgpack issue #328, Peterbe, guides Jsonic/JSONCraft) montrent que le JSON compresse extraordinairement bien (clés répétées, chiffres ASCII), et qu'après gzip/zstd l'écart MessagePack/CBOR vs JSON tombe à **~10–20 %**, voire s'inverse (cas OpenAI embeddings : msgpack compressé *plus gros* que JSON compressé). Le format binaire vaut pour le **CPU** (2–4× plus rapide à parser) mais peu pour la **taille** une fois compressé.

4. **PostgreSQL : compresser côté application contourne TOAST et sa configuration.** Le seuil TOAST est ~2 ko ; un blob déjà compressé côté application est stocké tel quel. Sur PG14+, `default_toast_compression = lz4` est plus rapide que pglz pour un ratio proche (blogs Crunchy Data / The Build : *« pglz compressing perhaps 7% tighter on average while lz4 runs several times faster to compress and decompress »*), et **PG19 basculera le défaut sur LZ4** (commit d'Euler Taveira, revu par Peter Eisentraut & Aleksander Alekseev, début 2026). Mais **ni pglz ni LZ4 (TOAST) n'utilisent de dictionnaire partagé** : pour des petits JSON répétitifs, un zstd+dict côté application les battra sur les deux moteurs.

5. **Sécurité : les bombes de décompression sont réelles et amplifiées par un dictionnaire.** DEFLATE plafonne à ~1032:1 par couche ; zstd/brotli peuvent amplifier davantage via de grandes fenêtres. Un fichier blunderDB étant échangeable, un dictionnaire ou un blob malveillant est un vecteur d'attaque (cf. CVE Envoy GHSA-m3p9-47wh-88wg : ratio d'inflation vérifié au mauvais niveau de boucle → OOM). `klauspost/compress` offre `WithDecoderMaxMemory` (godoc : *« control memory usage of potentially hostile content. Maximum is 1<<63 bytes. Default is 64GiB »* — **beaucoup trop haut, à réduire**), `WithDecodeAllCapLimit`, `WithDecoderMaxWindow`. Il faut aussi valider le `Frame_Content_Size` et borner via `io.LimitReader`.

## Details

### Axe 1 — Dictionnaire partagé : chiffres, taille, formation, versionnage

**Gain de ratio (publié).** Man page zstd : *« Typical gains range from ~10% (at 64KB) to x5 better (at <1KB) »*. Wiki Facebook (tableau mesuré) : small JSON records (~300 o) ~×4,7 ; Mercurial events (~1,5 ko) ~×1,5 ; large JSON docs (~6 ko) ~×2,8, *« achieved without any speed loss, and even some faster decompression »*. Pour blunderDB (2–20 ko), attendez-vous à un gain réel plus proche de **×1,5 à ×3** sur le blob individuel, le gain relatif décroissant avec la taille.

**Preuve concrète en base.** `sqlite-zstd` (phiresky) applique zstd + dictionnaire entraîné par groupes de lignes. Logs publiés : 163,77 Mo → 2,12 Mo (moyenne 24,33 ko/ligne → 315 o, **~×77**) ; 69,28 Mo → 1,60 Mo (~×43) ; 91,97 Mo → 1,41 Mo (~×65). README : réduction typique *« by 80% »* ; billet de blog : *« reduce the size of your database by 50 to 95% »*. C'est le cas d'usage exact de blunderDB, mais l'extension est en Rust/cgo, inutilisable avec `modernc.org/sqlite`.

**Taille de dictionnaire optimale.** Défaut du CLI : la man page confirme *« --maxdict # limit dictionary to specified size (default : 112640) »* (110 ko) ; la doc ZDICT recommande *« a reasonable dictionary has a size of ~ 100 KB »*. `--maxdict` n'est pas qu'une borne supérieure — l'issue #4127 montre une sensibilité extrême (512 ko → ×90, 550 ko → ×14 sur un cas). Pour blunderDB, testez 16 ko, 32 ko, 64 ko, 110 ko.

**Nombre d'échantillons.** Doc officielle : *« provide a few thousands samples »* et *« total size of all samples be about ~x100 times the target size of dictionary »* (man page : *« for example, 10 MB for a 100 KB dictionary »*). Le CLI exige > 100 fichiers ; l'entraînement échoue si trop peu d'échantillons ou si la plupart font < 8 octets. blunderDB (50 000–500 000 analyses) a largement de quoi.

**Coût CPU de la formation.** `ZDICT_trainFromBuffer` consomme ~6 Mo de RAM ; c'est **hors ligne**, une fois, sans impact utilisateur. Algorithmes : `--train` = `--train-fastcover=d=8,steps=4` (rapide, défaut) ; `--train-cover` (plus lent, souvent meilleur) ; `--train-legacy` (sélectivité réglable). Sur un cas mesuré (issue #1572), fastCover a donné ×6,21 en 31 s vs ×5,18 en 102 s pour le trainFromBuffer par défaut.

**Versionnage du dictionnaire (point crucial).** Le frame zstd contient un champ **`Dictionary_ID`** (RFC 8878 §3.1.1.1) que le décodeur utilise pour vérifier le bon dictionnaire. IDs courts avantageux : < 256 = 1 octet, < 65536 = 2 octets. Plages réservées à éviter en distribution publique (≤ 32767 et ≥ 2³¹) ; en usage privé, tout ID convient. **Pratique recommandée** (calquée sur sqlite-zstd) : stocker chaque dictionnaire dans une table `_zstd_dicts(id, dict_blob)` et **faire mémoriser à chaque ligne l'ID du dictionnaire utilisé**. On introduit ainsi un nouveau dictionnaire **sans recompresser toute la base** : anciennes lignes lues avec l'ancien dict, nouvelles avec le nouveau. `klauspost/compress` gère plusieurs dictionnaires via `WithDecoderDicts(dicts ...[]byte)`, sélectionnés automatiquement par ID.

**zlib SetDictionary (repli sans zstd).** `deflateSetDictionary` accepte un dictionnaire d'au plus **32 768 octets** (fenêtre DEFLATE 32 ko) ; au-delà, seul le sous-ensemble final est utilisé — d'où la consigne de placer les chaînes les plus fréquentes **à la fin**. L'en-tête zlib stocke l'**Adler-32 du dictionnaire entier** (champ DICTID si FDICT set, RFC 1950), et `inflateSetDictionary` renvoie `Z_DATA_ERROR` si l'Adler-32 ne correspond pas. En raw deflate, pas d'Adler-32. Sur des JSON de 2–20 ko, un dictionnaire zlib de 32 ko apporte un gain réel mais **inférieur à zstd+dict**. Avantage : `compress/flate` (stdlib) et `klauspost/compress/flate` supportent `SetDictionary` en Go pur, et l'output reste du DEFLATE lisible par d'anciennes versions si le conteneur est conservé.

### Axe 2 — Bibliothèques Go pures (sans cgo)

| Bibliothèque | Algo | Dictionnaire | Entraînement Go pur ? | Maturité / adoption |
|---|---|---|---|---|
| `klauspost/compress/zstd` | zstd | `WithEncoderDict`, `WithDecoderDicts`, `WithEncoderDictRaw` | Oui via sous-paquet `dict` (**expérimental**, algo maison) ; sinon `zstd --train` hors-ligne | STABLE, fuzz-testé en continu, utilisé par Docker/containerd/Kubernetes |
| `klauspost/compress/flate`,`/zlib`,`/gzip` | DEFLATE | `SetDictionary` (≤ 32 ko) | N/A (dict = buffer) | Drop-in stdlib, jusqu'à 3× plus rapide en compression haut niveau |
| `andybalholm/brotli` | Brotli (RFC 7932) | Dictionnaire statique intégré ; **custom dict retiré** en amont | Non | Traduit du C par c2go, API stable SemVer |
| `pierrec/lz4/v4` | LZ4 | `UncompressBlockWithDict`, dict via API bloc | Non (dict = buffer) | Go pur, décompression très rapide |
| `klauspost/compress/s2` | S2/Snappy | `MakeDict`/`NewDict` (peut réutiliser le contenu d'un dict zstd) | Partiel | Go pur, > 10 Go/s |
| `valyala/gozstd` | zstd | `BuildDict` (entraînement !) | **Non — cgo** (wrappe libzstd/ZDICT) | Mûr mais viole « pas de cgo » |

**Niveaux `klauspost/compress/zstd` :** `SpeedFastest` (~zstd niveau 1), `SpeedDefault` (~niveau 3), `SpeedBetterCompression`, `SpeedBestCompression`. Attention : le godoc indique *« SpeedBetterCompression will (in the future) yield better compression than the default... For now this is not implemented. SpeedBetterCompression = SpeedDefault »* (idem `SpeedBestCompression`). **Vérifiez impérativement l'état dans la version que vous figez** (l'implémentation « better/best » a évolué depuis). L'encodeur ne produit **pas** le même bitstream que le zstd C de référence (ne pas hasher la sortie pour comparer).

**Vitesse (ordres de grandeur publiés).** Corpus Silesia (lzbench, Facebook) : zstd -1 ratio 2,877 @ 330 Mo/s compression / 940 Mo/s décompression ; zlib -1 ratio 2,730 @ 95 / 360 Mo/s. Le zstd Go de klauspost est plus lent que le C mais du même ordre ; benchmark gzhttp : handler zstd 219 Mo/s vs gzip stdlib 137 Mo/s (2 ko), et jusqu'à ~10 Go/s en parallèle. Le décodeur zstd Go a reçu de l'assembleur amd64 (~2× plus rapide).

**Empreinte mémoire du décodeur (critique à 500 000 blobs).** Le décodeur zstd Go peut consommer beaucoup de RAM par instance et « lit en avance ». Bonnes pratiques : **créer UN décodeur (`zstd.NewReader(nil)`) et le réutiliser** via `DecodeAll` (conçu pour fonctionner sans allocation après warm-up, sûr en concurrence) ; limiter avec `WithDecoderConcurrency(1)` et/ou `WithDecoderLowmem(true)` ; idem encodeur avec `WithEncoderConcurrency(1)`. Un utilisateur rapporte 5× la mémoire attendue avec de nombreux décodeurs concurrents ; la réponse du mainteneur est de limiter la concurrence et réutiliser l'instance.

### Axe 3 — Alternatives structurelles

**Formats binaires (CBOR/MessagePack/Protobuf).** Bibliothèques Go : `fxamacker/cbor` (RFC 8949), `vmihailenco/msgpack` et `tinylib/msgp`. Gain **avant** compression : ~30–50 % vs JSON. Gain **après** compression : ~10–20 %, souvent moins, parfois négatif. Protobuf/Cap'n Proto/FlatBuffers descendent plus bas (schéma qui élimine les noms de champs) mais imposent un schéma des deux côtés et compliquent l'évolution — lourd pour un format de fichier échangeable. **Verdict : ne changez pas de format pour la taille.** Envisagez CBOR seulement si le CPU de sérialisation devient un goulet.

**Décomposition en colonnes scalaires.** Déjà fait dans blunderDB. Pousser plus loin (sortir les tableaux numériques répétitifs) réduit le blob mais complexifie le schéma. Comme les colonnes filtrées sont déjà externes, **le coût de garder un blob opaque est faible** : aucune requête ne le pénètre.

**Compression par page / extensions SQLite.** ZIPVFS (hwaci, commercial), `sqlite_zstd_vfs` (mlin), CEROD et `sqlite-zstd` (phiresky) offrent la compression transparente — mais **toutes sont des extensions C/Rust** que `modernc.org/sqlite` **ne peut pas charger**. En Go pur, il faut **compresser côté application** (ce que fait déjà blunderDB). L'architecture de sqlite-zstd (table de dictionnaires, ID par ligne, recompression incrémentale, `VACUUM` pour récupérer l'espace) est le **meilleur modèle de conception** à réimplémenter en Go.

**PostgreSQL.** Seuil TOAST ~2 ko ; un blob déjà compressé passe TOAST sans re-compression utile. `default_toast_compression` (PG14+) : pglz (~7 % plus dense) vs lz4 (plusieurs fois plus rapide) ; PG19 basculera sur lz4 par défaut. Aucun n'utilise de dictionnaire. **Pour blunderDB, zstd+dict côté application battra TOAST** et garantit un format identique sur les deux moteurs. Ne stockez pas en `jsonb` côté PG (perte du bénéfice du blob opaque + double stockage) : gardez `bytea` compressé applicativement, avec `STORAGE EXTERNAL` pour éviter une double compression pglz inutile.

### Axe 4 — Sécurité

**Bombes de décompression.** DEFLATE plafonne à ~1032:1 par couche ; zstd/brotli amplifient davantage via de grandes fenêtres. Risque : un blob (ou dictionnaire) forgé qui décompresse en gigaoctets → OOM/DoS. Cas réel : CVE Envoy (GHSA-m3p9-47wh-88wg) — le contrôle du ratio d'inflation était fait au mauvais niveau de boucle.

**Dictionnaire malveillant.** Le dictionnaire embarqué et les blobs d'un fichier partagé sont des entrées **non fiables**. Mitigations : ne charger que des dictionnaires dont l'ID est attendu ; borner la taille du dictionnaire ; refuser un `Dictionary_ID` inconnu.

**Bornes à poser côté Go.** (1) `WithDecoderMaxMemory(n)` — défaut 64 Gio, à **réduire** à une valeur réaliste (ex. 4–16 Mo, vos blobs décompressés faisant ≤ ~20–50 ko). (2) `WithDecodeAllCapLimit(true)` + `DecodeAll(src, make([]byte,0,maxOut))` pour plafonner la sortie. (3) `WithDecoderMaxWindow` pour rejeter les grandes fenêtres. (4) Vérifier le `Frame_Content_Size` de l'en-tête avant d'allouer. (5) En streaming, envelopper dans `io.LimitReader`. (6) Valider que la taille décompressée correspond à une taille attendue stockée en colonne scalaire.

**Fuzzing & CVE.** `klauspost/compress` est fuzz-testé en continu (OSS-Fuzz, native Go fuzzing) ; le zstd C de Facebook aussi. Des CVE mémoire historiques existent sur les décodeurs C (ex. c-blosc2 `ZSTD_createDDict` heap-overflow, CVE-2025-29476), brotli et zlib — d'où l'intérêt d'une implémentation **Go pure memory-safe** doublée de bornes explicites. Ajoutez vos propres tests de fuzzing Go (`FuzzDecompress`) sur le chemin de décompression.

### Axe 5 — Chemin de migration

**Octet de version / magic byte.** Préfixez chaque blob d'un tag de format : ex. `0x00`=zlib legacy, `0x01`=zstd sans dict, `0x02`=zstd+dict (+ ID de dict). À la lecture, lisez ce tag ; en secours, la détection par signature fonctionne (zlib niveau 9 commence par `0x78 0x9C`, zstd par `0x28 B5 2F FD` ; brotli n'a **pas** de magic number — argument de plus pour ne pas le choisir ici).

**Recompression progressive.** Recompressez en tâche de fond par **batchs** (1 000–5 000 lignes/transaction) pendant les périodes creuses (le modèle `zstd_incremental_maintenance` de phiresky prend un paramètre de charge DB pour ne pas bloquer les écritures). Les nouvelles écritures peuvent rester non compressées ou déjà au nouveau format ; seule la lecture doit gérer les deux formats.

**Taille de fichier & VACUUM (piège majeur SQLite).** Après recompression, SQLite **ne réduit pas** le fichier : les pages libérées vont dans la freelist et sont réutilisées. Il faut un **`VACUUM`** (réécrit toute la base) pour récupérer l'espace, OU `VACUUM INTO 'copie.db'` (copie compacte, utile pour migrer sans verrou long), OU activer `auto_vacuum` (mais fragmente et doit être décidé à la création). En mode **WAL**, `VACUUM`/`incremental_vacuum` ne rétrécit le fichier qu'après un **checkpoint** ; le WAL peut gonfler jusqu'à la taille de la base — prévoir ~200 % d'espace disque transitoire. Bornez le WAL avec `PRAGMA journal_size_limit`.

**Compatibilité descendante.** Utilisez `PRAGMA user_version` (entier libre) et/ou `PRAGMA application_id` pour taguer le format. Stratégies, de la moins à la plus intrusive :
- **Additive/compatible** : ajouter `_zstd_dicts` et un octet de tag par blob **sans changer le schéma existant**. Une ancienne version qui ignore le tag lira mal les nouveaux blobs → risque. Donc, si le format de blob change, **incrémentez `user_version`** et faites l'ancienne version **refuser proprement** d'ouvrir une base de version supérieure (message clair plutôt que corruption silencieuse).
- **Tag autodescriptif vs colonne « format »** : préférez le tag dans le blob, qui voyage avec la donnée et survit aux copies partielles.
- **Idéal** : conserver la capacité de **lire les deux formats** (zlib legacy + zstd) indéfiniment côté logiciel récent, et n'imposer la version minimale que si l'utilisateur exécute explicitement la migration destructive.

**Pièges à retenir.** (1) Ne pas oublier le `VACUUM`. (2) Le dictionnaire DOIT être stocké dans la base (sinon les blobs deviennent illisibles s'il est perdu). (3) Ne jamais réutiliser un `Dictionary_ID` pour un dict différent. (4) Tester qu'une base migrée ouverte par une ancienne version échoue proprement. (5) Le décodeur zstd doit connaître **tous** les dictionnaires jamais utilisés dans la base.

## Recommandations

**Étape 0 — Mesurer avant de décider (obligatoire).** Constituez un échantillon de **10 000 blobs** représentatifs. Faites un **split train/test** : entraînez le dictionnaire sur ~8 000 blobs, mesurez le ratio uniquement sur les ~2 000 restants (jamais vus), sinon vous **surestimez** le gain. Entraînez hors-ligne avec `zstd --train` (ou `builddict` de klauspost pour rester 100 % Go, en validant la qualité contre `zstd --train`). Testez tailles de dict {16, 32, 64, 110} ko et niveaux zstd {3, 9, 19}.

**Étape 1 — Décision par défaut recommandée :**
- **Format de sérialisation** : garder JSON (ou CBOR via `fxamacker/cbor` si le CPU de (dé)sérialisation devient limitant — gain de taille négligeable après compression).
- **Bibliothèque** : `github.com/klauspost/compress/zstd` (Go pur, cgo-free, mûr, fuzz-testé, mémoire-sûr).
- **Dictionnaire** : OUI, un dictionnaire zstd ~32–110 ko entraîné hors-ligne, stocké dans une table `_zstd_dicts`, versionné par `Dictionary_ID`, chaque ligne référençant son ID.
- **Niveau** : **zstd 19** pour la compression (analyses écrites une fois, lues souvent ; compression lente acceptable ; décompression rapide quel que soit le niveau). Repli sur un niveau intermédiaire si l'insertion massive est trop lente.
- **Gains attendus estimés** : table `analysis` réduite de **~50–80 %** vs zlib-9 ; fichier global réduit d'autant après `VACUUM`. Latence de décompression du même ordre ou meilleure que zlib. Ce sont des **estimations** à confirmer par le protocole.

**Étape 2 — Implémenter la migration** (tag d'octet, double lecture, recompression par batchs, `VACUUM INTO`, `user_version`).

**Étape 3 — Sécuriser** : `WithDecoderMaxMemory` réduit (~16 Mo), `WithDecodeAllCapLimit`, décodeur unique réutilisé avec `WithDecoderConcurrency(1)`, validation de taille, fuzzing du chemin de décompression.

**Seuils qui changeraient la reco :**
- Gain mesuré du dictionnaire sur le set de test **< 15 %** vs zstd sans dict → abandonnez le dictionnaire, gardez zstd-19 simple.
- Dict Go pur (`builddict`) **> 20 % moins bon** que `zstd --train` → entraînez hors-ligne avec le CLI C, livrez le `.dict` comme asset (aucun cgo au runtime).
- Latence p99 de décompression au-dessus du budget UI → baissez le niveau ou envisagez LZ4 (décompression ultra-rapide, ratio moindre).
- CPU d'insertion massive qui explose à zstd-19 → niveau intermédiaire (9).

## Protocole de mesure reproductible (10 000 blobs)

**Constitution de l'échantillon.** Exporter 10 000 blobs décompressés représentatifs (tailles 2–20 ko variées). Split déterministe (seed fixe) : 80 % train / 20 % test. **Entraîner le dictionnaire uniquement sur le train ; mesurer le ratio uniquement sur le test.** Répéter avec plusieurs seeds (5 folds) pour un ratio moyen robuste.

**Ce qu'on mesure.**
- Taille : somme des tailles compressées sur le test ; ratio = taille_brute/taille_compressée, pour chaque combinaison {algo, niveau, dict oui/non, taille dict}.
- Latence : p50/p95/p99 de compression et décompression **par blob** (histogramme sur les 2 000 blobs test, ≥ 10 répétitions chacun).
- Mémoire : allocations/op via `-benchmem` (B/op, allocs/op) ; RSS pic pour un décodeur unique réutilisé vs par-opération.
- Taille fichier réelle : construire une base SQLite de test, insérer les blobs recompressés, **exécuter `VACUUM`**, mesurer la taille du `.db` (métrique qui compte, pas la somme des blobs).

**Harnais Go.** Écrire des `func BenchmarkX(b *testing.B)` : un bench par combinaison, avec `b.ReportAllocs()`, `b.SetBytes(len(blob))` (pour obtenir des Mo/s), et un décodeur/encodeur **réutilisé** (créé hors de la boucle). Lancer avec `go test -bench=. -benchmem -count=10` et agréger avec **`benchstat`** (moyenne, variance, significativité entre variantes). Pour les percentiles (que `testing.B` ne donne pas), instrumenter une boucle séparée qui enregistre chaque durée et calcule p50/p95/p99.

**Répétitions.** `-count=10` minimum par bench ; ≥ 10 mesures de latence par blob ; ≥ 3 seeds de split. Machine au repos, `GOMAXPROCS` fixé, fréquence CPU épinglée si possible.

## Caveats
- Les gains **×2–×5** et **~×4,7 sur small JSON** sont des chiffres **publiés par Facebook/zstd** sur *leurs* corpus ; vos blobs de 2–20 ko sont plus gros que les 300 o du benchmark le plus favorable, donc **attendez-vous à un gain réel plus modeste** (estimation ×1,5–×3 par blob, 50–80 % sur la base). À confirmer par le protocole.
- Les **80 % / 50–95 %** de `sqlite-zstd` et **~×77** sur certaines lignes sont mesurés sur les données de phiresky, pas les vôtres : preuve de concept, pas garantie.
- L'entraîneur de dictionnaire **Go pur** (`klauspost/compress/dict`) est **expérimental** et n'utilise pas l'algorithme COVER de référence — validez sa qualité ; en cas de doute, entraînez hors-ligne avec le CLI C (sans cgo au runtime).
- Les chiffres pglz/LZ4 et le basculement PG19 vers LZ4 proviennent de blogs d'éditeurs (Crunchy Data, The Build) et d'un commit récent ; le comportement exact dépend de vos données.
- Les niveaux `SpeedBetterCompression`/`SpeedBestCompression` de klauspost ont été historiquement des alias de `SpeedDefault` (*« For now this is not implemented »*) ; vérifiez l'état exact dans la version figée.
- Distinction mesuré vs estimé : tout ratio « attendu » pour blunderDB est une **estimation** ; seuls les chiffres attribués à une source (man page zstd, wiki Facebook, logs sqlite-zstd, benchmarks Silesia/gzhttp) sont **mesurés/publiés**.