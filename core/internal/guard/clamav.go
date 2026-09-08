package guard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func scanClamAV(path string, cfg map[string]any) (string, bool, error) {
	mode := "auto"
	if v, ok := cfg["clamav"]; ok {
		switch t := v.(type) {
		case bool:
			if !t {
				return "", false, nil
			}
			mode = "force"
		case string:
			if t == "false" || t == "off" || t == "no" {
				return "", false, nil
			}
			if t == "force" || t == "true" {
				mode = "force"
			}
		}
	}
	bin := ""
	if p, err := exec.LookPath("clamdscan"); err == nil {
		bin = p
	} else if p, err := exec.LookPath("clamscan"); err == nil {
		bin = p
	}
	if bin == "" {
		if mode == "force" {
			return "", false, fmt.Errorf("clamav diminta tapi clamdscan/clamscan tak ada di PATH")
		}
		return "", false, nil // auto + absen = diam
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--no-summary", path).CombinedOutput()
	s := string(out)
	if strings.Contains(s, "FOUND") {
		return parseClamSig(s), true, nil
	}
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		return "", false, fmt.Errorf("clamav timeout")
	}
	return "", true, nil
}

func parseClamSig(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "FOUND") {
			continue
		}
		rest := strings.TrimSpace(strings.Replace(line, "FOUND", "", 1))
		if i := strings.LastIndex(rest, ":"); i >= 0 {
			return strings.TrimSpace(rest[i+1:])
		}
		return rest
	}
	return "unknown-signature"
}
