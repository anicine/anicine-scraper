package scrape

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/models"
)

const (
	clID     = "android-app2"
	clSecret = "7befba6263cc14c90d2f1d6da2c5cf9b251bfbbd"
)

var (
	asExp = regexp.MustCompile(`\?(.*)`)
)

type animeSlayerSearch struct {
	Response struct {
		Data []struct {
			AnimeID          string `json:"anime_id,omitempty"`
			AnimeName        string `json:"anime_name,omitempty"`
			AnimeType        string `json:"anime_type,omitempty"`
			AnimeReleaseYear string `json:"anime_release_year,omitempty"`
		} `json:"data,omitempty"`
	} `json:"response,omitempty"`
}

type animeSlayerItem struct {
	Response struct {
		AnimeID        string `json:"anime_id,omitempty"`
		MoreInfoResult struct {
			AiredFrom string `json:"aired_from,omitempty"`
			AiredTo   string `json:"aired_to,omitempty"`
		} `json:"more_info_result,omitempty"`
	} `json:"response,omitempty"`
}

type animeSlayerEp struct {
	Response struct {
		Data []struct {
			EpisodeID   string `json:"episode_id,omitempty"`
			EpisodeName string `json:"episode_name,omitempty"`
		} `json:"data,omitempty"`
	} `json:"response,omitempty"`
}

type animeSlayerData struct {
	Response struct {
		Data []struct {
			EpisodeUrls []struct {
				EpisodeURL string `json:"episode_url,omitempty"`
			} `json:"episode_urls,omitempty"`
		} `json:"data,omitempty"`
	} `json:"response,omitempty"`
}

type animeSlayer struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	id       string
	headers  map[string]string
	log      *slog.Logger
	ctx      context.Context
}

func AnimeSlayer(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := animeSlayer{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		headers: map[string]string{
			"Accept":          "application/*+json",
			"Accept-Language": "en",
			"Content-Type":    "application/x-www-form-urlencoded",
			"Client-Id":       clID,
			"Client-Secret":   clSecret,
		},
		log: animeSlayerLog,
		ctx: ctx,
	}

	ids, err := x.search()
	if err != nil {
		return nil, err
	}

	page, err := x.check(ids)
	if err != nil {
		return nil, err
	}

	nodes, err := x.fetch(page)
	if err != nil {
		return nil, err
	}

	return x.scrape(nodes...)
}

func (x *animeSlayer) search() ([]string, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "anslayer.com",
		Path:     "/anime/public/animes/get-published-animes",
		RawQuery: "json=" + url.QueryEscape(fmt.Sprintf(`{"_offset":0,"_limit":100,"_order_by":"latest_first","list_type":"filter","anime_name":"%s","just_info":"Yes"}`, strings.ReplaceAll(x.info.Query, "-", " "))),
	}

	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
		Headers: map[string]string{
			"Accept":          "application/*+json",
			"Accept-Language": "en",
			"Client-Id":       clID,
			"Client-Secret":   clSecret,
		},
	})
	if err != nil {
		x.log.Error("cannot query the anime", "error", err)
		return nil, err
	}

	var search animeSlayerSearch
	err = json.NewDecoder(body).Decode(&search)
	if err != nil {
		x.log.Error("cannot parse anime search data", "error", err)
		return nil, err
	}

	if len(search.Response.Data) == 0 {
		return nil, errs.ErrNotFound
	}

	var ids []string
	for _, v := range search.Response.Data {
		if !strings.Contains(strings.ToLower(v.AnimeType), x.info.Type) {
			continue
		}

		if !strings.Contains(v.AnimeReleaseYear, strconv.Itoa(x.info.SD.Year)) {
			continue
		}

		ids = append(ids, strings.TrimSpace(v.AnimeID))
	}

	if len(ids) == 0 {
		return nil, errs.ErrNotFound
	}

	return ids, nil
}

