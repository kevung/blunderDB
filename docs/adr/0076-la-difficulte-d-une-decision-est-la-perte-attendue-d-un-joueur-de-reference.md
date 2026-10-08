# ADR-0076 — La difficulté d'une décision est la perte attendue d'un joueur de référence

Statut : acceptée.
Voir aussi : ADR-0046 (seuils d'erreur de la bibliothèque), ADR-0019 (échelle d'équité
affichée), `docs/recherche/P20-metriques-force-backgammon.md` section B (M3).

## Contexte

La perte de MWC d'une décision (ℓ) dit ce qu'un coup a coûté, pas s'il était difficile de
trouver le bon. Une même perte de 2 % est une inattention dans une position évidente et une
erreur pardonnable dans une position où quatre coups se tiennent à quelques millièmes. Le
PR mélange les deux et dépend de l'adversaire : celui qui crée des positions difficiles fait
monter l'erreur de l'autre. La métrique M3 de P20 corrige cela en comparant ℓ à la perte
qu'aurait subie, dans la même position, un joueur de référence.

Ses paramètres doivent être fixés **avant** de regarder ce qu'elle donne sur des matchs réels,
faute de quoi on les ajuste au résultat qu'on attend. Cette ADR les fixe.

## Décision

1. **Coûts des candidats.** Pour une décision de pions, chaque candidat de l'analyse stockée
   coûte son erreur d'équité (le premier candidat 0), en équité normalisée. Pour une décision
   de videau, les options sont celles du joueur qui décide, au sens d'`engine.CubeActionError` :
   le joueur qui a le videau choisit entre pas de double (coût : erreur de ND) et double (coût :
   min(erreur D/T, erreur D/P), la meilleure réponse adverse supposée) ; celui qui répond
   choisit entre prendre et passer, chacun coûtant son écart à la meilleure réponse. Les coûts
   sont décalés pour que le meilleur vaille 0. Une analyse sans coût exploitable n'a pas de
   difficulté.
2. **Joueur de référence.** Il choisit l'option i avec la probabilité
   π(i) = exp(−Δᵢ/τ) / Σⱼ exp(−Δⱼ/τ), Δ en équité normalisée, **τ = 0,025** (25 millièmes).
   La difficulté est d = Σᵢ π(i)·Δᵢ, puis convertie en MWC par la même conversion que la
   perte (`decisionMWCLoss`, linéaire dans l'équité normalisée à score et videau donnés) :
   d et ℓ sont dans la même unité, et la décision jouée a pour Δ exactement ℓ.
3. **Pourquoi τ = 0,025.** Sur une décision à deux options séparées de Δ, d = Δ/(1 + e^{Δ/τ}),
   maximale vers Δ ≈ 1,28τ ≈ 0,032 : la décision binaire la plus difficile est celle dont
   l'écart tombe entre « douteux » et « erreur » des barèmes usuels (XG : 0,02 / 0,04 ;
   bibliothèque par défaut : 0,050). Un écart de 0,1 n'y coûte plus que 0,002 (l'évidence
   pèse ~0), et une position à cinq candidats espacés de 0,01 vaut d ≈ 0,012, l'erreur
   moyenne d'un joueur de PR ≈ 6. C'est un a priori de joueur fort (ordre du PR 5), pas un
   étalonnage : un étalonnage sur une population (DR = 1 pour la population, ou pour les
   joueurs de PR 5) est une décision future qui remplacera celle-ci.
4. **τ s'exprime en équité normalisée, pas en MWC.** Comme la conversion est linéaire,
   c'est la même chose qu'un τ en MWC égal à k·τ, k étant le prix en MWC d'une unité
   d'équité à ce score et ce videau : le joueur de référence a la même force (un PR) à tous
   les scores, et sa perte se compte en MWC comme celle du joueur.
5. **Domaine.** La difficulté est calculée pour les décisions dont la perte est comptée et
   convertie (même prédicat que la perte de MWC) ; ailleurs elle est absente, jamais zéro.
6. **Erreur évitable.** Une décision est une *erreur évitable* quand sa perte atteint le
   seuil d'erreur de la bibliothèque (ADR-0046) et que d ≤ ℓ/10 : dans le cas binaire,
   le joueur de référence ne la commettrait pas une fois sur dix.
7. **Au niveau du match**, par joueur, sur les décisions qui ont ℓ et d : excès
   Σ(ℓ − d) (en MWC) et ratio Σℓ/Σd (1 = joue comme la référence). Le ratio n'est pas
   donné quand Σd < 0,5 % de MWC : sur un match trop facile il ne mesure que du bruit,
   l'excès reste.
8. **Troncature.** Les candidats absents de la liste (XG en garde au plus quelques-uns,
   gnubg selon ses filtres) ne pèsent rien : d est un minorant, d'autant plus serré que les
   absents sont loin. Aucune correction n'est inventée pour eux.
9. **Calcul à la lecture, pas de colonne.** d dépend du score et du videau du coup et de τ ;
   il se calcule à la lecture des pertes d'un match à partir du blob d'analyse décodé, ce
   qui évite un changement de schéma et laisse τ révisable sans migration.

## Conséquences

- Les trois modes (interface, CLI, serveur) lisent la difficulté par
  `Stats().MatchDecisionLosses`, sur les deux backends.
- Changer τ, le seuil 1/10 ou le plancher du ratio change tous les chiffres affichés : c'est
  une nouvelle ADR, pas un réglage.
- d ne dit rien d'un coup forcé (absent de la perte) et peu d'une position dont la liste de
  candidats est courte.
