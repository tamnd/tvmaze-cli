package tvmaze

// Show is the public output record for a TV show.
type Show struct {
	Rank      int      `json:"rank"`
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"` // "Scripted", "Animation", etc.
	Genres    []string `json:"genres"`
	Status    string   `json:"status"`    // "Running", "Ended", "In Development"
	Premiered string   `json:"premiered"` // YYYY-MM-DD
	Rating    float64  `json:"rating"`
	Network   string   `json:"network"`
	Summary   string   `json:"summary"` // HTML stripped
	URL       string   `json:"url"`     // tvmaze.com URL
}

// unexported: only used for JSON decode

type searchResult struct {
	Score float64 `json:"score"`
	Show  rawShow `json:"show"`
}

type scheduleItem struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Season int     `json:"season"`
	Number int     `json:"number"`
	Show   rawShow `json:"show"`
}

type rawShow struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Genres    []string `json:"genres"`
	Status    string   `json:"status"`
	Premiered string   `json:"premiered"`
	Rating    struct {
		Average float64 `json:"average"`
	} `json:"rating"`
	Network struct {
		Name string `json:"name"`
	} `json:"network"`
	Summary string `json:"summary"`
	URL     string `json:"url"`
}
