package tvmaze

// Show is the public output record for a TV show.
type Show struct {
	ID        int    `kit:"id" json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Premiered string `json:"premiered"`
	Ended     string `json:"ended"`
	Rating    string `json:"rating"`  // formatted as "%.1f"
	Network   string `json:"network"` // network.name
	Genres    string `json:"genres"`  // comma-joined
	Summary   string `json:"summary"` // HTML stripped, first 200 chars
}

// Episode is one episode record from the /shows/{id}/episodes endpoint.
type Episode struct {
	ID      int    `kit:"id" json:"id"`
	Name    string `json:"name"`
	Season  int    `json:"season"`
	Number  int    `json:"number"`
	Airdate string `json:"airdate,omitempty"`
	Runtime int    `json:"runtime,omitempty"`
	Rating  string `json:"rating"`            // formatted as "%.1f"
	Summary string `json:"summary,omitempty"` // HTML stripped, first 150 chars
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
	ShowName string `kit:"id" json:"show"`
	Episode  string `json:"episode"`
	Season   int    `json:"season"`
	Number   int    `json:"number"`
	Airdate  string `json:"airdate"`
	Airtime  string `json:"airtime"`
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
	Ended     string `json:"ended"`
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
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Birthday string `json:"birthday"`
		Country  struct {
			Name string `json:"name"`
		} `json:"country"`
	} `json:"person"`
	Character struct {
		Name string `json:"name"`
	} `json:"character"`
}
