# ADR-0048 — Le panneau de transcription est ordonné par la fréquence, non par le voisinage

- **Statut** : accepté — décidé le 2026-09-10, après un entretien de conception le même jour
  (huit décisions, simulation persona du panneau livré).
- **Détaille** : ADR-0045 (le brouillon et ses gestes) — **sur sa surface seulement**. Rien ici
  ne touche au document, au Replay, ni à ce qui est enregistré : les Actions, les Incohérences
  et le Match produit sont inchangés.
- **Renverse** : l'arbitrage du clavier consigné dans `tasks/transcription/ux.md` §3
  (« Arbitrage mesuré le 2026-09-07 »).
- **Applique** : ADR-0008 (une seule échelle de type), ADR-0018 règle 4 (un appelant masque
  les colonnes qui ne portent aucune information chez lui), ADR-0034 (l'aide en ligne est
  engendrée depuis les sources Sphinx).
- **Voir aussi** : ADR-0044 (transcrire n'est pas jouer).

## Le problème

Le panneau livré est correct action par action et impraticable de bout en bout. Un persona
qui transcrit une finale en 7 points dans la configuration par défaut — dock bas, 250 px de
haut (`panelLayoutStore.js`, `DEFAULT_PANEL_HEIGHT`) — ne voit **aucune ligne de candidat sans
défiler**. Le calcul, fait sur les tokens (`--space-2` = 8 px), la taille de case mesurée du
triangle (28 px) et les styles de `CandidateMovesTable` (lignes de 18,4 px) : **354 px** de
contrôles avant la première ligne de la liste, pour **168 px** de boîte disponible.

Quatre causes, toutes documentées dans le code qui les porte, aucune fautive isolément.

1. **Le voisinage ne se compose pas.** Trois blocs ont été placés « SOUS le triangle », chacun
   par rapport à son voisin du dessus, chacun avec une justification juste : le triangle sous
   les cases du jet (une seconde entrée pour les dés), la rangée `[D][T][P][R]` sous le
   triangle (les gestes de videau sont rares au regard des jets), la bascule et le champ de
   notation sous la rangée (ils servent une fois par match). Personne n'a fait la somme :
   290 px s'intercalent entre les dés et la liste qu'ils appellent.

2. **La chaîne de hauteur est cassée.** `.draft` et `.draft-body` ne portent ni `flex: 1` ni
   `min-height: 0`. Les deux `overflow: auto` déjà écrits — la liste des candidats, le
   `.scroller` de `TranscriptView` — sont donc **inertes**, faute d'une hauteur qui les borne,
   et le seul conteneur qui défile est `.tab-content`, c'est-à-dire le panneau entier. D'où
   un effet de bord que personne n'a voulu : le `scrollIntoView({block:'nearest'})` de
   `TranscriptView`, écrit pour cadrer la cellule du Cursor, ne trouve pas de conteneur borné,
   remonte jusqu'à `.tab-content` et **fait sauter tout le panneau à chaque Action validée** —
   au moment précis où l'utilisateur va taper les dés suivants.

3. **Le compteur qui a servi à décider était aveugle à ce qu'il coûtait.** `ux.md` §1 déclare
   l'opérateur M « identique entre designs, donc hors comparaison ». L'arbitrage du 2026-09-07
   a donné à la touche chiffrée deux sens séparés par un état invisible — depuis « jet
   corrigeable » elle recommence le jet, depuis « candidat choisi » elle valide — et a
   compensé par deux phrases à l'écran qu'il faut lire. Ce que cet arbitrage coûte est
   précisément un M par tour, la seule grandeur que le modèle avait exclue.

4. **`ux.md` §5 avait déjà sorti l'état du panneau, et n'a jamais été câblé.** Il place la
   longueur, le score, Crawford, le camp au trait, le videau et l'état d'enregistrement dans
   la **barre de match** (`MatchInfoBar`, déjà au-dessus du plateau), et l'Action attendue en
   un mot dans la **barre d'état**. Ni l'une ni l'autre ne connaît la transcription ; le
   panneau les a réimplémentées en sept badges et une phrase, et c'est là que sont passés
   66 des 354 px.

## La décision

Deux règles engendrent les huit décisions, et répondent d'avance à la neuvième.

> **R1 — Ce qui sert à chaque tour est visible sans défiler ; ce qui sert une fois est à un
> geste.** Aucun bloc n'est placé par rapport à son voisin ; chacun l'est par rapport au
> contrat ci-dessous.
>
> **R2 — Un message habite là où est ce dont il parle.** Un seul domicile par message.

Le contrat, mesurable et tenu par une spec Playwright qui compte des pixels et non des
présences dans le DOM :

> Dans tout régime de dock où le panneau est utilisable, **les deux cases du jet et les cinq
> premières lignes de candidats sont visibles sans défiler**, la phrase du camp au trait
> l'étant en permanence dans la barre d'état. **Rien ne s'intercale verticalement entre les
> cases du jet et la première ligne de candidats.**

`N = 5` et non 3 : à 3, `j` sur le rang 4 fait défiler, donc le geste le plus fréquent après
le meilleur coup rend le contrat faux dès qu'on s'en sert. Rangs 1 à 5 ≈ 88 % des tours.
164 px.

1. **La touche chiffrée commence un jet là où le Cursor est.** En bout de document il n'y a
   rien sous le Cursor : le chiffre valide le candidat sélectionné et ouvre le jet suivant.
   Sur une Action existante il y a quelque chose : le chiffre en **recommence le jet, sur
   place**. `Entrée` reste un synonyme de la validation (elle sert le dernier coup d'une
   partie, qui n'a pas de tour suivant pour la porter) ; `Retour arrière` garde le sien, il
   efface le jet en cours. Le discriminant est `entry.replacing`, que le moteur expose et que
   le panneau lit déjà.

   C'est **un seul sens**, et c'est là que l'arbitrage du 2026-09-07 se trompait : il
   distinguait aussi deux comportements, mais sur un **historique invisible** — « avez-vous
   touché la liste ? ». Ici la différence est un **objet dessiné** : la cellule encadrée dans
   le Transcript, où l'utilisateur s'est rendu délibérément une frappe plus tôt. Un état que
   l'on voit n'est pas un mode.

   Le meilleur coup joué revient à **2 K** — le docstring de `transcriptionKeys.js` redevient
   vrai — l'état « jet corrigeable » disparaît avec ses deux phrases, et les budgets de
   correction d'`ux.md` §4.3 sont tous **inchangés** : dé mal lu vu aussitôt 2 K, vu k tours
   plus tard `h`×k `4` `1` soit 3 K. Sur 250 Actions dont ~60 % de meilleurs coups :
   **−150 K, soit −42 s par match**, sans contrepartie sur la relecture.

   La formulation « le chiffre valide toujours, partout » a été essayée et écartée en séance :
   `GestureValidate` appelle `validate`, qui — contrairement à `commitCorrection` — n'est pas
   gardé par `entryDiffers`, réécrit l'Action à l'identique et rend le Cursor à `doc.Return`.
   Un seul chiffre égaré pendant une relecture aurait donc **mis fin à la relecture** et renvoyé
   le Cursor en bout de document, quand la règle retenue le garde local. Le filet `Ctrl+Z`
   n'aurait pas rattrapé la faute le lendemain : la pile vit dans le `transcript.Editor` de la
   session et un brouillon réouvert n'en a pas (ADR-0045 règle 1).

2. **L'état du brouillon quitte le panneau**, comme `ux.md` §5 le disait : six badges vers
   `MatchInfoBar`, la phrase du jet vers la barre d'état. **Exception** : l'état
   d'enregistrement reste dans la barre du brouillon, collé à son bouton — c'est le seul badge
   qui répond à une question qu'on se pose en regardant ce bouton.

3. **La barre de correction disparaît.** Ses six boutons agissent sur « l'Action au Cursor »,
   c'est-à-dire sur une cible que l'utilisateur ne voit pas nécessairement ; le clic droit sur
   une cellule du Transcript offre déjà les quatre mêmes gestes en nommant sa cible, et
   `Ctrl+Z` / `Ctrl+Maj+Z` sont universels. Un bouton dont la cible est invisible est **moins**
   découvrable que le menu qui la désigne.

4. **La liste des candidats a deux projections nommées**, décidées dans `analysisRows.js` et
   nulle part ailleurs : `judge` (les neuf colonnes plus la provenance — panneaux Eval et
   Analyse) et `identify` (`move` élastique, `equity`, `error` — panneau Transcription).
   Transcrire, c'est *identifier* le coup qu'on a vu jouer, non le *juger* : les six colonnes
   de probabilités ne sont pas lues, elles invitent à l'être, et elles coûtent 290 px de
   largeur — exactement ce qui empêche la disposition à trois colonnes de tenir à 1024 px.
   `error` reste, parce qu'elle dit au moment de la validation que le coup consigné était une
   bourde, ce qui est la raison d'être du produit. Deux projections **nommées** plutôt qu'une
   prop `columns` libre : un troisième appelant devra justifier une troisième projection au
   lieu d'inventer sa liste.

5. **La chaîne de hauteur est réparée** (`flex: 1; min-height: 0` sur `.draft` et
   `.draft-body`, les deux colonnes bornées), ce qui fait trois choses d'un coup : le panneau
   cesse de défiler en bloc, les deux `overflow: auto` déjà écrits redeviennent effectifs, et
   le `scrollIntoView` retombe dans le Transcript où il était censé agir. La **palette** — le
   triangle des 21 jets, la rangée `[D][T][P][R]`, le secours de saisie à la main — est à
   gauche quand la boîte est large (dock bas : trois colonnes `palette | candidats |
   Transcript`, la palette sous les dés, donc la proximité mesurée du triangle est conservée)
   et **sous** la liste quand elle est étroite. Un seul point de rupture, et il n'est pas
   porteur : le contrat tient des deux côtés. L'onglet demande une **hauteur minimale de dock
   de 280 px** : un panneau qui a un contrat mesurable a le droit de dire de combien il a
   besoin.

6. **Le texte `.mat` devient une modale**, ouverte depuis la barre du brouillon à côté
   d'`Exporter .mat`. Ce n'est pas une question d'encombrement mais d'alignement : le `.mat`
   est de l'ASCII aligné en colonnes — c'est sa seule raison d'être regardé — et sa ligne la
   plus longue (`testdata/test.mat`) fait 62 caractères, soit ~409 px en monospace 11 px, là
   où la colonne Transcript en offre 320. Un `.mat` affiché à 320 px n'est pas une version
   dégradée du `.mat`, c'est autre chose. `TranscriptView` y perd `matText`, `onMatToggle`
   et `onCopy`, et redevient ce que son docstring promet : présentationnel, sans presse-papiers
   ni volet propre au brouillon, donc montable un jour sur un match stocké.

7. **La palette contient ce qui sert, et déplie ce qui sauve.** Ligne 1 : les cases du jet et
   la rangée `[D][T][P][R]` côte à côte — ce sont les cinq réponses possibles à une seule
   question, « qu'a fait le camp au trait ? » — plus un bouton qui déplie la bascule
   « déplacement libre » et le champ de notation **à la place du triangle**. Le secours et la
   cible ordinaire ne servent jamais en même temps ; aujourd'hui `handEntryOpen` est vrai à
   chaque tour de pions, soit 250 fois pour un usage attendu d'une fois.

8. **Les messages ont chacun un domicile** (R2), et le panneau n'en porte plus en prose.
   *Ce que le document attend* (jet de X, réponse de Y, relance, danse, match fini, correction
   en place, coup à revoir) → **barre d'état**, un emplacement, une phrase, toujours.
   *Comment faire* (six lignes d'instruction permanentes) → **l'aide en ligne et la visite
   guidée** : une instruction permanente est l'aveu qu'un geste ne se devine pas, et sa place
   n'est pas 13 px sous les dés lus 250 fois par quelqu'un qui les connaît depuis le troisième
   tour. *Où en est un geste* → collé à son objet : le triangle dit déjà l'ambiguïté d'un jet
   en n'allumant que les cases encore possibles, le plateau dit déjà les pas joués, et le
   filtre par point devient une **puce sur l'en-tête de la liste** — ce qui donne enfin le
   moyen de le lever. *Alerte* (Incohérence) → un **bandeau en tête de la colonne Transcript**,
   là où est la cellule fautive.

## Options écartées

- **Garder l'arbitrage du 2026-09-07** (la touche chiffrée recommence le jet partout, `Entrée`
  valide), au motif que la relecture est l'usage qui gouverne. Écarté : la relecture gouverne
  les *gestes de correction*, pas la boucle de frappe, et le coût réel de l'arbitrage — un mode
  caché vérifié 250 fois — est ce que le compteur retenu ne savait pas voir. La décision 1 lui
  donne raison là où il avait raison, sur l'Action que l'on relit, et lui retire le bout du
  document, où il n'y a rien à corriger.
- **Le chiffre valide partout, sans regarder le Cursor.** Écarté en séance après lecture du
  chemin Go : voir la décision 1. Une règle plus courte à énoncer, mais qui rend non locale la
  faute de frappe commise précisément là où l'on tâtonne.
- **Replier la palette derrière une bascule** pour garder les neuf colonnes de la liste.
  Écarté : une cible souris cachée par défaut est une cible morte, et le triangle a été
  mesuré et retenu (`ux.md` §4.1) — ce n'est pas le moment de le tuer par la mise en page.
- **Une page dédiée plein écran pour la transcription.** Écartée pour l'instant : le contrat
  tient dans le dock une fois les quatre causes traitées, et un douzième mode d'affichage se
  paie en documentation, en visite guidée et en tests. À rouvrir si le contrat cède.
- **Une ligne de message unique dans le panneau**, montrant au plus un message par priorité.
  Écartée : un ordre de priorité veut dire qu'un jour « filtre actif » masquera
  « incohérence ».
- **Une prop `columns` libre sur `CandidateMovesTable`.** Écartée au profit des deux
  projections nommées : une liste libre est une porte ouverte à une troisième, une quatrième,
  et à la dérive silencieuse entre panneaux.

## Conséquences

- `MatchInfoBar` acquiert un troisième contexte (match, provenance d'une position, brouillon
  en cours de frappe). C'est le prix de la règle R2, et il est explicite.
- Retirer les six lignes d'instruction est une **régression de découvrabilité si elle part
  seule**. Elle doit atterrir dans la même branche que ses entrées `raccourcis.rst` /
  `manuel.rst`, le `make help` qui régénère les bundles (ADR-0034) et les huit `.po` : on ne
  supprime pas l'endroit où un geste s'apprenait sans en ouvrir un autre.
- `tasks/transcription/ux.md` est amendé, pas réécrit : §3 note son arbitrage renversé **avec
  sa raison** (M non constant entre les deux designs), §4.1 repasse le meilleur coup à 2 K,
  §5 dit « modale » là où il disait « volet dépliable », et §2 remplace la maquette à deux
  colonnes par celle à trois. La méthode de mesure de §1 reçoit une note : M n'est hors
  comparaison que lorsque les designs comparés n'ajoutent pas de mode.
- Le contrat `N = 5` est **testé en pixels**, pas en présence dans le DOM, dans les deux
  régimes de dock. Une spec qui vérifie que la liste est montée ne verrait rien de tout ce que
  cet enregistrement corrige.
- Les chiffres de cet enregistrement sont **calculés** depuis les tokens et les styles des
  composants, non relevés dans l'application en marche. La spec Playwright ci-dessus est aussi
  ce qui les confirmera ou les démentira ; le cas échéant, c'est elle qui a raison.
