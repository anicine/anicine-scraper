package scrape

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/internal/shared"
	"github.com/anicine/anicine-scraper/models"
)

type okAnimeSearch struct {
	Results []struct {
		ID    int    `json:"id,omitempty"`
		Title string `json:"title,omitempty"`
		URL   string `json:"url,omitempty"`
		Year  int    `json:"year,omitempty"`
	} `json:"results,omitempty"`
}

type okAnimeItem struct {
	Data struct {
		Attributes struct {
			URL string `json:"url,omitempty"`
		} `json:"attributes,omitempty"`
	} `json:"data,omitempty"`
}

type okAnime struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func OkAnime(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := okAnime{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      okAnimeLog,
		ctx:      ctx,
	}

	page, err := x.search()
	if err != nil {
		return nil, err
	}

	nodes, err := x.fetch(page)
	if err != nil {
		return nil, err
	}

	return x.scrape(nodes...)
}

func (x *okAnime) search() (*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "okanime.tv",
		Path:     "/json/search",
		RawQuery: "term=" + analyze.CleanQuery(x.info.Query),
	}

	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
		Headers: map[string]string{
			"referer":          "https://okanime.tv/search",
			"x-requested-with": "XMLHttpRequest",
		},
	})
	if err != nil {
		x.log.Error("cannot query the anime", "error", err)
		return nil, err
	}

	var data okAnimeSearch
	err = json.NewDecoder(body).Decode(&data)
	if err != nil {
		x.log.Error("cannot parse the search data", "error", err)
		return nil, err
	}

	for _, v := range data.Results {
		if v.Year != x.info.SD.Year {
			continue
		}
		if shared.TextAdvancedSimilarity(analyze.CleanTitle(v.Title), x.info.Query) < 50 {
			continue
		}

		query, err := url.Parse("https://okanime.tv/partials/anime_tab?anime_id=" + strconv.Itoa(v.ID) + "&expires_in=86400")
		if err != nil {
			continue
		}

		return query, nil
	}

	return nil, errs.ErrNotFound
}

func (x *okAnime) fetch(endpoint *url.URL) ([]*EpisodeNode, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Method:   http.MethodGet,
		Endpoint: endpoint,
		Headers: map[string]string{
			"referer":          endpoint.String(),
			"sec-fetch-site":   "same-origin",
			"x-requested-with": "XMLHttpRequest",
		},
	})
	if err != nil {
		x.log.Error("cannot get the anime page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime page", "error", err)
		return nil, err
	}

	var nodes []*EpisodeNode
	doc.Find("div.enable-photos-box").Find("div.row").Each(func(_ int, s *goquery.Selection) {
		s.Find(".item").Each(func(_ int, z *goquery.Selection) {
			info := z.Find(".video-subtitle").Text()
			if info == "" {
				return
			}
			if !x.isMovie {
				for _, v := range x.episodes {
					if strings.ReplaceAll(info, " ", "") == fmt.Sprintf("الحلقة%d", v) {
						if href, ok := z.Attr("href"); ok {
							if href != "" {
								link, err := url.Parse(endpoint.Scheme + "://" + endpoint.Host + href)
								if err != nil {
									x.log.Error("cannot parse the episode url", "ep", v, "error", err)
									return
								}
								x.log.Info("found episode url", "ep", v, "link", link.Path)
								nodes = append(nodes, &EpisodeNode{
									Number: v,
									Link:   link,
								})
							}
						}
						break
					}
				}
			} else {
				if href, ok := z.Attr("href"); ok {
					if href != "" {
						link, err := url.Parse(endpoint.Scheme + "://" + endpoint.Host + href)
						if err != nil {
							x.log.Error("cannot parse the episode url", "error", err)
							return
						}
						x.log.Info("found movie episode url", "link", link.Path)
						nodes = append(nodes, &EpisodeNode{
							Number: -1,
							Link:   link,
						})
					}
				}
				return
			}

		})
	})
	if len(nodes) == 0 {
		return nil, errs.ErrNotFound
	}

	return nodes, nil
}

