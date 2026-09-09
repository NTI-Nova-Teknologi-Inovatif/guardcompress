package guard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func scanVirusTotal(path string, cfg map[string]any) (string, bool, error) {
	key, _ := cfg["virustotal_api_key"].(string)
	if key == "" {
		return "", false, nil
	}
	base, _ := cfg["virustotal_base"].(string)
	if base == "" {
		base = "https://www.virustotal.com/api/v3"
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return "", false, err
	}
	sig, err := virustotalLookup(base, key, sum)
	if err != nil {
		if failClosed(cfg) {
			return "", false, err
		}
		return "", true, nil
	}
	if sig != "" {
		return sig, true, nil
	}
	return "", true, nil
}

func failClosed(cfg map[string]any) bool {
	v, _ := cfg["virustotal_fail_closed"].(bool)
	return v
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func virustotalLookup(base, key, sha string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/files/"+sha, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-apikey", key)
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("virustotal HTTP %d", res.StatusCode)
	}
	var body struct {
		Data struct {
			Attributes struct {
				Stats struct {
					Malicious  int `json:"malicious"`
					Suspicious int `json:"suspicious"`
				} `json:"last_analysis_stats"`
			} `json:"attributes"`
		} `json:"data"`
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", err
	}
	st := body.Data.Attributes.Stats
	if st.Malicious > 0 {
		return fmt.Sprintf("virustotal: %d engine jahat", st.Malicious), nil
	}
	if st.Suspicious > 0 {
		return fmt.Sprintf("virustotal: %d engine curiga", st.Suspicious), nil
	}
	return "", nil
}
