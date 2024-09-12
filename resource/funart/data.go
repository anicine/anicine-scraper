package funart

import (
	"log/slog"
	"sync"
)

var (
	tokens = []string{}
	idx    int
	mutex  sync.RWMutex
	logger = slog.Default().WithGroup("[FUN-ART]")
)

func generate() string {
	mutex.RLock()
	defer mutex.RUnlock()
	if len(tokens) == 0 {
		return ""
	}

	token := tokens[idx]
	idx = (idx + 1) % len(tokens)

	return token
}

func length() int {
	mutex.RLock()
	defer mutex.RUnlock()

	return len(tokens)
}

func SetTokens(data ...string) {
	mutex.Lock()
	defer mutex.Unlock()
	tokens = append(tokens, data...)
}

type FunArtMovie struct {
	Name        string `json:"name,omitempty"`
	TmdbID      string `json:"tmdb_id,omitempty"`
	ImdbID      string `json:"imdb_id,omitempty"`
	HdMovieLogo []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"hdmovielogo,omitempty"`
	MovieDisc []struct {
		ID       string `json:"id,omitempty"`
		URL      string `json:"url,omitempty"`
		Lang     string `json:"lang,omitempty"`
		Likes    string `json:"likes,omitempty"`
		Disc     string `json:"disc,omitempty"`
		DiscType string `json:"disc_type,omitempty"`
	} `json:"moviedisc,omitempty"`
	MovieLogo []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"movielogo,omitempty"`
	MoviePoster []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"movieposter,omitempty"`
	HdMovieClearArt []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"hdmovieclearart,omitempty"`
	MovieArt []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"movieart,omitempty"`
	MovieBackground []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"moviebackground,omitempty"`
	MovieBanner []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"moviebanner,omitempty"`
	MovieThumb []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"moviethumb,omitempty"`
}

type FunArtTv struct {
	Name      string `json:"name,omitempty"`
	TheTvdbID string `json:"thetvdb_id,omitempty"`
	ClearLogo []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"clearlogo,omitempty"`
	HdTvLogo []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"hdtvlogo,omitempty"`
	ClearArt []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"clearart,omitempty"`
	ShowBackground []struct {
		ID     string `json:"id,omitempty"`
		URL    string `json:"url,omitempty"`
		Lang   string `json:"lang,omitempty"`
		Likes  string `json:"likes,omitempty"`
		Season string `json:"season,omitempty"`
	} `json:"showbackground,omitempty"`
	TvThumb []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"tvthumb,omitempty"`
	SeasonPoster []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"seasonposter,omitempty"`
	SeasonThumb []struct {
		ID     string `json:"id,omitempty"`
		URL    string `json:"url,omitempty"`
		Lang   string `json:"lang,omitempty"`
		Likes  string `json:"likes,omitempty"`
		Season string `json:"season,omitempty"`
	} `json:"seasonthumb,omitempty"`
	HdClearArt []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"hdclearart,omitempty"`
	TvBanner []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"tvbanner,omitempty"`
	CharacterArt []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"characterart,omitempty"`
	TvPoster []struct {
		ID    string `json:"id,omitempty"`
		URL   string `json:"url,omitempty"`
		Lang  string `json:"lang,omitempty"`
		Likes string `json:"likes,omitempty"`
	} `json:"tvposter,omitempty"`
	SeasonBanner []struct {
		ID     string `json:"id,omitempty"`
		URL    string `json:"url,omitempty"`
		Lang   string `json:"lang,omitempty"`
		Likes  string `json:"likes,omitempty"`
		Season string `json:"season,omitempty"`
	} `json:"seasonbanner,omitempty"`
}
