# ADR-0049 — Une transcription se corrige là où le curseur est, et le transcript montre ce qui est tapé

- **Statut** : accepté — décidé le 2026-09-24, après une séance d'usage du panneau livré.
- **Détaille** : ADR-0045 (le brouillon et ses gestes) et ADR-0048 (la surface du panneau).
  Rien ici ne change ce qui est ENREGISTRÉ : les Actions, le Replay, les Incohérences et le
  Match produit sont les mêmes.
- **Applique** : ADR-0044 (transcrire n'est pas jouer — rien n'est refusé, tout est marqué),
  ADR-0048 décision 1 (la touche chiffrée commence un jet là où le Cursor est).

## Le problème

Le panneau livré tape un match correctement d'un bout à l'autre **tant qu'on ne se
reprend pas**. Or transcrire, c'est se reprendre : une passe lue pour une prise, une
ouverture attribuée au mauvais camp, un coup oublié trois tours plus haut. Quatre
constats d'usage, tous du même jour, tous la même faille.

1. **Le transcript ne montrait que le document.** L'Entry — l'Action en cours de saisie —
   n'en fait pas partie : une correction laissait donc la cellule afficher l'Action
   enregistrée jusqu'à la validation, et une insertion n'apparaissait nulle part avant
   elle. L'utilisateur lisait une chose en en tapant une autre. C'est exactement l'écart
   que le transcript existe pour fermer, et le panneau le rouvrait à chaque correction.
2. **Les gestes de videau écrivaient toujours en bout de document.** Le curseur posé sur
   une passe, `t` insérait une prise DEVANT elle et laissait la passe debout — deux
   Actions à corriger, une partie qui se terminait quand même. La touche chiffrée, elle,
   corrigeait déjà sur place (ADR-0048 décision 1) : deux règles pour un même geste.
3. **Une ouverture ne se ressaisissait pas.** La cellule sous le curseur n'avait pas
   d'attente propre : le panneau lisait ce que le document attend APRÈS sa dernière
   Action (`next.expects`). Revenir sur l'ouverture d'une partie tapait donc deux dés de
   coup de pions — pas de validation au second dé, une liste de candidats pour un camp
   que l'ouverture n'avait pas encore désigné —, et la seule convention qui décide qui
   commence était hors d'atteinte là où elle s'applique.
4. **Le plateau perdait le curseur sur deux sortes d'Action.** `has_position` dit qu'une
   Action ne produit ni Move ni Position dans le Match enregistré — une ouverture, un
   abandon — et le Replay remplit `before` pour toutes, « for the panel only ». Le panneau
   le lisait comme une absence et retombait sur la position de FIN du document : cliquer
   sur la première cellule d'une partie montrait le damier de la dernière.

Pris ensemble, ces quatre points rendent impossible le geste le plus banal du métier :
revenir sur une passe qui était une prise, finir la partie qu'elle avait close, et garder
ce qui a déjà été tapé de la partie suivante.

## La décision

**Une seule règle**, dont tout le reste découle : *le geste écrit où le curseur est, et ce
qui est tapé se voit à cette place avant d'être écrit.*

1. **Le transcript dessine l'Action en cours de saisie**, en pointillés, dans le créneau
   qu'elle occupera : par-dessus la cellule qu'une correction remplace, entre ses deux
   voisines pour une insertion, au bas de la partie en cours pour une saisie neuve. Les
   dés s'y lisent au fur et à mesure, la notation dès qu'un candidat est choisi, et le
   camp se lit à la colonne. Rien n'est écrit dans le document avant la validation —
   c'est une promesse d'ADR-0045, et elle est maintenant VISIBLE au lieu d'être invisible.
   Le moteur fournit ce qu'il faut pour la dessiner (`transcript.EntryInfo` gagne `Kind`
   et `Notation`) ; le composant ne dérive rien, comme son docstring l'exige.
2. **Les quatre gestes de videau écrivent au rang de l'Entry**, comme la touche chiffrée :
   sur une cellule relue ils REMPLACENT, dans un créneau ouvert par `i`/`a` ils
   remplissent, en bout de document ils ajoutent comme avant. `t` et `p` cessent donc
   d'être réservées à l'offre en attente : elles atteignent aussi la cellule tenue par le
   curseur — et rien de plus, car dès qu'un dé est tapé l'utilisateur saisit un jet et `p`
   retourne au compte de pips du répartiteur global.
