<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot J — Étape 4 : chantiers de fond (L)

Chacun mérite sa décision (ADR) avant lancement, souvent une session de
grilling, et parfois un rapport de deep search (`prompts-deep-search.md`).
Ne pas les commencer « en passant ». Ordre proposé : J.1 → J.2 → J.3 → J.4 ;
J.5 à J.7 selon la demande ; J.8 et J.9 opportunistes ; J.10 écarté.

---

## J.1 — Classification automatique du type de jeu, thèmes d'erreur, erreurs récurrentes

La question à laquelle blunderDB ne sait pas répondre et que tous ses
utilisateurs se posent : « montre-moi mes erreurs en holding game ». #60 a
raison qu'un bundle de filtres sauvegardés n'y suffit pas ; un classifieur, si.

- **J.1a Type de jeu** (course, holding, backgame, blitz, prime vs prime,
  containment, ace-point, crunch, mutual holding) par heuristiques publiées
  (prompt P5 : Robertie, Magriel, Woolsey, Trice, gnubg `ClassifyPosition`,
  seuils). Étiquette **dérivée, jamais éditable, recalculée à l'ouverture**
  (comme la phase I.8), donc jamais exportée comme vérité. Corpus étiqueté à
  la main (200 positions) pour valider ; règles publiées dans la doc.
- **J.1b Thèmes d'erreur** (trop passif, trop agressif, timing, gammon
  sous-estimé, take trop lâche, double trop tôt/tard, sécurité) dérivés de la
  comparaison coup joué / meilleur coup sur des axes mesurables (Δ gammon,
  Δ blots, Δ points faits, direction de l'erreur de videau qui existe déjà).
  5-6 thèmes défendables, pas 20 (prompt P18 pour ce que font Lichess,
  Chess.com, GTO Wizard).
- **J.1c Erreurs récurrentes** : regroupement des blunders par signature
  (type + thème + motif du coup manqué), classé par coût MWC cumulé — la
  promesse du nom du produit, tenue à moitié aujourd'hui.
- Prérequis : I.8, I.10 (l'onglet Stats qui les ventile), B.10 (pagination :
  le volume traversant l'IPC augmente). Schéma : colonnes indexées, vague 2.17.0.
- Risque principal : une taxonomie contestée est pire que pas de taxonomie ;
  mitigé par « dérivée, non éditable, règles publiées ».

## J.2 — Rollouts tronqués

La seule façon de départager deux coups à 0,005 et la principale raison de
retourner dans XG. 288/1296 parties tronquées à N coups puis évaluées à
2-ply, dés en miroir, écart-type et intervalle de confiance affichés, arrêt
quand l'intervalle sépare les deux meilleurs (prompt P8 : variance, critères
d'arrêt, paramètres publiés de gnubg/XG).
- C'est une **Configuration** au sens de CONTEXT.md (nouveau `EngineVersion`,
  jauge de force) → décision amont gammonNet d'abord ; l'infrastructure
  d'ici est prête (`Searcher` réutilisable, `Verdict` exporté pour ça,
  parallélisme #147/#148, travail long/annulable/reprenable d'`analyze`).
- ADR-0013 : un rollout n'écrase pas une analyse importée → une seconde
  `Analysis` étiquetée, ou un régime supplémentaire.
- Préalable de crédibilité : I.14.
- Ticket successeur de #119 (A.14).

## J.3 — Similarité : « positions comme celle-ci »

Personne ne rencontre deux fois la même position ; c'est ce que l'utilisateur
croit demander quand il cherche par structure. Métrique : distance sur le
vecteur de 26 points, ou sur l'espace latent de gammonNet (l'avant-dernière
couche est un embedding gratuit et déjà dans le binaire) ; index : scan
linéaire SIMD suffit sous 100 k positions, sinon LSH/HNSW pur Go (prompt P7).

- Prototyper d'abord (skill `prototype`) : 50 positions, deux métriques, un
  joueur juge si « proche » selon la métrique = proche selon lui.
