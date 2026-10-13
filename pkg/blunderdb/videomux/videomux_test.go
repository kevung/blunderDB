package videomux

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"
)

// delta-cluster.webm: 4 s of VP9 + Opus, keyframes at 0 and 2.5 s, clusters cut every 0.7 s
// (ffmpeg -cluster_time_limit 700): the cluster holding the 2.5 s keyframe opens on a delta.
// remuxed.webm: the same file through RemuxCommand.

func TestCheckFile_FindsClusterOpeningOnDelta(t *testing.T) {
	rep, err := CheckFile(context.Background(), "testdata/delta-cluster.webm")
	if err != nil {
		t.Fatal(err)
	}
	if rep.NoCues || rep.Partial {
		t.Fatalf("report = %+v, want a complete one", rep)
	}
	if rep.Clusters != 2 {
		t.Errorf("Clusters = %d, want 2 (one per keyframe)", rep.Clusters)
	}
	if len(rep.Faulty) != 1 {
		t.Fatalf("Faulty = %v, want one cluster", rep.Faulty)
	}
	if f := rep.Faulty[0]; f <= 0 || f >= 2500*time.Millisecond {
		t.Errorf("faulty cluster starts at %v, want before the 2.5 s keyframe", f)
	}
}

func TestCheckFile_RemuxedFileIsClean(t *testing.T) {
	rep, err := CheckFile(context.Background(), "testdata/remuxed.webm")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Faulty) != 0 || rep.Clusters == 0 {
		t.Errorf("report = %+v, want referenced clusters and none faulty", rep)
	}
}

// The real recording that froze in WebKitGTK; read in place, never copied into the repository.
func TestCheckFile_RealRecording(t *testing.T) {
	path := os.Getenv("BLUNDERDB_WEBM_SAMPLE")
	if path == "" {
		t.Skip("BLUNDERDB_WEBM_SAMPLE names a local recording to check")
	}
	rep, err := CheckFile(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d clusters referenced, %d faulty: %v", rep.Clusters, len(rep.Faulty), rep.Faulty)
	if want := os.Getenv("BLUNDERDB_WEBM_SAMPLE_FAULTY"); want != "" && want != strconv.Itoa(len(rep.Faulty)) {
		t.Errorf("%d faulty clusters, want %s", len(rep.Faulty), want)
	}
}

func TestCheck_TruncatedAndForeign(t *testing.T) {
	data, err := os.ReadFile("testdata/delta-cluster.webm")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, 3, 40, 200, len(data) / 2, len(data) - 300, len(data) - 1} {
		_, _ = Check(context.Background(), bytes.NewReader(data[:n]), int64(n))
	}
	if _, err := Check(context.Background(), bytes.NewReader([]byte("not a video")), 11); !errors.Is(err, ErrNotMatroska) {
		t.Errorf("err = %v, want ErrNotMatroska", err)
	}
}

func TestCheck_CancelledContextIsPartial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rep, err := CheckFile(ctx, "testdata/delta-cluster.webm")
	if err == nil && !rep.Partial {
		t.Errorf("report = %+v, want an error or a partial report", rep)
	}
}

func TestRemuxCommand(t *testing.T) {
	cases := []struct{ in, goos, want string }{
		{"/v/ronde 4.webm", "linux", `ffmpeg -i '/v/ronde 4.webm' -map 0 -c copy -cluster_time_limit 30000 '/v/ronde 4.remux.webm'`},
		{"/v/l'été.mkv", "darwin", `ffmpeg -i '/v/l'\''été.mkv' -map 0 -c copy -cluster_time_limit 30000 '/v/l'\''été.remux.mkv'`},
		{`C:\v\a b.webm`, "windows", `ffmpeg -i "C:\v\a b.webm" -map 0 -c copy -cluster_time_limit 30000 "C:\v\a b.remux.webm"`},
	}
	for _, c := range cases {
		if got := RemuxCommand(c.in, c.goos); got != c.want {
			t.Errorf("RemuxCommand(%q, %s) =\n %s\nwant\n %s", c.in, c.goos, got, c.want)
		}
	}
}

func FuzzCheck(f *testing.F) {
	for _, name := range []string{"testdata/delta-cluster.webm", "testdata/remuxed.webm"} {
		data, err := os.ReadFile(name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
		f.Add(data[:len(data)/3])
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = Check(ctx, bytes.NewReader(data), int64(len(data)))
	})
}
