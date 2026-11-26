# Import des positions d'un document — étude de conception

Étude du 2026-11-25, sans code de production. Banc hors dépôt : `~/bkgm/bancs/import-document/`
(`JOURNAL.md` tient la reprise). Corpus : `~/bkgm/bibliotheque/livres` (17 PDF, 1 EPUB, 1 DOC).

## Synthèse

- But : lire toutes les positions d'un livre, d'un article ou d'une page web que l'utilisateur
  fournit, et les ranger dans l'ordre du document dans une Collection dédiée.
- Un diagramme de livre est un dessin, pas un plateau filmé : un style par livre. Le lecteur est
  `capture` (thèmes, Go pur, sans modèle appris) étendu, pas le lecteur N1 de la vidéo.
- L'exactitude ne vient pas du lecteur seul mais des recoupements que le livre imprime : pips,
  15 pions par camp, légende, coup de la solution.
- Seules les positions entrent en base. Aucune image, aucune capture, aucune prose de l'auteur.

## 1. Ce que contient le corpus

Sondage de la page 60 de huit livres (`pdftotext`, `pdffonts`, `pdfimages`, rendu à 60 dpi).

| Famille | Livres | Lecture |
|---|---|---|
| Police de plateau (texte) | Deep Cube 1-3 (Norberg), 1 546 pages | Décodage des glyphes ; le texte donne aussi score, longueur du match, pips, analyse XG |
| Diagramme d'imprimerie scanné | Robertie 501, Trice, Woolsey, Magriel, Bagai, Jacoby | Thème par livre ; biais, ombre de reliure, double page |
| Capture de logiciel imprimée | Olsen, Mochizuki, Kageyama | Thème XG ou Galaxy |

Sans couche texte : Robertie 501, Bagai, Jacoby, un Kageyama (Opening Concepts, édition
Z-Library). Couche texte mauvaise : Olsen. Le décodage de la police XG est à prouver (tranche 0).

## 2. Règles

1. **Seules les positions entrent en base** : position, dés, videau, score, trait, et la
   référence dans le document (page, libellé comme « Problem 191 »). Aucune image n'est stockée,
   ni en base ni dans le relevé : le relevé garde la page et le rectangle du diagramme, les
   vignettes de vérification sont redessinées depuis le fichier d'origine.
2. **La prose de l'auteur n'est pas importée** : une Collection exportée ou partagée
   l'emporterait.
3. **L'origine de chaque position est gardée.** La description de la Collection porte le
   titre, l'auteur et l'édition du document (ou son URL) ; chaque position porte sa référence
   (page, libellé) en commentaire, pour rester lisible hors de la Collection. Le support exact
   (description, commentaire, champ dédié) se tranche à la tranche 1.
4. **L'analyse n'écrit rien en base.** Elle produit un relevé ; seul l'import du relevé écrit, par
   `SaveIndividualPosition`.
5. **Trois états** (ADR-0108 règle 5) : sûr, à vérifier, incertain. Une position n'est sûre que si
   un recoupement indépendant du lecteur la confirme. Dans le doute, à vérifier.
6. **Le document vient de l'utilisateur** : un fichier local ou une URL qu'il donne. Une URL est
   une page, sans exploration du site. Aucune source n'est nommée ni suggérée.

## 2 bis. Robustesse : un document jamais rencontré

Exigence de l'utilisateur (2026-11-25) : l'outil doit avoir de bonnes chances de marcher sur un
document qu'il n'a pas vu. Un thème par livre n'y répond pas. Conséquences :

1. **Le lecteur cherche la structure, pas un style.** Ce que tout diagramme partage : deux rangées
   de douze pointes alternées, coupées par une barre ; des pions qui sont des disques de deux
   classes, empilés depuis la base d'une pointe. Couleurs, épaisseurs, hachures, sens et
   numérotation ne sont pas supposés.
2. **Le style s'apprend du document lui-même, sans intervention.** Un livre dessine tous ses
   diagrammes de la même façon : la géométrie, les deux classes de pions et l'emplacement des pips
   s'estiment sur l'ensemble des pages puis s'appliquent à chacune. Rien n'est appris ni gardé
   d'un document à l'autre.
3. **Les recoupements ne dépendent d'aucun style** (§3) : ce sont eux qui disent, sur un livre
   inconnu, si la lecture tient. Un style mal compris donne des positions « à vérifier », jamais
   des positions fausses déclarées sûres.
