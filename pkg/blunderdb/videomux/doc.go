// Package videomux tells whether a Matroska or WebM file can be seeked into by a player that
// starts decoding at the beginning of the cluster a cue points to.
//
// A cue names the cluster that holds a keyframe and, optionally, the keyframe's offset inside
// it. GStreamer's matroskademux in push mode (WebKitGTK's video element) ignores that offset:
// it decodes from the cluster's first block. When that block is a delta frame, the decoder
// fails on it and WebKit treats the failure as fatal — the seek never lands and the picture
// stays frozen. Muxers that cut clusters on size or time rather than on keyframes (Lavf as
// yt-dlp runs it) write such files; remuxing with clusters long enough to start on keyframes
// repairs them without re-encoding (RemuxCommand).
//
// Check reads only the SeekHead, Tracks, Cues and a few element headers at the start of each
// cluster a cue references: a few kilobytes per cluster, whatever the file's size. Every read
// is bounded, and a truncated or malformed file yields an error or a partial report, never a
// panic.
package videomux
