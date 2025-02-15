package gnubgid

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

type MatchKey struct {
	cubevalue   int
	cubeowner   int
	diceowner   int
	crawford    int
	gamestate   int
	turnowner   int
	double      int
	resign      int
	dice        [2]int
	matchlength int
	matchscore  [2]int
}

type GnubgID struct {
	// Structure containing the GNUbgID string, the Position ID, Match ID
	// and checker positions for both sides of the board, respectively called:
	// Position1 and Position2.
	// The ID constists of: "PositionID:MatchID"
	// example: "4HPwATDgc/ABMA:VgkGAAAAAAAA"
	ID         string  // example: "4HPwATDgc/ABMA:VgkGAAAAAAAA"
	PositionID string  // example: "4HPwATDgc/ABMA"
	MatchID    string  // example: "VgkGAAAAAAAA"
	Position1  [26]int // An array of nb of checkers on each point.
	Position2  [26]int // 0: bear-off checkers, 1: pt 1, …, 24: pt 24, 25: on bar
	MatchKey   MatchKey
}

func PrintHumanReadableMatchKey(mk *MatchKey) error {

	fmt.Println("Match Information:")
	// Cube Value
	fmt.Println("Cube:", mk.cubevalue)

	// Cube Ownership
	var own string
	switch mk.cubeowner {
	case 0:
		own = "Player 0"
	case 1:
		own = "Player 1"
	case 3:
		own = "Centered Cube"
	default:
		return errors.New("bad cube ownership value")
	}
	fmt.Println("Cube ownership:", own)

	// Player on roll (Dice owner)
	fmt.Println("Player on roll: Player", mk.diceowner)

	// Crawford flag
	fmt.Println("Crawford flag:", mk.crawford)

	// Game state
	var gs string
	switch mk.gamestate {
	case 0b000:
		gs = "No game started"
	case 0b001:
		gs = "Playing"
	case 0b010:
		gs = "Game over"
	case 0b011:
		gs = "Resigned"
	case 0b100:
		gs = "Dropped"
	default:
		return errors.New("bad gamestate value")
	}
	fmt.Println("Game state:", gs)

	// Turn
	fmt.Println("Player turn:", mk.turnowner)

	// Double offered
	fmt.Println("Double offered:", mk.double)

	// Resign
	var rv string
	switch mk.resign {
	case 0b00:
		rv = "No resignation"
	case 0b01:
		rv = "Resign Single game"
	case 0b10:
		rv = "Resign Gammon"
	case 0b11:
		rv = "Resign Backgaammon"
	default:
		return errors.New("bad resign value")
	}
	fmt.Println("Resign value:", rv)

	// Dice
	fmt.Printf("Dice: %d-%d\n", mk.dice[0], mk.dice[1])

	// Match length
	fmt.Println("Match length:", mk.matchlength)

	// Match score
	fmt.Printf("Match score: %d-%d\n", mk.matchscore[0], mk.matchscore[1])

	return nil

}

func checkPositionID(s string) (string, error) {
	// Check the GNUbg Position ID string (in base64)
	// and append "==" at the end of the string if it is missing.
	//
	// From https://www.gnu.org/software/gnubg/manual/gnubg.html#gnubg-tech_postionid :
	// "The ID format is simply the Base64 encoding of the key.
	//  (Technically, a Base64 encoding of 80 binary bits should consist
	//  of 14 characters followed by two = padding characters,
	//  but this padding is omitted in the ID format.)"
	//
	// Check if the size is 16 characters
	if len(s) != 16 && len(s) == 14 {
		s = s + "=="
		//fmt.Println("Making PositionID Correction:", s)
	}
	if len(s) != 16 {
		//fmt.Println("Bad Position ID, wrong size:", len(s))
		return "", errors.New("bad position ID, wrong size")
	}
	return s, nil

}

