package scrape

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/internal/shared"
	"github.com/anicine/anicine-scraper/models"
)

type jkAnimeFrame struct {
	Remote string `json:"remote,omitempty"`
}

type jkAnimeSearch struct {
	Animes []struct {
		ID    string `json:"id,omitempty"`
		Slug  string `json:"slug,omitempty"`
		Title string `json:"title,omitempty"`
		Type  string `json:"type,omitempty"`
	} `json:"animes,omitempty"`
}

// var jkExp = regexp.MustCompile(`(?m)var servers = \[{(.*)}\];`)
var jkExp = regexp.MustCompile(`var servers = \[\{(.*?)\}\];`)

type jkAnime struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func JKAnime(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := jkAnime{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      jkAnimeLog,
		ctx:      ctx,
	}

	queries, err := x.search()
	if err != nil {
		return nil, err
	}

	pages, err := x.filter(queries)
	if err != nil {
		return nil, err
	}

	nodes, err := x.fetch(pages)
	if err != nil {
		return nil, err
	}

	return x.scrape(nodes)
}

func (x *jkAnime) search() ([]string, error) {
	args := &client.Args{
		Proxy:  true,
		Method: http.MethodGet,
		Endpoint: &url.URL{
			Scheme:   "https",
			Host:     "jkanime.net",
			Path:     "/ajax/ajax_search/",
			RawQuery: "q=" + x.info.Query,
		},
		Headers: map[string]string{
			"Referer":          "https://jkanime.net/",
			"X-Requested-With": "XMLHttpRequest",
			"sec-fetch-site":   "same-origin",
		},
	}
	x.log.Info(args.Endpoint.String())

	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot query anime data", "error", err)
		return nil, err
	}

	var search jkAnimeSearch
	err = json.NewDecoder(body).Decode(&search)
	if err != nil {
		return nil, errors.New("failed to parse search results")
	}

	if len(search.Animes) == 0 {
		return nil, errs.ErrNotFound
	}
	var paths []string
	for _, v := range search.Animes {
		if x.isMovie {
			if !strings.Contains(strings.ToLower(v.Type), "movie") {
				continue
			}

		} else {
			if !strings.Contains(strings.ToLower(v.Type), "tv") {
				continue
			}
		}

		if shared.TextAdvancedSimilarity(analyze.CleanTitle(v.Title), analyze.CleanTitle(x.info.Title)) > 50 {
			slug := strings.TrimSpace(v.Slug)
			if slug != "" {
				paths = append(paths, slug)
			}
		}
	}

	if len(paths) == 0 {
		return nil, errs.ErrNotFound
	}

	return paths, nil
}

func (x *jkAnime) filter(slugs []string) (*url.URL, error) {
	var (
		err  error
		link *url.URL
		args = &client.Args{
			Proxy:  true,
			Method: http.MethodGet,
			Endpoint: &url.URL{
				Scheme: "https",
				Host:   "jkanime.net",
			},
		}
	)
	for _, v := range slugs {
		if err = func() error {
			args.Endpoint.Path = "/" + v
			body, err := client.Do(x.ctx, args)
			if err != nil {
				x.log.Error("cannot check the anime page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse the anime page", "error", err)
				return err
			}

			doc.Find("div.anime__details__widget").EachWithBreak(func(i int, s *goquery.Selection) bool {
				if strings.Contains(strings.ToLower(s.Find("span").Text()), "emitido") {
					year := strings.Contains(s.Text(), "de "+strconv.Itoa(x.info.SD.Year))
					day := strings.Contains(s.Text(), " "+strconv.Itoa(x.info.SD.Day)+" ")
					if year && day {
						link = args.Endpoint
						return false
					}
				}
				return true
			})

			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
		if link != nil {
			return link, nil
		}
	}

	return nil, errs.ErrNotFound
}

func (x *jkAnime) fetch(page *url.URL) ([]*EpisodeNode, error) {
	if x.isMovie {
		path := page.Path + "/1"
		x.log.Info("found movie episode url", "link", path)
		return []*EpisodeNode{
			{
				Number: -1,
				Link: &url.URL{
					Scheme: page.Scheme,
					Host:   page.Host,
					Path:   path,
				},
			},
		}, nil
	}

	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: page,
	})
	if err != nil {
		x.log.Error("cannot fetch anime page data", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse anime page data", "error", err)
		return nil, err
	}

	txt := doc.Find(".anime__pagination").Find("a").Last().Text()
	if strings.TrimSpace(txt) == "" {
		return nil, errs.ErrNotFound
	}

	pages := strings.Split(txt, "-")
	if len(pages) == 0 {
		return nil, errs.ErrNotFound
	}

	last, err := strconv.Atoi(pages[len(pages)-1])
	if err != nil {
		return nil, err
	}
	if last == 0 {
		last, err = strconv.Atoi(pages[0])
		if err != nil {
			return nil, err
		}
	}

	if last == 0 {
		return nil, errs.ErrNotFound
	}

	var nodes []*EpisodeNode
	for _, v := range x.episodes {
		for z := range last {
			if v == z {
				path := page.Path + "/" + strconv.Itoa(v) + "/"
				x.log.Info("found episode url", "ep", v, "link", path)
				nodes = append(nodes, &EpisodeNode{
					Number: v,
					Link: &url.URL{
						Scheme: page.Scheme,
						Host:   page.Host,
						Path:   page.Path + "/" + strconv.Itoa(v) + "/",
					},
				})
			}
		}
	}

	return nodes, nil
}

func (x *jkAnime) scrape(nodes []*EpisodeNode) (*[]*EmbedNode, error) {
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
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				return err
			}

			match := jkExp.FindStringSubmatch(doc.Text())
			if len(match) > 1 {
				txt := "[{" + match[1] + "}]"
				var iframe []jkAnimeFrame
				err = json.Unmarshal([]byte(txt), &iframe)
				if err != nil {
					x.log.Error("cannot parse the iframe", "error", err)
					return err
				}

				var videos []models.AnimeVideo
				for _, z := range iframe {
					code, err := base64.StdEncoding.DecodeString(z.Remote)
					if err != nil || len(code) == 0 {
						continue
					}

					x.log.Info("found iframe source", "src", string(code))

					videos = append(videos, models.AnimeVideo{
						Source:   string(code),
						Type:     "sub",
						Quality:  "fhd",
						Language: analyze.CleanLanguage("es"),
					})
				}
				if len(videos) == 0 {
					return errs.ErrNotFound
				}

				result[i] = &EmbedNode{
					Number: v.Number,
					Videos: videos,
				}
			}

			return errs.ErrNotFound
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	return &result, nil
}
