# Mesure de l'import de document sur les livres réels — 2026-11-26

Issue #996. Lecteur mesuré : `main` à 03755df32 (Relevé v2, lecture du contexte par le lecteur de
structure, défaut de contexte appris #991). Comparaison : le même lecteur juste avant #991
(2bde8b2a5). Aucune modification du lecteur dans cette branche.

## Protocole

- Banc hors dépôt : `~/bkgm/bancs/import-document/` (`LISEZMOI.md`). Les livres y sont des liens vers
  `tmp/livres/` ; aucune page ni extrait n'entre dans le dépôt, les exemples ne citent que des pages.
- Tirage fait une fois (`tirage.json`, lecture seule) : par livre, chaque page passe en réserve
  avec la probabilité 0,3 (graine `sha256("blunderdb-import-document-996" + fichier)`). Les défauts
  et les issues ci-dessous ne viennent que des pages de réglage ; la réserve a été comptée une
  seule fois, en dernier, après l'ouverture des issues.
- Chaque PDF est lu par `document.AnalyzeFile` (même chemin que `blunderdb document analyze`) ; le
  programme `outil/` compte les relevés (`./banc mesure reglage|reserve <étiquette>`).
- Sans vérité extérieure (hors Deep Cube, déjà mesuré par `TestMeasureDeepCube`), « lu » et
  « sûr » sont les certitudes que le lecteur se donne par ses recoupements ; la mesure dit ce qu'un
  import accepterait, pas l'exactitude des lectures.

### Colonnes

- **Pions** : sûrs (recoupés par les pips imprimés) / complets (15 + 15 sans doute, non recoupés) /
  à vérifier / incertains (pas une position, ou contradiction).
