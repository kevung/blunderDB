# L'archive papier code un match par code QR, sans dépendre du moteur ; elle se réimporte par l'import ordinaire

Statut : acceptée. Voir aussi : ADR-0007 (rien chez le destinataire), ADR-0045 (la
transcription), ADR-0082 (repères vidéo), ADR-0001 et ADR-0028 (identité des positions).

## Contexte

L'utilisateur veut garder une bibliothèque de matchs sur papier, en peu de place, et la
réimporter d'un scan ou d'une photo. Un prototype (hors dépôt) a mesuré sur 298 matchs réels
un aller-retour exact à 201 octets par match en médiane (0,80 octet par coup), contre 1 089
avec zlib sur le texte : un match en 7 points tient dans un code QR de version 10 à 12. Une
archive papier vit des années ; elle ne doit pas cesser d'être lisible parce que le moteur,
ses poids ou son générateur de coups ont changé.

## Décision

1. **Un match s'archive comme son document de transcription** (`transcript.Document` : en-tête,
   Actions, temps de décision, repères vidéo), plus le commentaire du match et, par Action, les
   commentaires et la marque d'étude de la position jouée. Les analyses ne sont pas archivées :
   elles se recalculent. Le document est lu du match stocké comme le rouvre un brouillon
   (`.mat` puis `FromMAT`, repères remis).
2. **Le codec est un codeur arithmétique entier** (`pkg/blunderdb/arkqr`, registres de 48 bits,
   produits sur 128) qui pilote trois modèles : la loi exacte des dés ; le rang du coup joué
   parmi les plateaux que laissent les coups légaux, sous un softmax à poids entiers (25
   caractéristiques, régimes course et contact) ; des tables statiques et des bits adaptatifs
   pour le genre d'Action, le camp, la résignation. Aucun flottant n'entre dans le codec ; la
   table d'exponentielles est écrite en clair.
3. **Le format ne dépend pas du moteur** : le paquet a son propre générateur de coups légaux,
   et la liste des candidats est ordonnée par le plateau qu'ils laissent. Un test le recoupe
   avec `domain.LegalMoves` sur tout le corpus. Le premier octet est `FormatVersion` ; poids et
   tables sont figés pour elle : en changer un est une nouvelle version, et le test de taille
   du corpus le signale.
4. **Ce que les modèles ne prévoient pas s'écrit en clair** — coup non noté, plateau saisi à la
   main, dés impossibles, coup incohérent, plateau que la relecture atteint autrement — de sorte
   que tout document que `Replay` lit fait l'aller-retour. Un coup légal est réécrit avec les
   étapes que trouve le générateur du paquet pour le plateau laissé : deux graphies d'un même
   coup sont une seule archive, et les positions (donc leurs hachages Zobrist) sont inchangées.
5. **Un code QR porte une trame** : un match quand il tient (au plus 1 000 octets, sinon il est
   coupé), l'identifiant de l'archive, le rang, le nombre de trames et un CRC-32, en Base45
   (alphabet alphanumérique des QR, que tout lecteur rend intact), correction d'erreur M. Un
   code de parité (XOR) suit chaque groupe de cinq et reconstruit l'un d'eux.
6. **La page est un HTML autonome de feuilles A4** : quatre quarts, une gouttière blanche de
   12 mm sur les deux lignes de pliure (aucun code à cheval sur un pli), une ligne lisible sous
   chaque code. Les trames y sont aussi inscrites, en base64 : la page se réimporte sans
   lecture optique, par `archive import` ou par l'`import` ordinaire d'un fichier.
7. **Le réimport passe par le chemin d'import des matchs** : le graphe de la transcription
   (`ingest.ArchiveGraph` sur `ingest.TranscriptGraph`, les hachages de `TranscriptHashes`),
   écrit par `ingest.WriteMatch`, donc dans le lot d'import et avec le dédoublonnage — un match
   déjà en base, de la même archive ou d'un autre format (hachage canonique), est un doublon.
   L'archive vit sur le contrat de stockage (`ingest/archive.go`) : la base (CLI, interface),
   le démon (`/v1/exports.archive`, `/v1/imports.archive`) et l'import par lot (une page
   `.html` dans `mapFile`, donc le glisser-déposer et le dossier surveillé) la lisent et
   l'écrivent de la même façon. Un `.html` qui n'est pas une page d'archive est écarté à la
   collecte (`ingest.IsImportableFile`, qui regarde le contenu) : un dossier où des exports
   HTML voisinent avec les matchs n'est pas une liste de fichiers refusés. Les images de scan
   ne passent que par `archive import` et `/v1/imports.archive`.
