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

func TestClassifyNumeric(t *testing.T) {
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

func TestClassifyQuery(t *testing.T) {
	typ, id, err := Domain{}.Classify("breaking bad")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if typ != "query" {
		t.Errorf("type = %q, want query", typ)
	}
	if id != "breaking bad" {
		t.Errorf("id = %q, want breaking bad", id)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify(\"\") should return error")
	}
}

func TestLocateShow(t *testing.T) {
	got, err := Domain{}.Locate("show", "169")
	want := "https://www.tvmaze.com/shows/169"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateQuery(t *testing.T) {
	got, err := Domain{}.Locate("query", "breaking+bad")
	want := "https://www.tvmaze.com/search?q=breaking+bad"
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

func TestStripHTML(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"<p>Hello world</p>", "Hello world"},
		{"plain text", "plain text"},
		{"<b>bold</b> and <i>italic</i>", "bold and italic"},
		{"  <p>  spaces  </p>  ", "spaces"},
		{"", ""},
	}
	for _, tc := range cases {
		got := stripHTML(tc.in)
		if got != tc.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeShow(t *testing.T) {
	avg := 9.2
	r := rawShow{
		ID:     169,
		Name:   "Breaking Bad",
		Status: "Ended",
		Rating: struct {
			Average *float64 `json:"average"`
		}{Average: &avg},
		Network: struct {
			Name string `json:"name"`
		}{Name: "AMC"},
		Summary: "<p>A teacher.</p>",
	}
	s := normalizeShow(r)
	if s.ID != 169 {
		t.Errorf("ID = %d, want 169", s.ID)
	}
	if s.Rating != 9.2 {
		t.Errorf("Rating = %v, want 9.2", s.Rating)
	}
	if s.Network != "AMC" {
		t.Errorf("Network = %q, want AMC", s.Network)
	}
	if s.Summary != "A teacher." {
		t.Errorf("Summary = %q, want %q", s.Summary, "A teacher.")
	}
}

func TestNormalizeShowNilRating(t *testing.T) {
	r := rawShow{ID: 1, Name: "Test"}
	s := normalizeShow(r)
	if s.Rating != 0 {
		t.Errorf("Rating = %v, want 0 for nil average", s.Rating)
	}
}
