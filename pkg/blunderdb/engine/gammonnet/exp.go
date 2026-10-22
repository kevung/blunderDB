// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in Go's LICENSE file.

package gammonnet

import "math"

// portableExp is e**x computed the same way, bit for bit, on every
// platform. math.Exp is not: on amd64 it switches at run time to a
// fused-multiply-add path when the CPU has FMA, and on arm64 its assembly
// fuses too, so the sigmoid of one network output could differ in its last
// bit between two machines. This is the pure-Go algorithm of Go's math/exp.go
// with every product rounded on its own (float64(a*b)), which the compiler
// may not fuse into the following addition.
func portableExp(x float64) float64 {
	const (
		ln2Hi = 6.93147180369123816490e-01
		ln2Lo = 1.90821492927058770002e-10
		log2e = 1.44269504088896338700e+00

		overflow  = 7.09782712893383973096e+02
		underflow = -7.45133219101941108420e+02
		nearZero  = 1.0 / (1 << 28)
	)

	switch {
	case math.IsNaN(x):
		return x
	case x > overflow:
		return math.Inf(1)
	case x < underflow:
		return 0
	case -nearZero < x && x < nearZero:
		return 1 + x
	}

	// Reduce: x = k·ln2 + r, r computed as hi - lo for extra precision.
	var k int
	switch {
	case x < 0:
		k = int(float64(log2e*x) - 0.5)
	case x > 0:
		k = int(float64(log2e*x) + 0.5)
	}
	hi := x - float64(float64(k)*ln2Hi)
	lo := float64(k) * ln2Lo
	return expMulti(hi, lo, k)
}

// expMulti returns e**r × 2**k where r = hi - lo and |r| <= ln(2)/2.
func expMulti(hi, lo float64, k int) float64 {
	const (
		p1 = 1.66666666666666657415e-01
		p2 = -2.77777777770155933842e-03
		p3 = 6.61375632143793436117e-05
		p4 = -1.65339022054652515390e-06
		p5 = 4.13813679705723846039e-08
	)

	r := hi - lo
	t := r * r
	poly := p4 + float64(t*p5)
	poly = p3 + float64(t*poly)
	poly = p2 + float64(t*poly)
	poly = p1 + float64(t*poly)
	c := r - float64(t*poly)
	y := 1 - ((lo - (r*c)/(2-c)) - hi)
	return math.Ldexp(y, k)
}
