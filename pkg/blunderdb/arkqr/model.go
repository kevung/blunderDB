package arkqr

// The play model of format version 1: a softmax over the legal plays, with
// integer weights over integer features of the board each play leaves. The
// weights and the table below are part of the format: changing one number
// changes every archive, so a new model is a new format version.

const nFeats = 25

// weights[regime] is the model of a race (0) and of a contact position (1).
var weights = [2][nFeats]int{
	{0, -10, 0, 0, 0, -47, 51, 17, -9, -13, -40, -6, 17, -28, -2, -76, 88, -17, -6, 30, 25, 63, 0, 29, 0},
	{92, 10, -16, 14, -3, 34, 24, 35, 18, 4, 65, -60, 24, -5, -15, -39, -30, -3, 10, 7, 42, 54, 16, -9, 0},
}

// expTable[d] is round(4096·e^(−d/32)), at least 1: the weight of a play
// whose score is d below the best one. Written out rather than computed, so
// no floating point enters the codec.
var expTable = [401]int{
	4096, 3970, 3848, 3729, 3615, 3503, 3396, 3291, 3190, 3092, 2997, 2904, 2815, 2729, 2645, 2563,
	2484, 2408, 2334, 2262, 2192, 2125, 2060, 1996, 1935, 1875, 1818, 1762, 1707, 1655, 1604, 1555,
	1507, 1460, 1416, 1372, 1330, 1289, 1249, 1211, 1174, 1137, 1102, 1069, 1036, 1004, 973, 943,
	914, 886, 859, 832, 807, 782, 758, 734, 712, 690, 669, 648, 628, 609, 590, 572,
	554, 537, 521, 505, 489, 474, 460, 445, 432, 418, 406, 393, 381, 369, 358, 347,
	336, 326, 316, 306, 297, 288, 279, 270, 262, 254, 246, 238, 231, 224, 217, 210,
	204, 198, 192, 186, 180, 174, 169, 164, 159, 154, 149, 145, 140, 136, 132, 128,
	124, 120, 116, 113, 109, 106, 103, 99, 96, 93, 90, 88, 85, 82, 80, 77,
	75, 73, 70, 68, 66, 64, 62, 60, 58, 57, 55, 53, 52, 50, 48, 47,
	46, 44, 43, 41, 40, 39, 38, 37, 35, 34, 33, 32, 31, 30, 29, 28,
	28, 27, 26, 25, 24, 24, 23, 22, 21, 21, 20, 20, 19, 18, 18, 17,
	17, 16, 16, 15, 15, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 10,
	10, 10, 10, 9, 9, 9, 8, 8, 8, 8, 7, 7, 7, 7, 7, 6,
	6, 6, 6, 6, 5, 5, 5, 5, 5, 5, 5, 4, 4, 4, 4, 4,
	4, 4, 4, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1,
}

// feats are the features of a board from the side that just played.
func feats(r rel) [nFeats]int {
	my, opp := &r.my, &r.opp
	var sh [25]int8
	sh[0] = opp[25]
	for i := 1; i <= 24; i++ {
		sh[i] = opp[25-i]
	}
	ob := int(opp[25])
	var blots, direct, indirect, risk, made, home, stack, deep, builders, homecnt, hblots, anch int
	back, tot, rear := int(my[25]), int(my[25]), 0
	if my[25] > 0 {
		rear = 25
	}
	for p := 24; p >= 1; p-- {
		n := int(my[p])
		if n == 0 {
			continue
		}
		tot += n
		rear = max(rear, p)
		if n == 1 {
			blots++
			d, ind := 0, 0
			for o := max(0, p-6); o < p; o++ {
				if sh[o] != 0 {
					d++
				}
			}
			for o := max(0, p-11); o < max(0, p-6); o++ {
				if sh[o] != 0 {
					ind++
				}
			}
			direct += d
			indirect += ind
			if d != 0 || ind != 0 {
				risk += 25 - p
			}
			if p <= 6 {
				hblots++
			}
		} else {
			made++
			if p <= 6 {
				home++
			}
			if p >= 19 {
				anch++
			}
			if n > 3 {
				stack += n - 3
			}
		}
		if p <= 3 {
			deep += n
		}
		if p >= 19 {
			back += n
		}
		if p >= 7 && p <= 11 {
			if n == 1 {
				builders++
			} else {
				builders += n - 2
			}
		}
		if p <= 6 {
			homecnt += n
		}
	}
	run, best := 0, 0
	for p := 1; p <= 12; p++ {
		if my[p] >= 2 {
			run++
			best = max(best, run)
		} else {
			run = 0
		}
	}
	b := func(c bool) int {
		if c {
			return 1
		}
		return 0
	}
	return [nFeats]int{ob, blots, direct, indirect, risk, made, home, b(my[5] >= 2), b(my[4] >= 2), b(my[7] >= 2),
		b(my[20] >= 2 || my[21] >= 2), anch, best, back, stack, deep, 15 - tot, rear, builders, homecnt,
		b(my[6] >= 2), b(my[3] >= 2), ob * home, hblots, int(my[25])}
}

// playFreqs are the coder frequencies of the candidates of a play.
func playFreqs(ps []play, regime int) []uint64 {
	w := &weights[regime]
	sc := make([]int, len(ps))
	mx := 0
	for i, p := range ps {
		f := feats(p.after)
		s := 0
		for k := range nFeats {
			s += w[k] * f[k]
		}
		sc[i] = s
		if i == 0 || s > mx {
			mx = s
		}
	}
	out := make([]uint64, len(ps))
	for i, s := range sc {
		out[i] = uint64(expTable[min(400, mx-s)])
	}
	return out
}
