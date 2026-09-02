# Conception d'un filigrane signé Ed25519 lié au contenu d'une base SQLite et authentification d'en-tête AEAD — étude technique pour blunderDB

## TL;DR
- Le défaut central (signature ne couvrant aucun digest de contenu) doit être corrigé en signant un **document canonique unique = digest du contenu ‖ métadonnées**, encodé sans ambiguïté via un schéma longueur-préfixé (DSSE/PAE ou CBOR déterministe), jamais du JSON re-sérialisé ; en parallèle, l'en-tête du conteneur AEAD doit être **intégralement passé en AAD**, faute de quoi les paramètres Argon2id, le sel et la version restent falsifiables (downgrade).
- Pour le digest, deux stratégies complémentaires : (a) pragmatique et recommandée par défaut — **hacher le flux d'octets de l'export produit par `VACUUM INTO`** (fichier traité comme immuable) avec BLAKE3/SHA-256 ; (b) robuste aux changements de représentation — un **digest logique** de type `dbhash`/`.sha3sum` mais reconstruit avec un encodage strictement injectif (longueur-préfixé) et un `ORDER BY` explicite.
- Aucune de ces mesures n'empêche la copie du contenu ni un détenteur de la clé privée de re-signer ; une signature prouve **authenticité + intégrité**, pas la non-copie. Pour la traçabilité des fuites, il faut un mécanisme orthogonal (fingerprinting/watermark robuste par destinataire, littérature Agrawal-Kiernan). Le modèle de menace doit être explicité avant de choisir.

## Key Findings

1. **Le « watermark transplantation » est un défaut de liaison (binding), pas de signature.** La signature Ed25519 est valide mais ne lie rien au contenu. La correction standard, tirée de C2PA, est le *hard binding* : inclure dans le message signé un digest cryptographique du contenu. La C2PA Technical Specification (ch. « Binding to Content ») définit textuellement le *hard binding* comme « A hard binding (also known as a cryptographic binding) enables the validator to ensure that (a) this manifest belongs with this asset and (b) that the asset has not been modified, by determining values that can match only this asset and no other, not even other assets derived from it », par opposition au *soft binding* qui « is computed from the digital content of an asset, rather than its raw bits » (hash perceptuel / watermark, survit aux transformations). La spec définit d'ailleurs plusieurs types d'assertions de hard binding (data hash / byte-range, BMFF, collection hash). Ce vocabulaire est directement applicable ici.

