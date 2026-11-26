package j2k

import "errors"

// FormatError reports that the input is not a valid JPEG 2000 stream.
type FormatError string

func (e FormatError) Error() string { return "jpeg2000: invalid format: " + string(e) }

// UnsupportedError reports that the input uses a valid JPEG 2000
// feature that this decoder does not implement yet.
type UnsupportedError string

func (e UnsupportedError) Error() string { return "jpeg2000: unsupported feature: " + string(e) }

func errFormat(msg string) error { return FormatError(msg) }

var errShortPacket = errors.New("jpeg2000: unexpected end of packet data")
