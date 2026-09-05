# Persona 5 — Lecteur non francophone (japonais, puis allemand)

## Qui je suis (3 lignes)

Kenji, 34 ans, joueur de club à Osaka. Je lis le japonais, un anglais scolaire, pas un mot de français.
Mon navigateur est en japonais et je viens d'un tweet qui vantait « un ChessBase du backgammon ».
Je veux voir en trente secondes si ce logiciel me parle ma langue, puis vérifier l'allemand pour un ami de Düsseldorf.

## Parcours suivi (liste ordonnée des pages lues)

1. Page produit (racine du site), en anglais uniquement
2. `ja/` — accueil de la documentation
3. `ja/telecharge_install` — ダウンロードとインストール
4. `ja/guide_utilisateur` — ユーザーガイド
5. `ja/manuel` — マニュアル (premières ~400 lignes, puis panneaux Analyse / Eval)
6. `ja/faq` — よくある質問（FAQ）
7. `ja/glossaire` — 用語集
8. `ja/raccourcis` — キーボードショートカット
9. `ja/cmd_mode` — コマンド一覧
10. `ja/historique` — バージョン履歴
11. `ja/a_propos` — blunderDB について
12. Détours : `ja/cli`, `ja/mode_headless` (cités dans la barre latérale, donc lus)
13. Survol allemand : `de/index`, `de/telecharge_install`, `de/guide_utilisateur`, `de/manuel`, `de/raccourcis`, `de/cmd_mode`, `de/cli`

## Ce que j'ai trouvé en cinq minutes

Que la documentation japonaise existe, et qu'elle est bien plus qu'une traduction automatique.
La barre latérale est entièrement en japonais, sections comprises (`はじめに` / `リファレンス` / `付録`),
le manuel raisonne en japonais et non en français transposé, et le glossaire a de vraies entrées pensées
(`参照系`, `ロール前ベースライン`, `トゥーグッド`). La page produit dit « no internet connection, no external
engine to install » ; l'accueil japonais le redit correctement. Le fichier `.deb` associe même l'extension
`.dbx`. Quelqu'un a travaillé.

Mais je ne l'ai pas trouvée depuis la page produit. J'y suis arrivé en devinant.

## Où je me suis égaré

La page produit est en anglais, entièrement, et ne montre aucun choix de langue : ni drapeau, ni menu, ni
mention que la doc existe en neuf langues. Le seul bouton est « Documentation ». Je l'ai cliqué en espérant.

Arrivé sur la documentation en japonais, j'ai voulu vérifier que je n'étais pas tombé dessus par accident.
J'ai cherché un sélecteur de langue. Il n'est pas en haut, ni près du logo : il est tout en bas de la barre
latérale, **après** les vingt entrées de la table des matières, sous un titre `言語`, et il liste des codes
latins en minuscules : `fr en de / el es fi / it ja ru`. Rien n'indique laquelle est active. Sur mon
téléphone, la barre latérale est un menu escamotable : ce bloc est au bout d'un défilement de deux écrans.

Ensuite, la page d'installation m'a proposé le PDF hors-ligne :
`blunderDB-0.36.0-en.pdf`. Anglais. Trois lignes plus bas, la même page m'écrit
« ページ上のその他のファイルは […] 9 言語の PDF ドキュメント ». Il y a donc un PDF japonais quelque part,
mais la page japonaise ne me donne que l'anglais.

## Ce qui a entamé ma confiance

Le balisage cassé. En japonais, les marqueurs `**` et ` `` ` de la source n'ont pas été rendus, parce qu'ils
touchent directement des kanji. Ils s'affichent tels quels, 29 fois. Dans le glossaire, à l'entrée
« ブランダー » — l'entrée qui donne son nom au logiciel :

> `bl``（または ``blunders）コマンドは、検索の最悪の誤りを直接読み込みます。`

