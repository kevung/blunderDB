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

   across
     POST /v1/across.analysesLoadByIds             JSON
     POST /v1/across.clubRanking                   JSON
     POST /v1/across.collectionPositions           NDJSON
     POST /v1/across.collectionsList               NDJSON
     POST /v1/across.commentsByZobrist             NDJSON
     POST /v1/across.matchMovePositions            NDJSON
     POST /v1/across.matchesGet                    JSON
     POST /v1/across.matchesList                   NDJSON
     POST /v1/across.playerTable                   JSON
     POST /v1/across.searchFind                    NDJSON
     POST /v1/across.statsCompute                  JSON
   analyses
     POST /v1/analyses.delete                      JSON
     POST /v1/analyses.load                        JSON
     POST /v1/analyses.loadByIds                   JSON
     POST /v1/analyses.repair                      JSON
     POST /v1/analyses.save                        JSON
   anki
     POST /v1/anki.buryCard                        JSON
     POST /v1/anki.createDeck                      JSON
     POST /v1/anki.createStudyDeck                 JSON
     POST /v1/anki.deckPositionCount               JSON
     POST /v1/anki.deckPositionIds                 JSON
     POST /v1/anki.deckPositions                   NDJSON
     POST /v1/anki.deckStats                       JSON
     POST /v1/anki.deleteDeck                      JSON
     POST /v1/anki.forecast                        JSON
     POST /v1/anki.indexOfDeckPosition             JSON
     POST /v1/anki.linkedCard                      JSON
     POST /v1/anki.listDecks                       NDJSON
     POST /v1/anki.nextCard                        JSON
     POST /v1/anki.removeCard                      JSON
     POST /v1/anki.resetDeck                       JSON
     POST /v1/anki.retention                       JSON
     POST /v1/anki.reviewCard                      JSON  (Idempotency-Key)
     POST /v1/anki.reviewLog                       NDJSON
     POST /v1/anki.reviewsByGameType               JSON
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
     POST /v1/collections.countPositions           JSON
     POST /v1/collections.create                   JSON  (Idempotency-Key)
     POST /v1/collections.delete                   JSON
     POST /v1/collections.freeze                   JSON
     POST /v1/collections.get                      JSON
     POST /v1/collections.indexOfPosition          JSON
     POST /v1/collections.list                     NDJSON
     POST /v1/collections.movePosition             JSON
     POST /v1/collections.positionIds              JSON
     POST /v1/collections.positionIndexMap         JSON
     POST /v1/collections.positions                NDJSON
     POST /v1/collections.removePosition           JSON
     POST /v1/collections.removePositions          JSON
     POST /v1/collections.reorder                  JSON
     POST /v1/collections.reorderPositions         JSON
     POST /v1/collections.setFilter                JSON
     POST /v1/collections.update                   JSON
   comments
     POST /v1/comments.add                         JSON
     POST /v1/comments.byPosition                  NDJSON
     POST /v1/comments.delete                      JSON
     POST /v1/comments.deleteForPosition           JSON
     POST /v1/comments.listAll                     NDJSON
     POST /v1/comments.search                      NDJSON
     POST /v1/comments.tags                        JSON
     POST /v1/comments.text                        JSON
     POST /v1/comments.update                      JSON
   directions
     POST /v1/directions.addNote                   JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.addPair                   JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.addParticipant            JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.attachMatch               JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.brackets                  JSON  (ETag)
     POST /v1/directions.cancelMatch               JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.clock                     JSON  (ETag)
     POST /v1/directions.close                     JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.confirmAllProposals       JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.confirmProposal           JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.correctResult             JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.create                    JSON  (Idempotency-Key)
     POST /v1/directions.detachMatch               JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.directory                 JSON  (ETag)
     POST /v1/directions.enterForfeit              JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.enterParticipants         JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.enterResult               JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.freeParticipants          JSON  (ETag)
     POST /v1/directions.get                       JSON  (ETag)
     POST /v1/directions.history                   JSON  (ETag)
     POST /v1/directions.lastDecision              JSON  (ETag)
     POST /v1/directions.list                      JSON  (ETag)
     POST /v1/directions.makeAbsent                JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.makeAvailable             JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.moveMatchToTable          JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.pageHtml                  JSON  (ETag)
     POST /v1/directions.pairingSheetHtml          JSON  (ETag)
     POST /v1/directions.participants              JSON  (ETag)
     POST /v1/directions.previewConfig             JSON
     POST /v1/directions.reinstate                 JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.reopen                    JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.setConfig                 JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.setTables                 JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.slots                     JSON  (ETag)
     POST /v1/directions.standings                 JSON  (ETag)
     POST /v1/directions.standingsCsv              JSON  (ETag)
     POST /v1/directions.startMatch                JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.tableGrid                 JSON  (ETag)
     POST /v1/directions.updatePair                JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.updateParticipant         JSON  (Idempotency-Key)  (If-Match)
     POST /v1/directions.withdraw                  JSON  (Idempotency-Key)  (If-Match)
   events
     POST /v1/events.alias.list                    JSON
     POST /v1/events.alias.remove                  JSON
     POST /v1/events.alias.set                     JSON
     POST /v1/events.alias.suggest                 JSON
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
     POST /v1/filters.setPinned                    JSON
     POST /v1/filters.update                       JSON
   gammonnet
     POST /v1/gammonnet.analyzeMissing             custom
     POST /v1/gammonnet.analyzeMissing.cancel      custom
     POST /v1/gammonnet.compare                    custom
     POST /v1/gammonnet.cubeMatrix                 custom
     POST /v1/gammonnet.evaluate                   custom
     POST /v1/gammonnet.sweepStale                 custom
   history
     POST /v1/history.clear                        JSON
     POST /v1/history.load                         JSON
     POST /v1/history.save                         JSON
   imports
     POST /v1/imports.batch                        custom
     POST /v1/imports.batch.cancel                 JSON
     POST /v1/imports.batch.status                 JSON
     POST /v1/imports.bgf                          custom
     POST /v1/imports.cancel                       custom
     POST /v1/imports.db                           custom
     POST /v1/imports.files                        JSON
     POST /v1/imports.gnubg                        custom
     POST /v1/imports.json                         custom
     POST /v1/imports.list                         JSON
     POST /v1/imports.ogxm                         custom
     POST /v1/imports.position                     custom
     POST /v1/imports.report                       JSON
     POST /v1/imports.studyQueue                   JSON
     POST /v1/imports.xg                           custom
   lessons
     POST /v1/lessons.addStep                      JSON  (Idempotency-Key)
     POST /v1/lessons.create                       JSON  (Idempotency-Key)
     POST /v1/lessons.delete                       JSON
     POST /v1/lessons.get                          JSON
     POST /v1/lessons.list                         JSON
     POST /v1/lessons.removeStep                   JSON
     POST /v1/lessons.reorderSteps                 JSON
     POST /v1/lessons.update                       JSON
     POST /v1/lessons.updateStep                   JSON
   librarySettings
     POST /v1/librarySettings.load                 JSON
     POST /v1/librarySettings.save                 JSON
   maintenance
     POST /ops/maintenance.vacuum                  custom
     POST /v1/maintenance.reencode                 custom
   matches
     POST /v1/matches.count                        JSON
     POST /v1/matches.createGame                   JSON
     POST /v1/matches.createMove                   JSON
     POST /v1/matches.delete                       JSON
     POST /v1/matches.duplicates                   JSON
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
     POST /v1/matches.scoreMoves                   JSON
     POST /v1/matches.setLastVisitedPosition       JSON
     POST /v1/matches.swapPlayers                  JSON
     POST /v1/matches.update                       JSON
     POST /v1/matches.updateComment                JSON
   met
     POST /v1/met.import                           JSON
     POST /v1/met.list                             JSON
     POST /v1/met.ofAnalysis                       JSON
     POST /v1/met.setCurrent                       JSON
   metadata
     POST /v1/metadata.counts                      JSON
     POST /v1/metadata.countsEstimate              JSON
     POST /v1/metadata.version                     JSON
   players
     POST /v1/players.alias.list                   JSON
     POST /v1/players.alias.remove                 JSON
     POST /v1/players.alias.set                    JSON
     POST /v1/players.alias.suggest                JSON
   positions
     POST /v1/positions.count                      JSON
     POST /v1/positions.delete                     JSON
     POST /v1/positions.epc                        JSON
     POST /v1/positions.exists                     JSON
     POST /v1/positions.explain                    JSON
     POST /v1/positions.fromOGID                   JSON
     POST /v1/positions.fromXGID                   JSON
     POST /v1/positions.fromXGP                    JSON
     POST /v1/positions.indexOf                    JSON
     POST /v1/positions.legalMoves                 JSON
     POST /v1/positions.list                       NDJSON
     POST /v1/positions.listIds                    JSON
     POST /v1/positions.load                       JSON
     POST /v1/positions.loadByIds                  JSON
     POST /v1/positions.parseText                  JSON
     POST /v1/positions.reclassifyPhases           JSON
     POST /v1/positions.repairCrawford             JSON
     POST /v1/positions.save                       JSON  (Idempotency-Key)
     POST /v1/positions.similar                    JSON
     POST /v1/positions.update                     JSON
   quiz
     POST /v1/quiz.gradeChecker                    JSON
     POST /v1/quiz.gradeCheckerMove                JSON
     POST /v1/quiz.gradeCube                       JSON
   rencontres
     POST /v1/rencontres.attach                    JSON  (Idempotency-Key)  (If-Match)
     POST /v1/rencontres.create                    JSON  (Idempotency-Key)
     POST /v1/rencontres.detach                    JSON  (Idempotency-Key)  (If-Match)
     POST /v1/rencontres.get                       JSON  (ETag)
     POST /v1/rencontres.list                      JSON  (ETag)
     POST /v1/rencontres.pageHtml                  JSON  (ETag)
     POST /v1/rencontres.ranking                   JSON
     POST /v1/rencontres.setBreaks                 JSON  (Idempotency-Key)  (If-Match)
     POST /v1/rencontres.setEventRooms             JSON  (Idempotency-Key)  (If-Match)
     POST /v1/rencontres.setTableOutOfService      JSON  (Idempotency-Key)  (If-Match)
     POST /v1/rencontres.setTables                 JSON  (Idempotency-Key)  (If-Match)
     POST /v1/rencontres.trash                     JSON  (Idempotency-Key)  (If-Match)
     POST /v1/rencontres.update                    JSON  (Idempotency-Key)  (If-Match)
   rollout
     POST /v1/rollout.filter                       custom
     POST /v1/rollout.filter.cancel                custom
     POST /v1/rollout.list                         JSON
     POST /v1/rollout.position                     custom
   search
     POST /v1/search.count                         JSON
     POST /v1/search.find                          NDJSON
     POST /v1/search.ids                           JSON
     POST /v1/search.indexOf                       JSON
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
     POST /v1/stats.headToHead                     JSON
     POST /v1/stats.matchBadges                    JSON
     POST /v1/stats.matchDetail                    JSON
     POST /v1/stats.matchMoveGrades                JSON
     POST /v1/stats.playerContrast                 JSON
     POST /v1/stats.playerNames                    JSON
     POST /v1/stats.playerTable                    JSON
     POST /v1/stats.positionIdsByMatch             JSON
     POST /v1/stats.positionIdsBySelection         JSON
     POST /v1/stats.positionIdsByTournament        JSON
     POST /v1/stats.prByWindow                     JSON
     POST /v1/stats.ranking                        JSON
     POST /v1/stats.rebuildMatchStats              JSON
     POST /v1/stats.recurringErrors                JSON
     POST /v1/stats.report                         JSON
     POST /v1/stats.studyIds                       JSON
     POST /v1/stats.tournamentBadges               JSON
     POST /v1/stats.training                       JSON
   tenant
     POST /ops/tenant.purge                        custom
   tenants
     POST /v1/tenants.quota                        JSON
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
   training
     POST /v1/training.missed                      JSON
     POST /v1/training.numberStats                 JSON
     POST /v1/training.save                        JSON  (Idempotency-Key)
     POST /v1/training.sessions                    JSON
   transcriptions
     POST /v1/transcriptions.abandon               JSON  (Idempotency-Key)  (If-Match)
     POST /v1/transcriptions.apply                 JSON  (Idempotency-Key)  (If-Match)
     POST /v1/transcriptions.close                 JSON
     POST /v1/transcriptions.create                JSON
     POST /v1/transcriptions.editMatch             JSON
     POST /v1/transcriptions.exportMat             JSON
     POST /v1/transcriptions.finish                JSON  (Idempotency-Key)  (If-Match)
     POST /v1/transcriptions.get                   custom
     POST /v1/transcriptions.list                  JSON
     POST /v1/transcriptions.losses                JSON
     POST /v1/transcriptions.open                  JSON
     POST /v1/transcriptions.redo                  JSON  (Idempotency-Key)  (If-Match)
     POST /v1/transcriptions.undo                  JSON  (Idempotency-Key)  (If-Match)
   trash
     POST /v1/trash.count                          JSON
     POST /v1/trash.deleteCollection               JSON
     POST /v1/trash.deleteComment                  JSON
     POST /v1/trash.deletePosition                 JSON
     POST /v1/trash.discard                        JSON
     POST /v1/trash.empty                          JSON
     POST /v1/trash.list                           JSON
     POST /v1/trash.restore                        JSON
   events
     GET  /v1/events                               SSE


Idempotence
-----------

La plupart des méthodes n'ont besoin d'aucun mécanisme particulier : les
lectures sont sans effet de bord, et les écritures de ``positions.*`` sont
idempotentes dans leur effet grâce au hachage Zobrist du contenu — enregistrer
deux fois la même position renvoie la même ligne, jamais un doublon. 46 méthodes
acceptent un en-tête ``Idempotency-Key`` optionnel : celles dont deux appels
sont deux effets distincts, et ``positions.save``, dont la réponse dit si
l'appel a créé la position (``created``) et le dirait faux si elle était
rejouée après une réponse perdue. Un appel rejoué avec la même clé renvoie le
résultat de la première tentative au lieu de répéter son effet — voir la marque
« (Idempotency-Key) » dans le tableau ci-dessus. Aucune autre méthode n'a besoin ou n'accepte cet en-tête.

Lectures conditionnelles
------------------------

Les méthodes marquées « (ETag) » rendent un en-tête ``ETag``. Le renvoyer dans
``If-None-Match`` obtient une réponse ``304`` sans corps tant que rien de ce que la
méthode lit n'a changé — voir :ref:`headless_direction`.
