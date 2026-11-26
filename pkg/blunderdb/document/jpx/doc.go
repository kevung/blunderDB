// Package jpx reads JPEG 2000 images (a JP2 file, or the bare codestream a
// PDF's JPXDecode filter holds) in pure Go, behind guards the decoder does
// not have.
//
// The decoder (github.com/d-fuji/jpeg2000) reads untrusted input without a
// work budget: a handful of header bytes can ask for more memory or time than
// any request may take, and a decode cannot be cancelled. Inspect therefore
// walks the main header and every tile-part header, skipping the packet data,
// and refuses before anything is allocated what this reader does not take:
//
//   - not one or three components, or components subsampled;
//   - a reference grid that is not the image's own, or over MaxPixels;
//   - more tiles than the image can fill with 64×64 tiles, or 65535;
//   - a precinct under 32×32 or a code-block under 16×16, at any level;
//   - high-throughput JPEG 2000 (Rsiz bit 14, the CAP marker, a code-block
//     style with the HT bit);
//   - a JP2 palette, component mapping or channel definition.
//
// The decoder is not registered with the image package, so no other reader
// in the process (a screen capture, a position image) can reach it.
package jpx
