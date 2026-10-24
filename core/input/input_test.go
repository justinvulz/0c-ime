package input

import (
	"testing"

	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func TestExpandAbbreviated(t *testing.T) {
	lat := Exact{}.Expand([]zhuyin.Syllable{zhuyin.Abbreviate("ㄓㄨ"), zhuyin.Untone("ㄋㄧ"), "ㄏㄠˇ"})
	row := lat[0]
	if len(row) != len(zhuyin.Syllable("ㄓㄨ*").Completions()) {
		t.Fatalf("ㄓㄨ* 展開成 %d 個假設", len(row))
	}
	for k, h := range row {
		if h.Cost != 0 || !h.Untoned || h.Prefix != "ㄓㄨ" {
			t.Fatalf("%+v：成本 0、Untoned、Prefix ㄓㄨ", h)
		}
		if k > 0 && row[k-1].Syl >= h.Syl {
			t.Fatalf("同一列依 Syl 排序：%q 在 %q 之後", h.Syl, row[k-1].Syl)
		}
		if exact := h.Syl.Base() == "ㄓㄨ"; exact != (h.Penalty == 0) || !exact && h.Penalty != CompletionPenalty {
			t.Fatalf("%+v：打出的音節沒有罰分，更長的音節罰 CompletionPenalty", h)
		}
	}
	if len(lat[1]) != 5 || lat[1][0].Prefix != "" || lat[1][0].Penalty != 0 || !lat[1][0].Untoned {
		t.Fatalf("沒打聲調照舊是五個聲調：%+v", lat[1])
	}
	if len(lat[2]) != 1 || lat[2][0] != (Hyp{Syl: "ㄏㄠˇ"}) {
		t.Fatalf("打了聲調只有自己：%+v", lat[2])
	}
}
