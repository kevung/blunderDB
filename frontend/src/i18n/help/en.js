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
<h3>Introduction</h3>
<p>
    blunderDB is a software for creating backgammon position databases. Its main strength is to provide a single place to aggregate positions that a player has encountered (online, in tournaments) and
    to be able to re-study these positions by filtering them according to various arbitrarily combinable filters. blunderDB can also be used to create catalogs of reference positions.
</p>
<p>Positions are stored in a database represented by a .db file.</p>

<h3>Main Interactions</h3>
<p>The main interactions possible with blunderDB are:</p>
<ul>
    <li>adding a new position,</li>
    <li>modifying an existing position,</li>
    <li>copying the board as a PNG image to the clipboard (<strong>Ctrl+X</strong>), or the board with its analysis (<strong>Ctrl+X, Ctrl+X</strong>),</li>
    <li>deleting an existing position,</li>
    <li>searching for one or more positions,</li>
    <li>importing matches from various sources (XG, GNUbg, BGBlitz, Jellyfish), including comments from XG files,</li>
    <li>browsing the moves of an imported match,</li>
    <li>organizing positions into collections,</li>
    <li>organizing matches into tournaments,</li>
    <li>batch-analyzing, from a terminal, positions missing an analysis using the embedded gammonNet evaluator (blunderDB's <strong>analyze</strong> command).</li>
</ul>
<p>The user can freely tag positions and annotate them with comments.</p>

<h3>Description of the GUI</h3>
<p>The blunderDB GUI is structured from top to bottom as follows:</p>
<ul>
    <li>[at the top] the toolbar, which gathers all the main operations that can be performed on the database,</li>
    <li>[in the middle] the main display area, which allows displaying or editing backgammon positions,</li>
    <li>[at the bottom] the status bar, which integrates the command line and presents various information about the current position.</li>
</ul>
<p>Panels can be displayed to:</p>
<ul>
    <li>display analysis data associated with the current position (from XG, GNUbg, or BGBlitz),</li>
    <li>display, add, or modify comments,</li>
    <li>browse imported matches and navigate through their moves (Match panel),</li>
    <li>manage collections of positions (Collection panel),</li>
    <li>study positions with spaced repetition (Anki panel),</li>
    <li>manage tournaments (Tournament panel),</li>
    <li>display performance statistics (Stats panel),</li>
    <li>evaluate any position with the built-in engine, and compute the EPC of a bearoff position (Eval panel),</li>
    <li>browse saved search filters (Filter Library panel),</li>
    <li>browse search history (Search History panel).</li>
</ul>
<p>The main display area provides the user with:</p>
<ul>
    <li>a board to display or edit a backgammon position,</li>
    <li>the level and owner of the cube,</li>
    <li>the race count of each player,</li>
    <li>the score of each player,</li>
    <li>the dice to be played. If no value is displayed on the dice, the position of the dice indicates which player has the turn and that the position is a cube decision.</li>
</ul>
<p>The status bar displays from left to right:</p>
<ul>
    <li>the command line (press <strong>Space</strong> to open),</li>
    <li>an information message related to the last operation performed,</li>
    <li>the index of the current position, followed by the total number of positions (or move/game info when navigating a match).</li>
</ul>
<p>In the case of positions resulting from a user search, the number of positions indicated in the status bar corresponds to the number of filtered positions.</p>

<h3>Browsing Positions</h3>
<p>By default, blunderDB allows you to:</p>
<ul>
    <li>scroll through the different positions in the current library,</li>
    <li>display analysis information associated with a position,</li>
    <li>display, add, and modify comments on a position.</li>
</ul>

<h3>Editing Positions</h3>
<p>
    Pressing the <strong>Tab</strong> key opens the search panel and allows editing a position on the board to add it to the database or to define a position structure to search for. The distribution
    of checkers, the cube, the score, and the turn can be modified using the mouse.
</p>

<h3>Command Line</h3>
<p>
    The command line, integrated in the status bar, allows performing all the functionalities of blunderDB: database operations, position navigation, displaying analysis and comments, searching for
    positions with filters... After getting familiar with the interface, it is recommended to gradually use the command line, which allows powerful and smooth use of blunderDB, especially for position
    search functionalities.
</p>
<p>
    To open the command line, press the <strong>Space</strong> key. A prompt appears in the status bar. Type your command and press <strong>Enter</strong> to execute. Press
    <strong>Escape</strong>
    to cancel.
</p>
<p>blunderDB executes the queries sent by the user provided they are valid and immediately modifies the state of the database if necessary. There are no explicit save actions required by the user.</p>
<p>
    To refine a search within previously filtered positions, use the <strong>ss</strong> command followed by filters (e.g., <strong>ss nc</strong>). This restricts the search to only the positions
    currently displayed, allowing progressive narrowing of results. The search panel (<strong>Ctrl+F</strong>) also offers a "Search in current results" checkbox for the same functionality.
</p>

<h3>Eval Panel</h3>
<p>
    The <strong>Eval</strong> panel evaluates whatever position sits on the board: winning, gammon and backgammon probabilities, equity, ranked candidate moves, and the one decision the position calls
    for — play a move, or double. The computation is done by gammonNet, built in: neither eXtreme Gammon nor GNU Backgammon is required.
</p>
<p>
    To open it, press <strong>Ctrl+E</strong>, click the Eval tab in the bottom panel, or type <strong>epc</strong> in the command line. The board opens on a standard bearoff configuration (15
    checkers), unless a position from the database was sent to it. Checkers are freely added and removed with the mouse; the evaluation follows every change.
</p>
<p>On a bearoff position the panel <strong>specialises</strong>: a second table, per player, carries the EPC (Effective Pip Count) computed from GNUbg's one-sided 6-point bearoff database —</p>
<ul>
    <li><strong>EPC</strong>: the average number of pips needed to bear off all checkers,</li>
    <li><strong>Pip Count</strong>: the raw pip count,</li>
    <li><strong>Wastage</strong>: the difference between EPC and pip count,</li>
    <li><strong>Avg Rolls</strong>: the average number of rolls to bear off all checkers,</li>
    <li><strong>Std Dev</strong>: the standard deviation of that number of rolls.</li>
</ul>
<p>When both players have checkers in their home board, a comparison section shows the EPC and pip count differences.</p>
<p>
    On a pure race, a further table shows both players' winning probabilities and, when the position is covered by a two-sided database (a 6-checker-per-player table computed on first launch, an
    extended 11-checker table computed from the Bearoff tab of the configuration), the exact money equities and the best cube decision. Outside that domain the winning probability is estimated (an
    "estimated" badge with its error margin) and no decision is shown. The player on roll is edited by clicking a player's off/score rectangle, the cube position by clicking the cube on the board.
