package lexicon

import (
	"sort"
	"strings"

	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

// Continuer 是 Source 的選用擴充，給「注音打一半」的展開剪枝用：對每個字串以 base 開頭、
// 而且 HasPrefix(syls 加上 y) 為真的音節 y 呼叫 fn（順序不拘、可能重複）。結果必須與逐一 HasPrefix 相同；
// 實作可以只花「真的接得下去的音節數」的成本，不必逐一試以 base 開頭的上百個帶聲調音節。
type Continuer interface {
	Continuations(syls []zhuyin.Syllable, base zhuyin.Syllable, fn func(zhuyin.Syllable))
}

// Continue 對 src 列舉 syls 之後、以 base 開頭的音節（見 Continuer）。src 沒有實作 Continuer 時，
// 逐一試 base 的所有展開（zhuyin.Completions），結果相同、只是比較慢。
func Continue(src Source, syls []zhuyin.Syllable, base zhuyin.Syllable, fn func(zhuyin.Syllable)) {
	if c, ok := src.(Continuer); ok {
		c.Continuations(syls, base, fn)
		return
	}
	probe := append(syls[:len(syls):len(syls)], "")
	for _, y := range zhuyin.Syllable(string(base) + string(zhuyin.PartialMark)).Completions() {
		probe[len(syls)] = y
		if src.HasPrefix(probe) {
			fn(y)
		}
	}
}

// Continuations：詞條依音節 ID 序列排序、音節 ID 依字串排序，所以字串以 base 開頭的音節是一段連續的 ID [lo, hi)，
// 讀音以 syls 開頭的詞條也是連續的一段；在這一段裡從 lo 起找下一個出現的 ID x、回報，再跳到 x+1 二分搜尋。
// 二分搜尋的次數是接得下去的音節數加一。
func (c *Compact) Continuations(syls []zhuyin.Syllable, base zhuyin.Syllable, fn func(zhuyin.Syllable)) {
	q, ok := c.lookupIDs(syls)
	if !ok {
		return
	}
	if !c.sortedSyls {
		for id, y := range c.syls { // 理論上不會發生（Build 一定排序）；照樣給出正確結果
			if strings.HasPrefix(string(y), string(base)) && c.HasPrefix(append(syls[:len(syls):len(syls)], c.syls[id])) {
				fn(y)
			}
		}
		return
	}
	lo := sort.Search(len(c.syls), func(i int) bool { return c.syls[i] >= base })
	hi := lo + sort.Search(len(c.syls)-lo, func(k int) bool { return !strings.HasPrefix(string(c.syls[lo+k]), string(base)) })
	n := len(q)
	key := make([]uint16, n+1)
	copy(key, q)
	for x := lo; x < hi; {
		key[n] = uint16(x)
		i := c.lower(key)
		if i == len(c.ent) {
			return
		}
		r := c.reading(&c.ent[i])
		if len(r) <= n || cmpIDs(r[:n], q) != 0 || int(r[n]) >= hi {
			return
		}
		fn(c.syls[r[n]])
		x = int(r[n]) + 1
	}
}

// Continuations 見 Continuer。
func (m *Mem) Continuations(syls []zhuyin.Syllable, base zhuyin.Syllable, fn func(zhuyin.Syllable)) {
	n := m.find(syls)
	if n == nil {
		return
	}
	for _, y := range n.order {
		if strings.HasPrefix(string(y), string(base)) {
			fn(y)
		}
	}
}

// Continuations 見 Continuer：底層詞庫的音節，加上句首單一輕聲音節補的語氣詞（ㄛ˙ 喔）。
func (n neutralParticles) Continuations(syls []zhuyin.Syllable, base zhuyin.Syllable, fn func(zhuyin.Syllable)) {
	Continue(n.Source, syls, base, fn)
	if len(syls) > 0 {
		return
	}
	for _, b := range zhuyin.CompletionBases(base) {
		if y := b.WithTone(5); len(n.fallback([]zhuyin.Syllable{y})) > 0 {
			fn(y)
		}
	}
}
