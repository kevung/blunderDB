# ADR-0059 — blunderDB s'offre à un assistant par MCP

Statut : acceptée.
Remplace : la piste « grammaire d'intentions » de #283, écartée (`cabdad533`).
Laisse intacte : ADR-0005 (le démon n'authentifie personne).
Voir aussi : ADR-0005, ADR-0019, ADR-0055, ADR-0057.

## Contexte

Un joueur veut demander en phrases ce que la barre de commande demande en jetons :
« mes erreurs de videau au score 2-4 », « explique-moi ce coup ». Une grammaire fermée ne
couvre pas la phrase qu'on écrit vraiment ; un modèle embarqué pèse des centaines de Mo et
le chantier taille du binaire a refusé UPX pour quelques-uns. Le Model Context Protocol
renverse la question : blunderDB n'embarque aucun modèle, il **offre des outils** à
l'assistant que l'utilisateur a déjà (Claude Code, Claude Desktop, un client local). Le
présent lot livre le serveur ; l'assistant interne (lot 2) en sera un client.

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
   | `recurring_errors` | `stats.compute` | les pertes qui reviennent : blunders, actions de videau, étiquettes |
   | `list_matches`, `get_match` | `matches.list`, `matches.get`, `stats.matchDetail` | un match et ses deux performances |
   | `list_tournaments` | `tournaments.list` | |
   | `list_collections`, `collection_positions` | `collections.*` | le travail déjà rangé |
   | `study_decks` | `anki.listDecks` | ce qui est dû à la révision |
   | `quiz_draw`, `quiz_grade` | `search.query`, `quiz.grade*` | un quiz conversationnel, la réponse cachée jusqu'à la note |

   Hors de la liste, et pourquoi : la Direction et la transcription (un geste à version,
   ADR-0057, n'est pas un outil de conversation), l'import et l'export (des fichiers, pas des
   phrases), gammonNet (un calcul de plusieurs minutes, pas un appel d'outil), le rollout
   (aucune route ne l'expose encore).
4. **Lecture seule par défaut.** Quatre outils écrivent — `save_position`,
   `comment_position`, `create_collection`, `add_to_collection` — et ne sont offerts que
   derrière un drapeau : `blunderdb mcp --write`, `serve --mcp-write`, `Config.MCPWrite` pour
   un embarqueur. Rien n'efface. `save_position` lève la provenance « importée seule »
   (ADR-0001), comme un collage dans l'application.
5. **Deux transports dans ce lot.**
   - **HTTP**, `POST /mcp`, monté par `internal/server` et donc par le démon et par
     `pkg/blunderdb/server` (gammonGo l'hérite). Sans état : chaque requête se suffit, aucune
     session n'attache un client à une instance. `/mcp` n'est pas un chemin public : le
     middleware de tenant exige `X-Tenant-ID` comme sur `/v1`, et chaque outil appelle `/v1`
     sous le tenant de **sa** requête, jamais un tenant par défaut. La garde anti-rebinding
     DNS du SDK est coupée : elle refuse un `Host` non local sur une connexion locale, ce
     qu'envoie justement le proxy authentifiant de l'ADR-0005. Aucune authentification
     n'est ajoutée.
   - **stdio**, `blunderdb mcp --db <fichier>` : l'assistant lance la commande ; tenant 1,
     journaux sur stderr, stdout réservé au protocole.
6. **Le troisième transport s'ajoute sans toucher aux outils.** Le serveur hébergé par la
   GUI (ouvrir une vue, montrer une position) n'est pas dans ce lot. `Options.Extensions`
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
- Le lot 2 (assistant interne) est un client MCP de ces outils, pas un second accès aux
  données.
