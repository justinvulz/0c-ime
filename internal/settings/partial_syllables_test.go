package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 「注音打一半也能選字」預設關閉，舊設定檔沒有欄位時也是關；可以開、改回並存檔。
// 聲調選填關著時反灰、指向「不打聲調也能選字」；倚天、許氏時以排列為準。值一律保留。
func TestPartialSyllablesDefaultsOffAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if s, err := Load(path); err != nil || s.PartialSyllables {
		t.Fatalf("預設關閉：%+v %v", s, err)
	}
	if v, _ := Default().Value("partial-syllables"); v != "off" {
		t.Fatalf("命令列預設值：%q", v)
	}
	for _, value := range []string{"on", "off", "on"} {
		s, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Set("partial-syllables", value); err != nil {
			t.Fatal(err)
		}
		if err := Save(path, s); err != nil {
			t.Fatal(err)
		}
		back, err := Load(path)
		if err != nil || back.PartialSyllables != (value == "on") {
			t.Fatalf("重新讀取：%+v %v", back, err)
		}
		b, _ := os.ReadFile(path)
		if !strings.Contains(string(b), `"partial_syllables": `+map[string]string{"on": "true", "off": "false"}[value]) {
			t.Fatalf("settings.json：%s", b)
		}
	}
	if d := Default(); d.Set("partial-syllables", "maybe") == nil {
		t.Fatal("不認得的值應拒絕")
	}
}

func TestPartialSyllablesNeedsToneless(t *testing.T) {
	item := func(s Settings) Item {
		for _, it := range s.Items() {
			if it.Key == "partial-syllables" {
				return it
			}
		}
		t.Fatal("設定程式看不到注音打一半的開關")
		return Item{}
	}
	s := Default()
	s.PartialSyllables = true
	if it := item(s); !it.Disabled || it.DisabledReason != partialNeedsTonelessReason || it.DisabledBy != "toneless" || it.Value != "on" {
		t.Fatalf("聲調選填關著：反灰並指向不打聲調，值保留：%+v", it)
	}
	s.Toneless = true
	if it := item(s); it.Disabled || !strings.Contains(it.Description, "ㄓㄍ") {
		t.Fatalf("聲調選填開著：可以設定：%+v", it)
	}
	for layout, reason := range map[string]string{LayoutETen: "倚天 41 鍵時關閉", LayoutHsu: "許氏排列時關閉"} {
		s := s
		if err := s.Set("keyboard-layout", layout); err != nil {
			t.Fatal(err)
		}
		if it := item(s); it.DisabledReason != reason || it.DisabledBy != "keyboard-layout" {
			t.Fatalf("%s：以排列為準：%+v", layout, it)
		}
	}
}
