// Package input 定義輸入層交給解碼器的契約：帶成本的音節格。
//
// 鍵盤精確輸入時每個位置只有一個音節、成本 0；容錯通道、部分注音與未來的語音前端
// 會在同一位置放入多個帶成本的假設。解碼器對所有前端一視同仁。
package input

import "github.com/0c-biggo/0c-ime/core/zhuyin"

// CompletionPenalty 是打一半的音節展開成「比打出的注音更長」的音節時，每個音節的計分罰分（負對數）：
// 打出完整音節的讀音優先（ㄓㄨ 先是「主」），前後文夠強時才換成補完的讀音（ㄓㄨ ㄍㄨ 是「中國」）。
// 1.5 是在 0.2.5 資料包、400 句文件句子上掃過 0–4 的折衷：開著它照常打完整音節時字正確率 82.7%
// （沒開 84.5%、罰分 0 時 79.2%），每字打兩個注音 67.0%（最佳）、只打聲母 21.0%（最佳 21.8%）。
const CompletionPenalty = 1.5

// Hyp 是某個位置的一個音節假設。Cost 為負對數機率，0 表示確定。
type Hyp struct {
	Syl  zhuyin.Syllable
	Cost float64
	// Untoned：這個假設是沒打聲調的音節（#150）展開出來的其中一個聲調。解碼器據此裁切這些位置的弧數；
	// 全部打了聲調時沒有這種假設。
	Untoned bool
	// Penalty 是只計入分數的罰分（負對數），不是 Cost：不算近音、不受 MaxCost 限制、學習時照常換回讀音。
	// 打一半的音節展開成更長的音節時是 CompletionPenalty，其他假設為 0。
	Penalty float64
	// Prefix 是打一半的音節（zhuyin.PartialMark）展開出來的假設所屬的、使用者打出的注音（不帶聲調）；
	// 其他假設為空。同一列的這些假設排在列首、依 Syl 排序，解碼器據此改用詞庫的 Continuer 剪枝。
	Prefix zhuyin.Syllable
}

// Lattice 的每個元素是一個位置的所有假設。
type Lattice [][]Hyp

// Frontend 把使用者輸入的音節序列展開成音節格。
type Frontend interface {
	Expand(syls []zhuyin.Syllable) Lattice
}

// Exact 是精確通道：使用者打什麼就是什麼。沒打聲調的音節（#150，只在使用者開啟「聲調選填」時出現）
// 展開成五個聲調，成本都是 0，由語言模型依前後文決定；打一半的音節（只在開啟「注音打一半也能選字」時出現）
// 展開成以它開頭的所有音節的五個聲調，成本也是 0，比打出的注音更長的帶 CompletionPenalty；其他音節只有自己。
type Exact struct{}

func (Exact) Expand(syls []zhuyin.Syllable) Lattice {
	lat := make(Lattice, len(syls))
	for i, s := range syls {
		if s.Abbreviated() {
			if ys := s.Completions(); len(ys) > 0 {
				base := s.Base()
				row := make([]Hyp, len(ys))
				for k, y := range ys {
					row[k] = Hyp{Syl: y, Untoned: true, Prefix: base}
					if y.Base() != base {
						row[k].Penalty = CompletionPenalty
					}
				}
				lat[i] = row
				continue
			}
		}
		if s.Untoned() {
			for _, y := range s.Tones() {
				lat[i] = append(lat[i], Hyp{Syl: y, Untoned: true})
			}
			continue
		}
		lat[i] = []Hyp{{Syl: s}}
	}
	return lat
}
