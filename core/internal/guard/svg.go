package guard

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

const (
	svgMaxBytes = 8 << 20
	svgMaxDepth = 64
)

var svgSafeElements = map[string]bool{
	"svg": true, "g": true, "defs": true, "symbol": true, "use": true,
	"path": true, "rect": true, "circle": true, "ellipse": true,
	"line": true, "polyline": true, "polygon": true,
	"text": true, "tspan": true, "textpath": true,
	"lineargradient": true, "radialgradient": true, "stop": true,
	"clippath": true, "mask": true, "pattern": true,
	"title": true, "desc": true, "metadata": true,
}

func isSVG(head []byte) bool {
	p := skipXMLPrefix(head)
	if len(p) < 4 {
		return false
	}
	if p[0] != '<' {
		return false
	}
	tag := strings.ToLower(string(p[1:min(len(p), 8)]))
	return strings.HasPrefix(tag, "svg") &&
		(len(tag) == 3 || tag[3] == ' ' || tag[3] == '>' || tag[3] == '/' || tag[3] == '\t' || tag[3] == '\n' || tag[3] == '\r')
}

func skipXMLPrefix(b []byte) []byte {
	for {
		b = skipSpace(b)
		if bytes.HasPrefix(b, []byte("<?")) {
			if i := bytes.Index(b, []byte("?>")); i >= 0 {
				b = b[i+2:]
				continue
			}
			return b
		}
		if bytes.HasPrefix(b, []byte("<!--")) {
			if i := bytes.Index(b, []byte("-->")); i >= 0 {
				b = b[i+3:]
				continue
			}
			return b
		}
		if bytes.HasPrefix(b, []byte("<!DOCTYPE")) || bytes.HasPrefix(b, []byte("<!doctype")) {
			return nil
		}
		return b
	}
}

func skipSpace(b []byte) []byte {
	i := 0
	for i < len(b) {
		c := b[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == 0xEF || c == 0xFE || c == 0xFF {
			i++
			continue
		}
		break
	}
	return b[i:]
}

func SanitizeSVG(data []byte) ([]byte, error) {
	if len(data) > svgMaxBytes {
		return nil, fmt.Errorf("svg too large")
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	dec.Entity = xml.HTMLEntity
	var out bytes.Buffer
	depth := 0
	root := false
	limited := &limitWriter{w: &out, n: svgMaxBytes}
	enc := xml.NewEncoder(limited)
	seenRoot := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth > svgMaxDepth {
				return nil, fmt.Errorf("svg too deep")
			}
			name := strings.ToLower(t.Name.Local)
			if !seenRoot {
				seenRoot = true
				if name != "svg" {
					return nil, fmt.Errorf("svg bad root")
				}
				root = true
			}
			if !svgSafeElements[name] {
				if err := skipElement(dec, &depth); err != nil {
					return nil, err
				}
				continue
			}
			var attrs []xml.Attr
			for _, a := range t.Attr {
				kept, ok := sanitizeAttr(a)
				if !ok {
					continue
				}
				attrs = append(attrs, kept)
			}
			if t.Name.Space == "http://www.w3.org/2000/svg" {
				t.Name.Space = ""
			}
			if name == "svg" && !hasXMLNS(attrs) {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: "http://www.w3.org/2000/svg"})
			}
			t.Attr = attrs
			if err := enc.EncodeToken(t); err != nil {
				return nil, err
			}
		case xml.EndElement:
			depth--
			if t.Name.Space == "http://www.w3.org/2000/svg" {
				t.Name.Space = ""
			}
			if err := enc.EncodeToken(t); err != nil {
				return nil, err
			}
		case xml.CharData:
			if err := enc.EncodeToken(t); err != nil {
				return nil, err
			}
		default:
			continue
		}
		if limited.over {
			return nil, fmt.Errorf("svg output too large")
		}
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	if !root {
		return nil, fmt.Errorf("svg no root")
	}
	return out.Bytes(), nil
}

func skipElement(dec *xml.Decoder, depth *int) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			*depth++
			if *depth > svgMaxDepth {
				return fmt.Errorf("svg too deep")
			}
		case xml.EndElement:
			*depth--
			return nil
		}
	}
}

func sanitizeAttr(a xml.Attr) (xml.Attr, bool) {
	local := strings.ToLower(a.Name.Local)
	space := strings.ToLower(a.Name.Space)
	if space == "xmlns" {
		return a, true
	}
	if strings.HasPrefix(local, "on") {
		return a, false
	}
	switch local {
	case "href":
		if space != "" && space != "http://www.w3.org/1999/xlink" {
			return a, false
		}
		v := strings.TrimSpace(strings.ToLower(a.Value))
		if strings.HasPrefix(v, "#") || strings.HasPrefix(v, "data:image/") {
			return a, true
		}
		return a, false
	case "style", "formaction", "xlink:href":
		if local == "style" {
			v := strings.ToLower(a.Value)
			if strings.Contains(v, "javascript:") || strings.Contains(v, "expression(") ||
				strings.Contains(v, "binding(") {
				return a, false
			}
			return a, true
		}
		return a, false
	}
	if space != "" && space != "http://www.w3.org/1999/xlink" && space != "http://www.w3.org/XML/1998/namespace" {
		return a, false
	}
	return a, true
}

func hasXMLNS(attrs []xml.Attr) bool {
	for _, a := range attrs {
		if strings.ToLower(a.Name.Local) == "xmlns" {
			return true
		}
	}
	return false
}

type limitWriter struct {
	w    *bytes.Buffer
	n    int
	over bool
}

func (l *limitWriter) Write(p []byte) (int, error) {
	if l.w.Len()+len(p) > l.n {
		l.over = true
		return 0, fmt.Errorf("limit")
	}
	return l.w.Write(p)
}
