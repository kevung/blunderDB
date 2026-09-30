# Transcription et direction de tournoi en mode headless — plan 2026-10

Plan de conception, aucun code. Objet : exposer la **Transcription** (ADR-0044, ADR-0045) et
la **Direction** / **Rencontre** (ADR-0047, ADR-0056) par le démon `serve` et par `call`, pour
qu'un frontend web puisse transcrire et diriger. Base : commit `2bcd82568`.

Conventions : un fait vérifié porte `fichier:ligne` ; **[H]** marque une hypothèse.

## 0. Ce plan contredit quatre ADR acceptées — à trancher d'abord

| ADR | Ce qu'elle dit | Ligne |
|---|---|---|
| 0045 règle 9 | « the CLI has `transcribe`; `serve` exposes nothing » | `docs/adr/0045-…md:48` |
| 0047 | « aucune route de consultation, rien dans le front web (ADR-0039 règle 1) » ; « Le démon ne reçoit rien » | `docs/adr/0047-…md:26-28`, `:36` |
| 0056 | CLI en lecture seule ; « Le démon ne reçoit rien (ADR-0047) » | `docs/adr/0056-…md:63-64` |
| 0039 règle 1 | le front web ne fait que consulter, chercher, réviser ; « ni édition, ni import, …, matchs, tournois » ; l'ouvrir, « c'est remplacer cette ADR et reposer “faut-il une seconde application ?” » | `docs/adr/0039-…md:15-19`, `:31-32` |

Rien de ce qui suit ne se code avant une ADR qui remplace 0039 et amende 0045 §9, 0047 et 0056.
Le plan distingue donc **l'API** (routes `/v1/`, utiles aussi à `call`, à gammonGo et aux
scripts, sans rouvrir 0039) et **le front web** (qui la rouvre).

## 1. L'existant

### 1.1 Où sont serve et call

- Dispatch : `main.go:36-43` ; `serve` → `server.RunServe` (`main.go:87-88`), `call` →
  `server.RunCall` (`main.go:96`) ; CLI via `cli.IsCommand` sur `handlers()`
  (`internal/cli/cli.go:101-130`, `:136`) ; GUI sinon (`main.go:134`).
- Le serveur est `internal/server/` ; `pkg/blunderdb/server/embed.go` n'est que l'enveloppe
  pour l'embarquer (gammonGo).
- Routes : `internal/server/routes.go`, `domainRoutes()` (`:138-157`), une famille par
  magasin du contrat. Toutes en `POST /v1/<famille>.<méthode>`, enveloppées par
  `rpc` / `rpcVoid` / `rpcStream` (`internal/server/handlers_rpc.go:135`, `:156`, `:167`) sur
  `s.opts.Storage.<Store>()`. Exemple : `handlers_tournaments.go:44-90`.
- Tenant : `X-Tenant-ID` (`internal/server/middleware/tenant.go:17`, lu `:50`), `scopeOf(r)`
  (`handlers_rpc.go:52`). Chaîne : RequestID, Recover, Metrics, Logging, CORS, Compress,
  Tenant, RateLimit, limitBody (`internal/server/server.go:157-183`) ; `Idempotency-Key`
  (`internal/server/idempotency.go:18`).
- Flux : NDJSON seulement (`internal/server/ndjson.go`, `streamingPaths`,
  `routes.go:159-180`). **Aucun SSE, websocket, ETag ni If-Match** dans le code non-test.
- `call` (`internal/server/call.go:37-140`) fabrique un `POST /v1/<méthode>` et le passe au
  même `Handler()` : **un processus par appel**. Tout état en mémoire du serveur meurt avec lui.
- Documentation : `ARCHITECTURE.md:12-67` (cinq modes, deux backends un contrat, parité) ;
  `doc/source/mode_headless.rst` ne nomme ni transcription ni direction.

### 1.2 Ce qui est exposé

