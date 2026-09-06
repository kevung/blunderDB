.. _api_reference:

Contrat d'API
=============

Cette annexe est générée depuis le code source (``go run ./cmd/openapi-gen``,
voir ``openapi.yaml`` à la racine du dépôt pour le contrat complet, schémas
compris) : ne pas éditer directement, la prochaine génération écraserait toute
modification. Chaque famille regroupe les méthodes ``POST /v1/<famille>.<méthode>``
exposées par le démon (voir :ref:`headless`), avec leur forme de réponse
(JSON, ou NDJSON pour une liste en flux) et, quand elle existe, la mention de
l'en-tête ``Idempotency-Key`` optionnel.

.. code-block:: text

   analyses
     POST /v1/analyses.delete                      JSON
     POST /v1/analyses.load                        JSON
     POST /v1/analyses.repair                      JSON
     POST /v1/analyses.save                        JSON
   anki
     POST /v1/anki.buryCard                        JSON
     POST /v1/anki.createDeck                      JSON
     POST /v1/anki.deckPositions                   NDJSON
     POST /v1/anki.deckStats                       JSON
     POST /v1/anki.deleteDeck                      JSON
     POST /v1/anki.forecast                        JSON
     POST /v1/anki.listDecks                       NDJSON
     POST /v1/anki.nextCard                        JSON
     POST /v1/anki.removeCard                      JSON
     POST /v1/anki.resetDeck                       JSON
     POST /v1/anki.retention                       JSON
     POST /v1/anki.reviewCard                      JSON  (Idempotency-Key)
     POST /v1/anki.reviewLog                       NDJSON
     POST /v1/anki.suspendCard                     JSON
     POST /v1/anki.sync                            JSON
     POST /v1/anki.syncWithPositions               JSON
     POST /v1/anki.updateDeck                      JSON
     POST /v1/anki.updateDeckParams                JSON
   collections
     POST /v1/collections.addPosition              JSON
     POST /v1/collections.addPositions             JSON
     POST /v1/collections.collectionsOf            NDJSON
     POST /v1/collections.copyPosition             JSON
     POST /v1/collections.create                   JSON  (Idempotency-Key)
     POST /v1/collections.delete                   JSON
     POST /v1/collections.get                      JSON
     POST /v1/collections.list                     NDJSON
     POST /v1/collections.movePosition             JSON
     POST /v1/collections.positionIndexMap         JSON
     POST /v1/collections.positions                NDJSON
     POST /v1/collections.removePosition           JSON
     POST /v1/collections.removePositions          JSON
     POST /v1/collections.reorder                  JSON
     POST /v1/collections.reorderPositions         JSON
     POST /v1/collections.update                   JSON
   comments
     POST /v1/comments.add                         JSON
     POST /v1/comments.byPosition                  NDJSON
     POST /v1/comments.delete                      JSON
     POST /v1/comments.deleteForPosition           JSON
     POST /v1/comments.listAll                     NDJSON
     POST /v1/comments.search                      NDJSON
     POST /v1/comments.text                        JSON
     POST /v1/comments.update                      JSON
   exports
     POST /v1/exports.json                         custom
     POST /v1/exports.sqlite                       custom
   filters
     POST /v1/filters.delete                       JSON
     POST /v1/filters.list                         NDJSON
     POST /v1/filters.loadEditPosition             JSON
     POST /v1/filters.loadExcludePosition          JSON
     POST /v1/filters.save                         JSON
     POST /v1/filters.saveEditPosition             JSON
     POST /v1/filters.saveExcludePosition          JSON
     POST /v1/filters.update                       JSON
   gammonnet
     POST /v1/gammonnet.analyzeMissing             custom
     POST /v1/gammonnet.analyzeMissing.cancel      custom
     POST /v1/gammonnet.compare                    custom
     POST /v1/gammonnet.sweepStale                 custom
   history
     POST /v1/history.clear                        JSON
     POST /v1/history.load                         JSON
     POST /v1/history.save                         JSON
   imports
     POST /v1/imports.bgf                          custom
     POST /v1/imports.cancel                       custom
     POST /v1/imports.db                           custom
     POST /v1/imports.gnubg                        custom
     POST /v1/imports.json                         custom
     POST /v1/imports.list                         JSON
     POST /v1/imports.position                     custom
     POST /v1/imports.report                       JSON
     POST /v1/imports.xg                           custom
   maintenance
     POST /ops/maintenance.vacuum                  custom
   matches
     POST /v1/matches.createGame                   JSON
     POST /v1/matches.createMove                   JSON
     POST /v1/matches.delete                       JSON
     POST /v1/matches.exportMat                    custom
     POST /v1/matches.findByHash                   JSON
     POST /v1/matches.games                        NDJSON
     POST /v1/matches.get                          JSON
     POST /v1/matches.lastVisited                  JSON
     POST /v1/matches.list                         NDJSON
     POST /v1/matches.mergePlayers                 JSON
     POST /v1/matches.movePositions                NDJSON
     POST /v1/matches.moves                        NDJSON
     POST /v1/matches.movesByMatch                 NDJSON
     POST /v1/matches.save                         JSON
     POST /v1/matches.setLastVisitedPosition       JSON
     POST /v1/matches.swapPlayers                  JSON
     POST /v1/matches.update                       JSON
     POST /v1/matches.updateComment                JSON
   metadata
     POST /v1/metadata.counts                      JSON
     POST /v1/metadata.version                     JSON
   positions
     POST /v1/positions.delete                     JSON
     POST /v1/positions.epc                        JSON
     POST /v1/positions.exists                     JSON
     POST /v1/positions.fromXGID                   JSON
     POST /v1/positions.fromXGP                    JSON
     POST /v1/positions.legalMoves                 JSON
     POST /v1/positions.list                       NDJSON
     POST /v1/positions.listIds                    JSON
     POST /v1/positions.load                       JSON
     POST /v1/positions.loadByIds                  JSON
     POST /v1/positions.parseText                  JSON
     POST /v1/positions.reclassifyPhases           JSON
     POST /v1/positions.save                       JSON
     POST /v1/positions.update                     JSON
   search
     POST /v1/search.find                          NDJSON
     POST /v1/search.parse                         JSON
     POST /v1/search.query                         custom
   searchHistory
     POST /v1/searchHistory.deleteEntry            JSON
     POST /v1/searchHistory.list                   NDJSON
     POST /v1/searchHistory.save                   JSON
   session
     POST /v1/session.clear                        JSON
     POST /v1/session.load                         JSON
     POST /v1/session.save                         JSON
   stats
     POST /v1/stats.compute                        JSON
     POST /v1/stats.dateRange                      JSON
     POST /v1/stats.matchBadges                    JSON
     POST /v1/stats.matchDetail                    JSON
     POST /v1/stats.playerNames                    JSON
     POST /v1/stats.playerTable                    JSON
     POST /v1/stats.positionIdsByMatch             JSON
     POST /v1/stats.positionIdsBySelection         JSON
     POST /v1/stats.positionIdsByTournament        JSON
     POST /v1/stats.tournamentBadges               JSON
   tenant
     POST /ops/tenant.purge                        custom
   tournaments
     POST /v1/tournaments.addMatch                 JSON
     POST /v1/tournaments.create                   JSON  (Idempotency-Key)
     POST /v1/tournaments.delete                   JSON
     POST /v1/tournaments.get                      JSON
     POST /v1/tournaments.list                     NDJSON
     POST /v1/tournaments.matches                  NDJSON
     POST /v1/tournaments.removeMatch              JSON
     POST /v1/tournaments.reorderMatches           JSON
     POST /v1/tournaments.setMatchByName           JSON
     POST /v1/tournaments.tournamentOf             JSON
     POST /v1/tournaments.update                   JSON
     POST /v1/tournaments.updateComment            JSON
   trash
     POST /v1/trash.count                          JSON
     POST /v1/trash.deleteCollection               JSON
     POST /v1/trash.deleteComment                  JSON
     POST /v1/trash.deletePosition                 JSON
     POST /v1/trash.discard                        JSON
     POST /v1/trash.empty                          JSON
     POST /v1/trash.list                           JSON
     POST /v1/trash.restore                        JSON


Idempotence
-----------

La plupart des méthodes n'ont besoin d'aucun mécanisme particulier : les
lectures sont sans effet de bord, et ``positions.save`` (comme le reste de
``positions.*``) est naturellement idempotente grâce au hachage Zobrist du
contenu — enregistrer deux fois la même position renvoie la même ligne, jamais
un doublon. 3 méthodes n'ont pas cette propriété (deux appels sont deux effets
distincts) et acceptent un en-tête ``Idempotency-Key`` optionnel : un appel
rejoué avec la même clé renvoie le résultat de la première tentative au lieu de
répéter son effet — voir la marque « (Idempotency-Key) » dans le tableau
ci-dessus. Aucune autre méthode n'a besoin ou n'accepte cet en-tête.