8. **Règles d'export et ADR-0007** : la vidéo ne sort que par son URL
   (`issuance.CarriedVideoSource`) ; aucune métadonnée de la bibliothèque n'est copiée : l'archive
   porte des matchs, pas une base, et rien n'y entre par exclusion (la liste
   `issuance.CarriedMetadataKeys` ne concerne que l'export d'une base) ; la ligne d'origine est
   écrite par le producteur à l'export, en pied de feuille. L'import n'écrit rien sur l'archive,
   sa provenance ni son lecteur.
9. **L'en-tête porte ce qu'un `.mat` perd** : règles Jacoby et Beaver d'une session money, date
   et heure, classements Elo et expérience, version du moteur, transcripteur. Le tournoi n'est
   pas porté : l'import le redéduit de l'évènement, comme pour tout format qui le nomme.

## Conséquences

- Un match en 7 points coûte de 200 à 300 octets ; une feuille porte une vingtaine de matchs.
- Les commentaires archivés reviennent d'origine inconnue : l'archive ne dit pas qui les a
  écrits, et ils ne sont pas signés (ADR-0099).
- L'entrée d'interface suit avec les feuilles imprimables. Le démon expose l'archive sans
  chemin de fichier : `/v1/exports.archive` rend la page, `/v1/imports.archive` prend une page
  ou une image téléversée.

## Amendement — le code de réimport d'une liste de positions

Une liste de positions imprimée (`printsheet.BuildPositions`, option `reimportCode`) porte
ses propres codes, lus par le même chemin que l'archive.

1. **Un format de trame distinct** : une trame de liste s'ouvre sur l'octet `0xA8` (celles des
   matchs gardent `0xA7`, et leur identifiant d'archive ne change pas), avec la même
   disposition, le même découpage en parties, la même parité et le même CRC (`Frame.Kind`,
   `KindPositions`). Un lecteur de matchs ne prend jamais un code de liste pour le sien ;
   `Assemble` rend les matchs et les tranches à part.
2. **Une position, c'est son ArkID** (ADR-0118) : score (money, ou les deux scores Away sur 7
   bits et la marque Crawford), dés, jeton du videau et son exposant, puis les cases de chaque
   camp en unaire ; 109 bits au plus, 12,6 octets en moyenne. Le décodage réécrit l'ArkID
   canonique et le confie à `domain.DecodeArkID` : le code dit exactement ce que la ligne
   imprimée dit, et l'empreinte de Zobrist est la sienne. Ni analyse, ni commentaire, ni
   provenance.
3. **Des tranches fixes, pas de pagination en Go** : la mise en page se décide dans le
   navigateur, donc Go code la liste par tranches de `arkqr.Tranche` (145) positions, chacune
   une archive autonome — deux codes de données et leur parité, que la borne de 109 bits
   garantit. Le rendu marque chaque bloc du numéro de sa première position (`data-from`) ; le
   script de mise en page le pose sur la page recto qui porte cette ligne, et le retire avec
   elle quand elle passe à la page suivante. Une page scannée seule se relit donc tranche par
   tranche, un code perdu rebâti par la parité ; le fichier enregistré embarque toutes les
   trames, comme une page d'archive.
4. **Le réimport est celui d'une position apportée seule** : `storage.SaveIndividual` (que
   `Database.SaveIndividualPosition` et le panier utilisent aussi) lève le drapeau de
   provenance et dédoublonne sur l'empreinte ; une position déjà en base est comptée, pas
   réécrite. La base (CLI `archive import`, interface), le démon (`/v1/imports.archive`,
   champs `list_positions` et `list_positions_known`) suivent le même chemin. L'import par lot
   (`mapFile`) ne porte que des matchs : une liste y est écartée avec son motif.
