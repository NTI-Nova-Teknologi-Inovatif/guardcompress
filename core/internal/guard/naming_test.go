package guard

import "testing"

func TestReservedNames(t *testing.T) {
	mime := map[string]string{
		".jpg": "image/jpeg", ".png": "image/png",
		".mp4": "video/mp4", ".gif": "image/gif",
	}
	for _, tc := range []struct{ in, want string }{
		{"CON.jpg", "file_CON.jpg"},
		{"nul.png", "file_nul.png"},
		{"COM1.mp4", "file_COM1.mp4"},
		{"lpt9.gif", "file_lpt9.gif"},
		{"AUX", "file_AUX.jpg"},
		{"foto.jpg", "foto.jpg"},
		{"report.png", "report.png"},
	} {
		ext := ".jpg"
		if len(tc.in) > 4 {
			if e := tc.in[len(tc.in)-4:]; e == ".png" || e == ".mp4" || e == ".gif" || e == ".jpg" {
				ext = e
			}
		}
		got := OutputName(tc.in, mime[ext], map[string]any{})
		if got != tc.want {
			t.Fatalf("%s -> %s, mau %s", tc.in, got, tc.want)
		}
	}
}