| Domaine | GUI (Wails, `*database.Database`) | CLI | serve / call |
|---|---|---|---|
| Transcription | `ListTranscriptions`…`ApplyTranscriptionGesture` (`database/db_transcription.go:60-191`), `SaveTranscriptionAsMatch` (`db_transcription_save.go:43`), export MAT | `transcribe` (lecture seule, `internal/cli/cli_transcribe.go:20`) | **rien** |
| Direction | ~60 méthodes (`database/db_direction_*.go`) : config, participants, propositions, résultats, tables, classement, historique, emplacements, annuaire, pages | `tournament list/verify/standings/page/export` (lecture, `cli_tournament.go:47-51`) | **rien** |
| Rencontre | `database/db_rencontre.go:41-243`, `db_rencontre_page.go:130-197` | `tournament page --rencontre` | **rien** |
| Tournoi importé | — | — | `tournaments.*`, 12 routes (`handlers_tournaments.go:44-90`) |

### 1.3 Où vit la logique — le vrai obstacle

- **Transcription.** Le moteur est pur (`pkg/blunderdb/transcript/`, ADR-0045 §9). La session
  d'édition est un `transcript.Editor` (`transcript/apply.go:973` ; pile `past, future`,
  `Undo`/`Redo` `:1027-1057`) tenu dans `Database.transcriptSessions map[int64]*Editor`
  (`database/db.go:56-57`), jamais évincé sauf `CloseTranscription`. Seul le document durable
  est réécrit après un geste (`db_transcription.go:191-232`) ; curseur et pile d'annulation
  ne sont pas persistés. Le contrat a `Transcriptions()` (`storage/storage.go:67`, CRUD
  opaque), implémenté SQLite et PostgreSQL.
- **Direction.** Le paquet `pkg/blunderdb/direction` définit `direction.Store`
  (`direction/direction.go:60-68`) dont le commentaire affirme « the desktop wrapper and both
  storage backends implement it » (`:54-55`) : **inexact**, seul `database.directionStore`
  l'implémente (`database/db_direction.go:20`), sur `d.db` et `d.mu`. Aucun `DirectionStore`
  dans `storage.Storage`. Toute la logique (`EnterResult`, `db_direction_result.go:149-163`,
  etc.) est une méthode de `*database.Database`, que le serveur n'utilise pas (ARCHITECTURE.md
  §2 : serveur → `storage.Storage`).
- **Rencontre.** `Rencontres()` est au contrat (`storage/storage.go:55`) ; la logique de salle
  (gestes écrits dans chaque Direction membre) est sur `Database`.
- **PostgreSQL.** Le schéma existe : `021_transcription.sql`, `025_direction.sql`,
  `027_rencontre.sql` sous `storage/postgres/migrations/`, avec `tenant_id` et RLS. Il manque
  le code d'accès PostgreSQL de `direction.Store` et les tests du contrat.

### 1.4 Concurrence et événements aujourd'hui

- `Database.mu` RWMutex, SQLite à une connexion (`database/db.go:26`, `:351`).
- `Direction.append` pose `ev.Seq = len(d.journal)` (`direction/direction.go:289`) et
  `AppendEvent` refuse un `seq` existant (PK `(tournament_id, seq)`, contrat
  `direction.go:57-59`) : **un compare-and-swap existe déjà**, sur le numéro d'événement.
- Transcription : `Save` réécrit la ligne, sans contrôle de version.
- Aucun `EventsEmit` pour la direction ; la page murale est un fichier réécrit après chaque
  geste (`db_rencontre_page.go:100-107`, `:184-197`), best-effort.

## 2. Conception

### 2.1 Étape préalable : la logique descend sous le contrat

Parité CLI/GUI/serveur (CLAUDE.md) : pas de seconde implémentation pour le serveur.

1. **`storage.DirectionStore`** au contrat : `Get/List/Create/Update/Delete`,
   `AppendEvent`, `LoadEvents`, par tenant, implémenté `storage/sqlite` et `storage/postgres`,
   avec une suite `storagetest/contract_direction.go` (refus d'un `seq` existant compris).
2. **Un service** `pkg/blunderdb/direction/service` [H : nom] qui prend un `storage.Stores`
   (ou un `storage.Tx`) et porte ce que `db_direction_*.go` fait aujourd'hui : il reçoit la
   liste des méthodes de §2.3. `*database.Database` devient une façade mince (verrou `d.mu`
   + appel du service) ; le GUI ne change pas.
3. Même chose pour la transcription : un `transcription.Service` sur
   `Transcriptions()` + le chemin d'enregistrement en Match (`db_transcription_save.go:43`,
   qui passe aujourd'hui par l'import du wrapper) [H : le chemin d'ingest du serveur,
   `ingestRoutes`, sait écrire un Match complet — à vérifier].
