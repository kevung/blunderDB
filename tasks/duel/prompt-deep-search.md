# Prompt de recherche approfondie — jouer un match contre le moteur (Duel)

À coller tel quel dans Claude web, mode recherche approfondie. Le rapport rendu se verse dans
`docs/recherche/P20-duel-bot-horloge-des.md` et s'ajoute au tableau de
`docs/recherche/README.md`.

---

Tu es chargé d'une recherche documentaire approfondie pour un logiciel libre de backgammon.
Réponds en français. Je veux des faits sourcés, pas des recommandations générales.

## Contexte

blunderDB est une base de données d'erreurs de backgammon (application de bureau Go + Svelte,
qui tourne aussi en serveur HTTP sans interface). Elle embarque un évaluateur neuronal,
gammonNet, dont la force et le coût sont mesurés :

| Profondeur | Performance Rating (arbitre gnubg 3-ply) | Coût par décision (16 cœurs) |
|---|---|---|
| 0-ply | 1,09 | 1,4 ms |
| 1-ply | 0,50 | — |
| 2-ply filtré | 0,27 | 100 à 400 ms |
| 3-ply | non mesuré | 8,4 s |

À 2-ply il est mesuré équivalent à GNU Backgammon 2-ply (50,42 % de MWC sur 50 000 paires
dupliquées). Ses faiblesses connues : les backgames (erreur 1,9 fois la moyenne), le videau à
2-away/4-away, et les positions « trop bon pour doubler ». Il dispose d'un modèle de videau de
Janowski, d'une table d'équité de match, de tables de bearoff exactes et d'un rollout à
réduction de variance.

Nous ajoutons un mode où l'utilisateur **joue un match complet** contre ce moteur (un « Bot »),
sous l'arbitrage du logiciel (dés lancés par le logiciel, règles imposées, horloge), puis revoit
son match analysé. Le même arbitre sera exposé par une API HTTP à un site web tiers. Nous
voulons aussi enregistrer la **durée de chaque décision** du joueur pour la montrer à la revue.

Ce que nous savons déjà et qu'il est inutile de redécrire : le code de GNU Backgammon
(`ComputerTurn`, niveaux à bruit 0,060/0,050/0,040/0,015 appliqué aux sorties du réseau et
déterministe par position, abandon évalué à 0-ply seulement en course, filtres de coups, sept
générateurs de dés, mode tuteur à seuil 0,03, absence d'horloge et de temps par coup).

## Questions

### A. Affaiblir un bot de façon crédible

1. Quelles méthodes d'affaiblissement sont documentées pour les bots de backgammon (eXtreme
   Gammon, BGBlitz, Snowie, Backgammon Galaxy, Backgammon NJ / Ace, Open Sage / Backgammon
   Sage, bgammon.org, Wildbg) : bruit sur l'évaluation, tirage parmi les N meilleurs coups,
   température sur l'équité, réduction de profondeur, budget d'erreur visant un PR cible ?
   Pour chacun, la définition exacte de ses niveaux si elle est publiée.
2. Existe-t-il des travaux sur un bot **qui se trompe comme un humain** plutôt que
   uniformément ? Je pense à Maia (échecs) et à ses suites, au « skill level » et à
   `UCI_Elo` de Stockfish, aux niveaux de KataGo. Qu'est-ce qui en est transposable à un jeu
   à dés, et y a-t-il des tentatives au backgammon (modèle d'erreur appris sur un corpus de
   matchs humains, erreurs concentrées sur certains types de position ou sur le videau) ?
3. Comment calibrer un niveau : relation publiée entre Performance Rating et classement Elo
   (formule FIBS, tables de Kees van den Doel, correspondance PR ↔ niveau de joueur de XG et
   de la BMAB), et volume de matchs nécessaire pour vérifier qu'un niveau « PR 8 » joue bien
   à PR 8.
4. Déterminisme : vaut-il mieux qu'un bot affaibli refasse la même erreur sur la même
   position (gnubg) ou tire au sort ? Arguments et retours d'utilisateurs.

### B. La politique de match au-delà de l'évaluateur

5. Règles d'abandon pratiquées par les bots (XG, BGBlitz, Galaxy) : quand ils proposent, ce
   qu'ils acceptent, abandon de gammon et de backgammon, et les abus connus côté humain.
6. Décisions de videau d'un bot en match : traitement du « trop bon », du double optionnel,
   du beaver et du raccoon en argent, du videau automatique post-Crawford, et les erreurs
   typiques qu'un modèle de Janowski commet face à un rollout.