func (x *animeSlayer) check(ids []string) (*url.URL, error) {
	var (
		err  error
		link *url.URL
	)
	for _, v := range ids {
		if err = func() error {
			endpoint := &url.URL{
				Scheme:   "https",
				Host:     "anslayer.com",
				Path:     "/anime/public/anime/get-anime-details",
				RawQuery: "anime_id=" + v + "&fetch_episodes=No&more_info=Yes",
			}
			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Headers:  x.headers,
				Endpoint: endpoint,
			})
			if err != nil {
				x.log.Error("cannot get anime data", "error", err)
				return err
			}

			var anime animeSlayerItem
			err = json.NewDecoder(body).Decode(&anime)
			if err != nil {
				x.log.Error("cannot parse anime data", "error", err)
				return err
			}

			data := url.Values{
				"inf":  {""},
				"json": {fmt.Sprintf(`{"more_info":"No","anime_id":%s}`, v)},
			}

			if anime.Response.MoreInfoResult.AiredFrom != "" {
				sd, err := time.Parse(time.DateTime, anime.Response.MoreInfoResult.AiredFrom)
				if err != nil {
					return err
				}

				if sd.Year() == x.info.SD.Year && int(sd.Month()) == x.info.SD.Month {
					x.id = v
					link = &url.URL{
						Scheme:   endpoint.Scheme,
						Host:     endpoint.Host,
						Path:     "/anime/public/episodes/get-episodes-new",
						RawQuery: data.Encode(),
					}
					return nil
				}
			}
			if anime.Response.MoreInfoResult.AiredTo != "" {
				ed, err := time.Parse(time.DateTime, anime.Response.MoreInfoResult.AiredTo)
				if err != nil {
					return err
				}

				if ed.Year() == x.info.ED.Year && int(ed.Month()) == x.info.ED.Month {
					x.id = v
					link = &url.URL{
						Scheme:   endpoint.Scheme,
						Host:     endpoint.Host,
						Path:     "/anime/public/episodes/get-episodes-new",
						RawQuery: data.Encode(),
					}
					return nil
				}
			}

			return errs.ErrNoData
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}

		if link != nil {
			x.log.Info("found anime page", "url", link.String())
			return link, nil
		}
	}

	return nil, errs.ErrNotFound
}

func (x *animeSlayer) fetch(endpoint *url.URL) ([]*EpisodeNode, error) {
	args := &client.Args{
		Proxy:    true,
		Method:   http.MethodPost,
		Endpoint: endpoint,
		Body:     strings.NewReader(endpoint.RawQuery),
		Headers:  x.headers,
	}
	endpoint.RawQuery = ""
	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot get the anime episodes data", "error", err)
		return nil, err
	}

	var data animeSlayerEp
	err = json.NewDecoder(body).Decode(&data)
	if err != nil {
		x.log.Error("cannot parse the anime episodes data", "error", err)
		return nil, err
	}

	if len(data.Response.Data) == 0 {
		x.log.Warn("no episodes found")
		return nil, errs.ErrNotFound
	}

	var nodes []*EpisodeNode
	if x.isMovie {
		for _, y := range data.Response.Data {
			if strings.Contains(y.EpisodeName, "خاص") {
				continue
			}
			data := url.Values{
				"inf":  {""},
				"json": {fmt.Sprintf(`{"anime_id":%s,"episode_id":"%s"}`, x.id, y.EpisodeID)},
			}

			x.log.Info("found movie episode url", "link", endpoint.Path)

			nodes = append(nodes, &EpisodeNode{
				Number: -1,
				Link: &url.URL{
					Scheme:   endpoint.Scheme,
					Host:     endpoint.Host,
					Path:     endpoint.Path,
					RawQuery: data.Encode(),
				},
			})
		}
	} else {
		for _, y := range data.Response.Data {
			if strings.Contains(y.EpisodeName, "خاص") {
				continue
			}

			ep := analyze.ExtractNum(y.EpisodeName)
			if ep == 0 {
				continue
			}

			for _, z := range x.episodes {
				if ep == z {
					data := url.Values{
						"inf":  {""},
						"json": {fmt.Sprintf(`{"anime_id":%s,"episode_id":"%s"}`, x.id, y.EpisodeID)},
					}

					x.log.Info("found episode url", "ep", ep, "link", endpoint.Path)
					nodes = append(nodes, &EpisodeNode{
						Number: z,
						Link: &url.URL{
							Scheme:   endpoint.Scheme,
							Host:     endpoint.Host,
							Path:     endpoint.Path,
							RawQuery: data.Encode(),
						},
					})
				}
			}
		}
	}

	if len(nodes) == 0 {
		return nil, errs.ErrNotFound
	}

	return nodes, nil
}