</p>
<p>
    The <strong>Challenge</strong> checkbox hides the results on every change to the position; click an area to reveal it — ideal for practising an equity, an EPC or a cube decision before checking.
</p>
<p>To close the Eval panel, press <strong>Ctrl+E</strong> again or switch to another tab.</p>

<h3>Match Navigation</h3>
<p>
    blunderDB allows browsing through the moves of imported matches. Open the Match panel with <strong>Ctrl+Tab</strong> and double-click a match (or press <strong>Enter</strong>) to load its
    positions.
</p>
<p>
    When navigating a match, the last visited position is automatically saved and restored. Use the <strong>Left</strong>/<strong>Right</strong> keys to move between positions, and
    <strong>PageUp</strong>/<strong>PageDown</strong> to jump between games.
</p>
<p>The analysis panel (<strong>Ctrl+L</strong>) shows the analysis for each move, with the played move highlighted. Press <strong>d</strong> to toggle between checker and cube analysis.</p>

<h3>Collections</h3>
<p>
    Collections allow organizing positions into custom groups. Open the Collection panel with <strong>Ctrl+B</strong>, then double-click a collection to browse its positions. Collections and positions
    within them can be reordered by drag-and-drop.
</p>

<h3>Anki (Spaced Repetition)</h3>
<p>The Anki panel (<strong>Ctrl+K</strong>) provides spaced repetition for studying backgammon positions using the FSRS algorithm.</p>
<p>
    <strong>Creating decks:</strong> Click <em>New Deck</em> to create a deck from a collection or from the current search results. Search-based decks automatically sync when the Anki tab is
    activated.
</p>
<p>
    <strong>Reviewing:</strong> Select a deck and click <em>Study</em> (or double-click a deck) to start reviewing due cards. Each card displays the corresponding position on the board. Rate your
    recall with keys <strong>1</strong> (Again), <strong>2</strong> (Hard), <strong>3</strong> (Good), or <strong>4</strong> (Easy). Press <strong>Esc</strong> to stop and return to the deck list.
</p>
<p>
    <strong>Limiting the session:</strong> In the deck settings you can bound a sitting to a number of cards. The session then stops and says so, and free drill remains available to carry on without
    touching the schedule. A limit of <em>0</em> serves no card — which is not the same as no limit.
</p>
<p>
    <strong>Retention:</strong> Desired retention is your choice on the workload/quality trade-off. The settings show the <em>measured</em> retention from your own reviews beside it — information,
    never a control. Changing the target is not retroactive: each card adopts the new rhythm at its next review.