2. **Ne jamais signer du JSON re-sérialisé.** JWS/JWT sont déconseillés (attaques de confusion d'algorithme via l'en-tête `alg` contrôlé par l'attaquant, y compris `alg:none` et RS256→HS256). La bonne pratique (JWS, PASETO, DSSE) est de signer des **octets exacts** encodés. DSSE fournit exactement l'outil : le PAE (Pre-Authentication Encoding) `DSSEv1 SP LEN(type) SP type SP LEN(body) SP body`, longueur-préfixé et donc non ambigu pour tout payload binaire.

3. **`dbhash` et `.sha3sum` existent mais ont des sémantiques différentes et des limites documentées.** `dbhash` (SHA-1) et `.sha3sum`/`sha3_query()` (SHA3-256 par défaut) hachent le *contenu logique* — invariants sous VACUUM, page_size, encodage UTF-8/16, auto_vacuum, REINDEX, ANALYZE. Mais **aucun n'émet d'`ORDER BY` explicite** : ils reposent sur un comportement « historique mais non garanti par la doc » de SQLite (ordre PK/rowid). L'encodage de `dbhash` n'est pas longueur-préfixé pour TEXT/BLOB (risque théorique d'ambiguïté), tandis que `sha3_query()` corrige cela avec des préfixes de longueur `T<len>:` / `B<len>:` et un marqueur de ligne `R`.

4. **AES-GCM n'authentifie que le ciphertext et l'AAD.** Tout champ d'en-tête hors AAD (paramètres KDF, sel, nonce, version) est malléable. Concrètement : un attaquant peut abaisser `m`/`t`/`p` d'Argon2id pour rendre le brute-force du mot de passe réalisable, substituer le sel, ou faire un rollback de version — sans invalider le tag GCM. **Correction : passer tout l'en-tête en AAD** (approche age, qui applique un HMAC-SHA-256 sur tout l'en-tête).

5. **Le chiffrement en flux exige un construct dédié (STREAM).** Charger 2 Go en mémoire est à proscrire ; mais un simple découpage naïf ouvre des attaques de troncature (dernier chunk non marqué), de réordonnancement (numéro de chunk absent du nonce) et de splicing entre fichiers. Le construct STREAM de Viet Tung Hoang, Reza Reyhanitabar, Phillip Rogaway et Damian Vizár (« Online Authenticated-Encryption and its Nonce-Reuse Misuse-Resistance », CRYPTO 2015, LNCS vol. 9215, pp. 493–517, doi:10.1007/978-3-662-47989-6_24 ; ePrint 2015/189), implémenté par Tink (`streamingaead`) et age (ChaCha20-Poly1305, chunks de 64 KiB, compteur 11 octets big-endian + 1 octet drapeau de dernier bloc), résout ces trois attaques.

6. **Précédent le plus proche du besoin : minisign « trusted comment ».** minisign signe le fichier (ou son hash BLAKE2b-512 en mode pré-hashé), puis produit une **seconde signature globale sur `signature ‖ trusted_comment`** — c'est exactement le mécanisme « métadonnées authentifiées liées au contenu » recherché. À noter un piège documenté : en mode pré-hashé, l'absence de séparation de domaine entre `Ed`/`ED` permet une forge ciblée ; il faut donc inclure explicitement un identifiant de format/domaine dans ce qui est signé.

## Details

### AXE 1 — Digest canonique d'une base SQLite

**Objectif.** Obtenir un hash du *contenu logique* (tables, lignes) indépendant de la représentation physique : espace libre/freelist, ordre d'insertion, effets de VACUUM, change counter, `application_id`, `user_version`, mode journal/WAL, `page_size`, encodage.

**Outils officiels existants.**
- `dbhash` (utilitaire autonome, `tool/dbhash.c`, SHA-1). Documentation : le hash est « computed over just the content of the database. Free space inside of the database file, and alternative on-disk representations of the same content (ex: UTF8 vs UTF16) do not affect the hash ». Il est invariant sous VACUUM, PRAGMA page_size, PRAGMA journal_mode, REINDEX, ANALYZE, copie via backup API. Les tables système `sqlite_stat1`, `sqlite_stat4`, `sqlite_sequence` (et tout `sqlite_%`) sont omises du contenu.
- `.sha3sum` (commande shell) et sa fonction sous-jacente `sha3_query()` (extension `ext/misc/shathree.c`, SHA3-256 par défaut, options `--sha3-224/256/384/512`). Le schéma (`sqlite_schema`) n'est pas inclus par défaut ; option `--schema` pour l'ajouter ; un argument = motif LIKE de filtrage des tables.

**Sémantique interne exacte de `dbhash.c` (source master).**
- Énumération des tables : `SELECT name FROM sqlite_schema WHERE type='table' AND sql NOT LIKE 'CREATE VIRTUAL%' AND name NOT LIKE 'sqlite_%' AND name LIKE '%q' ORDER BY name COLLATE nocase`. Les tables VIRTUAL et `sqlite_%` sont exclues.
- Parcours des lignes : littéralement `SELECT * FROM "table"` — **sans ORDER BY**. Commentaire du code source : « We want rows of the table to be hashed in PRIMARY KEY order. Technically, an ORDER BY clause is required to guarantee that order. However, though not guaranteed by the documentation, every historical version of SQLite has always output rows in PRIMARY KEY order when there is no WHERE or GROUP BY clause, so the ORDER BY can be safely omitted. » C'est l'hypothèse de correction majeure et le principal point d'attention pour une base WITHOUT ROWID ou multi-colonnes PK.
- Encodage des valeurs (préfixe de type ASCII 1 octet + valeur) : NULL → `"0"` ; INTEGER → `"1"` + 8 octets big-endian ; REAL → `"2"` + 8 octets big-endian (motif IEEE-754) ; TEXT → `"3"` + octets UTF-8 bruts **sans préfixe de longueur** ; BLOB → `"4"` + octets bruts **sans préfixe de longueur**. La discrimination de type utilise `sqlite3_column_type()` (classe de stockage réelle), donc l'INTEGER `1` et le TEXT `'1'` hachent différemment.
- Schéma haché par défaut via `SELECT type, name, tbl_name, sql FROM sqlite_schema WHERE tbl_name LIKE '%q' ORDER BY name COLLATE nocase` ; options `--schema-only`, `--without-schema`, `--like` (à noter : `dbhash` n'a pas d'option `--schema` — celle-ci appartient à `.sha3sum`).

**Sémantique interne exacte de `sha3_query()` (`shathree.c`).** Grammaire du flux haché : `S<n>:<sql>` (texte SQL de chaque instruction), puis pour chaque ligne un octet `R`, puis pour chaque colonne : NULL → `N` ; INTEGER → `I` + 8 octets ; REAL → `F` + 8 octets ; TEXT → `T<size>:` + UTF-8 ; BLOB → `B<size>:` + octets. Le `:` final du préfixe est « needed to separate the prefix from the content in cases where the content starts with a digit ». **Ce schéma est donc auto-délimité (injectif)** grâce aux préfixes de longueur et au marqueur de ligne — supérieur à `dbhash` sur ce point. À noter une contradiction dans les commentaires source (le commentaire de `sha3_query` dit « little-endian » alors que le code, le commentaire de `sha3_agg` et les vecteurs de test montrent du **big-endian**) : le code fait foi (big-endian).

**Pièges de canonicalisation à traiter dans une implémentation maison.**
- **Ordre des lignes** : ne PAS reposer sur l'ordre naturel ; émettre un `ORDER BY` explicite sur la PK (ou sur le rowid pour les tables rowid, ou sur toutes les colonnes triées pour les tables sans PK stable). C'est la seule façon d'avoir un digest réellement canonique et portable entre versions de SQLite.
- **Encodage injectif** : préférer le schéma longueur-préfixé de `sha3_query` (ou un TLV/DSSE-PAE), jamais une simple concaténation. Une concaténation naïve `"a"+"bc"` vs `"ab"+"c"` est ambiguë.
- **Affinités et classes de stockage** : hacher la classe de stockage effective (`sqlite3_column_type`) avec un tag de type, pour distinguer `1` (INTEGER) de `'1'` (TEXT).
- **Tables WITHOUT ROWID, colonnes, collations** : fixer l'ordre des colonnes (ordre du schéma) et l'ordre des tables (par nom). Décider explicitement du sort des vues, index, triggers, `sqlite_sequence`, `sqlite_stat1` : par défaut, comme `dbhash`, exclure les `sqlite_%` du contenu ; inclure ou non le schéma est un choix de politique (recommandé : inclure le schéma pour empêcher une altération de structure).

**Coût / performance.**
- Un SHA-256 du fichier entier est limité par le débit du hash et de l'I/O. Avec SHA-NI (crypto/sha256 en Go), l'ordre de grandeur est de plusieurs centaines de Mo/s à ~1-2 Go/s ; BLAKE3 (zeebo/blake3) est nettement plus rapide (multi-Go/s, parallélisable). Un fichier de 500 Mo se hache donc en une fraction de seconde à ~1 s ; 2 Go en quelques secondes, dominé par l'I/O disque.
- Un **digest logique** est bien plus coûteux : il faut exécuter `SELECT *` sur toutes les tables via le moteur SQLite (désérialisation des pages, décodage des enregistrements, tri). L'ordre de grandeur est typiquement plusieurs fois le coût d'un simple hash de fichier (souvent 5-20×), car la lecture ligne-à-ligne via l'API SQLite est bien plus lente que le hachage séquentiel d'un flux d'octets. Sur 500 Mo–2 Go cela peut représenter plusieurs secondes à dizaines de secondes.

**L'alternative pragmatique : signer l'export au moment où il est produit.** Puisque blunderDB produit l'export (probablement via `VACUUM INTO`), le plus simple et le plus robuste est de traiter ce fichier de sortie comme **immuable** et de hacher son flux d'octets. Question clé : `VACUUM INTO` produit-il un fichier bit-à-bit reproductible ? La documentation SQLite décrit `VACUUM`/`VACUUM INTO` comme recopiant tout le contenu dans un nouveau fichier compact. En pratique, la sortie est déterministe **à contenu et paramètres identiques**, mais plusieurs éléments varient : le **change counter** (octets 24-27 de l'en-tête, incrémenté à chaque transaction d'écriture), le champ version-valid-for (octets 92-95), éventuellement le numéro de version SQLite qui a écrit le fichier (octets 96-99). Il n'y a pas de timestamp interne dans le format de fichier SQLite. **Conséquence pratique** : deux `VACUUM INTO` successifs de la même base logique peuvent différer de quelques octets d'en-tête. Donc, pour un digest de fichier stable, soit on hache le fichier tel que produit une seule fois (et on le fige), soit on **exclut/normalise** les quelques octets d'en-tête volatils avant de hacher — approche « ByteRange » analogue aux zones exclues des PDF signés / du bloc de signature APK.

**Approche hybride recommandée.** Signer un document contenant **les deux** : `digest_fichier` (rapide, du flux d'octets de l'export figé, avec exclusion documentée des octets d'en-tête volatils) **et** optionnellement `digest_logique` (pour vérifier l'équivalence de contenu après une transformation légitime type VACUUM). Pour permettre une vérification partielle sur les très grosses bases, un **hash de Merkle par table** (une feuille = digest logique d'une table, racine = hash des feuilles ordonnées) est envisageable, à l'image du hachage par chunks de l'APK Signature Scheme v2 (chunks de 1 Mo, hash à deux niveaux) ou de fs-verity (APK v4).

### AXE 2 — Schéma de signature

**Signer (digest ‖ métadonnées) comme un document canonique unique.** Le message signé doit contenir, au minimum : version de format, digest(s) et leur algorithme, origine, nom d'émetteur, note, date, algorithme de signature, key id. Options d'encodage :

| Option | Avantages | Inconvénients / pièges |
|---|---|---|
| **JSON canonique (RFC 8785 / JCS)** | lisible, interopérable | Pièges nombreux : sérialisation des nombres IEEE 754 façon ECMAScript, tri des clés par unités de code UTF-16, NaN/Infinity interdits, I-JSON requis. Risque « signer du JSON re-sérialisé ». |
| **CBOR déterministe (RFC 8949 §4.2 / dCBOR)** | compact, binaire, règles déterministes | il faut un encodeur conforme ; dCBOR (draft) resserre les choix laissés ouverts par 8949. |
| **COSE_Sign1 (RFC 9052/9053)** | standard IETF, Ed25519 supporté, structure claire (Sig_structure) | dépendance CBOR/COSE ; plus lourd. |
| **PASETO v4.public (Ed25519)** | anti-confusion d'algorithme par conception | format opinioné, moins courant en Go pour ce cas. |
| **JWS/JWT** | ubiquitaire | **déconseillé** : confusion d'algorithme (`alg` dans l'en-tête contrôlé par l'attaquant, `alg:none`, RS256→HS256). |
| **DSSE + PAE** | résout la canonicalisation par un encodage longueur-préfixé ; agnostique au payload | pas de structure de payload imposée (à définir). **Recommandé.** |

**DSSE / PAE — le cœur de la solution.** `PAE(type, body) = "DSSEv1" SP LEN(type) SP type SP LEN(body) SP body`, où LEN est l'entier décimal ASCII de la longueur en octets. On signe `PAE(payloadType, payload)`, jamais le JSON de l'enveloppe. Cela élimine par construction toute ambiguïté de canonicalisation (le problème que in-toto/TUF réglaient auparavant par une canonicalisation JSON fragile). Bibliothèque Go officielle : `github.com/secure-systems-lab/go-securesystemslib/dsse` (fonctions `PAE`, `EnvelopeSigner`, `SHA256KeyID`).

**Séparation de domaine et champs obligatoires dans le message signé.**
- Inclure un **préfixe de domaine / context string** propre à blunderDB (ex. `payloadType = "application/vnd.blunderdb.watermark+json;v=2"`) pour éviter les attaques cross-protocol. Le cas minisign illustre le danger inverse : en mode pré-hashé, comme `Ed`/`ED` n'est jamais signé, il n'y a pas de séparation de domaine et une forge ciblée devient possible.
- **Inclure impérativement dans les octets signés** : le numéro de version de format, l'algorithme, le key id. Ne jamais laisser l'algorithme être choisi par un champ non authentifié (leçon JWS).

**Gestion des versions et de l'extensibilité.**
- Champ `v` en tête. **Rejet strict des versions inconnues** (fail closed) plutôt que tolérance.
- Dans un contexte signé, **rejeter les champs inconnus** (ne pas les ignorer) : un champ non reconnu peut être un vecteur de contournement ; le comportement « ignorer les champs inconnus » est acceptable pour l'interopérabilité mais dangereux pour la sécurité.
- Migration v1→v2 : supporter la lecture v1 en lecture seule / avec avertissement, écrire uniquement v2.

**Sémantique de vérification après modification de la base.** Trois modèles, à choisir selon le produit :
- (a) **fail closed** : le filigrane devient invalide dès toute modification (comportement naturel d'un hard binding). Simple et sûr.
- (b) **« attesté à la date X sur le contenu Y »** : si le contenu actuel diffère du digest signé, afficher « dérivé de / provenance : … » plutôt qu'« invalide ». C'est le modèle C2PA (manifeste actif + `ingredient` référençant les manifestes antérieurs, formant une chaîne de provenance).
- (c) **chaîne de provenance / lignage** complète (in-toto attestations, SLSA provenance). Plus lourd, pertinent si blunderDB veut tracer une généalogie d'éditions.

**Où stocker la signature — le problème d'auto-référence.** Si la signature est dans la table `metadata`, le digest ne peut pas couvrir la table qui contient le digest. Solutions, par ordre de clarté :
1. **Signature détachée** (fichier `.blunderdbsig` à côté de l'export), à la minisign/age. Le plus simple et sans ambiguïté.
2. **Exclusion explicite** de la table `metadata` (ou seulement des lignes de signature) du calcul du digest, avec une règle non ambiguë et versionnée. Précédents de « trou de signature » : le *signature hole* d'Authenticode (PE), le `ByteRange` des PDF signés, la zone exclue (APK Signing Block) de l'APK Signature Scheme v2/v3 (« The hash covers everything except the APK signing block »). L'important est que la zone exclue soit définie de manière déterministe et elle-même bornée, sinon on rouvre une malléabilité.

**Révocation, rotation, distribution, transparence, horodatage.**
- Distribution de la clé publique de l'émetteur : embarquée dans l'app (clé de confiance), ou liste de confiance type C2PA Trust List.
- Rotation : prévoir un key id dans chaque signature (comme APK v3 gère la rotation via proof-of-rotation).
- Transparence / horodatage : Rekor (Sigstore transparency log), horodatage RFC 3161. Le bundle Sigstore (`application/vnd.dev.sigstore.bundle.v0.3+json`, `github.com/sigstore/sigstore-go`) lie signature, certificat, entrée de log et timestamps, et supporte à la fois `messageSignature` (sur un digest d'artefact) et `dsseEnvelope`. Réutilisable si blunderDB veut une provenance vérifiable en ligne, sinon surdimensionné.

### AXE 3 — AEAD et en-tête

**Passer l'en-tête en additionalData (AAD).** AES-GCM (comme tout AEAD) n'authentifie QUE le ciphertext et l'AAD. Tout ce qui est en clair hors AAD (version, filigrane, sel, nonce, paramètres Argon2id) est **non authentifié et malléable**. Attaques concrètes si l'en-tête n'est pas en AAD :
- **Downgrade Argon2id** : abaisser `m`, `t`, `p` pour rendre le brute-force du mot de passe réalisable. C'est l'attaque la plus grave ici.
- **Substitution du sel** / **rollback de version**.

Correction : mettre **tout l'en-tête** (version ‖ filigrane ‖ paramètres KDF ‖ sel ‖ nonce ‖ longueurs) en AAD du premier bloc AEAD. Précédent de référence : age. La spécification age v1.1.0 (c2sp.org/age@v1.1.0) prescrit textuellement : « The final header line starts with `---` and is followed after a space by the base64-encoded MAC of the header. The MAC is computed with HMAC-SHA-256 (see RFC 2104) over the whole header up to and including the `---` mark (excluding the space following it) ». La clé HMAC est dérivée par HKDF-SHA-256 (ikm = file key, salt = none, info = "header"), précisément pour empêcher l'ajout/retrait de destinataires et l'altération des paramètres.

**Paramètres Argon2id à stocker et à borner.**
- Stocker : variante (id), version d'Argon (0x13), `m` (mémoire KiB), `t` (itérations), `p` (parallélisme), longueur du sel, longueur de sortie.
- Valeurs par défaut recommandées (OWASP Password Storage Cheat Sheet). L'OWASP donne **deux profils équivalents** (et non des seuils « min » vs « courants ») : verbatim « m=47104 (46 MiB), t=1, p=1 (Do not use with Argon2i) » et « m=19456 (19 MiB), t=2, p=1 (Do not use with Argon2i) », avec la précision « These configuration settings provide an equal level of defense, and the only difference is a trade off between CPU and RAM usage ». Pour du chiffrement de fichier local (pas de contrainte de latence serveur), on peut viser bien plus haut (128-256 MiB), conformément à RFC 9106.
- **Bornes minimales à la lecture** : refuser des paramètres sous le seuil (ex. `m<19 MiB` ou `t<2`) pour empêcher un downgrade — même avec l'AAD, un fichier légitimement produit avec des paramètres faibles reste faible ; l'AAD empêche la falsification mais pas la faiblesse d'origine, d'où la double protection.
- **Bornes maximales** : refuser un `m` absurde (ex. > 1-2 GiB) pour éviter un DoS mémoire : un en-tête hostile demandant 16 GiB de RAM ferait planter l'application avant même le déchiffrement.

**Bornes de taille avant allocation.** Ne jamais faire `make([]byte, n)` avec `n` lu depuis l'en-tête sans validation (memory exhaustion). Utiliser `io.LimitReader`, valider toutes les longueurs déclarées contre des bornes strictes avant allocation.

**Gestion des nonces.** AES-GCM : nonce 96 bits. Avec nonce **aléatoire**, la limite de sécurité (birthday bound) est d'environ 2³² messages par clé avant risque de collision. Le nonce doit être unique par clé ; réutiliser (clé dérivée du même sel) + (même nonce) est **catastrophique** (perte totale de confidentialité et d'authenticité GCM). Comme la clé est dérivée par Argon2id d'un sel aléatoire par fichier, le risque est maîtrisé si sel ET nonce sont tirés d'un CSPRNG à chaque `seal`.

**Déchiffrement en flux / chunking (ne pas charger 2 Go).** Utiliser le construct **STREAM** (Hoang-Reyhanitabar-Rogaway-Vizár, CRYPTO 2015, ePrint 2015/189). Pièges et parades :
- **Truncation attack** → marquer le dernier chunk (age : 1 octet de drapeau 0x00/0x01 dans le nonce ; rejet désormais du dernier chunk vide, cf. C2SP/C2SP#13).
- **Reordering attack** → inclure le numéro de chunk (compteur) dans le nonce.
- **Chunk splicing entre fichiers** → dériver une clé de flux par fichier (age : payload key = HKDF-SHA-256(file key, nonce 16 octets, "payload"), sortie 32 octets).

Implémentations réutilisables en Go : `github.com/google/tink/go/streamingaead` (AES-256-GCM-HKDF-STREAMING ou AES-CTR-HMAC-STREAMING) ; le format age (`filippo.io/age`, ChaCha20-Poly1305, chunks de 64 KiB) ; `github.com/mrknow-all/go-oae`. Taille de chunk recommandée : 64 KiB (age) à quelques centaines de Ko/1 Mo (compromis débit/mémoire).

**Comparaison des AEAD.**

| AEAD | Nonce | Nonce aléatoire sûr ? | Notes |
|---|---|---|---|
| AES-256-GCM | 96 bits | Limité (~2³² msg/clé) | Rapide avec AES-NI ; fragile au nonce reuse ; matériel requis pour la perf. |
| ChaCha20-Poly1305 | 96 bits | Limité | Rapide sans matériel, résistant aux timing cache ; choix d'age. |
| XChaCha20-Poly1305 | 192 bits | **Oui** (nonce aléatoire sûr) | Le nonce large rend le tirage aléatoire sans risque de collision. Bon défaut pour un conteneur. |

**Ordre des opérations : chiffrer-puis-signer ou signer-puis-chiffrer ?** Le principe de « cryptographic doom » (T. Ptacek) recommande de **vérifier l'authentification avant tout traitement** du texte chiffré ; avec un AEAD, l'authenticité de l'en-tête (via AAD) et du corps (via les tags) est vérifiée avant déchiffrement utile. Ici, le filigrane signé Ed25519 est dans l'en-tête **en clair** : c'est un choix défendable car il permet de **vérifier l'émetteur sans déchiffrer** (utile pour l'UI et le tri), au prix d'une fuite de métadonnées (nom d'émetteur, note, date visibles). Recommandation : signer le contenu (hard binding) ET inclure la signature dans l'en-tête en AAD ; ainsi l'en-tête signé en clair authentifie l'émetteur, et l'AAD empêche toute altération des paramètres.

### AXE 4 — Précédents étudiés

- **minisign (Frank Denis)** — le précédent le plus pertinent. Format `.minisig` : commentaire non fiable, puis `base64(signature_algorithm ‖ key_id ‖ signature)`, puis `trusted_comment`, puis `base64(global_signature)`. Mode pré-hashé (recommandé, « ED ») : `signature = ed25519(BLAKE2b-512(<file data>))` ; `global_signature = ed25519(signature ‖ trusted_comment)`. **Le trusted comment est authentifié par la signature globale** — mécanisme exact de « métadonnées liées au contenu ». Bibliothèques Go : `github.com/jedisct1/go-minisign`, `aead.dev/minisign`. Piège documenté (issue #104) : absence de séparation de domaine `Ed`/`ED` → forge ciblée possible ; leçon → inclure un tag de domaine/format dans les octets signés.
- **age (filippo.io/age)** — en-tête texte, HMAC-SHA-256 sur tout l'en-tête avec une clé dérivée (authentification d'en-tête exactement recherchée), stanzas de destinataires, recipient scrypt (mot de passe), payload en chunks 64 KiB ChaCha20-Poly1305 (STREAM). Spec : `c2sp.org/age@v1.1.0`. Filippo souligne (blog « age and Authenticated Encryption ») qu'age n'est PAS une AE asymétrique : quiconque a la file key peut réécrire le corps et recalculer le HMAC → une signature reste nécessaire pour l'authenticité de l'auteur.
- **DSSE / in-toto / SLSA** — PAE longueur-préfixé, enveloppe `{payload, payloadType, signatures[]}`, signature sur `PAE(payloadType, payload)`.
- **Sigstore bundle** — `application/vnd.dev.sigstore.bundle.v0.3+json`, protobuf, `messageSignature` (sur digest) ou `dsseEnvelope`, matériel de vérification (certificat, tlog Rekor, timestamps). `github.com/sigstore/sigstore-go`, `cosign verify-blob --new-bundle-format`.
- **C2PA (Content Credentials)** — hard binding (hash cryptographique, SHA-256 recommandé) vs soft binding (hash perceptuel/watermark), manifest store, manifeste actif, assertions, `ingredient` (chaîne de provenance), gestion de la modification ultérieure. Vocabulaire directement transposable.
- **APK Signature Scheme v2/v3/v4** — v2/v3 hachent tout le zip sauf le bloc de signature (zone exclue), chunks de 1 Mo, hash à deux niveaux ; v3 gère la rotation de clés ; v4 = arbre de Merkle (fs-verity), signature dans un fichier `.idsig` séparé. Anti-stripping : la v1 déclare que l'APK est v2-signé, ce qui empêche le downgrade silencieux.
- **JAR signing** — `MANIFEST.MF` avec un digest par entrée : précédent pour signer des parties d'un conteneur.
- **PDF signé (ByteRange)**, **Authenticode (signature hole)** — zones exclues explicites.
- **Anki `.apkg` / `.colpkg`** — zip contenant une base SQLite (`collection.anki2`/`.anki21`/`.anki21b` compressé Zstd) + médias (JSON de mapping). **Aucune signature ni filigrane** dans le format : la provenance des decks partagés sur AnkiWeb n'est pas cryptographiquement garantie. Les protections des decks payants (AnkiHub, decks commerciaux) reposent sur le compte/serveur, pas sur le format. → confirme qu'il n'existe pas de standard de signature de base SQLite embarquée à réutiliser tel quel.
- **Chessable** — distribution de cours d'échecs via plateforme/compte (DRM côté service, pas de format ouvert signé). Pas de watermark PGN standardisé public documenté. → même conclusion : la protection réelle du contenu d'échecs distribué est côté service, pas cryptographique dans le fichier.
- **Watermarking de bases relationnelles** — Rakesh Agrawal & Jerry Kiernan, « Watermarking Relational Databases », Proc. 28th VLDB, Hong Kong, 20 août 2002, pp. 155–166 ; version étendue : Rakesh Agrawal, Peter J. Haas & Jerry Kiernan, « Watermarking relational data: framework, algorithms and analysis », *The VLDB Journal* 12(2):157–169, 2003. La technique « ensures that some bit positions of some of the attributes of some of the tuples contain specific values » selon une clé secrète (MAC). Extensions de Li et al. pour du **fingerprinting** (marque = identifiant du destinataire, traçabilité des fuites, résistance aux attaques de collusion — Boneh-Shaw). **Distinction cruciale : signature = authenticité/intégrité (casse à l'édition) ; watermark robuste = traçabilité qui survit à l'édition.** Les deux sont orthogonaux et complémentaires.

### Conception recommandée (synthèse)

**Format du filigrane (payload signé).** Un document CBOR déterministe (ou JSON I-JSON strict) contenant : `v` (version format, entier), `alg` ("Ed25519"), `kid` (identifiant clé), `origin`, `issuer`, `note`, `date` (RFC 3339 UTC), `content` = { `file_digest` : {`alg`:"BLAKE3"|"SHA-256", `value`, `excluded_ranges` : liste des octets d'en-tête SQLite exclus}, `logical_digest` (optionnel) : {`alg`:"SHA3-256", `value`, `scheme`:"blunderdb-canonical-v1"} }. On signe `PAE("application/vnd.blunderdb.watermark+cbor;v=2", cbor_bytes)` avec `crypto/ed25519`. Enveloppe DSSE via `go-securesystemslib/dsse`.

**Stratégie de digest.** Par défaut : digest du **flux d'octets de l'export figé** (produit par `VACUUM INTO`), avec exclusion documentée et bornée des octets d'en-tête volatils (change counter offsets 24-27, version-valid-for 92-95, write/read version 18-19, SQLite version 96-99). Optionnellement : digest logique canonique reconstruit avec `ORDER BY <pk>` explicite, encodage longueur-préfixé façon `sha3_query`, exclusion des `sqlite_%`, schéma inclus.

**Stockage de la signature.** Détachée (fichier `.blunderdbsig`) de préférence ; si embarquée dans `metadata`, exclure de façon déterministe et versionnée les lignes de signature du digest logique (analogue ByteRange/APK signing block).

**Format du conteneur v2 (layout d'en-tête).** `magic (8) ‖ version (2) ‖ header_len (4) ‖ [kdf_id (1) ‖ argon_version (1) ‖ m (4) ‖ t (4) ‖ p (1) ‖ salt_len (2) ‖ salt ‖ nonce_len (1) ‖ base_nonce ‖ watermark_dsse_len (4) ‖ watermark_dsse] ‖ tag_chunks...`. **Tout l'en-tête (du magic à la fin de watermark_dsse) est passé en AAD du premier chunk** (ou couvert par un HMAC dédié façon age). Corps chiffré en STREAM (XChaCha20-Poly1305 recommandé pour le nonce large, ou AES-256-GCM/ChaCha20-Poly1305), chunks de 64 KiB, compteur + drapeau de dernier bloc dans le nonce, clé de flux dérivée par HKDF du secret Argon2id + nonce de base.

**Pseudo-code Go idiomatique (4 chemins).**

```go
// --- SIGN : lier le contenu et signer les métadonnées ---
func Sign(exportPath string, meta Meta, priv ed25519.PrivateKey, kid string) ([]byte, error) {
    // 1. digest du flux d'octets, en excluant les octets d'en-tête volatils
    fd, err := fileDigestBLAKE3(exportPath, sqliteVolatileRanges) // 24-27, 92-95, 18-19, 96-99
    if err != nil { return nil, err }
    // 2. payload canonique (CBOR déterministe)
    payload := cborDet(Watermark{
        V: 2, Alg: "Ed25519", Kid: kid,
        Origin: meta.Origin, Issuer: meta.Issuer, Note: meta.Note,
        Date: time.Now().UTC().Format(time.RFC3339),
        Content: Content{FileDigest: Digest{Alg: "BLAKE3", Value: fd}},
    })
    // 3. signer PAE, jamais le CBOR re-sérialisé nu
    const pt = "application/vnd.blunderdb.watermark+cbor;v=2"
    sig := ed25519.Sign(priv, dsse.PAE(pt, payload))
    return json.Marshal(dsse.Envelope{PayloadType: pt,
        Payload: b64(payload), Signatures: []dsse.Signature{{KeyID: kid, Sig: b64(sig)}}})
}

// --- VERIFY : rejet strict des versions/champs inconnus, comparaison de digest ---
func Verify(exportPath string, env []byte, pub ed25519.PublicKey) (Status, error) {
    var e dsse.Envelope
    if err := json.Unmarshal(env, &e); err != nil { return Invalid, err }
    if e.PayloadType != "application/vnd.blunderdb.watermark+cbor;v=2" {
        return Invalid, errUnknownVersion // fail closed
    }
    payload := unb64(e.Payload)
    if !ed25519.Verify(pub, dsse.PAE(e.PayloadType, payload), unb64(e.Signatures[0].Sig)) {
        return Invalid, errBadSig
    }
    var w Watermark
    if err := cborDetStrict(payload, &w); err != nil { return Invalid, err } // rejette champs inconnus
    got, _ := fileDigestBLAKE3(exportPath, sqliteVolatileRanges)
    if !bytes.Equal(got, w.Content.FileDigest.Value) {
        return DerivedFrom, nil // modèle C2PA : "dérivé de", pas "invalide"
    }
    return Authentic, nil
}

// --- SEAL : en-tête en AAD + chiffrement en flux ---
func Seal(dst io.Writer, plaintext io.Reader, pass []byte, wm []byte) error {
    salt := randBytes(16); nonce := randBytes(24) // XChaCha20 : nonce 192 bits
    p := Argon2Params{M: 65536, T: 3, P: 1}       // bornées à la lecture
    key := argon2.IDKey(pass, salt, p.T, p.M, p.P, 32)
    hdr := buildHeader(2, p, salt, nonce, wm)     // magic..watermark_dsse
    if _, err := dst.Write(hdr); err != nil { return err }
    streamKey := hkdf32(key, nonce, "payload")
    return streamSeal(dst, plaintext, streamKey, aad(hdr)) // STREAM, chunks 64 KiB, dernier bloc marqué
}

// --- OPEN : valider bornes AVANT allocation, vérifier l'AAD ---
func Open(src io.Reader, pass []byte) (io.Reader, Header, error) {
    hdr, err := readHeaderLimited(io.LimitReader(src, maxHeader)) // pas de make() non validé
    if err != nil { return nil, hdr, err }
    if hdr.Version != 2 { return nil, hdr, errUnknownVersion }
    if hdr.P.M < 19456 || hdr.P.T < 2 || hdr.P.M > 2<<20 { // bornes min ET max Argon2id
        return nil, hdr, errBadKDFParams
    }
    key := argon2.IDKey(pass, hdr.Salt, hdr.P.T, hdr.P.M, hdr.P.P, 32)
    streamKey := hkdf32(key, hdr.Nonce, "payload")
    return streamOpen(src, streamKey, aad(hdr.Raw)), hdr, nil // échoue si AAD (en-tête) altéré
}
```

**Bibliothèques Go concrètes.** `crypto/ed25519` (signature) ; `golang.org/x/crypto/argon2` (`IDKey`) ; `golang.org/x/crypto/chacha20poly1305` (XChaCha20) ou `crypto/cipher`+`crypto/aes` (GCM) ; `golang.org/x/crypto/hkdf` ; `github.com/google/tink/go/streamingaead` ou `filippo.io/age` (chunking éprouvé) ; `github.com/secure-systems-lab/go-securesystemslib/dsse` (PAE/enveloppe) ; `zeebo/blake3` (digest fichier rapide) ; `crypto/sha3` pour le digest logique ; `github.com/jedisct1/go-minisign` si compatibilité minisign souhaitée.

## Recommendations

**Étape 1 — Corriger le binding (priorité absolue).** Passer en v2 du format de filigrane : signer `PAE(domaine;v=2, CBOR{ métadonnées + file_digest })`. Traiter l'export `VACUUM INTO` comme immuable et hacher son flux d'octets (BLAKE3 ou SHA-256), avec exclusion documentée des octets d'en-tête SQLite volatils. Cela seul supprime l'attaque de transplantation. Bibliothèques : `crypto/ed25519` + `go-securesystemslib/dsse` + `zeebo/blake3`.
- *Seuil de bascule* : si vous devez tolérer des transformations légitimes (VACUUM ultérieur, changement de page_size), ajoutez le digest logique canonique (Étape 3) ; sinon le digest de fichier suffit.

**Étape 2 — Corriger l'AEAD (priorité haute).** Format conteneur v2 : mettre **tout l'en-tête en AAD**, stocker les paramètres Argon2id et les borner à la lecture (min `m=19 MiB, t=2, p=1` ; max `m≈1-2 GiB`), valider toutes les longueurs avant allocation (`io.LimitReader`, pas de `make` non validé). Migrer vers un chiffrement en flux (Tink `streamingaead` ou age) avec marquage du dernier chunk et numéro de chunk dans le nonce. Envisager XChaCha20-Poly1305 (nonce 192 bits, tirage aléatoire sûr).
- *Downgrade v1→v2* : lire v1 en avertissant l'utilisateur (« format hérité, en-tête non authentifié »), n'écrire que v2, et refuser à terme la lecture v1 (rejet strict des versions inconnues/obsolètes).

**Étape 3 — Robustesse et provenance (optionnel selon le modèle de menace).** Ajouter le digest logique canonique (`ORDER BY` explicite + encodage longueur-préfixé) pour distinguer « invalide » de « dérivé de » (modèle C2PA). Si traçabilité des fuites requise → fingerprinting par destinataire (Agrawal-Kiernan), mécanisme orthogonal à la signature. Si provenance vérifiable publiquement souhaitée → bundle Sigstore + Rekor + horodatage RFC 3161.

**Étape 4 — Gouvernance des clés.** Key id dans chaque signature, plan de rotation (proof-of-rotation façon APK v3), distribution de la clé publique via liste de confiance embarquée, séparation de domaine dans tous les messages signés.

**Décider d'abord le modèle de menace.** Question centrale : *que cherche-t-on à empêcher ?* Si c'est « prouver qu'une base vient bien de l'émetteur X et n'a pas été altérée » → signature + hard binding (Étapes 1-2) suffisent. Si c'est « empêcher la copie/redistribution » → aucune signature n'y parvient ; il faut du DRM côté service (modèle Chessable/AnkiHub) et/ou du fingerprinting robuste (traçabilité, pas prévention).

## Caveats

**Ce que la conception ne protège PAS :**
- **Un attaquant détenant la clé privée** peut re-signer n'importe quel contenu : la sécurité repose entièrement sur la confidentialité de la clé de l'émetteur.
- **La recopie du contenu sans le filigrane** : rien n'empêche d'exporter les données dans une base neuve non signée. Une signature prouve l'authenticité, pas l'unicité ni la non-copie.
- **Un filigrane signé n'empêche pas la copie** du fichier signé lui-même ; il permet seulement de détecter une altération ou une transplantation.
- **La fuite de métadonnées** : le filigrane en clair dans l'en-tête expose émetteur/note/date sans déchiffrement (choix assumé pour la vérifiabilité).
- **Le digest logique n'est canonique que si l'implémentation force un `ORDER BY` explicite** : `dbhash`/`.sha3sum` reposent sur un ordre « non garanti par la documentation » (source dbhash.c) — à ne pas reproduire tel quel pour un usage cryptographique.

**Points où les sources sont ambiguës ou se contredisent :**
- **Endianness dans `shathree.c`** : les commentaires de `sha3_query` disent « little-endian », mais le code, le commentaire de `sha3_agg` et les vecteurs de test sont **big-endian**. Le code fait foi.
- **Reproductibilité bit-à-bit de `VACUUM INTO`** : la documentation SQLite ne garantit pas explicitement une sortie bit-à-bit identique entre deux exécutions ; le change counter et les champs de version varient. Aucune source officielle ne promet la reproductibilité binaire — d'où la nécessité d'exclure/normaliser les octets d'en-tête volatils.
- **Paramètres Argon2id « recommandés »** : OWASP présente `m=19 MiB, t=2, p=1` et `m=46 MiB, t=1, p=1` comme deux profils d'égale robustesse (compromis CPU/RAM) ; d'autres guides recommandent 64-256 MiB. Il n'y a pas de valeur unique « correcte » — à calibrer selon le matériel cible et la latence acceptable, avec des bornes min/max imposées à la lecture.
- **age n'est pas une AE asymétrique authentifiée** (Filippo) : l'authentification d'en-tête par HMAC empêche l'altération par un tiers sans la file key, mais pas la réécriture par quelqu'un qui la connaît — d'où la nécessité de la couche de signature Ed25519 par-dessus.