4. **Repli assisté quand le style échappe** : l'utilisateur clique les coins d'un diagramme et
   désigne un pion de chaque camp ; ce profil vaut pour tout le document. L'outil reste utilisable,
   il coûte seulement plus de gestes.
5. **La robustesse se mesure sur une réserve.** Livres de développement : Deep Cube, Robertie 501,
   Trice, Woolsey, Olsen, Mochizuki, Kageyama (Endgame). Réserve, à ne pas ouvrir avant la mesure :
   Bagai, Jacoby, Magriel, Kageyama (Back Checker), l'EPUB, le livre FT, plus deux pages web et
   deux documents que l'utilisateur choisira le jour de la mesure. Critères fixés avant : aucune
   position fausse parmi les sûres ; taux de sûres ; gestes de l'utilisateur pour 100 positions.
   Une réserve ouverte est consommée : la mesure suivante en demande une neuve.

### Replis quand la lecture seule ne suffit pas (demande de l'utilisateur, 2026-11-26)

Trois leviers, du moins coûteux au plus coûteux pour l'utilisateur :

1. **Retrouver l'information autrement.** Une pile douteuse est d'abord détectée et nommée
   (jointive, chiffrée, plafonnée, pâlie). Sa hauteur se cherche alors par une autre voie : sa
   longueur le long de la pointe ; les pips imprimés et le compte de 15, deux équations
   indépendantes ; le coup cité dans la légende ou la solution ; le diagramme voisin, quand un
   livre de référence enchaîne des variantes à un pion près. Une voie qui a servi à déduire ne
   sert plus à vérifier.
2. **Calibration optionnelle, une fois par document.** Gestes simples sur un seul diagramme :
   les coins ; un pion de chaque camp ; le nombre de pions d'une pile haute (donne le pas
   d'empilement) ; un exemple de pile chiffrée et son chiffre ; le sens de jeu ; où sont les pips
   et le videau. Jamais exigée : proposée quand le rendement de l'analyse automatique est bas.
3. **Vérifier par familles, pas une à une.** Les positions à vérifier sont groupées par cause
   (même doute, même forme). L'utilisateur en vérifie quelques-unes par famille :
   - une correction devient une règle rejouée sur toute la famille (« ce signe est un 6 » relit
     toutes les piles portant ce signe) ;
   - si les k premières sont justes, le reste de la famille peut être accepté en bloc, sous un
     état distinct, « validé par échantillon », visible et réversible. Zéro erreur sur k tirages
     borne le taux d'erreur à environ 3/k : c'est une confiance statistique, pas une preuve, et
     l'état ne se confond jamais avec « sûr » ni « vérifié ».
   À mesurer avant de l'offrir : les erreurs se regroupent-elles vraiment par famille, et une
   famille jugée bonne sur un échantillon l'est-elle en entier.

Les thèmes de `capture` restent la voie des captures de logiciels connus (XG, Galaxy). Le lecteur
de structure est un second lecteur, qui rend le même `capture.Reading`.

## 3. Recoupements

- Pips imprimés à côté du diagramme (Trice, Woolsey, XG).
- 15 pions par camp ; barre et pions sortis cohérents.
- Légende : trait et dés (« Black to play 62 »).
- Solution : le coup cité est légal pour la position et les dés.
- Jamais l'équité ni l'avis du moteur (ADR-0108 règle 4).

## 4. Objets

- **Relevé** : fichier texte (JSON) relisible. En-tête : empreinte du fichier source, profil.
  Par entrée : page, rang, chapitre, libellé, position, dés, videau, score, trait, état, doutes
  (`capture.Doubt`), rectangle du diagramme, vérifié ou non. Il porte l'avancement de la
  vérification : un livre se traite en plusieurs séances.
- **Profil du document**, fixé sur le premier diagramme : sens du plateau (jan à gauche ou à
  droite), couleur du joueur du bas, emplacement des pips et du videau, numérotation imprimée.
- **Collection** au nom du document, dans l'ordre de lecture (page, colonne, haut en bas). Les
  chapitres viennent des signets du PDF. Une position répétée dans le livre a un seul hash : à
  trancher à la tranche 1 (une seule appartenance, première occurrence).

## 5. Parcours