func (x *animeSlayer) extract(endpoint *url.URL, data url.Values) ([]string, error) {
	values := url.Values{}
	for k, s := range data {
		matches := asExp.FindStringSubmatch(k)
		x := ""
		if len(s) > 0 {
			x = s[len(s)-1]
		}

		if len(matches) > 1 {
			values.Add(matches[1], x)
			continue
		}
		values.Add(k, x)
	}

	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodPost,
		Endpoint: endpoint,
		Body:     strings.NewReader(values.Encode()),
		Headers:  x.headers,
	})
	if err != nil {
		x.log.Error("cannot get servers data", "error", err)
		return nil, err
	}

	var result []string
	err = json.NewDecoder(body).Decode(&result)
	if err != nil {
		x.log.Error("cannot parse servers data", "error", err)
		return nil, err
	}

	return result, nil
}

func (x *animeSlayer) scrape(nodes ...*EpisodeNode) (*[]*EmbedNode, error) {
	var (
		err    error
		result = make([]*EmbedNode, len(nodes))
	)
	for i, v := range nodes {
		if v == nil || v.Link == nil {
			continue
		}
		if err = func() error {
			args := &client.Args{
				Proxy:    true,
				Method:   http.MethodPost,
				Endpoint: v.Link,
				Body:     strings.NewReader(v.Link.RawQuery),
				Headers:  x.headers,
			}
			v.Link.RawQuery = ""

			body, err := client.Do(context.Background(), args)
			if err != nil {
				x.log.Error("cannot get the episode data", "error", err)
				return err
			}

			var item animeSlayerData
			err = json.NewDecoder(body).Decode(&item)
			if err != nil {
				x.log.Error("cannot parse the episode data", "error", err)
				return err
			}

			for _, y := range item.Response.Data {
				for _, z := range y.EpisodeUrls {
					var sources []string
					if strings.Contains(z.EpisodeURL, "v-qs.php") {
						data, err := url.ParseQuery(z.EpisodeURL)
						if err != nil {
							x.log.Error("cannot parse the episode server", "error", err)
							continue
						}
						data.Add("inf", "")
						endpoint := &url.URL{
							Scheme: v.Link.Scheme,
							Host:   v.Link.Host,
							Path:   "/anime/public/v-qs.php",
						}

						values, err := x.extract(endpoint, data)
						if err != nil {
							if errors.Is(err, context.Canceled) {
								return err
							}
							continue
						}

						sources = append(sources, values...)
					}

					if strings.Contains(z.EpisodeURL, "api/f2") {
						data, err := url.ParseQuery(z.EpisodeURL)
						if err != nil {
							x.log.Error("cannot parse the episode server", "error", err)
							continue
						}
						data.Add("inf", "")
						endpoint := &url.URL{
							Scheme: v.Link.Scheme,
							Host:   v.Link.Host,
							Path:   "/la/public/api/fw",
						}

						values, err := x.extract(endpoint, data)
						if err != nil {
							if errors.Is(err, context.Canceled) {
								return err
							}
							continue
						}

						sources = append(sources, values...)
					}

					var videos []models.AnimeVideo
					for _, src := range sources {
						x.log.Info("found iframe source", "src", src)
						videos = append(videos, models.AnimeVideo{
							Source:   src,
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
				}
			}

			return errs.ErrNoData
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	return &result, nil
}
