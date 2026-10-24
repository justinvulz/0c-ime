package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

// newPartialSession 是「注音打一半也能選字」的測試詞庫：ㄓㄍ 可以是 中國、這個、主顧；ㄓㄨ 可以是 主 或（補完）中。
// 中國 比 主顧 常見得多（勝過兩個位置的補完罰分），主 與 中 一樣常見（完整打出的 主 優先）。
func newPartialSession(toneless, partial bool) *Session {
	lex, sys := lexicon.NewMem(), lm.NewMem()
	for _, e := range []struct {
		w, r string
		p    float64
	}{
		{"中國", "ㄓㄨㄥ ㄍㄨㄛˊ", 1e-3}, {"這個", "ㄓㄜˋ ㄍㄜ˙", 5e-4}, {"主顧", "ㄓㄨˇ ㄍㄨˋ", 1e-5},
		{"中", "ㄓㄨㄥ", 1e-3}, {"主", "ㄓㄨˇ", 1e-3}, {"這", "ㄓㄜˋ", 1e-3}, {"知", "ㄓ", 1e-4},
		{"國", "ㄍㄨㄛˊ", 1e-3}, {"個", "ㄍㄜ˙", 1e-3}, {"顧", "ㄍㄨˋ", 1e-4},
		{"你", "ㄋㄧˇ", 1e-2}, {"好", "ㄏㄠˇ", 5e-3}, {"你好", "ㄋㄧˇ ㄏㄠˇ", 2e-3}, {"西", "ㄒㄧ", 1e-3}, {"一", "ㄧ", 1e-3},
	} {
		lex.Add(lexicon.Entry{Surface: e.w, Reading: zhuyin.ParseReading(e.r), PRW: 1})
		sys.SetUni(e.w, e.p)
	}
	s := New(personal.NewModel(lex, sys))
	s.Toneless, s.PartialSyllables = toneless, partial
	return s
}

func TestPartialSyllablesCommit(t *testing.T) {
	for typed, want := range map[string]string{
		"ㄓㄍ":   "中國", // 每個字只打聲母
		"ㄓㄨㄍㄨ": "中國", // 聲母＋介音
		"ㄋㄏ":   "你好", // ㄋ 自己不是音節，打一半時也切音節（不再是 ㄏ 取代 ㄋ）
		"ㄓㄨ":   "主",  // 完整打出的音節優先
		"ㄓㄨˇㄍ": "主顧", // 有打聲調的音節照聲調
	} {
		s := newPartialSession(true, true)
		snap := typeBare(s, typed)
		if strings.ContainsAny(snap.Text(), "?*") {
			t.Errorf("%s：畫面不顯示記號：%q", typed, snap.Text())
		}
		if got := key(s, proto.KeyEnter).Commit; got != want {
			t.Errorf("%s：送出 %q，要 %q", typed, got, want)
		}
	}
}

// 只開「注音打一半」、沒開聲調選填，或不是標準排列時不生效：同類注音照舊取代。
func TestPartialSyllablesNeedToneless(t *testing.T) {
	if snap := typeBare(newPartialSession(false, true), "ㄋㄧㄏ"); snap.Text() != "ㄏㄧ" {
		t.Fatalf("沒開聲調選填：新聲母照舊取代舊聲母：%q", snap.Text())
	}
	if snap := typeBare(newPartialSession(true, false), "ㄋㄏㄠ"); snap.Text() != "ㄏㄠ" {
		t.Fatalf("只開聲調選填：不完整的音節照舊取代：%q", snap.Text())
	}
	for _, l := range []layout.Layout{layout.ETen, layout.Hsu} {
		s := newPartialSession(true, true)
		s.SetLayout(l)
		var snap proto.Snapshot
		for _, r := range "ㄓㄍ" {
			snap = key(s, string([]rune(l.Keys(zhuyin.Syllable(string(r))))[0]))
		}
		if strings.Contains(snap.Text(), "中") {
			t.Errorf("%s：注音打一半不生效：%q", l.Name(), snap.Text())
		}
	}
}

// 學習收詞庫讀音，個人資料沒有內部記號；在打一半的位置手選，下次照選過的出。
func TestPartialSyllablesLearnLexiconReading(t *testing.T) {
	s := newPartialSession(true, true)
	typeBare(s, "ㄓㄍ")
	if c := key(s, proto.KeyEnter).Commit; c != "中國" {
		t.Fatalf("沒選過時照詞頻：%q", c)
	}
	typeBare(s, "ㄓㄍ")
	key(s, proto.KeyHome)
	pickHere(t, s, "這個")
	if c := key(s, proto.KeyEnter).Commit; c != "這個" {
		t.Fatalf("送出 %q", c)
	}
	if !s.model.Store.Readings["這個\tㄓㄜˋ ㄍㄜ˙"] {
		t.Fatalf("手選的讀音是詞庫讀音：%v", s.model.Store.Readings)
	}
	b, err := json.Marshal(s.model.Store)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(string(b), "?*") {
		t.Fatalf("個人資料不能有內部記號：%s", b)
	}
	typeBare(s, "ㄓㄍ")
	if c := key(s, proto.KeyEnter).Commit; c != "這個" {
		t.Fatalf("選過之後 ㄓㄍ 出這個：%q", c)
	}
}

func TestPartialSyllablesSeparatorAndTone(t *testing.T) {
	s := newPartialSession(true, true)
	typeBare(s, "ㄒ")
	key(s, "'")
	typeBare(s, "ㄧ")
	s.finishUntoned()
	if len(s.syls) != 2 || !s.syls[0].Abbreviated() {
		t.Fatalf("' 把 ㄒ 切成打一半的音節：%q", s.syls)
	}
	// 不完整的開頭打聲調：照舊原樣留下。
	s = newPartialSession(true, true)
	typeBare(s, "ㄍˊ")
	if c := key(s, proto.KeyEnter).Commit; c != "ㄍˊ" {
		t.Fatalf("ㄍˊ 原樣送出：%q", c)
	}
}

// 開了注音打一半，打完整音節的情境照舊（聲調選填的例子）。
func TestPartialSyllablesKeepTonelessCommits(t *testing.T) {
	for typed, want := range map[string]string{"ㄋㄧㄏㄠ": "你好", "ㄒㄧㄢㄗㄞ": "現在", "ㄨㄛㄇㄣ": "我們", "ㄏㄨㄟㄧ": "會議"} {
		s := newTonelessSession(true)
		s.PartialSyllables = true
		typeBare(s, typed)
		if got := key(s, proto.KeyEnter).Commit; got != want {
			t.Errorf("%s：送出 %q，要 %q", typed, got, want)
		}
	}
}
