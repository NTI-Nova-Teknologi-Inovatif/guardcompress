package guard

import (
	"strings"
	"testing"
)

func TestOutputNameOriginal(t *testing.T) {
	got := OutputName(`C:\upl\video liburan anak.mp4`, "video/mp4", map[string]any{})
	if got != "video_liburan_anak.mp4" {
		t.Fatalf("dapat %q", got)
	}
}

func TestOutputNameEvilNeutralized(t *testing.T) {
	// Nama ".php" tidak boleh lolos jadi extension output.
	got := OutputName("/tmp/evil.mp4.php", "video/mp4", map[string]any{})
	if !strings.HasSuffix(got, ".mp4") || strings.Contains(got, ".php") {
		t.Fatalf("berbahaya: %q", got)
	}
}

func TestOutputNameTraversalNeutralized(t *testing.T) {
	got := OutputName("/tmp/../../etc/passwd", "image/png", map[string]any{})
	if strings.Contains(got, "/") || strings.Contains(got, `\`) || !strings.HasSuffix(got, ".png") {
		t.Fatalf("berbahaya: %q", got)
	}
}

func TestOutputNameUUID(t *testing.T) {
	a := OutputName("/tmp/a.mp4", "video/mp4", map[string]any{"output": "uuid"})
	b := OutputName("/tmp/a.mp4", "video/mp4", map[string]any{"output": "uuid"})
	if a == b || !strings.HasSuffix(a, ".mp4") || len(a) != 16+4 {
		t.Fatalf("uuid harus unik 16 hex: %q vs %q", a, b)
	}
}

func TestOutputNameCustom(t *testing.T) {
	got := OutputName("/tmp/a.mp4", "audio/mpeg", map[string]any{"output": "podcast episode #1!"})
	if got != "podcast_episode_1.mp3" {
		t.Fatalf("dapat %q", got)
	}
}

func TestOutputNameEmptyFallsBack(t *testing.T) {
	got := OutputName("/tmp/.mp4", "video/mp4", map[string]any{})
	if !strings.HasSuffix(got, ".mp4") || len(got) <= 4 {
		t.Fatalf("dapat %q", got)
	}
}