Je ne sais plus si la commande est `bl`, `` bl`` ``, ou `` ``blunders ``. Dans le manuel :
`マネーゲームのポジションで有効な**Jacoby**ルールと**Beaver**ルール`. Dans l'historique, une puce entière
part en morceaux, gras et code mélangés, le gras s'ouvrant après « イメージ » et se refermant vingt mots plus
loin au milieu d'une autre proposition. Aucune de ces lignes n'est cassée en allemand : l'allemand a des
espaces, le japonais n'en a pas, et personne n'a vérifié la page rendue.

Puis les raccourcis. La page `キーボードショートカット` m'apprend `CTRL-PageUp, SHIFT-J`. Le manuel, sur la
même fonction, écrit `（またはMAJ-J / MAJ-K）` et, deux lignes plus bas,
`新しい名前を入力してENTREEで確定します`. `MAJ` et `ENTREE` ne sont pas des touches : ce sont les noms
français de Shift et Entrée, laissés tels quels. J'ai cherché `MAJ` sur mon clavier.

Puis les commandes à taper. La page `コマンド一覧` me donne `t'mot1;mot2;...'` et `m'motif1,motif2,...'`.
`mot` et `motif` sont des mots français que je dois remplacer, mais rien ne me le dit. Sur la même page, le
filtre joueur, lui, a été traduit : `pl'名前'`. La version anglaise, elle, écrit `t'word1;word2;…'`. Le
japonais a été traduit à moitié.

Enfin les termes. « Position » se dit `ポジション` dans le guide et `局面` dans la liste des commandes — et
dans **le même tableau** de cette page : `現在の局面を削除します` (ligne `delete`) puis
`識別子 x のポジションを検索します` (ligne `idx`). Pour la décision de videau j'ai compté cinq formes.
Le glossaire distingue soigneusement `決定` (la question posée) de `判断` (le verdict) — puis le manuel
appelle vingt et une fois `キューブの判断` ce que le glossaire nomme `キューブの決定`. La page qui devait
fixer le vocabulaire est la seule à employer le sien.

## Ce qui manque

- Un chemin depuis la page produit vers ma langue. Un seul lien `日本語` suffirait.
- Le PDF japonais, ou au moins son lien depuis la page japonaise.
- Le terme d'origine en regard de chaque entrée du glossaire (`参照系 (referential)`). Sans lui, je ne peux
  recouper ni avec l'interface, ni avec la littérature anglophone du backgammon, ni avec XG.
- Une note disant quels libellés cités entre guillemets sont ceux de l'interface anglaise. Le manuel
  m'annonce que l'interface existe en japonais, puis me demande de cliquer sur « New Database »,
  « Matches », « Analysis », « Evaluate this position », « Open positions », `"Import Database"`. Si
  l'interface est en japonais chez moi, aucun de ces boutons n'existe sous ce nom.
- Une relecture typographique : la ponctuation alterne `:` et `：` d'une ligne à l'autre, y compris entre
  deux entrées voisines de la même barre latérale.

## Relevé systématique

### Négociation de langue

| Point | Constat |
|---|---|
| Clics jusqu'au japonais depuis la page produit | Indéterminé : aucun lien de langue sur la page produit. Un seul bouton « Documentation », sans indication de cible. |
| Choix de langue sur la page produit | Absent (voir `shots/landing.png`) : logo, 3 paragraphes anglais, 3 boutons de téléchargement, 4 liens de pied de page. |
| Emplacement du sélecteur dans la doc | Tout en bas de la barre latérale, sous la table des matières complète (`shots/accueil-ja.png`, `shots/install-ja.png`). |
| Libellé du sélecteur | `言語` — traduit. Mais les langues sont des codes latins minuscules : `fr en de / el es fi / it ja ru`. Pas de `日本語`, pas de marquage de la langue active. |
| Barre latérale, titres de sections | Traduits : `はじめに`, `リファレンス`, `付録`. |
| Champ de recherche | `Search docs` — anglais (`shots/accueil-ja.png`). |
| Navigation bas de page | `Next` / `Previous Next` — anglais, sur les 18 pages `ja/`. L'allemand affiche `Zurück Weiter`. |
| Titres d'encadrés | `注釈` et `警告` traduits ; `Tip` non traduit 15 fois (`ja/guide_utilisateur` ×9, `ja/manuel` ×6). L'allemand a `Tipp` (15 + 20 occurrences). |
| Pied de page | `© Copyright 2024-2026, Kevin UNGER <blunderdb@proton.me>.` — non traduit (identique en allemand). |
| PDF hors-ligne | `ja/telecharge_install` pointe `blunderDB-0.36.0-en.pdf` alors que la même page annonce `9 言語の PDF ドキュメント`. |

