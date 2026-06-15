package money

import (
	"errors"
	"math"
	"testing"
)

func TestString(t *testing.T) {
	cases := map[Cents]string{
		0:        "0.00",
		5:        "0.05",
		50:       "0.50",
		123:      "1.23",
		12345:    "123.45",
		-12345:   "-123.45",
		-5:       "-0.05",
		100000:   "1000.00",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("Cents(%d).String() = %q, want %q", in, got, want)
		}
	}
}

func TestParse(t *testing.T) {
	cases := map[string]Cents{
		"0":        0,
		"123.45":   12345,
		"123,45":   12345,
		"-0.50":    -50,
		"-0,5":     -50,
		"1000":     100000,
		"  42.00 ": 4200,
		"+7.7":     770,
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q) erreur inattendue : %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("Parse(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseRoundTrip(t *testing.T) {
	for _, c := range []Cents{0, 1, -1, 99, 100, 12345, -67890, 100000000} {
		got, err := Parse(c.String())
		if err != nil {
			t.Fatalf("Parse(%q) : %v", c.String(), err)
		}
		if got != c {
			t.Errorf("round-trip %d -> %q -> %d", c, c.String(), got)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"", "abc", "1.234", "1.2.3", "12,345.6"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) aurait dû échouer", in)
		}
	}
}

func TestAddSub(t *testing.T) {
	sum, err := Add(12345, -345)
	if err != nil || sum != 12000 {
		t.Errorf("Add(12345,-345) = %d, %v ; want 12000, nil", sum, err)
	}
	diff, err := Sub(10000, 2500)
	if err != nil || diff != 7500 {
		t.Errorf("Sub(10000,2500) = %d, %v ; want 7500, nil", diff, err)
	}
}

func TestSum(t *testing.T) {
	total, err := Sum(100, 200, -50, 4200)
	if err != nil || total != 4450 {
		t.Errorf("Sum(...) = %d, %v ; want 4450, nil", total, err)
	}
	empty, err := Sum()
	if err != nil || empty != 0 {
		t.Errorf("Sum() = %d, %v ; want 0, nil", empty, err)
	}
}

func TestOverflow(t *testing.T) {
	if _, err := Add(math.MaxInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Errorf("Add(MaxInt64,1) aurait dû renvoyer ErrOverflow, got %v", err)
	}
	if _, err := Sub(math.MinInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Errorf("Sub(MinInt64,1) aurait dû renvoyer ErrOverflow, got %v", err)
	}
}

func TestAbsAndEuros(t *testing.T) {
	if Abs(-12345) != 12345 || Abs(12345) != 12345 {
		t.Error("Abs incorrect")
	}
	if FromUnits(50) != 5000 {
		t.Error("FromUnits incorrect")
	}
	if Cents(12345).Euros() != 123 {
		t.Error("Euros incorrect")
	}
}
