package scrape

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/models"
)

type sAnimeSearch []struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Year string `json:"year,omitempty"`
}

type sAnimeItem struct {
	ID           string       `json:"id,omitempty"`
	Name         string       `json:"name,omitempty"`
	AnimeRelease string       `json:"anime_release,omitempty"`
	Type         string       `json:"type,omitempty"`
	Ep           [][]sAnimeEp `json:"ep,omitempty"`
}

type sAnimeEp struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	EpName any    `json:"epName,omitempty"`
	Date   string `json:"date,omitempty"`
}

type sAnimeVideo struct {
	Sd string `json:"sd,omitempty"`
	Hd string `json:"hd,omitempty"`
}

type sAnime struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func SAnime(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := sAnime{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      sAnimeLog,
		ctx:      ctx,
	}

	pages, err := x.search()
	if err != nil {
		return nil, err
	}

	nodes, err := x.fetch(pages)
	if err != nil {
		return nil, err
	}

	return x.scrape(nodes)
}

func (x *sAnime) search() ([]string, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:  true,
		Method: http.MethodGet,
		Endpoint: &url.URL{
			Scheme:   "https",
			Host:     "app.sanime.net",
			Path:     "/function/h10.php",
			RawQuery: "page=search&name=" + url.QueryEscape(strings.ReplaceAll(x.info.Title, "-", " ")),
		},
	})
	if err != nil {
		x.log.Error("cannot query the anime", "error", err)
		return nil, err
	}

	var queries sAnimeSearch
	err = json.NewDecoder(body).Decode(&queries)
	if err != nil {
		x.log.Error("cannot parse the search data", "error", err)
		return nil, err
	}

	var ids []string
	for _, v := range queries {
		if v.Year == strconv.Itoa(x.info.SD.Year) {
			ids = append(ids, v.ID)
		}
	}
	if len(ids) == 0 {
		return nil, errs.ErrNotFound
	}

	return ids, nil
}

func (x *sAnime) fetch(ids []string) ([]*EpisodeNode, error) {
	var (
		err   error
		nodes []*EpisodeNode
	)
	for _, v := range ids {
		if err = func() error {
			endpoint := &url.URL{
				Scheme:   "https",
				Host:     "app.sanime.net",
				Path:     "/function/h10.php",
				RawQuery: "page=info&id=" + v,
			}
			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: endpoint,
			})
			if err != nil {
				x.log.Error("cannot query the anime", "error", err)
				return err
			}

			var anime sAnimeItem
			err = json.NewDecoder(body).Decode(&anime)
			if err != nil {
				x.log.Error("cannot parse the anime data", "error", err)
				return err
			}

			show := strings.ToLower(anime.Type)
			if x.isMovie {
				if !(strings.Contains(show, "movie") || strings.Contains(show, "فلم")) {
					return errs.ErrNotFound
				}
			} else {
				if !(strings.Contains(show, "serie") || strings.Contains(show, "مسلسل")) {
					return errs.ErrNotFound
				}
			}

			points := 1
			name := analyze.CleanTitle(analyze.ExtractEngChars(anime.Name))
			if name == x.info.Query || name == x.info.Title {
				points += 1
			}

			func() {
				date, err := time.Parse(time.DateOnly, anime.AnimeRelease)
				if err != nil {
					return
				}
				if date.Year() == x.info.SD.Year && int(date.Month()) == x.info.SD.Month {
					points += 1
				}
			}()

			if !x.isMovie {
				if len(anime.Ep) > 0 {
					if len(anime.Ep[0]) > 0 {
						date, _ := time.Parse(time.DateOnly, anime.Ep[0][len(anime.Ep[0])-1].Date)
						if !date.IsZero() {
							if date.Year() == x.info.SD.Year && int(date.Month()) == x.info.SD.Month {
								points += 1
							}
						}
					} else {
						return errs.ErrNotFound
					}
				} else {
					return errs.ErrNotFound
				}
			}

			if points > 1 {
				if len(anime.Ep) > 0 {
					if len(anime.Ep[0]) > 0 {
						if x.isMovie {
							raw, err := json.Marshal(&anime.Ep[0][0])
							if err != nil {
								return errs.ErrNotFound
							}

							node := &EpisodeNode{
								Number: -1,
								Link: &url.URL{
									Scheme:   endpoint.Scheme,
									Host:     endpoint.Host,
									Path:     endpoint.Path,
									RawQuery: "page=openAnd&id=" + base64.StdEncoding.EncodeToString(raw),
								},
							}
							x.log.Info("found movie episode url", "link", node.Link.Path)

							nodes = append(nodes, node)
						} else {
							for _, y := range anime.Ep[0] {
								for _, z := range x.episodes {
									if strings.Contains(y.Name, "خاص") {
										continue
									}
									if fmt.Sprintf("%sEP-%d", v, z) == y.ID {
										raw, err := json.Marshal(&y)
										if err != nil {
											return errs.ErrNotFound
										}
										node := &EpisodeNode{
											Number: z,
											Link: &url.URL{
												Scheme:   endpoint.Scheme,
												Host:     endpoint.Host,
												Path:     endpoint.Path,
												RawQuery: "page=openAnd&id=" + base64.StdEncoding.EncodeToString(raw),
											},
										}
										x.log.Info("found episode url", "ep", z, "link", node.Link.Path)

										nodes = append(nodes, node)
									}
								}
							}
						}
					}
				}
			}
			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	if len(nodes) == 0 {
		return nil, errs.ErrNotFound
	}

	return nodes, nil
}

func (x *sAnime) scrape(nodes []*EpisodeNode) (*[]*EmbedNode, error) {
	var (
		err    error
		result = make([]*EmbedNode, len(nodes))
	)
	for i, v := range nodes {
		if v == nil || v.Link == nil {
			continue
		}
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

			var video sAnimeVideo
			err = json.NewDecoder(body).Decode(&video)
			if err != nil {
				x.log.Error("cannot parse the episode data", "error", err)
				return err
			}

			var videos []models.AnimeVideo
			if video.Hd != "" {
				x.log.Info("found iframe source", "src", video.Hd)
				videos = append(videos, models.AnimeVideo{
					Source:   video.Hd,
					Type:     "sub",
					Quality:  "hd",
					Language: analyze.CleanLanguage("arabic"),
				})
			}

			if video.Sd != "" {
				x.log.Info("found iframe source", "src", video.Sd)
				videos = append(videos, models.AnimeVideo{
					Source:   video.Sd,
					Type:     "sub",
					Quality:  "sd",
					Language: analyze.CleanLanguage("arabic"),
				})
			}
			result[i] = &EmbedNode{
				Number: v.Number,
				Videos: videos,
			}
			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	if len(result) == 0 {
		return nil, errs.ErrNotFound
	}

	return &result, nil
}