7. Automatismes de confort admis ou imposés par les logiciels et les sites (lancer
   automatique, coup forcé, bearoff automatique, fin de partie abrégée quand l'issue est
   certaine) et ce que les règlements de tournoi en ligne en disent.

### C. Temps de réflexion du bot

8. Gestion du temps d'un moteur de backgammon sous horloge : allocation par difficulté de
   décision (détecter une décision triviale à faible profondeur, approfondir les décisions
   serrées, micro-rollout), ce que font XG et BGBlitz, et ce que la littérature dit du
   « pondering » dans un jeu stochastique (le lancer invalide-t-il le travail ?).
9. Délai artificiel : quelles durées de réponse sont perçues comme naturelles, et y a-t-il
   des retours sur des bots jugés trop rapides ou trop lents ?

### D. Horloges de backgammon

10. Règles d'horloge officielles en vigueur : USBGF, WBF, EUBGF, UK Backgammon Federation,
    BMAB, tournois en ligne (Galaxy, Heroes, Backgammon Studio). Pour chacune : réserve de
    temps (par point de longueur de match ?), délai par coup (Bronstein, délai simple,
    incrément Fischer), ce qui arrête l'horloge (ramasser les dés), sanction au dépassement
    (perte du match, pénalité en points), cas du videau et de l'abandon.
11. Comment les logiciels implémentent ces horloges (XG, gnubg n'en a pas, BGBlitz, les
    sites) et quelles cadences ils proposent par défaut.

### E. Le temps par décision comme donnée d'analyse

12. Quels formats de fichier enregistrent un temps par coup ou un temps restant (format `.xg`
    d'eXtreme Gammon : champs exacts ; exports de Galaxy et de Heroes ; SGF de gnubg ;
    `.mat`) ? Lesquels sont relus par les outils d'analyse ?
13. Recherche sur le lien entre temps de réflexion et erreur : littérature aux échecs
    (zeitnot, temps passé contre qualité du coup, données Lichess) et tout ce qui existe au
    backgammon. Quelles métriques sont retenues (temps par type de décision, temps contre
    difficulté de la position, erreurs sous pression de temps) ?
14. Comment les outils montrent le temps à la revue (graphique de temps de Lichess et de
    Chess.com, affichages de XG et de Galaxy) et ce que les utilisateurs en retirent.

### F. Des dés auxquels on croit

15. Le soupçon « le bot triche aux dés » : son ampleur documentée, et les mécanismes que les
    logiciels et sites opposent — graine affichée, engagement cryptographique (hash de la
    graine publié avant le match, graine révélée après), source externe (random.org), dés
    manuels, fichier de dés, statistiques de chance après match. Lesquels sont réellement
    utilisés et lesquels convainquent ?
16. Schémas de dés vérifiables pour un serveur arbitrant deux humains à distance
    (commit-reveal à deux parties, graine combinée) : description, coût, implémentations
    existantes dans des jeux à dés ou de cartes.

### G. L'expérience de jeu dans les logiciels existants

17. Pour XG, gnubg, BGBlitz, Backgammon NJ / Ace, Galaxy contre bot et bgammon.org : les
    options offertes au lancement d'un match (longueur, niveau, horloge, règles de session,
    couleur, sens du plateau), le mode tuteur (seuil, moment de l'avertissement), la
    politique d'annulation d'un coup, et l'enchaînement vers la revue du match terminé.
18. Protocoles d'API de jeux au tour par tour ouverts à un client tiers (Lichess Board API,
    protocole de bgammon.org, FIBS CLIP) : forme de l'état, flux d'événements, reconnexion,
    idempotence d'un coup, gestion de l'horloge côté serveur.

## Forme du rapport

- Un **TL;DR** de cinq à huit points, puis une section par lettre (A à G).
- Chaque affirmation chiffrée porte l'un de ces marqueurs : `[MESURE]` (mesure publiée et
  reproductible), `[ÉDITEUR/DÉCLARÉ]` (affirmé par l'éditeur ou l'auteur, non vérifié),
  `[EXTRAPOLÉ]` (déduction de ta part), `[NON TROUVÉ]` (cherché sans résultat — dis-le
  plutôt que de combler).
- Cite les sources avec leur URL, et privilégie les sources primaires : code source, manuels,
  règlements officiels, articles, fils de forum d'auteurs de moteurs.
- Termine par la liste des questions restées sans réponse fiable.
