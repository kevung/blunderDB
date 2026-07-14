// Japanese help content.
// Each value is the verbatim inner HTML of the corresponding HelpModal tab.
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
                        <li>マッチをトーナメントに整理する。</li>
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
                        <li>ベアオフポジションの EPC 値を計算する（EPC パネル）、</li>
                        <li>保存した検索フィルターを閲覧する（フィルターライブラリパネル）、</li>
                        <li>検索履歴を閲覧する（検索履歴パネル）、</li>
                        <li>操作ログを表示する（ログパネル）。</li>
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
                        ステータスバーに統合されたコマンドラインでは、blunderDB のすべての機能を実行できます: データベース操作、ポジションのナビゲーション、分析や
                        コメントの表示、フィルターによるポジション検索など... インターフェースに慣れたら、徐々にコマンドラインを使うことをお勧めします。コマンドラインは、特にポジション検索機能において、blunderDB を
                        強力かつスムーズに使えるようにします。
                    </p>
                    <p>
                        コマンドラインを開くには、<strong>Space</strong> キーを押します。ステータスバーにプロンプトが表示されます。コマンドを入力して <strong>Enter</strong> を押すと実行されます。
                        <strong>Escape</strong>
                        を押すとキャンセルされます。コマンド履歴と結果は <strong>ログ</strong> パネルに記録されます。
                    </p>
                    <p>
                        blunderDB はユーザーが送ったクエリが有効であれば実行し、必要に応じて直ちにデータベースの状態を変更します。ユーザーによる明示的な保存操作は
                        必要ありません。
                    </p>
                    <p>
                        以前に絞り込んだポジションの中でさらに検索を絞り込むには、<strong>ss</strong> コマンドの後にフィルターを続けます（例: <strong>ss nc</strong>）。これは現在表示されている
                        ポジションだけに検索を限定し、結果を段階的に絞り込めるようにします。検索パネル（<strong>Ctrl+F</strong>）にも「現在の結果内を検索」というチェックボックスがあり、
                        同じ機能を提供します。
                    </p>

                    <h3>EPC 計算機</h3>
                    <p>EPC（Effective Pip Count）計算機は、ベアオフポジションの実効ピップカウントを計算します。正確な EPC 値のために GNUbg の片側 6 ポイントベアオフデータベースを使用します。</p>
                    <p>
                        EPC パネルを開くには、<strong>Ctrl+E</strong> を押すか、下部パネルの EPC タブをクリックするか、コマンドラインに <strong>epc</strong> と入力します。ボードは標準的な
                        ベアオフ構成（15 チェッカー）で初期化されます。
                    </p>
                    <p>
                        マウスを使ってホームボードのポイント上のチェッカーを自由に追加または削除できます。EPC 値は専用の EPC パネルにリアルタイムで表示され、各プレイヤーについて次を表示します:
                    </p>
                    <ul>
                        <li><strong>EPC</strong>: すべてのチェッカーをベアオフするのに必要なピップの平均数、</li>
                        <li><strong>Pip Count</strong>: 素のピップカウント、</li>
                        <li><strong>Wastage</strong>: EPC とピップカウントの差、</li>
                        <li><strong>Avg Rolls</strong>: ベアオフするまでのロールの平均回数、</li>
                        <li><strong>Std Dev</strong>: ロール回数の標準偏差。</li>
                    </ul>
                    <p>両プレイヤーがホームボードにチェッカーを持っている場合、比較セクションに EPC とピップカウントの差が表示されます。</p>
                    <p>EPC パネルを閉じるには、もう一度 <strong>Ctrl+E</strong> を押すか、別のタブに切り替えます。</p>

                    <h3>マッチのナビゲーション</h3>
                    <p>
                        blunderDB ではインポートしたマッチの手順を閲覧できます。<strong>Ctrl+Tab</strong> でマッチパネルを開き、マッチをダブルクリック（または <strong>Enter</strong> を押す）して
                        そのポジションを読み込みます。
                    </p>
                    <p>
                        マッチを閲覧しているとき、最後に訪れたポジションは自動的に保存・復元されます。<strong>Left</strong>/<strong>Right</strong> キーでポジション間を移動し、
                        <strong>PageUp</strong>/<strong>PageDown</strong> でゲーム間をジャンプします。
                    </p>
                    <p>
                        分析パネル（<strong>Ctrl+L</strong>）は各手の分析を表示し、実際にプレイされた手がハイライトされます。<strong>d</strong> を押すとチェッカー分析とキューブ分析を切り替えられます。
                    </p>

                    <h3>コレクション</h3>
                    <p>
                        コレクションを使うとポジションをカスタムグループに整理できます。<strong>Ctrl+B</strong> でコレクションパネルを開き、コレクションをダブルクリックしてそのポジションを閲覧します。
                        コレクションとその中のポジションはドラッグ＆ドロップで並べ替えられます。
                    </p>

                    <h3>Anki（間隔反復）</h3>
                    <p>Anki パネル（<strong>Ctrl+K</strong>）は、FSRS アルゴリズムを使ってバックギャモンのポジションを学習するための間隔反復を提供します。</p>
                    <p>
                        <strong>デッキの作成:</strong> <em>New Deck</em> をクリックして、コレクションまたは現在の検索結果からデッキを作成します。検索ベースのデッキは Anki
                        タブがアクティブになると自動的に同期されます。
                    </p>
                    <p>
                        <strong>復習:</strong> デッキを選択して <em>Study</em> をクリック（またはデッキをダブルクリック）すると、期限が来たカードの復習が始まります。各カードは対応するポジションを
                        ボードに表示します。キー <strong>1</strong>（Again）、<strong>2</strong>（Hard）、<strong>3</strong>（Good）、<strong>4</strong>（Easy）で記憶度を評価します。<strong>Esc</strong> を押すと中断して
                        デッキ一覧に戻ります。
                    </p>
                    <p>
                        <strong>中断/再開:</strong> <strong>Esc</strong> を押せばいつでも復習セッションを中断できます。ボタンが <em>Resume</em> に変わり、進捗が表示されます。それをクリックすると
                        中断したところから続けられます。
                    </p>
                    <p>
                        <strong>デッキの管理:</strong> 操作ボタンでデッキの名前変更、同期、リセット、削除ができます。FSRS パラメーター（目標保持率、最大間隔、ファズ）はデッキごとに
                        設定（歯車アイコン）で構成できます。
                    </p>

                    <h3>トーナメント</h3>
                    <p>トーナメントを使うとマッチをイベントごとにグループ化できます。<strong>Ctrl+Y</strong> でトーナメントパネルを開き、トーナメントを管理してマッチを割り当てます。</p>

                    <h3>統計</h3>
                    <p>
                        統計パネル（<strong>Ctrl+D</strong>）は、インポートしたすべてのポジションから計算されたパフォーマンス統計（PR と MWC コスト）を表示します。フィルターバーを使って、
                        プレイヤー、トーナメント、日付範囲、決定の種類、マッチの長さで分析を絞り込めます。任意の指標をクリックすると、対応するポジションを掘り下げて表示できます。
                    </p>