1. Déposer un fichier ou coller une URL ouvre l'assistant.
2. Confirmer le profil sur le premier diagramme.
3. Analyse en fond ; trois compteurs cliquables (sûr, à vérifier, incertain).
4. Planche-contact : vignette d'origine à côté du plateau redessiné dans le même sens ; doutes
   entourés ; une touche valide, une autre ouvre EDIT.
5. Import, ouverture de la Collection.

## 6. Intégration

- Paquet neuf `pkg/blunderdb/document` : extraction (pages, images, texte), découpe des
  diagrammes, appel de `capture`, recoupements, relevé. Méthodes sur `Database` (parité).
- CLI : `document analyze <fichier|url> -o releve.json` ; `document import releve.json
  --collection …`, qui refuse une entrée non sûre et non vérifiée sauf `--force`.
- Serveur et MCP : import du relevé seulement en v1 ; pas d'analyse de PDF sur le daemon.
- EPUB, DOCX, HTML : conteneurs d'images et de texte, Go pur. `.doc` binaire : à convertir avant.
- Rendu des pages PDF (décision de l'utilisateur, 2026-11-25) : pdf.js dans la webview pour le
  GUI ; pour le CLI, un rasteriseur externe détecté (ADR-0004) ou un dossier d'images. pdf.js sous
  WebKitGTK reste à vérifier.
- Pas d'OCR généraliste : la couche texte quand elle existe ; sinon lecture de chiffres seulement
  (pips, videau, dés de la légende).
- ADR-0117 : la lecture d'un document est une reconnaissance absolue vérifiée par
  recoupement, en écart à ADR-0108 règles 2 et 3. Accepté dans son principe par l'utilisateur.

## 7. Tranches

| # | Tranche | Mesure |
|---|---|---|
| 0 | Hors dépôt : décoder la police XG de Deep Cube ; vérité exacte ; rasteriser ces pages et mesurer `capture` contre elle | zéro position fausse parmi les sûres ; taux de sûres |
| 1 | Format du relevé, import en Collection, CLI, doc | contrat de stockage, parité |
| 2 | Lecteur à glyphes (Deep Cube en entier) | pips du texte = pips de la position, toutes pages |
| 3 | Lecteur de structure (§2 bis) : redressement, double page, localisation, profil estimé sur le document, comptage par pointe | pips imprimés de Trice et Woolsey ; puis la réserve, en un seul passage |
| 3b | Repli assisté : coins et un pion par camp, profil pour tout le document | gestes pour 100 positions sur un livre où 3 échoue |
| 4 | Lecture des chiffres | même corpus |
| 5 | Assistant GUI : profil, planche-contact, reprise | e2e |
| 6 | HTML et URL, EPUB, DOCX, thèmes XG et Galaxy imprimés | par source |

Premier périmètre : Deep Cube, puis Robertie 501 et Woolsey.

## Résultats

### Tranche 0 (2026-11-26) — banc `~/bkgm/bancs/import-document/`, `JOURNAL.md`

Seuls Deep Cube 1-3 ont été ouverts. La réserve est intacte.

- **Police décodée** : 1 518 pages à diagramme, 1 518 sûres (15 pions par camp, pips calculés =
  pips imprimés des deux côtés). Contrôle à l'œil : 33 pages au hasard et 14 cas rares, toutes
  concordantes. 1 491 positions distinctes. `pdftotext -layout` n'est pas fiable (lignes tracées
  par morceaux) : il faut lire le flux de contenu avec la position de chaque glyphe. Non prouvé,
  faute d'occurrence : pile de plus de 9, barre de plus de 3, videau chez Player 2, dés.
- **`capture` tel quel** (210 pages, 150 et 300 dpi) : 0 exacte, 0 fausse sans doute. Mais il ne
  refuse pas : il prend le tableau d'analyse gris pour un plateau (fit 0,66-0,72, `MinFit` 0,6)
  et rend 13 à 20 doutes. Sur le diagramme découpé : refus. À corriger avant de le brancher sur
  des pages entières.
- **Prototype de lecteur de structure** (`scripts/structure.py`, 150 dpi, un profil par volume) :
  1 421 sûres (93,6 %), toutes exactes ; 97 à vérifier, dont 3 fausses (deux piles chiffrées dans
  le même camp) ; 0 refus. 300 dpi : identique. 100 dpi : presque tout faux mais « à vérifier »,
  aucune sûre fausse — résolution minimale de rendu : 150 dpi.
