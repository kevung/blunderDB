// GENERATED FILE — do not edit by hand, and do not translate it here.
//
// Produced by `go run ./cmd/help-gen` (make help) from:
//   - doc/source/raccourcis.rst  → the "shortcuts" tab
//   - doc/source/cmd_mode.rst    → the "commands" tab
//   - doc/source/locale/<lang>/LC_MESSAGES/*.po for the eight translations
//   - frontend/src/i18n/help/prose/<lang>.html → the "manual" and "about" tabs
//
// Fix the documentation (and its .po catalogues), or the prose fragment, then
// run `make help`. TestHelpBundlesAreCurrent fails if this file is stale.
export default {
    manual: `
<h3>はじめに</h3>
<p>
    blunderDB はバックギャモンのポジションデータベースを作成するためのソフトウェアです。主な強みは、プレイヤーが（オンラインや
    トーナメントで）遭遇したポジションを一か所に集約し、さまざまな自由に組み合わせ可能なフィルターで絞り込みながら、それらのポジションを再学習できることです。blunderDB は参照ポジションの
    カタログを作成するためにも使えます。
</p>
<p>ポジションは .db ファイルで表されるデータベースに保存されます。</p>

<h3>主な操作</h3>
<p>blunderDB で可能な主な操作は次のとおりです:</p>
<ul>
    <li>新しいポジションを追加する、</li>
    <li>既存のポジションを変更する、</li>
    <li>ボードを PNG 画像としてクリップボードにコピーする（<strong>Ctrl+X</strong>）、またはボードを分析とともにコピーする（<strong>Ctrl+X, Ctrl+X</strong>）、</li>
    <li>既存のポジションを削除する、</li>
    <li>1 つ以上のポジションを検索する、</li>
    <li>さまざまなソース（XG、GNUbg、BGBlitz、Jellyfish）からマッチをインポートする（XG ファイルからのコメントを含む）、</li>
    <li>インポートしたマッチの手順を閲覧する、</li>
    <li>ポジションをコレクションに整理する、</li>
    <li>マッチをトーナメントに整理する、</li>
    <li>ターミナルから、内蔵の gammonNet 評価エンジンを使って未解析のポジションを一括解析する（blunderDB の <strong>analyze</strong> コマンド）。</li>
</ul>
<p>ユーザーは自由にポジションにタグを付け、コメントで注釈を付けることができます。</p>

<h3>GUI の説明</h3>
<p>blunderDB の GUI は上から下へ次のように構成されています:</p>
<ul>
    <li>[上部] ツールバー。データベースに対して実行できる主な操作をまとめています、</li>
    <li>[中央] メイン表示エリア。バックギャモンのポジションを表示または編集できます、</li>
    <li>[下部] ステータスバー。コマンドラインを内蔵し、現在のポジションに関するさまざまな情報を表示します。</li>
</ul>
<p>次の用途でパネルを表示できます:</p>
<ul>
    <li>現在のポジションに関連する分析データを表示する（XG、GNUbg、または BGBlitz から）、</li>
    <li>コメントを表示、追加、または変更する、</li>
    <li>インポートしたマッチを閲覧し、その手順をたどる（マッチパネル）、</li>
    <li>ポジションのコレクションを管理する（コレクションパネル）、</li>
    <li>間隔反復でポジションを学習する（Anki パネル）、</li>
    <li>トーナメントを管理する（トーナメントパネル）、</li>
    <li>パフォーマンス統計を表示する（統計パネル）、</li>
    <li>ベアオフポジションの EPC 値を計算する（評価パネル）、</li>
    <li>保存した検索フィルターを閲覧する（フィルターライブラリパネル）、</li>
    <li>検索履歴を閲覧する（検索履歴パネル）。</li>
</ul>
<p>メイン表示エリアでは次の情報がユーザーに提供されます:</p>
<ul>
    <li>バックギャモンのポジションを表示または編集するためのボード、</li>
    <li>キューブのレベルと所有者、</li>
    <li>各プレイヤーのレースカウント、</li>
    <li>各プレイヤーのスコア、</li>
    <li>プレイすべきダイス。ダイスに値が表示されていない場合、ダイスの位置はどちらのプレイヤーの手番かを示し、そのポジションがキューブの決定であることを示します。</li>
</ul>
<p>ステータスバーは左から右へ次を表示します:</p>
<ul>
    <li>コマンドライン（<strong>Space</strong> を押して開く）、</li>
    <li>直近に実行された操作に関する情報メッセージ、</li>
    <li>現在のポジションのインデックスと、それに続くポジションの総数（マッチを閲覧中は手番/ゲーム情報）。</li>
</ul>
<p>ユーザー検索の結果として得られたポジションの場合、ステータスバーに表示されるポジション数は絞り込まれたポジションの数に対応します。</p>

<h3>ポジションの閲覧</h3>
<p>デフォルトでは、blunderDB で次のことができます:</p>
<ul>
    <li>現在のライブラリ内のさまざまなポジションをスクロールする、</li>
    <li>ポジションに関連する分析情報を表示する、</li>
    <li>ポジションのコメントを表示、追加、変更する。</li>
</ul>

<h3>ポジションの編集</h3>
<p>
    <strong>Tab</strong> キーを押すと検索パネルが開き、ボード上でポジションを編集してデータベースに追加したり、検索するためのポジション構造を定義したりできます。
    チェッカーの配置、キューブ、スコア、手番はマウスで変更できます。
</p>

<h3>コマンドライン</h3>
<p>
    ステータスバーに統合されたコマンドラインでは、blunderDB のすべての機能を実行できます: データベース操作、ポジションのナビゲーション、分析や コメントの表示、フィルターによるポジション検索など...
    インターフェースに慣れたら、徐々にコマンドラインを使うことをお勧めします。コマンドラインは、特にポジション検索機能において、blunderDB を 強力かつスムーズに使えるようにします。
</p>
<p>
    コマンドラインを開くには、<strong>Space</strong> キーを押します。ステータスバーにプロンプトが表示されます。コマンドを入力して <strong>Enter</strong> を押すと実行されます。
    <strong>Escape</strong>
    を押すとキャンセルされます。
</p>
<p>blunderDB はユーザーが送ったクエリが有効であれば実行し、必要に応じて直ちにデータベースの状態を変更します。ユーザーによる明示的な保存操作は 必要ありません。</p>
<p>
    以前に絞り込んだポジションの中でさらに検索を絞り込むには、<strong>ss</strong> コマンドの後にフィルターを続けます（例: <strong>ss nc</strong>）。これは現在表示されている
    ポジションだけに検索を限定し、結果を段階的に絞り込めるようにします。検索パネル（<strong>Ctrl+F</strong>）にも「現在の結果内を検索」というチェックボックスがあり、 同じ機能を提供します。
</p>

<h3>Eval パネル</h3>
<p>
    <strong>Eval</strong> パネルは、盤上にあるどんなポジションでも評価します。勝率・ギャモン率・バックギャモン率、equity、順位づけされた候補手、そしてそのポジションが求めるただ一つの判断 —
    手を選ぶか、ダブルするか。計算は内蔵の gammonNet が行い、eXtreme Gammon も GNU Backgammon も必要ありません。
</p>
<p>
    開くには <strong>Ctrl+E</strong> を押すか、下部パネルの Eval タブをクリックするか、コマンドラインで <strong>epc</strong> と入力します。盤はベアオフの標準配置（15
    チェッカー）で開きます。データベースからポジションを送った場合はそのポジションです。チェッカーはマウスで自由に追加・削除でき、評価はその都度追随します。
</p>
<p>ベアオフのポジションでは、パネルは<strong>専門化</strong>します。プレイヤーごとの second テーブルが、GNUbg の片側 6 ポイントベアオフデータベースから求めた EPC（Effective Pip Count）を示します —</p>
<ul>
    <li><strong>EPC</strong>: 全チェッカーをベアオフするのに必要な平均ピップ数、</li>
    <li><strong>Pip Count</strong>: 素のピップカウント、</li>
    <li><strong>Wastage</strong>: EPC とピップカウントの差、</li>
    <li><strong>Avg Rolls</strong>: 全チェッカーをベアオフするまでの平均ロール数、</li>
    <li><strong>Std Dev</strong>: そのロール数の標準偏差。</li>
</ul>
<p>両プレイヤーが自陣にチェッカーを持つときは、比較セクションが EPC とピップカウントの差を表示します。</p>
<p>
    純粋なレースでは、さらに別のテーブルが両プレイヤーの勝率を示し、ポジションが両側データベースの範囲内にあるとき（内蔵はプレイヤーあたり 6 チェッカーまで、設定の Bearoff
    タブからダウンロードできる拡張版は 11 まで）、正確な money equity
    と最善のキューブ判断も示します。その範囲外では勝率は推定値となり（誤差幅つきの「推定」バッジ）、判断は表示されません。手番のプレイヤーはプレイヤーのオフ／スコアの矩形をクリックして、キューブの位置は盤上のキューブをクリックして変更します。
</p>
<p>
    <strong>チャレンジ</strong>チェックボックスは、ポジションを変更するたびに結果を隠します。領域をクリックすると表示されます。equity・EPC・キューブ判断を答え合わせの前に見積もる練習に向いています。
</p>
<p>Eval パネルを閉じるには、もう一度 <strong>Ctrl+E</strong> を押すか、別のタブに切り替えます。</p>

<h3>マッチのナビゲーション</h3>
<p>
    blunderDB ではインポートしたマッチの手順を閲覧できます。<strong>Ctrl+Tab</strong> でマッチパネルを開き、マッチをダブルクリック（または <strong>Enter</strong> を押す）して
    そのポジションを読み込みます。
</p>
<p>
    マッチを閲覧しているとき、最後に訪れたポジションは自動的に保存・復元されます。<strong>Left</strong>/<strong>Right</strong> キーでポジション間を移動し、 <strong>PageUp</strong>/<strong
        >PageDown</strong
    >
    でゲーム間をジャンプします。
</p>
<p>分析パネル（<strong>Ctrl+L</strong>）は各手の分析を表示し、実際にプレイされた手がハイライトされます。<strong>d</strong> を押すとチェッカー分析とキューブ分析を切り替えられます。</p>

<h3>コレクション</h3>
<p>
    コレクションを使うとポジションをカスタムグループに整理できます。<strong>Ctrl+B</strong> でコレクションパネルを開き、コレクションをダブルクリックしてそのポジションを閲覧します。
    コレクションとその中のポジションはドラッグ＆ドロップで並べ替えられます。
</p>

<h3>Anki（間隔反復）</h3>
<p>Anki パネル（<strong>Ctrl+K</strong>）は、FSRS アルゴリズムを使ってバックギャモンのポジションを学習するための間隔反復を提供します。</p>
<p><strong>デッキの作成:</strong> <em>New Deck</em> をクリックして、コレクションまたは現在の検索結果からデッキを作成します。検索ベースのデッキは Anki タブがアクティブになると自動的に同期されます。</p>
<p>
    <strong>復習:</strong> デッキを選択して <em>Study</em> をクリック（またはデッキをダブルクリック）すると、期限が来たカードの復習が始まります。各カードは対応するポジションを ボードに表示します。キー
    <strong>1</strong>（Again）、<strong>2</strong>（Hard）、<strong>3</strong>（Good）、<strong>4</strong>（Easy）で記憶度を評価します。<strong>Esc</strong> を押すと中断して デッキ一覧に戻ります。
</p>
<p>
    <strong>セッションの制限：</strong> デッキ設定で、1 回のセッションの枚数を制限できます。上限に達するとその旨を伝えてセッションが終わり、自由練習は予定に影響を与えずに続けられます。<em>0</em> は 1
    枚も出題しない設定であり、「制限なし」とは別物です。
</p>
<p>
    <strong>保持率：</strong>
    目標保持率は、負荷と質のバランスについてのあなたの選択です。設定画面にはその横に、実際の復習から<em>測定した</em>保持率が表示されます。参考情報であり、制御ではありません。目標の変更は遡及しません。各カードは次の復習から新しいペースになります。
</p>
<p>
    <strong>解答の表示：</strong> カードは問いを出します。考えたうえで
    <strong>スペース</strong>
    キーを押す（またはマスクされた領域をクリックする）と、その局面に保存された解析が表示されます。評価ボタンのすぐ下に現れ、ボタンは手の届く位置に留まります。評価するために解答を表示する必要はありません。次のカードで再びマスクされますが、タブを切り替えただけでは戻りません。
</p>
<p>
    <strong>中断/再開:</strong> <strong>Esc</strong> を押せばいつでも復習セッションを中断できます。ボタンが <em>Resume</em> に変わり、進捗が表示されます。それをクリックすると
    中断したところから続けられます。
</p>
<p><strong>デッキの管理:</strong> 操作ボタンでデッキの名前変更、同期、リセット、削除ができます。FSRS パラメーター（目標保持率、最大間隔、ファズ）はデッキごとに 設定（歯車アイコン）で構成できます。</p>

<h3>トーナメント</h3>
<p>トーナメントを使うとマッチをイベントごとにグループ化できます。<strong>Ctrl+Y</strong> でトーナメントパネルを開き、トーナメントを管理してマッチを割り当てます。</p>

<h3>統計</h3>
<p>
    統計パネル（<strong>Ctrl+D</strong>）は、インポートしたすべてのポジションから計算されたパフォーマンス統計（PR と MWC コスト）を表示します。フィルターバーを使って、
    プレイヤー、トーナメント、日付範囲、決定の種類、マッチの長さで分析を絞り込めます。任意の指標をクリックすると、対応するポジションを掘り下げて表示できます。「<strong>プレイヤー</strong>」タブには、
    プレイヤーごとにマッチ数、勝敗、判断数、PR（チェッカーとキューブ）、Snowie、ブランダー、既知のロールから測定した運が一覧表示されます。
</p>

<h3>透かしと保護されたエクスポート</h3>
<p>エクスポート時（<strong>export_db</strong>、またはエクスポートダイアログ）には、互いに独立した2つの保護を自由に有効にできます。片方だけ、もう片方だけ、または両方同時に:</p>
<ul>
    <li>
        <strong>透かし：</strong
        >エクスポートしたファイルに出所（誰が作成したか、任意のメモ）を記載します。透かしはあなたの発行者の識別情報で署名されており、他人の名前を騙って改変や偽造をすることはできません —
        ただし削除は防げず、コピーも一切禁止しません。
    </li>
    <li>
        <strong>パスワード：</strong>エクスポートを暗号化された
        <strong>.dbx</strong> コンテナに格納します。データベース自体ではなく、受け渡し中のファイルを保護します。パスワードを渡した相手は開くことができ、 出所はパスワードなしでも読み取れます。
    </li>
</ul>
<p>あなたの発行者の識別情報（透かしに署名する鍵）は、出所を記載した最初のエクスポート時に自動的に作成されます。設定の<strong>発行者の識別情報</strong>タブから確認、エクスポート、再生成できます。</p>
`,
    shortcuts: `
<h3>データベース</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-N</td>
<td>新しいデータベースを作成する。</td>
</tr>
<tr>
<td>CTRL-O</td>
<td>既存のデータベースを開く。</td>
</tr>
<tr>
<td>CTRL-SHIFT-I</td>
<td>データベースをインポートする。</td>
</tr>
<tr>
<td>CTRL-SHIFT-S</td>
<td>データベースをエクスポートする。</td>
</tr>
<tr>
<td>CTRL-Q</td>
<td>blunderDBを閉じる。</td>
</tr>
<tr>
<td>CTRL-M</td>
<td>データベースのメタデータを編集する。</td>
</tr>
</tbody>
</table>
<h3>ポジション</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-I</td>
<td>1つまたは複数のポジション/マッチをファイル（xg、xgp、sgf、mat、txt、bgf）からインポートする。</td>
</tr>
<tr>
<td>CTRL-SHIFT-F</td>
<td>マッチ/ポジションファイルのフォルダを再帰的にインポートする。</td>
</tr>
<tr>
<td>CTRL-C</td>
<td>ポジションをクリップボードにコピーする。</td>
</tr>
<tr>
<td>CTRL-X</td>
<td>ボード画像をクリップボードにコピーする（PNG）。</td>
</tr>
<tr>
<td>CTRL-X CTRL-X</td>
<td>ボードと分析の画像をクリップボードにコピーする（PNG）。</td>
</tr>
<tr>
<td>CTRL-V</td>
<td>クリップボードからポジションを貼り付ける（形式の自動検出）。</td>
</tr>
<tr>
<td>CTRL-S</td>
<td>ポジションを保存する。</td>
</tr>
<tr>
<td>CTRL-U</td>
<td>ポジションを更新する。</td>
</tr>
<tr>
<td>Del</td>
<td>現在の局面を削除する（確認を求められます）。</td>
</tr>
<tr>
<td>BACKSPACE</td>
<td>ボード、キューブ、スコア、ダイスをリセットする。</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>ポジションのメタデータを表示する。</td>
</tr>
</tbody>
</table>
<h3>ナビゲーション</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-R</td>
<td>データベースからすべてのポジションを再読み込みする。</td>
</tr>
<tr>
<td>PageUp, h</td>
<td>最初のポジション / 前のゲーム（マッチナビゲーション）。</td>
</tr>
<tr>
<td>LEFT, k</td>
<td>前のポジション。</td>
</tr>
<tr>
<td>RIGHT, j</td>
<td>次のポジション。</td>
</tr>
<tr>
<td>UP, k</td>
<td>前の手（分析で手が選択されている場合）。</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>次の手（分析で手が選択されている場合）。</td>
</tr>
<tr>
<td>PageDown, l</td>
<td>最後のポジション / 次のゲーム（マッチナビゲーション）。</td>
</tr>
<tr>
<td>r</td>
<td>ランダムなポジションを読み込む。</td>
</tr>
</tbody>
</table>
<h3>表示</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-LEFT</td>
<td>ボードの向きを左にする。</td>
</tr>
<tr>
<td>CTRL-RIGHT</td>
<td>ボードの向きを右にする。</td>
</tr>
<tr>
<td>p</td>
<td>ピップカウントを表示/非表示にする。</td>
</tr>
</tbody>
</table>
<h3>アクション</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>TAB</td>
<td>検索パネル（ポジションエディタ）を開く。</td>
</tr>
<tr>
<td>SPACE</td>
<td>コマンドラインを開く。</td>
</tr>
</tbody>
</table>
<h3>ツール</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-L</td>
<td>分析を表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>コメントを表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-K</td>
<td>Ankiパネル（間隔反復）を表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-F</td>
<td>検索パネルを表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-Tab</td>
<td>マッチパネルを表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-B</td>
<td>コレクションパネルを表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-Y</td>
<td>トーナメントパネルを表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-D</td>
<td>統計パネルを表示/非表示にする。</td>
</tr>
<tr>
<td>CTRL-E</td>
<td>Eval パネルを表示/非表示にする。</td>
</tr>
<tr>
<td>?</td>
<td>ヘルプを表示/非表示にする。</td>
</tr>
</tbody>
</table>
<h3>ビュータブ</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-T</td>
<td>新しいビューを作成する（現在のビューのコピー）。</td>
</tr>
<tr>
<td>CTRL-W</td>
<td>現在のビューを閉じる。</td>
</tr>
<tr>
<td>CTRL-PageUp, SHIFT-J</td>
<td>前のビュー。</td>
</tr>
<tr>
<td>CTRL-PageDown, SHIFT-K</td>
<td>次のビュー。</td>
</tr>
<tr>
<td>CTRL-1 … CTRL-9</td>
<td>n番目のビューに直接移動する。</td>
</tr>
<tr>
<td>タブをダブルクリック</td>
<td>ビューの名前を変更する。</td>
</tr>
</tbody>
</table>
<h3>コマンドライン</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>UP</td>
<td>コマンド履歴を上方向に参照する。</td>
</tr>
<tr>
<td>DOWN</td>
<td>コマンド履歴を下方向に参照する。</td>
</tr>
</tbody>
</table>
<h3>検索履歴</h3>
<p>検索履歴は、検索パネル（<em>CTRL-F</em>または<em>TAB</em>）の<em>履歴</em>タブです。</p>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>クリック</td>
<td>検索を選択/選択解除する（ポジションを表示）。</td>
</tr>
<tr>
<td>ダブルクリック</td>
<td>検索を実行する。</td>
</tr>
</tbody>
</table>
<h3>フィルターライブラリ</h3>
<p>フィルターライブラリは、検索パネル（<em>CTRL-F</em>または<em>TAB</em>）の<em>保存済み</em>タブです。</p>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>クリック</td>
<td>フィルターを選択/選択解除する（ポジションを表示）。</td>
</tr>
<tr>
<td>ダブルクリック</td>
<td>フィルター検索を実行する。</td>
</tr>
</tbody>
</table>
<h3>分析パネル</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>クリック</td>
<td>手を選択/選択解除する（矢印を表示/非表示）。</td>
</tr>
<tr>
<td>UP, k</td>
<td>前の手を選択する（手が選択されている場合）。</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>次の手を選択する（手が選択されている場合）。</td>
</tr>
<tr>
<td>d</td>
<td>チェッカー分析とキューブ分析を切り替える（マッチナビゲーションのみ）。</td>
</tr>
<tr>
<td>Esc</td>
<td>手の選択を解除する。手が選択されていない場合はパネルを閉じる。</td>
</tr>
</tbody>
</table>
<h3>Eval パネル</h3>
<p><em>Eval</em>パネルの手のリスト（Eval パネル を参照）は分析パネルのリストと同じように移動できます。これらのショートカットは、クリックで手を選択した時点から有効になります。</p>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>クリック</td>
<td>手を選択/選択解除する（矢印を表示/非表示）。</td>
</tr>
<tr>
<td>UP, k</td>
<td>前の手を選択する（手が選択されている場合）。</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>次の手を選択する（手が選択されている場合）。</td>
</tr>
<tr>
<td>Esc</td>
<td>手の選択を解除する。</td>
</tr>
</tbody>
</table>
<h3>マッチパネル</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>クリック</td>
<td>マッチを選択する。</td>
</tr>
<tr>
<td>ダブルクリック</td>
<td>マッチ内をナビゲートする。</td>
</tr>
<tr>
<td>UP, k</td>
<td>前のマッチを選択する。</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>次のマッチを選択する。</td>
</tr>
<tr>
<td>ENTER</td>
<td>選択したマッチを読み込む。</td>
</tr>
<tr>
<td>Del</td>
<td>選択したマッチを削除する。</td>
</tr>
<tr>
<td>Esc</td>
<td>選択を解除する/パネルを閉じる。</td>
</tr>
</tbody>
</table>
<h3>Ankiパネル（間隔反復）</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>SPACE、クリック</td>
<td>答えを表示する（ポジションに記録された分析）。</td>
</tr>
<tr>
<td>1</td>
<td>評価：もう一度（失敗、すぐに再確認）。</td>
</tr>
<tr>
<td>2</td>
<td>評価：難しい。</td>
</tr>
<tr>
<td>3</td>
<td>評価：良い。</td>
</tr>
<tr>
<td>4</td>
<td>評価：簡単。</td>
</tr>
<tr>
<td>p</td>
<td>ピップカウントの表示／非表示（一般のショートカットと同じで、復習中も使えます）。</td>
</tr>
<tr>
<td>Esc</td>
<td>レビューを停止してデッキリストに戻る（後で再開可能）。</td>
</tr>
</tbody>
</table>
<h3>トーナメントパネル</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>クリック、ダブルクリック</td>
<td>トーナメントを選択する（詳細を表示）。</td>
</tr>
<tr>
<td>UP, k</td>
<td>前のトーナメントを選択する。</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>次のトーナメントを選択する。</td>
</tr>
<tr>
<td>ダブルクリック（トーナメントのマッチ上で）</td>
<td>マッチ内をナビゲートする。</td>
</tr>
<tr>
<td>Esc</td>
<td>編集中ならそれを取り消し、そうでなければマッチ追加の検索をクリアし、そうでなければトーナメントの選択を解除し、そうでなければパネルを閉じる（段階的に）。</td>
</tr>
</tbody>
</table>
<h3>コレクションパネル</h3>
<table>
<thead>
<tr>
<th>ショートカット</th>
<th>アクション</th>
</tr>
</thead>
<tbody>
<tr>
<td>クリック</td>
<td>ポインタが乗っているコレクションに現在の局面を追加、または取り除く。</td>
</tr>
<tr>
<td>ダブルクリック</td>
<td>コレクションを開く。</td>
</tr>
<tr>
<td>Del</td>
<td>開いているコレクションから、現在の局面（またはチェックした局面）を取り除く。</td>
</tr>
<tr>
<td>Esc</td>
<td>コレクション一覧に戻り、そうでなければコレクションの選択を解除し、そうでなければパネルを閉じる（段階的に）。</td>
</tr>
</tbody>
</table>
`,
    commands: `
<p>コマンドラインはステータスバーにあり、 <em>スペース</em> キーを押すと開きます。コマンドを入力すると、候補のリストが自動的に表示されます。 <em>TAB</em> キー（または <em>Shift-TAB</em> ）は候補を順に巡回してコマンドを補完し、 <em>ESC</em> はリストを閉じます（2回目の <em>ESC</em> でコマンドラインが閉じます）。 <em>上</em> キーと <em>下</em> キーは引き続きコマンド履歴用に予約されています。</p>
<h3>全体操作</h3>
<table>
<thead>
<tr>
<th>コマンド</th>
<th>動作</th>
</tr>
</thead>
<tbody>
<tr>
<td>new, ne, n</td>
<td>新しいデータベースを作成します。</td>
</tr>
<tr>
<td>open, op, o</td>
<td>既存のデータベースを開きます。</td>
</tr>
<tr>
<td>import_db, idb</td>
<td>別のデータベースをインポートして統合します。</td>
</tr>
<tr>
<td>export_db, edb</td>
<td>現在の選択を新しいデータベースにエクスポートします。</td>
</tr>
<tr>
<td>quit, q</td>
<td>blunderDB を終了します。</td>
</tr>
<tr>
<td>help, he, h</td>
<td>blunderDB のヘルプを開きます。</td>
</tr>
<tr>
<td>tutorial, tour</td>
<td>インターフェースのガイドツアーのカタログを開きます。</td>
</tr>
<tr>
<td>demo</td>
<td>ツールを試すためのサンプルデータベース（マッチ、トーナメント、コレクション、コメント、Anki デッキ、解析）を読み込みます。</td>
</tr>
<tr>
<td>meta</td>
<td>データベースのメタデータを表示します。</td>
</tr>
<tr>
<td>epc</td>
<td>Eval パネル（Effective Pip Count、勝率、ベアオフでのキューブ判定）を開きます。</td>
</tr>
<tr>
<td>met</td>
<td>Kazaross-XG2 のマッチエクイティ表を開きます。</td>
</tr>
<tr>
<td>tp2</td>
<td>キューブ値 2 のテイクポイント表を開きます。</td>
</tr>
<tr>
<td>tp2_live</td>
<td>ロングレース用のキューブ値 2 のテイクポイント表を開きます。</td>
</tr>
<tr>
<td>tp2_last</td>
<td>デッドキューブ値 2 のテイクポイント表を開きます。</td>
</tr>
<tr>
<td>tp4</td>
<td>キューブ値 4 のテイクポイント表を開きます。</td>
</tr>
<tr>
<td>tp4_live</td>
<td>ロングレース用のキューブ値 4 のテイクポイント表を開きます。</td>
</tr>
<tr>
<td>tp4_last</td>
<td>デッドキューブ値 4 のテイクポイント表を開きます。</td>
</tr>
<tr>
<td>gv1</td>
<td>キューブ値 1 のギャモン値表を開きます。</td>
</tr>
<tr>
<td>gv2</td>
<td>キューブ値 2 のギャモン値表を開きます。</td>
</tr>
<tr>
<td>gv4</td>
<td>キューブ値 4 のギャモン値表を開きます。</td>
</tr>
</tbody>
</table>
<h3>局面とナビゲーション</h3>
<table>
<thead>
<tr>
<th>コマンド</th>
<th>動作</th>
</tr>
</thead>
<tbody>
<tr>
<td>import, i</td>
<td>ファイル（xg、xgp、sgf、mat、txt、bgf）から 1 つ以上の局面／マッチをインポートします。</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>現在の局面を削除します（確認を求められます）。</td>
</tr>
<tr>
<td>[number]</td>
<td>指定したインデックスの局面に移動します。</td>
</tr>
<tr>
<td>list, l</td>
<td>現在の局面の解析を表示します。</td>
</tr>
<tr>
<td>comment, co</td>
<td>コメントを表示／記入します。</td>
</tr>
<tr>
<td>history, hi</td>
<td>検索パネルを開きます（検索履歴はその<em>履歴</em>タブにあります）。</td>
</tr>
<tr>
<td>stats, st</td>
<td>統計パネルを表示／非表示にします。</td>
</tr>
<tr>
<td>match, ma</td>
<td>マッチパネルを表示／非表示にします。</td>
</tr>
<tr>
<td>collection, coll</td>
<td>コレクションパネルを表示／非表示にします。</td>
</tr>
<tr>
<td>#tag1 tag2 ...</td>
<td>現在の局面にタグを付けます。</td>
</tr>
<tr>
<td>e</td>
<td>データベースのすべての局面を読み込みます。</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>現在の統計フィルターに従って、最悪のミス（エクイティ/MWC）を分析ビューに読み込みます。</td>
</tr>
<tr>
<td>m</td>
<td>最後に閲覧したマッチ内を移動します。</td>
</tr>
</tbody>
</table>
<h3>編集と検索</h3>
<table>
<thead>
<tr>
<th>コマンド</th>
<th>動作</th>
</tr>
</thead>
<tbody>
<tr>
<td>write, wr, w</td>
<td>現在の局面を保存します。</td>
</tr>
<tr>
<td>write!, wr!, w!</td>
<td>現在の局面を更新します。</td>
</tr>
<tr>
<td>s</td>
<td>フィルタを使って局面を検索します。</td>
</tr>
<tr>
<td>ss</td>
<td>現在フィルタされている局面の中から検索します。</td>
</tr>
</tbody>
</table>
<h3>検索フィルタ</h3>
<p>以下のフィルタは検索時、すなわちコマンドの先頭 <code>s</code> の後に並べて指定する必要があります。</p>
<table>
<thead>
<tr>
<th>クエリ</th>
<th>動作</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>局面がキューブの配置を満たします。</td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>局面がスコアを満たします。</td>
</tr>
<tr>
<td>d</td>
<td>局面が決定の種類（チェッカーまたはキューブ）を満たします。</td>
</tr>
<tr>
<td>D</td>
<td>局面がダイスの目を満たします（順序を問わず両方のダイス）。</td>
</tr>
<tr>
<td>D1</td>
<td>局面が 1 つ目のダイスのみについてダイスの目を満たします（1 つ目のダイスの値が局面の 2 つのダイスのいずれかに現れる）。</td>
</tr>
<tr>
<td>xD65</td>
<td>局面が 6-5 の出目で<strong>プレイされていない</strong>状態です（順序は問いません）。値はトークン内に指定します。複数の出目を除外するために繰り返し使用できます（<code>xD65 xD54</code>）。</td>
</tr>
<tr>
<td>nc</td>
<td>局面が接触のない状態です。</td>
</tr>
<tr>
<td>M</td>
<td>局面またはその鏡像がフィルタを満たします。</td>
</tr>
<tr>
<td>i</td>
<td>この局面は単独でインポートされたもので、マッチのインポートで入ったものではありません。</td>
</tr>
<tr>
<td>fl</td>
<td>eXtreme Gammon のマッチをインポートした際、元のソフトウェアでフラグが付けられた局面。</td>
</tr>
<tr>
<td>x</td>
<td>局面が除外構造（検索パネルの「Except」タブ）の駒を 1 つも含みません。</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>プレイヤーがレースで少なくとも x ピップ遅れています。</td>
</tr>
<tr>
<td>p&lt;x</td>
<td>プレイヤーがレースで最大 x ピップ遅れています。</td>
</tr>
<tr>
<td>px,y</td>
<td>プレイヤーがレースで x から y ピップ遅れています。</td>
</tr>
<tr>
<td>P&gt;x</td>
<td>プレイヤーのレースが少なくとも x ピップです。</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>プレイヤーのレースが最大 x ピップです。</td>
</tr>
<tr>
<td>Px,y</td>
<td>プレイヤーのレースが x から y ピップです。</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>局面のエクイティ（ミリポイント単位）が x より大きいです。</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>局面のエクイティ（ミリポイント単位）が x より小さいです。</td>
</tr>
<tr>
<td>ex,y</td>
<td>局面のエクイティ（ミリポイント単位）が x から y の範囲です。</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>プレイヤー 1 が打った手の誤差（ミリポイント単位）が x より大きいです。</td>
</tr>
<tr>
<td>E&lt;x</td>
<td>プレイヤー 1 が打った手の誤差（ミリポイント単位）が x より小さいです。</td>
</tr>
<tr>
<td>Ex,y</td>
<td>プレイヤー 1 が打った手の誤差（ミリポイント単位）が x から y の範囲です。</td>
</tr>
<tr>
<td>w&gt;x</td>
<td>プレイヤーの勝率が x % より大きいです。</td>
</tr>
<tr>
<td>w&lt;x</td>
<td>プレイヤーの勝率が x % より小さいです。</td>
</tr>
<tr>
<td>wx,y</td>
<td>プレイヤーの勝率が x % から y % の範囲です。</td>
</tr>
<tr>
<td>g&gt;x</td>
<td>プレイヤーのギャモン率が x % より大きいです。</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>プレイヤーのギャモン率が x % より小さいです。</td>
</tr>
<tr>
<td>gx,y</td>
<td>プレイヤーのギャモン率が x % から y % の範囲です。</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>プレイヤーのバックギャモン率が x % より大きいです。</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>プレイヤーのバックギャモン率が x % より小さいです。</td>
</tr>
<tr>
<td>bx,y</td>
<td>プレイヤーのバックギャモン率が x % から y % の範囲です。</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>相手の勝率が x % より大きいです。</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>相手の勝率が x % より小さいです。</td>
</tr>
<tr>
<td>Wx,y</td>
<td>相手の勝率が x % から y % の範囲です。</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>相手のギャモン率が x % より大きいです。</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>相手のギャモン率が x % より小さいです。</td>
</tr>
<tr>
<td>Gx,y</td>
<td>相手のギャモン率が x % から y % の範囲です。</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>相手のバックギャモン率が x % より大きいです。</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>相手のバックギャモン率が x % より小さいです。</td>
</tr>
<tr>
<td>Bx,y</td>
<td>相手のバックギャモン率が x % から y % の範囲です。</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>プレイヤーが少なくとも x 個の駒を上がっています。</td>
</tr>
<tr>
<td>o&lt;x</td>
<td>プレイヤーが最大 x 個の駒を上がっています。</td>
</tr>
<tr>
<td>ox,y</td>
<td>プレイヤーが x から y 個の駒を上がっています。</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>相手が少なくとも x 個の駒を上がっています。</td>
</tr>
<tr>
<td>O&lt;x</td>
<td>相手が最大 x 個の駒を上がっています。</td>
</tr>
<tr>
<td>Ox,y</td>
<td>相手が x から y 個の駒を上がっています。</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>プレイヤーが少なくとも x 個の後方の駒を持っています。</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>プレイヤーが最大 x 個の後方の駒を持っています。</td>
</tr>
<tr>
<td>kx,y</td>
<td>プレイヤーが x から y 個の後方の駒を持っています。</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>相手が少なくとも x 個の後方の駒を持っています。</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>相手が最大 x 個の後方の駒を持っています。</td>
</tr>
<tr>
<td>Kx,y</td>
<td>相手が x から y 個の後方の駒を持っています。</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>プレイヤーがゾーン内に少なくとも x 個の駒を持っています。</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>プレイヤーがゾーン内に最大 x 個の駒を持っています。</td>
</tr>
<tr>
<td>zx,y</td>
<td>プレイヤーがゾーン内に x から y 個の駒を持っています。</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>相手がゾーン内に少なくとも x 個の駒を持っています。</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>相手がゾーン内に最大 x 個の駒を持っています。</td>
</tr>
<tr>
<td>Zx,y</td>
<td>相手がゾーン内に x から y 個の駒を持っています。</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>プレイヤーがアウトフィールドに少なくとも x 個のブロットを持っています。</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>プレイヤーがアウトフィールドに最大 x 個のブロットを持っています。</td>
</tr>
<tr>
<td>box,y</td>
<td>プレイヤーがアウトフィールドに x から y 個のブロットを持っています。</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>相手がアウトフィールドに少なくとも x 個のブロットを持っています。</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>相手がアウトフィールドに最大 x 個のブロットを持っています。</td>
</tr>
<tr>
<td>BOx,y</td>
<td>相手がアウトフィールドに x から y 個のブロットを持っています。</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>プレイヤーが内盤に少なくとも x 個のブロットを持っています。</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>プレイヤーが内盤に最大 x 個のブロットを持っています。</td>
</tr>
<tr>
<td>bjx,y</td>
<td>プレイヤーが内盤に x から y 個のブロットを持っています。</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>相手が内盤に少なくとも x 個のブロットを持っています。</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>相手が内盤に最大 x 個のブロットを持っています。</td>
</tr>
<tr>
<td>BJx,y</td>
<td>相手が内盤に x から y 個のブロットを持っています。</td>
</tr>
<tr>
<td>t'mot1;mot2;...'</td>
<td>局面のコメントに単語のうち少なくとも 1 つが含まれます。</td>
</tr>
<tr>
<td>co</td>
<td>内容を問わず、局面にコメントが付いている。</td>
</tr>
<tr>
<td>xco</td>
<td>局面にコメントが付いていない。</td>
</tr>
<tr>
<td>m'motif1,motif2,...'</td>
<td>最善のチェッカープレイがパターンのうち少なくとも 1 つを含みます。</td>
</tr>
<tr>
<td>m'ND,DT,DP,...'</td>
<td>最善のキューブ判断が No Double/Take、Double Take、Double Pass です。</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>局面の追加日が x より後（YYYY/MM/DD）。</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>局面の追加日が x より前（YYYY/MM/DD）。</td>
</tr>
<tr>
<td>Tx,y</td>
<td>局面の追加日が x から y の範囲（YYYY/MM/DD）。</td>
</tr>
<tr>
<td>max</td>
<td>識別子 x のマッチ内を検索します（例: ma3）。</td>
</tr>
<tr>
<td>max,y</td>
<td>識別子 x から y のマッチ内を検索します（例: ma2,5）。</td>
</tr>
<tr>
<td>tnx</td>
<td>識別子 x のトーナメント内を検索します（例: tn1）。</td>
</tr>
<tr>
<td>tnx,y</td>
<td>識別子 x から y のトーナメント内を検索します（例: tn1,3）。</td>
</tr>
<tr>
<td>idx</td>
<td>識別子 x のポジションを検索します（例: id12）。</td>
</tr>
<tr>
<td>idx,y</td>
<td>識別子 x から y までのポジションを検索します（例: id5,10）。</td>
</tr>
<tr>
<td>pl'名前'</td>
<td>指定したプレイヤーが（どちらの席でも）参加した対局の局面を検索します（例：pl'Alice'）。大文字・小文字は区別しません。</td>
</tr>
</tbody>
</table>
<p>例えば、コマンド <code>s s c p-20,-5 w&gt;60 z&gt;10 K2,3</code> は、編集中の局面の駒の配置、スコア、キューブを考慮し、プレイヤーがレースで 20 から 5 ピップ進んでおり、勝率が少なくとも 60%、ゾーン内に少なくとも 10 個の駒があり、相手が 2 から 3 個の後方の駒を持つすべての局面をフィルタします。</p>
<h3>その他のコマンド</h3>
<table>
<thead>
<tr>
<th>コマンド</th>
<th>動作</th>
</tr>
</thead>
<tbody>
<tr>
<td>clear, cl</td>
<td>コマンド履歴を消去します。</td>
</tr>
</tbody>
</table>
`,
    about: `
<h3>バージョン</h3>
<p>アプリケーションバージョン: {appVersion}</p>
<p>データベースバージョン: {dbVersion}</p>

<h3>作者</h3>
<p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
<p>Heroes では <strong>postmanpat</strong> というニックネームでも見つけられます。</p>
<p>
    blunderDB はもともと、自分のミスのパターンを検出するための個人的な用途で開発しました。しかし、特に設計、コーディング、デバッグに多くの時間を費やしたあとでは、
    フィードバックをもらえるのはとても嬉しいものです。ですので、ぜひ感想をお寄せください。
</p>
<p>連絡方法はいくつかあります:</p>
<ul>
    <li>blunderDB の Discord サーバーに参加してください：<a href="https://discord.gg/DA5PpzM9En" target="_blank" rel="noopener noreferrer">discord.gg/DA5PpzM9En</a>、</li>
    <li>トーナメントで会ったら声をかけてください、</li>
    <li>メールを送ってください、</li>
</ul>
<h3>ライセンス</h3>
<p>
    blunderDB は MIT ライセンスの下で提供されています。これは、元の著作権表示とこの許諾表示をソフトウェアのすべての複製または重要な部分に含めることを条件として、
    ソフトウェアの使用、複製、変更、結合、公開、配布、サブライセンス、および/または販売を自由に行えることを意味します。
</p>
<h3>謝辞</h3>
<p>このささやかなソフトウェアを、パートナーの <strong>Anne-Claire</strong> と愛する娘の <strong>Perrine</strong> に捧げます。特に何人かの友人に感謝したいと思います:</p>
<ul>
    <li>
        <strong>Tristan Remille</strong>。喜びと優しさをもってバックギャモンを教えてくれたこと。この素晴らしいゲームを理解する「道」を示してくれたこと。私の拙い上達の試みにもかかわらず
        支え続けてくれたことに。
    </li>
    <li><strong>Nicolas Harmand</strong>。10 年以上にわたり素晴らしい冒険を共にした陽気な相棒であり、バックギャモンにはまって以来の最高のゲームパートナーに。</li>
</ul>
<h3>クレジット</h3>
<p>blunderDB には他の人々が作成したコード、データ、フォントが含まれています。主なもの:</p>
<ul>
    <li>
        ニューラルネットワーク <strong>strehl-prob5-512-512-256-128</strong> は
        <strong>Alexander Strehl</strong> の著作物です（<em>alexstrehl/backgammon-ai-engine</em>、MIT）。それを取り巻く探索、キューブモデル、マッチエクイティテーブルは
        <strong>gammonNet</strong> 独自の構成です（<a href="https://github.com/kevung/gammonNet" target="_blank" rel="noopener noreferrer">github.com/kevung/gammonNet</a>、MIT）。
    </li>
    <li>Kazaross-XG2 マッチエクイティテーブル（MET）は <strong>Neil Kazaross</strong> の著作物です。</li>
    <li>テイクポイントとギャモン値の表は <strong>Dirk Schiemann</strong> の著書 <em>The Theory of Backgammon</em> から引用しています。</li>
    <li>
        片側ベアオフデータベース（6 ポイント、15 駒、EPC 用）と両側ベアオフデータベース（6 ポイント、6 駒、レースでのキューブ判定用）は <strong>GNU Backgammon</strong>（GNUbg）で生成されました。GNUbg
        は GPL の下で提供されるフリーソフトウェアであり、これらの表はそれが生成したデータとしてクレジットされています。
    </li>
    <li>マッチファイルは <em>xgparser</em>、<em>gnubgparser</em>、<em>bgfparser</em>（MIT）で読み込まれます。</li>
    <li>Go 側: <em>modernc.org/sqlite</em>（BSD-3-Clause）、<em>pgx</em>、<em>Wails</em>、<em>go-fsrs</em>（MIT）。</li>
    <li>インターフェース側: <em>Svelte</em>、<em>two.js</em>、<em>Chart.js</em>、<em>driver.js</em>（MIT）。</li>
    <li>フォント <em>Nunito</em> と <em>Noto Sans JP</em>（SIL Open Font License 1.1）。</li>
</ul>
<p>
    ライセンス本文を含む完全な一覧は、blunderDB に同梱される <strong>THIRD_PARTY.md</strong> ファイルです（<a
        href="https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md"
        target="_blank"
        rel="noopener noreferrer"
        >github.com/kevung/blunderDB</a
    >）。
</p>
`
};
