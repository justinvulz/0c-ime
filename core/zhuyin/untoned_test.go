package zhuyin

import (
	"slices"
	"strings"
	"testing"
)

func TestUntoned(t *testing.T) {
	y := Untone("ㄋㄧ")
	if !y.Untoned() || y.Valid() || y.Legal() || y.ToneOf() != 0 || y.Base() != "ㄋㄧ" || y.Text() != "ㄋㄧ" {
		t.Fatalf("%q：Untoned %v Valid %v Legal %v ToneOf %d Base %q Text %q", y, y.Untoned(), y.Valid(), y.Legal(), y.ToneOf(), y.Base(), y.Text())
	}
	if Untone("ㄋㄧˇ") != y {
		t.Fatal("Untone 去掉原本的聲調")
	}
	if want := []Syllable{"ㄋㄧ", "ㄋㄧˊ", "ㄋㄧˇ", "ㄋㄧˋ", "ㄋㄧ˙"}; !slices.Equal(y.Tones(), want) {
		t.Fatalf("五個聲調：%v", y.Tones())
	}
	for _, s := range []Syllable{"ㄋㄧ", "ㄋㄧˇ", "?", "ˇ?", "ㄋㄧˇ?", ""} {
		if s.Untoned() {
			t.Errorf("%q 不是沒打聲調的音節", s)
		}
		if s != "" && !slices.Equal(s.Tones(), []Syllable{s}) {
			t.Errorf("%q 的 Tones 只有自己", s)
		}
	}
	if Syllable("?").Base() != "?" || Syllable("?").Text() != "?" {
		t.Fatal("字面的 ? 不受影響")
	}
	if !HasUntoned([]Syllable{"ㄨㄛˇ", y}) || HasUntoned([]Syllable{"ㄨㄛˇ", "ㄋㄧ"}) {
		t.Fatal("HasUntoned")
	}
}

func TestLegalPrefix(t *testing.T) {
	for _, s := range []Syllable{"ㄒ", "ㄒㄧ", "ㄒㄧㄢ", "ㄓㄨ", "ㄢ", "ㄧ", "ㄧㄢ", "ㄈㄧ"} {
		if !LegalPrefix(s) {
			t.Errorf("%q 是音節的開頭", s)
		}
	}
	for _, s := range []Syllable{"ㄉㄧㄤ", "ㄅㄩ", "ㄒㄚ", "ㄒㄧˋ", ""} {
		if LegalPrefix(s) {
			t.Errorf("%q 不是任何音節的開頭", s)
		}
	}
}

func TestAbbreviate(t *testing.T) {
	for base, want := range map[Syllable]Syllable{
		"ㄓ": "ㄓ*", "ㄍ": "ㄍ*", "ㄓㄨ": "ㄓㄨ*", "ㄋㄧ": "ㄋㄧ*", "ㄈㄧ": "ㄈㄧ*", "ㄓㄨˋ": "ㄓㄨ*",
		"ㄓㄨㄥ": "ㄓㄨㄥ?", "ㄏㄠ": "ㄏㄠ?", "ㄅㄩ": "ㄅㄩ?",
	} {
		if got := Abbreviate(base); got != want {
			t.Errorf("Abbreviate(%q) = %q，要 %q", base, got, want)
		}
	}
	y := Abbreviate("ㄓㄨ")
	if !y.Abbreviated() || !y.Untoned() || y.Valid() || y.Legal() || y.ToneOf() != 0 || y.Base() != "ㄓㄨ" || y.Text() != "ㄓㄨ" {
		t.Fatalf("%q：Abbreviated %v Untoned %v Valid %v Legal %v ToneOf %d Base %q Text %q",
			y, y.Abbreviated(), y.Untoned(), y.Valid(), y.Legal(), y.ToneOf(), y.Base(), y.Text())
	}
	if want := []Syllable{"ㄓㄨ", "ㄓㄨˊ", "ㄓㄨˇ", "ㄓㄨˋ", "ㄓㄨ˙"}; !slices.Equal(y.Tones(), want) {
		t.Fatalf("Tones 只是打出的注音的五個聲調：%v", y.Tones())
	}
	if Untone("ㄓㄨ").Abbreviated() || Syllable("*").Untoned() || Syllable("ㄓㄨˇ*").Untoned() || Syllable("*").Base() != "*" {
		t.Fatal("只有注音後面接 * 才是打一半的音節")
	}
	if !HasUntoned([]Syllable{"ㄨㄛˇ", "ㄍ*"}) {
		t.Fatal("HasUntoned 包含打一半的音節")
	}
}

func TestCompletions(t *testing.T) {
	got := Syllable("ㄓㄨ*").Completions()
	if len(got) != 5*len(CompletionBases("ㄓㄨ")) || got[0] != "ㄓㄨ" || !slices.IsSorted(got) {
		t.Fatalf("ㄓㄨ* 展開成以 ㄓㄨ 開頭的音節的五個聲調、排序、ㄓㄨ 排第一：%v", got)
	}
	for _, want := range []Syllable{"ㄓㄨㄥ", "ㄓㄨㄥˋ", "ㄓㄨㄛˊ", "ㄓㄨㄤ˙"} {
		if !slices.Contains(got, want) {
			t.Errorf("ㄓㄨ* 應包含 %q", want)
		}
	}
	for _, y := range Syllable("ㄓ*").Completions() {
		if !strings.HasPrefix(string(y), "ㄓ") || !y.Legal() {
			t.Errorf("ㄓ* 不應包含 %q", y)
		}
	}
	if b := CompletionBases("ㄍ"); slices.Contains(b, "ㄍ") || !slices.Contains(b, "ㄍㄨㄛ") {
		t.Fatalf("ㄍ 不是音節，但可以接成 ㄍㄨㄛ：%v", b)
	}
	if want := Untone("ㄋㄧ").Tones(); !slices.Equal(Untone("ㄋㄧ").Completions(), want) {
		t.Fatal("沒打聲調的音節的 Completions 就是 Tones")
	}
}

func TestCovers(t *testing.T) {
	for _, c := range []struct {
		typed, y Syllable
		want     bool
	}{
		{"ㄓ*", "ㄓㄨㄥ", true}, {"ㄓ*", "ㄓˋ", true}, {"ㄓ*", "ㄗㄨㄥ", false},
		{"ㄓㄨ*", "ㄓㄨㄥˋ", true}, {"ㄓㄨ*", "ㄓㄜˋ", false},
		{"ㄓㄨ?", "ㄓㄨˇ", true}, {"ㄓㄨ?", "ㄓㄨㄥ", false},
		{"ㄓㄨˇ", "ㄓㄨˇ", true}, {"ㄓㄨˇ", "ㄓㄨ", false},
	} {
		if got := c.typed.Covers(c.y); got != c.want {
			t.Errorf("%q.Covers(%q) = %v", c.typed, c.y, got)
		}
	}
}