3. **Une insertion au milieu du document continue d'insérer** : la validation rouvre un
   créneau vide à la suite. C'est ce qui permet de rattraper toute la fin d'une partie
   sans écraser la suivante, et ce n'est pas un mode : le créneau est dessiné (décision 1)
   et déplacer le curseur y met fin.
4. **La cellule sous le curseur a sa propre attente** (`entry.kind`), distincte de ce que
   le document attend en bout (`next.expects`). Ressaisir une ouverture se comporte donc
   en ouverture : dé du joueur 1, dé du joueur 2, validation au second, le plus fort
   commence — gros dé d'abord, le trait revient au joueur 1 en bas du plateau ; petit dé
   d'abord, au joueur 2 en haut.
5. **`has_position` n'est pas « rien à montrer »** : le plateau suit le curseur sur toute
   Action et lit `before`, la fin du document n'étant la réponse que passé la dernière.

## Ce qui n'est pas décidé

**L'ordre des dés ne nomme le camp que sur l'ouverture.** L'étendre à tout jet saisi a été
posé puis écarté le même jour : la saisie courante devrait alors alterner l'ordre à chaque
tour (« 53 » pour le joueur 1, « 24 » pour le joueur 2), et deux jets tapés gros dé d'abord
donneraient deux coups au même camp. Le trait continue d'être proposé par le Replay, et
`s` reste le geste qui le change sur une Action écrite.

**Le premier coup de pions ne suit pas l'ouverture corrigée.** Le `side` appartient à
l'Action (ADR-0045 §4) : corriger l'ouverture ne réécrit pas le camp du coup qui la suit.
Le transcript le marque, le curseur s'y pose de lui-même — c'est la règle « après un Replay,
le curseur va à la première Incohérence » — et `s` le donne au bon camp, d'une touche.

**Amendé le 2026-09-24, le jour même, par la mesure.** Le paragraphe ci-dessus était faux
sur son fait central : **rien ne le marque**. Le premier coup reste LÉGAL pour l'un comme
pour l'autre depuis le damier de départ, et une ouverture ne porte pas le trait
(`bearsTurn`), si bien que le Replay ne trouve ni coup illégal ni double trait et que le
curseur ne saute nulle part. Le document était donc silencieusement faux — le seul cas que
« marquer, jamais refuser » (ADR-0044) ne couvre pas, puisqu'il n'y a rien à marquer.

**Le premier coup de pions suit donc l'ouverture corrigée**, et c'est la seule entorse à
« le `side` appartient à l'Action ». Elle est étroite, et c'est ce qui la rend acceptable :
ce camp-là n'a jamais été choisi par l'utilisateur. Après une ouverture, « le camp du
gagnant du jet joue un `checker` avec les deux dés de l'ouverture — l'utilisateur ne les
ressaisit pas » (`fonctionnel.md` §1.2) : le panneau propose le camp et reporte le jet,
l'utilisateur ne désigne que le coup. Corriger l'ouverture emporte donc cette Action, faute
de quoi le document dit deux choses à la fois — le joueur 2 commence, le joueur 1 joue le
premier. Rien d'autre ne bouge : la suite de la partie garde ses camps, et un premier coup
déjà donné à l'autre camp par `s` est laissé tel quel — son camp n'est alors plus celui de
l'ancien vainqueur, ce qui est exactement le test (`followOpening`).

## Les conséquences

- `transcript.EntryInfo` grossit de deux champs dérivés ; `entryExpects` est écrit une fois
  et lu deux fois (par `expectsOpening`, qui paie un Replay, et par `Replayer.Replay`, qui a
  déjà l'état en main) pour que la règle ne se dédouble pas.
- `TranscriptView` dessine une sorte de cellule de plus. Son docstring reste vrai : elle ne
  tient aucun magasin et ne dérive rien — la cellule provisoire lui est donnée.
- Un `t` sur une cellule relue détruit ce qu'elle portait. C'est la même main que le chiffre
  qui recommence un jet sur place, et le même recours : `Ctrl+Z`, plus le dessin qui montre
  d'avance ce que la validation écrira.