</p>
<p>
    <strong>Showing the answer:</strong> The card asks a question; think it through, then press <strong>Space</strong> (or click the masked area) to reveal the position's stored analysis. It appears
    below the grading buttons, which stay within reach. You are never required to reveal it in order to grade, and it re-hides on the next card — not when you simply switch tabs.
</p>
<p>
    <strong>Stop/Resume:</strong> You can stop a review session at any time by pressing <strong>Esc</strong>. The button changes to <em>Resume</em> showing your progress. Click it to continue where
    you left off.
</p>
<p>
    <strong>Deck management:</strong> Use the action buttons to rename, sync, reset, or delete decks. FSRS parameters (retention target, max interval, fuzz) can be configured per deck in Settings
    (gear icon).
</p>

<h3>Tournaments</h3>
<p>
    Tournaments allow grouping matches by event. On import, a match enters the tournament its file names, created if needed; a match already sorted is never moved. Open the Tournament panel with
    <strong>Ctrl+Y</strong> to manage tournaments and assign matches to them.
</p>

<h3>Stats</h3>
<p>
    The Stats panel (<strong>Ctrl+D</strong>) displays performance statistics (PR and MWC cost) computed from all imported positions. Use the filter bar to restrict the analysis by player, tournament,
    date range, decision type, or match length. Click any indicator to drill down into the corresponding positions. The <strong>Players</strong> tab lists, per player, the number of matches, the
    record, decisions, PR (checker and cube), Snowie, blunders, and luck measured over the known rolls.
</p>

<h3>Watermark and protected export</h3>
<p>When exporting (<strong>export_db</strong>, or the Export dialog), two independent protections can be turned on freely, one, the other, or both together:</p>
<ul>
    <li>
        <strong>Watermark:</strong> marks the exported file with its origin (who produced it, an optional note). The watermark is signed with your issuer identity: it cannot be altered or forged in
        someone else's name — but it is not unremovable and prevents no copy.
    </li>
    <li>
        <strong>Password:</strong> places the export in an encrypted <strong>.dbx</strong> container. It protects the file while it travels, not the database itself — whoever you give the password to
        can open it — and the origin stays readable without it.
    </li>
</ul>
<p>
    Your issuer identity, the key that signs your watermarks, is created automatically on the first export marked with its origin. View it, export it, or regenerate it from the
    <strong>Issuer identity</strong> tab of the settings.
</p>
`,
    shortcuts: `