`,
    shortcuts: `
                    <h3>データベース</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + N</td>
                                <td>新規データベース</td>
                            </tr>

                            <tr>
                                <td>Ctrl + O</td>
                                <td>データベースを開く</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + I</td>
                                <td>データベースをインポート</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + S</td>
                                <td>データベースをエクスポート</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Q</td>
                                <td>blunderDB を終了</td>
                            </tr>

                            <tr>
                                <td>Ctrl + M</td>
                                <td>メタデータを編集</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>ポジション</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + I</td>
                                <td>ポジションまたはマッチをインポート</td>
                            </tr>

                            <tr>
                                <td>Ctrl + C</td>
                                <td>ポジションをコピー（ボードクリップボードにもコピー）</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X</td>
                                <td>ボード画像をクリップボードにコピー（PNG）</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X, Ctrl + X</td>
                                <td>ボード＋分析画像をクリップボードにコピー（PNG）</td>
                            </tr>

                            <tr>
                                <td>Ctrl + V</td>
                                <td>ポジションを貼り付け（検索パネルでは: ボードに貼り付け）</td>
                            </tr>

                            <tr>
                                <td>Ctrl + S</td>
                                <td>ポジションを保存</td>
                            </tr>

                            <tr>
                                <td>Ctrl + U</td>
                                <td>ポジションを更新</td>
                            </tr>

                            <tr>
                                <td>Del</td>
                                <td>ポジションを削除</td>
                            </tr>

                            <tr>
                                <td>Backspace</td>
                                <td>ボード、キューブ、スコア、ダイスをリセット</td>
                            </tr>

                            <tr>
                                <td>Ctrl + G</td>
                                <td>ポジションのメタデータを表示</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>ナビゲーション</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + R</td>
                                <td>すべてのポジションを再読み込み</td>
                            </tr>

                            <tr>
                                <td>PageUp, h</td>
                                <td>最初のポジション / 前のゲーム（マッチナビゲーション）</td>
                            </tr>

                            <tr>
                                <td>Left, k</td>
                                <td>前のポジション</td>
                            </tr>

                            <tr>
                                <td>Right, j</td>
                                <td>次のポジション</td>
                            </tr>

                            <tr>
                                <td>PageDown, l</td>
                                <td>最後のポジション / 次のゲーム（マッチナビゲーション）</td>
                            </tr>

                            <tr>
                                <td>r</td>
                                <td>ランダムなポジションを読み込む</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>表示</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + ArrowLeft</td>
                                <td>ボードの向きを左に設定</td>
                            </tr>

                            <tr>
                                <td>Ctrl + ArrowRight</td>
                                <td>ボードの向きを右に設定</td>
                            </tr>

                            <tr>
                                <td>p</td>
                                <td>ピップカウント表示切替</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>アクション</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Tab</td>
                                <td>検索パネルを開く（ポジションエディター）</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>コマンドラインを開く</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>ツール</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + L</td>
                                <td>分析を表示</td>
                            </tr>

                            <tr>
                                <td>Ctrl + P</td>
                                <td>コメントを書く</td>
                            </tr>

                            <tr>
                                <td>Ctrl + K</td>
                                <td>Anki パネルを表示</td>
                            </tr>

                            <tr>
                                <td>Ctrl + F</td>
                                <td>検索パネル</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Tab</td>
                                <td>マッチパネル</td>
                            </tr>

                            <tr>
                                <td>Ctrl + B</td>
                                <td>コレクションパネル</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Y</td>
                                <td>トーナメントパネル</td>
                            </tr>

                            <tr>
                                <td>Ctrl + D</td>
                                <td>統計パネル</td>
                            </tr>

                            <tr>
                                <td>Ctrl + E</td>
                                <td>EPC パネル</td>
                            </tr>

                            <tr>
                                <td>?</td>
                                <td>ヘルプを開く</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>コマンドライン</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Up</td>
                                <td>コマンド履歴を上にたどる</td>
                            </tr>
                            <tr>
                                <td>Down</td>
                                <td>コマンド履歴を下にたどる</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>分析パネル</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>手を選択/選択解除（矢印を表示/非表示）</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>前の手を選択（手が選択されているとき）</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>次の手を選択（手が選択されているとき）</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>チェッカー分析とキューブ分析を切り替え（マッチナビゲーションのみ）</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>手の選択を解除。手が選択されていない場合はパネルを閉じる。</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>検索履歴パネル</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>検索を選択/選択解除（ポジションを表示）</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>検索を実行してパネルを閉じる</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>前の検索を選択（より新しい、上）</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>次の検索を選択（より古い、下）</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>検索の選択を解除。検索が選択されていない場合はパネルを閉じる。</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>フィルターライブラリパネル</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>フィルターを選択/選択解除（ポジションを表示）</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>フィルター検索を実行</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>前のフィルターを選択（フィルターが選択されているとき）</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>次のフィルターを選択（フィルターが選択されているとき）</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>フィルターの選択を解除。フィルターが選択されていない場合はパネルを閉じる。</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>マッチパネル</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>マッチを選択/選択解除</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>マッチのポジションを読み込む</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>前のマッチを選択</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>次のマッチを選択</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>選択したマッチを削除</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>マッチの選択を解除。マッチが選択されていない場合はパネルを閉じる。</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>コレクションパネル</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>コレクションを選択（そのポジションを表示）</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>コレクションを開いてそのポジションを閲覧</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>現在のポジションをアクティブなコレクションから削除</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>コレクションの選択を解除。コレクションが選択されていない場合はパネルを閉じる。</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Anki パネル</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>ショートカット</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>1</td>
                                <td>評価: Again（復習に失敗、すぐ再表示）</td>
                            </tr>
                            <tr>
                                <td>2</td>
                                <td>評価: Hard（思い出しにくい）</td>
                            </tr>
                            <tr>
                                <td>3</td>
                                <td>評価: Good（正しく思い出せた）</td>
                            </tr>
                            <tr>
                                <td>4</td>
                                <td>評価: Easy（楽に思い出せた）</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>復習を中断してデッキ一覧に戻る（後で再開可能）</td>
                            </tr>
                        </tbody>
                    </table>
