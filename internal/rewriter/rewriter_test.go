package rewriter

import (
	"encoding/base64"
	"testing"
)

func TestRewrite(t *testing.T) {
	r, err := New("https://cdn.example.com")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cases := []struct {
		name   string
		base   string
		input  string
		output string
	}{
		{
			name:  "relative and absolute",
			base:  "https://origin.example.com/live/playlist.m3u8",
			input: "#EXTM3U\nseg-1.ts\nhttps://origin.example.com/live/seg-2.ts\n",
			output: "#EXTM3U\n" +
				"https://cdn.example.com/seg?payload=" + encode("https://origin.example.com/live/seg-1.ts") + "\n" +
				"https://cdn.example.com/seg?payload=" + encode("https://origin.example.com/live/seg-2.ts") + "\n",
		},
		{
			name:   "keep tags",
			base:   "https://origin.example.com/live/playlist.m3u8",
			input:  "#EXTM3U\n#EXT-X-VERSION:3\nseg.ts\n",
			output: "#EXTM3U\n#EXT-X-VERSION:3\nhttps://cdn.example.com/seg?payload=" + encode("https://origin.example.com/live/seg.ts") + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.Rewrite(tc.input, tc.base)
			if err != nil {
				t.Fatalf("rewrite failed: %v", err)
			}
			if got != tc.output {
				t.Fatalf("unexpected output:\nwant: %q\n got: %q", tc.output, got)
			}
		})
	}
}

func TestNew(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatalf("expected error")
	}
}

func BenchmarkRewrite(b *testing.B) {
	r, err := New("https://cdn.example.com")
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}
	base := "https://origin.example.com/live/playlist.m3u8"
	input := "#EXTM3U\nseg-1.ts\nseg-2.ts\nseg-3.ts\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.Rewrite(input, base); err != nil {
			b.Fatalf("rewrite failed: %v", err)
		}
	}
}

func encode(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}
