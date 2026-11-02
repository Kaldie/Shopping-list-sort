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

func TestParseNameAbovePrice(t *testing.T) {
	text := "Aardappelschijfjes\n0,99\nJumbo Wortelen Ef\n1,05\nJumbo 100% Appelmoes\n1,19\n" +
		"Mandarijnen\n2,19\nActie Mandarijnen\n-0,20\nHalfvolle Melk Uht\n2 X 0,84\n1,68\n" +
		"Baguette Wit 2st\n0,67\nRozijnen Kr Bol 6st\n1,49\n"
	got := Parse(text)
	want := []string{"Aardappelschijfjes", "Wortelen Ef", "Appelmoes", "Mandarijnen", "Halfvolle Melk Uht", "Baguette Wit", "Rozijnen Kr Bol"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}