func validateGnubgID(g *GnubgID) error {
	// Position ID length: 14 ASCII characters
	// Match ID length: 12 characters
	// GNUbgID length: 14+1+12 characters

	// Validate GNUbgID
	//fmt.Println("GNUbgID length:", len(g.ID))

	switch len(g.ID) {
	case 12:
		//fmt.Println("The GNUbgID is a MatchID only")
		g.PositionID = ""
		g.MatchID = g.ID
	case 14:
		//fmt.Println("The GNUbgID is a PositionID only")
		var err error
		g.PositionID, err = checkPositionID(g.ID)
		if err != nil {
			return err
		}
		g.MatchID = ""
	case 27:
		//fmt.Println("This is a GNUbgID with PositionID:MatchID")
		if len(g.ID) >= 27 && strings.Contains(g.ID, ":") {
			ss := strings.Split(g.ID, ":")
			// Check PositionID
			var err error
			g.PositionID, err = checkPositionID(ss[0])
			if err != nil {
				return err
			}
			g.MatchID = ss[1]
		}
	default:
		return errors.New("bad GNUbgID length")
	}
	return nil
}

func getMatchID(g *GnubgID) error {

	// Decode the Base64 string match key (mk)
	mk, err := base64.StdEncoding.DecodeString(g.MatchID)
	if err != nil {
		return errors.New("error decoding match key (base64)")
	}
	// Get Match key
	var mkbs [9 * 8]int //The match key is a bit string (mkbs) of length 66
	for i, ct := 0, 0; i < len(mk); i++ {
		for j := 0; j < 8; j++ {
			if (mk[i] & (1 << j)) != 0 {
				mkbs[ct] = 1
			} else {
				mkbs[ct] = 0
			}
			ct++
		}
	}

	// Bit 1-4 contains the 2-logarithm of the cube value. For example, a 8-cube is encoded as 0011 binary (or 3), since 2 to the power of 3 is 8. The maximum value of the cube in with this encoding is 2 to the power of 15, i.e., a 32768-cube.
	n := 0
	for i := 0; i < 4; i++ {
		if mkbs[i] != 0 {
			n |= (1 << i)
		}
	}
	g.MatchKey.cubevalue = 1 << n

	// Bit 5-6 contains the cube owner. 00 if player 0 owns the cube, 01 if player 1 owns the cube, or 11 for a centered cube.
	n = 0
	n |= mkbs[4] << 0
	n |= mkbs[5] << 1
	g.MatchKey.cubeowner = n

	// Bit 7 is the player on roll or the player who did roll (0 and 1 for player 0 and 1, respectively).
	g.MatchKey.diceowner = mkbs[6]

	// Bit 8 is the Crawford flag: 1 if this game is the Crawford game, 0 otherwise.
	g.MatchKey.crawford = mkbs[7]

	// Bit 9-11 is the game state: 000 for no game started, 001 for playing a game, 010 if the game is over, 011 if the game was resigned, or 100 if the game was ended by dropping a cube.
	n = 0
	n |= mkbs[8] << 0
	n |= mkbs[9] << 1
	n |= mkbs[10] << 2
	g.MatchKey.gamestate = n

	// Bit 12 indicates whose turn it is. For example, suppose player 0 is on roll then bit 7 above will be 0. Player 0 now decides to double, this will make bit 12 equal to 1, since it is now player 1's turn to decide whether she takes or passes the cube.
	g.MatchKey.turnowner = mkbs[11]

	// Bit 13 indicates whether an doubled is being offered. 0 if no double is being offered and 1 if a double is being offered.
	g.MatchKey.double = mkbs[12]

	// Bit 14-15 indicates whether an resignation was offered. 00 for no resignation, 01 for resign of a single game, 10 for resign of a gammon, or 11 for resign of a backgammon. The player offering the resignation is the inverse of bit 12, e.g., if player 0 resigns a gammon then bit 12 will be 1 (as it is now player 1 now has to decide whether to accept or reject the resignation) and bit 13-14 will be 10 for resign of a gammon.
	n = 0
	n |= mkbs[13] << 0 // put lsb at value of bit 14
	n |= mkbs[14] << 1 // put lsb at value of bit 15
	g.MatchKey.resign = n

	// Bit 16-18 and bit 19-21 is the first and second die,
	// respectively. 0 if the dice has not yet be rolled, otherwise
	// the binary encoding of the dice,
	// e.g., if 5-2 was rolled bit 16-21 will be 101-010.
	dice := [2]int{}
	for i, k := 0, 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			dice[i] |= mkbs[15+k] << j
			k++
		}
	}
	g.MatchKey.dice = dice

	//    Bit 22 to 36 is the match length. The maximum value for the match length is 32767. A match score of zero indicates that the game is a money game.
	ml := 0
	for i := 0; i < 36-22+1; i++ {
		ml |= mkbs[21+i] << i
	}
	g.MatchKey.matchlength = ml

	//    Bit 37-51 and bit 52-66 is the score for player 0 and player 1 respectively. The maximum value of the match score is 32767.
	score := [2]int{}
	for i, k := 0, 0; i < 2; i++ {
		for j := 0; j < 51-37+1; j++ {
			score[i] |= mkbs[36+k] << j
			k++
		}
	}
	g.MatchKey.matchscore = score

	return nil
}

