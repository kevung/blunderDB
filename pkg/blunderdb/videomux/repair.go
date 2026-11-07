package videomux

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrNoFFmpeg is returned by Repair when no ffmpeg is on the PATH and no sound copy exists.
var ErrNoFFmpeg = errors.New("ffmpeg not found")

// RepairOptions are Repair's dependencies; the zero value finds ffmpeg on the PATH and caches
// under the user's cache directory.
type RepairOptions struct {
	// FFmpeg is the executable to run; "" looks "ffmpeg" up on the PATH.
	FFmpeg string
	// CacheDir receives the copy when the video's own directory refuses it; "" is
	// os.UserCacheDir()/blunderDB/remux.
	CacheDir string
}

// remuxArgs are RemuxCommand's, run without a shell: no re-encoding, every stream kept,
// clusters long enough to open on keyframes.
func remuxArgs(in, out string) []string {
	return []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-i", in, "-map", "0", "-c", "copy", "-cluster_time_limit", "30000", out}
}

// Repair returns a copy of in whose seeks land: RemuxOutput(in) beside it, or a copy in the
// cache directory named after in's path, size and modification time when the directory is
// not writable. A sound copy already there is returned as is, without ffmpeg. The copy is
// written to a temporary file of the destination directory and renamed once checked, so a
// cancelled or failed run leaves nothing a later call would mistake for a copy. in itself is
// only read.
func Repair(ctx context.Context, in string, opt RepairOptions) (string, error) {
	abs, err := filepath.Abs(in)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", abs)
	}
	beside := RemuxOutput(abs)
	cached := ""
	if dir, err := remuxCacheDir(opt.CacheDir); err == nil {
		cached = filepath.Join(dir, cacheName(abs, st))
	}
	for _, c := range []string{beside, cached} {
		if c != "" && sound(ctx, c) {
			return c, nil
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	ff := opt.FFmpeg
	if ff == "" {
		ff = "ffmpeg"
	}
	if ff, err = exec.LookPath(ff); err != nil {
		return "", ErrNoFFmpeg
	}

	target := beside
	tmp, err := createPart(beside)
	if err != nil {
		if cached == "" {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
			return "", err
		}
		if tmp, err = createPart(cached); err != nil {
			return "", err
		}
		target = cached
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmp)
		}
	}()

	cmd := exec.CommandContext(ctx, ff, remuxArgs(abs, tmp)...)
	hideConsole(cmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if line := lastLine(stderr.String()); line != "" {
			return "", fmt.Errorf("ffmpeg: %s", line)
		}
		return "", fmt.Errorf("ffmpeg: %w", err)
	}
	rep, err := CheckFile(ctx, tmp)
	if err != nil {
		return "", err
	}
	if rep.Partial || len(rep.Faulty) > 0 {
		return "", fmt.Errorf("the remuxed copy still has %d clusters opening on a delta frame", len(rep.Faulty))
	}
	if err := os.Rename(tmp, target); err != nil {
		return "", err
	}
	keep = true
	return target, nil
}

// sound reports whether path is a Matroska file read to the end none of whose
// cue-referenced clusters opens on a delta frame.
func sound(ctx context.Context, path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	rep, err := CheckFile(ctx, path)
	return err == nil && !rep.Partial && len(rep.Faulty) == 0
}

// createPart creates the temporary file the copy for target is written to, in target's
// directory, so the final rename never crosses a file system. It keeps target's extension:
// ffmpeg picks the container from it.
func createPart(target string) (string, error) {
	ext := filepath.Ext(target)
	base := strings.TrimSuffix(filepath.Base(target), ext)
	f, err := os.CreateTemp(filepath.Dir(target), "."+base+".*.part"+ext)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func remuxCacheDir(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "blunderDB", "remux"), nil
}

// cacheName is stable for one state of one file, and changes when the file is replaced or
// rewritten, so a stale copy is never served for a new recording.
func cacheName(abs string, st os.FileInfo) string {
	h := sha256.Sum256([]byte(abs + "\x00" + strconv.FormatInt(st.Size(), 10) + "\x00" +
		strconv.FormatInt(st.ModTime().UnixNano(), 10)))
	ext := filepath.Ext(abs)
	return strings.TrimSuffix(filepath.Base(abs), ext) + "-" + hex.EncodeToString(h[:8]) + ".remux" + ext
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
