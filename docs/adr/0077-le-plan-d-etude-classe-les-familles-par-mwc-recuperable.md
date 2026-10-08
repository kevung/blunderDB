# ADR-0077 — Le plan d'étude classe les familles d'erreurs par MWC récupérable

Statut : acceptée.
Voir aussi : ADR-0076 (difficulté d'une décision), ADR-0075 (perte MWC à sept points),
ADR-0046 (seuils d'erreur), ADR-0019 (échelle d'équité), `docs/recherche/P20-metriques-force-backgammon.md`.

## Contexte

Les erreurs récurrentes (`stats recurring`) groupent les erreurs d'un filtre par plan de jeu et
thème et les classent par coût cumulé. Ce classement répond à « où ai-je perdu le plus ? », pas
à « que dois-je travailler maintenant ? » : il ne distingue pas une famille d'erreurs qu'un
joueur solide aurait évitées d'une famille de positions simplement difficiles, et il pousse au
même rang une famille de trois erreurs (du bruit) et une de trente. Le plan d'étude répond à la
seconde question. Ses définitions et seuils sont fixés ici **avant** d'avoir regardé ce qu'il
donne sur une base réelle.

## Décision

1. **Famille.** Une famille est un triplet (plan de jeu, nature de la décision, thème) des
   erreurs récurrentes, avec le même classement : thème de pions d'`engine.ExplainChecker`,
   direction de l'erreur de videau. Ses membres sont les erreurs du filtre (décisions comptées
   coûtant au moins le seuil Erreur de la bibliothèque). Une erreur sans thème n'entre dans
   aucune famille : elle n'indique rien à étudier.
2. **Membre chiffré.** Un membre compte s'il porte une perte de MWC ℓ et une difficulté d
   (ADR-0076), convertis par la même fonction que le panneau Match. Une décision en partie
   libre (pas de MWC) ou sans coûts d'options exploitables est comptée à part (« non
   chiffrées ») et n'entre dans aucune priorité.
3. **Priorité : MWC récupérable.** R = Σ (ℓᵢ − dᵢ) sur les membres chiffrés, en fraction de
   MWC : fréquence × perte moyenne en excès de la difficulté. C'est ce qu'on regagnerait en
   jouant ces positions comme le joueur de référence de l'ADR-0076. La somme est signée, sans
   écrêtage, pour rester sans biais.
4. **Intervalle.** Le nombre de membres est traité comme un comptage de Poisson et leurs excès
   comme indépendants : Var(R) ≈ Σ (ℓᵢ − dᵢ)², intervalle à 95 % R ± 1,96·√Σ(ℓᵢ − dᵢ)².
   L'estimateur est analytique et déterministe : les deux moteurs de stockage donnent le même
   nombre, sans tirage.
5. **Preuve minimale.** Une famille entre au plan si elle a **au moins 5 membres chiffrés et
   une borne basse strictement positive**. Les autres sont rendues à part (« à confirmer »),
   classées par R, jamais mêlées au plan.
6. **Rang.** Le plan est classé par borne basse décroissante (puis R, nombre de membres, plan,
   nature, thème) : à récupérable égal, la famille la mieux établie passe devant. Dans une
   famille, les positions sont classées par excès décroissant.
7. **Familles par similarité : différées.** Un regroupement par distance `like` demande les
   distances de paire entre erreurs. Mesuré : 166 ns par paire (`engine.SimilarityDistance`,
   28 cœurs, un seul utilisé) ; 2 000 erreurs coûtent 0,3 s, 10 000 erreurs 8 s, 50 000 erreurs
   plus de 3 min — hors de portée d'une carte de tableau de bord sur une grosse base sans index
   de voisinage. Elles reviennent avec les positions de référence proposées, qui ont besoin du
   même index.
8. **Usage.** Une famille du plan alimente la file d'étude (motif `plan`, positions classées
   par excès), un quiz (tirage parmi les positions des trois premières familles, ou de celle que
   désigne son rang) et un deck Anki ; la même fonction de stockage sert la GUI,
   la CLI (`stats plan`) et le serveur (`/v1/stats.studyPlan`).

## Conséquences

- Une famille aux erreurs « difficiles » (d proche de ℓ) descend : elle coûte, mais l'étude y
  rapporte peu. Une famille d'erreurs évitables monte.
- Un petit échantillon ne produit pas de plan plutôt qu'un plan trompeur : un nouveau joueur
  voit d'abord des familles « à confirmer ».
- Les seuils (5 membres, 95 %) ne changent qu'avec une nouvelle ADR.
