package scrape

import (
	"log/slog"
	"net/url"

	"github.com/anicine/anicine-scraper/models"
)

var (
	witAnimeLog    = slog.Default().WithGroup("[WIT-ANIME]")
	anime4upLog    = slog.Default().WithGroup("[ANIME-4-UP]")
	animeLekLog    = slog.Default().WithGroup("[ANIME-LEK]")
	animeRcoLog    = slog.Default().WithGroup("[ANIME-RCO]")
	animeSaturnLog = slog.Default().WithGroup("[ANIME-SATURN]")
	animeUnityLog  = slog.Default().WithGroup("[ANIME-UNITY]")
	animeSlayerLog = slog.Default().WithGroup("[ANIME-SLAYER]")
	gogoAnimeLog   = slog.Default().WithGroup("[GOGO-ANIME]")
	jkAnimeLog     = slog.Default().WithGroup("[JK-ANIME]")
	okAnimeLog     = slog.Default().WithGroup("[OK-ANIME]")
	shahidAnimeLog = slog.Default().WithGroup("[SHAHID-ANIME]")
	sAnimeLog      = slog.Default().WithGroup("[S-ANIME]")
)

type EmbedNode struct {
	Number   int
	Videos   []models.AnimeVideo
	Download []models.AnimeVideo
}

type EpisodeNode struct {
	Number int
	Type   string
	Link   *url.URL
}
