package receipt

import "testing"

func TestMatchLines(t *testing.T) {
	items := []string{"melk", "ei", "bananen", "halfvolle melk"}
	cases := map[string]string{
		"AH HALFV MELK 1L":     "melk",
		"AH HALFVOLLE MELK 1L": "halfvolle melk",
		"MELKCHOCOLADE":        "",
		"PREI":                 "",
		"AH BANANEN":           "bananen",
		"VRIJE UITLOOP EI 10":  "ei",
	}
	for line, want := range cases {
		m := MatchLines([]string{line}, items)[0]
		if (want == "") == m.Matched || (want != "" && m.Item != want) {
			t.Errorf("%q → %+v, want %q", line, m, want)
		}
	}
}
