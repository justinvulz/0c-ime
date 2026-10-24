package zhuyin

import (
	"slices"
	"strings"
)

// UntonedMark 是「沒打聲調」的內部記號（#150，聲調選填）：接在音節的注音之後，表示這個音節的聲調由解碼決定。
// 它只出現在 session 的音節序列與音節格展開之前；不進詞庫、學習或任何儲存。
// Valid 與 Legal 都拒絕帶記號的音節，所以任何沒處理它的查詢都只會「查不到」，不會被當成一聲。
const UntonedMark = '?'

// PartialMark 是「注音打一半」的內部記號（設定「注音打一半也能選字」）：接在注音之後，表示這個音節是
// 任何以這些注音開頭的音節、任何聲調（ㄓ* 可以是 ㄓㄨㄥ、ㄓㄜˋ…）。帶它的音節也算沒打聲調（Untoned 為真），
// 所以跟 UntonedMark 一樣只出現在 session 的音節序列，不進詞庫、學習或任何儲存。
const PartialMark = '*'

// Untone 回傳 base（去掉聲調的注音）的「沒打聲調」音節。
func Untone(base Syllable) Syllable { return Syllable(string(base.Base()) + string(UntonedMark)) }

// Abbreviate 回傳 base 的「打一半」音節：base 還能接成更長的合法音節時帶 PartialMark（ㄓ*、ㄓㄨ*）；
// 已經接不下去（ㄓㄨㄥ）或不是任何音節的開頭時同 Untone，展開與個人資料查詢都和沒打聲調一樣。
func Abbreviate(base Syllable) Syllable {
	b := base.Base()
	if n := len(completionBases[string(b)]); n == 0 || n == 1 && legalBases[string(b)] {
		return Untone(b)
	}
	return Syllable(string(b) + string(PartialMark))
}

// Untoned 回報音節是否為沒打聲調的音節（注音結構合法、結尾是 UntonedMark 或 PartialMark）。
func (s Syllable) Untoned() bool {
	t := string(s)
	if !strings.HasSuffix(t, string(UntonedMark)) && !strings.HasSuffix(t, string(PartialMark)) {
		return false
	}
	base := Syllable(t[:len(t)-1]) // 兩個記號都是一個位元組
	return base != "" && base.Valid() && base.ToneOf() == 1 && !strings.ContainsRune(string(base), Tone1)
}

// Abbreviated 回報音節是否為「打一半」的音節（結尾是 PartialMark）。
func (s Syllable) Abbreviated() bool {
	return strings.HasSuffix(string(s), string(PartialMark)) && s.Untoned()
}

// Tones 回傳沒打聲調的音節在詞庫裡可能的五個聲調（一聲、二聲、三聲、四聲、輕聲，依序）；
// 其他音節原樣回傳只含自己的切片。打一半的音節回傳打出的注音本身的五個聲調（完整的展開見 Completions）。
func (s Syllable) Tones() []Syllable {
	if !s.Untoned() {
		return []Syllable{s}
	}
	b := s.Base()
	return []Syllable{b, b.WithTone(2), b.WithTone(3), b.WithTone(4), b.WithTone(5)}
}

// Text 是音節在畫面上的樣子：沒打聲調的音節不顯示記號。
func (s Syllable) Text() string {
	if s.Untoned() {
		return string(s.Base())
	}
	return string(s)
}

// HasUntoned 回報 syls 是否含沒打聲調的音節（含打一半的音節）。
func HasUntoned(syls []Syllable) bool {
	for _, y := range syls {
		if y.Untoned() {
			return true
		}
	}
	return false
}

// completionBases 把音節表內所有音節依「聲母 介音 韻母」順序的前綴（含完整音節，不帶聲調）
// 對到以它開頭的合法音節，依字串排序：ㄓㄨ → ㄓㄨ、ㄓㄨㄚ、ㄓㄨㄛ…。
// completions 是同一組前綴對到那些音節的五個聲調，依字串排序（前綴本身是合法音節時，它的一聲排第一）。
var completionBases, completions = func() (map[string][]Syllable, map[string][]Syllable) {
	bases := map[string][]Syllable{}
	for b := range legalBases {
		r := []rune(b)
		for i := 1; i <= len(r); i++ {
			bases[string(r[:i])] = append(bases[string(r[:i])], Syllable(b))
		}
	}
	toned := make(map[string][]Syllable, len(bases))
	for p, bs := range bases {
		slices.Sort(bs)
		out := make([]Syllable, 0, 5*len(bs))
		for _, b := range bs {
			out = append(out, b, b.WithTone(2), b.WithTone(3), b.WithTone(4), b.WithTone(5))
		}
		slices.Sort(out)
		toned[p] = out
	}
	return bases, toned
}()

// LegalPrefix 回報不帶聲調的 s 是否為音節表內某個音節的開頭（含完整音節本身），
// 例如 ㄒㄧ、ㄒㄧㄢ 是，ㄉㄧㄤ、ㄅㄩ 不是。不打聲調時用它判斷下一個注音要接在目前的音節後面還是開新的音節。
func LegalPrefix(s Syllable) bool { return len(completionBases[string(s)]) > 0 }

// CompletionBases 回傳以 prefix（不帶聲調）開頭的合法音節，依字串排序；不是任何音節的開頭時為 nil。
// 回傳的切片是共用的，呼叫端不得修改。
func CompletionBases(prefix Syllable) []Syllable { return completionBases[string(prefix)] }

// Completions 回傳打一半的音節可能代表的所有帶聲調音節（以它的注音開頭的合法音節 × 五個聲調），
// 依字串排序；其他音節同 Tones。打一半的音節回傳的切片是共用的，呼叫端不得修改。
func (s Syllable) Completions() []Syllable {
	if !s.Abbreviated() {
		return s.Tones()
	}
	return completions[string(s.Base())]
}

// Covers 回報使用者打的音節 s 是否涵蓋詞庫讀音 y：打一半的音節涵蓋以它開頭的音節的任何聲調，
// 沒打聲調的音節涵蓋同一個音節的任何聲調，其他音節只涵蓋自己。
func (s Syllable) Covers(y Syllable) bool {
	switch {
	case s.Abbreviated():
		return strings.HasPrefix(string(y.Base()), string(s.Base()))
	case s.Untoned():
		return y.Base() == s.Base()
	}
	return y == s
}
