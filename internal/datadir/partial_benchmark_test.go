package datadir

import (
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/0c-biggo/0c-ime/core/engine"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
	"github.com/0c-biggo/0c-ime/internal/settings"
)

// partialBenchmarkData 是真的資料包目錄（ZC_BENCH_DATA，例如 make assets 的 assets/data）；沒有設定時跳過。
// 「注音打一半」每個位置有上百個假設，只有真的詞庫量得出每鍵延遲。
func partialBenchmarkData(tb testing.TB) *engine.Engine {
	tb.Helper()
	dir := os.Getenv("ZC_BENCH_DATA")
	if dir == "" {
		tb.Skip("ZC_BENCH_DATA 沒有設定")
	}
	e, err := Open(dir)
	if err != nil {
		tb.Fatal(err)
	}
	return e
}

// zhuyinKeys 把注音（標準排列）換成按鍵；空白照打。
func zhuyinKeys(text string) []proto.Request {
	var out []proto.Request
	for _, r := range text {
		k := " "
		if r != ' ' {
			k = string([]rune(layout.Standard.Keys(zhuyin.Syllable(string(r))))[0])
		}
		out = append(out, proto.Request{Type: proto.Key, Key: k})
	}
	return out
}

func keys(ks ...string) []proto.Request {
	var out []proto.Request
	for _, k := range ks {
		out = append(out, proto.Request{Type: proto.Key, Key: k})
	}
	return out
}

var partialJourneys = map[string][][]proto.Request{
	// 不打聲調、每個字打完整：注音打一半開著時仍會展開成以它開頭的音節（ㄐㄧ → ㄐㄧㄣ…）。
	"full": {zhuyinKeys("ㄐㄧㄣㄊㄧㄢㄊㄧㄢㄑㄧㄏㄣㄏㄠ"), zhuyinKeys("ㄨㄛㄇㄣㄑㄩㄍㄨㄤㄐㄧㄝ")},
	// 只打聲母：今天天氣很好、中華民國、我們去逛街。
	"initials": {zhuyinKeys("ㄐㄊㄊㄑㄏㄏ"), zhuyinKeys("ㄓㄏㄇㄍ"), zhuyinKeys("ㄨㄇㄑㄍㄐ")},
	// 聲母＋介音或韻母。
	"medials": {zhuyinKeys("ㄓㄨㄏㄨㄇㄧㄍㄨ"), zhuyinKeys("ㄐㄧㄊㄧㄊㄧㄑㄧㄏㄏ")},
	// 十個字只打聲母。
	"initials10": {zhuyinKeys("ㄐㄊㄊㄑㄏㄏㄨㄇㄑㄍ")},
	// 句中按 ↓ 開候選（右側前後文經過打一半的位置）再關掉。
	"down": {append(zhuyinKeys("ㄓㄏㄇㄍㄉㄒ"), keys(proto.KeyHome, proto.KeyRight, proto.KeyDown, proto.KeyEscape)...)},
}

func BenchmarkPartialSyllablesKeys(b *testing.B) {
	names := make([]string, 0, len(partialJourneys))
	for name := range partialJourneys {
		names = append(names, name)
	}
	sort.Strings(names)
	e := partialBenchmarkData(b)
	for _, partial := range []bool{false, true} {
		for _, name := range names {
			mode := "toneless"
			if partial {
				mode = "partial"
			}
			b.Run(mode+"/"+name, func(b *testing.B) {
				e.Toneless, e.PartialSyllables = true, partial
				e.Fuzzy = settings.Default().FuzzyConfig() // 出貨預設：四組模糊音都是「僅供選字」
				id := e.OpenSession()
				var events []proto.Request
				for _, j := range partialJourneys[name] {
					events = append(events, j...)
					events = append(events, proto.Request{Type: proto.Cancel})
				}
				for _, ev := range events { // 先走一次，量的是穩定狀態（快取已建立）
					if _, err := e.Do(id, ev); err != nil {
						b.Fatal(err)
					}
				}
				samples := make([]int64, 0, min(b.N, 100000))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					start := time.Now()
					var err error
					benchmarkSnapshot, err = e.Do(id, events[i%len(events)])
					if err != nil {
						b.Fatal(err)
					}
					if len(samples) < cap(samples) {
						samples = append(samples, time.Since(start).Nanoseconds())
					}
				}
				b.StopTimer()
				sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
				if len(samples) > 0 {
					b.ReportMetric(float64(samples[(len(samples)-1)*95/100]), "p95-ns")
					b.ReportMetric(float64(samples[(len(samples)-1)*99/100]), "p99-ns")
					b.ReportMetric(float64(samples[len(samples)-1]), "max-ns")
				}
			})
		}
	}
}

// 真的詞庫上的幾個例子（只在有 ZC_BENCH_DATA 時跑）：印出結果供人看，只檢查不出內部記號、確實組成中文。
func TestPartialSyllablesRealPack(t *testing.T) {
	e := partialBenchmarkData(t)
	e.Toneless, e.PartialSyllables = true, true
	for _, text := range []string{"ㄓㄍ", "ㄓㄨㄍㄨ", "ㄋㄏ", "ㄓㄨ", "ㄐㄊㄊㄑㄏㄏ", "ㄓㄏㄇㄍ", "ㄨㄇㄑㄍㄐ", "ㄐㄧㄊㄧㄊㄧㄑㄧㄏㄏ", "ㄊㄞㄨㄢ", "ㄉㄚㄒㄩㄝ", "ㄒㄧㄢㄗㄞ"} {
		id := e.OpenSession()
		for _, ev := range zhuyinKeys(text) {
			if _, err := e.Do(id, ev); err != nil {
				t.Fatal(err)
			}
		}
		snap, err := e.Do(id, proto.Request{Type: proto.Key, Key: proto.KeyEnter})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s → %s", text, snap.Commit)
		if snap.Commit == "" || strings.ContainsAny(snap.Commit, "?*ㄅㄆㄇㄈㄉㄊㄋㄌㄍㄎㄏㄐㄑㄒㄓㄔㄕㄖㄗㄘㄙ") {
			t.Errorf("%s：%q", text, snap.Commit)
		}
	}
}
