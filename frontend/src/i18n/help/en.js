// GENERATED FILE — do not edit by hand, and do not translate it here.
//
// Produced by `go run ./cmd/help-gen` (make help) from:
//   - doc/source/manuel.rst      → the "manual" tab
//   - doc/source/raccourcis.rst  → the "shortcuts" tab
//   - doc/source/cmd_mode.rst    → the "commands" tab
//   - doc/source/locale/<lang>/LC_MESSAGES/*.po for the eight translations
//   - frontend/src/i18n/help/prose/<lang>.html → the "about" tab
//
// Fix the documentation (and its .po catalogues), or the prose fragment, then
// run `make help`. TestHelpBundlesAreCurrent fails if this file is stale.
export default {
    manual: `
<h3>Introduction</h3>
<p>blunderDB is software for creating backgammon position databases. Its main strength is to provide a single place to aggregate positions that a player has encountered (online, in tournaments) and to be able to re-study these positions by filtering them according to various arbitrarily combinable filters. blunderDB can also be used to create catalogs of reference positions.</p>
<p>Positions are stored in a database represented by a <em>.db</em> file. The desktop application opens this file directly, never a network address: server mode (Headless mode (server)) is another mode of the same binary, and moving from one to the other means exporting or migrating the database, not pointing the application at a URL.</p>
<h3>Main Interactions</h3>
<p>The main interactions possible with blunderDB are:</p>
<ul>
<li>add a new position,</li>
<li>modify an existing position,</li>
<li>copy the board image to the clipboard (PNG) via <strong>CTRL-X</strong>, or with the full analysis via <strong>CTRL-X CTRL-X</strong>,</li>
<li>delete an existing position,</li>
<li>search for one or more positions,</li>
<li>import matches from various sources (XG, GNUbg, BGBlitz, Jellyfish), including comments from XG files,</li>
<li>navigate through the moves of an imported match,</li>
<li>organize positions into collections,</li>
<li>organize matches into tournaments.</li>
</ul>
<p>The user can freely tag positions and annotate them with comments.</p>
<h3>Description of the interface</h3>
<p>The interface of blunderDB is composed, from top to bottom, of:</p>
<ul>
<li>[top] the toolbar, which gathers all the main operations that can be performed on the database,</li>
<li>[in the middle] the main display area, which allows for displaying or editing backgammon positions,</li>
<li>[at the bottom] the status bar, which provides various information about the database or the current position, and integrates the command line.</li>
</ul>
<p>Panels can be displayed to:</p>
<ul>
<li>display the analysis data associated with the current position from eXtreme Gammon (XG), GNUbg, or BGBlitz,</li>
<li>display, add, or modify comments,</li>
<li>search and filter positions using combinable criteria,</li>
<li>display and manage position collections (collections panel),</li>
<li>display the list of imported matches and navigate through the moves of a match (match panel),</li>
<li>display and manage tournaments (tournaments panel),</li>
<li>display performance statistics (Stats panel),</li>
<li>compute the EPC (Effective Pip Count) of a bearoff position (Eval panel),</li>
<li>study positions with spaced repetition (Anki panel),</li>
<li>display the database metadata (metadata panel).</li>
</ul>
<p>Modal windows can be displayed to:</p>
<ul>
<li>display the blunderDB help,</li>
<li>display the catalogue of guided tours (see Guided tours and sample database),</li>
<li>configure database export settings,</li>
<li>configure blunderDB, in particular the interface language (see Configuration).</li>
</ul>
<p>The main display area provides the user with:</p>
<ul>
<li>a board to display or edit a backgammon position,</li>
<li>the level and owner of the cube,</li>
<li>the pip count of each player,</li>
<li>the score of each player,</li>
<li>the dice to play. If no values are shown on the dice, the position of the dice indicates which player is on roll and that the position is a cube decision. When the cube decision is a response to a double (take/pass), the offered cube is shown in the centre of the board, at the offered value.</li>
</ul>
<p>A right click on the board opens a context menu offering: evaluate the position as displayed in the Eval panel, evaluate its mirror, copy the board image with its analysis to the clipboard (the equivalent of <em>CTRL-X CTRL-X</em>, harder to discover), <strong>save the image to a file</strong> as SVG or PNG, open a new view on this position, and — if the position already comes from the database — add it to an Anki deck (spaced repetition).</p>
<p>The clipboard is the everyday gesture; saving is the other need — the illustration for an article, a forum post, a lesson. <strong>SVG</strong> is offered because the board is one: it is the form that survives being enlarged, the one you put in a document without blurring it. PNG derives from it, as does the clipboard copy: one rendering, three destinations, so none of them can drift from the others. This menu does not appear in the Eval panel or in the Search panel, where the right button already places the other colour's checkers. See Bringing a position into the Eval panel for bringing a position into the Eval panel.</p>
<p>The status bar is structured from left to right with the following information:</p>
<ul>
<li>the command line, accessible by pressing the <em>SPACE</em> key,</li>
<li>an informational message related to an operation performed by the user,</li>
<li>the index of the current position, followed by the number of positions in the current library (or move/game info when navigating a match),</li>
<li>the <strong>library counter</strong> — “412 positions · 38 blunders · 5 matches” — where every number <strong>opens what it counts</strong>: the positions, the <code>E&gt;100</code> search prepared in the command line, or the match list. A figure you cannot follow is a decoration. The blunder threshold is the statistics' own, one hundred millipoints: two thresholds would make the same word mean two things.</li>
</ul>
<div class="admonition note">
<p>In the case of positions resulting from a user search, the number of positions indicated in the status bar corresponds to the number of filtered positions.</p>
</div>
<p>The <strong>Anki</strong> tab carries a <strong>badge</strong> when cards are due, across every deck. That figure is the reason to open the tab; it has no business behind it. Zero shows nothing: a badge saying “0” is noise.</p>
<p>The <code>log</code> command opens the <strong>activity log</strong>: the last two hundred lines of the log file, a button to copy them — what it takes to attach a report to a bug — and another to open the folder holding them. The log is neither filtered nor reformatted: a log you tidy up is a log you can no longer quote.</p>
<p>In the <strong>search history</strong> of the Search panel, each token of a saved command shows as a named chip — <em>No Contact</em>, <em>Move Error</em> — rather than a bare token. The exact command stays in the tooltip, since that is what gets re-run; and a token blunderDB does not recognise shows <strong>as it is</strong> rather than translated to the nearest thing.</p>
<h3>View Tabs</h3>
<p>Below the toolbar, a tab bar lets you work with several <strong>views</strong> in parallel. Each view is an independent workspace that keeps its own position list, the index of the current position, the displayed position, the analysis and the selected move, the active panel, the comment being edited, as well as the navigation context within a match. This makes it possible, for example, to keep a search open in one view while browsing a match in another.</p>
<ul>
<li><strong>Create a view</strong>: click the <em>+</em> button of the tab bar or press <em>CTRL-T</em>. The new view starts as a copy of the current view.</li>
<li><strong>Close a view</strong>: click the cross on the tab or press <em>CTRL-W</em>. The last view cannot be closed.</li>
<li><strong>Switch view</strong>: click a tab, press <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (or <em>SHIFT-J</em> / <em>SHIFT-K</em>) to move to the previous / next view, or <em>CTRL-1</em> to <em>CTRL-9</em> to jump directly to the n-th view.</li>
<li><strong>Rename a view</strong>: double-click the tab, type the new name and confirm with <em>ENTER</em>.</li>
</ul>
<p>Views are saved with the database session state and restored when it is reopened.</p>
<h3>Configuration</h3>
<p>The settings button (gear icon) in the toolbar, to the left of the help button, opens blunderDB's settings window. It is organised in six tabs:</p>
<ul>
<li><strong>Interface</strong> — language, display scale, panel position;</li>
<li><strong>Colours</strong> — the board's colours;</li>
<li><strong>Bearoff</strong> — the bearoff tables used by the Eval panel;</li>
<li><strong>gammonNet</strong> — the settings of the embedded evaluator, described below;</li>
<li><strong>Watched folder</strong> — the automatic import of matches arriving in a folder, described below;</li>
<li><strong>Issuer identity</strong> — the key that signs your watermarks, described in Handing out a database: origin and password.</li>
</ul>
<p>The <em>Interface</em> tab starts with a <strong>theme</strong>: <em>follow the system</em>, <em>light</em>, <em>dark</em>, <em>high contrast</em> or <em>printable</em>. The theme sets the interface colours and <strong>proposes a board palette</strong> — a dark interface around a light board is not a dark theme, it is half of one, since the board occupies most of the window.</p>
<p>You keep the last word, and the mechanism guarantees it rather than promising it: the <em>Colours</em> tab still sets the board directly, and a colour chosen after the theme is yours. At start-up only the interface tokens are applied, never the board palette — the one you set is already loaded, and rewriting it at every launch would erase your work one session at a time. See <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Follow the system</em> is the default: it obeys the desktop's light/dark preference, including when it changes mid-session. A tool does not impose its light or its dark on a desktop that has already decided.</p>
<p>The <em>Interface</em> tab also lets you choose the language among English, French, German, Italian, Spanish, Finnish, Japanese, Greek and Russian. The whole interface (toolbar, panels, messages, help) is translated into the selected language. The language choice is saved and kept from one session to the next.</p>
<p>The same tab also offers a <strong>Compact database</strong> button, which reclaims the disk space left behind by deletions (matches, tournaments, purges): the database never shrinks by itself when data is deleted, that compaction has to be requested explicitly. The operation can take a while on a large database and temporarily needs about twice its size in free disk space (blunderDB refuses to start rather than risk an interrupted compaction); a confirmation is therefore asked before it runs. The result — the space gained, in megabytes — is then shown in the status bar. The same operation is available on the command line through <code>blunderdb vacuum</code> (see Command Line Interface (CLI)).</p>
<p>The <strong>Open the log folder</strong> button just below it opens the folder holding the application log — useful for attaching details to a bug report, especially when blunderDB was started from a shortcut or a double-click, with no terminal attached to show anything.</p>
<p>The <strong>Check for updates at startup</strong> checkbox, off by default, queries the GitHub repository's releases page once per launch and shows a message in the status bar when a newer version is available — never a window that gets in the way. This check stays automatically disabled on an installation that came through a package manager (Flatpak, Homebrew, a distribution package…): that channel is the one handling updates then, not blunderDB itself.</p>
<p>The <em>Board colours</em> tab lets you customise the board's colours. Each element has its own colour picker: the background, the border, the light and dark points, player 1's and player 2's checkers, the dice, the dice pips and the cube. The <em>Reset</em> button restores all the default colours. Like the language, the chosen colours are kept across sessions.</p>
<p>The <em>Bearoff</em> tab manages the Eval panel's bearoff tables (see Eval Panel). They are <strong>neither embedded in the executable nor downloaded</strong>: blunderDB computes them on the machine that uses them, and the result is identical byte for byte to what gnubg produces — the SHA-256 fingerprint is checked before a table is accepted.</p>
<p>The two ordinary tables (TS-06-06 for the cube verdict, OS-06 for the EPC) are computed on first launch, in the background and without asking: about six seconds on one core, during which the application is used normally. The Eval panel mentions it only if a position is placed there that needs a table which is not ready yet.</p>
<p>The tab shows the active domain and its origin, the state of the one-sided table the EPC reads, the folder where all this lives, and the list of the tables present with their size and their verdict. Each row can be deleted individually, after confirmation.</p>
<p><strong>Verified or unverified.</strong> A <em>verified</em> table has exactly the bytes gnubg produces for its domain: its SHA-256 fingerprint is recorded in blunderDB and was found again. The fingerprints recorded for one-sided tables (OS-06 to OS-10) are the ones produced by GNUbg 1.08's <code>makebearoff</code> tool. An <em>unverified</em> table is well formed but its domain has no recorded fingerprint — nothing is held against it, simply nobody has compared it to the reference. A <em>corrupt</em> table contradicts itself and is never read; it is recomputed.</p>
<p><strong>Computing a wider table.</strong> The domain is picked from a list of two families, together with the number of cores to give it (by default all but one, so the machine stays usable):</p>
<ul>
<li><strong>exact cube (two-sided)</strong>, from TS-06-06 to TS-06-15: widens the domain where the winning probability and the cube verdict are read rather than estimated;</li>
<li><strong>EPC beyond the home board (one-sided)</strong>, from OS-06 to OS-10: widens how far from home a chequer may stand without the EPC block going silent. This sweep reads only positions smaller than the one it is computing, so it is sequential by construction and the core count buys it nothing — the picker says so by greying out.</li>
</ul>
<p>Before anything starts, the tab states three figures for the chosen domain: the size on disk, the memory needed during the computation, and the time it should take <em>on this machine</em>. The last one starts as an estimate and becomes a measurement: every run wide enough records its own speed and keeps it. A domain the available memory cannot hold is offered greyed out, with the reason — "it would need 24 GB, 12 are left" is an answer, an absent row would not be.</p>
<p>As an order of magnitude, on a sixteen-thread machine: TS-06-09 weighs 191 MB and takes about ten seconds, TS-06-11 weighs 1.2 GB and a few minutes, TS-06-13 exceeds what most machines can hold in memory. On the one-sided side, on one core: OS-07 weighs 4.9 MB and takes 17 s, OS-08 15 MB and 1 min 20, OS-10 117 MB and half an hour.</p>
<p><strong>Pause and resume.</strong> During the computation, the progress shows the <em>measured</em> remaining time and two distinct buttons: <em>Pause</em> and <em>Cancel</em>. Pausing writes the state of the computation beside the table; running it again continues where it stopped instead of starting over. Cancelling keeps nothing. Closing the configuration window interrupts nothing — the computation carries on in the background.</p>
<p>A paused computation is found again at the next launch, named and quantified ("TS-06-09 interrupted at 43%"), with <em>Resume</em> and <em>Delete</em>. Nothing restarts on its own: the user is the one who asked it to stop.</p>
<p>The tab finally allows pointing to an external two-sided <code>.bd</code> file, for example a database produced by gnubg itself: the table with the widest domain wins.</p>
<p>The <em>General</em> tab finally carries <strong>Repair the analyses</strong>: the analysis columns that search and statistics query are a projection of the stored analyses, which stay intact. A fault in the projection is therefore repairable without re-importing anything. It is explicit and never automatic — rewriting someone's analysis columns on the mere act of opening their database is not something a tool should do behind their back. The same <code>blunderdb repair</code> is available on the command line.</p>
<p>The <strong>gammonNet</strong> tab configures the embedded evaluator (see <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Two search depths can be set there, named and kept separately — lowering one never changes the other:</p>
<ul>
<li><strong>Display depth</strong> — the interactive comfort while editing the board; never written to the database.</li>
<li><strong>Analysis depth</strong> — what the post-import analysis batch writes into a position's Analysis.</li>
</ul>
<p>Both default to <strong>2-ply</strong>, the canonical configuration. The tab also offers <strong>pruning</strong> (<code>k=12</code> by default) and the <strong>number of candidate moves shown</strong> (10 by default), as well as an <strong>auto-analyze after import</strong> box which, once ticked, checks after every import whether positions <strong>with no analysis at all</strong> remain (neither gammonNet, nor XG, nor GNUbg, nor BGBlitz — the rule is “an evaluation only fills a hole”, never a replacement) and, if so, starts a gammonNet analysis in the background at the configured analysis depth. An <strong>Analyze now</strong> button re-runs the same catch-up manually, useful for a library built before this feature existed.</p>
<p>A second button, <strong>Re-analyze stale positions</strong>, covers the opposite case: a position already analyzed by gammonNet, but whose stored analysis was written by an engine version older than the one currently running, or at a depth different from the analysis depth configured above, is flagged there as stale and re-evaluated. A position that also carries an XG, GNUbg or BGBlitz analysis is never touched by this button, whatever its gammonNet content — <code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__'s protection remains unconditional. The count shown next to each button (positions with no analysis, stale positions) is purely informational; the batch recomputes its own list when it starts.</p>
<p>Both batches are <strong>bounded, visible and cancellable, never a silent daemon</strong>: their progress (<code>positions analysed / total</code>) and a cancel button appear in the status bar for their whole duration, and disappear once finished in favor of a message summarizing the result — how many positions were <strong>analyzed</strong>, how many were <strong>refused</strong> (a position gammonNet declines to evaluate, such as a match score beyond its MET's range, which is never a failure) and how many <strong>failed</strong> (retried, unchanged, on the next run). Closing the application during either loses nothing: each analyzed position is written as it goes, and the next run resumes exactly where the analysis stopped, with no journal to keep.</p>
<p><strong>A match imported without an analysis now gets a Performance Rating.</strong> That is the case of a match played online, or of a Jellyfish <code>.mat</code> file, that nobody ran through XG: blunderDB knew its positions and the moves played, but no analysis said what they were worth. Once the batch has run, the move actually played is compared with gammonNet's ranking and the gap feeds the PR, the error rate, the worst decisions and every other indicator, exactly as for a match analysed by XG. The comparison invents nothing: the played move comes from the match's own move table, written at import whether or not the file carried an analysis.</p>
<p>A database analysed with a version older than this one does not need to be re-evaluated: <code>blunderdb repair</code> recomputes the columns from the analyses and the moves already stored and gives those matches their PR back (see repair).</p>
<p>One honest caveat: a position is identified by its structure, so a position met twice — played well once and badly the other time — carries only one gap, that of its first recorded occurrence. This is not specific to this computation: an XG library has exactly the same shape.</p>
<h4>Watched folder</h4>
<p>The <strong>Watched folder</strong> tab asks blunderDB to look at a folder while it runs and import each match file that <strong>appears</strong> in it. Play a session in eXtreme Gammon, come back to blunderDB, and find the matches already there.</p>
<p>Nothing is guessed. Until a folder is named there is no watch: blunderDB does not start reading a directory because it supposed where your matches live. The <strong>Suggest</strong> button looks at the usual places on this machine and offers one only if it really exists; otherwise it says so, and naming the folder is up to you.</p>
<p>Three things are worth knowing before ticking the box:</p>
<ul>
<li><strong>Only files that appear are imported.</strong> Whatever the folder already holds when the watch starts is recorded as known and left alone: pointing a watch at four years of matches must not import all of them. To import what is there, use the folder import, which exists for that — and the two compose very well, the import first, the watch after.</li>
<li><strong>A file is imported only once its size has settled.</strong> A match another program is writing grows from one glance to the next; importing it half-written would give a parse error nobody can act on. blunderDB therefore waits to see the same file unchanged twice.</li>
<li><strong>The import is silent.</strong> You were studying a position when your matches arrived: taking the screen back from you would be the worst possible moment. The import runs without a window, and the status bar shows a strip giving the count of matches imported, skipped (duplicates) and failed, with a button that opens the full report if you want it. Everything else is identical to a manual import: same duplicates detected, same import batch, same automatic analysis if it is on.</li>
</ul>
<p>The default interval is ten seconds; the floor is two. The folder is not walked recursively: a watched folder is where a tool drops its matches, not a tree to crawl. An unmounted network share does not stop the watch, nor does it make its contents pass for new when it comes back.</p>
<p>The same watch exists on the command line, with <code>blunderdb import --type batch --dir &lt;folder&gt; --watch</code> (see Command Line Interface (CLI)): it is the form a server, a scheduled task or a script can use.</p>
<p>The configuration window also provides interface display settings. An <strong>interface scale</strong> slider lets you enlarge or shrink all interface elements, which is useful on high-density screens or to improve readability. A <strong>panel position</strong> menu sets where the panels (search, matches, analysis) appear relative to the board: <em>bottom</em>, <em>side</em> or <em>automatic</em> (the side is then chosen on wide screens to make better use of the available space). Like the other settings, these choices are kept from one session to the next.</p>
<h3>Guided tours and sample database</h3>
<p>To make getting started easier, blunderDB offers <strong>guided tours</strong> of the interface. The tour catalogue opens from the toolbar or with the <code>tour</code> command (alias <code>tutorial</code>). Seven tours are available: a general tour of the interface, and tours dedicated to searching positions, reviewing matches, reviewing tournaments, the Eval panel, Anki review and statistics. Each tour highlights the relevant interface elements, step by step, opens the panel it talks about along the way, and can be replayed at any time. On first launch, the general tour is offered automatically.</p>
<p>The <code>demo</code> command loads a <strong>sample database</strong> that lets you explore the tool's features without importing your own games: three matches (two of them grouped in a tournament) analysed by eXtreme Gammon, BGBlitz and gammonNet, three thematic collections, tagged comments (<code>#blunder</code>, <code>#cube</code>) and an Anki deck with its review log. The players, the tournament and the venue are fictional. The guided tours rely on this database when no database is open.</p>
<h3>Browsing positions</h3>
<p>By default, blunderDB allows you to:</p>
<ul>
<li>scroll through the various positions of the current library — which is never loaded as a single block: blunderDB only keeps the list of identifiers and loads positions in windows of fifty around the one displayed, so that a database of several tens of thousands of positions opens as fast as a small one,</li>
<li>displaying the analysis information associated with a position.</li>
<li>displaying, adding, and modifying comments on a position.</li>
</ul>
<p>The toolbar's <strong>Go to position</strong> button opens a window where a position's index can be typed in directly to jump to it, without scrolling. It is the graphical equivalent of the <code>[number]</code> command line command (see Positions and navigation).</p>
<div class="admonition tip">
<p>Refer to Keyboard shortcuts for available shortcuts.</p>
</div>
<h3>Editing positions</h3>
<p>Pressing the <em>TAB</em> key opens the search panel and allows editing a position on the board to add it to the database or to define a position structure to search for. The distribution of checkers, the cube, the score, and the turn can be modified using the mouse (see Edit a position).</p>
<div class="admonition tip">
<p>Refer to Keyboard shortcuts for available shortcuts.</p>
</div>
<h3>The command line</h3>
<p>The command line, integrated into the status bar, allows you to perform all the functionalities of blunderDB available in the graphical interface: general operations on the database, position navigation, displaying analysis and/or comments, searching for positions based on filters... After getting familiar with the interface, it is recommended to gradually use the command line for a powerful and smooth use of blunderDB, especially for position search functionalities.</p>
<p>To open the command line, press the <em>SPACE</em> key. To submit a query and close the command line, press the <em>ENTER</em> key.</p>
<p>blunderDB executes the queries sent by the user as long as they are valid and immediately modifies the state of the database if necessary. There are no explicit save actions required from the user.</p>
<div class="admonition tip">
<p>Refer to list of commands for the list of available commands in the command line.</p>
</div>
<h3>Analysis Panel</h3>
<p>The <strong>Analysis</strong> panel (<em>CTRL-L</em>) displays the analysis data for the current position, imported from eXtreme Gammon (XG), GNUbg, or BGBlitz. It shows the best alternatives (checker moves or cube decisions) with their equity values and corresponding errors. The <em>d</em> key toggles between checker and cube analysis. During match navigation, the actually played move is highlighted in the list of alternatives. Press <em>CTRL-L</em> or run the <code>list</code> command to show or hide the panel.</p>
<p>Under the tables, a <strong>sentence</strong> sometimes says what the played decision cost and why: “You lose 120 mMWC: the move played leaves three blots where 13/7 8/7 leaves only one.” It comes from six measurable rules — exposure, a home point made or missed, gammon chances given up, a safety that costs more than it earns, and the two directions of a cube error (doubling too late or too early, taking too loose or passing too tight).</p>
<p>The rule that matters is <strong>silence</strong>: the sentence appears only when a rule applies confidently, and on an error past the threshold from which the engines agree it is one. The rest of the time there is no sentence — no empty frame, no “we do not know”. A wrong explanation costs more than none: it teaches something inaccurate.</p>
<p>When a position has been judged by <strong>several engines</strong>, a strip at the top of the panel puts them side by side: one line per engine, with its depth and its answer — the cube verdict, or its own best move. It says first whether they agree, and it is the disagreement that justifies it: “XG says double, take; gammonNet says no double” reads at a glance, where two tables had to be compared diagonally.</p>
<p>An engine's best move is the best <strong>of that engine</strong>: the candidate list is sorted by equity across all engines, so its first entry is nobody's best move in particular.</p>
<p>The strip appears only when there really are several engines, and it exists in this panel alone: the Eval panel presents <strong>one</strong> decision, the embedded engine's (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), and a comparison would have no place there.</p>
<p>Moves are written as they read on the board, here as in the Eval panel: the least advanced checker moves first, and <strong>a checker that chains several dice is written only once</strong> — a 64 played with the same checker reads <code>24/14</code>, and <code>24/14*</code> if it hits on arrival. The detail of the chain only reappears when it says something more: a hit <em>on the way</em> keeps its intermediate point, <code>24/18* 18/14</code>, without which the hit on the 18 would vanish from the notation.</p>
<p>An imported analysis' equity follows the same rule as the Eval panel: the column states its own referential, “Equity (money)” or “Equity (match)” depending on the score of the analysed position, never a plain “Equity” silent on the scale. The <strong>Jacoby</strong> and <strong>Beaver</strong> rules active on a money-game position are also shown, in badges under the cube decision table.</p>
<h3>Comments Panel</h3>
<p>The <strong>Comments</strong> panel (<em>CTRL-P</em>) shows, adds and edits the comments attached to the current position. A position may carry several: all of them are shown, most recent first. Comments imported from XG files are automatically attached to the matching positions. Press <em>CTRL-P</em> or run the <code>comment</code> command to show or hide the panel.</p>
<p>Every comment that came out of a file carries a <strong>provenance badge</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, or <em>imported</em> when the provenance was never recorded). Comments you wrote carry none: that is the ordinary case, and marking every line would be noise. Editing an imported comment makes it yours: after the edit, the sentence is yours.</p>
<p>That distinction shows elsewhere: deleting a match no longer destroys a position <strong>you</strong> had written on. A note lifted from the source file does still go with the match that brought it in.</p>
<h4>Tags</h4>
<p>A <strong>tag</strong> is a <code>#word</code> written in a comment. Nothing declares it, no table holds it, and that is deliberate: the vocabulary is your own prose, and requiring a declaration before you could tag would turn a habit into paperwork.</p>
<p>What was missing was the other half: <strong>seeing</strong> the vocabulary you have built, and clicking a tag rather than remembering how you spelt it. The <code>tags</code> command, or the <code>#</code> button beside the input box, opens the vocabulary window: this database's tags, each with the <strong>number of positions</strong> carrying it, clickable to run the corresponding search. Below the list are the recommended tags this database does not use yet — a vocabulary taken from the backgammon literature (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), suggested and never imposed: a tag absent from that list is worth exactly as much as one on it.</p>
<p>While typing, a <code>#</code> offers the tags <strong>this database</strong> already uses, then the recommended ones. That is what keeps you from writing <code>#back-game</code> one day and <code>#backgame</code> the next, which nothing else would catch.</p>
<p>A tag search is written <code>#prime</code> on the command line. It is <strong>delimited</strong>: <code>#prime</code> does not find <code>#priming</code>, where an ordinary text search, which looks for a substring, cannot tell them apart. Several tags <strong>add up</strong> — <code>s #prime #backgame</code> asks for the positions carrying both — because a position carries several tags: naming two can only mean “both”. This is the opposite of the phase or provenance filter, where a position has only one value and naming two can only mean “either”.</p>
<p>The same list is available outside the interface with <code>blunderdb list --type tags</code> (see Command Line Interface (CLI)).</p>
<h3>The trash</h3>
<p>Deleting a position, a collection or a comment now goes through a <strong>trash</strong>: the delete does happen, but a copy of what disappears is kept for thirty days. The <code>trash</code> command opens the window that lists them, each with <em>Restore</em> and <em>Delete</em>.</p>
<p>A restored position comes back with <strong>its analysis and its comments</strong> — giving it back bare would be a restore in name only. It does not come back under its old number: the original row no longer exists, and blunderDB re-saves it by its fingerprint, which guarantees it never creates a duplicate but gives it a new identifier. A collection comes back with its list; the positions it held were never deleted — a collection is a view over them.</p>
<p>Anything older than thirty days is dropped by the <code>vacuum</code> command, never on opening a database: not running <code>vacuum</code> is keeping everything.</p>
<div class="admonition note">
<p>The trash does not travel. An export does not carry it, and deleting a match puts nothing in it: the orphan purge that follows a match deletion is automatic housekeeping, not a user's gesture — see the retention rule in Matches Panel.</p>
</div>
<h3>Search Panel</h3>
<p>The <strong>Search</strong> panel (<em>CTRL-F</em> or <em>TAB</em>) filters positions using freely combinable criteria: checker structure, cube decision type, error magnitude, dates, tags, etc. The <em>TAB</em> key simultaneously opens the search panel and the position editor, allowing a checker structure to be defined directly on the board.</p>
<p>To refine a search among the currently filtered positions, use the <code>ss</code> command followed by filters (e.g.: <code>ss nc</code>, <code>ss E&gt;40</code>). The search panel also offers a <em>Search in current results</em> checkbox for the same functionality.</p>
<p>The panel offers an explicit control over the <strong>decision type</strong> searched for: <em>Indifferent</em> (no filter), <em>Checker</em> (checker decisions) or <em>Cube</em> (cube decisions). When <em>Cube</em> is selected, a second list specifies the sub-type: <em>All</em>, <em>Double / No double</em> (the player on roll has to decide whether to double) or <em>Take / Pass</em> (response to an opponent's double). The control is synchronised with the board: editing the dice or the cube on the board updates the decision type, and vice versa. In <em>Take / Pass</em> mode, the cube is shown in the centre of the board at the offered value; that value remains editable.</p>
<p>The <strong>game phase</strong> — opening, middlegame, race, bearoff — is a label blunderDB computes from the board alone. It is never editable, and is searchable through the command line's <code>ph:</code> token (<code>ph:race</code>, repeatable: <code>ph:race ph:bearoff</code>). Three of its four boundaries are the ones GNU Backgammon uses to route its networks; the fourth, where the opening stops, is a blunderDB convention: a position is still in the opening as long as neither side has moved more than four checkers off its starting points, nothing has been borne off and nothing is on the bar.</p>
<div class="admonition note">
<p>The label is recomputed by the <code>blunderdb repair</code> command. On a database opened for the first time with this version, it is computed once, at that opening. A database whose phases were never computed returns nothing for <code>ph:</code> — nothing, rather than a wrong answer.</p>
</div>
<p>The <code>like</code> command answers a different question from the tokens: it replaces the browsed list by the positions <strong>closest</strong> to the current one, nearest first. Closeness is a transport distance, expressed in checker-pips — the amount of checker movement separating the two positions — and the point of view is always the player on roll's. It is not a filter: similarity <strong>ranks</strong> the whole library instead of narrowing it, and therefore does not combine with the tokens.</p>
<p>The <code>n</code> token counts <strong>encounters</strong>: <code>n&gt;3</code> keeps the positions more than three moves reach, across every match. That is a different question from “what did I get wrong” — a position met twenty times and played correctly nineteen is still the one to know cold. The count is of moves, not matches: the same position twice in one match counts twice, because those were two decisions.</p>
<p>A plain phrase can replace the tokens, with the <code>ask</code> command: <code>ask my cube blunders at a score</code>. The phrase is <strong>translated into tokens</strong>, written into the command bar — read them, then run. Nothing is guessed and nothing leaves the machine: the vocabulary is fixed, the same phrase always gives the same query, and whatever was not understood is <strong>said</strong> rather than passed over. A wrong translation is therefore seen before it returns wrong results, and the tokens are learnt by reading them.</p>
<p>Two intentions are not tokens and are set on the search board rather than in the line: “cube” or “checker” (the kind of decision) and “at a score” or “money”. <code>ask</code> sets them there.</p>
<p>The <strong>plan of play</strong> is a second derived label, beside the phase, and it answers the question a bundle of saved filters cannot ask: “show me my errors in a holding game”. Token <code>gt:</code>, repeatable (<code>gt:holding gt:mutualholding</code>), from the point of view of the <strong>player on roll</strong> — the plan the decision was being made in.</p>
<p>The ten recognised plans, in the order the rules exhaust them, from the most specific to the most general:</p>
<ul>
<li><code>race</code> — the rearmost checkers of both sides have crossed: no contact is possible any more. GNU Backgammon's boundary.</li>
<li><code>bearin</code> — the player on roll is bearing in while the opponent still holds an anchor in their home board.</li>
<li><code>crunch</code> — the player on roll has at most six checkers outside their points 1 and 2. GNU Backgammon's rule, its author's threshold.</li>
<li><code>backgame</code> — two or more anchors in the opponent's home board.</li>
<li><code>acepoint</code> — a single anchor, on the opponent's ace point, at least twenty pips behind.</li>
<li><code>blitz</code> — three or more home points made, and the opponent on the bar or with a blot to hit in that home board.</li>
<li><code>primevprime</code> — both sides hold a prime of at least four points, and each has a checker trapped behind the other's.</li>
<li><code>mutualholding</code> — both sides hold a high anchor.</li>
<li><code>holding</code> — the player on roll holds a high anchor, the opponent does not.</li>
<li><code>contact</code> — contact, and none of the plans above. The opening lands here.</li>
</ul>
<p>Three of these rules are GNU Backgammon's own and are sourced; the others are <strong>blunderDB conventions</strong>. The backgammon literature describes the plans of play without putting numbers on their boundaries, and no inter-classifier agreement has been published for this problem. The unsourced thresholds — three home points for a blitz, four points for a prime, twenty pips behind for an ace-point game — are therefore stated here rather than hidden in the code, and they are versioned: change them, run <code>blunderdb repair</code>, and the whole database is relabelled.</p>
<div class="admonition note">
<p>One label is kept per position, that of the player on roll. A derived label is never editable, never exported as a truth, and a database whose plans have never been computed returns nothing for <code>gt:</code> — as for <code>ph:</code>.</p>
</div>
<p>The <strong>Flagged</strong> filter keeps the positions you flagged in the software the match came from. Only eXtreme Gammon produces this information, recorded move by move in the <code>.xg</code> file; blunderDB reads it on import and keeps it. A flagged cube decision yields two flagged positions, the double and the take/pass, blunderDB splitting in two what the source file records as a single decision.</p>
<div class="admonition note">
<p>Flagging is not retroactive: matches already in the database do not carry this information, since it exists only in the source files. Simply import the relevant <code>.xg</code> file again — the import detects the duplicate and adds nothing but the flags, leaving existing comments and analyses untouched. A flag can neither be set nor removed from within blunderDB: for a temporary working list, use a collection instead.</p>
</div>
<p>The <strong>Comment</strong> filter queries the comments attached to positions in three exclusive modes. <em>contains text</em> searches for one or more words in the comment text (input field, words separated by <code>;</code>, at least one must match); <em>has a comment</em> keeps any position carrying a comment, whatever its content; <em>no comment</em> keeps, on the contrary, the positions that are not annotated — useful, combined with an error or date filter, to draw up the list of what remains to be commented.</p>
<div class="admonition note">
<p>Comments imported from a match file (XG, GNUbg) count as comments. To keep only your own, add the <code>co:user</code> token on the command line (<code>co:xg</code>, <code>co:gnubg</code>, <code>co:bgf</code> and <code>co:unknown</code> name the other provenances). Comments attached to a <em>match</em> or a <em>tournament</em> are not concerned either way: they annotate the match or the tournament, not its positions.</p>
</div>
<p>The <strong>Matches &amp; Tournaments</strong> filter is backed by a shared picker (a modal window) instead of typed numeric IDs: two checkbox lists, one for matches and one for tournaments, each text-filterable (player, date, event for matches; name, date, location for tournaments), with <em>All</em> / <em>None</em> buttons that act only on the currently filtered subset. Checking a tournament automatically checks (and greys out) its member matches in the match list, making visible the fact that a tournament is equivalent to the set of its matches.</p>
<p>The search panel has three tabs along its left edge: <em>Search</em> (the filters), <em>History</em> and <em>Saved</em>. The <strong>History</strong> tab lists past searches with their date and command: a click selects a search and displays the associated position on the board, a double-click re-runs it. Each entry can be saved to the filter library (bookmark icon, by giving the filter a name) or deleted. The <strong>Saved</strong> tab contains the <strong>filter library</strong>: double-click a saved filter to re-run the corresponding search (see Annex: Advanced Filter Usage). The <code>history</code> command (alias <code>hi</code>) opens the search panel.</p>
<div class="admonition tip">
<p>Refer to list of commands for the list of available filters.</p>
</div>
<h3>Collections Panel</h3>
<p>The <strong>Collections</strong> panel (<em>CTRL-B</em>) manages collections of positions. Collections can be created, renamed and deleted. Positions can be added to them or removed (<em>Del</em> key, confirmation asked). Double-click a collection to browse its positions with the <em>LEFT</em> and <em>RIGHT</em> keys. The order of the collections, and of the positions within a collection, can be changed by drag and drop. Press <em>CTRL-B</em> or run the <code>collection</code> command to show or hide the panel.</p>
<h3>Import: what is written, what never is</h3>
<p>Importing a match, a position or another database adds what is missing; it does not replace what is already there.</p>
<ul>
<li><strong>A position is never duplicated.</strong> It is its identity — checkers, cube, dice, score — that recognises it, never the file it came from: the same position met in two matches stays a single row.</li>
<li><strong>One analysis per engine.</strong> eXtreme Gammon, GNUbg, BGBlitz and the embedded evaluator coexist on the same position, and the Analysis panel shows where each one came from. Importing one never erases another.</li>
<li><strong>An imported analysis is never recomputed.</strong> blunderDB stores it as-is, with its level label ("3-ply", "XG Roller++", "Book"), its equities, its errors, its probabilities and the roll's luck. The rule is "an evaluation only fills a gap": automatic analysis after import only visits positions with <strong>no</strong> analysis at all, and <em>Re-analyze stale positions</em> leaves untouched any position carrying an imported analysis (see Configuration).</li>
<li><strong>Reimporting the same file rewrites nothing.</strong> The match is recognised as already present; only the flags set in the originating software are added, without touching comments or analyses.</li>
<li><strong>What blunderDB never writes</strong>: a recomputed luck value — it is read from the source file, or stays unknown — and a rollout, whose data it neither opens from a <code>.xg</code> file nor knows how to produce.</li>
</ul>
<p>A collection can be <strong>living</strong>: its content is no longer a hand-made list but the result of a <strong>search</strong>, re-evaluated every time it is opened. The ◇ button at the head of the collection makes it living with the last search run; ◈ says it already is, and the same button gives it back its list. Nothing is destroyed by making it living: the positions it held are still there when you go back.</p>
<p>A living collection whose query carries a token this version no longer knows <strong>refuses to open</strong>, and says so, rather than returning the whole database. That is the one failure a saved filter must not have: widening in silence.</p>
<h3>Matches Panel</h3>
<p>The <strong>Matches</strong> panel (<em>CTRL-Tab</em>) lists imported matches. Double-click a match (or press <em>ENTER</em>) to navigate through its moves. The <code>m</code> command resumes navigation in the last visited match.</p>
<p>The user can:</p>
<ul>
<li>browse through the moves of a match using the <em>LEFT</em> and <em>RIGHT</em> keys,</li>
<li>switch between games using the <em>PageUp</em> and <em>PageDown</em> keys,</li>
<li>display the move analysis (checker and cube) by pressing <em>CTRL-L</em>,</li>
<li>toggle between checker move and cube analysis with the <em>d</em> key,</li>
<li>see the actually played move highlighted in the analysis.</li>
</ul>
<p>The last visited position in each match is saved and restored automatically. Press <em>CTRL-Tab</em> or run the <code>match</code> command to show or hide the panel.</p>
<p>A row's <strong>⊕</strong> button enriches that match from a file. There is nothing new behind it: re-importing the same match in another format already enriches it in place — the canonical hash recognises that it is the same match, and the analyses and comments of the second file complete the first. What the button adds is that it can be found: nobody guesses that an import is also an enrichment. The report that follows says which of the two happened — “enriched: 1” rather than “imported: 1”.</p>
<p>Each match can be exported as a Jellyfish <code>.mat</code> transcript via the ⬇ button in the match list or the <em>.mat</em> button of the match sheet.</p>
<p>The <strong>Merge players</strong> button in the panel toolbar opens a window listing all the player names in the database with their number of matches: select the spelling variants of the same player, choose the canonical name to keep, then merge. Useful to unify per-player statistics when the same player appears under several names.</p>
<p>When a match is open, an <strong>information bar</strong> appears above the board: it recalls the players involved (<em>player 1</em> versus <em>player 2</em>) as well as the match context (event, location, round, date and match length, when this information is available). This bar is also shown outside match mode: when a studied position (from a search, a collection or a direct access) comes from one or several matches, it indicates its <strong>provenance</strong> — the first match concerned and, where applicable, a "+N" badge listing the others on hover. A position imported on its own, which no match references, shows nothing.</p>
<p>When opening a database that contains matches, the <strong>Matches</strong> panel is shown right away and the review starts directly on the first position, so you can begin navigating immediately.</p>
<div class="admonition note">
<p>A database can be opened for writing by only one window at a time. If you open a database already open in another blunderDB window, it opens <strong>read-only</strong>: navigation, search and analysis remain possible, but any modification is disabled and the title bar shows "[read-only]".</p>
</div>
<div class="admonition tip">
<p>Refer to Keyboard shortcuts for available shortcuts.</p>
</div>
<h3>Tournaments Panel</h3>
<p>The <strong>Tournaments</strong> panel (<em>CTRL-Y</em>) groups matches into tournaments for organised tracking and per-event statistical analysis. Tournaments can be created, renamed, and deleted; matches can be assigned to them. Stats panel statistics can be filtered by tournament. Press <em>CTRL-Y</em> to show or hide the panel.</p>
<p>Tournaments fill themselves at import time. XG, GnuBG and BGF files name their event; when a new match is imported, blunderDB files it under the tournament of that name, creating it if it does not exist yet. The tournament's date and location are left empty — this panel is where they are filled in. A match already in the database is never refiled: re-importing its file does not undo what was arranged by hand.</p>
<p>The <strong>PR</strong> column of each tournament shows the PR of the <strong>reference player</strong> — that is, the player appearing in the greatest number of the tournament's matches (in case of a tie, the one who made the most decisions). The PR therefore does not mix your play with your opponents': for your own tournaments, it reflects your performance alone. The reference player's name appears in a tooltip when hovering over the value.</p>
<h3>Stats Panel</h3>
<h4>Introduction</h4>
<p>The <strong>Stats</strong> panel lets you analyse your play level and track your progress over time using the positions imported in the database. It computes and displays <strong>PR</strong> (Performance Rating) and <strong>MWC cost</strong> (Match Winning Chance cost) for all positions or a filtered subset.</p>
<p>The Stats panel is especially useful for:</p>
<ul>
<li><strong>gauging your level</strong> against the level bands (<em>World Class</em>, <em>Expert</em>, *Advanced*…) using the global PR;</li>
<li><strong>tracking your progress</strong> tournament by tournament or match by match using the Progression tab charts;</li>
<li><strong>identifying your weak spots</strong>: the Errors tab shows the breakdown between checker plays and cube decisions, and the distribution of error magnitudes;</li>
<li><strong>compare the players in the database</strong> with one another, one row per player, through the Players tab — useful for following an entire competition;</li>
<li><strong>navigating directly to the relevant positions</strong> by clicking any indicator (drill-down).</li>
</ul>
<h4>Opening the panel</h4>
<p>To open the Stats panel:</p>
<ul>
<li>Press <em>CTRL-D</em>.</li>
<li>Type the <code>stats</code> or <code>st</code> command in the command line.</li>
</ul>
<div class="admonition note">
<p>The panel refreshes automatically whenever the filter is changed. It does not recalculate statistics on a simple PR ↔ MWC toggle: both metrics are computed simultaneously by the backend.</p>
</div>
<h4>Filter bar</h4>
<p>The filter bar at the top of the panel restricts the computation to a subset of positions.</p>
<h5>Player perspective</h5>
<p>The <strong>Player</strong> drop-down filters statistics to the analysed player. blunderDB automatically selects the player whose name appears most often in the database — changeable at any time.</p>
<div class="admonition tip">
<p>Changing the player does not cause any data loss; simply re-select the previous player in the list.</p>
</div>
<h5>Available filters</h5>
<ul>
<li><strong>Tournament(s)</strong> — restrict to one or more tournaments. Multiple tournaments can be selected simultaneously.</li>
<li><strong>Dates</strong> — time range (<em>From</em> … <em>To</em>). If only the start date is set, more recent positions are included.</li>
<li><strong>Decision type</strong> — All / Checker plays / Cube decisions.</li>
<li><strong>Match length</strong> — restrict to specific match lengths (1, 3, 5, 7, 9, 11, 13, 15, 21 points). Multiple lengths can be combined.</li>
</ul>
<p>A <strong>Reset</strong> button clears all filters (except the auto-detected player).</p>
<div class="admonition note">
<p>Filters are saved in the blunderDB configuration (<code>config.yaml</code>) and restored on the next launch.</p>
</div>
<h4>PR / MWC toggle</h4>
<p>The <strong>PR / MWC</strong> button at the top of the panel toggles the metric displayed across all tabs.</p>
<p><strong>PR (Performance Rating)</strong></p>
<blockquote>
<p>The average equity error per counted decision, multiplied by 500 as eXtreme Gammon and GNUbg do: a PR of 5.0 is worth 0.010 of lost equity per decision, i.e. 10 millipoints (mpt). The exact counting rule — which decisions enter the denominator, how the score is converted — is the one in Annex: Statistics model — XG / gnuBG / blunderDB alignment.</p>
<p>The level bands the panel draws behind the progress curve are an <strong>indicative marker specific to blunderDB</strong>: no publication is authoritative on these thresholds. The upper bound of each band is exclusive: a PR of 4 is <em>Advanced</em>, not <em>Expert</em>.</p>
<table>
<thead>
<tr>
<th>Level</th>
<th>PR</th>
</tr>
</thead>
<tbody>
<tr>
<td>World Class</td>
<td>&lt; 2</td>
</tr>
<tr>
<td>Expert</td>
<td>2 – 4</td>
</tr>
<tr>
<td>Advanced</td>
<td>4 – 6</td>
</tr>
<tr>
<td>Intermediate</td>
<td>6 – 9</td>
</tr>
<tr>
<td>Casual</td>
<td>9 – 12</td>
</tr>
<tr>
<td>Beginner</td>
<td>≥ 12</td>
</tr>
</tbody>
</table>
</blockquote>
<p><strong>MWC cost (Match Winning Chance cost)</strong></p>
<blockquote>
<p>Cumulative match winning probability lost due to errors, over the full filtered dataset. Computed using the Kazaross-XG2 MET embedded in blunderDB.</p>
<div class="admonition caution">
<p>MWC cost <strong>does not apply</strong> to <em>money-game</em> positions (with no match stake). Those positions are excluded from the MWC computation. MWC values depend on the MET used; they are not directly comparable across software using different METs.</p>
</div>
</blockquote>
<p>The PR ↔ MWC toggle is instant: no backend recalculation is performed.</p>
<h4>The HTML report</h4>
<p>The <strong>HTML report</strong> button in the panel's header produces a <strong>self-contained</strong> document: a single file, with no external image, no remote stylesheet, no script. The diagrams are inline SVG, drawn by the same renderer as the board on screen, with your palette. It opens in any browser, travels by e-mail, and <strong>prints to PDF from the browser itself</strong> — which avoids embedding a PDF generator to produce what everybody already has.</p>
<p>It carries the current scope's figures (positions, matches, counted decisions, global, checker and cube PR), then the <strong>ten most expensive decisions</strong>, each with its diagram, its cost, the match it comes from and the best move when an analysis gives one.</p>
<p>The report carries the Stats panel's <strong>current filter</strong>. A report that does not state its scope is a report whose figures mean nothing: set the filter — a tournament, a date range, a player — before producing it.</p>
<h4>Dashboard tab</h4>
<p>The <strong>Dashboard</strong> tab gives a summary view of key indicators.</p>
<h5>Level cards</h5>
<p>Three cards display the PR (or MWC) for:</p>
<ul>
<li><strong>PR Global</strong> — all decisions (checker + cube);</li>
<li><strong>PR Checker</strong> — checker plays only;</li>
<li><strong>PR Cube</strong> — cube decisions only.</li>
</ul>
<p>Clicking a card loads the positions in the corresponding subset into the analysis panel (drill-down).</p>
<div class="admonition note">
<p>The total number of decisions is shown at the bottom of each card on hover.</p>
</div>
<h5>Rolling PR over last N decisions</h5>
<p>A row of PR (or MWC) values computed over the last <em>N</em> decisions (N = 5, 10, 50, 100, 250, 500, 1000) lets you measure the recent trend. Greyed values correspond to an N larger than the number of available decisions.</p>
<p>Clicking a value loads the corresponding last <em>N</em> positions.</p>
<h5>Top blunders</h5>
<p>The list of the 10 worst errors (or MWC cost), sorted by descending magnitude. Clicking a row loads the relevant position in the analysis panel.</p>
<h4>Progression tab</h4>
<p>The <strong>Progression</strong> tab shows how your level evolves over time.</p>
<p>At the top of the tab, a <strong>goal</strong>: “PR &lt; 5 within twelve weeks”. A target, a deadline, and a trend that says where you are heading — nothing more. A goal that started grading, congratulating or reminding would be a different feature, not this one.</p>
<p>The <strong>Suggest</strong> button proposes a target from your current level: the lower bound of the band you are in, that is, the entry into the next one. Proposing “a bit better” would be anchored to nothing; proposing a band says something — going from intermediate to advanced can be seen and told.</p>
<p>The <strong>trend</strong> is a least-squares fit over your matches' PR, projected to the deadline. It refuses to speak below three matches: drawing a line between two points would be a claim that cannot be held. And the sentence says so every time — <em>a trend is not a prediction</em>.</p>
<p>The goal is stored in the <strong>database's metadata</strong>, not in the configuration: it is about that library, so it follows the file rather than the machine. No schema change: <code>metadata</code> is already a key/value table, readable by <code>blunderdb info</code> as by the daemon.</p>
<h5>Tournament line chart</h5>
<p>A line chart displays the PR (or MWC) for each tournament (X axis: tournament order, Y axis: metric value). Colour bands materialise the level thresholds.</p>
<p>Clicking a point on the chart opens a context menu with two options:</p>
<ul>
<li><strong>Open tournament</strong> — opens the tournament in the Tournaments panel.</li>
<li><strong>Open positions</strong> — loads all positions from the tournament into the analysis panel.</li>
</ul>
<h5>Match scatter plot</h5>
<p>A scatter plot represents each match (X axis: date, Y axis: PR or MWC). Point size is proportional to the number of decisions in the match.</p>
<p>Clicking a point opens a context menu:</p>
<ul>
<li><strong>Open match</strong> — opens the match in the Matches panel.</li>
<li><strong>Open positions</strong> — loads all positions from the match into the analysis panel.</li>
</ul>
<h4>Errors tab</h4>
<p>The <strong>Errors</strong> tab breaks down error sources.</p>
<h5>Breakdown by cube action</h5>
<p>A bar chart displays the PR (or MWC) for each type of cube decision: <em>NoDouble</em>, <em>DoubleTake</em>, <em>DoublePass</em>, <em>TooGood</em>. Each bar also shows the number of decisions and the blunder rate in a tooltip.</p>
<p>Clicking a bar loads the positions matching that cube action, <strong>only those with an error</strong> (drill-down).</p>
<h5>Direction of cube errors</h5>
<p>The breakdown above says <em>how much</em> cube decisions cost; this table says in <em>which direction</em> they go wrong.</p>
<p>A cube position carries two decisions taken by two different players, presented here as two rows:</p>
<ul>
<li><strong>Offer</strong> — the player holding the cube doubles or does not. Their errors are the <strong>missed doubles</strong> (a double was called for) and the <strong>premature doubles</strong> (it was not).</li>
<li><strong>Answer</strong> — the player being offered the cube takes or passes. Their errors are the <strong>wrong passes</strong> (a correct take was passed) and the <strong>wrong takes</strong> (a correct pass was taken).</li>
</ul>
<p>The two rows are deliberately kept apart: a player can perfectly well double late <em>and</em> take loosely, and a single figure would call that "balanced" while losing both halves of the information.</p>
<p>Each cell shows the number of decisions; the tooltip gives the cumulated equity lost. Clicking a cell loads the matching positions. A cell at zero is not clickable.</p>
<div class="admonition note">
<p>This table counts decisions, it passes no judgement. At what gap a tendency deserves to be named depends on the sample size and on a point of reference, neither of which the engine holds.</p>
</div>
<h5>Checker / Cube comparison</h5>
<p>A comparison chart places checker plays and cube decisions side by side. Clicking a bar loads the positions in the subset with an error.</p>
<h5>Error magnitude histogram</h5>
<p>A histogram distributes errors by magnitude in millipoints (mpt, buckets: 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Clicking a bar loads the positions in the bucket.</p>
<h4>Breakdowns tab</h4>
<p>The <strong>Breakdowns</strong> tab slices the same decisions the global figures count along four axes. None of them redefines what counts as a decision: that would be a second PR wearing the same name.</p>
<ul>
<li><strong>By game phase</strong> — opening, middlegame, race, bearoff. This is what answers “my PR in the race versus my PR in contact”. The label is computed from the board (see Search Panel); a database whose phases were never computed files everything under <em>Unclassified</em>, and <code>blunderdb repair</code> fills it in.</li>
<li><strong>By plan of play</strong> — race, blitz, holding, backgame, prime vs prime… This is the breakdown the classifier exists for: “where do I lose the most?”, plan by plan. The same derived label as the phase, the same caveats, and <code>blunderdb repair</code> fills it the same way.</li>
<li><strong>By tag</strong> — the <code>#word</code> written in the comments. A position may carry several: <strong>these rows do not sum to the total</strong>, and the panel says so under the table. A tag labels; it does not partition.</li>
<li><strong>By score</strong> — both sides' away score, read from the side of the player on roll, that is from the side of whoever is deciding. The <em>Money</em> row is money play. A cell with fewer than ten decisions is <strong>greyed with its count still visible</strong> rather than hidden: too few to read, but the omission stays auditable.</li>
</ul>
<div class="admonition note">
<p>The Crawford game is not distinguished: blunderDB does not record that flag on a position. The practical effect is small — a Crawford game has no cube decision at all — but the omission is real and is better written down than left to be guessed.</p>
</div>
<h4>Study and real play</h4>
<p>The command <code>blunderdb list --type study --days 30</code> puts three numbers side by side, plan of play by plan of play: how many <strong>distinct positions</strong> were revised over the period, what the PR was <strong>before</strong> it, what the PR is <strong>since</strong>.</p>
<p>Three numbers, and no fourth. There is <strong>no gain column and no arrow</strong>, because nothing here controls for anything: the player may have met stronger opponents, changed format, or simply played more races this month. The rapprochement is the reader's; a column announcing an effect would claim a causality these data do not carry. The numbers themselves are exact.</p>
<p>Reviews are counted as <strong>distinct positions</strong>: a card revised four times in the month is one position studied, and counting the repetitions would make a month of cramming look like a month of coverage. The PR's decisions, on the other hand, are all counted — each was taken once. A PR resting on fewer than ten decisions shows <code>—</code>, with its sample visible beside it.</p>
<h4>Players tab</h4>
<p>The four previous tabs describe <strong>one</strong> player; the <strong>Players</strong> tab compares them all. It shows one row per player in the database, which answers the need of an organiser following a whole competition rather than one player.</p>
<p>Columns, in order:</p>
<table>
<thead>
<tr>
<th>Column</th>
<th>Meaning</th>
</tr>
</thead>
<tbody>
<tr>
<td>Player</td>
<td>The name <strong>as it appears in the matches</strong>. A player recorded under two spellings therefore shows up on two rows; use the player merge to bring them together.</td>
</tr>
<tr>
<td>Matches</td>
<td>Number of matches played within the retained period.</td>
</tr>
<tr>
<td>W–L</td>
<td>Wins and losses. An unfinished match (truncated log, resignation) counts as neither: W + L can therefore be lower than the number of matches.</td>
</tr>
<tr>
<td>Decisions</td>
<td>Number of counted decisions — the PR's denominator. This is the column that says what the neighbouring rates are worth: a PR computed over twelve decisions means nothing.</td>
</tr>
<tr>
<td>PR</td>
<td>Overall Performance Rating.</td>
</tr>
<tr>
<td>Checker PR, Cube PR</td>
<td>The PR split by decision type.</td>
</tr>
<tr>
<td>Snowie</td>
<td>Snowie Error Rate (see Annex: Statistics model — XG / gnuBG / blunderDB alignment).</td>
</tr>
<tr>
<td>Blunders</td>
<td>Number of serious errors (at least 0.100 EMG).</td>
</tr>
<tr>
<td>Luck</td>
<td>Average luck per roll, in millipoints (mpt), signed: positive if the dice were favourable.</td>
</tr>
</tbody>
</table>
<p>Use:</p>
<ul>
<li><strong>Sort</strong> — click a column header. The table opens sorted by ascending PR, best player first. Players for whom nothing was measured stay at the bottom whichever way the sort goes: a zero for lack of data is not a perfect performance.</li>
<li><strong>Open a player's detail</strong> — click a row. The player is selected in the filter bar and the display switches to the Dashboard tab.</li>
<li><strong>Narrow the period</strong> — the date, tournament and match-length filters apply as usual, which makes it possible to bound the table to the dates of a competition.</li>
</ul>
<div class="admonition note">
<p>In this tab, the <strong>Player</strong> list and the <strong>decision type</strong> choice are disabled: the table shows every player, and already splits checker and cube decisions into separate columns.</p>
</div>
<div class="admonition important">
<p>A dash ("—") marks a value that was <strong>never measured</strong>, not to be confused with zero. That is notably the case of the Luck column for any match imported before schema version 2.15.0: luck was not stored back then, and nothing allows it to be reconstructed afterwards — the source files must be re-imported. Formats that do not carry it (BGF, Jellyfish <code>.mat</code>) never will.</p>
</div>
<h4>Aggregation rule</h4>
<div class="admonition important">
<p>The PR of a tournament (or any subset) is computed using the <strong>sum/sum</strong> rule — never as an average of individual match PRs.</p>
<p>Formula:</p>
<pre class="math">PR_&#123;tournament&#125; = 500 \\times \\frac&#123;\\sum_&#123;i&#125; \\text&#123;error&#125;_i&#125;&#123;\\text&#123;total number of decisions&#125;&#125;</pre>
<p><strong>Example:</strong> a player plays two matches in a tournament —</p>
<ul>
<li>Match A: 10 decisions, 0.100 of lost equity → PR = 5.0</li>
<li>Match B: 90 decisions, 0.540 of lost equity → PR = 3.0</li>
</ul>
<p>Naive average of PRs: (5.0 + 3.0) / 2 = <strong>4.0</strong> <em>(incorrect)</em></p>
<p>Sum/sum rule: 500 × 0.640 / (10 + 90) = <strong>3.2</strong> <em>(correct)</em></p>
<p>The sum/sum rule is the only one that handles varying match lengths correctly (a 21-point match carries more weight than a 1-point match).</p>
</div>
<h4>MWC: limitations</h4>
<ul>
<li>MWC cost is computed from the <strong>Kazaross-XG2 MET</strong>, the de facto reference table in competitive backgammon. Results are not directly comparable with software using other METs. It is the same table, read through the same entry point, that the embedded evaluator uses for its cube decisions at a match score: the statistics and the engine cannot diverge on this. It gives its own values up to 25 points to go on each side; beyond that, it is extended by a Zadeh table computed the same way as GNUbg's, up to 64.</li>
<li><em>Money-game</em> positions (with no match score) are <strong>excluded</strong> from the MWC computation. If your database contains many money-game positions, the MWC cost may be underestimated or unavailable.</li>
<li>The MWC cost is cumulative over the full filtered dataset — not a per-decision indicator. It measures the total impact of your errors on your winning chances.</li>
</ul>
<h3>Eval Panel</h3>
<p>The <strong>Eval</strong> panel (<em>CTRL-E</em>) evaluates live whatever position sits on the board; on a bearoff position it specialises and additionally computes the EPC (Effective Pip Count). It is opened by pressing <em>CTRL-E</em>, by clicking the Eval tab in the lower panel, or by running the <code>epc</code> command. This command keeps its original name: the panel was called <em>EPC</em>, then <em>Bearoff</em>, before becoming <em>Eval</em> — so this is where to look for what an earlier version called the Bearoff panel, the name now only naming the tab that configures the bearoff tables.</p>
<p>The panel always shows the <strong>single decision</strong> the position on the board calls for — never two at once — and the facts that go with it. Each quantity is read in the axis that suits it rather than in a single imposed axis: the winning, gammon and backgammon probabilities and the cubeless equity of each player, computed <em>before the roll</em>, are read <strong>per player</strong> (bottom, top, then Δ), to the left of the cube decision, when no dice are showing. Facts and decision stay side by side: the cube decision never drops below the figures that justify it, whatever the interface language and the position on the board. As soon as dice are showing, these same <em>before the roll</em> values change axis: they are read <strong>on roll</strong>, at the head of the candidate-move list, as an italic <em>before the roll</em> row — not one more candidate move, a reference against which to read each move. The gap between that row and a move contains the luck of the roll, never the merit of the move, and so it carries no error column. On a pure bearoff position, a second table, still <strong>per player</strong> and always present, dice showing or not, carries the EPC, the pip count, the wastage, the average number of rolls and the standard deviation; these five columns never migrate. The two tables are stacked and share the same column grid: same edges, same column guides, a single column of dots — they read as one two-storey object. The regime badge, the engine attribution (the depth of the last evaluation appears there too) and the <em>Challenge</em> box form a separate strip, right-aligned above the tables.</p>
<p>Only the candidate-move list scrolls — the <em>before the roll</em> row, too, stays pinned above it; the rest of the panel (facts, badge, cube decision) always remains visible, with no particular adjustment of the panel size.</p>
<p>The facts table and the decision are computed by gammonNet, embedded, without XG or gnubg. The computation follows the position without ever freezing the interface: a 0-ply depth is displayed immediately on every gesture, then, after half a second of stillness, a deeper evaluation (2 plies by default, adjustable in the <em>gammonNet</em> tab of the settings) replaces it in the background — any new gesture cancels that background computation. The depth shown in the badge strip, or inside the regime badge on a race position, is always the one that actually produced the figure shown, never the one requested; it is not repeated on every row, since a live evaluation shares the same depth for all moves. The equity of the candidate moves and of the cube decision follows the score of the position: in money game it is expressed in points, at a match score in <strong>normalised equity</strong> — the same scale as XG and GNU Backgammon, where winning the value of the current cube is worth +1 and losing it −1 — never mixed in the same table. The column header states it explicitly rather than leaving the scale to guess: “Equity (money)” in money game, “Equity (match)” at a match score. It accounts for the <strong>live cube</strong>: the search values every terminal position through the cube model (Janowski, measured efficiency) in the position's cube state, the way XG and GNU Backgammon do in <em>cubeful</em> evaluation. This is what makes the gammon-go and gammon-save effects visible at the score — at 4-away/2-away, the player behind plays 8/2 6/2 on an opening 6-4 because an early double will give the gammon the value of the match, something a cubeless evaluation cannot see. The <em>before the roll</em> row, by contrast, stays a <strong>cubeless</strong> equity: it is a fact about the position, not a decision. This panel never modifies the database: it is a computation, not a stored analysis. Clicking a candidate move shows it on the board as arrows, exactly as in the Analysis panel. The discreet <strong>?</strong> button, in the badge strip, leads to the <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_ engine repository; the full attribution (Strehl network, gammonNet configuration) appears in the Acknowledgments of the help.</p>
<p>The user edits the checker position over the whole board, exactly as in edit mode: left click places a checker of the bottom player, right click a checker of the top player. The second table, the race one, only appears when the resulting position is a pure bearoff (all checkers of both players in their home board); on any other position, only the table of the four common columns (win, gammon, backgammon, cubeless) responds, and the decision bears on the checkers or on a generic cube depending on whether dice are showing.</p>
<p>In each facts table, one row per player — identified by its coloured dot, the black player always at the bottom. The first carries, as long as no dice are showing, the player's win, gammon and backgammon (probabilities, without the % sign) and cubeless equity; the second, on a bearoff position and dice showing or not, the EPC, the pip count, the wastage (difference between the EPC and the pip count), the average number of rolls and the standard deviation. When both players have values to compare, a <strong>Δ</strong> row gives the <em>signed</em> differences (bottom − top: negative when the black player is ahead). Outside a race position, showing dice therefore makes the facts tables themselves disappear: the four columns they carried have just changed axis, on roll, at the head of the move list.</p>
<p>The cube decision always has the same shape, whatever the origin of the figures — exact table, evaluated regime or ordinary gammonNet evaluation: <strong>one row per option</strong>, in the order <em>no double</em>, <em>double/take</em>, <em>double/pass</em>, with its equity in the position's frame of reference and its gap to the best option. The order never changes, unlike the move list: the three options have names, so it is the name one reads, not the rank. The best one is recognised by its highlighting and by its gap cell left empty. When the cube has already been turned, the options read <em>no redouble</em>, <em>redouble/take</em>, <em>redouble/pass</em>.</p>
<p>A last row gives the <strong>verdict</strong>. It takes four values: <em>no double</em>, <em>double, take</em>, <em>double, pass</em> and <em>too good to double</em>, the last when playing the position on is worth more than cashing the point: doubling would then be a mistake for the opposite reason to that of a plain <em>no double</em>. It is also the only place where the panel says there is <strong>no</strong> verdict, rather than suggesting a computation in progress:</p>
<ul>
<li><em>no decision</em> — the regime is not entitled to one; the cube verdict is never estimated (see the <em>estimated</em> badge);</li>
<li><em>not evaluable at this score</em> — the engine refuses the position, typically a score beyond the horizon of the match equity table, i.e. a side with more than 64 points to go;</li>
<li><em>opponent owns the cube</em> and <em>cube dead (Crawford)</em> — the cube cannot be turned. The equities remain displayed, for information, but no option carries a gap: an error is what a choice costs, and there is no choice.</li>
</ul>
<p>In money game, the <strong>Jacoby</strong> and <strong>Beaver</strong> rules active on the position appear under the cube table, in small badges next to the verdict they change: the <em>no double</em> verdict of a position under the Jacoby rule is not the same computation as without it, and nothing else on screen said so.</p>
<p>A third badge, <strong>Max cube</strong>, appears when the source identifier caps the cube — at a match score as well as in a money game. That one does not describe the computation shown above it: the built-in evaluator does not model a ceiling, so the verdict is the one for a free cube. That is precisely why the badge is there: a capped cube is the one visible reason blunderDB and eXtreme Gammon can announce two different verdicts on the same position.</p>
<p>The regime badge, the evaluation depth, the link to the engine and the <em>Challenge</em> box form a separate strip, right-aligned above the tables.</p>
<p>The <strong>player on roll</strong> and the <strong>cube position</strong> are edited directly on the board, as in edit mode: clicking a player's bearoff/score rectangle gives that player the roll; clicking the cube cycles centred → owned bottom → owned top (right click cycles the other way). The cube value stays pinned — in money game the equities are expressed in units of the current cube, only its owner matters. The analysis is recomputed immediately. In the estimated regime, the badge itself is clickable and opens the <em>Bearoff</em> tab of the settings directly; its tooltip explains why (cube verdict not estimable, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) and how to widen the exact domain.</p>
<p>The <strong>score</strong> is also edited directly on the board, as in edit mode: left click on a player's score rectangle decrements their number of points to go, right click increments it. Leaving the <em>money</em> score (-1, -1) by editing one side alone automatically aligns the other side on the same value rather than leaving an inconsistent score. On a bearoff position in the <em>exact</em> regime, moving from a money score to a match score leaves the winning probability as it is (a database lookup, valid whatever the frame of reference) but switches the displayed equity and cube verdict to those of the <em>evaluated</em> regime — the exact table being money by construction, it cannot answer the question asked at the score. The badge then becomes composite (“exact (win) · evaluated (cube)”) to say so explicitly.</p>
<p>The <strong>dice</strong>, finally, are edited the same way, and they are what decides the question being asked: dice on the board make a checker decision (the list of candidate moves), no dice a cube decision. Left-clicking a die raises its value (6 wraps to 1), right-clicking lowers it (1 wraps to 6); clicking a die on a board that has none puts down two at once — a single die would be neither a checker decision nor a cube decision. Clicking a player's rectangle removes the dice to ask a cube question, and the next click on a die puts them back as they were.</p>
<p><em>BACKSPACE</em>, or a double-click outside the board, clears the position: empty board, money score (-1, -1), no dice showing — values specific to the Eval panel, different from those used in edit mode (7 everywhere, dice 3-1), to stay consistent with what the panel shows by default.</p>
<h4>Cube matrix</h4>
<p>A cube decision is not a property of the board. The same checkers, the same pip count, are a double at 2-away/4-away and a no-double at 4-away/2-away; a player who has learnt the money answer has learnt one cell of a grid. The Eval panel shows the cell the position carries; the <strong>cube matrix</strong> shows the whole grid.</p>
<p>The <code>cm</code> command opens it on the position on screen. Each cell gives the verdict at one score: the row is the number of points the player on roll still needs, the column the number the opponent still needs. The four verdicts read <em>ND</em> (no double), <em>DT</em> (double, take), <em>DP</em> (double, pass) and <em>TG</em> (too good); a cell the engine refuses carries a question mark and says why on hover, which also gives the cell's three equities. Three match lengths are offered: 5, 7 and 9 points.</p>
<p>The position's own score is replaced by each cell's; its <strong>cube</strong> is kept. The grid answers “at what score would I turn <em>this</em> cube”, not what a centred position would do. It is post-Crawford throughout: during the Crawford game the cube is not in play, and a column of “you may not double” would say nothing about the position.</p>
<p>Every cell is its own search. The engine is match-aware — it does not play the same game at 2-away as at 7-away — so a single search read through different match equities would be wrong exactly where the score matters. The grid arrives at 0-ply first, then recomputes at the configured display depth once the window is at rest: the same escalation as the rest of the panel, for a 9-point grid costing about a second and a half.</p>
<p>The same grid is computed outside the interface, with the command line's cubematrix command.</p>
<h4>Bringing a position into the Eval panel</h4>
<p>The panel opens by default on a bearoff position, but a study most often starts from a position already at hand. Two gestures bring it there:</p>
<ul>
<li><strong>Right click on the board</strong>, in an analysis panel or while navigating a match, then <em>Evaluate this position</em>: the Eval panel opens directly on that position, as displayed. The context menu does not appear in the Eval panel or in the Search panel, where the right button already serves to place checkers of the other colour.</li>
<li><strong>CTRL-C then CTRL-V</strong>: copy the position from the analysis panel, then paste it once in the Eval panel. Pasting also accepts an identifier from elsewhere — an XGID (eXtreme Gammon, GNU Backgammon, another instance of blunderDB) or an OGID (OpenGammon): it only has to be in the clipboard.</li>
<li><strong>The command</strong> <code>import XGID=…</code> (or <code>import OGID=…</code>) for when the identifier is not in the clipboard but in a message, on a forum read in a terminal, or produced by a script. It is the same verb as plain <code>import</code>: with no argument it opens a file picker, with one it reads the identifier. The path is then identical to pasting — same reading, same deduplication, same opening of the imported position.</li>
</ul>
<p>An OGID carries a position and nothing else: no evaluation, no comment. The position therefore arrives without an analysis, exactly like a bare XGID, and the built-in evaluator can fill the gap afterwards.</p>
<p>The Eval panel's board is a draft: the position arrives there without its database identifier, so that no change made here can rewrite the record it came from. All the usual board edits remain available there (checkers, cube, dice, score), and the evaluation follows every change.</p>
<p>In the other direction, <em>CTRL-C</em> copies the Eval panel's board to the clipboard, with an XGID recomputed from the checkers on the board — hence pasteable directly into eXtreme Gammon or into another instance of blunderDB. Only the position travels: the evaluation shown by the panel is not a database record and does not accompany the copy.</p>
<p>On leaving the Eval panel, the position previously viewed is restored: the draft is never saved on its own.</p>
<p>When the position is a pure bearoff (all checkers of both players in their home board) and no dice are showing, the cube decision shows, for the player on roll:</p>
<ul>
<li>in the <em>exact</em> regime: the money equities (cubeless, no double, double/take, double/pass) and the <strong>money cube verdict</strong> (no double, double/take, double/pass or too good to double) — outside a match score, see above for the case of the score,</li>
<li>in the <em>evaluated</em> regime: the same equities and the same four-valued verdict, but <strong>played out by gammonNet</strong> (search + Janowski cube model) rather than read from a table — available <strong>even at a match score</strong>, which the estimated regime could never offer;</li>
<li>in the <em>estimated</em> regime: the cube verdict is then deliberately not shown — only the winning probability, in the facts table, along with its error margin, remains available.</li>
</ul>
<p>As soon as dice are showing on a race position, this <em>before the roll</em> cube decision disappears — the board then calls for a checker decision, not a cube one — but the winning probability, for its part, remains a fact of the position, not a decision: it joins the <em>before the roll</em> row at the head of the move list, next to the EPC which, for its part, stays displayed just to the left.</p>
<p>A badge indicates the regime: <strong>exact</strong> (value read from a two-sided database), <strong>evaluated · &lt;depth&gt;</strong> (played out by gammonNet — the depth shown is the one that actually produced the figure shown), <strong>estimated ± margin</strong>, or, at a match score within the exact domain, <strong>exact (win) · evaluated (cube)</strong> — see above. The exact regime wins wherever it is available; otherwise the evaluated regime is displayed as soon as it has finished computing, replacing in place the estimated regime shown while waiting. See Methodology and assumptions of the Eval panel for the precise definition of the three regimes and their assumptions.</p>
<p><strong>Widening the exact domain.</strong> The table computed on first launch covers 6 chequers a side. Two ways to go further, in the configuration's <em>Bearoff</em> tab:</p>
<ul>
<li>compute a wider two-sided table — up to TS-06-15 if the machine has the memory for it. The tab states the size, the memory and the time on this machine before starting, and the computation pauses and resumes. A cancelled computation leaves a <code>.part</code> file which is never read as a table;</li>
<li>point to any two-sided gnubg <code>.bd</code> file. The database with the widest domain automatically wins.</li>
</ul>
<p><strong>The panel's board is a scratch board, and it is remembered.</strong> Leaving the Eval panel and coming back finds the position it was left on, not the default bearoff board: that one is only served the first time the panel is opened in a session. Sending a position from the database to the panel wins over that memory, and <em>BACKSPACE</em> hands back the default board at any time. Nothing is written to the database along the way — the scratch board has no position identity, and its evaluation is recomputed on arrival rather than carried over.</p>
<p><strong>Challenge mode.</strong> The <em>Challenge</em> box, in the badge strip, enables a training mode: on every change to the position, the values of three zones are masked (replaced by “···”); clicking a zone reveals that zone only. Without dice, these are the bottom player's row, the top player's row and the cube decision — the Δ row only appears once both player rows are revealed. The decision block then keeps its three rows: it is its values, its verdict and the highlighting of the best option that disappear, failing which the exercise would be solved by looking for the bold row. With dice showing on a race position, each player's EPC row is masked as before, but the third zone then covers the <em>before the roll</em> row and the move list <strong>together</strong>: the list being sorted from best move to worst, revealing it partially would already give the answer away. With dice showing outside a race position, that same single zone alone covers everything the panel displays. One can thus practise estimating each side's EPC, then deciding on the cube or on the move to play, before checking. The setting is remembered.</p>
<p>To close the Eval panel, press <em>CTRL-E</em> or switch to another tab.</p>
<h4>Methodology and assumptions of the Eval panel</h4>
<p>Every value displayed by the panel rests on precise assumptions, stated here exhaustively.</p>
<p><strong>Domain.</strong> The <em>race zone</em> — winning probability and cube verdict — covers pure bearoffs only: every remaining chequer of both players in their home board. The position is evaluated <em>before the roll</em>; any dice set on it are ignored.</p>
<p>The <strong>EPC blocks</strong>, on the other hand, go further: a side gets its EPC as soon as its farthest chequer fits in the loaded one-sided table. With the default table (six points) that is the old home-board rule; with an eight-point table, computed from the <em>Bearoff</em> tab, a side with a chequer on the 8-point is treated like any other. Nothing is extrapolated: a chequer one point too far simply has no EPC, exactly as a chequer on the 7-point had none before. When the table that answered is not the six-point one, its name appears in the corner of the race block ("OS-08") — without it one would read "six" by default and believe the side entirely home.</p>
<p><strong>EPC blocks (always exact).</strong> The EPC, the average number of rolls and the standard deviation come from the exact distribution of the number of rolls needed to bear every chequer off, read from GNUbg's one-sided database (6 to 10 points, 15 chequers, computed on the machine). EPC = average rolls × 49/6 (49/6 ≈ 8.167 is the exact average of pips per roll, doubles counted four times); wastage = EPC − pip count. The only idealisation is <em>one-sided optimal play</em>: each player minimises their own rolls, ignoring the opponent — that is the standard definition of the EPC.</p>
<p><strong>Winning probability, exact regime.</strong> Direct lookup in the widest available two-sided database (TS-06-06 computed on first launch, an external file, or TS-06-11 computed from the <em>Bearoff</em> tab). These databases result from a complete retrograde analysis under optimal two-sided play by both sides: no additional assumption, error limited to quantisation (&lt; 0.002%).</p>
<p><strong>Winning probability, estimated regime.</strong> Outside the database's domain: the probability is obtained by convolving the two one-sided distributions (the player on roll wins if their number of rolls is less than or equal to the opponent's), then applying a frozen polynomial correction, calibrated offline against the TS-06-11 database. Three assumptions:</p>
<ul>
<li><strong>independence</strong> of the two bearoff processes — structural in a race, with no contact there is no interaction whatsoever;</li>
<li><strong>optimal one-sided play by both sides</strong> — this is <em>the approximation</em>: in reality the trailing player deviates to play for variance and the leader for safety. The measured effect is an antisymmetric bias (the convolution overstates the leader's advantage) which the correction absorbs statistically;</li>
<li>the <strong>correction</strong> was calibrated and validated on the oracle's domain (up to 11 checkers per player). Measured residual error: standard deviation 0.05%, 99th percentile 0.17%, observed maximum 0.9% (in winning-probability points). <strong>Beyond 11 checkers per player, this bound is extrapolated</strong> — the trend is monotonic but no oracle certifies it.</li>
</ul>
<p><strong>Equities and cube verdict (exact regime only).</strong> The displayed equities are those of the <strong>money game, without Jacoby</strong>, the reference framework of the bearoff literature. Within the ≤ 11 checkers per player domain, gammons are impossible (each side has already borne off at least 4 checkers): this is not an approximation. The verdict (no double / double, take / double, pass) is reconstructed exactly from the stored equities, following GNUbg's rule, validated verdict for verdict against its analysis.</p>
<div class="admonition note">
<p>The cubeful equities assume <strong>optimal cube play by both sides all the way to the end</strong>: future recubes are fully valued (complete retrograde analysis). In the very volatile races at the end of the game, the cascade of recubes eats up almost all of the advantage of the side on roll — the “no double” and “double/take” equities can then be close to zero where an engine such as XG, whose cube model does not value this cascade, shows values close to the dead cube (for instance 2 checkers on the 3 point against 2 checkers on the 2 point: 62% winning chances, exact D/T +0.006 versus +0.475 for XG). The displayed <strong>decision</strong>, however, coincides with the engines'.</p>
</div>
<p><strong>Winning probability and verdict, evaluated regime.</strong> Outside the exact domain, the winning probability comes from gammonNet's raw output (0- or 2-ply search depending on the gesture, never read from a table), and the verdict from a Janowski “Decide” applied to that output — the search <em>plays out</em> the trajectory instead of summarising a snapshot of it, which is precisely what the estimated regime could not do (see below) and allows, alone among the three regimes together with the exact one, a verdict <strong>at the match score</strong>.</p>
<p>This regime was measured, not merely assumed, against the built-in two-sided table (<code>TestEvalMeasure</code>, 4000 sampled money decisions, canonical parameters 2-ply k=12): money verdict agreement <strong>93.4%</strong> (3735/4000), broken down by distance to gammonNet's take point — 61.1% within 1% of the take point (the zone most sensitive to a coin toss), 88.3% between 1 and 5%, 91.5% between 5 and 10%, 94.0% between 10 and 20%, 94.4% beyond. Winning-probability gap: mean 0.85%, median 0.44%, 95th percentile 3.21%, maximum 8.30%. Cubeful-equity gap: mean 0.039, median 0.018, 95th percentile 0.151, maximum 0.406. The shape is the expected one: most of the disagreement concentrates exactly at the take point, where two legitimately different methods diverge most on a close decision — not a diffuse error that would cost equity everywhere.</p>
<p>This measurement covers <strong>money</strong> decisions, in a race. The match-score verdict — which only this regime can render — and contact positions have no published measurement: none of the above carries over to those cases.</p>
<p><strong>Why not deeper than 2-ply?</strong> Because the measurement says it buys nothing. A checker decision costs 99 ms at 2-ply and 8.4 s at 3-ply on the same machine — <strong>eighty-five times more</strong>. Over forty real decisions replayed at both depths, the deeper search changed its mind <strong>twice</strong>, and both times the gain it claimed for itself was at most 0.0005 normalised equity: two orders of magnitude below 0.020, the threshold at which eXtreme Gammon calls a decision an error at all. Per decision, all cases together, the gain is 0.0000.</p>
<p>The setting is therefore not offered. This does not say 3-ply is worthless in general, only that on <em>this</em> network, at the canonical filter, it does not pay for the wait of someone sitting in front of a panel. The measurement is reproducible (<code>TestThreePlyMeasure</code>) and the conclusion is re-decidable if the network changes.</p>
<p><strong>Why is there no estimated verdict?</strong> What follows targets specifically the <em>convolution</em> method (estimated regime), not the evaluated regime above: cubeful equity is a <em>trajectory</em> problem (when to double) that no statistical summary of the position captures — the best static model measured leaves a residual error (standard deviation 0.016 of equity, maximum 0.20) large enough to flip every close decision. Likewise, converting the verdict to the match score through a match-equity table was measured to be insufficient (12% of disagreements with GNUbg's 2-ply analysis, including genuine blunders). Since a wrong verdict displayed with confidence is worse than no verdict, the convolution was never allowed to display a verdict — it is a search that plays out the trajectory, not a statistical summary, that fills this hole.</p>
<div class="admonition note">
<p>The bearoff databases are immutable mathematical tables. blunderDB computes them itself, identically to GNUbg's <code>makebearoff</code> tool — byte for byte — in the <em>Bearoff</em> tab of the configuration or with <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Anki Panel</h3>
<p>The <strong>Anki</strong> panel (<em>CTRL-K</em>) allows studying positions with spaced repetition using the FSRS algorithm. Users can create decks from collections or search results.</p>
<p><strong>Creating decks:</strong> Click <em>New Deck</em> to create a deck from a collection or the current search results. Search-based decks sync automatically when the Anki tab is opened.</p>
<p><strong>Reviewing:</strong> Select a deck then click <em>Study</em> (or double-click a deck) to start reviewing due cards. Each card shows the corresponding position on the board. Rate your recall with keys <em>1</em> (Again), <em>2</em> (Hard), <em>3</em> (Good), or <em>4</em> (Easy). Press <em>Esc</em> to stop and return to the deck list.</p>
<p><strong>Cube decisions make two cards, chained.</strong> A cube decision is two questions — “double?”, then “take?” — and blunderDB has always stored them as two positions. A deck that selects only one half gets the other: the decision is completed, not enlarged. And when both are due, the second comes <strong>immediately</strong> after the first.</p>
<p>Each keeps its own grade and its own schedule: these are not two stages of one card, they are two cards. Chaining advances no due date — it orders the cards already due, nothing more. Both being born together, they are due together the first time, and that is where it serves.</p>
<p><strong>Showing the answer:</strong> The card asks a question — which move to play, or which cube action. Think, then press <em>SPACE</em> (or click the masked area) to reveal the answer: the recorded analysis of the position, as the Analysis tab presents it. It appears below the rating buttons, which stay in place and within reach. Clicking a move in the list shows it on the board.</p>
<p>Nothing forces you to reveal the answer in order to rate: if you are sure of yourself, the <em>1</em> to <em>4</em> keys stay active. The answer is masked again on the next card, but not if you simply switch tabs — go and consult the Eval panel or the position's comment, it will be waiting for you when you return.</p>
<p>A position without a recorded analysis says so directly, with no masked area.</p>
<p><strong>Limiting the session.</strong> By default a review session runs through every card that is due. You can cap it at a number of cards, per deck, in the Settings: tick <em>Limit session</em> and give how many cards a session should serve. When the limit is reached the session stops and says so — the message tells “limit reached, so many cards still due” apart from a queue that is genuinely empty. To carry on anyway, free drill is there: it serves other positions without changing anything in the schedule.</p>
<p>A limit of <strong>0</strong> serves no card at all: it is a state in its own right, useful to freeze a deck while preparing for a tournament, and it is not the same thing as “no limit”. The <em>Study</em> button is then disabled.</p>
<p>The limit applies to the <strong>session</strong>, not to the day. A blunderDB deck is built on a collection or on a search: it is a finite corpus, introduced over a few sessions, whose daily volume is already bounded by its size. A daily cap would never bite, or else would build a backlog on a deck that fitted in a single session.</p>
<p><strong>Free drill (cram):</strong> The <em>Cram</em> button, next to <em>Study</em>, starts a free-drill session: random positions from the deck are shown to you regardless of the FSRS schedule. This mode <strong>never alters the spaced-repetition plan</strong> — ideal for warming up before a tournament or intensively reviewing a themed deck without disturbing its ordering. A <em>Cram</em> badge replaces the card state and a <em>Next</em> button (keys <em>1</em> to <em>4</em>) cycles through the positions. <em>Esc</em> returns to the list without saving an interrupted session.</p>
<p><strong>Setting a card aside, without grading it.</strong> During a review, a right-click on the card's header offers three gestures that take it out of the session without telling the scheduler anything:</p>
<ul>
<li><strong>Suspend</strong> — the card keeps its schedule and never comes up again while suspended. It is how a card that is wrong, or not useful yet, is set aside without losing the history attached to it.</li>
<li><strong>Bury</strong> — the card disappears until the next day. Unlike suspending, this says nothing about its worth: it is for the one you have just seen elsewhere, or would rather not meet twice in an evening.</li>
<li><strong>Remove</strong> — the card leaves the deck, after confirmation. The position itself stays in the database: a deck is a study list over the library, never a copy of it.</li>
</ul>
<p>None of these three records a grade: a card set aside is not a card answered, and it does not count towards the session's total.</p>
<p><strong>Review log.</strong> In a deck's Settings, the <em>Review log</em> button shows what the scheduler was <strong>told</strong> — date, position, grade, state, granted interval — as opposed to what it plans. It is the only place a grade entered by mistake can be seen. It cannot be corrected there: the schedule stays out of reach, and that rule is precisely what makes the log useful — the past cannot be rewritten, but it can be known.</p>
<p><strong>Pause/Resume:</strong> You can interrupt a review session at any time with <em>Esc</em>. The button changes to <em>Resume</em> and shows your progress. Click it to pick up where you left off.</p>
<p><strong>Deck management:</strong> Use the action buttons to rename, synchronise, reset or delete decks (a confirmation is asked for the last two). The FSRS parameters (target retention, maximum interval, fuzz) can be set per deck in the Settings (gear icon).</p>
<p><strong>Retention: the target and the measurement.</strong> The <em>target retention</em> is your own choice on the trade-off between workload and quality of recall: the higher it is, the shorter the intervals and the more you review. Alongside it, the Settings show the <strong>measured retention</strong> over your own reviews — information, never a control loop: blunderDB does not change your target to chase your success rate. Below some twenty reviews the measurement is not shown: it would read as a fact when it is only noise.</p>
<p>Changing the retention <strong>is not retroactive</strong>: each card takes up the new pace at its next review, and the due dates already set do not move. The effect is therefore gradual, and invisible on the day itself.</p>
<p>The <em>maximum interval</em> bounds the spacing. A recently created deck starts at one year: a position the algorithm would push back by several years has left the deck without you deciding so, and your own game changes faster than that. Older decks keep the value they had.</p>
<h3>Micro-trainings</h3>
<p>The Anki panel makes you revise a <strong>judgement</strong>; the micro-trainings work the three <strong>computations</strong> that happen at the table, on the clock, and that no amount of spaced repetition builds. The <code>train</code> command starts a five-question session:</p>
<ul>
<li><code>train pips</code> — count the pips of the player on roll, on the position shown.</li>
<li><code>train epc</code> — estimate that same player's EPC, on a race position the engine can evaluate.</li>
<li><code>train tp</code> — recall the take point of a long race at a randomly drawn score, the one in the <code>tp2_live</code> table.</li>
</ul>
<p>The question IS the position shown: the board is the application's own, and the bar above it carries only the question, the input and the correction. The answer is typed and validated at the keyboard (<em>Enter</em> checks, then moves on; <em>Esc</em> leaves the session).</p>
<p>The tolerance depends on the drill, and it is stated rather than guessed: the pip count has <strong>none</strong> — an addition right to within one pip is a wrong addition — the EPC accepts half a pip, the take point two percentage points. At the end, the session shows the number of right answers and the <strong>median</strong> time per question.</p>
<p>Only that summary is kept, in the database metadata: the session keeps no question-by-question trace, and nothing is written until it is finished. Leaving halfway therefore records nothing.</p>
<h4>Quiz: the training PR</h4>
<p><code>train quiz</code> asks a fourth kind of question. The Anki panel makes you memorise; the quiz <strong>tests</strong>. Five already-analysed positions are drawn from the browsed list, and a decision has to be made:</p>
<ul>
<li>on a checker decision, type the move at the keyboard, in notation (<code>13/7 8/7</code>);</li>
<li>on a cube decision, click <em>No double</em>, <em>Double, take</em> or <em>Double, pass</em>.</li>
</ul>
<p>The Analysis panel is masked until the question has an answer: it carries the answer, and a question whose answer is displayed beside it is not a question.</p>
<p>The correction keeps three outcomes apart, and collapsing them would lie. An <strong>illegal move</strong> is not a badly chosen move — it is a rules mistake. A <strong>legal move the engine never ranked</strong> is not a mistake at all: it simply has no price, and so costs the session nothing. A ranked move costs what the analysis says it costs, in millipoints.</p>
<p>At the end, the session shows a <strong>quiz PR</strong> computed by the formula the statistics apply to real play — 500 × mean error in normalised equity. That is what makes the two numbers comparable: a quiz PR of 6 and a match PR of 6 measure the same thing on the same scale.</p>
<h3>Metadata Panel</h3>
<p>The <strong>Metadata</strong> panel displays general information about the current database: name, description, number of positions, matches and games, schema version. Accessible via the <code>meta</code> command.</p>
<p>It also shows the database's origin <strong>when there is one</strong> — see Handing out a database: origin and password. An ordinary database does not show that section.</p>
<h3>Handing out a database: origin and password</h3>
<p>A teacher handing out a database of positions has two mechanisms, independent of each other, both optional and both chosen <strong>at export time</strong>: marking the file with its origin, and protecting it with a password.</p>
<div class="admonition note">
<p>Neither tracks what becomes of the file. blunderDB <strong>records nothing on the recipient's side</strong>: opening a marked database is exactly like opening any other, and nothing anywhere logs who opened it, when, or where its contents came from.</p>
</div>
<h4>Marking a database with its origin</h4>
<p>The export dialog fits on a single screen: the form, then a progress overlay laid over it while the file is written. It closes by itself when finished, and the result appears in the status bar.</p>
<p>Three points deserve attention:</p>
<ul>
<li><strong>The export covers the positions currently displayed</strong>, not the whole database. After a search, only the results go out — the dialog says so at the top.</li>
<li><strong>A collection whose positions are not all in the selection arrives truncated.</strong> The list therefore shows, for each collection, how much of it is covered (“12/40”), in red when it is partial.</li>
<li><strong>Tournaments can only be exported together with matches</strong>: without them the tournament–match link does not exist and the tournament would arrive empty. The box stays disabled until “include matches” is ticked.</li>
</ul>
<p>The <em>User</em>, <em>Description</em> and <em>Date</em> fields describe the <strong>file being produced</strong>; they are pre-filled from the source database. The <em>My saved filters</em> box is kept apart from the others: it exports not content but your own saved searches, which are of no use in someone else's database.</p>
<p>Ticking <strong>Mark this file with its origin</strong> reveals two fields:</p>
<ul>
<li><strong>Origin</strong> — what this file is and where it comes from, in your own words: “Jean Dupont's lesson — 12 March 2026”. This field is <strong>required</strong>: while it is empty the export button stays disabled.</li>
<li><strong>Note</strong>, optional — terms of use, a contact address, a request not to pass the file on.</li>
</ul>
<p>The mark is signed with your issuer identity. It is therefore <strong>tamper-evident and unforgeable</strong>: nobody can alter it, nor fabricate one in your name. It is however <strong>not unremovable</strong> — the distributed file is an ordinary SQLite database, and blunderDB is free software. It prevents nothing: it says where the file came from.</p>
<h4>Issuer identity</h4>
<p>Marks are signed with your <strong>issuer identity</strong>, created by itself the first time you mark a file; there is nothing to set up. It belongs to a person rather than to a database: every file you mark carries the same public fingerprint, of the form <code>A3F1-9C24-7B05-E1D8</code>.</p>
<p>You can give that fingerprint to your recipients so they can check that a file really comes from you. The identity moves from one machine to another as a single file (extension <code>.bdbid</code>), optionally protected by a passphrase. <strong>That file lets anyone holding it sign in your name: do not share it.</strong></p>
<p>In the preferences (the gear icon in the toolbar), the <em>Issuer identity</em> tab shows your name and fingerprint, and offers <em>Save identity…</em>, <em>Load identity…</em> and <em>Regenerate…</em>.</p>
<div class="admonition warning">
<p><strong>Regenerating revokes nothing.</strong> A watermark embeds the public key that signed it, so it verifies for ever, on its own. If your identity file has leaked, whoever holds it can keep signing under your old fingerprint, and those marks stay valid.</p>
<p>What protects you after a leak is not software: it is publishing your new fingerprint and disowning the old one to your recipients.</p>
<p>Regenerating overwrites the current key; blunderDB offers to save it before replacing it.</p>
</div>
<h4>Protecting a database with a password</h4>
<p>The password is typed masked, here as when opening a protected file; the eye icon reveals it <strong>while it is held down</strong>, and masks it again as soon as it is released.</p>
<p>Ticking <strong>Protect this file with a password</strong> produces a file with the <code>.dbx</code> extension — even if you chose a <code>.db</code> name in the save dialog, which opens before the password is asked for. To open it, use the usual open-database action: the file chooser accepts both <code>.db</code> and <code>.dbx</code>. blunderDB then asks for the password and installs an ordinary database beside it; nothing is asked afterwards.</p>
<p>The dialog offers to <strong>delete the protected file once opened</strong>: without that you keep the same content under two names. The box is not ticked by default — the protected file is yours to keep if you mean to pass it on — and the deletion only happens after a successful open.</p>
<div class="admonition warning">
<p>The password protects the file <strong>in transit</strong>, not the database. It stops a stranger opening a file left in a downloads folder or an attachment forwarded by mistake. It does not protect you from whoever you gave the password to.</p>
</div>
<p>The password is checked on <strong>every</strong> open, including when the file has already been opened on this machine before.</p>
<p>Technically, the database is encrypted with <strong>AES-256 in GCM mode</strong>, with the key derived from the password by <strong>Argon2id</strong> (64 MiB of memory, 3 passes, 4 lanes) and a random salt unique to each file. GCM authenticates the whole payload: a wrong password is detected as such, and so is any tampering with the encrypted file — you never silently end up with a corrupt database.</p>
<p>The protected file's header stays <strong>in the clear</strong>: its origin remains readable without the password.</p>
<h4>Reading a file's origin</h4>
<p>In the application, open the file and show the <strong>Metadata</strong> panel (the <code>meta</code> command). An <strong>Origin</strong> section appears at the top of the panel, read-only, stating what was written, by whom, when, and how the signature checks out:</p>
<ul>
<li>“✓ signature verified — marked by you”: the file carries your mark, intact;</li>
<li>“✓ signature verified”: the mark is intact and comes from another key — compare its fingerprint with the one the producer gave you;</li>
<li>“⚠ invalid signature”: the document has been altered or forged.</li>
</ul>
<p>This section does not appear on an ordinary database.</p>
<p>From the command line, <code>blunderdb info --db file.db</code> shows the origin and the state of the signature, <strong>without ever writing to the file</strong>. It works on a protected file too, without the password. See <code>CLI_USAGE.md</code> for <code>export</code>'s <code>--watermark</code> and <code>--password</code> options, and for <code>identity</code> and <code>open</code>.</p>
<h4>Publishing a database for others</h4>
<p>A marked database is distributed like any other file — email, a personal site, a USB stick. blunderDB <strong>provides no service</strong>: no repository, no hosted catalogue, no account. That follows directly from its design: nothing is ever recorded on the side of whoever receives a file, so there would be nothing to report to a service even if one existed.</p>
<p>What makes a published database usable by someone else comes down to four fields, all of them already there:</p>
<ul>
<li><strong>User</strong> — who built it, under the name you want cited.</li>
<li><strong>Description</strong> — what the database holds, in one sentence that fits in a list: “240 cube decisions at a score, commented, intermediate level”.</li>
<li><strong>Origin</strong> (of the watermark) — what this file is and who it was produced for. It is the first thing the recipient reads in the <em>Metadata</em> panel.</li>
<li><strong>Issuer fingerprint</strong> — publish it beside the file, not inside it: comparing it is how the recipient checks the file comes from you and not from someone who took your name.</li>
</ul>
<p>A database published without a watermark stays perfectly usable; it is simply anonymous, and the <em>Metadata</em> panel then shows no <em>Origin</em> section.</p>
<p>To make a database known, the <em>Show and tell</em> category of the <code>repository discussions &lt;https://github.com/kevung/blunderDB/discussions&gt;</code>_ serves as a directory: it is a list kept by those who publish, not a service blunderDB renders. Announcing one there takes the link, the four fields above and the fingerprint.</p>
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
<td>Merge a database into this one.</td>
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
<h3>Help Panel</h3>
<table>
<thead>
<tr>
<th>Shortcut</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>LEFT, h</td>
<td>Previous tab.</td>
</tr>
<tr>
<td>RIGHT, l</td>
<td>Next tab.</td>
</tr>
<tr>
<td>UP, k</td>
<td>Scroll up.</td>
</tr>
<tr>
<td>DOWN, j</td>
<td>Scroll down.</td>
</tr>
<tr>
<td>SPACE</td>
<td>Next page.</td>
</tr>
<tr>
<td>PageUp</td>
<td>Top of the content.</td>
</tr>
<tr>
<td>PageDown</td>
<td>Bottom of the content.</td>
</tr>
<tr>
<td>?, CTRL-F, Esc</td>
<td>Close the help.</td>
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
<td>Opens the Eval panel (Effective Pip Count, winning probability and cube verdict in bearoff). <code>epc</code> is this panel's former name, kept for compatibility.</td>
</tr>
<tr>
<td>met</td>
<td>Open the Kazaross-XG2 match equity table.</td>
</tr>
<tr>
<td>cm</td>
<td>Opens the cube matrix: the current position’s verdict at every score of a 5-, 7- or 9-point match.</td>
</tr>
<tr>
<td>tags</td>
<td>Opens the tag vocabulary: the tags used in this database, with the number of positions, clickable to run the search.</td>
</tr>
<tr>
<td>log</td>
<td>Opens the activity log: the last two hundred lines of the log file, with what it takes to copy them into a report, or to open the folder holding them.</td>
</tr>
<tr>
<td>ask</td>
<td>Translates a plain phrase — French or English — into search tokens: <code>ask my cube blunders at a score</code>. The tokens are written into the command bar, not run: read them, then Enter. Whatever was not understood is said, never guessed.</td>
</tr>
<tr>
<td>like</td>
<td>Replaces the browsed list by the positions closest to the current one — or to the one whose index is given (<code>like 42</code>). Closeness is a transport distance in checker-pips: it is not a filter, it ranks the whole database rather than narrowing it, and therefore does not combine with the search tokens.</td>
</tr>
<tr>
<td>train</td>
<td>Starts a micro-training session. Takes an argument: <code>train pips</code> (pip count), <code>train epc</code>, <code>train tp</code> (take point at a match score), <code>train quiz</code> (the move or the cube action, graded against the stored analysis). Five questions, timed, corrected on the spot.</td>
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
<td>Imports one or more positions/matches from a file (xg, xgp, sgf, mat, txt, bgf). With an argument — <code>import XGID=…</code> or <code>import OGID=…</code> — reads the identifier instead of opening a file picker, for when it comes from a message, a forum or a script.</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Deletes the current position (with a confirmation); the delete goes through the trash and stays undoable for thirty days.</td>
</tr>
<tr>
<td>trash</td>
<td>Opens the trash: what was deleted, and what it takes to restore it.</td>
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
<p>This table is the reference for the search grammar: the command line, the filter library and the <code>--query</code> flag of <code>blunderdb search</code> all read the same tokens. The <em>CLI Equivalent</em> column gives, when one exists, the <code>search</code> flag that does the same thing (see Command Line Interface (CLI)); a dash marks a filter that only the grammar expresses.</p>
<p>Five tokens do not carry their value: they read it from the search board. <code>cube</code> and <code>score</code> take up the cube and the score set there, <code>d</code> the decision type, <code>D</code> and <code>D1</code> the dice, <code>x</code> the structure drawn in the <em>Except</em> tab. A roll is therefore never written into the token: <code>D65</code> does not exist, only the exclusion form carries its digits (<code>xD65</code>). On the command line, where there is no board, these tokens compare against an empty board; it is the flags of the third column that must be used there instead.</p>
<p>Errors and equities are counted in <strong>thousandths of equity</strong> — the <em>millipoints</em> of the table below: <code>E&gt;100</code> keeps the moves that cost at least a tenth of a point, one point being worth 1000 millipoints.</p>
<p>Two full searches:</p>
<ul>
<li><code>s p&gt;30 w40,60 xco</code> — more than 30 pips behind, between 40% and 60% winning chances, no comment.</li>
<li><code>s ph:race E&gt;50 co:xg</code> — in a race, a move that cost at least 50 millipoints, and a comment coming from eXtreme Gammon.</li>
</ul>
<table>
<thead>
<tr>
<th>Query</th>
<th>Action</th>
<th>CLI Equivalent</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>The position checks the cube configuration.</td>
<td><code>--cube</code></td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>The position checks the score.</td>
<td><code>--score1</code> <code>--score2</code></td>
</tr>
<tr>
<td>d</td>
<td>The position checks the dice or the cube decision.</td>
<td><code>--decision</code></td>
</tr>
<tr>
<td>D</td>
<td>The position matches the dice roll (both dice, any order).</td>
<td><code>--dice 6,5</code></td>
</tr>
<tr>
<td>D1</td>
<td>The position matches the dice roll on the first die only (the first die's value appears on either die of the position).</td>
<td><code>--dice 6</code></td>
</tr>
<tr>
<td>xD65</td>
<td>The position was <strong>not</strong> played with the 6-5 roll (any order). The value is given in the token; repeatable to exclude several rolls (<code>xD65 xD54</code>).</td>
<td>—</td>
</tr>
<tr>
<td>nc</td>
<td>The position has no contact.</td>
<td>—</td>
</tr>
<tr>
<td>ph:race</td>
<td>The position is in a given phase of the game: <code>opening</code>, <code>middlegame</code>, <code>race</code> or <code>bearoff</code>. Repeatable (<code>ph:race ph:bearoff</code>). The label is derived from the board and never editable; <code>blunderdb repair</code> recomputes it.</td>
<td><code>--phase</code></td>
</tr>
<tr>
<td>gt:holding</td>
<td>The position falls under a given plan of play, from the point of view of the player on roll: <code>race</code>, <code>bearin</code> (bearing in under contact), <code>crunch</code>, <code>backgame</code>, <code>acepoint</code>, <code>blitz</code>, <code>primevprime</code>, <code>mutualholding</code>, <code>holding</code>, <code>contact</code>. Repeatable (<code>gt:holding gt:mutualholding</code>). A derived label like the phase: computed from the board, never editable, recomputed by <code>blunderdb repair</code>.</td>
<td><code>--game-type</code></td>
</tr>
<tr>
<td>#prime</td>
<td>The position carries this <strong>tag</strong> in one of its comments. A tag is a <code>#word</code> written in the prose; nothing declares it. The comparison is delimited, so <code>#prime</code> does not find <code>#priming</code> — that is the whole difference from the text filter, which looks for a substring. Repeatable, and tags <strong>add up</strong> (<code>#prime #backgame</code> asks for both): a position carries several tags, so naming two means “both”.</td>
<td>—</td>
</tr>
<tr>
<td>n&gt;x</td>
<td>The position was met more than x times in the database — the number of moves that reach it, across every match. Forms <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> and <code>n4</code> (exactly four).</td>
<td>—</td>
</tr>
<tr>
<td>M</td>
<td>The position or the mirror one meets the filters.</td>
<td>—</td>
</tr>
<tr>
<td>i</td>
<td>The position was imported on its own, not brought in by a match import.</td>
<td><code>--individual</code></td>
</tr>
<tr>
<td>fl</td>
<td>The position was flagged in the source software, when importing an eXtreme Gammon match.</td>
<td><code>--flagged</code></td>
</tr>
<tr>
<td>x</td>
<td>The position contains none of the checkers of the exclusion structure (the "Except" tab of the search panel).</td>
<td>—</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>The player has at least x pips behind in the race.</td>
<td><code>--pip-min</code></td>
</tr>
<tr>
<td>p&lt;x</td>
<td>The player has at most x pips behind in the race.</td>
<td><code>--pip-max</code></td>
</tr>
<tr>
<td>px,y</td>
<td>The player has between x and y pips behind in the race.</td>
<td><code>--pip-min</code> <code>--pip-max</code></td>
</tr>
<tr>
<td>P&gt;x</td>
<td>The player has a race of at least x pips.</td>
<td>—</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>The player has a race of at most x pips.</td>
<td>—</td>
</tr>
<tr>
<td>Px,y</td>
<td>The player has a race between x and y pips.</td>
<td>—</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>The equity (in millipoints) of the position is greater than x.</td>
<td>—</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>The equity (in millipoints) of the position is less than x.</td>
<td>—</td>
</tr>
<tr>
<td>ex,y</td>
<td>The equity (in millipoints) of the position is between x and y.</td>
<td>—</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>The error of the move played by player 1 (in millipoints) is greater than x.</td>
<td><code>--move-error-min</code></td>
</tr>
<tr>
<td>E&lt;x</td>
<td>The error of the move played by player 1 (in millipoints) is less than x.</td>
<td><code>--move-error-max</code></td>
</tr>
<tr>
<td>Ex,y</td>
<td>The error of the move played by player 1 (in millipoints) is between x and y.</td>
<td><code>--move-error-min</code> <code>--move-error-max</code></td>
</tr>
<tr>
<td>w&gt;x</td>
<td>The player has winning chances greater than x%.</td>
<td><code>--winrate-min</code></td>
</tr>
<tr>
<td>w&lt;x</td>
<td>The player has winning chances less than x%.</td>
<td><code>--winrate-max</code></td>
</tr>
<tr>
<td>wx,y</td>
<td>The player has winning chances between x% and y%.</td>
<td><code>--winrate-min</code> <code>--winrate-max</code></td>
</tr>
<tr>
<td>g&gt;x</td>
<td>The player has gammon chances greater than x%.</td>
<td>—</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>The player has gammon chances less than x%.</td>
<td>—</td>
</tr>
<tr>
<td>gx,y</td>
<td>The player has gammon chances between x% and y%.</td>
<td>—</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>The player has backgammon chances greater than x%.</td>
<td>—</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>The player has backgammon chances less than x%.</td>
<td>—</td>
</tr>
<tr>
<td>bx,y</td>
<td>The player has backgammon chances between x% and y%.</td>
<td>—</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>The opponent has winning chances greater than x%.</td>
<td>—</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>The opponent has winning chances less than x%.</td>
<td>—</td>
</tr>
<tr>
<td>Wx,y</td>
<td>The opponent has winning chances between x% and y%.</td>
<td>—</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>The opponent has gammon chances greater than x%.</td>
<td>—</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>The opponent has gammon chances less than x%.</td>
<td>—</td>
</tr>
<tr>
<td>Gx,y</td>
<td>The opponent has gammon chances between x% and y%.</td>
<td>—</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>The opponent has backgammon chances greater than x%.</td>
<td>—</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>The opponent has backgammon chances less than x%.</td>
<td>—</td>
</tr>
<tr>
<td>Bx,y</td>
<td>The opponent has backgammon chances between x% and y%.</td>
<td>—</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>The player has at least x checkers off.</td>
<td><code>--off1-min</code></td>
</tr>
<tr>
<td>o&lt;x</td>
<td>The player has at most x checkers off.</td>
<td>—</td>
</tr>
<tr>
<td>ox,y</td>
<td>The player has between x and y checkers off.</td>
<td>—</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>The opponent has at least x checkers off.</td>
<td><code>--off2-min</code></td>
</tr>
<tr>
<td>O&lt;x</td>
<td>The opponent has at most x checkers off.</td>
<td>—</td>
</tr>
<tr>
<td>Ox,y</td>
<td>The opponent has between x and y checkers off.</td>
<td>—</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>The player has at least x backcheckers.</td>
<td>—</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>The player has at most x backcheckers.</td>
<td>—</td>
</tr>
<tr>
<td>kx,y</td>
<td>The player has between x and y backcheckers.</td>
<td>—</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>The opponent has at least x backcheckers.</td>
<td>—</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>The opponent has at most x backcheckers.</td>
<td>—</td>
</tr>
<tr>
<td>Kx,y</td>
<td>The opponent has between x and y backcheckers.</td>
<td>—</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>The player has at least x checkers in the zone.</td>
<td>—</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>The player has at most x checkers in the zone.</td>
<td>—</td>
</tr>
<tr>
<td>zx,y</td>
<td>The player has between x and y checkers in the zone.</td>
<td>—</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>The opponent has at least x checkers in the zone.</td>
<td>—</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>The opponent has at most x checkers in the zone.</td>
<td>—</td>
</tr>
<tr>
<td>Zx,y</td>
<td>The opponent has between x and y checkers in the zone.</td>
<td>—</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>The player has at least x blots in the outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>The player has at most x blots in the outfield.</td>
<td>—</td>
</tr>
<tr>
<td>box,y</td>
<td>The player has between x and y blots in the outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>The opponent has at least x blots in the outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>The opponent has at most x blots in the outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BOx,y</td>
<td>The opponent has between x and y blots in the outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>The player has at least x blots in the jan.</td>
<td>—</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>The player has at most x blots in the jan.</td>
<td>—</td>
</tr>
<tr>
<td>bjx,y</td>
<td>The player has between x and y blots in the jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>The opponent has at least x blots in the jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>The opponent has at most x blots in the jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJx,y</td>
<td>The opponent has between x and y blots in the jan.</td>
<td>—</td>
</tr>
<tr>
<td><code>t'word1;word2;...'</code></td>
<td>The position comments contain at least one of the words.</td>
<td>—</td>
</tr>
<tr>
<td>co</td>
<td>The position carries a comment, whatever its content.</td>
<td><code>--has-comment</code></td>
</tr>
<tr>
<td>xco</td>
<td>The position carries no comment.</td>
<td><code>--no-comment</code></td>
</tr>
<tr>
<td>co:user</td>
<td>The position carries a comment of a given origin: <code>user</code> (written by you), <code>xg</code>, <code>gnubg</code>, <code>bgf</code> (brought in by a match import) or <code>unknown</code>. Repeatable (<code>co:xg co:gnubg</code>).</td>
<td><code>--comment-origin</code></td>
</tr>
<tr>
<td><code>m'pattern1,pattern2,...'</code></td>
<td>The best checker moves containing at least one of the patterns.</td>
<td>—</td>
</tr>
<tr>
<td><code>m'ND,DT,DP,...'</code></td>
<td>The best cube decisions for No Double/Take, Double Take, Double Pass.</td>
<td>—</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Date of position addition after x (YYYY/MM/DD).</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Date of position addition before x (YYYY/MM/DD).</td>
<td>—</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Date of position addition between x and y (YYYY/MM/DD).</td>
<td>—</td>
</tr>
<tr>
<td>max</td>
<td>Search in match with ID x (e.g. ma3).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>max,y</td>
<td>Search in matches with IDs from x to y (e.g. ma2,5).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>tnx</td>
<td>Search in tournament with ID x (e.g. tn1).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>tnx,y</td>
<td>Search in tournaments with IDs from x to y (e.g. tn1,3).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>idx</td>
<td>Search for the position with identifier x (e.g. id12).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td>idx,y</td>
<td>Search for the positions with identifiers x to y (e.g. id5,10).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td><code>pl'name'</code></td>
<td>Search positions from a match involving the named player, at either seat (e.g. <code>pl'Alice'</code>). Case-insensitive.</td>
<td>—</td>
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
