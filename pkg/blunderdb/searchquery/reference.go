package searchquery

// Reference is the grammar's token list as a reader learns it: what each token
// form selects. It is shared by the CLI's --query-help and by the MCP search
// tool's description, so a model and a terminal user read the same text.
const Reference = `
Flags (no value):
  cube score   match the cube / the score of the position on the board
  d            match the decision type (checker or cube)
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
  T            creation date, T>2026/01/01

Values:
  t"tag"       comment text (";" separates alternatives)
  m"13/11"     best move or cube decision
  pl"Name"     a player, at either seat
  xD65         exclude the 6-5 roll (repeatable)
  ma1 tn2 id7  match / tournament / position ids (repeatable)
  ph:race      game phase: opening, middlegame, race, bearoff (repeatable)
  co:user      comment origin: user, xg, gnubg, bgf, unknown (repeatable)


`
