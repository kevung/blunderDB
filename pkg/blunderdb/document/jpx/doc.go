// Package jpx reads JPEG 2000 images (a JP2 file, or the bare codestream a
// PDF's JPXDecode filter holds) in pure Go, under a work budget.
//
// The decoder is internal/j2k, a copy of github.com/d-fuji/jpeg2000 whose
// parser refuses, before anything proportional to the image is allocated,
// what this reader does not take:
//
//   - not one or three components, or components subsampled;
//   - a reference grid that is not the image's own, or over MaxPixels;
//   - more tiles than the image can fill with 64×64 tiles, or 65535;
//   - a precinct under 32×32 or a code-block under 16×16, at any level;
//   - high-throughput JPEG 2000 (Rsiz bit 14, the CAP marker, a code-block
//     style with the HT bit);
//   - a JP2 palette, component mapping or channel definition;
//   - a header the parser cannot read in one pass: a SIZ anywhere but at
//     byte 2, a SOD in the main header, a SOT in a tile-part header.
//
// The coded data is bounded where it is read: a count of zero bit-planes by
// the band's bit-planes, coding passes by 109, so that the costliest data
// measured, every code-block with all its passes, takes about 1.1 µs a
// pixel. Over a document it is bounded by the caller: a Budget of pixels,
// and a context, read before each tile, each packet and each code-block's
// decoding. The work between two reads is unchecked: one packet header (at
// most every code-block of a precinct), one code-block, or the structure,
// inverse transform and colour conversion of one tile, some 0.2 s for a
// tile of 12 Mpx in three components.
//
// Nothing is registered with the image package, so no other reader in the
// process (a screen capture, a position image, an archive sheet) reaches the
// decoder.
package jpx