- Bouton sur la position courante, jeton `like<id>`, route.

## J.4 — Mode quiz avec PR d'entraînement

Anki mémorise ; un quiz **teste**. N positions tirées d'un filtre, chronomètre
optionnel, le coup se **joue sur le plateau** (ou l'action de videau se
clique), erreur mesurée contre l'analyse enregistrée, session rendue comme un
**PR de quiz** comparable au PR réel — la métrique d'entraînement que personne
ne relie aux erreurs du joueur.
- Le mode édition existe ; la reconnaissance « ce coup légal correspond à tel
  candidat » se normalise par la génération de coups du moteur.
- I.17 (micro-entraînements) en est le prototype ; J.4 est le module complet.
- Prompt P10 : pratiques pédagogiques et FSRS à note dérivée d'une mesure
  (I.20/I.19 en profitent).

## J.5 — Interface web sur `serve`, puis mobile en consultation

135 routes et aucun client hors bureau. Un front web **limité** (consulter,
chercher, réviser Anki ; pas d'édition) donne tablette et mobile sans
réécrire l'application. Prérequis : I.22 (rendu du plateau extrait du
contexte Wails), G.8 (contrat d'API). Risque : un second front à maintenir →
périmètre verrouillé par ADR.

## J.6 — Mode club / coach sur le serveur

`serve` est étanche par tenant ; un coach veut l'inverse contrôlé : partager
une bibliothèque en lecture, recevoir les matchs des élèves, voir leurs stats.
**Nouvelle notion** (relation entre tenants) qui change la nature du produit et
heurte le glossaire : à griller (skill `grilling`) avant tout code ; ADR-0005
reste (l'auth au proxy) ; les parcours pédagogiques (I.21) et l'index de bases
(I.25) couvrent déjà une partie du besoin sans serveur.

## J.7 — Moteur : réseau distillé, cache persistant

- Réseau distillé 60-100 k MAC (priorité 1 du rapport P4 : ×5-9, le seul gain
  qui « survit à la recherche ») — **amont**, nouvelle Configuration, jauge de
  force, bases périmées → dépend de C.4 (`--stale` avec déclencheur) pour être
  supportable.
- Cache d'évaluation persistant (`evalCache` meurt avec le `Searcher`) : exact
  par construction (P4, technique 5), donc implémentation locale sans jauge ;
  contrôle de régression bit-exact.
- NEON arm64 (#151) après C.2 et une mesure du coût réel sur M1.

## J.8 — Expliquer un blunder en une phrase

« Vous perdez 62 mMWC : le coup joué laisse trois blots dans la zone alors que
le meilleur coup fait votre point 5. » Génération depuis les différences
mesurables, sans LLM ; ne parler que quand une règle est confiante. J.1b en
est la moitié.

## J.9 — Assistant optionnel via Ollama

Option 1 de #38 : strictement opt-in, jamais embarqué, non déterministe, après
I.27 (la grammaire d'intentions couvre 80 % du besoin à 0 Mo).

## J.10 — Jouer contre gammonNet — écarté

Changement de nature du produit (moteur de jeu), surface énorme (règles,
saisie, videau, historique). J.4 apporte la saisie d'un coup ; si un jour la
demande existe, c'est un autre produit (gammonGo). Une ligne dans une ADR
suffit pour ne pas rouvrir la question.

---

## Récapitulatif des décisions à prendre avant de lancer

| Chantier | Décision | Outil |
|---|---|---|
| J.1 | Étiquette dérivée jamais éditable ; taxonomie retenue | ADR + P5 + P18 |
| J.2 | Configuration amont ; seconde `Analysis` ou régime | ADR amont + P8 |
| J.3 | Métrique ; index | prototype + P7 |
| J.4 | Reconnaissance du coup ; note dérivée ou auto-évaluée | ADR + P10 |
| J.5 | Périmètre du client web | ADR |
| J.6 | Relation entre tenants | grilling + ADR |
| J.7 | Distillation amont ; cache persistant local | gammonNet ADR-0003 |
