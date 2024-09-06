package scrape

import (
	"bytes"
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
	"github.com/anicine/anicine-scraper/models"
)

type animeUnitySearch struct {
	Records []struct {
		ID    int    `json:"id,omitempty"`
		Date  string `json:"date,omitempty"`
		Type  string `json:"type,omitempty"`
		Slug  string `json:"slug,omitempty"`
		MalID int    `json:"mal_id,omitempty"`
	} `json:"records,omitempty"`
}

type animeUnityFrame struct {
	ID     int    `json:"id,omitempty"`
	Number string `json:"number,omitempty"`
	ScwsID int    `json:"scws_id,omitempty"`
}

type animeUnity struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	headers  map[string]string
	log      *slog.Logger
	ctx      context.Context
}

func AnimeUnity(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := animeUnity{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      animeUnityLog,
		ctx:      ctx,
	}

	err := x.refresh()
	if err != nil {
		return nil, err
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
func (x *animeUnity) refresh() error {
	endpoint := &url.URL{
		Scheme: "https",
		Host:   "animeunity.to",
		Path:   "/",
	}
	args := &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}

	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot refresh the cookies", "error", err)
		return err
	}

	var (
		xr string
		cr string
		sr string
	)

	for _, v := range args.Cookies() {
		if strings.Contains(strings.ToUpper(v.Name), "XSRF-TOKEN") {
			xr = v.Value
		}

		if strings.Contains(strings.ToLower(v.Name), "animeunity_session") {
			sr = v.Value
		}
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse anime unity page", "error", err)
		return err
	}

	doc.Find("head").Find("meta").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		name, ok := s.Attr("name")
		if ok {
			if strings.Contains(strings.ToLower(name), "csrf-token") {
				content, ok := s.Attr("content")
				if ok {
					cr = content
					if cr != "" {
						return false
					}
				}
			}
		}
		return true
	})

	if xr == "" || cr == "" || sr == "" {
		x.log.Error("cannot find the correct cookies", "error", err)
		return errs.ErrNoData
	}

	animeUnityLog.Info("cookies was renewed successfully")

	x.headers = map[string]string{
		"Content-Type":     "application/json; charset=UTF-8",
		"Accept":           "*/*",
		"Accept-Language":  "en-US,en;q=0.9",
		"X-Requested-With": "XMLHttpRequest",
		"Referer":          endpoint.Scheme + "://" + endpoint.Host + "/",
		"Origin":           endpoint.Scheme + "://" + endpoint.Host,
		"Cookie":           strconv.Quote(fmt.Sprintf("X-XSRF-TOKEN=%s; animeunity_session=%s", xr, sr)),
		"X-CSRF-TOKEN":     cr,
		"X-XSRF-TOKEN":     xr,
	}

	return nil
}

func (x *animeUnity) search() ([]*url.URL, error) {
	args := &client.Args{
		Proxy: true,
		Endpoint: &url.URL{
			Scheme: "https",
			Host:   "www.animeunity.to",
			Path:   "/livesearch",
		},
		Headers: x.headers,
		Method:  http.MethodPost,
		Body:    strings.NewReader(fmt.Sprintf(`{"title": "%s"}`, strings.ReplaceAll(x.info.Query, "-", " "))),
	}

	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot query anime data", "error", err)
		return nil, err
	}

	var queries animeUnitySearch
	err = json.NewDecoder(body).Decode(&queries)
	if err != nil {
		x.log.Error("cannot parse anime data", "error", err)
		return nil, err
	}

	var links []*url.URL
	for _, v := range queries.Records {
		if v.MalID == x.info.MalID {
			links = append(links, &url.URL{
				Scheme: "https",
				Host:   args.Endpoint.Host,
				Path:   "/anime/" + strconv.Itoa(v.ID) + "-" + v.Slug + "/",
			})
		}
	}

	if len(links) == 0 {
		return nil, errs.ErrNotFound
	}

	return links, nil
}

func (x *animeUnity) fetch(endpoints []*url.URL) ([]*EpisodeNode, error) {
	var (
		err   error
		nodes []*EpisodeNode
	)
	for _, v := range endpoints {
		if v == nil {
			continue
		}
		if err = func() error {
			time.Sleep(100 * time.Millisecond)
			body, err := client.Do(x.ctx, &client.Args{
				Method:   http.MethodGet,
				Endpoint: v,
				Headers:  x.headers,
			})
			if err != nil {
				x.log.Error("cannot get anime page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse anime page", "error", err)
				return err
			}

			episodes, ok := doc.Find("video-player ").Attr("episodes")
			if !ok {
				x.log.Error("cannot find anime episodes", "error", err)
				return errs.ErrNotFound
			}
			episodes = strings.ReplaceAll(episodes, `&quot;`, `"`)

			var code []animeUnityFrame

			err = json.Unmarshal([]byte(episodes), &code)
			if err != nil {
				x.log.Error("cannot parse anime episodes", "error", err)
				return err
			}

			track := "sub"
			if strings.Contains(v.String(), "-ita") {
				track = "dub"
			}

			for _, y := range code {
				if !x.isMovie {
					for _, z := range x.episodes {
						if y.Number == strconv.Itoa(z) {
							node := &EpisodeNode{
								Number: z,
								Type:   track,
								Link: &url.URL{
									Scheme: v.Scheme,
									Host:   v.Host,
									Path:   "/embed-url/" + strconv.Itoa(y.ID),
								},
							}
							x.log.Info("found episode url", "ep", y.Number, "link", node.Link.Path)
							nodes = append(nodes, node)
							break
						}
					}
				} else {
					node := &EpisodeNode{
						Number: -1,
						Type:   track,
						Link: &url.URL{
							Scheme: v.Scheme,
							Host:   v.Host,
							Path:   "/embed-url/" + strconv.Itoa(y.ID),
						},
					}
					x.log.Info("found movie episode url", "link", node.Link.Path)
					nodes = append(nodes, node)
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

func (x *animeUnity) scrape(nodes []*EpisodeNode) (*[]*EmbedNode, error) {
	var (
		err    error
		result = make([]*EmbedNode, len(nodes))
	)
	for i, v := range nodes {
		if v == nil || v.Link == nil {
			continue
		}
		if err = func() error {
			time.Sleep(100 * time.Millisecond)
			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Headers:  x.headers,
				Endpoint: v.Link,
			})
			if err != nil {
				x.log.Error("cannot get iframe data", "error", err)
				return err
			}

			buf := new(bytes.Buffer)
			buf.ReadFrom(body)

			link, err := url.Parse(strings.TrimSpace(buf.String()))
			if err != nil {
				x.log.Error("cannot parse iframe url", "error", err)
				return err
			}

			x.log.Info("found iframe source", "src", link.String())

			result[i] = &EmbedNode{
				Number: v.Number,
				Videos: []models.AnimeVideo{
					{
						Source:   link.Scheme + "://" + link.Host + link.Path,
						Referer:  v.Link.Scheme + "://" + v.Link.Host,
						Type:     v.Type,
						Quality:  "fhd",
						Language: analyze.CleanLanguage("italian"),
					},
				},
			}

			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	return &result, nil
}
