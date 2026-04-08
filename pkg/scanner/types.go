package scanner

type MediaType string

const (
	TypeMovie  MediaType = "movie"
	TypeSeries MediaType = "series"
	TypeFiles  MediaType = "files"
)

type ScanRoot struct {
	Path string
	Type MediaType
}

type MediaItem struct {
	Type       MediaType `json:"type"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Group      string    `json:"group,omitempty"`
	Series     string    `json:"series,omitempty"`
	Collection string    `json:"collection,omitempty"`
	Season     int       `json:"season,omitempty"`
	SeasonName string    `json:"season_name,omitempty"`
	Episode    int       `json:"episode,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Filename   string    `json:"filename"`
}
