package decode

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/0c-biggo/0c-ime/core/input"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

type partialWord struct {
	w, r string
	p    float64
}

// partialFixture 用 Compact（有 Continuer）建詞庫；pen 是 input.CompletionPenalty，機率依它推算，調罰分時測試照樣成立。
func partialFixture(t testing.TB, words []partialWord) *Decoder {
	t.Helper()
	var es []lexicon.Entry
	score := uni{}
	for _, w := range words {
		es = append(es, lexicon.Entry{Surface: w.w, Reading: zhuyin.ParseReading(w.r), PRW: 1})
		score[w.w] = w.p
	}
	c, err := lexicon.Build(es)
	if err != nil {
		t.Fatal(err)
	}
	return &Decoder{Lex: c, Score: score}
}

func partialLattice(typed ...string) input.Lattice {
	syls := make([]zhuyin.Syllable, len(typed))
	for i, y := range typed {
		syls[i] = zhuyin.Syllable(y)
		if r := []rune(y); zhuyin.ClassOf(r[len(r)-1]) != zhuyin.Tone {
			syls[i] = zhuyin.Abbreviate(zhuyin.Syllable(y))
		}
	}
	return input.Exact{}.Expand(syls)
}

const pen = input.CompletionPenalty

func TestPartialSyllablesDecode(t *testing.T) {
	base := 1e-4
	d := partialFixture(t, []partialWord{
		{"中國", "ㄓㄨㄥ ㄍㄨㄛˊ", base * math.Exp(2*pen+1)}, {"主顧", "ㄓㄨˇ ㄍㄨˋ", base}, {"這個", "ㄓㄜˋ ㄍㄜ˙", base * math.Exp(2*pen)},
		{"主", "ㄓㄨˇ", base}, {"中", "ㄓㄨㄥ", base * math.Exp(pen/2)}, {"知", "ㄓ", base / 10}, {"國", "ㄍㄨㄛˊ", base},
		{"你好", "ㄋㄧˇ ㄏㄠˇ", base}, {"你", "ㄋㄧˇ", base}, {"好", "ㄏㄠˇ", base},
	})
	for _, c := range []struct {
		typed []string
		want  string
	}{
		{[]string{"ㄓ", "ㄍ"}, "中國"},    // 兩個位置的補完罰分相同，詞頻決定
		{[]string{"ㄓㄨ", "ㄍㄨ"}, "中國"},  // 前後文夠強：中國 罰兩次仍勝過完整打出的 主顧
		{[]string{"ㄋ", "ㄏ"}, "你好"},    // 聲母都不是音節
		{[]string{"ㄓㄨㄥ", "ㄍ"}, "中國"},  // ㄓㄨㄥ 已經接不下去，照沒打聲調
		{[]string{"ㄓㄨˇ", "ㄍㄨ"}, "主顧"}, // 有打聲調是硬限制
	} {
		lat := partialLattice(c.typed...)
		for _, prefix := range []bool{false, true} {
			path := d.Decode(lat, nil, prefix)
			if got := text(path); got != c.want {
				t.Errorf("%v（prefix %v）：%q，要 %q", c.typed, prefix, got, c.want)
			}
			for _, g := range path {
				if g.Matched != "" {
					t.Errorf("%v：補完不是近音，%+v 不應有 Matched", c.typed, g)
				}
			}
		}
	}
}

