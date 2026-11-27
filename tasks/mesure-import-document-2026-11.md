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

## Pions : #998, #1000, #1004 (réglage seulement)

Mesure par `TestMeasureReglage` (`pkg/blunderdb/document/structure/measure_reglage_test.go`) :
le lecteur de structure ne décode que les pages de réglage de `tirage.json`, apprend le style du
livre sur elles seules (d'où des comptes distincts du tableau ci-dessus, qui lisait le livre
entier). Colonnes : sûrs (état confirmé) / complets (15 + 15, ou 15 avec le plateau lu, sans
doute) / à vérifier / incertains ; « sortis supposés » = doutes « taken as borne off ». Avant :
`main` 071775534 ; après : branche `feat/doc-pions`. Les autres livres de réglage (Boot Camp,
Cube Like a Boss, Classic Revisited, Back Checker, Opening Concepts 2019, 501 Problems,
Opening Concepts 2017) : aucune entrée changée, ni d'état ni de position.

| Livre | Avant | Après | Sortis supposés avant → après | Plateau ≠ 15 (nommé) |
|---|---|---|---:|---:|
| Magriel & Roberts | 0 / 0 / 0 / 441 | 0 / 0 / 0 / 441 | 437 → 0 | — |
| Endgame Technique | 11 / 80 / 153 / 4 | 11 / 89 / 144 / 4 | 136 → 88 | 39 |
| Masterclass | 50 / 71 / 36 / 16 | 50 / 71 / 36 / 16 | 52 → 45 | 6 |
| Backgammon Book (Jacoby) | 0 / 0 / 65 / 0 | 0 / 0 / 65 / 0 | 130 → 129 | 1 |

- **#998** : les 441 diagrammes de Magriel & Roberts restent incertains, mais par un seul doute
  qui nomme la cause (« the document draws no filled checker ») au lieu de piles sombres
  comptées sur les flèches grisées. Règle générale : les deux classes de pions apprises sont
  toutes deux plus claires que le gris moyen (0,5, papier = 1, encre = 0) — ici 0,73 et 0,95,
  les O sur flèche grisée et les O sur blanc ; les livres à pions pleins ont leur classe
  sombre entre 0,01 et 0,40. Lire les lettres X/O n'est pas fait.
- **#1000** : tranché pour la lecture du plateau quand le document le dessine. Les pions sortis
  vus de profil (barres) sont comptés dans une bande bornée par deux traits verticaux à côté de
  la rangée du camp ; une entrée n'est complète que si dessinés + plateau font quinze, sinon un
  doute nommé. Endgame : 9 entrées passent à complètes (positions inchangées, vérifiées à l'œil
  p. 27) ; 39 plateaux lus ne complètent pas quinze — barres jointives ou contours doubles
  (p. 113 : dix pions clairs sortis lus douze), doute nommé, jamais une entrée complète.
  Distinguer un demi-plateau d'une fin de partie n'apporte rien de plus : le demi-plateau est
  déjà un doute nommé (« half board »).
- **#1004** (diagnostic) : les classes de couleur de Masterclass sont saines (0,40 / 1). Les
  pips contraires et les 16 pions d'un camp viennent des diagrammes annotés : flèches de coup
  et pions fantômes hachurés dessinés sur le plateau (p. 28, p. 52), lus comme pions ou cachant
  des pions ; un camp « sans pion » est un camp tout sorti (p. 20, plateau de 15 barres serrées
  non compté). La couche texte imprime, à droite, le nombre de pions sortis sous chaque
  plateau (p. 91 : 13 et 3) : pris pour des pips, il contredit les pips de l'image — doute
  « the two disagree », juste. Restent ouverts : reconnaître une flèche de coup et nommer ce
  doute ; ne pas prendre les comptes de plateau imprimés pour des pips.
- Essai écarté : n'étendre une pile sombre par sa longueur qu'à partir d'un pion vu un à un.
  Il retirait les piles fantômes mais faisait tomber Cube Like a Boss de 157 à 78 sûrs et
  Masterclass de 50 à 23 : des pions sombres jointifs n'ont souvent aucun bord visible.

