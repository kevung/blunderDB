# Ark et impression : état de reprise

Chantier : notation ArkID (Animal Register Key), impression dense sans diagramme, archive
papier par codes QR. Issues #984 à #987.

## Fait, dans main

- #984 ArkID : codec `pkg/blunderdb/domain/arkid.go`, commande `arkid`, `--format arkid`,
  routes `positions.arkId` / `positions.fromArkID`, action « Copier l'ArkID », glossaire avec
  le bestiaire dans les neuf langues, ADR-0118.
- #985 archive papier : `pkg/blunderdb/arkqr` (codec, trames, QR, page A4), commande
  `archive export|import`, routes `/v1/exports.archive` et `/v1/imports.archive`, import d'une
  page `.html` par le chemin commun, ADR-0119.
- #986 feuille de match : `pkg/blunderdb/printsheet` (feuille A4, styles et script
  d'ajustement communs, modèle `MatchSheet`), `Database.MatchSheetHTML`, commande
  `match sheet`, route `/v1/matches.sheet`, action « Imprimer… » du panneau Match et sa
  boîte d'options. Le code de réimport est l'archive papier du match, dessinée et
  embarquée : la feuille enregistrée se réimporte comme une page d'archive.
- #987 listes de positions : `printsheet.BuildPositions` (source `ids` | `query` |
  `collection`, plafond `MaxPositions` = 3000 refusé par `storage.ErrInvalid`),
  `Database.PositionSheetHTML`, commande `sheet`, route `/v1/positions.sheet` (dans
  `heavyRoute`), mise en page `data-fill` du script (pages numérotées, recto positions /
  verso réponses par paire). Interface : « Imprimer la liste… » (menu du plateau, liste
  parcourue entière), « Imprimer… » d'une collection, « Imprimer les positions » du
  panier ; boîte « Archive papier… » du menu d'un match (export match / liste /
  bibliothèque, import d'un fichier, d'un scan ou d'une photo). Une entrée plus haute
  qu'une feuille se poursuit sur la suivante, coupée à un mot de son commentaire ; une
  entrée longue commence sur la page en cours, et la suite d'une réponse au verso occupe le
  recto suivant plutôt que de le laisser vide. L'ArkID garde sa largeur (`.de .nt`
  `flex: none`), l'analyse passe à la ligne. Import d'archive papier aussi par la liste des
  matchs vide et la commande `import_archive` (`iar`).
- Rendu des feuilles tenu par `frontend/tests/e2e/print-sheet.spec.js` (CI e2e) sur les
  fixtures que `TestPrintFixtures` écrit (`-update-fixtures` après un changement de rendu).

## À faire, dans l'ordre

Rien : #987 point 5 est fait (codes de réimport des listes, par tranches de 145 positions
posées par le script sur la page de leur première position ; amendement de l'ADR-0119).

## Décisions

- Minuscules = joueur au trait, toujours ; la position est écrite de son point de vue.
- Crawford = suffixe `c` ; money = `[$]`.
- Videau proposé (position de prise ou de passe) : `X?`, `X2?`. Ajout fait pendant #984, sans
  lequel ces positions changeaient d'empreinte ; confirmé par l'utilisateur le 2026-11-26.
- Un seul coup alternatif par ligne dans les listes de positions.
- Les blunders de l'adversaire ne sont pas exportés par défaut ; l'export le demande.
- Les commentaires s'exportent en option.

## Repères de mesure (base de 298 matchs)

- Tous les matchs tiennent sur une page A4 ; 251 coups imprimés en médiane, 756 au plus.
- Corps d'environ 8 pt pour un match en 7 ou 9 points, 6,5 pt en 11 ou 13 ; plancher 5,5 pt.
- ArkID : 29 caractères en médiane, 34 au plus.
- Listes : environ 160 positions par face avec analyse, 300 sans. Mesuré au corps 5,5 pt
  (Chromium de Playwright, fixtures de `TestPrintFixtures`) : 172 et 171 par face avec
  l'analyse sur la ligne, 345 sans.
- Codec d'archive (prototype Python) : 201 octets par match en médiane. Le codec Go n'a pas
  été mesuré sur cette base.

## Écarts connus

- L'interface ne copie que la forme compacte d'ArkID ; la développée existe en CLI et serveur.
- Le dossier surveillé relit chaque `.html` à chaque balayage.
- La dernière correction de #985 n'a pas eu de revue indépendante.
- Bestiaire hors français et anglais : à relire par un locuteur.
- Essai de pliage réel (imprimer, plier, scanner) : non fait.
- Feuille de match : l'ajustement du corps est fait par le navigateur (script) ; il n'est
  pas mesuré par un test, ni sur la base de 298 matchs. L'interface n'a pas de sélection
  multiple : le lot imprime les matchs que la liste affiche.
- Listes de positions : le code de réimport ne porte que les positions (leur ArkID), ni
  analyses ni commentaires. Il n'a pas été relu d'un vrai scan imprimé ; le test lit les
  trames embarquées et la parité, le rendu e2e contrôle la place des blocs. L'import par lot
  (glisser-déposer, dossier surveillé) écarte une liste : elle passe par l'archive papier.
- Le panier n'imprime que ses positions de la bibliothèque affichée ; celles des autres
  bibliothèques ouvertes sont ignorées.
- Le titre proposé de la liste parcourue est le nom de la collection active, sinon la
  dernière recherche : il peut se tromper quand les deux existent, et la boîte le laisse
  corriger. La mise en page des listes (paires recto verso, découpe d'une entrée) est
  faite par le script dans le navigateur ; le test e2e la contrôle sur des fixtures, pas sur
  une base réelle.
