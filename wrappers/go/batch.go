package guardcompress

import "sync"

type BatchItem struct {
	Path string
	Opts map[string]any
}

type BatchResult struct {
	Path   string
	Report map[string]any
	Err    error
}

func (b BatchResult) IsBlocked() bool {
	return IsBlocked(b.Err)
}

func isBusyReport(report map[string]any) bool {
	if report == nil {
		return false
	}
	d, ok := report["details"].(map[string]any)
	if !ok {
		return false
	}
	b, _ := d["busy"].(bool)
	return b
}

func IsBlocked(err error) bool {
	return err != nil && len(err.Error()) >= 7 && err.Error()[:7] == "blocked"
}

func IsBusy(err error) bool {
	return err != nil && len(err.Error()) >= 5 && err.Error()[:5] == "busy:"
}

func Batch(items []BatchItem, opts map[string]any) []BatchResult {
	jobs := 1
	if v, ok := opts["jobs"]; ok {
		switch n := v.(type) {
		case int:
			if n > 1 {
				jobs = n
			}
		case float64:
			if n > 1 {
				jobs = int(n)
			}
		}
	}
	out := make([]BatchResult, len(items))
	if jobs == 1 {
		for i, it := range items {
			out[i] = runOne(it, opts)
			if out[i].Err != nil && !out[i].IsBlocked() {
				return out[:i+1]
			}
		}
		return out
	}
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for i, it := range items {
		wg.Add(1)
		go func(i int, it BatchItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = runOne(it, opts)
		}(i, it)
	}
	wg.Wait()
	return out
}

func runOne(it BatchItem, opts map[string]any) BatchResult {
	merged := map[string]any{}
	for k, v := range opts {
		if k == "jobs" {
			continue
		}
		merged[k] = v
	}
	for k, v := range it.Opts {
		merged[k] = v
	}
	r, err := Process(it.Path, merged)
	return BatchResult{Path: r.Path, Report: r.Report, Err: err}
}
