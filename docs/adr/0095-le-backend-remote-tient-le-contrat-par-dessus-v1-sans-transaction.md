# Le backend remote tient le contrat par-dessus /v1, sans transaction et sans authentification

Status: accepted.
See also: ADR-0005 (le démon ne s'authentifie pas), ADR-0055 (budget de contexte), `pkg/blunderdb/storage/remote`

## Context

Un client qui veut travailler sur un tenant distant n'a aujourd'hui que deux voies : ouvrir la
base lui-même (SQLite local, ou la PostgreSQL du démon, ce qu'un client ne doit pas voir), ou
parler `/v1` à la main. Le contrat `storage.Storage` est déjà ce que le démon, la CLI `call`
et — demain — l'application de bureau consomment ; un troisième backend qui l'implémente
par-dessus `/v1` donne le tenant distant à tout code écrit contre le contrat, sans ajouter une
route au démon ni une ligne d'authentification au moteur.

`/v1` n'est pas le contrat : il expose plus de 300 routes, dont une partie appelle une méthode
de stockage telle quelle, une autre un service (Direction, Transcription, Duel, imports par
lots) ou un assistant qui fait plus que la méthode (`met.list` lit `mets.Overview`, `anki.sync`
lit `storage.SyncDeck`). Le contrat compte environ 300 méthodes.

## Decision

1. **Une méthode = un appel `POST /v1/<famille>.<méthode>`**, JSON à l'aller, JSON ou NDJSON
   au retour. Le backend ne tient ni cache ni connexion propre ; sa cohérence est celle de la
   route : un appel est atomique autant que son gestionnaire l'est, rien de plus.
2. **Une méthode n'est servie que par une route qui la sert telle quelle.** Une route qui
   passe par un assistant au comportement plus large n'est pas empruntée. Trois méthodes sans
   route de lot (`PositionStore.ExistsMany`, `CommentStore.ByPositions`,
   `MatchStore.CreateMoves`) bouclent sur la route unitaire : même résultat, sans
   l'atomicité — `CreateMoves` interrompu laisse écrits les coups d'avant.
3. **Toute autre méthode répond `*remote.UnsupportedError`**, qui correspond à
   `remote.ErrUnsupported` et à `storage.ErrInternal` : un échec, jamais un faux succès ni un
   « introuvable » qui se lirait comme une ligne absente. `remote.Unsupported` les nomme toutes
   (117 à ce jour) ; `unsupported_gen.go` est produit par `go generate` (`gen_stubs.go`) à
   partir des interfaces et des méthodes écrites à la main, et la compilation échoue tant qu'il
   n'est pas régénéré après un changement de l'un ou de l'autre. Une option que la route ne
   porte pas (`MatchListOpts.PlayerSpellings`) est refusée de la même façon plutôt
   qu'ignorée.
4. **Pas de transaction entre appels.** `BeginTx` est refusé : un `Tx` qui appliquerait chaque
   appel aussitôt ferait mentir `Rollback`. Le code qui a besoin d'une transaction (services de
   Direction, de Transcription, de Duel, imports) ne tourne pas sur ce backend ; il tourne dans
   le démon, derrière ses propres routes.
5. **Les erreurs reviennent en sentinelles.** L'enveloppe `{"error":{code,message,details}}`
   donne `not_found` → `storage.ErrNotFound`, `conflict` → `storage.ErrConflict` (et
   `*storage.DuplicatePositionError` quand `details.existingId` nomme la ligne, détail que
   l'erreur fournit désormais au démon), `invalid` → `storage.ErrInvalid`, `internal` →
   `storage.ErrInternal`, `unknown_route` (démon plus ancien) → `ErrUnsupported`. Les refus
   nommés du contrat (`ErrNoReviewToUndo`, `ErrSwapDraftOpen`, `ErrSwapPlayedHere`) sont
   reconnus à leur message. Une panne réseau, ou une réponse qui n'est pas l'enveloppe (un
   proxy qui répond en HTML), donne `remote.ErrUnavailable` : l'écriture a pu avoir lieu, on
   relit avant de réessayer.
6. **Aucune authentification ici** (ADR-0005). `Options.Header` est ajouté tel quel à chaque
   requête — un jeton du proxy, par exemple — et le backend n'en lit rien. Le `scope` d'une
   méthode part en `X-Tenant-ID` ; le `scope` vide de la CLI et du bureau part en
   `Options.Tenant`. Le proxy authentifiant devant le démon décide ; il réécrit normalement
   ces en-têtes. L'auteur d'un commentaire porté par le contexte part en `X-User-Name`.
7. **`Migrate` ne migre pas** : le démon migre sa base au démarrage. Il vérifie que la version
   du schéma distant est celle que parle ce binaire et refuse (`ErrConflict`) sinon.
8. **Preuve** : `storagetest.RunContractTestsOnly` exécute la suite de contrat contre un
   démon en processus (`httptest` sur `internal/server`, SQLite en mémoire) en ne gardant que
   les cas dont toutes les méthodes ont une route ; la liste `covered` de `remote_test.go` les
   nomme, et un cas n'en sort qu'avec la route qu'il demande.

## Consequences

- Rien ne change pour le démon, sinon que `DuplicatePositionError` remplit `details`.
- `blunderdb call` pourrait viser un démon distant sans fork : un cas `remote` de
  `OpenStorage` ferait passer ses gestionnaires en processus sur ce backend, au prix des
  routes de service qui demandent une transaction. Ce n'est pas fait ici. La CLI, elle, tourne
  sur `database.Database`, encore propre à SQLite : elle ne visera un démon distant qu'après
  sa migration vers le contrat, comme l'application de bureau.
- Étendre la couverture, c'est écrire la méthode à la main contre une route qui la sert telle
  quelle, relancer `go generate`, et ajouter à `covered` les cas qui passent.
