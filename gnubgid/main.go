package main

import (
	"fmt"
	"log"

	"local.shit/gnubgid/gnubgid"
)

func main() {
	// Various GNUbgID:
	//GNUBGID := "4HPwATDgc/ABMA:cAmrApAAkAAA" // match score: 9-18 / 21-length
	//GNUBGID := "poop_string" // BORKEN
	//GNUBGID := "4HPwAWDgc/ABUA:cAkKAAAAAAAA"
	//GNUBGID := "4PPgASiMZ/ABMA:QgkGAAAAAAAA"
	//GNUBGID := "4HPwATDgc/ABMA:VgkGAAAAAAAA"
	//GNUBGID := "4HPwATDgc/ABMA:cwlgAAAAAAAA" // center cube 8
	//GNUBGID := "4HPwATDgc/ABMA:UwlgAAAAAAAA" // down cube 8
	//GNUBGID := "4HPwATDgc/ABMA:QwlgAAAAAAAA" // up cube 8 player0
	//GNUBGID := "4HPwATDgc/ABMA:cIlqAAAAAAAE" //playingg
	//GNUBGID := "sGfwATDgc/ABMA:cAMAABAAAAAA" // resigned
	//GNUBGID := "4NvIAVCyHcMBKA:MAIgAAAACAAE" // ended
	//GNUBGID := "4NvIAVCyHcMBKA:cAQAAAAACAAA" //dropped
	//GNUBGID := "AAAAAAAA/v8AAA:cAkAAAAAAAAA" // 15 on bar 15 off
	//GNUBGID := "4HPwAyDgc/ABMA:cAlrAAAAAAAE" // Dice: 6-2
	//GNUBGID := "AAAAAAAA/v8AAA:cAkAAAAAAAAA" // 15 on bar 15 off

	// Test PositionID only:
	//POSITIONID := "4HPwATDgc/ABMA==" // default position w/ ==
	//POSITIONID := "4HPwATDgc/ABMA" // default position
	//POSITIONID := "AAAA/z8AAID/Dw" // 13bar/2off + 14bar/1off
	//POSITIONID := "AAAA/z8AAID/Dw" // 13bar/2off + 14bar/1off
	//POSITIONID := "AwAA/H8DAAD8fw" // 13bar/ 2 on pt1 for both side
	//POSITIONID := "2+4OAADb7g4AAA" // Homeboard 3-3-3-2-2-2 for both sides
	//GNUBGID := POSITIONID

	// Various Tests:

	// Test GNUbgID is complete (PositionID:MatchID)
	GNUBGID := "4HPwATDgc/ABMA:cAkAAAAAAAAA" // Default - Money game
	// Test GNUbgID is MatchID only
	//MATCHID := "cAlrAAAAAAAE" // Dice: 6-2
	//GNUBGID := MATCHID
	// Test GNUbgID is PositionID only
	//POSITIONID := "2+4OAADb7g4AAA" // Homeboard 3-3-3-2-2-2 for both sides
	//GNUBGID := POSITIONID

	// Read the GNUbgID
	result, err := gnubgid.ReadGnubgID(GNUBGID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result, err)

	// Print Position 1
	fmt.Println("Position 1", result.Position1)

	// Print Position 2
	fmt.Println("Position 2", result.Position2)

	// Print Match information
	fmt.Println("Match Key:", result.MatchKey)
	err = gnubgid.PrintHumanReadableMatchKey(&result.MatchKey)
	if err != nil {
		log.Fatal(err)
	}
}
