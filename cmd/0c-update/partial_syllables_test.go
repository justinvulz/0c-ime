package main

import "testing"

// 「注音打一半也能選字」經 settings --json 給設定程式；引擎載入的值不同時標示待套用，舊引擎沒回報時不猜。
// 聲調選填關著時反灰並指向不打聲調。
func TestPartialSyllablesContractReportsPending(t *testing.T) {
	base := t.TempDir()
	writeSettings(t, base, `{"toneless":true,"partial_syllables":true}`)
	for _, tc := range []struct{ runtime, want string }{{"false", "off"}, {"true", "on"}, {"", ""}} {
		v, ok := runningValues(map[string]string{"partial_syllables": tc.runtime})["partial-syllables"]
		if v != tc.want || ok != (tc.runtime != "") {
			t.Fatalf("runtime %q: %q %v", tc.runtime, v, ok)
		}
		doc := buildSettingsDoc(base, modelStatusDial(map[string]string{"partial_syllables": tc.runtime}, true))
		it := item(t, doc, "partial-syllables")
		if it.Value != "on" || it.Disabled || it.Pending != (tc.runtime == "false") || (it.Running != nil) != ok {
			t.Fatalf("pending contract: %+v", it)
		}
	}
	writeSettings(t, base, `{"partial_syllables":true}`)
	doc := buildSettingsDoc(base, modelStatusDial(map[string]string{}, true))
	if it := item(t, doc, "partial-syllables"); it.Value != "on" || !it.Disabled || it.DisabledBy != "toneless" {
		t.Fatalf("聲調選填關著時反灰：%+v", it)
	}
	if o, err := parseSetArgs([]string{"partial-syllables", "on", "-apply", "-json"}); err != nil || len(o.pairs) != 1 || !o.apply {
		t.Fatalf("%+v %v", o, err)
	}
}
