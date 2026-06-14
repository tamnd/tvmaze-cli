package tvmaze

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "tvmaze" {
		t.Errorf("Scheme = %q, want tvmaze", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "tvmaze" {
		t.Errorf("Identity.Binary = %q, want tvmaze", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("169")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if typ != "show" {
		t.Errorf("type = %q, want show", typ)
	}
	if id != "169" {
		t.Errorf("id = %q, want 169", id)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify(\"\") should return error")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("show", "169")
	want := "https://www.tvmaze.com/shows/169"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("episode", "42")
	if err == nil {
		t.Error("Locate with unknown type should return error")
	}
}