`,
    commands: `
                    <h3>データベース</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>コマンド</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>new, ne, n</td>
                                <td>新しいデータベースを作成</td>
                            </tr>
                            <tr>
                                <td>open, op, o</td>
                                <td>既存のデータベースを開く</td>
                            </tr>
                            <tr>
                                <td>import_db, idb</td>
                                <td>別のデータベースをインポートして統合</td>
                            </tr>
                            <tr>
                                <td>export_db, edb</td>
                                <td>現在の選択を新しいデータベースにエクスポート</td>
                            </tr>
                            <tr>
                                <td>quit, q</td>
                                <td>blunderDB を終了</td>
                            </tr>
                            <tr>
                                <td>meta</td>
                                <td>メタデータを編集</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>ポジション</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>コマンド</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>import, i</td>
                                <td>ポジションまたはマッチをインポート</td>
                            </tr>
                            <tr>
                                <td>write, wr, w</td>
                                <td>ポジションを保存</td>
                            </tr>
                            <tr>
                                <td>write!, wr!, w!</td>
                                <td>ポジションを更新</td>
                            </tr>
                            <tr>
                                <td>delete, del, d</td>
                                <td>ポジションを削除</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>ナビゲーション</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>コマンド</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>[number]</td>
                                <td>インデックスで特定のポジションへ移動</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>ツール</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>コマンド</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>list, l</td>
                                <td>分析を表示</td>
                            </tr>
                            <tr>
                                <td>comment, co</td>
                                <td>コメントを書く</td>
                            </tr>
                            <tr>
                                <td>filter, fl</td>
                                <td>フィルターライブラリを表示</td>
                            </tr>
                            <tr>
                                <td>history, hi</td>
                                <td>検索履歴を表示</td>
                            </tr>
                            <tr>
                                <td>match, ma</td>
                                <td>マッチパネルを表示</td>
                            </tr>
                            <tr>
                                <td>collection, coll</td>
                                <td>コレクションパネルを表示</td>
                            </tr>
                            <tr>
                                <td>epc</td>
                                <td>EPC 計算機（Effective Pip Count）</td>
                            </tr>
                            <tr>
                                <td>m</td>
                                <td>最後に訪れたマッチをナビゲート</td>
                            </tr>
                            <tr>
                                <td>help, he, h</td>
                                <td>ヘルプを開く</td>
                            </tr>
                            <tr>
                                <td>met</td>
                                <td>Kazaross-XG2 マッチエクイティテーブルを開く</td>
                            </tr>
                            <tr>
                                <td>tp2</td>
                                <td>2-キューブのテイクポイント（Live と Last）</td>
                            </tr>
                            <tr>
                                <td>tp2_live</td>
                                <td>ロングレースにおける 2-キューブのテイクポイント</td>
                            </tr>
                            <tr>
                                <td>tp2_last</td>
                                <td>ラストロールポジションにおける 2-キューブのテイクポイント</td>
                            </tr>
                            <tr>
                                <td>tp4</td>
                                <td>4-キューブのテイクポイント（Live と Last）</td>
                            </tr>
                            <tr>
                                <td>tp4_live</td>
                                <td>ロングレースにおける 4-キューブのテイクポイント</td>
                            </tr>
                            <tr>
                                <td>tp4_last</td>
                                <td>ラストロールポジションにおける 4-キューブのテイクポイント</td>
                            </tr>
                            <tr>
                                <td>gv1</td>
                                <td>1-キューブのギャモン値</td>
                            </tr>
                            <tr>
                                <td>gv2</td>
                                <td>2-キューブのギャモン値</td>
                            </tr>
                            <tr>
                                <td>gv4</td>
                                <td>4-キューブのギャモン値</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>ポジションにタグを付ける</td>
                            </tr>
                            <tr>
                                <td>s</td>
                                <td>フィルターでポジションを検索</td>
                            </tr>
                            <tr>
                                <td>ss</td>
                                <td>フィルターで現在の結果内を検索</td>
                            </tr>
                            <tr>
                                <td>e</td>
                                <td>すべてのポジションを再読み込み</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>フィルター</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>フィルター</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>cube, cub, cu, c</td>
                                <td>キューブを含める</td>
                            </tr>
                            <tr>
                                <td>score, sco, sc, s</td>
                                <td>スコアを含める</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>決定の種類を含める</td>
                            </tr>
                            <tr>
                                <td>D</td>
                                <td>ダイスロールを含める</td>
                            </tr>
                            <tr>
                                <td>nc</td>
                                <td>非接触ポジションを含める</td>
                            </tr>
                            <tr>
                                <td>M</td>
                                <td>ミラーポジションを含める</td>
                            </tr>
                            <tr>
                                <td>i</td>
                                <td>単独でインポートした局面のみ（マッチのインポートで入ったものを除く）</td>
                            </tr>
                            <tr>
                                <td>x</td>
                                <td
                                    >「Except」タブで描いた構造のチェッカーを<em>いずれか</em>含むポジションを除外する（例: 1、3、5 にチェッカーを描くと、それらのいずれも持たないポジションだけが残る）。
                                    フィルターの上にある「At least」/「Except」を切り替えると、除外するチェッカーをボード上に描けます（赤い印で表示）。ポイントごとの数は制限されません（あるポイントに 3 つ描くと、そこに 3 つ以上
                                    ある場合を除外します — スペアのないメイドポイント）。あるポイントを素早く 2 回クリックすると、そのポイントが空でなければならないものとしてマークされます（赤い斜線のセル、色は任意）。そのポイントを
                                    1 回クリックすると解除されます。共有ポイントでは、両者が矛盾する場合「Except」が「At least」より優先されます。</td
                                >
                            </tr>
                            <tr>
                                <td>p>x</td>
                                <td>ピップカウント &gt; x</td>
                            </tr>
                            <tr>
                                <td>p&lt;x</td>
                                <td>ピップカウント &lt; x</td>
                            </tr>
                            <tr>
                                <td>px,y</td>
                                <td>ピップカウントが x と y の間</td>
                            </tr>
                            <tr>
                                <td>P>x</td>
                                <td>プレイヤーの絶対ピップカウント &gt; x</td>
                            </tr>
                            <tr>
                                <td>P&lt;x</td>
                                <td>プレイヤーの絶対ピップカウント &lt; x</td>
                            </tr>
                            <tr>
                                <td>Px,y</td>
                                <td>プレイヤーの絶対ピップカウントが x と y の間</td>
                            </tr>
                            <tr>
                                <td>e>x</td>
                                <td>エクイティ &gt; x（ミリポイント単位）</td>
                            </tr>
                            <tr>
                                <td>e&lt;x</td>
                                <td>エクイティ &lt; x（ミリポイント単位）</td>
                            </tr>
                            <tr>
                                <td>ex,y</td>
                                <td>エクイティが x と y の間（ミリポイント単位）</td>
                            </tr>
                            <tr>
                                <td>E>x</td>
                                <td>プレイヤー 1 の手のエラー &gt; x（ミリポイント単位）</td>
                            </tr>
                            <tr>
                                <td>E&lt;x</td>
                                <td>プレイヤー 1 の手のエラー &lt; x（ミリポイント単位）</td>
                            </tr>
                            <tr>
                                <td>Ex,y</td>
                                <td>プレイヤー 1 の手のエラーが x と y の間（ミリポイント単位）</td>
                            </tr>
                            <tr>
                                <td>w>x</td>
                                <td>勝率 &gt; x</td>
                            </tr>
                            <tr>
                                <td>w&lt;x</td>
                                <td>勝率 &lt; x</td>
                            </tr>
                            <tr>
                                <td>wx,y</td>
                                <td>勝率が x と y の間</td>
                            </tr>
                            <tr>
                                <td>g>x</td>
                                <td>ギャモン率 &gt; x</td>
                            </tr>
                            <tr>
                                <td>g&lt;x</td>
                                <td>ギャモン率 &lt; x</td>
                            </tr>
                            <tr>
                                <td>gx,y</td>
                                <td>ギャモン率が x と y の間</td>
                            </tr>
                            <tr>
                                <td>b>x</td>
                                <td>バックギャモン率 &gt; x</td>
                            </tr>
                            <tr>
                                <td>b&lt;x</td>
                                <td>バックギャモン率 &lt; x</td>
                            </tr>
                            <tr>
                                <td>bx,y</td>
                                <td>バックギャモン率が x と y の間</td>
                            </tr>
                            <tr>
                                <td>W>x</td>
                                <td>相手の勝率 &gt; x</td>
                            </tr>
                            <tr>
                                <td>W&lt;x</td>
                                <td>相手の勝率 &lt; x</td>
                            </tr>
                            <tr>
                                <td>Wx,y</td>
                                <td>相手の勝率が x と y の間</td>
                            </tr>
                            <tr>
                                <td>G>x</td>
                                <td>相手のギャモン率 &gt; x</td>
                            </tr>
                            <tr>
                                <td>G&lt;x</td>
                                <td>相手のギャモン率 &lt; x</td>
                            </tr>
                            <tr>
                                <td>Gx,y</td>
                                <td>相手のギャモン率が x と y の間</td>
                            </tr>
                            <tr>
                                <td>B>x</td>
                                <td>相手のバックギャモン率 &gt; x</td>
                            </tr>
                            <tr>
                                <td>B&lt;x</td>
                                <td>相手のバックギャモン率 &lt; x</td>
                            </tr>
                            <tr>
                                <td>Bx,y</td>
                                <td>相手のバックギャモン率が x と y の間</td>
                            </tr>
                            <tr>
                                <td>o>x</td>
                                <td>プレイヤーのベアオフ済みチェッカー &gt; x</td>
                            </tr>
                            <tr>
                                <td>o&lt;x</td>
                                <td>プレイヤーのベアオフ済みチェッカー &lt; x</td>
                            </tr>
                            <tr>
                                <td>ox,y</td>
                                <td>プレイヤーのベアオフ済みチェッカーが x と y の間</td>
                            </tr>
                            <tr>
                                <td>O>x</td>
                                <td>相手のベアオフ済みチェッカー &gt; x</td>
                            </tr>
                            <tr>
                                <td>O&lt;x</td>
                                <td>相手のベアオフ済みチェッカー &lt; x</td>
                            </tr>
                            <tr>
                                <td>Ox,y</td>
                                <td>相手のベアオフ済みチェッカーが x と y の間</td>
                            </tr>
                            <tr>
                                <td>k>x</td>
                                <td>プレイヤーのバックチェッカー &gt; x</td>
                            </tr>
                            <tr>
                                <td>k&lt;x</td>
                                <td>プレイヤーのバックチェッカー &lt; x</td>
                            </tr>
                            <tr>
                                <td>kx,y</td>
                                <td>プレイヤーのバックチェッカーが x と y の間</td>
                            </tr>
                            <tr>
                                <td>K>x</td>
                                <td>相手のバックチェッカー &gt; x</td>
                            </tr>
                            <tr>
                                <td>K&lt;x</td>
                                <td>相手のバックチェッカー &lt; x</td>
                            </tr>
                            <tr>
                                <td>Kx,y</td>
                                <td>相手のバックチェッカーが x と y の間</td>
                            </tr>
                            <tr>
                                <td>z>x</td>
                                <td>プレイヤーのゾーン内チェッカー &gt; x</td>
                            </tr>
                            <tr>
                                <td>z&lt;x</td>
                                <td>プレイヤーのゾーン内チェッカー &lt; x</td>
                            </tr>
                            <tr>
                                <td>zx,y</td>
                                <td>プレイヤーのゾーン内チェッカーが x と y の間</td>
                            </tr>
                            <tr>
                                <td>Z>x</td>
                                <td>相手のゾーン内チェッカー &gt; x</td>
                            </tr>
                            <tr>
                                <td>Z&lt;x</td>
                                <td>相手のゾーン内チェッカー &lt; x</td>
                            </tr>
                            <tr>
                                <td>Zx,y</td>
                                <td>相手のゾーン内チェッカーが x と y の間</td>
                            </tr>
                            <tr>
                                <td>bo>x</td>
                                <td>プレイヤーのアウトフィールドのブロット &gt; x</td>
                            </tr>
                            <tr>
                                <td>bo&lt;x</td>
                                <td>プレイヤーのアウトフィールドのブロット &lt; x</td>
                            </tr>
                            <tr>
                                <td>box,y</td>
                                <td>プレイヤーのアウトフィールドのブロットが x と y の間</td>
                            </tr>
                            <tr>
                                <td>BO>x</td>
                                <td>相手のアウトフィールドのブロット &gt; x</td>
                            </tr>
                            <tr>
                                <td>BO&lt;x</td>
                                <td>相手のアウトフィールドのブロット &lt; x</td>
                            </tr>
                            <tr>
                                <td>BOx,y</td>
                                <td>相手のアウトフィールドのブロットが x と y の間</td>
                            </tr>
                            <tr>
                                <td>bj&gt;x</td>
                                <td>プレイヤーの Jan ブロット &gt; x</td>
                            </tr>
                            <tr>
                                <td>bj&lt;x</td>
                                <td>プレイヤーの Jan ブロット &lt; x</td>
                            </tr>
                            <tr>
                                <td>bjx,y</td>
                                <td>プレイヤーの Jan ブロットが x と y の間</td>
                            </tr>
                            <tr>
                                <td>BJ&gt;x</td>
                                <td>相手の Jan ブロット &gt; x</td>
                            </tr>
                            <tr>
                                <td>BJ&lt;x</td>
                                <td>相手の Jan ブロット &lt; x</td>
                            </tr>
                            <tr>
                                <td>BJx,y</td>
                                <td>相手の Jan ブロットが x と y の間</td>
                            </tr>

                            <tr>
                                <td>t"word1;word2;..."</td>
                                <td>テキストを検索</td>
                            </tr>
                            <tr>
                                <td>m"pattern1;pattern2;..."</td>
                                <td>指定したパターンの少なくとも 1 つを含む最善手</td>
                            </tr>
                            <tr>
                                <td>m"ND;DT;DP;..."</td>
                                <td>最善のキューブ決定が No Double/Take、Double/Take、Double/Pass</td>
                            </tr>
                            <tr>
                                <td>T&gt;x</td>
                                <td>作成日 &gt; x（年/月/日）</td>
                            </tr>
                            <tr>
                                <td>T&lt;x</td>
                                <td>作成日 &lt; x（年/月/日）</td>
                            </tr>
                            <tr>
                                <td>Tx,y</td>
                                <td>作成日が x と y の間</td>
                            </tr>
                            <tr>
                                <td>max</td>
                                <td>ID が x のマッチ内を検索（例: ma3）</td>
                            </tr>
                            <tr>
                                <td>max,y</td>
                                <td>ID が x から y までのマッチ内を検索（例: ma2,5）</td>
                            </tr>
                            <tr>
                                <td>tnx</td>
                                <td>ID が x のトーナメント内を検索（例: tn1）</td>
                            </tr>
                            <tr>
                                <td>tnx,y</td>
                                <td>ID が x から y までのトーナメント内を検索（例: tn1,3）</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>その他</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>コマンド</th>
                                <th>説明</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>clear, cl</td>
                                <td>コマンド履歴を消去</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_0_to_1_1</td>
                                <td>データベースをバージョン 1.0 から 1.1 に移行</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_1_to_1_2</td>
                                <td>データベースをバージョン 1.1 から 1.2 に移行</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_2_to_1_3</td>
                                <td>データベースをバージョン 1.2 から 1.3 に移行</td>
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
                    <p>Kazaross-XG2 マッチエクイティテーブル（MET）は <strong>Neil Kazaross</strong> によるものです。</p>
                    <p>テイクポイントとギャモン値の表は <strong>Dirk Schiemann</strong> の著書 <em>The Theory of Backgammon</em> から引用しています。</p>
                    <p>
                        EPC（Effective Pip Count）の計算に使用される片側 6 ポイントベアオフデータベースは <strong>GNU Backgammon</strong>（GNUbg）で生成されました。GNUbg は GNU 一般公衆利用許諾契約書の下で提供される
                        フリーかつオープンソースのバックギャモンプログラムです。
                    </p>
`
};
