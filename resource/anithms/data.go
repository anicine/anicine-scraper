package anithms

import "log/slog"

var (
	headers = map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	logger = slog.Default().WithGroup("[ANIME-THEMES]")
)

type animeThemesAnime struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	MediaFormat string `json:"media_format,omitempty"`
	Season      string `json:"season,omitempty"`
	Slug        string `json:"slug,omitempty"`
	Synopsis    string `json:"synopsis,omitempty"`
	Year        int    `json:"year,omitempty"`
	Images      []struct {
		ID    int    `json:"id,omitempty"`
		Facet string `json:"facet,omitempty"`
		Path  string `json:"path,omitempty"`
		Link  string `json:"link,omitempty"`
	} `json:"images,omitempty"`
	Animethemes []struct {
		ID                int    `json:"id,omitempty"`
		Sequence          any    `json:"sequence,omitempty"`
		Slug              string `json:"slug,omitempty"`
		Type              string `json:"type,omitempty"`
		Animethemeentries []struct {
			ID       int    `json:"id,omitempty"`
			Episodes string `json:"episodes,omitempty"`
			Notes    string `json:"notes,omitempty"`
			Nsfw     bool   `json:"nsfw,omitempty"`
			Spoiler  bool   `json:"spoiler,omitempty"`
			Version  any    `json:"version,omitempty"`
			Videos   []struct {
				ID         int    `json:"id,omitempty"`
				Basename   string `json:"basename,omitempty"`
				Filename   string `json:"filename,omitempty"`
				Lyrics     bool   `json:"lyrics,omitempty"`
				Nc         bool   `json:"nc,omitempty"`
				Overlap    string `json:"overlap,omitempty"`
				Path       string `json:"path,omitempty"`
				Resolution int    `json:"resolution,omitempty"`
				Size       int    `json:"size,omitempty"`
				Source     string `json:"source,omitempty"`
				Subbed     bool   `json:"subbed,omitempty"`
				Uncen      bool   `json:"uncen,omitempty"`
				Tags       string `json:"tags,omitempty"`
				Link       string `json:"link,omitempty"`
			} `json:"videos,omitempty"`
		} `json:"animethemeentries,omitempty"`
		Group any `json:"group,omitempty"`
		Song  struct {
			ID    int    `json:"id,omitempty"`
			Title string `json:"title,omitempty"`
		} `json:"song,omitempty"`
	} `json:"animethemes,omitempty"`
}

type animeThemesSearch struct {
	Anime []animeThemesAnime `json:"anime,omitempty"`
	Links struct {
		First string `json:"first,omitempty"`
		Last  string `json:"last,omitempty"`
		Prev  string `json:"prev,omitempty"`
		Next  string `json:"next,omitempty"`
	} `json:"links,omitempty"`
	Meta struct {
		CurrentPage int `json:"current_page,omitempty"`
		From        int `json:"from,omitempty"`
		LastPage    int `json:"last_page,omitempty"`
		Links       []struct {
			URL    string `json:"url,omitempty"`
			Label  string `json:"label,omitempty"`
			Active bool   `json:"active,omitempty"`
		} `json:"links,omitempty"`
		Path    string `json:"path,omitempty"`
		PerPage int    `json:"per_page,omitempty"`
		To      int    `json:"to,omitempty"`
		Total   int    `json:"total,omitempty"`
	} `json:"meta,omitempty"`
}
