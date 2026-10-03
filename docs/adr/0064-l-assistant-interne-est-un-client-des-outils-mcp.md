# ADR-0064 — L'assistant interne est un client des outils MCP de la fenêtre

Statut : acceptée.
Complète : ADR-0059 (le serveur MCP ; son point 6 annonçait ce lot).
Voir aussi : ADR-0005, ADR-0007, ADR-0019, ADR-0055.

## Contexte

ADR-0059 a livré les outils et deux transports (HTTP du démon, stdio). Il restait le troisième,
la GUI, et l'assistant qui en est le client : un joueur qui n'a ni Claude Desktop ni Claude
Code doit pouvoir écrire une phrase dans blunderDB et voir le résultat à l'écran.

## Décision

1. **La GUI sert les outils sur `127.0.0.1`, et seulement sur demande.** Réglage
   *Assistant et MCP* : interrupteur (désactivé par défaut), port (8765 par défaut), même
   interrupteur d'écriture que `mcp --write`. Garde anti-rebinding du SDK active et
   `http.CrossOriginProtection` en plus : ici l'attaquant est une page web que l'utilisateur
   visite. Écouter sur un port est une décision de l'utilisateur, pas un effet du lancement.
2. **Le moteur suit le fichier ouvert.** `database.CurrentStore` rend le `Storage` qui emprunte
   la connexion de la fenêtre et sa génération ; le gestionnaire `/v1` d'`internal/server` est
   reconstruit quand la génération bouge. Les outils passent donc par `/v1` comme sur les deux
   autres transports (ADR-0059 §2) et voient les lignes que la fenêtre montre ; aucune base
   n'est ouverte deux fois.
3. **Deux outils d'affichage**, `open_view` et `show_position` (`mcp.DisplayTools`), en lecture
   pour la base : ils changent l'écran, pas les données. Ils ne font que demander, par
   événement Wails ; la fenêtre agit sur son propre état — un onglet de vue (`viewStore`) et
   une recherche passée par la barre de commande, comme si l'utilisateur l'avait tapée. Hors
   des modes où une recherche tourne (match, collection…), la demande est refusée par son nom.
4. **L'assistant est un client MCP en processus** (`pkg/blunderdb/assistant`, transport en
   mémoire du SDK) des mêmes outils : pas un second accès aux données. Fournisseur compatible
   OpenAI (`/chat/completions` avec appels d'outils) ; préréglages Ollama (défaut, local),
   Groq, OpenRouter, Gemini, Anthropic, autre. Aucun modèle n'est embarqué.
5. **Strictement opt-in.** Désactivé par défaut ; un fournisseur distant n'est utilisé
   qu'après l'accord explicite de l'utilisateur à l'avertissement de confidentialité, accord
   lié au fournisseur (en changer le redemande). La clé d'API va dans le trousseau du système
   (`go-keyring`), jamais dans la base ni dans le fichier de réglages.
6. **Une phrase devient une vue.** La vue qu'ouvre l'assistant porte le nom de la phrase, pas
   l'étiquette que le modèle propose ; la recherche y est lancée, la fenêtre y bascule, et ses
   jetons restent dans l'historique du panneau Recherche.
7. **Toute écriture est préparée puis confirmée.** Les outils d'écriture ne sont offerts au
   modèle que si l'interrupteur d'écriture est mis, et même alors chaque appel suspend le tour :
   l'utilisateur voit l'outil et ses arguments, confirme ou refuse ; un refus est rendu au
   modèle comme tel.
8. **Le texte libre du modèle est marqué.** Le panneau sépare ce que les outils ont rendu (des
   faits de la base) de ce que le modèle a écrit, étiqueté *Texte libre du modèle*.
9. **La recommandation de modèle vient d'une mesure.** Un corpus (six intentions dans les neuf
   langues de l'application, `pkg/blunderdb/assistant/testdata/corpus.json`) et un banc
   (`go test -tags assistantbench`) envoient chaque phrase à un vrai modèle par les vrais outils
   sur la base de démonstration, et notent ce qu'il a fait : la vue ouverte et la forme
   canonique de sa requête, ou l'outil appelé. La documentation ne recommande un modèle
   qu'avec un score de ce banc ; sans mesure, elle n'en nomme aucun.

## Conséquences

- L'assistant est dans le panneau Recherche (sous-onglet visible seulement s'il est activé),
  pas dans un onglet de plus : pas de nouveau raccourci, et les jetons sont là où on les lit.
- Rien n'est écrit par une lecture : ouvrir une vue ne touche pas la base (ADR-0007).
- Une intention nouvelle se mesure en ajoutant une ligne par langue au corpus ;
  `TestCorpusCoversEveryLanguageAndParses` refuse un corpus déséquilibré ou une requête que la
  grammaire ne lit pas.