4. Un geste de salle écrit dans N Directions **dans une transaction** (ADR-0056 §2) :
   `storage.BeginTx` (`storage/storage.go:77`) le permet déjà.

### 2.2 Routes transcription

Famille `transcriptions.*`, `POST /v1/…`, corps JSON :

| Route | Rôle |
|---|---|
| `list`, `get`, `create`, `delete` | CRUD du brouillon (contrat existant) |
| `open` | rend `TranscriptionState` + `version` + `sessionId` |
| `apply` | `{id, version, gesture}` → état annoté ; 409 si `version` périmée |
| `undo`, `redo` | sur la session |
| `close` | libère la session |
| `saveAsMatch`, `exportMat` | enregistrement (analyse 2-ply comprise, ADR-0045 §8) ; long → `streamingPaths` |

`call transcriptions.apply --json '{…}'` marche mais chaque appel est un processus neuf : la
pile d'annulation doit donc être persistée (§2.4) ou `undo` indisponible par `call` [décision].

### 2.3 Routes direction et rencontre

Familles `directions.*` et `rencontres.*`, calquées sur les méthodes de `Database` (les noms
Go fixent les noms de route) :

- **lecture** (sans verrou d'écriture) : `get`, `list`, `participants`, `freeParticipants`,
  `tableGrid`, `brackets`, `standings`, `standingsCsv`, `history`, `clock`, `slots`,
  `lastDecision`, `directory`, `pageHtml`, `pairingSheetHtml`, `rencontres.pageHtml` ;
- **gestes** (un événement ou une transaction) : `create`, `setConfig`, `previewConfig`,
  `enterParticipants`, `addParticipant`, `updateParticipant`, `withdraw`, `reinstate`,
  `makeAbsent`, `makeAvailable`, `addPair`, `updatePair`, `confirmProposal`,
  `confirmAllProposals`, `startMatch`, `enterResult`, `enterForfeit`, `moveMatchToTable`,
  `cancelMatch`, `correctResult`, `close`, `reopen`, `addNote`, `attachMatch`,
  `detachMatch` ; `rencontres.create/update/attach/detach/trash/setTableOutOfService/setBreaks`.
- **exclus** : `OpenDirectionOutputDialog`, `SetDirectionOutputDir`, `WriteDirectionPage`,
  `WriteRencontrePage` — un chemin de fichier du serveur n'a pas de sens pour un client
  distant ; le client lit `pageHtml`. `SetDirectionStrings` (catalogue i18n poussé par le
  front) devient un paramètre `lang` des routes qui rendent du texte [H].

Chaque geste rend la `DirectionView` complète (comme le GUI) et sa `version`.

### 2.4 État de session côté serveur

Le seul état non durable est la **session de transcription** (pile `past/future`, curseur,
`Entry`). La Direction n'en a aucun : tout est rejoué (ADR-0047 §3).

Options :

| | A. Mémoire, par (tenant, brouillon) | B. Persistée en base | C. Sans état côté serveur |
|---|---|---|---|
| Contenu | `map[tenant]map[id]*Editor`, TTL d'inactivité (30 min [H]), plafond par tenant | table `transcription_session` (pile sérialisée, curseur) | le client garde la pile, le serveur applique `gesture` sur le document durable |
| `call` | undo perdu entre deux appels | marche | marche |
| Plusieurs instances derrière un proxy | faux sans affinité de session | marche | marche |
| Coût | nul en schéma | bump de `DatabaseVersion`, trois côtés | le `Editor` doit accepter un document et une pile fournis |
| Cohérence avec ADR-0045 §1 | oui (« pile en mémoire ») | amende | oui |

**Recommandation : A pour le démon + C comme repli** [H] : la session est un cache ; sa perte
(TTL, redémarrage, autre instance) ne perd que l'annulation, jamais un geste, puisque le
document durable est écrit après chaque geste (`db_transcription.go:191-232`). Le client
reçoit `sessionId` ; un `sessionId` inconnu → 410, le client rouvre (`open`), curseur en fin
de document — exactement le comportement du GUI à la réouverture. Le démon doit rester
utilisable sur plusieurs instances : l'annulation y est « au mieux », dit et documenté.

### 2.5 Concurrence multi-onglets, multi-TD

- **Direction : versionnage optimiste sur le `seq`.** `version` = nombre d'événements du
  journal. Tout geste porte `ifVersion` (corps) ou `If-Match: "<seq>"` (en-tête) ; si le
  journal a avancé, **409** avec la vue fraîche ; le client montre « Sophie vient de saisir
  table 4 » et rejoue le geste s'il reste valide. Le CAS existe déjà (`AppendEvent`) : il
  suffit de comparer avant d'appliquer, dans la transaction. Les lectures rendent
  `ETag: "<tenant>:<tournoi>:<seq>"` ; `If-None-Match` → 304 (la page murale interroge peu
  cher).
- **Geste de salle** : version de la Rencontre = somme ou vecteur des `seq` des membres [H] ;
  un seul 409 si l'un a bougé.
- **Transcription** : `version` = compteur de gestes du document ; colonne ou champ JSON
  `revision` (bump) [H : un champ dans `Document` évite le bump]. Deux onglets sur un même
  brouillon : le second reçoit 409 et relit. Le GUI de bureau et un client web sur la même
  base SQLite passent par le même contrôle.
- **Idempotence** : les gestes acceptent `Idempotency-Key` (`idempotency.go:18`) : un double
  clic ou une reprise réseau ne saisit pas deux résultats.

### 2.6 Notifications temps réel

- **SSE** (`GET /v1/events?rencontre=N` ou `?tournament=N`, `text/event-stream`) : un
  message `{kind, tournamentId, seq}` par geste ; le client relit ce qui l'intéresse (pas de
  vue poussée : un seul format de lecture). Pas de websocket : le sens serveur → client suffit.
