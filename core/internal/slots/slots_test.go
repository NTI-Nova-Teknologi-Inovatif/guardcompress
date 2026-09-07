package slots

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireUpToMax(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "slots")
	var rels []func()
	for range 3 {
		rel, err := Acquire(dir, 3)
		if err != nil {
			t.Fatalf("slot 1-3 harus dapat: %v", err)
		}
		rels = append(rels, rel)
	}
	if _, err := Acquire(dir, 3); err != ErrBusy {
		t.Fatal("slot ke-4 harus busy")
	}
	rels[0]()
	if _, err := Acquire(dir, 3); err != nil {
		t.Fatalf("setelah release harus dapat: %v", err)
	}
	for _, r := range rels[1:] {
		r()
	}
}

func TestUnlimited(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "slots")
	for range 5 {
		if _, err := Acquire(dir, 0); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStaleReclaimed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "slots")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "slot-yatim")
	if err := os.WriteFile(stale, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Skip("chtimes tak didukung: " + err.Error())
	}
	if _, err := Acquire(dir, 1); err != nil {
		t.Fatalf("lock yatim harus dibersihkan: %v", err)
	}
}
