package source

import "testing"

func TestStripHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plain text untouched", in: "Refined mineral.", want: "Refined mineral."},
		{name: "br converted to newline", in: "Line one.<br>Line two.", want: "Line one.\nLine two."},
		{name: "paragraphs preserved as blank line", in: "<p>First paragraph.</p><p>Second paragraph.</p>", want: "First paragraph.\n\nSecond paragraph."},
		{name: "showinfo link keeps text", in: `See <a href="showinfo:34">Tritanium</a> for details.`, want: "See Tritanium for details."},
		{name: "font and color tags stripped", in: `<font size="13" color="#bfffffff">Refined</font> <b>mineral</b>.`, want: "Refined mineral."},
		{name: "html entities decoded", in: "Tritanium &amp; Pyerite &lt;mineral&gt;", want: "Tritanium & Pyerite <mineral>"},
		{name: "non-breaking space collapsed", in: "Refined&nbsp;mineral.", want: "Refined mineral."},
		{name: "multiple consecutive br collapsed", in: "First.<br><br><br><br>Second.", want: "First.\n\nSecond."},
		{name: "leading and trailing whitespace trimmed", in: "   <p>Hello.</p>   ", want: "Hello."},
		{name: "self-closing br variants", in: `Line<br/>Two<BR />Three`, want: "Line\nTwo\nThree"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stripHTML(tt.in)
			if got != tt.want {
				t.Fatalf("stripHTML(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
