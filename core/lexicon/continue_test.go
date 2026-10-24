package lexicon

import (
	"slices"
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func continueEntries() []Entry {
	var es []Entry
	for _, w := range []struct{ s, r string }{
		{"中國", "ㄓㄨㄥ ㄍㄨㄛˊ"}, {"中國人", "ㄓㄨㄥ ㄍㄨㄛˊ ㄖㄣˊ"}, {"中共", "ㄓㄨㄥ ㄍㄨㄥˋ"}, {"中", "ㄓㄨㄥ"},
		{"主顧", "ㄓㄨˇ ㄍㄨˋ"}, {"主", "ㄓㄨˇ"}, {"這個", "ㄓㄜˋ ㄍㄜ˙"}, {"知", "ㄓ"}, {"真的", "ㄓㄣ ㄉㄜ˙"},
		{"資格", "ㄗ ㄍㄜˊ"}, {"喔", "ㄛ"}, {"國", "ㄍㄨㄛˊ"}, {"ㄩ", "ㄩ"}, {"女", "ㄋㄩˇ"},
	} {
		es = append(es, Entry{Surface: w.s, Reading: zhuyin.ParseReading(w.r), PRW: 1})
	}
	return es
}

// bruteContinuations 逐一試詞庫的所有音節與 base 的所有展開。
func bruteContinuations(src Source, all []zhuyin.Syllable, syls []zhuyin.Syllable, base zhuyin.Syllable) []zhuyin.Syllable {
	cands := append(slices.Clone(all), zhuyin.Syllable(string(base)+"*").Completions()...)
	var out []zhuyin.Syllable
	for _, y := range cands {
		if strings.HasPrefix(string(y), string(base)) && src.HasPrefix(append(slices.Clone(syls), y)) {
			out = append(out, y)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func collect(f func(fn func(zhuyin.Syllable))) []zhuyin.Syllable {
	var out []zhuyin.Syllable
	f(func(y zhuyin.Syllable) { out = append(out, y) })
	slices.Sort(out)
	return slices.Compact(out)
}

func TestContinuationsMatchHasPrefix(t *testing.T) {
	es := continueEntries()
	c, err := Build(es)
	if err != nil {
		t.Fatal(err)
	}
	mem := NewMem()
	for _, e := range es {
		mem.Add(e)
	}
	var all []zhuyin.Syllable
	for _, e := range es {
		all = append(all, e.Reading...)
	}
	all = append(all, "ㄛ˙", "ㄩ˙")
	sources := map[string]Source{"compact": c, "mem": mem, "neutral": NeutralParticles(c), "neutral-mem": NeutralParticles(mem),
		"generic": struct{ Source }{c}}
	prefixes := [][]zhuyin.Syllable{nil, {"ㄓㄨㄥ"}, {"ㄓㄨㄥ", "ㄍㄨㄛˊ"}, {"ㄓㄨˇ"}, {"ㄗ"}, {"ㄅㄩ"}, {"ㄍㄨㄛˊ"}}
	bases := []zhuyin.Syllable{"ㄓ", "ㄓㄨ", "ㄓㄨㄥ", "ㄍ", "ㄍㄨ", "ㄖ", "ㄛ", "ㄩ", "ㄋ", "ㄅㄩ", "ㄗ", "ㄉ"}
	for name, src := range sources {
		for _, p := range prefixes {
			for _, b := range bases {
				want := bruteContinuations(src, all, p, b)
				got := collect(func(fn func(zhuyin.Syllable)) { Continue(src, p, b, fn) })
				if !slices.Equal(got, want) {
					t.Errorf("%s：%v 之後以 %q 開頭：%v，要 %v", name, p, b, got, want)
				}
			}
		}
	}
	if got := collect(func(fn func(zhuyin.Syllable)) { Continue(NeutralParticles(c), nil, "ㄛ", fn) }); !slices.Contains(got, "ㄛ˙") {
		t.Fatalf("句首的 ㄛ 要包含補語氣詞的 ㄛ˙：%v", got)
	}
}
