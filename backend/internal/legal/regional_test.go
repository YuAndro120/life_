package legal

import (
	"reflect"
	"testing"
)

func TestRegionOfUsesPublicationNumberPrefix(t *testing.T) {
	cases := map[string]string{"1600202607290006": "16", "7700202609250001": "77", "8000202609250001": "", "": ""}
	for eo, want := range cases {
		if got := RegionOf(eo); got != want {
			t.Errorf("%q: %q, ждали %q", eo, got, want)
		}
	}
}

func TestClassifyRegional(t *testing.T) {
	cases := []struct {
		title string
		tags  []string
		ok    bool
	}{
		{`О внесении изменений в Закон Республики Татарстан "О транспортном налоге"`, []string{"transport:driver"}, true},
		{`О налоге на имущество физических лиц в Свердловской области`, []string{"housing:owner"}, true},
		{`О налоге на профессиональный доход в Краснодарском крае`, []string{"work:selfemployed"}, true},
		{`Об областном бюджете на 2027 год`, nil, false},
		{`О внесении изменений в Закон "О наделении органов местного самоуправления полномочиями по образованию"`, nil, false},
		{`О государственных наградах области`, nil, false},
		{`О порядке организации такси в Москве`, []string{"transport:driver", "sells:transport"}, true},
	}
	for _, c := range cases {
		tags, who, ok := ClassifyRegional(Doc{Title: c.title})
		if ok != c.ok || !reflect.DeepEqual(tags, c.tags) {
			t.Errorf("%q: %v %v, ждали %v %v", c.title, tags, ok, c.tags, c.ok)
		}
		if ok && who == "" {
			t.Errorf("%q: нет подписи «Касается»", c.title)
		}
	}
}
