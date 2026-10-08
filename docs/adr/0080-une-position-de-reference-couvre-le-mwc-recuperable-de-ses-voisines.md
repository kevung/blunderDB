# ADR-0080 — Une position de référence couvre le MWC récupérable de ses voisines

Statut : acceptée.
Voir aussi : ADR-0077 (plan d'étude), ADR-0076 (difficulté), ADR-0043 (distance `like`),
ADR-0046 (seuils d'erreur), ADR-0007 (rien n'est écrit chez le receveur).

## Contexte

Le plan d'étude (ADR-0077) dit quelles familles d'erreurs travailler ; il ne dit pas quelles
positions mettre dans une collection pour les travailler. Prendre les positions de plus grand
excès donne dix variantes d'un même problème, ou une position singulière qui n'apprend rien sur
les autres. Une bonne position de référence est au centre de beaucoup d'erreurs évitables, porte
une leçon nette et n'est pas déjà étudiée. Les critères et leurs seuils sont fixés ici **avant**
d'avoir lu ce que l'outil propose sur une base réelle.

Le regroupement par similarité a été différé par l'ADR-0077 pour son coût : 166 ns par paire,
8 s pour 10 000 erreurs en paires complètes.

## Décision

1. **Candidates.** Les erreurs chiffrées et thématisées du filtre, comme au plan d'étude, réunies
   par position (l'excès d'une position est la somme des excès positifs ℓ − d de ses erreurs :
   une erreur que le joueur de référence ferait autant n'a rien à récupérer). Un **groupe** est
   une famille (plan de jeu, nature, thème) ; pour le videau il est précisé par le **score**
   (écarts du joueur au trait et de l'adversaire) : une référence de videau vaut à un score, pas
   à un autre.
2. **Voisines.** Deux candidates d'un groupe sont voisines à au plus **ρ = 12 pions-pas** de
   distance `like` (ADR-0043 : un coup de dés en vaut 8 à 16). Une erreur jouée plusieurs fois
   dans une même position est une seule candidate.
3. **Gain d'une candidate.** G(p) = c(p) · Σ excès des candidates du voisinage de p (p comprise)
   que la sélection ne couvre pas encore, en fraction de MWC. La somme porte à la fois la
   **représentativité** (combien d'erreurs elle résume), la **fréquence** (une famille fréquente
   donne des voisinages peuplés) et le **coût évitable** (M3) : c'est le MWC qu'on regagnerait si
   la leçon de p servait à toutes ses voisines.
4. **Leçon nette.** c(p) = ½ par défaut relevé, au plus deux : **serrée** quand le second
   meilleur choix coûte moins de la moitié du seuil Erreur de la bibliothèque (le joueur se
   tromperait encore en apprenant « l'autre bon coup ») ; **instable** quand une autre profondeur
   d'analyse du videau ou un rollout donne un autre verdict sur la décision que le joueur avait
   à prendre (doubler, ou prendre). Un rollout qui confirme ne relève rien : il départage à gain
   égal et la raison le dit.
5. **Diversité.** Sélection gloutonne par gain marginal (évaluation paresseuse) : chaque
   position retenue couvre ses voisines, qui ne rapportent plus rien aux suivantes ; toute
   candidate à au plus ρ d'une position retenue, de même nature et pour le videau de même score,
   est écartée même hors de son groupe. Une famille très peuplée peut donner plusieurs
   références, jamais deux fois la même structure.
6. **Non traitée.** Une position déjà traitée — commentaire, carte Anki, collection, marque
   « étudiée » : le prédicat de la file d'étude, mot pour mot — n'est jamais proposée ; elle
   compte comme déjà retenue : ses voisines sont couvertes, ses quasi-doublons écartés.
7. **Raison affichée.** Chaque proposition rend ses composantes, pas une phrase : famille et
   score, nombre d'erreurs et de matchs couverts, taille de la famille rapportée aux décisions
   du filtre, MWC récupérable couvert, écart avec le second choix, drapeaux serrée / instable /
   confirmée par rollout. Chaque client écrit la phrase dans sa langue.
8. **Coût borné : un index de voisinage en grille.** La distance `like` est la somme des écarts
   absolus des sommes préfixes des deux camps ; elle majore donc la somme des écarts de leurs
   totaux sur quatre segments (moitié intérieure, points 0-12, et moitié extérieure, 13 à la
   barre, de chaque camp). Les candidates d'un groupe sont rangées dans une grille de ces quatre
   totaux, à cases larges de ρ : une paire n'est mesurée que si ses deux cases sont adjacentes et
   que la borne tient, et la distance est prise sur des sommes préfixes calculées une fois par
   position. Aucun stockage, aucune mise à jour : la grille est bâtie à chaque proposition.
9. **Portée et taille.** Le filtre des statistiques (joueur, tournois, dates, nature), restreint
   s'il le faut à des matchs ; 10, 20 ou 50 positions (20 par défaut, 50 au plus, comme la file
   d'étude). La liste se coche ; ce qui est coché devient une collection figée, un paquet Anki ou
   un quiz. La même fonction de stockage sert la GUI, la CLI (`collection suggest`) et le serveur
   (`/v1/collections.suggest`). Rien n'est écrit tant que l'utilisateur ne choisit pas.

## Conséquences

- Mesuré (`BenchmarkSuggestReferences`, un cœur, parties aléatoires sans doublon, pire cas
  dense : chaque partie finit dans le jan) : 10 000 erreurs en 20 groupes 45 ms, en un seul
  groupe 0,3 s ; 50 000 erreurs en 20 groupes 0,5 s, en un seul groupe 6 s. Une base réelle
  répartit ses erreurs sur des dizaines de familles : la proposition se fait à la demande, en
  moins d'une seconde, là où les paires complètes demandaient 8 s pour 10 000 erreurs.
- Une collection **vivante** n'est pas proposée : elle est une requête de la grammaire de
  recherche, réévaluée à chaque ouverture ; la sélection n'en est pas une (elle classe tout un
  filtre par couverture gloutonne), et un jeton qui la rejouerait ferait de chaque ouverture une
  passe de similarité.
- Rejeté : un score pondéré de sept critères — sept poids arbitraires et une raison illisible.
  Ici une seule quantité, en MWC, et deux facteurs de ½ nommés.
- Rejeté : les paires complètes sur toute la base, ou un index métrique général (arbre VP) — la
  bande de pips suffit au rayon fixé et ne demande ni stockage ni mise à jour.
- ρ, le seuil « serrée » et les facteurs ne changent qu'avec une nouvelle ADR.