- **Encore propre à Deep Cube dans le prototype** : pips lus dans le texte, cadre de traits
  pleins, cases, rayon et pas fixés à la main, seuils d'un rendu vectoriel net, pas de lecture des
  chiffres, des pions sortis ni du videau. Jamais essayé hors de Deep Cube.

Suites : le lecteur à glyphes est la voie de Deep Cube (tranche 2) ; le lecteur de structure se
généralise d'abord sur les livres de développement scannés (tranche 3), la réserve restant
fermée.

### Tranches 1 et 2 (2026-11-26) — fusionnées

- Tranche 1 : paquet `pkg/blunderdb/document`, Relevé (position en XGID), import en Collection
  dans l'ordre du document, `document check|import`, route `/v1/documents.importReleve`,
  ADR-0117. Origine sans changement de schéma : description de la Collection, et un commentaire
  par position (« Titre — p. 3, Problème 1 »), réécrit et jamais doublé.
- Tranche 2 : `document analyze` pour les PDF en police de plateau XG ; lecteur PDF minimal en
  bibliothèque standard (`document/pdf`, borné, fuzzé), aucune dépendance. `TestMeasureDeepCube`
  relancé sur `main` : 1 518 diagrammes, 1 518 sûrs, 1 518 identiques à la vérité, environ 1 s par
  volume. Deep Cube n'a ni signets ni libellés : chapitre et libellé y restent vides.

### Lecteur de structure sur scans (2026-11-26) — banc, `JOURNAL.md` D0-D6

Un seul code, aucune valeur par livre, sur Trice, Woolsey et Robertie 501. Échantillon de mesure
tiré avant le développement, passé une fois, vérifié à l'œil diagramme par diagramme (consommé).

| Livre | Présents | Trouvés | Sûres | Sûres fausses | À vérifier exactes / fausses |
|---|---|---|---|---|---|
| Trice | 43 | 43 | 27 | 0 | 10 / 5 (+ 1 incertaine fausse) |
| Woolsey | 41 | 40 | 16 | 0 | 18 / 6 |
| Robertie (sans pips) | 36 | 33 | — | — | 26 / 7 |
| Deep Cube (non-régression) | 1 518 | 1 518 | 1 420 | 0 | 0 / 97 (+ 1 incertaine) |

- Sûreté : aucune sûre fausse. C'est la concordance des pips qui protège, pas la lecture.
- Rendement insuffisant : environ la moitié de sûres sur un scan propre. Causes des 18 lectures
  fausses : pile noire jointive comptée trop courte (7), pile chiffrée (4), pion pâli ou collé au
  cadre (3), pile de plus de 6 plafonnée (2), piles haut et bas qui se touchent (2), barre (1).
- Sans pips imprimés (Robertie), rien n'est sûr : 79 % d'exactes. Les lectures à 15 + 15 pions
  sans doute sont toutes exactes (84 sur 84, trois livres) : confiance intermédiaire plausible,
  à confirmer sur une mesure neuve avant d'en faire un état.
- Localisation : 116 trouvés sur 120, aucun faux positif, ordre de lecture correct.
- Chiffres par gabarits appris sur la numérotation : échec sur Trice (hachures lues comme « 1 »).
- Non lus : pions sortis, videau, chiffre d'une pile. Le sens de jeu sans pips est une
  supposition à faire confirmer. Un cadre sans traits continus serait rejeté en bloc : le repli
  assisté (§2 bis.4) reste nécessaire.

Suite : le compte des piles (jointives, chiffrées, hautes) avant tout portage en Go ; nouvel
échantillon de mesure ; la réserve reste fermée.

## Décisions de l'utilisateur (2026-11-25)

- Recommandations suivies : rendu PDF, premier périmètre, ADR neuf.
- Seules les positions entrent en base, pas les captures ni les images (règle 1).
- Le nom ou l'origine du document reste attaché à chaque position, en commentaire ou dans la
  description de la Collection (règle 3).
- L'ordre d'import est l'ordre d'apparition dans le document (§4).
- Robustesse sur un document jamais rencontré (§2 bis).
- 2026-11-26 : replis à prévoir si la lecture ne suffit pas — calibration optionnelle, autre voie
  pour les piles douteuses, vérification par familles (§2 bis, « Replis »).
