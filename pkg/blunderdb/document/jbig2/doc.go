// Package jbig2 decodes the bi-level images a PDF compresses with its
// JBIG2Decode filter (ITU-T T.88, embedded organisation, the segments of a
// JBIG2Globals stream first), in pure Go, under a work budget.
//
// It takes what PDF producers write: generic regions, MMR (through
// golang.org/x/image/ccitt) or arithmetic with any of the four templates,
// typical prediction and adaptive pixels; refinement regions; arithmetic
// symbol dictionaries, refinement and aggregation included, with their
// retained contexts; arithmetic text regions. The parser refuses, in one
// pass over the segment headers and before anything is decoded:
//
//   - Huffman coding (a symbol dictionary or a text region with its Huffman
//     flag), a halftone region, an extended template, an extension marked
//     necessary, a segment of unknown length;
//   - a page over MaxPixels or a side over 2^20, of another size than its
//     container says; a region over the same bounds; a page and regions
//     of more than 2^28 pixels together, what a stream holds in memory;
//   - a dictionary of more than 2^20 symbols, with what it refers to, or
//     exporting more than it holds; a text region placing more than 2^22
//     symbols, or from more than 2^20;
//   - a segment longer than what remains, a reference to a segment not read
//     before it, a region before the page information.
//
// The coded data is bounded where it is read: the arithmetic decoder reads
// past the end as 0xFF, and every loop of the procedures stops on the counts
// the headers announced. Over a document it is bounded by the caller: a
// Budget of pixels, spent by the page and its regions before decoding, then
// by each symbol a dictionary decodes (2^28 pixels at most in a stream) and
// each symbol a text region places;
// and a context, read before each segment, each symbol and every 256 rows
// of a region or symbols of a text region.
//
// Nothing is registered with the image package: the decoder is reached only
// through this package.
package jbig2
