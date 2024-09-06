package mapping

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/models"
)

type notifyMoeItem struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	StartDate string `json:"startDate,omitempty"`
	Mappings  []struct {
		Service   string `json:"service,omitempty"`
		ServiceID string `json:"serviceId,omitempty"`
	} `json:"mappings,omitempty"`
}

type notifyMoe struct {
	id   string
	info *models.AnimeInfo
	ctx  context.Context
}

func NotifyMoe(ctx context.Context, info *models.AnimeInfo, id string) (*models.AnimeResource, error) {
	x := notifyMoe{
		id:   id,
		info: info,
		ctx:  ctx,
	}

	var (
		resource *models.AnimeResource
		err      error
	)

	if len(x.id) > 0 {
		resource, err = x.check()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
		if resource != nil {
			return resource, nil
		}
	}

	found, err := x.query()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
	}
	if found {
		resource, err = x.check()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
		if resource != nil {
			return resource, nil
		}
	}

	return nil, errs.ErrNotFound
}

func (x *notifyMoe) query() (bool, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Method: http.MethodGet,
		Endpoint: &url.URL{
			Scheme: "https",
			Host:   "notify.moe",
			Path:   "/_/anime-search/" + x.info.Query,
		},
		Headers: map[string]string{
			"authority": "notify.moe",
		},
	})
	if err != nil {
		return false, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return false, err
	}

	var found bool
	block := doc.Find(".anime-search")
	if block != nil {
		block.Find("a").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if title, ok := s.Attr("aria-label"); ok {
				if !strings.Contains(analyze.CleanTitle(title), x.info.Query) {
					return true
				}
			}

			if href, ok := s.Attr("href"); ok {
				if href == "" {
					return true
				}

				body, err := client.Do(x.ctx, &client.Args{
					Method: http.MethodGet,
					Endpoint: &url.URL{
						Scheme: "https",
						Host:   "notify.moe",
						Path:   "/api" + href,
					},
					Headers: map[string]string{
						"authority": "notify.moe",
						"referer":   "https://notify.moe",
					},
				})
				if err != nil {
					return true
				}

				var anime notifyMoeItem
				err = json.NewDecoder(body).Decode(&anime)
				if err != nil {
					return true
				}

				if anime.StartDate != "" {
					date, err := time.Parse(time.DateOnly, anime.StartDate)
					if err != nil {
						return true
					}

					if x.info.SD.Year == date.Year() && time.Month(x.info.SD.Month) == date.Month() {
						found = true
						x.id = anime.ID
						return false
					}
				}
			}
			return true
		})
	}

	return found, nil
}

func (x *notifyMoe) check() (*models.AnimeResource, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Method: http.MethodGet,
		Endpoint: &url.URL{
			Scheme: "https",
			Host:   "notify.moe",
			Path:   "/api/anime/" + x.id,
		},
		Headers: map[string]string{
			"authority": "notify.moe",
			"referer":   "https://notify.moe",
		},
	})
	if err != nil {
		return nil, err
	}

	var anime notifyMoeItem
	err = json.NewDecoder(body).Decode(&anime)
	if err != nil {
		return nil, err
	}

	date, err := time.Parse(time.DateOnly, anime.StartDate)
	if err != nil {
		return nil, err
	}

	if x.info.SD.Year == date.Year() && time.Month(x.info.SD.Month) == date.Month() {
		resource := new(models.AnimeResource)

		resource.NotifyMoe = anime.ID

		for _, v1 := range anime.Mappings {
			if strings.Contains(v1.Service, "myanimelist/anime") {
				id := analyze.ExtractNum(v1.ServiceID)
				if id != x.info.MalID {
					return nil, errs.ErrNoData
				}
				resource.Mal = id
				continue
			}
			if strings.Contains(v1.Service, "anilist/anime") {
				if id := analyze.ExtractNum(v1.ServiceID); id != 0 {
					resource.AniList = id
				}
				continue
			}
			if strings.Contains(v1.Service, "kitsu/anime") {
				if id := analyze.ExtractNum(v1.ServiceID); id != 0 {
					resource.Kitsu = strconv.Itoa(id)
				}
				continue
			}
			if strings.Contains(v1.Service, "anidb/anime") {
				if id := analyze.ExtractNum(v1.ServiceID); id != 0 {
					resource.AniDB = id
				}
				continue
			}
		}

		return resource, nil
	}

	return nil, errs.ErrNotFound
}