<h3>Database</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-N</td>
<td>Create a new database.</td>
</tr>
<tr>
<td>CTRL-O</td>
<td>Open an existing database.</td>
</tr>
<tr>
<td>CTRL-SHIFT-I</td>
<td>Import a database.</td>
</tr>
<tr>
<td>CTRL-SHIFT-S</td>
<td>Export the database.</td>
</tr>
<tr>
<td>CTRL-Q</td>
<td>Close blunderDB.</td>
</tr>
<tr>
<td>CTRL-M</td>
<td>Edit database metadata.</td>
</tr>
</tbody>
</table>
<h3>Position</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-I</td>
<td>Import one or more positions/matches from file (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>CTRL-SHIFT-F</td>
<td>Recursively import a folder of match/position files.</td>
</tr>
<tr>
<td>CTRL-C</td>
<td>Copy a position to the clipboard.</td>
</tr>
<tr>
<td>CTRL-X</td>
<td>Copy board image to clipboard (PNG).</td>
</tr>
<tr>
<td>CTRL-X CTRL-X</td>
<td>Copy board + analysis image to clipboard (PNG).</td>
</tr>
<tr>
<td>CTRL-V</td>
<td>Paste a position from the clipboard (automatic format detection).</td>
</tr>
<tr>
<td>CTRL-S</td>
<td>Save a position.</td>
</tr>
<tr>
<td>CTRL-U</td>
<td>Update a position.</td>
</tr>
<tr>
<td>Del</td>
<td>Delete the current position (confirmation required).</td>
</tr>
<tr>
<td>BACKSPACE</td>
<td>Reset board, cube, score, and dice.</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>Show the position metadata.</td>
</tr>
</tbody>
</table>
<h3>Navigation</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-R</td>
<td>Reload all the positions from the database.</td>
</tr>
<tr>
<td>PageUp, h</td>
<td>First position / Previous game (match navigation).</td>
</tr>
<tr>
<td>LEFT, k</td>
<td>Previous position.</td>
</tr>
<tr>
<td>RIGHT, j</td>
<td>Next position.</td>
</tr>
<tr>
<td>UP, k</td>
<td>Previous move (when a move is selected in the analysis).</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>Next move (when a move is selected in the analysis).</td>
</tr>
<tr>
<td>PageDown, l</td>
<td>Last position / Next game (match navigation).</td>
</tr>
<tr>
<td>r</td>
<td>Load a random position.</td>
</tr>
</tbody>
</table>
<h3>Display</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-LEFT</td>
<td>Board orientation to the left.</td>
</tr>
<tr>
<td>CTRL-RIGHT</td>
<td>Board orientation to the right.</td>
</tr>
<tr>
<td>p</td>
<td>Show/hide pipcount.</td>
</tr>
</tbody>
</table>
<h3>Actions</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>TAB</td>
<td>Open the search panel (position editor).</td>
</tr>
<tr>
<td>SPACE</td>
<td>Open the command line.</td>
</tr>
</tbody>
</table>
<h3>Tools</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-L</td>
<td>Show/hide the analysis.</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>Show/hide the comments.</td>
</tr>
<tr>
<td>CTRL-K</td>
<td>Show/hide the Anki panel (spaced repetition).</td>
</tr>
<tr>
<td>CTRL-F</td>
<td>Show/hide the search panel.</td>
</tr>
<tr>
<td>CTRL-Tab</td>
<td>Show/hide the match panel.</td>
</tr>
<tr>
<td>CTRL-B</td>
<td>Show/hide the collections panel.</td>
</tr>
<tr>
<td>CTRL-Y</td>
<td>Show/hide the tournaments panel.</td>
</tr>
<tr>
<td>CTRL-D</td>
<td>Show/hide the Stats panel.</td>
</tr>
<tr>
<td>CTRL-E</td>
<td>Show/hide the Eval panel.</td>
</tr>
<tr>
<td>?</td>
<td>Show/hide the help.</td>
</tr>
</tbody>
</table>
<h3>View Tabs</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-T</td>
<td>Create a new view (a copy of the current view).</td>
</tr>
<tr>
<td>CTRL-W</td>
<td>Close the current view.</td>
</tr>
<tr>
<td>CTRL-PageUp, SHIFT-J</td>
<td>Previous view.</td>
</tr>
<tr>
<td>CTRL-PageDown, SHIFT-K</td>
<td>Next view.</td>
</tr>
<tr>
<td>CTRL-1 … CTRL-9</td>
<td>Go directly to the n-th view.</td>
</tr>
<tr>
<td>Double-click the tab</td>
<td>Rename the view.</td>
</tr>
</tbody>
</table>
<h3>Command Line</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>UP</td>
<td>Browse command history up.</td>
</tr>
<tr>
<td>DOWN</td>
<td>Browse command history down.</td>
</tr>
</tbody>
</table>
<h3>Search History</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Click</td>
<td>Select/deselect a search (show position).</td>
</tr>
<tr>
<td>Double-click</td>
<td>Execute the search.</td>
</tr>
</tbody>
</table>
<h3>Filter Library</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Click</td>
<td>Select/deselect a filter (show position).</td>
</tr>
<tr>
<td>Double-click</td>
<td>Execute the filter search.</td>
</tr>
</tbody>
</table>
<h3>Analysis Panel</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Click</td>
<td>Select/deselect a move (show/hide arrows).</td>
</tr>
<tr>
<td>UP, k</td>
<td>Select previous move (when a move is selected).</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>Select next move (when a move is selected).</td>
</tr>
<tr>
<td>d</td>
<td>Toggle between checker and cube analysis (match navigation only).</td>
</tr>
<tr>
<td>Esc</td>
<td>Deselect move. If no move selected, close the panel.</td>
</tr>
</tbody>
</table>
<h3>Eval Panel</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Click</td>
<td>Select/deselect a move (show/hide arrows).</td>
</tr>
<tr>
<td>UP, k</td>
<td>Select previous move (when a move is selected).</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>Select next move (when a move is selected).</td>
</tr>
<tr>
<td>Esc</td>
<td>Deselect move.</td>
</tr>
</tbody>
</table>
<h3>Match Panel</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Click</td>
<td>Select a match.</td>
</tr>
<tr>
<td>Double-click</td>
<td>Navigate the match.</td>
</tr>
<tr>
<td>UP, k</td>
<td>Select previous match.</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>Select next match.</td>
</tr>
<tr>
<td>ENTER</td>
<td>Load the selected match.</td>
</tr>
<tr>
<td>Del</td>
<td>Delete the selected match.</td>
</tr>
<tr>
<td>Esc</td>
<td>Deselect/close the panel.</td>
</tr>
</tbody>
</table>
<h3>Anki panel (spaced repetition)</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>SPACE, Click</td>
<td>Show the answer (the recorded analysis of the position).</td>
</tr>
<tr>
<td>1</td>
<td>Rate: Again (failed, review soon).</td>
</tr>
<tr>
<td>2</td>
<td>Rate: Hard.</td>
</tr>
<tr>
<td>3</td>
<td>Rate: Good.</td>
</tr>
<tr>
<td>4</td>
<td>Rate: Easy.</td>
</tr>
<tr>
<td>p</td>
<td>Show/hide the pip count (same as the general shortcut, available during review).</td>
</tr>
<tr>
<td>Esc</td>
<td>Stop the review and return to the deck list (can resume later).</td>
</tr>
</tbody>
</table>
<h3>Tournament Panel</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Click, Double-click</td>
<td>Select a tournament (show its detail).</td>
</tr>
<tr>
<td>UP, k</td>
<td>Select previous tournament.</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>Select next tournament.</td>
</tr>
<tr>
<td>Double-click (on a match of the tournament)</td>
<td>Navigate the match.</td>
</tr>
<tr>
<td>Esc</td>
<td>Cancel the current edit, otherwise clear the add-match search, otherwise deselect the tournament, otherwise close the panel (one step at a time).</td>
</tr>
</tbody>
</table>
<h3>Collection Panel</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Click</td>
<td>Add/remove the current position to/from the hovered collection.</td>
</tr>
<tr>
<td>Double-click</td>
<td>Open the collection.</td>
</tr>
<tr>
<td>Del</td>
<td>Remove the current position (or the checked positions) from the open collection.</td>
</tr>
<tr>
<td>Esc</td>
<td>Go back to the collection list, otherwise deselect the collection, otherwise close the panel (one step at a time).</td>
</tr>
</tbody>
</table>
`,
    commands: `
<p>The command line, located in the status bar, opens by pressing the <em>SPACE</em> key. As you type a command, a list of suggestions appears automatically: the <em>TAB</em> key (or <em>SHIFT-TAB</em>) cycles through the suggestions and completes the command, while <em>ESC</em> closes the list (a second <em>ESC</em> closes the command line). The <em>UP</em> and <em>DOWN</em> keys remain reserved for the command history.</p>
<h3>Global operations</h3>
<table>
<thead>
<tr>
<th>Command</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>new, ne, n</td>
<td>Create a new database.</td>
</tr>
<tr>
<td>open, op, o</td>
<td>Open an existing database.</td>
</tr>
<tr>
<td>import_db, idb</td>
<td>Import and merge another database.</td>
</tr>
<tr>
<td>export_db, edb</td>
<td>Export the current selection to a new database.</td>
</tr>
<tr>
<td>quit, q</td>
<td>Close blunderDB.</td>
</tr>
<tr>
<td>help, he, h</td>
<td>Open blunderDB help.</td>
</tr>
<tr>
<td>tutorial, tour</td>
<td>Opens the catalogue of guided interface tours.</td>
</tr>
<tr>
<td>demo</td>
<td>Loads a sample database (matches, tournament, collections, comments, Anki deck, analyses) to explore the tool.</td>
</tr>
<tr>
<td>meta</td>
<td>Display database metadata.</td>
</tr>
<tr>
<td>epc</td>
<td>Opens the Eval panel (Effective Pip Count, winning probability and cube verdict in bearoff).</td>
</tr>
<tr>
<td>met</td>
<td>Open the Kazaross-XG2 match equity table.</td>
</tr>
<tr>
<td>tp2</td>
<td>Open the takepoint table with a 2-cube.</td>
</tr>
<tr>
<td>tp2_live</td>
<td>Open the takepoint table with a 2-cube for long race positions.</td>
</tr>
<tr>
<td>tp2_last</td>
<td>Open the takepoint table with a 2-cube for last roll positions.</td>
</tr>
<tr>
<td>tp4</td>
<td>Open the takepoint table with a 4-cube.</td>
</tr>
<tr>
<td>tp4_live</td>
<td>Open the takepoint table with a 4-cube for long race positions.</td>
</tr>
<tr>
<td>tp4_last</td>
<td>Open the takepoint table with a 4-cube for last roll positions.</td>
</tr>
<tr>
<td>gv1</td>
<td>Open the gammon value table with a 1-cube.</td>
</tr>
<tr>
<td>gv2</td>
<td>Open the gammon value table with a 2-cube.</td>
</tr>
<tr>
<td>gv4</td>
<td>Open the gammon value table with a 4-cube.</td>
</tr>
</tbody>
</table>
<h3>Positions and navigation</h3>
<table>
<thead>
<tr>
<th>Command</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>import, i</td>
<td>Import one or more positions/matches from file (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Deletes the current position (confirmation required).</td>
</tr>
<tr>
<td>[number]</td>
<td>Go to the specified index position.</td>
</tr>
<tr>
<td>list, l</td>
<td>Show the analysis of the current position.</td>
</tr>
<tr>
<td>comment, co</td>
<td>Show/write comments.</td>
</tr>
<tr>
<td>history, hi</td>
<td>Open the search panel (the search history is in its <em>History</em> tab).</td>
</tr>
<tr>
<td>stats, st</td>
<td>Show/hide the statistics panel.</td>
</tr>
<tr>
<td>match, ma</td>
<td>Show/hide the match panel.</td>
</tr>
<tr>
<td>collection, coll</td>
<td>Show/hide the collections panel.</td>
</tr>
<tr>
<td>#tag1 tag2 ...</td>
<td>Tag the current position.</td>
</tr>
<tr>
<td>e</td>
<td>Load all positions from the database.</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>Load the worst mistakes (equity/MWC) into the analysis view, according to the current statistics filter. An optional number chooses how many to load (<code>bl 50</code>); 10 by default.Load the worst mistakes (equity/MWC) into the analysis view, according to the current statistics filter.</td>
</tr>
<tr>
<td>m</td>
<td>Navigate to the last visited match.</td>
</tr>
</tbody>
</table>
<h3>Editing and search</h3>
<table>
<thead>
<tr>
<th>Command</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>write, wr, w</td>
<td>Save the current position.</td>
</tr>
<tr>
<td>write!, wr!, w!</td>
<td>Update the current position.</td>
</tr>
<tr>
<td>s</td>
<td>Search for positions with filters.</td>
</tr>
<tr>
<td>ss</td>
<td>Search among the currently filtered positions.</td>
</tr>
</tbody>
</table>
<h3>Search Filters</h3>
<table>
<thead>
<tr>
<th>Query</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>The position checks the cube configuration.</td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>The position checks the score.</td>
</tr>
<tr>
<td>d</td>
<td>The position checks the dice or the cube decision.</td>
</tr>
<tr>
<td>D</td>
<td>The position matches the dice roll (both dice, any order).</td>
</tr>
<tr>
<td>D1</td>
<td>The position matches the dice roll on the first die only (the first die's value appears on either die of the position).</td>
</tr>
<tr>
<td>xD65</td>
<td>The position was <strong>not</strong> played with the 6-5 roll (any order). The value is given in the token; repeatable to exclude several rolls (<code>xD65 xD54</code>).</td>
</tr>
<tr>
<td>nc</td>
<td>The position has no contact.</td>
</tr>
<tr>
<td>ph:race</td>
<td>The position is in a given phase of the game: <code>opening</code>, <code>middlegame</code>, <code>race</code> or <code>bearoff</code>. Repeatable (<code>ph:race ph:bearoff</code>). The label is derived from the board and never editable; <code>blunderdb repair</code> recomputes it.</td>
</tr>
<tr>
<td>M</td>
<td>The position or the mirror one meets the filters.</td>
</tr>
<tr>
<td>i</td>
<td>The position was imported on its own, not brought in by a match import.</td>
</tr>
<tr>
<td>fl</td>
<td>The position was flagged in the source software, when importing an eXtreme Gammon match.</td>
</tr>
<tr>
<td>x</td>
<td>The position contains none of the checkers of the exclusion structure (the "Except" tab of the search panel).</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>The player has at least x pips behind in the race.</td>
</tr>
<tr>
<td>p&lt;x</td>
<td>The player has at most x pips behind in the race.</td>
</tr>
<tr>
<td>px,y</td>
<td>The player has between x and y pips behind in the race.</td>
</tr>
<tr>
<td>P&gt;x</td>
<td>The player has a race of at least x pips.</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>The player has a race of at most x pips.</td>
</tr>
<tr>
<td>Px,y</td>
<td>The player has a race between x and y pips.</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>The equity (in millipoints) of the position is greater than x.</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>The equity (in millipoints) of the position is less than x.</td>
</tr>
<tr>
<td>ex,y</td>
<td>The equity (in millipoints) of the position is between x and y.</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>The error of the move played by player 1 (in millipoints) is greater than x.</td>
</tr>
<tr>
<td>E&lt;x</td>
<td>The error of the move played by player 1 (in millipoints) is less than x.</td>
</tr>
<tr>
<td>Ex,y</td>
<td>The error of the move played by player 1 (in millipoints) is between x and y.</td>
</tr>
<tr>
<td>w&gt;x</td>
<td>The player has winning chances greater than x%.</td>
</tr>
<tr>
<td>w&lt;x</td>
<td>The player has winning chances less than x%.</td>
</tr>
<tr>
<td>wx,y</td>
<td>The player has winning chances between x% and y%.</td>
</tr>
<tr>
<td>g&gt;x</td>
<td>The player has gammon chances greater than x%.</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>The player has gammon chances less than x%.</td>
</tr>
<tr>
<td>gx,y</td>
<td>The player has gammon chances between x% and y%.</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>The player has backgammon chances greater than x%.</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>The player has backgammon chances less than x%.</td>
</tr>
<tr>
<td>bx,y</td>
<td>The player has backgammon chances between x% and y%.</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>The opponent has winning chances greater than x%.</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>The opponent has winning chances less than x%.</td>
</tr>
<tr>
<td>Wx,y</td>
<td>The opponent has winning chances between x% and y%.</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>The opponent has gammon chances greater than x%.</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>The opponent has gammon chances less than x%.</td>
</tr>
<tr>
<td>Gx,y</td>
<td>The opponent has gammon chances between x% and y%.</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>The opponent has backgammon chances greater than x%.</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>The opponent has backgammon chances less than x%.</td>
</tr>
<tr>
<td>Bx,y</td>
<td>The opponent has backgammon chances between x% and y%.</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>The player has at least x checkers off.</td>
</tr>
<tr>
<td>o&lt;x</td>
<td>The player has at most x checkers off.</td>
</tr>
<tr>
<td>ox,y</td>
<td>The player has between x and y checkers off.</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>The opponent has at least x checkers off.</td>
</tr>
<tr>
<td>O&lt;x</td>
<td>The opponent has at most x checkers off.</td>
</tr>
<tr>
<td>Ox,y</td>
<td>The opponent has between x and y checkers off.</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>The player has at least x backcheckers.</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>The player has at most x backcheckers.</td>
</tr>
<tr>
<td>kx,y</td>
<td>The player has between x and y backcheckers.</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>The opponent has at least x backcheckers.</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>The opponent has at most x backcheckers.</td>
</tr>
<tr>
<td>Kx,y</td>
<td>The opponent has between x and y backcheckers.</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>The player has at least x checkers in the zone.</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>The player has at most x checkers in the zone.</td>
</tr>
<tr>
<td>zx,y</td>
<td>The player has between x and y checkers in the zone.</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>The opponent has at least x checkers in the zone.</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>The opponent has at most x checkers in the zone.</td>
</tr>
<tr>
<td>Zx,y</td>
<td>The opponent has between x and y checkers in the zone.</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>The player has at least x blots in the outfield.</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>The player has at most x blots in the outfield.</td>
</tr>
<tr>
<td>box,y</td>
<td>The player has between x and y blots in the outfield.</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>The opponent has at least x blots in the outfield.</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>The opponent has at most x blots in the outfield.</td>
</tr>
<tr>
<td>BOx,y</td>
<td>The opponent has between x and y blots in the outfield.</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>The player has at least x blots in the jan.</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>The player has at most x blots in the jan.</td>
</tr>
<tr>
<td>bjx,y</td>
<td>The player has between x and y blots in the jan.</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>The opponent has at least x blots in the jan.</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>The opponent has at most x blots in the jan.</td>
</tr>
<tr>
<td>BJx,y</td>
<td>The opponent has between x and y blots in the jan.</td>
</tr>
<tr>
<td><code>t'word1;word2;...'</code></td>
<td>The position comments contain at least one of the words.</td>
</tr>
<tr>
<td>co</td>
<td>The position carries a comment, whatever its content.</td>
</tr>
<tr>
<td>xco</td>
<td>The position carries no comment.</td>
</tr>
<tr>
<td>co:user</td>
<td>The position carries a comment of a given origin: <code>user</code> (written by you), <code>xg</code>, <code>gnubg</code>, <code>bgf</code> (brought in by a match import) or <code>unknown</code>. Repeatable (<code>co:xg co:gnubg</code>).</td>
</tr>
<tr>
<td><code>m'pattern1,pattern2,...'</code></td>
<td>The best checker moves containing at least one of the patterns.</td>
</tr>
<tr>
<td><code>m'ND,DT,DP,...'</code></td>
<td>The best cube decisions for No Double/Take, Double Take, Double Pass.</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Date of position addition after x (YYYY/MM/DD).</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Date of position addition before x (YYYY/MM/DD).</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Date of position addition between x and y (YYYY/MM/DD).</td>
</tr>
<tr>
<td>max</td>
<td>Search in match with ID x (e.g. ma3).</td>
</tr>
<tr>
<td>max,y</td>
<td>Search in matches with IDs from x to y (e.g. ma2,5).</td>
</tr>
<tr>
<td>tnx</td>
<td>Search in tournament with ID x (e.g. tn1).</td>
</tr>
<tr>
<td>tnx,y</td>
<td>Search in tournaments with IDs from x to y (e.g. tn1,3).</td>
</tr>
<tr>
<td>idx</td>
<td>Search for the position with identifier x (e.g. id12).</td>
</tr>
<tr>
<td>idx,y</td>
<td>Search for the positions with identifiers x to y (e.g. id5,10).</td>
</tr>
<tr>
<td><code>pl'name'</code></td>
<td>Search positions from a match involving the named player, at either seat (e.g. <code>pl'Alice'</code>). Case-insensitive.</td>
</tr>
</tbody>
</table>
<h3>Various commands</h3>
<table>
<thead>
<tr>
<th>Command</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>clear, cl</td>
<td>Clear the command history.</td>
</tr>
</tbody>
</table>
`,
    about: `
<h3>Version</h3>
<p>Application version: {appVersion}</p>
<p>Database version: {dbVersion}</p>
<p>
    <a href="https://kevung.github.io/blunderDB/en/" target="_blank" rel="noopener noreferrer">Online documentation</a> ·
    <a href="https://kevung.github.io/blunderDB/en/historique.html" target="_blank" rel="noopener noreferrer">Version history</a>
</p>

<h3>Author</h3>
<p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
<p>You can also find me on Heroes under the nickname <strong>postmanpat</strong>.</p>
<p>
    I developed blunderDB initially for my personal use to detect patterns in my mistakes. But it is very pleasant to have feedback, especially when a lot of hours have been spent on design, coding,
    debugging... So feel free to write to me to share your feedback.
</p>
<p>Here are several ways to reach out:</p>
<ul>
    <li>Join the blunderDB Discord server: <a href="https://discord.gg/DA5PpzM9En" target="_blank" rel="noopener noreferrer">discord.gg/DA5PpzM9En</a>,</li>
    <li>Discuss with me if we meet in a tournament,</li>
    <li>Send me an email,</li>
</ul>
<h3>License</h3>
<p>
    blunderDB is licensed under the MIT License. This means you are free to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the software, provided that the original
    copyright notice and this permission notice are included in all copies or substantial portions of the software.
</p>
<h3>Acknowledgements</h3>
<p>I dedicate this small software to my partner <strong>Anne-Claire</strong> and our dear daughter <strong>Perrine</strong>. I would like to especially thank some friends:</p>
<ul>
    <li>
        <strong>Tristan Remille</strong>, for introducing me to backgammon with joy and kindness; for showing the Way in understanding this wonderful game; for continuing to support me despite my poor
        attempts to play better.
    </li>
    <li><strong>Nicolas Harmand</strong>, a joyful companion for over a decade in great adventures, and a fantastic game partner since he caught the backgammon bug.</li>
</ul>
<h3>Credits</h3>
<p>blunderDB embeds code, data and fonts written by other people. The essentials:</p>
<ul>
    <li>
        The <strong>strehl-prob5-512-512-256-128</strong> neural network is the work of <strong>Alexander Strehl</strong> (<em>alexstrehl/backgammon-ai-engine</em>, MIT). The search, cube model and
        match equity table around it are <strong>gammonNet</strong>'s own configuration (<a href="https://github.com/kevung/gammonNet" target="_blank" rel="noopener noreferrer"
            >github.com/kevung/gammonNet</a
        >, MIT).
    </li>
    <li>The Kazaross-XG2 Match Equity Table (MET) is the work of <strong>Neil Kazaross</strong>.</li>
    <li>The take point and gammon value tables are taken from <strong>Dirk Schiemann</strong>'s book <em>The Theory of Backgammon</em>.</li>
    <li>
        The one-sided (6 points, 15 checkers, for EPC) and two-sided (6 points, 6 checkers, for cube verdicts in races) bearoff databases were generated with <strong>GNU Backgammon</strong> (GNUbg).
        GNUbg is free software under the GPL; these tables are data it produced, credited as such.
    </li>
    <li>Match files are read by <em>xgparser</em>, <em>gnubgparser</em> and <em>bgfparser</em> (MIT).</li>
    <li>On the Go side: <em>modernc.org/sqlite</em> (BSD-3-Clause), <em>pgx</em>, <em>Wails</em> and <em>go-fsrs</em> (MIT).</li>
    <li>On the interface side: <em>Svelte</em>, <em>two.js</em>, <em>Chart.js</em> and <em>driver.js</em> (MIT).</li>
    <li>The <em>Nunito</em> and <em>Noto Sans JP</em> fonts (SIL Open Font License 1.1).</li>
</ul>
<p>
    The full inventory, with the licence texts, is the <strong>THIRD_PARTY.md</strong> file shipped with blunderDB (<a
        href="https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md"
        target="_blank"
        rel="noopener noreferrer"
        >github.com/kevung/blunderDB</a
    >).
</p>
`
};