func (x *okAnime) scrape(nodes ...*EpisodeNode) (*[]*EmbedNode, error) {
	var (
		err    error
		result = make([]*EmbedNode, len(nodes))
		args   = &client.Args{
			Method: http.MethodGet,
			Headers: map[string]string{
				"sec-fetch-site":   "same-origin",
				"x-requested-with": "XMLHttpRequest",
			},
		}
	)
	for i, v := range nodes {
		select {
		case <-x.ctx.Done():
			return nil, context.Canceled
		default:
			if v == nil || v.Link == nil {
				continue
			}

			time.Sleep(100 * time.Millisecond)
			if err = func() error {
				body, err := client.Do(x.ctx, &client.Args{
					Proxy:    true,
					Method:   http.MethodGet,
					Endpoint: v.Link,
				})
				if err != nil {
					x.log.Error("cannot get the episode page", "error", err)
					return err
				}

				doc, err := goquery.NewDocumentFromReader(body)
				if err != nil {
					x.log.Error("cannot parse the episode page", "error", err)
					return err
				}

				var videos []models.AnimeVideo
				doc.Find("#myTabContent").Find("#watch").Find(".servers-list").Each(func(_ int, s *goquery.Selection) {
					if href, ok := s.Attr("data-href"); ok {
						if href != "" {
							args.Endpoint, err = url.Parse(v.Link.Scheme + "://" + v.Link.Host + href)
							if err != nil {
								return
							}

							time.Sleep(100 * time.Millisecond)
							page, err := client.Do(x.ctx, args)
							if err != nil {
								x.log.Error("cannot get the episode page", "error", err)
								return
							}

							var result okAnimeItem
							err = json.NewDecoder(page).Decode(&result)
							if err != nil {
								return
							}
							if result.Data.Attributes.URL != "" {
								quality := strings.ToLower(strings.TrimSpace(s.Find("small").Text()))
								if strings.Contains(quality, "hd") {
									quality = "hd"
								}
								if strings.Contains(quality, "fhd") {
									quality = "fhd"
								}
								if strings.Contains(quality, "sd") || strings.Contains(quality, "ld") {
									quality = "sd"
								}

								x.log.Info("found iframe source", "src", result.Data.Attributes.URL)

								videos = append(videos, models.AnimeVideo{
									Source:   strings.TrimSpace(result.Data.Attributes.URL),
									Referer:  v.Link.Scheme + "://" + v.Link.Host,
									Quality:  quality,
									Type:     "sub",
									Language: analyze.CleanLanguage("arabic"),
								})
							}
						}
					}
				})

				var downloads []models.AnimeVideo
				doc.Find("#myTabContent").Find("#download").Find(".webinars-inner-wrap").Find("a").Each(func(_ int, s *goquery.Selection) {
					if href, ok := s.Attr("href"); ok {
						if href != "" {
							quality := strings.ToLower(strings.TrimSpace(s.Find("small").Text()))
							if strings.Contains(quality, "hd") {
								quality = "hd"
							}
							if strings.Contains(quality, "fhd") {
								quality = "fhd"
							}
							if strings.Contains(quality, "sd") || strings.Contains(quality, "ld") {
								quality = "sd"
							}

							x.log.Info("found download url", "url", strings.TrimSpace(href))

							downloads = append(downloads, models.AnimeVideo{
								Source:   strings.TrimSpace(href),
								Quality:  quality,
								Type:     "sub",
								Language: analyze.CleanLanguage("arabic"),
							})
						}
					}
				})

				result[i] = &EmbedNode{
					Number:   v.Number,
					Videos:   videos,
					Download: downloads,
				}
				return nil
			}(); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil, err
				}
			}
		}
	}

	return &result, nil
}
