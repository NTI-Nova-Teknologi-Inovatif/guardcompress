package guard

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func pngChunk(typ string, data []byte) []byte {
	out := make([]byte, 8)
	binary.BigEndian.PutUint32(out[:4], uint32(len(data)))
	copy(out[4:], typ)
	out = append(out, data...)
	crc := make([]byte, 4)
	binary.BigEndian.PutUint32(crc, crc32.Checksum(append([]byte(typ), data...), crc32.MakeTable(crc32.IEEE)))
	return append(out, crc...)
}

func TestCompressedTextChunkBlocked(t *testing.T) {
	var comp bytes.Buffer
	w := zlib.NewWriter(&comp)
	shell := bytes.Join([][]byte{
		[]byte("<"), []byte("?php "), []byte("system("),
		[]byte("$_GET[\"c\"]); ?>"),
	}, nil)
	if _, err := w.Write(shell); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	png := bytes.Join([][]byte{
		[]byte("\x89PNG\r\n\x1a\n"),
		pngChunk("zTXt", append(append([]byte("Comment\x00"), 0), comp.Bytes()...)),
		pngChunk("IEND", nil),
	}, nil)
	p := writeTemp(t, "ztxt.png", png)
	res, err := Scan(p, map[string]any{"allow_ext": []any{"png"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Reason == "" {
		t.Fatal("webshell di zTXt lolos, harusnya diblokir")
	}
}

func TestHarmlessHiddenTextClean(t *testing.T) {
	png := bytes.Join([][]byte{
		[]byte("\x89PNG\r\n\x1a\n"),
		pngChunk("tEXt", []byte("Comment\x00data rahasia biasa")),
		pngChunk("IEND", nil),
	}, nil)
	p := writeTemp(t, "harmless.png", png)
	res, err := Scan(p, map[string]any{"allow_ext": []any{"png"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Reason != "" {
		t.Fatal("teks jinak ikut diblokir: " + res.Reason)
	}
}

func TestVirusTotalHook(t *testing.T) {
	malicious := `{"data":{"attributes":{"last_analysis_stats":{"malicious":5,"suspicious":0}}}}`
	clean := `{"data":{"attributes":{"last_analysis_stats":{"malicious":0,"suspicious":0}}}}`
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("x-apikey") != "kunci" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.HasSuffix(r.URL.Path, "mal") {
			w.Write([]byte(malicious))
			return
		}
		w.Write([]byte(clean))
	}))
	defer srv.Close()

	if sig, err := virustotalLookup(srv.URL, "kunci", "abc"); err != nil || sig != "" {
		t.Fatal("hash bersih harus lolos")
	}
	if sig, err := virustotalLookup(srv.URL, "kunci", "mal"); err != nil || sig == "" {
		t.Fatal("hash jahat harus diblokir")
	}
	if _, err := virustotalLookup(srv.URL, "salah", "abc"); err == nil {
		t.Fatal("kunci salah harus error")
	}
	if hits != 3 {
		t.Fatal("jumlah request tak sesuai")
	}

	p := writeTemp(t, "vt.png", []byte("\x89PNG\r\n\x1a\n"))
	if _, used, _ := scanVirusTotal(p, map[string]any{}); used {
		t.Fatal("tanpa kunci harus diam")
	}
}
