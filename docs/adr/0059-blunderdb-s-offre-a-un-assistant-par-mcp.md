# ADR-0059 — blunderDB s'offre à un assistant par MCP

Statut : acceptée.
Écarte : une « grammaire d'intentions » propre à blunderDB comme interface d'un assistant.
Laisse intacte : ADR-0005 (le démon n'authentifie personne).
Voir aussi : ADR-0005, ADR-0019, ADR-0055, ADR-0057.

## Contexte

Un joueur veut demander en phrases ce que la barre de commande demande en jetons :
« mes erreurs de videau au score 2-4 », « explique-moi ce coup ». Une grammaire fermée ne
couvre pas la phrase qu'on écrit vraiment ; un modèle embarqué pèse des centaines de Mo et
le chantier taille du binaire a refusé UPX pour quelques-uns. Le Model Context Protocol
renverse la question : blunderDB n'embarque aucun modèle, il **offre des outils** à
l'assistant que l'utilisateur a déjà (Claude Code, Claude Desktop, un client local). Le
serveur est cette offre ; l'assistant interne en est un client (ADR-0064).

## Décision

1. **Un paquet, `pkg/blunderdb/mcp`, sur le SDK Go officiel**
   (`github.com/modelcontextprotocol/go-sdk`). Sa version 1.8 demande Go 1.25 : la directive
   `go` de `go.mod` ne bouge pas. Aucun transport n'est écrit à la main.
2. **Les outils appellent `/v1` en processus**, par le gestionnaire chaîné du démon, comme
   `call`. Une seule vérification de tenant, une seule limite de débit, une seule enveloppe
   d'erreur, les mêmes règles métier : un outil ne réimplémente rien et ne voit rien que
   `/v1` ne montre pas. Le paquet n'importe pas `internal/server` ; il reçoit un
   `http.Handler`.
3. **Des outils de haut niveau, pas un par route.** Plus de deux cents routes noieraient le
   modèle ; un outil répond à une question de joueur et compose les routes qu'il faut. Liste
   et raison :

   | Outil | Routes | Pourquoi un outil |
   |---|---|---|
   | `database_overview` | `metadata.*`, `stats.dateRange`, `stats.playerNames` | le premier appel de toute conversation |
   | `search_positions` | `search.parse`, `search.query` | la grammaire de la barre, décrite dans l'outil ; forme canonique et jetons sans effet rendus |
   | `search_comments` | `comments.search` | ce que l'utilisateur a écrit |
   | `saved_searches` | `filters.list` | le vocabulaire que l'utilisateur s'est déjà fait |
   | `get_position` | `positions.load`, `analyses.load`, `comments.text` | une position, son analyse tronquée aux meilleurs coups, ce qui a été joué |
   | `explain_error` | `positions.explain` | J.8 : un thème mesuré, jamais une phrase inventée |
   | `similar_positions` | `positions.similar` | « d'autres positions comme celle-ci » |
   | `decode_position` | `positions.parseText` | lire un XGID collé sans l'enregistrer |
   | `legal_moves` | `positions.legalMoves` | vérifier un coup avant de le juger |
   | `race_epc` | `positions.epc` | la course, mesurée par la base exacte |
   | `list_players` | `stats.playerNames` | les graphies d'un même joueur |
   | `player_stats` | `stats.compute` | PR global, pions, videau, phases, tournois |
   | `recurring_errors` | `stats.compute`, `stats.recurringErrors` | les pertes qui reviennent : blunders, actions de videau, étiquettes |
   | `training_stats` | `stats.training` | le PR du quiz et la rétention Anki, par fenêtre, contre le PR réel |
   | `list_matches`, `get_match` | `matches.list`, `matches.get`, `stats.matchDetail` | un match et ses deux performances |
   | `list_tournaments` | `tournaments.list` | |
   | `list_collections`, `collection_positions` | `collections.*` | le travail déjà rangé |
   | `study_decks` | `anki.listDecks` | ce qui est dû à la révision |
   | `quiz_draw`, `quiz_grade` | `search.query`, `quiz.grade*` | un quiz conversationnel, la réponse cachée jusqu'à la note |

   D'autres outils composent de la même façon les routes de l'évaluation (`evaluate`,
   gammonNet sans rien stocker), du rollout (`rollout`, ADR-0060), de la révision Anki
   (`anki_next`, `anki_review`), de la transcription (`transcribe_*`), de la direction en
   lecture (`direction_list`, `direction_standings`, `direction_season`), des Leçons
   (`list_lessons`, `lesson`) et de la lecture à travers les tenants (`club_*`, ADR-0065) ;
   `pkg/blunderdb/mcp/tools.go` et `club.go` en tiennent la liste. Hors de la liste, et
   pourquoi : les gestes de la Direction (un geste à version, ADR-0057, reste au client de
   direction), l'import et l'export (des fichiers, pas des phrases), l'abandon d'un brouillon
   de transcription (aucun outil n'efface).
4. **Lecture seule par défaut.** Les outils qui écrivent — `save_position`,
   `comment_position`, `create_collection`, `add_to_collection`, `anki_review`, `rollout` avec
   `store`, et les gestes de transcription (`transcribe_create`, `transcribe_open`,
   `transcribe_apply`, `transcribe_undo`, `transcribe_redo`, `transcribe_finish`, servis
   seulement par un démon lancé avec `--transcription`, chacun nommant la révision sur
   laquelle il a été tapé) — ne sont offerts que derrière un drapeau : `blunderdb mcp
   --write`, `serve --mcp-write`, `Config.MCPWrite` pour un embarqueur. Aucun outil n'efface :
   `transcribe_finish` remplace le brouillon par son Match. `save_position` lève la provenance « importée seule »
   (ADR-0001), comme un collage dans l'application.
5. **Deux transports pour un client extérieur.**
   - **HTTP**, `POST /mcp`, monté par `internal/server` et donc par le démon et par
     `pkg/blunderdb/server` (gammonGo l'hérite). Sans état : chaque requête se suffit, aucune
     session n'attache un client à une instance. `/mcp` n'est pas un chemin public : le
     middleware de tenant exige `X-Tenant-ID` comme sur `/v1`, et chaque outil appelle `/v1`
     sous le tenant de **sa** requête, jamais un tenant par défaut. Chaque appel `/v1` d'un
     outil retraverse toute la chaîne : il est journalisé, compté dans les métriques et
     imputé à la limite de débit du tenant, en plus du `POST /mcp` qui le porte. Ce coût est
     voulu — un outil ne doit pas contourner la limite — et rien n'en est exempté. La garde
     anti-rebinding DNS du SDK est coupée sur le démon seul (`Options.AllowRemoteHost`) : elle refuse un `Host` non local sur une connexion locale, ce
     qu'envoie justement le proxy authentifiant de l'ADR-0005. Aucune authentification
     n'est ajoutée.
   - **stdio**, `blunderdb mcp --db <fichier>` : l'assistant lance la commande ; tenant 1,
     journaux sur stderr, stdout réservé au protocole. Même sans `--write`, l'ouverture migre
     le schéma d'une base ancienne, comme `call`.
6. **Le troisième transport s'ajoute sans toucher aux outils.** Le serveur hébergé par la
   GUI (`internal/gui/mcphost.go` ; ouvrir une vue, montrer une position, ADR-0064) écoute
   sur localhost : il garde la
   garde anti-rebinding (`AllowRemoteHost` faux, le défaut) et y ajoute
   `CrossOriginProtection`, car là l'attaquant est une page web que l'utilisateur visite. `Options.Extensions`
   reçoit des fonctions `func(*Toolbox)` qui enregistrent leurs outils par `mcp.Add`, sous le
   même interrupteur d'écriture.
7. **Les unités restent celles de l'application** (ADR-0019) : erreurs en millipoints,
   équités telles que `/v1` les rend. Les listes sont bornées (20 par défaut, 200 au plus) :
   une réponse d'outil entre dans une fenêtre de contexte, pas sur un écran.

## Conséquences

- La description de `search_positions` cite `searchquery.Reference`, le texte que
  `blunderdb search --query-help` imprime : une seule source, la grammaire changée change les
  deux.
- Un outil nouveau = une fonction dans `pkg/blunderdb/mcp/tools.go`, appelée par un test
  client en mémoire sur la base de démonstration.
- L'assistant interne (ADR-0064) est un client MCP de ces outils, pas un second accès aux
  données.
