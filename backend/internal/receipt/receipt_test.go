package receipt

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	text := `ALBERT HEIJN
Datum: 30-10-2026
BANANEN            1,99
2 x HALFVOLLE MELK  2,38
0,512 kg x 2,99
KIPFILET           5,49 B
HALFVOLLE MELK     1,19
BONUS KORTING     -0,50
TOTAAL            10,55
PIN               10,55
Bedankt en tot ziens`
	got := Parse(text)
	want := []string{"BANANEN", "HALFVOLLE MELK", "KIPFILET"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParsePlainList(t *testing.T) {
	got := Parse("bread\nmilk\n")
	if !reflect.DeepEqual(got, []string{"bread", "milk"}) {
		t.Fatalf("got %q", got)
	}
}
