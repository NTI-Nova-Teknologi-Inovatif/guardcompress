package guard

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestShortTagWithPayloadBlocked(t *testing.T) {
	// JPEG dengan webshell di segmen COM, dirakit dari potongan agar
	// source test ini sendiri tidak cocok signature AV.
	com := bytes.Join([][]byte{
		[]byte("\xff\xfe\x00\x12"),
		[]byte("<"), []byte("?="), []byte("`"),
		[]byte("$_"), []byte("GET[x]"), []byte("`"),
		[]byte("?>"), []byte("\n"),
	}, nil)
	jpg := bytes.Join([][]byte{
		[]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), com,
		[]byte("\xff\xd9"),
	}, nil)
	p := writeTemp(t, "shell.jpg", jpg)
	res, err := Scan(p, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Reason == "" {
		t.Fatal("shell COM lolos, harusnya diblokir")
	}
}

func TestAspShortTagWithPayloadBlocked(t *testing.T) {
	png := bytes.Join([][]byte{
		[]byte("\x89PNG\r\n\x1a\n"),
		[]byte("<"), []byte("%eval(request(\"x\"))%"), []byte(">"),
		[]byte("\n"),
	}, nil)
	p := writeTemp(t, "asp.png", png)
	res, err := Scan(p, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Reason == "" {
		t.Fatal("shell ASP lolos, harusnya diblokir")
	}
}

func TestXmlDeclarationClean(t *testing.T) {
	png := bytes.Join([][]byte{
		[]byte("\x89PNG\r\n\x1a\n"),
		[]byte("<"), []byte("?xml version=\"1.0\"?"), []byte(">"),
		[]byte("\n"),
	}, nil)
	p := writeTemp(t, "xml.png", png)
	res, err := Scan(p, map[string]any{"allow_ext": []any{"png"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Reason != "" {
		t.Fatal("deklarasi XML ikut diblokir: " + res.Reason)
	}
}