// 補完的成本是 0（不是近音）：MaxCost 不裁它，近音視窗（OnlyCosted）不列它；罰分只影響分數。
func TestPartialCompletionsAreNotFuzzy(t *testing.T) {
	d := partialFixture(t, []partialWord{{"中", "ㄓㄨㄥ", 1e-3}, {"主", "ㄓㄨˇ", 1e-4}})
	lat := partialLattice("ㄓㄨ")
	d.MaxCost = 0.1
	if got := text(d.Decode(lat, nil, false)); got != "中" {
		t.Fatalf("MaxCost 不限制補完：%q", got)
	}
	d.MaxCost = 0
	cands := d.Candidates(lat, 0, 1, "", nil)
	if len(cands) != 2 || cands[0].Entry.Surface != "中" {
		t.Fatalf("候選列出補完與完整打出的字：%+v", cands)
	}
	if diff := cands[0].Score - cands[1].Score; math.Abs(diff-(math.Log(10)-pen)) > 1e-9 {
		t.Fatalf("候選分數差是對數機率差減罰分：%v", diff)
	}
	d.OnlyCosted = true
	if cands := d.Candidates(lat, 0, 1, "", nil); len(cands) != 0 {
		t.Fatalf("近音視窗不列補完：%+v", cands)
	}
}

// decode.Reading 跟解碼一樣算補完罰分：同一個字有好幾個讀音時取罰分後權重最高的。
func TestPartialReadingPrefersTypedSyllable(t *testing.T) {
	lex := lexicon.NewMem()
	lex.Add(lexicon.Entry{Surface: "著", Reading: zhuyin.ParseReading("ㄓㄨˋ"), PRW: 0.5})
	lex.Add(lexicon.Entry{Surface: "著", Reading: zhuyin.ParseReading("ㄓㄨㄛˊ"), PRW: 1})
	lex.Add(lexicon.Entry{Surface: "中國", Reading: zhuyin.ParseReading("ㄓㄨㄥ ㄍㄨㄛˊ"), PRW: 1})
	if r, ok := Reading(lex, partialLattice("ㄓㄨ"), 0, 1, "著"); !ok || zhuyin.Key(r) != "ㄓㄨˋ" {
		t.Fatalf("ㄓㄨ 的「著」是 ㄓㄨˋ：%v %v", r, ok)
	}
	if r, ok := Reading(lex, partialLattice("ㄓㄨㄛ"), 0, 1, "著"); !ok || zhuyin.Key(r) != "ㄓㄨㄛˊ" {
		t.Fatalf("ㄓㄨㄛ 的「著」是 ㄓㄨㄛˊ：%v %v", r, ok)
	}
	if r, ok := Reading(lex, partialLattice("ㄓ", "ㄍ"), 0, 2, "中國"); !ok || zhuyin.Key(r) != "ㄓㄨㄥ ㄍㄨㄛˊ" {
		t.Fatalf("ㄓㄍ 的「中國」：%v %v", r, ok)
	}
}

// 每組（起點、終點）只留前 maxUntonedArcs 個弧時，排名算進補完罰分：完整打出的字不會被補完的字擠掉。
func TestPartialArcCapKeepsTypedSyllable(t *testing.T) {
	var words []partialWord
	for k := 0; k < 2*maxUntonedArcs; k++ {
		words = append(words, partialWord{string(rune(0x4e00 + k)), "ㄓㄨㄥ", 1e-3})
	}
	words = append(words, partialWord{"主", "ㄓㄨˇ", 1e-3 * math.Exp(-pen/2)})
	d := partialFixture(t, words)
	arcs := d.wordArcs(partialLattice("ㄓㄨ"), 0, nil)
	if len(arcs) != maxUntonedArcs || !slices.ContainsFunc(arcs, func(a arc) bool { return a.e.Surface == "主" }) {
		t.Fatalf("留 %d 個弧、包含完整打出的「主」：%d 個", maxUntonedArcs, len(arcs))
	}
}

