package anithms

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/internal/shared"
	"github.com/anicine/anicine-scraper/models"
)

// this function search for the anime theme and return it
func Fetch(ctx context.Context, original, english, romanji string, year int, season string) (*models.AnimeThemes, error) {
	args := &client.Args{
		Proxy:   true,
		Method:  http.MethodGet,
		Headers: headers,
		Endpoint: &url.URL{
			Scheme:   "https",
			Host:     "api.animethemes.moe",
			Path:     "/anime",
			RawQuery: "page%5Bsize%5D=15&page%5Bnumber%5D=1&q=" + url.QueryEscape(original) + "&include=animethemes.group,animethemes.animethemeentries.videos,animethemes.song,images",
		},
	}

	var (
		stop  bool
		err   error
		anime *animeThemesAnime
	)
	for !stop {
		if err = func() error {
			stop = true
			body, err := client.Do(ctx, args)
			if err != nil {
				logger.Error("cannot get data", "error", err)
				return err
			}

			var data animeThemesSearch
			err = json.NewDecoder(body).Decode(&data)
			if err != nil {
				logger.Error("cannot decode JSON data", "error", err)
				return err
			}

			if len(data.Anime) == 0 {
				return nil
			}

			for _, v := range data.Anime {
				if v.Year == year {
					if analyze.CleanTitle(season) != analyze.CleanTitle(v.Season) {
						t0 := analyze.CleanTitle(v.Name)
						t1 := shared.TextAdvancedSimilarity(analyze.CleanTitle(romanji), t0)
						t2 := shared.TextAdvancedSimilarity(analyze.CleanTitle(english), t0)
						if !(t1 > 90 || t2 > 90) {
							continue
						}
					}
					anime = &v
					return nil
				}
			}

			if strings.TrimSpace(data.Links.Next) == "" || data.Links.Last == data.Links.Next {
				return errors.New("not found")
			}

			args.Endpoint, err = url.Parse(data.Links.Next)
			if err != nil {
				return err
			}

			stop = false
			return nil
		}(); err != nil {
			return nil, err
		}
	}

	if anime == nil {
		return nil, errs.ErrNotFound
	}

	return clean(anime), nil
}

func clean(anime *animeThemesAnime) *models.AnimeThemes {
	themes := new(models.AnimeThemes)
	for _, v1 := range anime.Animethemes {
		var theme models.AnimeThemeItem
		theme.Song = analyze.CleanUnicode(v1.Song.Title)
		for _, v2 := range v1.Animethemeentries {
			var entry models.AnimeThemeEntry
			entry.Episodes = analyze.ExtractIntsWithRanges(v2.Episodes)
			for _, v3 := range v2.Videos {
				if v3.Basename == "" {
					continue
				}
				entry.Videos = append(entry.Videos, models.AnimeThemeVideo{
					Source:     v3.Source,
					Resolution: v3.Resolution,
					Link:       "https://v.animethemes.moe/" + v3.Basename,
				})
			}
			theme.Entries = append(theme.Entries, entry)
		}

		if strings.Contains(strings.ToLower(v1.Type), "op") {
			themes.OP = append(themes.OP, theme)
		}
		if strings.Contains(strings.ToLower(v1.Type), "ed") {
			themes.ED = append(themes.ED, theme)
		}
	}

	return themes
}
