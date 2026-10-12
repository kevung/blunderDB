// Package capture reads a backgammon position from a screen capture of
// another program's board, in pure Go, without a learned model: each known
// board is a theme — its geometry in units of the point width and the colours
// it draws — and Read looks for it at every scale.
//
// Locating: a grid of probes (triangle centres, the field between the
// triangles, the bar, the edges) is slid over a reduced copy of the image at
// each scale; the triangles' alternation is what fixes the board, since
// checkers are allowed almost everywhere. The best placements are refined at
// full resolution.
//
// Reading: each point is read from its base as a run of one colour; the bar
// by the length of its runs; the checkers borne off are what the board lacks
// of fifteen. The dice are two squares of one player's colour in the middle
// band, counted by their spots, and give the player on roll; the cube is read
// in the tray, its owner by its height and its value by the shape of its
// digits. The score is never read.
//
// Nothing is guessed in silence: what cannot be settled — a point hidden by a
// dialog, a stack taller than the slots the program draws, dice that cannot
// be read, the side each colour plays from — is returned as a Doubt, for the
// correction step to show on the board before the position is saved.
package capture
