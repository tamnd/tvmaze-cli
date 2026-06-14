package tvmaze

// Show is the public output record for a TV show.
type Show struct {
	ID        int      `kit:"id" json:"id"`
	Name      string   `json:"name"`
	Genres    []string `json:"genres"`
	Status    string   `json:"status"`
	Rating    float64  `json:"rating"`
	Network   string   `json:"network"`
	Premiered string   `json:"premiered"`
	Language  string   `json:"language"`
	Type      string   `json:"type"`
	Summary   string   `json:"summary"`
}

// Episode is one episode record from the /shows/{id}/episodes endpoint.
type Episode struct {
	ID      int     `kit:"id" json:"id"`
	Name    string  `json:"name"`
	Season  int     `json:"season"`
	Number  int     `json:"number"`
	Airdate string  `json:"airdate,omitempty"`
	Runtime int     `json:"runtime,omitempty"`
	Rating  float64 `json:"rating"`
	Summary string  `json:"summary,omitempty"`
}

// CastMember is one cast entry from the /shows/{id}/cast endpoint.
type CastMember struct {
	PersonID   int    `kit:"id" json:"person_id"`
	PersonName string `json:"person_name"`
	Birthday   string `json:"birthday,omitempty"`
	Country    string `json:"country,omitempty"`
	Character  string `json:"character"`
}

// ScheduleItem is one entry from the /schedule endpoint.
type ScheduleItem struct {
	ID       int    `kit:"id" json:"id"`
	Name     string `json:"name"`
	Season   int    `json:"season"`
	Number   int    `json:"number"`
	Airdate  string `json:"airdate"`
	Airtime  string `json:"airtime"`
	ShowID   int    `json:"show_id"`
	ShowName string `json:"show_name"`
	Network  string `json:"network"`
}

// unexported: only used for JSON decode

type searchResult struct {
	Score float64 `json:"score"`
	Show  rawShow `json:"show"`
}

type rawScheduleItem struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Season  int     `json:"season"`
	Number  int     `json:"number"`
	Airdate string  `json:"airdate"`
	Airtime string  `json:"airtime"`
	Show    rawShow `json:"show"`
}

type rawShow struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Language string   `json:"language"`
	Genres   []string `json:"genres"`
	Status   string   `json:"status"`
	Rating   struct {
		Average *float64 `json:"average"`
	} `json:"rating"`
	Network struct {
		Name string `json:"name"`
	} `json:"network"`
	Premiered string `json:"premiered"`
	Summary   string `json:"summary"`
}

type rawEpisode struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Season  int    `json:"season"`
	Number  int    `json:"number"`
	Airdate string `json:"airdate"`
	Summary string `json:"summary"`
	Runtime int    `json:"runtime"`
	Rating  struct {
		Average *float64 `json:"average"`
	} `json:"rating"`
}

type rawCast struct {
	Person struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Birthday string `json:"birthday"`
		Country struct {
			Name string `json:"name"`
		} `json:"country"`
	} `json:"person"`
	Character struct {
		Name string `json:"name"`
	} `json:"character"`
}