func getPositionID(g *GnubgID) error {
	// Get PositionID given a GnubgID structure
	// Fills in the fields Position1 and Position2

	// Decode the Base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(g.PositionID)
	if err != nil {
		return errors.New("error decoding position key (base64)")
	}

	// TODO Most probably a more elegant way to do this but I needed an array
	// for my own sanity
	// Reference: https://www.gnu.org/software/gnubg/manual/html_node/A-technical-description-of-the-Position-ID.html#A-technical-description-of-the-Position-ID
	var key [80]int //  The worst-case representation will require 80 bits
	sum, count := 0, 0
	for i := 0; i < len(decodedBytes); i++ {
		for j := 0; j < 8; j++ {
			if (decodedBytes[i] & (1 << j)) != 0 {
				sum += 1
			} else {
				if sum != 0 {
					key[count] = int(sum)
				} else {
					key[count] = int(0)
				}
				sum = 0
				count++
			}
		}
	}

	// Fill in Position1 and Position2 from the bitstream array
	for i := 0; i < 25; i++ {
		g.Position1[i+1] = key[i]
	}
	for i := 25; i < 50; i++ {
		g.Position2[i-25+1] = key[i]
	}

	// Fill in off checkers
	// 15 = nb_on_board + nb_on_bar + nb_off
	cnt := 0
	for _, v := range g.Position1 {
		cnt += v
	}
	// Only add Off checkers if there are some
	nb_off := 15 - cnt
	if nb_off > 0 {
		g.Position1[0] = nb_off
	}
	cnt = 0
	for _, v := range g.Position2 {
		cnt += v
	}
	// Only add Off checkers if there are some
	nb_off = 15 - cnt
	if nb_off > 0 {
		g.Position2[0] = 15 - int(cnt)
	}
	return nil
}

func ReadGnubgID(s string) (GnubgID, error) {
	// Read the input string and fill in the GNUbgID Structure
	// Technical reference: https://www.gnu.org/software/gnubg/manual/gnubg.html#gnubg-tech_postionid
	//fmt.Println("Gnubgid(input):\t", s)
	// Create GnubgID structure
	g := GnubgID{}
	// Fill in the GNUbgID string
	g.ID = s
	// Validate the GNUbgID
	err := validateGnubgID(&g)
	if err != nil {
		//fmt.Println("validate GnubgID")
		return g, err
	}
	// Get MatchID
	err = getMatchID(&g)
	if err != nil {
		//fmt.Println("getMatchID")
		return g, err
	}
	// Get PositionID
	err = getPositionID(&g)
	if err != nil {
		//fmt.Println("getPositionID")
		return g, err
	}

	return g, nil
}
