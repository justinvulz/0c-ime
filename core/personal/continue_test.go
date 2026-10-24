package personal

import (
	"slices"
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/vocab"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

// 個人化視圖的 Continuations 是系統詞庫、自訂詞與個人片語的聯集，與逐一 HasPrefix 相同。
func TestViewContinuationsMatchHasPrefix(t *testing.T) {
	lex := lexicon.NewMem()
	for _, w := range []struct{ s, r string }{{"中國", "ㄓㄨㄥ ㄍㄨㄛˊ"}, {"主", "ㄓㄨˇ"}, {"這個", "ㄓㄜˋ ㄍㄜ˙"}} {
		lex.Add(lexicon.Entry{Surface: w.s, Reading: zhuyin.ParseReading(w.r), PRW: 1})
	}
	m := NewModel(lex, lm.NewMem())
	if _, err := m.VocabAdd([]vocab.Entry{{Text: "中壢", Readings: []string{"ㄓㄨㄥ ㄌㄧˋ"}}, {Text: "周杰倫", Readings: []string{"ㄓㄡ ㄐㄧㄝˊ ㄌㄨㄣˊ"}}}, vocab.SourceManual); err != nil {
		t.Fatal(err)
	}
	m.Store.Phrases["中鋼"] = "ㄓㄨㄥ ㄍㄤ"
	m.Store.PhraseOrder = append(m.Store.PhraseOrder, "中鋼")
	src := m.Source()
	all := zhuyin.ParseReading("ㄓㄨㄥ ㄍㄨㄛˊ ㄓㄨˇ ㄓㄜˋ ㄍㄜ˙ ㄌㄧˋ ㄓㄡ ㄐㄧㄝˊ ㄌㄨㄣˊ ㄍㄤ")
	for _, p := range [][]zhuyin.Syllable{nil, {"ㄓㄨㄥ"}, {"ㄓㄡ"}, {"ㄓㄡ", "ㄐㄧㄝˊ"}, {"ㄍㄤ"}} {
		for _, b := range []zhuyin.Syllable{"ㄓ", "ㄍ", "ㄌ", "ㄐ", "ㄓㄨ"} {
			var want []zhuyin.Syllable
			for _, y := range all {
				if strings.HasPrefix(string(y), string(b)) && src.HasPrefix(append(slices.Clone(p), y)) {
					want = append(want, y)
				}
			}
			slices.Sort(want)
			want = slices.Compact(want)
			var got []zhuyin.Syllable
			lexicon.Continue(src, p, b, func(y zhuyin.Syllable) { got = append(got, y) })
			slices.Sort(got)
			got = slices.Compact(got)
			if !slices.Equal(got, want) {
				t.Errorf("%v 之後以 %q 開頭：%v，要 %v", p, b, got, want)
			}
		}
	}
}
