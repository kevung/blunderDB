// Package pdf is a minimal PDF reader, written for what package document
// needs and nothing more: the page tree, the document information and
// outline, each page's content streams, its fonts (widths and ToUnicode), and
// an interpreter of the text operators that places every glyph a page draws.
// It renders nothing and decodes no image.
//
// It is pure Go on the standard library alone, so the single binary and the
// cgo-free daemon build carry it, and nothing outside the repository has to be
// audited for it. A PDF here is a file a person downloaded: the reader never
// trusts it. The file, each decoded stream and the sum of decoded streams have
// size limits, nesting and indirection have depth limits, every loop over the
// file's own structure has a bound or a visited set, DefaultLimits fixing the
// sizes (a file of 512 MiB, a stream of 64 MiB, 20 000 pages, one million
// glyphs), and a malformed file is
// an error (ErrMalformed and its kin), never a panic — every exported entry
// point recovers, and FuzzOpen holds the reader to it.
//
// What it reads: classic cross-reference tables and cross-reference streams
// (PDF 1.5), object streams, incremental updates, and, when the
// cross-reference is broken, a scan of the file's objects. Filters: Flate
// (with PNG and TIFF predictors), ASCIIHex and ASCII85; any other filter is
// ErrUnsupported for that stream. Encrypted files are refused (ErrEncrypted).
package pdf