// spans 用詞庫的 Continuer 剪枝，結果（順序、成本、罰分）與逐一 HasPrefix 完全相同，
// 包含近音（Widen）與鎖定加在列尾的假設。
func TestSpansContinuerMatchesHasPrefix(t *testing.T) {
	d := partialFixture(t, []partialWord{
		{"中國", "ㄓㄨㄥ ㄍㄨㄛˊ", 1}, {"中國人", "ㄓㄨㄥ ㄍㄨㄛˊ ㄖㄣˊ", 1}, {"這個", "ㄓㄜˋ ㄍㄜ˙", 1}, {"資格", "ㄗ ㄍㄜˊ", 1},
		{"主顧", "ㄓㄨˇ ㄍㄨˋ", 1}, {"知", "ㄓ", 1}, {"國", "ㄍㄨㄛˊ", 1}, {"人", "ㄖㄣˊ", 1}, {"日", "ㄖˋ", 1},
	})
	lat := partialLattice("ㄓ", "ㄍ", "ㄖ")
	lat[0] = append(lat[0], input.Hyp{Syl: "ㄗ", Cost: 1.2}) // 近音
	lat[1] = append(lat[1], input.Hyp{Syl: "ㄍㄜˊ", Cost: 0}) // 鎖定加回的讀音
	walk := func(lex lexicon.Source) []string {
		var out []string
		dd := *d
		dd.Lex = lex
		for at := range lat {
			dd.spans(lat, at, func(r []zhuyin.Syllable, cost, pen float64) {
				out = append(out, fmt.Sprintf("%d %s %.1f %.1f", at, zhuyin.Key(r), cost, pen))
			})
		}
		return out
	}
	var brute []string
	for at := range lat {
		var rec func(pos int, r []zhuyin.Syllable, cost, pen float64)
		rec = func(pos int, r []zhuyin.Syllable, cost, pen float64) {
			if pos == len(lat) {
				return
			}
			for _, h := range lat[pos] {
				next := append(r[:len(r):len(r)], h.Syl)
				if d.Lex.HasPrefix(next) {
					brute = append(brute, fmt.Sprintf("%d %s %.1f %.1f", at, zhuyin.Key(next), cost+h.Cost, pen+h.Penalty))
					rec(pos+1, next, cost+h.Cost, pen+h.Penalty)
				}
			}
		}
		rec(at, nil, 0, 0)
	}
	if fast, slow := walk(d.Lex), walk(struct{ lexicon.Source }{d.Lex}); !slices.Equal(fast, brute) || !slices.Equal(slow, brute) {
		t.Fatalf("Continuer：%v\n沒有 Continuer：%v\n逐一 HasPrefix：%v", fast, slow, brute)
	}
	if len(brute) < 8 {
		t.Fatalf("測試要走到多個讀音：%v", brute)
	}
}

// 候選視窗的右側前後文經過注音打一半的位置時，詞圖每個終點也只留前幾名；候選照樣依前後文排序。
func TestPartialCandidatesWithRightContext(t *testing.T) {
	d := partialFixture(t, []partialWord{
		{"中國", "ㄓㄨㄥ ㄍㄨㄛˊ", 1e-3}, {"中", "ㄓㄨㄥ", 1e-4}, {"鐘", "ㄓㄨㄥ", 2e-4}, {"國", "ㄍㄨㄛˊ", 1e-4}, {"果", "ㄍㄨㄛˇ", 2e-4},
	})
	lat := input.Exact{}.Expand([]zhuyin.Syllable{"ㄓㄨㄥ", zhuyin.Abbreviate("ㄍ")})
	cands := d.CandidatesInContext(lat, 0, 1, nil, nil)
	if len(cands) == 0 || cands[0].Entry.Surface != "中" {
		t.Fatalf("右側是 ㄍ* 時「中」可以接成「中國」，排第一：%+v", cands)
	}
}

// 完整打出的音節優先：「中」的機率高 e^(pen/2) 倍，ㄓㄨ 仍是「主」。
func TestPartialPrefersTypedSyllable(t *testing.T) {
	d := partialFixture(t, []partialWord{{"主", "ㄓㄨˇ", 1e-4}, {"中", "ㄓㄨㄥ", 1e-4 * math.Exp(pen/2)}})
	for _, prefix := range []bool{false, true} {
		if got := text(d.Decode(partialLattice("ㄓㄨ"), nil, prefix)); got != "主" {
			t.Errorf("prefix %v：%q", prefix, got)
		}
	}
}