### Français résiduel (page › extrait)

| Page | Extrait |
|---|---|
| `ja/manuel:212` | `CTRL-PageUp / CTRL-PageDown（またはMAJ-J / MAJ-K）を押して` — `MAJ` = Shift en français ; `ja/raccourcis:158` écrit `SHIFT-J`. |
| `ja/manuel:215` | `新しい名前を入力してENTREEで確定します。` — `ENTREE` = Entrée. |
| `ja/manuel:1340` | `blunderdb info --db fichier.db が出所と署名の状態を表示します` — `fichier.db`. |
| `ja/cmd_mode:294` | `t'mot1;mot2;...'` — `mot` non traduit (EN : `t'word1;word2;…'`). |
| `ja/cmd_mode:300` | `m'motif1,motif2,...'` — `motif` non traduit (EN : `pattern1`). |
| `ja/telecharge_install` | `yay -S blunderdb-bin      # ou : paru -S blunderdb-bin` — `ou`. |
| `ja/cli` | **39 lignes** de commentaires de blocs de code en français, ex. `# Décisions de videau, 30 pips de retard, 50 millipoints d'erreur`, `# Rechercher les positions que vous avez ajoutées vous-même`, `# Sortie JSON limitée à 10 résultats`, `# Export filigrané et protégé par mot de passe (fichier .dbx)`. |
| `ja/cli:99` | `./blunderdb <commande> [options]` — méta-variable française. |
| `ja/cli:547, 592, 708, 863` | `<fichier>`, `<fichier.db>` — méta-variables françaises. |
| `ja/mode_headless` | **15 lignes** de français, dont `# dans le volume monté, une fois`, `# Authentifié en tant qu'« alice », qui est mappée au tenant 1 : le démon`, `echo "démon disponible"`. |
| `ja/mode_headless:108,110,132,136,623,633` | Méta-variables françaises dans les tableaux d'options : `--dsn <chaîne>`, `--addr <hôte:port>`, `<fichier>`, `<répertoire>`, `<famille>.<méthode>`. |
| `ja/cli:473` | `videau` employé 3 fois en clair (`# Décisions de videau`, `# Rechercher les décisions de videau`, `# Extraire les erreurs de videau`). |

> Le même défaut existe en **anglais** (`en/cli` : 34 lignes françaises, `en/mode_headless` : 15) et en
> **allemand** (`de/cli` : 41, `de/mode_headless` : 15). Les commentaires des blocs de code de `cli` et
> `mode_headless` ne sont traduits dans aucune langue.

### Balisage cassé (page › extrait)

| Page | Extrait |
|---|---|
| `ja/glossaire:148` | ``誤り（最善の選択とのエクイティ差）が閾値を超える手またはキューブの決定。bl``（または ``blunders）コマンド`` |
| `ja/guide_utilisateur:97` | ``どこからでも、bl``（または ``blunders、SPACE でコマンドラインを開く）コマンドを実行すると`` |
| `ja/manuel:417` | `マネーゲームのポジションで有効な**Jacoby**ルールと**Beaver**ルールも、キューブ判断テーブルの下に` |
| `ja/manuel:953` | `そのポジションで有効な**Jacoby**ルールと**Beaver**ルールが、キューブテーブルの下に` |
| `ja/historique:119-120` | ``サーバーモードは Docker イメージ**（``ghcr.io/kevung/blunderdb-serve``、amd64 と arm64）として公開され、``maintenance.vacuum`` ルートを獲得し、サーバーの発行者アイデンティティ（``--identity-dir``）で**エクスポートに透かしを入れます。`` |
| `ja/historique` | `**` littéraux également en 123, 129, 149, 151, 157, 162-164, 211, 222 (12 lignes au total). |
| `ja/cli:481,499,505` | ``オプション（``stats`` タイプのみ）：`` ; ``出力形式：text、json、csv``（既定：``text）。`` |
| `ja/cli:784` | ``並列処理（``--jobs``）。`` |
| `ja/mode_headless:224,545-553` | ``BLUNDERDB_POSTGRES_MAX_CONNS``（デフォルト 50）``, ``（``1h）``, ``（``30s）``… 8 lignes. |
| `ja/stats_parity:147` | ``参照：gnubg/analysis.c:199–269``（``LuckNormal / LuckFirst`` |
| **Total** | **29 lignes** avec `` `` `` ou `**` littéraux dans `ja/`. Contre **4** en `de/`. |