### #1004 : flèches de coup et comptes du plateau

Deux règles générales. (1) Une flèche de coup est un trait droit, fin, oblique (ni à moins de
8° d'une rangée ni à moins de 16° d'un côté de flèche), qui court seul sur le papier sur au
moins 2,5 points (`structure/lines.go`) ; ses pixels fins sont repeints de la teinte de part et
d'autre avant la lecture des pions : il ne ferme plus le contour d'un pion absent ni ne salit
l'intérieur d'un pion réel ; là où il croise un contour il n'est plus fin et le contour reste.
Une hachure ne passe pas : son trait voisin est à côté (« seul » exigé sur la moitié du trait)
et la marche la quitte à chaque ligne. (2) Des nombres de quinze ou moins à côté du plateau
peuvent être les pions sortis imprimés près du plateau : ils n'apprennent plus où le document
imprime ses pips. Mesure de réglage, tous livres, avant `main` 8947fa0d5 : seul Masterclass
change ; aucune entrée des douze autres livres ne change (ni état, ni position, ni pips).
Masterclass 61 / 60 / 36 / 16 → 67 / 61 / 35 / 10 : pp. 91, 105, 154, 286 sûres (pips de
l'image, plus contredits par les comptes du plateau) ; pp. 74, 100 sûres et 174, 186, 194
complètes (pion fantôme fermé par une flèche retiré). Ouvert : une tête de flèche au pied d'une
flèche fait encore un pion (pp. 58, 84, seize pions, incertaines, doute nommé) ; une flèche
posée sur un pion le cache (p. 28, à vérifier, pips contraires nommés) ; une flèche horizontale
ou verticale n'est pas reconnue.

### Après revue (plateau durci)

Videau chevauchant la bande : bande non lue ; barres d'épaisseur et de pas réguliers, rectangles
creux refusés ; course coupée par la séparation des rangées écartée ; un plateau qui porte des
barres doit faire quinze même quand le camp en a quinze dessinés (doute nommé sinon, pips ou
non). Mesure de réglage, tous livres : seuls Endgame et Masterclass changent. Endgame
11/80/153/4 → 7/65/172/4 (1 entrée gagnée, 16 complètes et 4 sûres perdues par un plateau lu
à tort à côté d'un camp de quinze) ; Masterclass 50/71/36/16 → 49/71/37/16 (p. 144). Ouvert :
ces lectures de plateau à tort (p. 61, 116, 131) — résorbé ci-dessous.

### Traits et piles flottantes écartés, seuil creux relatif à l'encre

Les lectures à tort (p. 61, 116, 131, 134 d'Endgame) étaient des traits de 1 à 3 px — contour
du videau « 2 », bord de cadre — comptés comme barres. Règles générales ajoutées à `trays` :
une barre fait au moins 0,08 point (un pion vu de profil n'est jamais un trait) ; les barres
reposent contre le bord extérieur du plateau, au niveau du champ (à moins d'un demi-point),
sinon le plateau n'est pas lu (pile coupée par les traits du cadre, Endgame p. 151 et p. 200 :
6 lus pour 13, 1 pour 2 ; cadre séparé de Masterclass p. 144 : 7 lus pour 10). Le seuil
« document sans pion plein » est à mi-chemin entre l'encre mesurée (10e centile du gris sous
l'encre, médiane des diagrammes) et le papier, au lieu de 0,5.

Mesure de réglage, tous livres, `main` 368befef4 / branche avant (fusion f28322f7c) / après
(2bfbe600c), sûrs / complets / à vérifier / incertains :

| Livre | main | avant | après |
|---|---|---|---|
| Endgame Technique | 12 / 80 / 152 / 4 | 8 / 65 / 171 / 4 | 12 / 80 / 152 / 4 |
| Masterclass | 61 / 60 / 36 / 16 | 60 / 60 / 37 / 16 | 61 / 60 / 36 / 16 |
| Magriel 1976 | 0 / 0 / 331 / 94 | 0 / 0 / 0 / 425 | 0 / 0 / 0 / 425 |
| Magriel & Roberts | 0 / 0 / 0 / 441 | 0 / 0 / 0 / 441 | 0 / 0 / 0 / 441 |

Les neuf autres livres de réglage : identiques à `main`, entrée par entrée. Après = `main` entrée
par entrée sauf Magriel 1976 (pions en lettres X/O, vérifié p. 18 : doute « no filled checker »
au lieu de positions lues sur des lettres). Aucune entrée complète ou sûre nouvelle. Le plateau
lu n'apporte encore rien d'autre : 6 diagrammes d'Endgame perdent le doute « taken as borne
off » (plateau lu et quinze faits) sans changer d'état, retenus par d'autres doutes. Ouvert :
un plateau dont le trait du champ est coupé par la pile (p. 27) ou à cadre séparé (Masterclass)
n'est pas lu.

Après revue : les côtés d'un rectangle creux sont cherchés aux extrémités de l'étendue encrée
de chaque barre, où qu'elles soient dans la bande (un rectangle en retrait, contour d'un videau
non trouvé, passait pour deux pions sortis et complétait un camp de treize). Mesure refaite,
`main` 2c5358ae5 → branche 249cf9688 : tous les livres identiques à `main` entrée par entrée
sauf Magriel 1976 (0 / 0 / 331 / 94 → 0 / 0 / 0 / 425, pions en lettres) ; aucune entrée
complète ou sûre nouvelle.

## Diagrammes en police Snowie (#1003) — réglage, 2026-11-27

Lecteur `document/snowieboard.go` + `ttshape.go` : diagramme trouvé par ses deux rangées de numéros
de points, pions lus d'après la **forme** des glyphes TrueType embarqués (anneau d'encre = pion,
intérieur encré = noir), jamais d'après le code ni le nom de la police ; codes partagés lus en
texte : numéros, chiffres des pips, pions numérotés, videau et dés à leur place. Pages de réglage
seulement (`tirage.json`), aucune page de réserve lue.

| PDF | Diagrammes (rangées « 24 » dessinées) | Pions sûrs / à vér. / incert. | Contexte à vér. / incert. |
|---|---:|---|---|
| FT *Livre du backgammon* (Snowie 4) | 100 (104) | 97 / 3 / 0 | 97 / 3 |
| FT *Livre du backgammon* restauré (Snowie 4 puis Atchoum reconstruite) | 173 (175) | 143 / 9 / 21 | 169 / 4 |

- **Sûr** = 15 + 15 pions et pips imprimés égaux aux pips lus (seule hypothèse de couleur du joueur
  du bas qui les vérifie). Recoupement indépendant : 59 positions sûres identiques dans les deux PDF,
  dont 25 lues dans le restauré avec l'autre police (pion noir = disque posé sur la flèche) et dans
  l'original avec Snowie 4.
- **Contexte** jamais lu en entier : la police n'imprime pas le score (doute nommé) ; le trait n'est
  lu que si des dés sont dessinés (5 diagrammes). 7 videaux dessinés deux ou trois fois → incertain.
- **Incertains** (21, restauré) : planches d'illustration sans pips (demi-plateaux, pions isolés),
  qui ne sont pas des positions légales.
- **Non trouvés** (4 + 2) : rangée de numéros décalée d'une colonne (original p. 256), demi-rangée
  ou planche partielle (original p. 25, restauré p. 23, 25, 75, 183).
- Durée : les deux PDF en 1,2 s.

**Autres livres, avant / après** : sur les pages de réglage des 16 autres PDF, aucune grille Snowie
n'est trouvée (recensement page par page) ; le lecteur n'agit que sur une grille trouvée, donc leurs
relevés sont inchangés par construction. Vérifié en relançant l'analyse avant (`main` 3e6b26122) et
après sur Deep Cube 1-3, Boot Camp et Cube Like a Boss : relevés de réglage identiques.

**Réglages tirés de ce seul livre** (à revoir sur un autre document Snowie) : tolérance
d'alignement numéros/colonnes portée à 0,4 glyphe (p. 28, numéros décalés de 3 pt) ; écart entre
rangées de numéros accepté de 11,7 à 13,5 glyphes, les lignes du plateau ancrées sur le plateau
lui-même (p. 89) ; le pion numéroté décide de la couleur de sa case (Atchoum dessine un pion blanc
sous le noir numéroté) ; le nombre imprimé à côté du plateau de sortie l'emporte sur ses barres.
Seuils de forme (anneau couvrant 85 % des rayons, moins de 35 % d'encre juste au-delà ; intérieur
noir à partir de 60 %, blanc sous 35 %) choisis sur les deux polices du livre et vérifiés sur des
glyphes synthétiques.

## Effet de #999 et #1001 (réglage seulement)

Lecteur de structure seul (`TestMeasureContextOnPages`, `document/measure_reglage_test.go`) :
chaque livre scanné est lu sur ses seules pages de réglage (`tirage.json`), aucune page de
réserve n'est composée. Avant = `main` à 071775534. Deep Cube (police XG) ne passe pas par ce
lecteur et n'est pas affecté. Règles : ADR-0121, après revue (attribution des mots et des
carrés au plateau le plus proche, face vierge attestée sur trois diagrammes, aspect jugé
contre l'étirement des pions, seuil de face vierge relatif au contraste du carré).

| Livre | Diag. | videau lu | doutes videau | dés lus | doutes dés | trait lu |
|---|---:|---|---|---|---|---|
| 501 Problems (Robertie) | 359 | 0 → 256 | 242 → 103 | 0 → 0 | 0 → 1 | 0 → 0 |
| Boot Camp (Trice) | 179 | 65 → 65 | 79 → 5 | 0 → 0 | 0 → 0 | 0 → 0 |
| Cube Like a Boss | 230 | 61 → 61 | 53 → 53 | 0 → 0 | 0 → 0 | 0 → 0 |
| Endgame Technique | 248 | 189 → 189 | 59 → 59 | 0 → 3 | 80 → 77 | 52 → 52 |
| Masterclass | 173 | 3 → 3 | 43 → 43 | 1 → 42 | 90 → 49 | 88 → 88 |
| Opening Concepts 2017 | 240 | 6 → 6 | 1 → 1 | 1 → 4 | 49 → 46 | 21 → 21 |
| Back Checker Strategy | 217 | 41 → 41 | 2 → 2 | 0 → 0 | 29 → 29 | 6 → 6 |
| Classic Backgammon Revisited | 126 | 47 → 51 | 13 → 5 | 0 → 0 | 0 → 0 | 0 → 0 |
| Opening Concepts (Kageyama, Herrera) | 100 | 1 → 1 | 22 → 5 | 0 → 0 | 20 → 20 | 18 → 18 |
| Magriel & Roberts | 441 | 0 → 5 | 15 → 11 | 0 → 0 | 5 → 5 | 0 → 0 |
| Jacoby & Crawford | 65 | 0 → 0 | 0 → 0 | 0 → 0 | 0 → 0 | 0 → 0 |
| Woolsey, Encyclopedia | 142 | 0 → 0 | 0 → 0 | 0 → 0 | 0 → 0 | 0 → 0 |
| **Total** | | 413 → 678 | 529 → 287 | 2 → 49 | 273 → 227 | 185 → 185 |

- **Aucune lecture perdue** : videau, dés et trait lus ne baissent sur aucun livre.
- **Videau vierge** (501 Problems, Magriel et Roberts) : lu centré seulement là où le document
  dessine un carré seul vierge au même endroit à côté d'au moins trois diagrammes. Magriel et
  Roberts : 5 lus, 2 faces vierges isolées laissées en doute. Vérifié à l'œil sur 501 Problems
  p. 21 et sur Magriel et Roberts p. 105.
- **Aspect et attribution des carrés** : les piles de pions sortis de Classic
  Revisited ne passent plus pour un second videau. Les 4 nouvelles lectures (p. 165, 179,
  183 et 185) ont été vérifiées à l'œil : un 2 du côté de son propriétaire. 501 Problems
  garde 103 carrés non trouvés.
- **Carré cherché plus loin** (Boot Camp 77 → 2 « none is found », Opening Concepts 21 → 4) :
  un « 64 » au milieu trouvé mais non lu ne laisse pas de doute, il reste sous le défaut.
- **Doute qui nomme pourquoi** : chiffres non reconnus, valeur qui n'est pas celle d'un videau,
  aucun chiffre trouvé (clair sur foncé, ou trop petit), ou face vierge isolée.
- **Dés** : confirmés par un jet nommé (« rolls a 65 ») ou un coup écrit (« 2/0 2/1 »,
  « 6/5*(2) »), du texte attribué à ce diagramme, deux fois si le texte cite plusieurs jets.
  Vérifié à l'œil : Masterclass p. 26 et p. 183, Endgame p. 17, p. 138 et p. 213.
- Restent : Cube Like a Boss 53 videaux clair sur foncé hors du cadre cherché, dés non comptés
  (« N dice drawn »), du ressort de la localisation.

### Garde de la face vierge (réglage seulement)

Règle ajoutée : une face vierge n'est lue centrée que si le document n'imprime jamais, à la même
place (hauteur et côté du champ) sur un autre diagramme, un videau dont la face n'est pas vierge
et n'est pas lue 64 ; sinon doute nommé (« where N other diagrams print a value »). Même banc
(`TestMeasureContextOnPages`, `tirage.json`, pages de réglage seulement), lecteur à 56176ce81.

- Les douze livres scannés donnent exactement les comptes de la colonne « après » du tableau
  ci-dessus (videau lu 678, doutes videau 287, dés lus 49) : aucune lecture perdue, aucune
  gagnée.
- La garde ne se déclenche sur aucun diagramme du réglage : là où ces livres dessinent un
  videau vierge (501 Problems, Magriel et Roberts), aucun autre diagramme n'imprime de valeur à
  la même place. Elle vaut pour les documents inconnus, où un chiffre perdu au scan ferait
  sinon lire « centré » à tort.

## Légende lue dans l'image et score OCR (#1002, #1005) — réglage seulement

Lecteur de structure seul (`TestMeasureContextOnPages`, `tirage.json`, pages de réglage
seulement, aucune page de réserve composée). Avant = `main` à 368befef4.

| Livre | Diag. | dés lus | jet de légende (doute) | légende au jet non lu (doute) | score lu | doutes score |
|---|---:|---|---:|---:|---|---|
| 501 Problems (Robertie) | 359 | 0 → 0 | 69 | 135 | 0 → 0 | 0 → 0 |
| Classic Backgammon Revisited | 126 | 0 → 0 | 8 | 19 | 0 → 0 | 0 → 0 |
| Opening Concepts (Kageyama, Herrera) | 100 | 0 → 1 | 0 | 0 | 0 → 0 | 0 → 0 |
| Cube Like a Boss | 230 | 0 → 0 | 0 | 0 | 205 → 221 | 25 → 9 |
| Les huit autres livres scannés | | inchangés | 0 | 0 | inchangés | inchangés |

- **Aucune lecture perdue** : videau, dés, trait et score lus ne baissent sur aucun des douze
  livres (comparaison diagramme par diagramme).
- **Légende dans l'image** (page sans couche texte) : lignes au-dessus et au-dessous du plateau
  coupées en mots par l'encre ; un mot n'est lu que comme un mot de légende (couleur, « to play »,
  « rolls »…) dessiné dans des polices courantes, ou comme un nombre dont chaque chiffre est
  reconnu ; sinon « ? ». Une légende = mot de couleur ouvrant la ligne (ou après un nombre, une
  fin de phrase, un mot non lu), le verbe, le jet finissant la ligne : la prose (« if White rolls
  15 and plays ») n'en est pas une. Sans dés dessinés, le jet et le trait de la légende sont
  donnés **en doute** (une seule source), jamais lus. Avec des dés dessinés, le mot lu sert de
  confirmation comme un mot de couche texte : Opening Concepts p. 27, « How does Black play 62 »
  confirme les dés 6-2 comptés.
- Vérifié à l'œil : 501 Problems p. 21 (11, 65) et p. 22 (62) ; Classic Revisited p. 17 (54).
  Deux livres de mise en page différente (légende centrée sous le plateau, légende en italique
  gras à droite). Les 135 + 19 jets non lus sont surtout des 3 et des 2 gras d'un scan à faible
  résolution (un 3 dont les boucles se ferment) : doute nommé, pas de chiffre deviné.
- **Score OCR** (Cube Like a Boss) : tag collé à son nombre (« pip:9 »), point-virgule pour
  deux-points, « po int » coupé ; deux scores égaux se passent d'attribution, mais chaque pip
  imprimé doit valoir l'un des pips lus (sinon doute nommé) et sans pip imprimé le score reste
  en doute (une seule source). Les 9 doutes restants : pips manquants avec scores différents ou
  scores égaux sans pip, bandeau incomplet.
- **Videau** : un 64 lu en haut ou en bas est un videau possédé ; il n'exempte plus la garde de
  face vierge (comptes de videau inchangés au réglage).
- **Gabarits** : `words.bin` et `figures.bin` tirés des dix polices nommées dans
  `words_gen_test.go` (Liberation, C059, DejaVu), régénérés à l'identique octet pour octet.
- **Réglages** : aucun propre à un livre. Seuils choisis sur 501 Problems et vérifiés sur
  Classic Revisited et sur des légendes synthétiques en police Go (absente des gabarits) :
  corrélation de mot 0,60 avec 0,04 d'avance (mots justes 0,72-0,88, autres ≤ 0,37 sur
  501 Problems), chiffre de légende par corrélation centrée 0,70 avec 0,06 d'avance.

### Le plateau seul ne complète plus un camp

Règle : un camp dont les quinze reposent sur des pions comptés dans le plateau de sortie ne rend
l'entrée complète ou sûre que si des pips imprimés concordent avec la position ; sinon doute
nommé (« N borne-off checkers counted in the tray, no printed pips confirm them »). Une forme
parasite lue comme barres (« [ » ouvert, contour pointillé, contour de videau non détecté) ne
complète donc jamais une entrée. Mesure de réglage, tous livres, branche caaf9b438 → d075f3440 :
2 945 diagrammes, comptes sûrs / complets / à vérifier / incertains identiques livre par livre ;
une seule entrée change (Endgame Technique p. 88, demi-plateau déjà à vérifier) : elle gagne le
doute nommé, sans changer d'état. Aucune entrée complète de ces livres ne dépendait du plateau.

## Chiffres des pips dans l'image : victoire loyale (#1008)

Lecteur de structure seul (`TestMeasureContextOnPages`, `tirage.json`, pages de réglage seulement).
Avant = `main` à 2c5358ae5. 18 PDF ; 13 passent par le lecteur d'image, les cinq autres (Deep Cube
1-3, deux PDF du *Livre du backgammon* FT) n'y trouvent aucun diagramme et ne sont pas concernés.

- **Constat corrigé** : la lecture des pips était déjà une corrélation centrée (`unitVector` finit
  par `centred`), avec les mêmes gabarits que la légende ; la seule différence est la requête
  (chiffre redressé et flouté pour les pips, glyphe brut pour la légende). Le commentaire de
  `caption.go` qui parlait d'un cosinus non centré est corrigé ; les deux lecteurs partagent
  désormais `figureScores` et `bestFigure`.
- **Essai rejeté** : exiger que le glyphe brut classe aussi le chiffre en tête coûte 705 → 449
  pips lus et 742 → 660 sûres (Trice, Classic Revisited, Endgame surtout, chiffres 7 et 1) ; non
  retenu.
- **Mécanisme reproduit** (synthétique, Go Bold épaissi de 1 à 3 px, 18-36 pt) : un 2 lourd est
  lu 1 quand le document a appris ses propres 1 lourds mais pas ses 2 — le 1 appris l'emporte sur
  le 2 léger des polices par le seul poids du trait, sous les deux mesures. Garde retenue
  (`fairWin`) : un chiffre gagné par un gabarit appris n'est lu que si, polices seules, aucun
  chiffre que le document n'a pas appris n'arrive à `figureMargin` de lui ; sinon doute. Avant :
  10 lectures « 1 » sur 15 cas de 2 épaissi de 2-3 px ; après : aucune.
- **Mesure** : sorties identiques octet pour octet sur les 13 livres (pips lus dans l'image 705,
  égaux aux pips comptés sur un diagramme complet 549, contraires 4, contraires à la couche
  texte 15, sûres 742) : la garde ne se déclenche sur aucune page de réglage ; aucune sûre ne
  change de position, aucune nouvelle sûre.

### Garde tenue par poids, rivaux excusés par les seuls gabarits appris (#1008, suite)

Revue de la garde : tenue par chiffre, elle ne faisait plus rien dès que la numérotation
enseignait les dix chiffres (un 2 lourd lu 1 dans au moins 40 cas synthétiques, Go Bold 16-34 pt, 2 et 1
épaissis de 1 à 3 px, dix chiffres maigres appris). Chaque gabarit porte désormais le poids de son
trait (épaisseur sur hauteur). Un chiffre gagné par un gabarit appris exige, pour chaque rival que
le document n'a pas appris au poids du gagnant, que les polices seules placent le gagnant devant de
`figureMargin`, sinon doute. Seul un gabarit appris excuse un rival : un 2 gras des polices, du
poids du 1 lourd appris, ne dit rien des 2 du document (c'était la régression de la première
version par poids, 2 de 16 pt épaissi de 2 px lu 1).

- **Tests** : `TestTypefaceBoldIsNoAlibi` (rouge si les polices comptent parmi les rivaux de même
  poids), `TestTenTaughtFiguresKeepTheGuard` (dix chiffres appris maigres, un 1 lourd appris : le
  2 lourd n'est jamais lu 1, le 1 lourd lu comme sans la numérotation), `TestLearnedRivalIsNoAlibi`,
  `TestMisreadNeverTeaches` (un 2 mal lu ne nourrit jamais le 1 au tour suivant).
- **Mesure**, réglage seulement, avant `main` 8947fa0d5 : sorties identiques octet pour octet sur
  les 13 livres lus par l'image (2 945 diagrammes, pips image 705, égaux aux comptés 549,
  contraires 4, contraires à la couche texte 15, complets 1 372, sûres 742). Les cinq autres
  (Deep Cube 1-3, deux PDF FT) n'ont aucun diagramme lu par l'image.

### Un diagramme complété par son plateau de sortie vote pour la place des pips

Depuis que le plateau seul ne complète plus un camp, un diagramme dont les quinze reposent sur des
pions comptés dans le plateau ne votait plus dans `learnPipSlots` (le vote se fait sans pips
imprimés, donc `Complete` faux). Il vote de nouveau (`Diagram.pipVoter`) : un pion sorti ne compte
aucun pip, une forme parasite mal comptée dans le plateau laisse donc le compte de pips intact.
`Complete` et l'état ne changent pas. Test : `TestTrayDiagramsVoteForPipPlaces` (livre où chaque
diagramme a des pions sortis, pips dans l'image : aucune place apprise sans la correction). Mesure
ci-dessus : aucune sortie ne change sur les pages de réglage (les livres à plateau y votent déjà
assez par leurs autres diagrammes).
