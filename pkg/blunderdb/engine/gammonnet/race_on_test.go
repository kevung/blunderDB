// SPDX-License-Identifier: MIT

//go:build race

package gammonnet

// raceEnabled lets a test that proves bit-identity over a corpus keep, under
// the race detector, only what exercises each concurrent path once: the
// detector slows a search six- to sevenfold, and the full proof runs without
// it.
const raceEnabled = true