Typographie japonaise :

| Page | Extrait |
|---|---|
| `ja/manuel:916` | `バッジのバンドにある控えめな?ボタンは` (`?` demi-chasse collé) ; `gammonNet へ導きます；完全な帰属表示は` (`；` en milieu de phrase, calque du `;` français). |
| `ja/manuel:224-226` | `Bearoff— Bearoff パネルが使う…` et `gammonNet— 内蔵の評価エンジンの設定。以下で説明します；` — cadratin collé, alors que la puce voisine écrit `インターフェース — 言語、表示倍率`. |
| `ja/index` (barre latérale) | `付録: フィルターの高度な使い方` (deux-points ASCII) juste au-dessus de `付録：統計モデル` (deux-points pleine chasse). Même liste, deux conventions. |
| `ja/telecharge_install` | 18 `：` contre 6 `:` sur la même page : `利用可能です：` vs `置き換えてください:` vs `自動的に更新されます:`. |
| `ja/telecharge_install` | Latin collé : `amd64とx86_64のファイルは`, `迷ったときはuname -mが答えます`, `blunderdb-binは両方に対応し` — alors que la ligne précédente écrit `blunderdb-bin パッケージは` avec espaces. |
| `ja/guide_utilisateur:499-501` | `パフォーマンス指標（PRおよびMWCコスト）`, `下部パネルのStatsタブ` — collés ; la même page écrit `Eval パネル`, `マッチごとの PR と MWC コスト` avec espaces. |
| `ja/guide_utilisateur:545` | `XG、GNUbg、またはBGBlitzによって分析されたポジションがblunderDBにインポートされている場合` — section entière sans espaces. |
| `ja/a_propos:92-110` | `主なもの:` (ASCII) ; `パートナーのAnne-Claireと愛する娘のPerrineに`, `フォントNunitoとNoto Sans JP` — collés. |
| `ja/faq` | `"Import Database" 機能を使用してください` — guillemets droits ASCII, alors que le reste de la page utilise `「 」`. |
| `ja/raccourcis:139-153` | `Ankiパネル（間隔反復）` collé vs `Eval パネル` espacé, dans le même tableau. |

### Variantes terminologiques (terme › variantes › pages)

