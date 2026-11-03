package notify

import "testing"

func TestEscapeAndNl2br(t *testing.T) {
	if got := EscapeHTML(`<a href="x">'&'</a>`); got != "&lt;a href=&quot;x&quot;&gt;&#039;&amp;&#039;&lt;/a&gt;" {
		t.Fatalf("escape: %s", got)
	}
	if got := Nl2br("a\nb"); got != "a<br />\nb" {
		t.Fatalf("nl2br: %q", got)
	}
}

func TestBuildSafeVarsContentModes(t *testing.T) {
	v := BuildSafeVars(map[string]any{"name": "<X>", "content": "<b>hi</b>\nline", "content_mode": "text"})
	if v["name"] != "&lt;X&gt;" {
		t.Fatalf("scalar not escaped: %s", v["name"])
	}
	if v["content"] != "&lt;b&gt;hi&lt;/b&gt;<br />\nline" {
		t.Fatalf("text content: %q", v["content"])
	}
	raw := BuildSafeVars(map[string]any{"content": "<p>ok</p>"})
	if raw["content"] != "<p>ok</p>" {
		t.Fatalf("html content should stay raw: %s", raw["content"])
	}
}

func TestRenderDBTemplateFallsBackSubject(t *testing.T) {
	subj, body := RenderDBTemplate("orig", "", "hi {{name}}", map[string]any{"name": "XB"})
	if subj != "orig" || body != "hi XB" {
		t.Fatalf("got %q %q", subj, body)
	}
	subj, _ = RenderDBTemplate("orig", "New {{name}}", "", map[string]any{"name": "XB"})
	if subj != "New XB" {
		t.Fatalf("subject override: %q", subj)
	}
}
