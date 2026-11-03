package notify

import "testing"

func TestRenderPlaceholders(t *testing.T) {
	vars := map[string]any{"name": "XBoard", "empty": "", "flag": true, "n": 3.0}
	cases := map[string]string{
		"hi {{name}}":          "hi XBoard",
		"x {{ missing }} y":    "x {{ missing }} y",
		"x {{missing|none}} y": "x none y",
		"x {{empty|dflt}}":     "x dflt",
		"{{flag}}-{{n}}":       "1-3",
		"no vars here":         "no vars here",
	}
	for in, want := range cases {
		if got := RenderPlaceholders(in, vars); got != want {
			t.Errorf("%q: want %q got %q", in, want, got)
		}
	}
	if got := RenderPlaceholders("{{a}}", nil); got != "{{a}}" {
		t.Errorf("no vars should return template unchanged, got %q", got)
	}
}
