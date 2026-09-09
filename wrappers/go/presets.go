package guardcompress

func Image(inPath string, opts map[string]any) (Result, error) {
	return Process(inPath, withDefault(opts, "allow_ext",
		[]any{"jpg", "jpeg", "png", "webp", "gif"}))
}

func Video(inPath string, opts map[string]any) (Result, error) {
	return Process(inPath, withDefault(opts, "allow_ext",
		[]any{"mp4", "mov", "webm", "mkv", "avi"}))
}

func Audio(inPath string, opts map[string]any) (Result, error) {
	return Process(inPath, withDefault(opts, "allow_ext",
		[]any{"mp3", "wav", "ogg", "oga", "m4a", "flac"}))
}

func withDefault(opts map[string]any, key string, val any) map[string]any {
	out := map[string]any{}
	for k, v := range opts {
		out[k] = v
	}
	if _, ok := out[key]; !ok {
		out[key] = val
	}
	return out
}
