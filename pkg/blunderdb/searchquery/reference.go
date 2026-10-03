package searchquery

// Reference is the grammar's token list as a reader learns it: what each token
// form selects. It is shared by the CLI's --query-help and by the MCP search
// tool's description, so a model and a terminal user read the same text.
const Reference = `
Flags (no value):
  cube score   match the cube / the score of the position on the board
  d            match the decision type (checker or cube)
  dd  dr       cube decisions only: double / no double, or take / pass response
  D  D1        match both dice / the first die
  nc           no contact
  M            search the mirrored position too
  i            imported on its own, not inside a match
  fl           flagged for study in the source tool
  co  xco      carries a comment / carries none

Ranges — each takes x>n, x<n or xa,b (lower-case: you; upper-case: the opponent):
  p P          pip count difference / absolute pip count
  w W  g G  b B   win / gammon / backgammon rate
  o O          checkers borne off        k K   checkers back
  z Z          checkers in the zone      bo BO  outfield blots
  bj BJ        blots in the jan          e      equity
  E            error of the played move, in millipoints
  T            date the analysis was made (not the match's date), T>2026/01/01
  ml           match length: ml:7, ml:5,9, ml>5, ml<9
  md           DATE OF THE MATCH: md:2024-01..2024-12, md:2024, md>2024-06, md<2024-06
               (each bound a year, month or day, covering its whole span)
  pr           PR of the match for the player who took the decision, pr>8, pr<5, pr4,9
               (matches with no PR are left out)

Values:
  t"tag"       comment text (";" separates alternatives)
  m"13/11"     best move or cube decision
  pl"Name"     a player, at either seat; case-insensitive, * is a wildcard
  pl!"Name"    only the decisions that player took (their own seat)
  op"Name"     the opponent: with pl, matches between the two; alone, a player at either seat
  tn"Name"     tournament by name (case-insensitive, * wildcard); tn7 is by id
  rd:3         round, as the match records it (repeatable, * wildcard)
  ad:xg        analysis engine (xg, gnubg, bgblitz, hedgehog, gammonnet) or depth
               (3ply, 3ply+, book, rollout); engines are alternatives, depths too,
               one of each kind must hold
  xD65         exclude the 6-5 roll (repeatable)
  ma1 tn2 id7  match / tournament / position ids (repeatable)
  ph:race      game phase: opening, middlegame, race, bearoff (repeatable)
  co:user      comment origin: user, xg, gnubg, bgf, unknown (repeatable)


`
