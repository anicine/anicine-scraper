package scrape

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/anicine/anicine-scraper/models"
)

type anime4upWatch struct {
	HD  map[string]any `json:"hd,omitempty"`
	FHD map[string]any `json:"fhd,omitempty"`
	SD  map[string]any `json:"sd,omitempty"`
}

type anime4upDownload struct {
	SD  []string `json:"sd,omitempty"`
	HD  []string `json:"hd,omitempty"`
	FHD []string `json:"fhd,omitempty"`
}

type anime4up struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func Anime4up(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := anime4up{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      anime4upLog,
		ctx:      ctx,
	}

	var err error
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

func (x *anime4up) query(args *client.Args) (*url.URL, error) {
	var err error
	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot get the anime page", "error", err)
		return nil, err
	}

	html, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime page", "error", err)
		return nil, err
	}

	if href, ok := html.Find("a.anime-mal").Attr("href"); ok {
		if strings.Contains(href, strconv.Itoa(x.info.MalID)) {
			x.log.Info("anime page url", "page", args.Endpoint.String())
			return args.Endpoint, nil
		}
	}

	return nil, errs.ErrNotFound
}

func (x *anime4up) search() (*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "anime4up.lol",
		Path:     "/",
		RawQuery: "search_param=animes&s=" + analyze.CleanQuery(x.info.Query),
	}

	args := &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}
	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot query the anime", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the search page", "error", err)
		return nil, err
	}

	var (
		found bool
		link  *url.URL
	)
	doc.Find(".anime-card-details").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if !strings.Contains(strings.ToLower(s.Find(".anime-card-type").Text()), x.info.Type) {
			return true
		}

		if title := s.Find(".anime-card-title"); title != nil {
			if strings.Contains(analyze.CleanTitle(title.Text()), x.info.Query) {
				href, ok := title.Find("a").Attr("href")
				if !ok {
					return true
				}

				args.Endpoint, err = url.Parse(href)
				if err != nil {
					x.log.Error("cannot parse the anime page url", "error", err)
					return true
				}

				time.Sleep(100 * time.Millisecond)

				link, err = x.query(args)
				if err != nil {
					return true
				}
				found = true
				return false
			}
		}
		return true
	})
	if found {
		return link, nil
	}

	return nil, errs.ErrNotFound
}

func (x *anime4up) fetch(endpoint *url.URL) ([]*EpisodeNode, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
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

	if x.isMovie {
		if href, ok := doc.Find("div.DivEpisodeContainer").Find("a").Attr("href"); ok {
			link, err := url.Parse(href)
			if err != nil {
				x.log.Error("cannot parse the page url", "error", err)
				return nil, err
			}

			x.log.Info("found movie episode url", "link", link.Path)

			return []*EpisodeNode{
				{
					Number: -1,
					Link:   link,
				},
			}, nil
		}
	} else {
		nodes := make([]*EpisodeNode, len(x.episodes))
		doc.Find("#DivEpisodesList").Find("div.DivEpisodeContainer").Each(func(_ int, s *goquery.Selection) {
			a := s.Find("a")
			if a == nil {
				return
			}

			for i, v := range x.episodes {
				if strings.TrimSpace(a.Text()) != fmt.Sprintf("الحلقة %d", v) {
					continue
				}
				if href, ok := a.Attr("href"); ok {
					link, err := url.Parse(href)
					if err != nil {
						x.log.Error("cannot parse the episode url", "ep", v, "error", err)
						continue
					}

					x.log.Info("found episode url", "ep", v, "link", link.Path)

					nodes[i] = &EpisodeNode{
						Number: v,
						Link:   link,
					}
					break
				}
			}
		})

		return nodes, nil
	}

	return nil, errs.ErrNotFound
}

func (x *anime4up) scrape(nodes ...*EpisodeNode) (*[]*EmbedNode, error) {
	var (
		err    error
		result = make([]*EmbedNode, len(nodes))
	)
	for i, v := range nodes {
		if err = func() error {
			time.Sleep(100 * time.Millisecond)
			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: v.Link,
			})
			if err != nil {
				x.log.Error("cannot load the episode page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse the episode page", "error", err)
				return err
			}

			var (
				videos    []models.AnimeVideo
				downloads []models.AnimeVideo
			)
			doc.Find("div.watchForm").Find("input").Each(func(_ int, s *goquery.Selection) {
				if name, ok := s.Attr("name"); ok {
					name = strings.TrimSpace(name)
					if name == "wl" {
						if value, ok := s.Attr("value"); ok {
							body, err := base64.StdEncoding.DecodeString(value)
							if err != nil {
								x.log.Error("cannot decode videos body", "error", err)
								return
							}

							var data anime4upWatch
							err = json.Unmarshal(body, &data)
							if err != nil {
								x.log.Error("cannot parse videos body", "error", err)
								return
							}

							for _, v := range data.FHD {
								if v != nil {
									if src, ok := v.(string); ok {
										if src != "" {
											x.log.Info("found iframe source", "src", src)

											videos = append(videos, models.AnimeVideo{
												Source:   src,
												Type:     "sub",
												Language: analyze.CleanLanguage("arabic"),
												Quality:  "fhd",
											})
										}
									}
								}
							}
							for _, v := range data.HD {
								if v != nil {
									if src, ok := v.(string); ok {
										if src != "" {
											x.log.Info("found iframe source", "src", src)

											videos = append(videos, models.AnimeVideo{
												Source:   src,
												Type:     "sub",
												Language: analyze.CleanLanguage("arabic"),
												Quality:  "hd",
											})
										}
									}
								}
							}
							for _, v := range data.SD {
								if v != nil {
									if src, ok := v.(string); ok {
										if src != "" {
											x.log.Info("found iframe source", "src", src)

											videos = append(videos, models.AnimeVideo{
												Source:   src,
												Type:     "sub",
												Language: analyze.CleanLanguage("arabic"),
												Quality:  "sd",
											})
										}
									}
								}
							}
						}
						return
					} else if name == "dl" {
						if value, ok := s.Attr("value"); ok {
							body, err := base64.StdEncoding.DecodeString(value)
							if err != nil {
								x.log.Error("cannot decode videos body", "error", err)
								return
							}

							var data anime4upDownload
							err = json.Unmarshal(body, &data)
							if err != nil {
								x.log.Error("cannot parse videos body", "error", err)
								return
							}
							for _, v := range data.FHD {
								if v != "" {
									x.log.Info("found download url", "url", v)

									downloads = append(downloads, models.AnimeVideo{
										Source:   v,
										Type:     "sub",
										Language: analyze.CleanLanguage("arabic"),
										Quality:  "fhd",
									})
								}
							}
							for _, v := range data.HD {
								if v != "" {
									x.log.Info("found download url", "url", v)

									downloads = append(downloads, models.AnimeVideo{
										Source:   v,
										Type:     "sub",
										Language: analyze.CleanLanguage("arabic"),
										Quality:  "hd",
									})
								}
							}
							for _, v := range data.SD {
								if v != "" {
									x.log.Info("found download url", "url", v)

									downloads = append(downloads, models.AnimeVideo{
										Source:   v,
										Type:     "sub",
										Language: analyze.CleanLanguage("arabic"),
										Quality:  "sd",
									})
								}
							}
						}

						return
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
			return nil, err
		}
	}

	return &result, nil
}