- **Bus** : en mémoire par processus, par tenant, publié après `commit`. Plusieurs instances :
  `LISTEN/NOTIFY` PostgreSQL (canal par tenant) [H] ; SQLite = une instance par construction
  (`SingleTenant`, verrou de fichier, `database/db.go:49`).
- **Infrastructure à adapter** : la route SSE entre dans `streamingPaths` (pas d'échéance),
  `Compress` doit la laisser passer ou la vider à chaque message, `RateLimit` compte une
  connexion et non ses messages, en-tête de battement toutes les 25 s pour les proxys.
- **Page murale** : une route `rencontres.pageHtml` + SSE suffit à un affichage web qui se
  recharge ; le fichier `index.html` reste la voie du bureau. [Décision : la page murale est-
  elle servie sans tenant, comme les statiques de `--web` (ADR-0039 règle 2) ? Elle contient
  des données (noms de joueurs) : non, elle exige un tenant.]
- **GUI de bureau** : le même bus émet un `EventsEmit` Wails ; deux fenêtres ou un GUI + un
  client web sur une même base se voient. [H : utile seulement si SQLite partagée entre
  bureau et démon, ce que `fileLock` interdit aujourd'hui.]

### 2.7 Authentification (ADR-0005)

Aucune dans le moteur. Le démon fait confiance à `X-Tenant-ID` derrière un proxy
authentifiant. Diriger un tournoi depuis un téléphone sur le Wi-Fi d'un club rend la tentation
forte : la documentation (`mode_headless.rst`, « proxy authentifiant ») reçoit un exemple
dédié et un avertissement, et les routes d'écriture de direction restent **désactivées par
défaut** (`serve --direction`), comme `--web` [H]. Aucun rôle (TD, arbitre, lecteur) dans le
moteur : un rôle est une règle du proxy sur le préfixe `/v1/directions.` [H].

### 2.8 Parité (invariant CLAUDE.md)

| Méthode | GUI | CLI | serve/call |
|---|---|---|---|
| Lectures direction | façade `Database` → service | `tournament …` → service | route → service |
| Gestes direction | façade | **à décider** : sous-commandes `direction result …` (ux.md §3 les décrit) ou `call` suffit | route |
| Transcription | façade | `transcribe` (lecture) | route |

`call` rend déjà le CLI d'écriture inutile pour les scripts ; ajouter des sous-commandes
d'écriture au CLI relève de ADR-0056 (« CLI en lecture seule ») et n'est pas nécessaire.

## 3. Lots ordonnés

| Lot | Contenu | Garde |
|---|---|---|
| **H0** ADR | remplace 0039, amende 0045 §9, 0047, 0056 ; tranche §5 | — |
| **H1** `DirectionStore` au contrat | interface, SQLite, PostgreSQL (code d'accès sur 025/027), `storagetest/contract_direction.go` ; corriger le commentaire `direction.go:54-55` | contrat vert sur les deux backends |
| **H2** Service de direction | logique de `db_direction_*.go` et `db_rencontre*.go` déplacée ; `Database` façade ; aucun changement GUI | tests existants de `database` inchangés, verts |
| **H3** Routes de lecture | `directions.*`, `rencontres.*` en lecture, `ETag`/`If-None-Match`, `openapi.yaml` régénéré, `mode_headless.rst` + 8 `.po` | tests `internal/server`, suite `call` |
| **H4** Routes de gestes | `ifVersion`/`If-Match` → 409, `Idempotency-Key`, `serve --direction` | test de course : deux gestes concurrents, un 409 |
| **H5** SSE | `/v1/events`, bus mémoire, `streamingPaths`, Compress/RateLimit | test : un geste → un message |
| **H6** Service de transcription | `open/apply/undo/redo/close`, sessions (A) avec TTL, 410 ; enregistrement en Match sur le contrat | test : TTL expiré → 410 → réouverture, aucun geste perdu |
| **H7** PostgreSQL multi-instance | `LISTEN/NOTIFY` pour le bus | test d'intégration PG (nightly) |
| **H8** Front web | selon H0 : client de H3–H6 ; hors de ce plan | — |

H1 → H2 est le gros du travail et sert le bureau aussi (une seule implémentation, testée sur
PostgreSQL). H3 livre la page murale web sans ouvrir l'écriture. Aucun lot ne change le hash
Zobrist ; seul le choix B de §2.4 ou une colonne `revision` imposent un bump de schéma.

## 4. Risques

- `saveAsMatch` sur PostgreSQL déclenche l'analyse 2-ply (ADR-0045 §8) : ADR-0015 dit que le
  démon « n'expose pas d'évaluateur » ; analyser ce qu'on enregistre n'est pas en exposer un,
  mais la charge CPU d'un démon multi-tenant est à borner (file, `gammonnet` existant) [H].
- Rejouer une Direction à chaque lecture (ADR-0047 §3) : budget < 50 ms à 64 joueurs × 300
  événements (ux.md §1, non mesuré) ; un client web qui relit à chaque SSE multiplie les
  rejeux → cache de `State` par (tenant, tournoi, seq) dans le service [H].
- Le catalogue i18n de direction est poussé par le front (`SetDirectionStrings`) : état
  global du processus, faux en multi-tenant et multi-langue.

## 5. Décisions à trancher (ADR à écrire)

1. **Rouvrir le périmètre web** : remplacer ADR-0039 (front web éditeur) ou n'exposer que
   l'API (`call`, gammonGo, scripts) et garder le front web en consultation ?
2. **Session de transcription** : mémoire + TTL (A), persistée (B) ou sans état (C) ; que
   promet `undo` par `call` ?
3. **Versionnage** : `seq` du journal comme version et `If-Match` obligatoire sur les gestes,
   ou facultatif (dernier qui écrit gagne, signalé) ; `revision` de transcription en colonne
   (bump) ou dans le document ?
4. **Temps réel** : SSE seul, et `LISTEN/NOTIFY` pour plusieurs instances, ou une instance par
   tenant imposée pour la direction ?
5. **Garde-fou d'exposition** : `serve --direction` éteint par défaut, et la page murale web
   exige-t-elle un tenant ?
6. **CLI d'écriture** : `call` suffit-il, ou la CLI reçoit-elle les gestes décrits dans
   ux.md §3 (amende ADR-0056) ?
