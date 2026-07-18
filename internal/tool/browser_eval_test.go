package tool

import "testing"

func TestWrapBrowserEvalJS(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"document.title", "() => (document.title)"},
		{"JSON.stringify({a:1})", "() => (JSON.stringify({a:1}))"},
		{"() => document.title", "() => document.title"},
		{"(a,b) => a+b", "(a,b) => a+b"},
		{"async () => 1", "async () => 1"},
		{"function(){return 1}", "function(){return 1}"},
		{"x => x", "x => x"},
	}
	for _, c := range cases {
		got := wrapBrowserEvalJS(c.in)
		if got != c.want {
			t.Fatalf("wrapBrowserEvalJS(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
