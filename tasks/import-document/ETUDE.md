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
3. **L'analyse n'écrit rien en base.** Elle produit un relevé ; seul l'import du relevé écrit, par
   `SaveIndividualPosition`.
4. **Trois états** (ADR-0108 règle 5) : sûr, à vérifier, incertain. Une position n'est sûre que si
   un recoupement indépendant du lecteur la confirme. Dans le doute, à vérifier.
5. **Le document vient de l'utilisateur** : un fichier local ou une URL qu'il donne. Une URL est
   une page, sans exploration du site. Aucune source n'est nommée ni suggérée.

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
- ADR neuf (0115 si libre) : la lecture d'un document est une reconnaissance absolue vérifiée par
  recoupement, en écart à ADR-0108 règles 2 et 3. Accepté dans son principe par l'utilisateur.

## 7. Tranches

| # | Tranche | Mesure |
|---|---|---|
| 0 | Hors dépôt : décoder la police XG de Deep Cube ; vérité exacte ; rasteriser ces pages et mesurer `capture` contre elle | zéro position fausse parmi les sûres ; taux de sûres |
| 1 | Format du relevé, import en Collection, CLI, doc | contrat de stockage, parité |
| 2 | Lecteur à glyphes (Deep Cube en entier) | pips du texte = pips de la position, toutes pages |
| 3 | Thèmes d'imprimerie : redressement, double page, localisation, comptage par pointe | pips imprimés de Trice et Woolsey |
| 4 | Lecture des chiffres | même corpus |
| 5 | Assistant GUI : profil, planche-contact, reprise | e2e |
| 6 | HTML et URL, EPUB, DOCX, thèmes XG et Galaxy imprimés | par source |

Premier périmètre : Deep Cube, puis Robertie 501 et Woolsey.

## Décisions de l'utilisateur (2026-11-25)

- Recommandations suivies : rendu PDF, premier périmètre, ADR neuf.
- Seules les positions entrent en base, pas les captures ni les images (règle 1).