- **Contexte** (bloc) : lu / sous le défaut du document / à vérifier / incertain.
- **Imprimés appris** : champs que le défaut appris (#991) sait imprimés à côté des diagrammes.
- **Brut sûrs** : entrées sûres telles que le relevé sort, défauts non confirmés.
- **Certifiés / complets / repli / à vér. / incert.** : après la confirmation unique, par une
  personne, des défauts du document et du sens de jeu (ce que l'interface demande une fois) —
  certifiés = importés sans option, complets = importés avec `--accept-complete`, repli = importés
  avec `--fallback --accept-complete` sous le contexte que porte leur position, champs à revoir nommés.
- **Contexte par champ** (lu / défaut / en doute / non lu) : *lu* quand le contexte est lu ou que le
  champ est appris imprimé et lu ; *défaut* quand le champ prend la valeur du défaut du document ;
  *en doute* quand un doute porte sur ce champ ; *non lu* quand l'entrée est à vérifier sans
  lecture du champ. Le trait suit les dés.

## Corpus

23 fichiers ; 18 PDF mesurés. Non mesurés : *Backgammon for Serious Player* (EPUB, format non lu),
l'archive `.tar.gz` et les deux `.doc` du *Livre du backgammon* FT (ses deux PDF sont mesurés).

Illisibles ou presque, à cause du format et non d'une lecture :

- **Magriel 1976** (Optimum) : pages en JPEG 2000 (848 images) **et JBIG2** (424) — illisible tant
  que #993 (JPEG 2000) **et** #997 (JBIG2) ne sont pas fusionnées.
- **Woolsey, *Encyclopedia*** : 133 pages en JPEG 2000 non décodées ; seules 7 pages en image
  ordinaire sont lues (0 diagramme au réglage, 2 en réserve). Attend #993.
- **FT, *Livre du backgammon*** (deux PDF) : diagrammes dessinés en police Snowie
  (AtchoumSnowie4), non reconnue : 0 diagramme → #1003.
- **Magriel & Roberts** : 62 pages vectorielles non rendues (p. 4-10, 404-413 et des pages isolées) ;
  les 441 diagrammes trouvés sont tous incertains → #998.

## Réglage (pages hors réserve)

| Livre | Diag. | Pions sûrs / complets / à vér. / incert. | Contexte lu / défaut / à vér. / incert. | Imprimés appris | Brut sûrs | Certifiés / complets / repli / à vér. / incert. |
|---|---:|---|---|---|---:|---|
| 501 Backgammon Problems (Bill Robertie) | 359 | 0 / 239 / 119 / 1 | 0 / 118 / 241 / 0 | — | 0 | 0 / 96 / 143 / 119 / 1 |
| Backgammon (Paul Magriel, Renée Magriel Roberts) (1 avert.) | 441 | 0 / 0 / 0 / 441 | 0 / 421 / 20 / 0 | — | 0 | 0 / 0 / 0 / 0 / 441 |
| Backgammon - Back Checker Strategy — Michihito Kageyama 2021 | 217 | 0 / 63 / 150 / 4 | 0 / 184 / 33 / 0 | — | 0 | 0 / 52 / 11 / 150 / 4 |
| Backgammon - Endgame Technique — Michihito Kageyama, Roland Herrera … | 248 | 106 / 24 / 106 / 12 | 0 / 32 / 216 / 0 | cube | 0 | 11 / 0 / 119 / 106 / 12 |
| Backgammon - Opening Concepts — Michihito Kageyama 2017 | 240 | 0 / 136 / 39 / 65 | 0 / 184 / 56 / 0 | — | 0 | 0 / 92 / 44 / 39 / 65 |
| Backgammon — Magriel, Paul 1976 | — | illisible : document: no diagram drawn with eXtreme Gammon's board font, and no page made of images (a page drawn with vectors is not rendered); give its pages as images: a folder of PNG or JPEG files, one a page | | | | |
| Backgammon Boot Camp (Walter Trice) | 179 | 171 / 1 / 7 / 0 | 0 / 86 / 93 / 0 | cube | 0 | 82 / 1 / 89 / 7 / 0 |
| Backgammon Deep Cube Vol. 1 (Michael Norberg) | 302 | 302 / 0 / 0 / 0 | 302 / 0 / 0 / 0 | — | 302 | 302 / 0 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 2 (Michael Norberg) | 366 | 366 / 0 / 0 / 0 | 366 / 0 / 0 / 0 | — | 366 | 366 / 0 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 3 (Michael Norberg) | 375 | 375 / 0 / 0 / 0 | 375 / 0 / 0 / 0 | — | 375 | 375 / 0 / 0 / 0 / 0 |
| Backgammon Masterclass (Masayuki Mochy Mochizuki, Marc Olsen) | 173 | 93 / 7 / 52 / 21 | 0 / 77 / 96 / 0 | — | 0 | 49 / 4 / 47 / 52 / 21 |
| Classic Backgammon Revisited (Jeremy Paul Bagai) (1 avert.) | 126 | 69 / 42 / 14 / 1 | 0 / 91 / 35 / 0 | cube | 0 | 50 / 34 / 27 / 14 / 1 |
| Cube Like a Boss (in Backgammon) (Marc Olsen) | 230 | 222 / 0 / 8 / 0 | 0 / 158 / 72 / 0 | score | 0 | 157 / 0 / 65 / 8 / 0 |
| FT_Livre_du_backgammon_VE-mars04 | — | illisible : document: no backgammon diagram found in the PDF's pages: structure: no backgammon diagram was found in the pages | | | | |
| FT_Livre_du_backgammon_VE-mars04_restaure | — | illisible : document: no backgammon diagram found in the PDF's pages: structure: no backgammon diagram was found in the pages | | | | |
| Opening Concepts (of Backgammon) (Michihito Michy Kageyama, Roland Her… | 100 | 0 / 57 / 40 / 3 | 0 / 68 / 32 / 0 | — | 0 | 0 / 45 / 12 / 40 / 3 |
| The Backgammon Book (Oswald Jacoby, John R. Crawford) | 65 | 0 / 0 / 53 / 12 | 0 / 65 / 0 / 0 | — | 0 | 0 / 0 / 0 / 53 / 12 |
| The Backgammon Encyclopedia Cube reference positions (Kit Woolsey) (1 avert.) | 0 | 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 | — | 0 | 0 / 0 / 0 / 0 / 0 |
| **Total** | 3421 | 1704 / 569 / 588 / 560 | 1043 / 1484 / 894 / 0 | | 1043 | 1392 / 324 / 557 / 588 / 560 |

### Contexte par champ (lu / défaut / en doute / non lu)

| Livre | videau | dés | trait | score |
|---|---|---|---|---|
| 501 Backgammon Problems (Bill Robertie) | 0 / 118 / 241 / 0 | 0 / 118 / 0 / 241 | 0 / 118 / 0 / 241 | 0 / 118 / 0 / 241 |
| Backgammon (Paul Magriel, Renée Magriel Roberts) | 0 / 421 / 15 / 5 | 0 / 421 / 5 / 15 | 0 / 421 / 0 / 20 | 0 / 421 / 0 / 20 |
| Backgammon - Back Checker Strategy — Michihito Kageyama 2021 | 0 / 184 / 2 / 31 | 0 / 184 / 29 / 4 | 0 / 184 / 2 / 31 | 0 / 184 / 0 / 33 |
| Backgammon - Endgame Technique — Michihito Kageyama, Roland Herrera … | 189 / 0 / 59 / 0 | 0 / 32 / 80 / 136 | 0 / 32 / 78 / 138 | 0 / 32 / 0 / 216 |
| Backgammon - Opening Concepts — Michihito Kageyama 2017 | 0 / 184 / 1 / 55 | 0 / 184 / 49 / 7 | 0 / 184 / 4 / 52 | 0 / 184 / 0 / 56 |
| Backgammon Boot Camp (Walter Trice) | 100 / 0 / 79 / 0 | 0 / 86 / 0 / 93 | 0 / 86 / 14 / 79 | 0 / 86 / 0 / 93 |
| Backgammon Deep Cube Vol. 1 (Michael Norberg) | 302 / 0 / 0 / 0 | 302 / 0 / 0 / 0 | 302 / 0 / 0 / 0 | 302 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 2 (Michael Norberg) | 366 / 0 / 0 / 0 | 366 / 0 / 0 / 0 | 366 / 0 / 0 / 0 | 366 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 3 (Michael Norberg) | 375 / 0 / 0 / 0 | 375 / 0 / 0 / 0 | 375 / 0 / 0 / 0 | 375 / 0 / 0 / 0 |
| Backgammon Masterclass (Masayuki Mochy Mochizuki, Marc Olsen) | 0 / 77 / 43 / 53 | 0 / 77 / 90 / 6 | 0 / 77 / 0 / 96 | 0 / 77 / 0 / 96 |
| Classic Backgammon Revisited (Jeremy Paul Bagai) | 113 / 0 / 13 / 0 | 0 / 91 / 0 / 35 | 0 / 91 / 22 / 13 | 0 / 91 / 0 / 35 |
| Cube Like a Boss (in Backgammon) (Marc Olsen) | 0 / 158 / 53 / 19 | 0 / 158 / 0 / 72 | 0 / 158 / 0 / 72 | 205 / 0 / 25 / 0 |
| Opening Concepts (of Backgammon) (Michihito Michy Kageyama, Roland Her… | 0 / 68 / 22 / 10 | 0 / 68 / 20 / 12 | 0 / 68 / 1 / 31 | 0 / 68 / 0 / 32 |
| The Backgammon Book (Oswald Jacoby, John R. Crawford) | 0 / 65 / 0 / 0 | 0 / 65 / 0 / 0 | 0 / 65 / 0 / 0 | 0 / 65 / 0 / 0 |
| The Backgammon Encyclopedia Cube reference positions (Kit Woolsey) | 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 |
| **Total** | 1445 / 1275 / 528 / 173 | 1043 / 1484 / 273 / 621 | 1043 / 1484 / 121 / 773 | 1248 / 1326 / 25 / 822 |

### Lecture

- **Diagrammes trouvés** : 3 421 sur 15 livres lisibles. **Pions** : 1 704 sûrs (50 %), 569
  complets, 588 à vérifier, 560 incertains (dont 441 Magriel & Roberts).
- **Contexte** : lu en bloc seulement par la police XG (Deep Cube, 1 043). Sur les livres scannés,
  le videau est lu sur 402 diagrammes (Endgame, Boot Camp, Classic Revisited) et le score sur 205
  (Cube Like a Boss) ; **dés et trait ne sont jamais lus** hors Deep Cube.
- **Après confirmation unique** : 1 392 certifiés (41 %, dont 1 043 Deep Cube), 324 complets,
  557 en repli, 588 à vérifier, 560 incertains. Hors Deep Cube : 349 certifiés sur 2 378.

## Effet de #991 (réglage)

| Livre | avant #991 : certifiés / complets / repli / à vér. / incert. | après |
|---|---|---|
| Deep Cube Vol. 1 | 302 / 0 / 0 / 0 / 0 | 302 / 0 / 0 / 0 / 0 |
| Deep Cube Vol. 2 | 366 / 0 / 0 / 0 / 0 | 366 / 0 / 0 / 0 / 0 |
| Deep Cube Vol. 3 | 375 / 0 / 0 / 0 / 0 | 375 / 0 / 0 / 0 / 0 |
| Masterclass | 49 / 4 / 0 / 99 / 21 | 49 / 4 / 47 / 52 / 21 |
| Cube Like a Boss | 0 / 0 / 0 / 230 / 0 | 157 / 0 / 65 / 8 / 0 |
| Total | 1182 / 312 / 0 / 1367 / 560 | 1392 / 324 / 557 / 588 / 560 |

- **Deep Cube** (3 volumes, 1 043 diagrammes) : inchangé, tout certifié avant comme après. Le
  lecteur de glyphes XG lit déjà le contexte entier (score compris) ; l'énoncé de REPRISE.md (« un
  score de match lu rend tout Deep Cube à vérifier ») ne se vérifie pas sur ces PDF.
- **Cube Like a Boss** (Olsen) : de 0 certifié sur 230 à **157 certifiés + 65 en repli**, 8 à
  vérifier. Le défaut apprend « score » imprimé : 205 scores lus (0 avant). Restent 25 scores non
  trouvés, 16 lus en partie, 9 non attribués → #1005.
- **Masterclass** (Mochizuki & Olsen) : 49 certifiés avant comme après ; 47 entrées passent de
  « à vérifier » au repli (champ à revoir nommé). Aucun champ appris imprimé : les gains de
  certification y sont bornés par les pions (#1004) et les dés (#1001).
- Total réglage : certifiés 1 182 → 1 392, à vérifier 1 367 → 588, repli 0 → 557.

## Réserve (lue une fois, en dernier)

| Livre | Diag. | Pions sûrs / complets / à vér. / incert. | Contexte lu / défaut / à vér. / incert. | Imprimés appris | Brut sûrs | Certifiés / complets / repli / à vér. / incert. |
|---|---:|---|---|---|---:|---|
| 501 Backgammon Problems (Bill Robertie) | 129 | 0 / 95 / 34 / 0 | 0 / 61 / 68 / 0 | — | 0 | 0 / 50 / 45 / 34 / 0 |
| Backgammon (Paul Magriel, Renée Magriel Roberts) (1 avert.) | 119 | 0 / 0 / 0 / 119 | 0 / 117 / 2 / 0 | — | 0 | 0 / 0 / 0 / 0 / 119 |
| Backgammon - Back Checker Strategy — Michihito Kageyama 2021 | 93 | 2 / 26 / 62 / 3 | 0 / 81 / 12 / 0 | — | 0 | 2 / 24 / 2 / 62 / 3 |
| Backgammon - Endgame Technique — Michihito Kageyama, Roland Herrera … | 105 | 40 / 9 / 54 / 2 | 0 / 8 / 97 / 0 | cube | 0 | 2 / 1 / 46 / 54 / 2 |
| Backgammon - Opening Concepts — Michihito Kageyama 2017 | 105 | 0 / 66 / 14 / 25 | 0 / 82 / 23 / 0 | — | 0 | 0 / 46 / 20 / 14 / 25 |
| Backgammon — Magriel, Paul 1976 | — | illisible : document: no diagram drawn with eXtreme Gammon's board font, and no page made of images (a page drawn with vectors is not rendered); give its pages as images: a folder of PNG or JPEG files, one a page | | | | |
| Backgammon Boot Camp (Walter Trice) | 56 | 54 / 1 / 0 / 1 | 0 / 26 / 30 / 0 | cube | 0 | 25 / 0 / 30 / 0 / 1 |
| Backgammon Deep Cube Vol. 1 (Michael Norberg) | 123 | 123 / 0 / 0 / 0 | 123 / 0 / 0 / 0 | — | 123 | 123 / 0 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 2 (Michael Norberg) | 191 | 191 / 0 / 0 / 0 | 191 / 0 / 0 / 0 | — | 191 | 191 / 0 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 3 (Michael Norberg) | 161 | 161 / 0 / 0 / 0 | 161 / 0 / 0 / 0 | — | 161 | 161 / 0 / 0 / 0 / 0 |
| Backgammon Masterclass (Masayuki Mochy Mochizuki, Marc Olsen) | 71 | 44 / 3 / 20 / 4 | 0 / 33 / 38 / 0 | — | 0 | 22 / 1 / 24 / 20 / 4 |
| Classic Backgammon Revisited (Jeremy Paul Bagai) (1 avert.) | 43 | 26 / 13 / 4 / 0 | 0 / 31 / 12 / 0 | cube | 0 | 20 / 7 / 12 / 4 / 0 |
| Cube Like a Boss (in Backgammon) (Marc Olsen) | 102 | 100 / 0 / 2 / 0 | 0 / 58 / 44 / 0 | score | 0 | 58 / 0 / 42 / 2 / 0 |
| FT_Livre_du_backgammon_VE-mars04 | — | illisible : document: no backgammon diagram found in the PDF's pages: structure: no backgammon diagram was found in the pages | | | | |
| FT_Livre_du_backgammon_VE-mars04_restaure | — | illisible : document: no backgammon diagram found in the PDF's pages: structure: no backgammon diagram was found in the pages | | | | |
| Opening Concepts (of Backgammon) (Michihito Michy Kageyama, Roland Her… | 46 | 0 / 26 / 18 / 2 | 0 / 31 / 15 / 0 | — | 0 | 0 / 22 / 4 / 18 / 2 |
| The Backgammon Book (Oswald Jacoby, John R. Crawford) | 22 | 0 / 0 / 14 / 8 | 0 / 22 / 0 / 0 | — | 0 | 0 / 0 / 0 / 14 / 8 |
| The Backgammon Encyclopedia Cube reference positions (Kit Woolsey) (1 avert.) | 2 | 0 / 2 / 0 / 0 | 0 / 2 / 0 / 0 | — | 0 | 0 / 2 / 0 / 0 / 0 |
| **Total** | 1368 | 741 / 241 / 222 / 164 | 475 / 552 / 341 / 0 | | 475 | 604 / 153 / 225 / 222 / 164 |

### Contexte par champ (lu / défaut / en doute / non lu)

| Livre | videau | dés | trait | score |
|---|---|---|---|---|
| 501 Backgammon Problems (Bill Robertie) | 0 / 61 / 68 / 0 | 0 / 61 / 0 / 68 | 0 / 61 / 0 / 68 | 0 / 61 / 0 / 68 |
| Backgammon (Paul Magriel, Renée Magriel Roberts) | 0 / 117 / 2 / 0 | 0 / 117 / 0 / 2 | 0 / 117 / 0 / 2 | 0 / 117 / 0 / 2 |
| Backgammon - Back Checker Strategy — Michihito Kageyama 2021 | 0 / 81 / 2 / 10 | 0 / 81 / 9 / 3 | 0 / 81 / 1 / 11 | 0 / 81 / 0 / 12 |
| Backgammon - Endgame Technique — Michihito Kageyama, Roland Herrera … | 77 / 0 / 28 / 0 | 0 / 8 / 32 / 65 | 0 / 8 / 37 / 60 | 0 / 8 / 0 / 97 |
| Backgammon - Opening Concepts — Michihito Kageyama 2017 | 0 / 82 / 0 / 23 | 0 / 82 / 23 / 0 | 0 / 82 / 0 / 23 | 0 / 82 / 0 / 23 |
| Backgammon Boot Camp (Walter Trice) | 29 / 0 / 27 / 0 | 0 / 26 / 0 / 30 | 0 / 26 / 3 / 27 | 0 / 26 / 0 / 30 |
| Backgammon Deep Cube Vol. 1 (Michael Norberg) | 123 / 0 / 0 / 0 | 123 / 0 / 0 / 0 | 123 / 0 / 0 / 0 | 123 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 2 (Michael Norberg) | 191 / 0 / 0 / 0 | 191 / 0 / 0 / 0 | 191 / 0 / 0 / 0 | 191 / 0 / 0 / 0 |
| Backgammon Deep Cube Vol. 3 (Michael Norberg) | 161 / 0 / 0 / 0 | 161 / 0 / 0 / 0 | 161 / 0 / 0 / 0 | 161 / 0 / 0 / 0 |
| Backgammon Masterclass (Masayuki Mochy Mochizuki, Marc Olsen) | 0 / 33 / 13 / 25 | 0 / 33 / 37 / 1 | 0 / 33 / 0 / 38 | 0 / 33 / 0 / 38 |
| Classic Backgammon Revisited (Jeremy Paul Bagai) | 35 / 0 / 8 / 0 | 0 / 31 / 0 / 12 | 0 / 31 / 4 / 8 | 0 / 31 / 0 / 12 |
| Cube Like a Boss (in Backgammon) (Marc Olsen) | 0 / 58 / 34 / 10 | 0 / 58 / 0 / 44 | 0 / 58 / 0 / 44 | 91 / 0 / 11 / 0 |
| Opening Concepts (of Backgammon) (Michihito Michy Kageyama, Roland Her… | 0 / 31 / 13 / 2 | 0 / 31 / 8 / 7 | 0 / 31 / 0 / 15 | 0 / 31 / 0 / 15 |
| The Backgammon Book (Oswald Jacoby, John R. Crawford) | 0 / 22 / 0 / 0 | 0 / 22 / 0 / 0 | 0 / 22 / 0 / 0 | 0 / 22 / 0 / 0 |
| The Backgammon Encyclopedia Cube reference positions (Kit Woolsey) | 0 / 2 / 0 / 0 | 0 / 2 / 0 / 0 | 0 / 2 / 0 / 0 | 0 / 2 / 0 / 0 |
| **Total** | 616 / 487 / 195 / 70 | 475 / 552 / 109 / 232 | 475 / 552 / 45 / 296 | 566 / 494 / 11 / 297 |

La réserve confirme le réglage : pions sûrs 54 % (741 / 1 368), certifiés après confirmation 44 %
(604, dont 475 Deep Cube), 164 incertains (119 Magriel & Roberts). Cube Like a Boss : 58
certifiés + 42 en repli sur 102 ; score lu sur 91. Hors Deep Cube, dés et trait toujours non lus.

## Défauts récurrents ouverts

| Issue | Défaut | Ampleur (réglage) | Exemple |
|---|---|---|---|
| #998 | pions-lettres X/O lus comme piles sombres | 441 / 441 Magriel & Roberts incertains ; Opening Concepts 2017 | Magriel & Roberts p. 12 |
| #999 | videau dessiné non lu (valeur, ou absent) | 517 doutes ; 501 Problems 241 / 359 | 501 Problems p. 21 |
| #1000 | plateau de sortie non lu | ≈ 1 080 entrées ; Jacoby & Crawford 0 certifié | Backgammon Book p. 57 |
| #1001 | dés comptés dans l'image jamais certifiés | 0 dé lu sur 2 378 diagrammes scannés | Masterclass p. 21 |
| #1002 | légende de contexte non lue sans couche texte | 501 Problems 359 / 359 | 501 Problems p. 21 |
| #1003 | diagrammes en police Snowie | FT : 0 diagramme | — (police AtchoumSnowie4) |
| #1004 | Masterclass : pions mal classés, pips contraires | 21 incertains, ≈ 40 pips contraires | Masterclass p. 20 |
| #1005 | score imprimé non lu en entier | 50 / 230 Cube Like a Boss | Cube Like a Boss p. 41 |

Formats : #993 (JPEG 2000 : Woolsey, Magriel 1976) et #997 (JBIG2 : Magriel 1976).

## Suites

- Rejouer `./banc analyse <étiquette>` puis `./banc mesure reglage <étiquette>` après chaque
  correctif ; la réserve déjà lue ne sert plus qu'en comparaison finale, jamais au réglage.
- Le sens de jeu supposé (1 422 entrées sans pips imprimés) et le contexte « sous le défaut »
  dépendent d'une confirmation par une personne : sans elle, seules les 1 043 entrées Deep Cube
  sont importables sans option.

## Effet de #999 et #1001 (réglage seulement)

Lecteur de structure seul (`TestMeasureContextOnPages`, `document/measure_reglage_test.go`) :
chaque livre scanné est lu sur ses seules pages de réglage (`tirage.json`), aucune page de
réserve n'est composée. Avant = `main` à 071775534. Deep Cube (police XG) ne passe pas par ce
lecteur et n'est pas affecté. Règles : ADR-0121.

| Livre | Diag. | videau lu | doutes videau | dés lus | doutes dés |
|---|---:|---|---|---|---|
| 501 Problems (Robertie) | 359 | 0 → 271 | 242 → 88 | 0 → 0 | 0 → 0 |
| Boot Camp (Trice) | 179 | 65 → 65 | 79 → 6 | 0 → 0 | 0 → 0 |
| Cube Like a Boss | 230 | 61 → 61 | 53 → 53 | 0 → 0 | 0 → 0 |
| Endgame Technique | 248 | 189 → 189 | 59 → 59 | 0 → 6 | 80 → 74 |
| Masterclass | 173 | 3 → 3 | 43 → 43 | 1 → 46 | 90 → 45 |
| Opening Concepts 2017 | 240 | 6 → 6 | 1 → 1 | 1 → 4 | 49 → 46 |
| Back Checker Strategy | 217 | 41 → 41 | 2 → 2 | 0 → 0 | 29 → 29 |
| Classic Backgammon Revisited | 126 | 47 → 47 | 13 → 9 | 0 → 0 | 0 → 0 |
| Opening Concepts (Kageyama, Herrera) | 100 | 1 → 1 | 22 → 5 | 0 → 0 | 20 → 20 |
| Magriel & Roberts | 441 | 0 → 10 | 15 → 10 | 0 → 0 | 5 → 5 |
| Jacoby & Crawford | 65 | 0 → 0 | 0 → 0 | 0 → 0 | 0 → 0 |
| Woolsey, Encyclopedia | 142 | 0 → 0 | 0 → 0 | 0 → 0 | 0 → 0 |
| **Total** | | 413 → 694 | 529 → 276 | 2 → 56 | 273 → 219 |

- **Videau vierge** (501 Problems, Magriel et Roberts) : un carré seul sans aucune marque est le
  videau centré, où qu'il soit dessiné. Vérifié à l'œil sur 501 Problems p. 21 et sur Magriel
  et Roberts p. 105.
- **Carré cherché plus loin et plus large** (Boot Camp 77 → 2 « none is found », Opening
  Concepts 21 → 4, 501 Problems 85 → 88 restants) : un « 64 » au milieu trouvé mais non lu ne
  laisse pas de doute, il reste sous le défaut. Le rapport largeur/hauteur 1,75 n'est motivé que
  par 501 Problems (scan étiré).
- **Doute qui nomme pourquoi** : chiffres non reconnus, valeur qui n'est pas celle d'un videau,
  ou aucun chiffre trouvé (clair sur foncé, ou trop petit : Masterclass, Cube Like a Boss).
- **Dés** : confirmés par un jet nommé (« rolls a 65 ») ou un coup écrit (« 2/0 2/1 »,
  « 6/5*(2) »). Vérifié à l'œil : Masterclass p. 26 et p. 183, Endgame p. 17, p. 138 et
  p. 213. Les contradictions d'Endgame (1 → 4) viennent de la légende du diagramme voisin
  (« After 8/2 7/2 ») : elles restent des doutes.
- Restent : 501 Problems 88 carrés non trouvés (trait gris clair), Cube Like a Boss 53 videaux
  clair sur foncé hors du cadre cherché, dés non comptés (« N dice drawn »), du ressort de la
  localisation.