| Terme | Variantes relevées (occurrences) | Pages |
|---|---|---|
| position | `ポジション` (94/124/38/24/19/47) vs `局面` (30 manuel, **34 cmd_mode contre 2 `ポジション`**, 11 cli, 6 historique) vs `盤面` (contexte plateau) | mélange **dans un même tableau** : `cmd_mode:102` `現在の局面を削除します` / `cmd_mode:310` `識別子 x のポジションを検索します` ; `manuel:366` `ツールバーの局面へ移動ボタン` au milieu d'un paragraphe en `ポジション` |
| décision de videau | `キューブの判断` (21) · `キューブの決断` (10) · `キューブの決定` (6) · `キューブ決定` (3) · `キューブアクション` (4) | manuel / guide_utilisateur+faq / glossaire+historique / guide+faq+historique / manuel+cli. Le glossaire **définit** `決定` = la question et `判断` = le verdict ; le manuel appelle `キューブの判断テーブル` la table des trois options, ce qui contredit sa propre définition |
| videau | `キューブ` (partout) vs `ダブリングキューブ` (`cli:473-474`) vs `videau` en français (`cli` ×3) | cli |
| équité | `エクイティ` (stable, 54 occ.) — seul terme cohérent du corpus | toutes |
| blunder | `ブランダー` (15) · `ミス` (7) · `誤り` (18) · `エラー` (26) · `誤差` (13) · `悪手`/`大悪手` (3) | `cmd_mode:120` `最悪のミス` vs `guide:97` `最も高くついた誤り` vs `glossaire` `ブランダー` défini par `誤り` puis `PR` défini par `平均エラーコスト` — quatre mots sur une seule page |
| coup | `手` (67 manuel) vs `ムーブ` (9 guide) vs `チェッカーの手` vs `チェッカームーブ` | `guide:545-550` : `上位5つのムーブ`, `関連するチェッカームーブ`, `ムーブのエクイティエラー` — dans une page dont le reste dit `最善手` |
| match | `マッチ` (stable, ~250 occ.) | toutes — cohérent |
| tournoi | `トーナメント` (~110) vs `大会` (5) | `manuel:587,785,828`, `historique:177`, `cli:501` (`たとえば大会の開催日`) |
| collection | `コレクション` (stable) | toutes — cohérent |
| PR | `PR（Performance Rate）` glossé une fois, puis `PR` partout | cohérent |

### Allemand

- **Chaînes de thème traduites**, contrairement au japonais : `Zurück Weiter`, `Sprachen`, `Tipp`,
  `Bemerkung`, `Warnung`. Le japonais garde `Next`, `Previous`, `Tip`, `Search docs`.
- **Balisage** nettement plus sain : 4 lignes avec ` `` ` littéraux (`de/cli:493,511,810`,
  `de/mode_headless:237`) contre 29 en japonais, et **aucun** `**` visible. L'espace germanique protège la
  syntaxe RST là où le japonais la casse.
- **Français résiduel identique** : `de/cli` 41 lignes, `de/mode_headless` 15
  (`# Servir une base SQLite locale sur le port 8080`, `echo "démon disponible"`,
  `# Export filigrané et protégé par mot de passe (fichier .dbx)`, `./blunderdb <commande> [options]`).
- **Option corrompue par la typographie** : `de/cli:810` écrit `Parallelität (``–jobs``)` avec un
  demi-cadratin `–` au lieu de `--`. Copiée-collée, la commande échoue. Même défaut en
  `fr/cli:803`, `en/cli:789`, `fr/manuel:1353`, `en/manuel:1294` (`–match`), `de/historique:501` (`–type`).
- **Libellés d'interface anglais** aussi : `de/guide_utilisateur:545` `dann „Evaluate this position“`,
  `de/manuel:786,797` `Open positions — lädt alle Stellungen…`.
- **Placeholders** : le tableau de `de/cmd_mode` ne comporte pas les lignes `t'…'` / `m'…'` que le japonais
  a laissées en français — à vérifier, l'allemand semble donc soit traduit soit absent sur ce point.
- Terminologie allemande stable : `Stellung`, `Cube-Entscheidung`, `Fehler`, `Zug`, `Match`, `Turnier`.

## Constats

