package marketplace

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
		{name: "showinfo link keeps text", in: `<a href="showinfo:34">Tritanium</a>`, want: "Tritanium"},
		{name: "html entities decoded", in: "Tritanium &amp; Pyerite", want: "Tritanium & Pyerite"},
		{name: "non-breaking space collapsed", in: "Refined&nbsp;mineral.", want: "Refined mineral."},
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