| # | Constat | Page › section | Gravité | Proposition |
|---|---|---|---|---|
| 1 | La page produit est en anglais seul et n'offre aucun choix de langue ; rien n'indique que la doc existe en neuf langues. Un lecteur japonais ne peut pas savoir que `/ja/` existe. | Racine du site › en-tête / pied de page | bloquant | Ajouter au pied de la page produit la même grille de langues que la doc, en **noms natifs** (`日本語`, `Deutsch`…), et une ligne « Documentation available in 9 languages ». |
| 2 | Le sélecteur de langue de la doc est en bas de la barre latérale, après toute la table des matières, en codes latins minuscules (`fr en de / el es fi / it ja ru`), sans marquage de la langue active. | Toutes pages `ja/` et `de/` › barre latérale | bloquant | Remonter le bloc `言語` juste sous le logo, écrire les langues en noms natifs, mettre la langue courante en gras/non cliquable. |
| 3 | La page d'installation japonaise propose le PDF **anglais** (`blunderDB-0.36.0-en.pdf`) tout en annonçant deux paragraphes plus bas « 9 言語の PDF ドキュメント ». | `ja/telecharge_install` › ダウンロードとインストール | bloquant | Rendre le suffixe de langue du lien PDF dépendant de la langue de build (`-ja.pdf` sur `/ja/`). |
| 4 | 29 lignes japonaises affichent des marqueurs RST littéraux (` `` `, `**`) parce qu'ils touchent des kanji : la commande phare `bl` / `blunders` est illisible dans le glossaire ET dans le guide. | `ja/glossaire:148`, `ja/guide_utilisateur:97`, `ja/manuel:417,953`, `ja/historique` ×12, `ja/cli` ×4, `ja/mode_headless` ×8 | bloquant | Insérer l'échappement d'espace `\ ` aux deux bornes CJK des `` `` `` et `**` (procédé déjà employé ailleurs dans les catalogues japonais), puis relire la page **rendue**, pas le `.po`. |
| 5 | Les commentaires des blocs de code de `cli` et `mode_headless` sont en français dans **toutes** les langues (ja 54 lignes, de 56, en 49). Le lecteur doit deviner ce que fait chaque exemple. | `ja/cli`, `ja/mode_headless` › tous les blocs `.. code-block::` | bloquant | Faire entrer les commentaires de blocs de code dans le périmètre de traduction, ou les réduire à un mot-clé neutre (`# import`, `# export`). |
| 6 | Deux pages de référence donnent deux noms différents au même modificateur, dont un en français : `SHIFT-J` (raccourcis) contre `MAJ-J` et `ENTREE` (manuel). | `ja/raccourcis:158` vs `ja/manuel:212,215` | bloquant | Aligner le manuel japonais sur `SHIFT` / `ENTER` (convention déjà retenue par `ja/raccourcis`). |
| 7 | La syntaxe de deux filtres à taper contient des mots français non signalés : `t'mot1;mot2;...'` et `m'motif1,motif2,...'`, alors que la même page a traduit `pl'名前'` et que l'anglais dit `word1` / `pattern1`. | `ja/cmd_mode` › 検索フィルター | bloquant | Traduire les méta-variables (`t'語1;語2;...'`, `m'パターン1,パターン2,...'`), comme l'a fait `pl'名前'`. |
| 8 | « Position » se dit `ポジション` ou `局面` selon la page, et selon la **ligne** à l'intérieur d'un même tableau. | `ja/cmd_mode:102` vs `ja/cmd_mode:310` ; `ja/manuel:366` | gênant | Choisir `ポジション` (majoritaire : 400+ occurrences contre ~85) et passer les pages `cmd_mode`, `cli`, `historique` à la moulinette. |
| 9 | « Décision de videau » a cinq formes, et le manuel emploie `キューブの判断` pour ce que le glossaire définit explicitement comme `キューブの決定` (`判断` y désigne le verdict). | `ja/manuel` (21) vs `ja/glossaire` › 決定 / 判断 | gênant | Fixer `キューブの決定` = la décision, `判定` = le verdict, et appliquer la distinction du glossaire dans le manuel. |
| 10 | « Erreur / blunder » se dit `エラー`, `誤差`, `誤り`, `ミス`, `悪手` — quatre de ces mots cohabitent dans le seul glossaire. | `ja/glossaire` › ブランダー / PR ; `ja/cmd_mode:120` | gênant | Un mot par notion : `エラー` pour la quantité chiffrée, `ブランダー` pour le seuil dépassé ; supprimer `ミス` et `誤差`. |
| 11 | `Tip` reste en anglais 15 fois alors que `注釈` et `警告` sont traduits ; l'allemand a `Tipp`. | `ja/guide_utilisateur` ×9, `ja/manuel` ×6 | gênant | Compléter le catalogue de locale japonais pour l'admonition `tip`. |
| 12 | La navigation bas de page (`Next`, `Previous`) et le champ de recherche (`Search docs`) restent en anglais sur les 18 pages japonaises ; l'allemand affiche `Zurück Weiter`. | Toutes pages `ja/` › pied de page et barre latérale | gênant | Même cause que #11 : la locale `ja` du thème est incomplète, la locale `de` ne l'est pas. |
| 13 | Le manuel japonais annonce que l'interface existe en japonais, puis demande de cliquer sur des boutons nommés en anglais : « New Database », « Matches », « Analysis », « Comments », « Stats », « Evaluate this position », « Open positions », `"Import Database"`. | `ja/guide_utilisateur:79,97,524`, `ja/manuel:720,730`, `ja/faq` | gênant | Donner le libellé japonais de l'interface, avec l'anglais entre parenthèses la première fois : `「新規データベース」(New Database)`. |
| 14 | Ponctuation mixte `:` / `：` jusque dans deux entrées voisines de la même barre latérale (`付録: データベーススキーマ` / `付録：統計モデル`). 18 contre 6 sur la seule page d'installation. | `ja/index` (barre latérale), `ja/telecharge_install`, `ja/a_propos:92,108,110` | gênant | Passe de normalisation : `：` partout en prose japonaise, `:` réservé au code. |
| 15 | L'espacement latin/japonais change de convention en cours de page : `Eval パネル` et `Statsパネル`, `PR と MWC コスト` et `PRおよびMWCコスト`, `blunderdb-bin パッケージ` et `blunderdb-binは`. | `ja/guide_utilisateur:499-501,545`, `ja/telecharge_install`, `ja/a_propos:100-110`, `ja/raccourcis:139` | gênant | Fixer une règle (espace fine autour du latin) et l'appliquer ; la section « XGからインポート… » du guide est à reprendre entièrement. |
| 16 | Ponctuation occidentale calquée : `；` en milieu de phrase, `?` demi-chasse collé au japonais, cadratin sans espace (`Bearoff—`), et puces terminées par `、` à la française. | `ja/manuel:916,224-226`, `ja/faq` › puces, `ja/a_propos` | mineur | Remplacer `；` par `。` ou une virgule, `?` par `？`, ajouter l'espace avant les cadratins, terminer chaque puce par `。`. |
| 17 | Le glossaire ne donne aucun terme d'origine : `参照系`, `レジーム`, `判断`, `決定`, `ロール前ベースライン` ne sont recoupables ni avec l'interface, ni avec la littérature anglophone, ni avec XG. | `ja/glossaire` › toutes entrées sauf EPC/PR/MWC/Snowie | mineur | Ajouter le terme anglais entre parenthèses après chaque vedette japonaise. |
| 18 | `ja/historique` 0.35.0 annonce « バイナリは不要な日本語フォント5.5MB を失いました » (« le binaire a perdu 5,5 Mo de police japonaise inutile ») pendant que `ja/a_propos` crédite `Noto Sans JP` comme police embarquée. Lu en japonais, c'est au mieux déroutant. | `ja/historique:129` vs `ja/a_propos:110` | mineur | Reformuler : « la police japonaise a été réduite à son sous-ensemble utile (−5,5 Mo) ». |
| 19 | Les guillemets varient : `「 」`, `„ “`, `" "` droits ASCII, `« »`. Le FAQ japonais utilise `"Import Database"` en guillemets droits au milieu d'une page en `「 」`. | `ja/faq` › 複数のデータベースを統合するには | mineur | `「 」` partout en japonais. |
| 20 | Un drapeau `--jobs` est rendu `–jobs` (demi-cadratin) : la commande copiée-collée échoue. Défaut absent du japonais, présent en de/fr/en. | `de/cli:810`, `en/cli:789`, `fr/cli:803`, `fr/manuel:1353`, `de/historique:501` | mineur | Mettre ces options en littéral `` `` `` pour couper la substitution typographique (`smartquotes`). |
