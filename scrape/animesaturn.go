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
	"github.com/anicine/anicine-scraper/models"
)

type animeSaturnSearch []struct {
	Name    string `json:"name,omitempty"`
	Link    string `json:"link,omitempty"`
	Release string `json:"release,omitempty"`
}

type animeSaturn struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func AnimeSaturn(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := animeSaturn{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      animeSaturnLog,
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

	nodes, err = x.clean(nodes)
	if nodes == nil {
		return nil, err
	}

	nodes, err = x.filter(nodes)
	if err != nil {
		return nil, err
	}

	return x.scrape(nodes)
}

func (x *animeSaturn) check(args *client.Args) (*url.URL, error) {
	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot get the anime page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime page", "error", err)
		return nil, err
	}

	var (
		found bool
		link  *url.URL
	)
	doc.Find(".container.shadow.rounded").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		s.Find("a").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if href, ok := s.Attr("href"); ok {
				if strings.Contains(strings.ToLower(href), fmt.Sprintf("myanimelist.net/anime/%d", x.info.MalID)) {
					link = args.Endpoint
					found = true
					return false
				}
			}
			return true
		})
		return !found
	})
	if found {
		return link, nil
	}

	return nil, errs.ErrNotFound
}

func (x *animeSaturn) search() ([]*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "www.animesaturn.mx",
		Path:     "/index.php",
		RawQuery: "search=1&key=" + analyze.CleanQuery(x.info.Query),
	}

	headers := map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"Accept":           "*/*",
		"Accept-Language":  "en",
		"Connection":       "Keep-Alive",
		"X-Requested-With": "XMLHttpRequest",
	}

	args := &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
		Headers:  headers,
	}

	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot query the anime", "error", err)
		return nil, err
	}

	var queries animeSaturnSearch
	err = json.NewDecoder(body).Decode(&queries)
	if err != nil {
		x.log.Error("cannot parse the search data", "error", err)
		return nil, err
	}

	var links []*url.URL
	for _, v := range queries {
		if !strings.Contains(v.Release, strconv.Itoa(x.info.SD.Year)) {
			continue
		}
		args.Endpoint, err = url.Parse(args.Endpoint.Scheme + "://" + args.Endpoint.Host + "/anime/" + v.Link)
		if err != nil {
			x.log.Error("cannot parse the anime link", "error", err)
			continue
		}

		link, err := x.check(args)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			continue
		}

		links = append(links, link)
	}

	return links, nil
}

func (x *animeSaturn) fetch(endpoints []*url.URL) ([]*EpisodeNode, error) {
	var (
		err   error
		nodes []*EpisodeNode
	)
	for _, v := range endpoints {
		if v == nil {
			continue
		}

		if err = func() error {
			args := &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: v,
			}
			body, err := client.Do(x.ctx, args)
			if err != nil {
				x.log.Error("cannot get the anime page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse the anime page", "error", err)
				return err
			}

			doc.Find(".tab-content").Find(".tab-pane").Each(func(_ int, s *goquery.Selection) {
				s.Find("a").Each(func(_ int, s *goquery.Selection) {
					if x.isMovie {
						if href, ok := s.Attr("href"); ok {
							link, err := url.Parse(href)
							if err != nil {
								x.log.Error("cannot parse the episode url", "error", err)
								return
							}

							x.log.Info("found episode url", "link", link)

							nodes = append(nodes, &EpisodeNode{
								Number: -1,
								Link:   link,
							})
						}
					} else {
						if ep := analyze.ExtractNum(s.Text()); ep != 0 {
							for _, v := range x.episodes {
								if v == ep {
									if href, ok := s.Attr("href"); ok {
										link, err := url.Parse(href)
										if err != nil {
											x.log.Error("cannot parse the episode url", "error", err)
											return
										}

										x.log.Info("found episode url", "ep", v, "link", link)

										nodes = append(nodes, &EpisodeNode{
											Number: v,
											Link:   link,
										})
										break
									}
								}
							}
						}
					}
				})
			})
			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	return nodes, err
}

func (x *animeSaturn) clean(nodes []*EpisodeNode) ([]*EpisodeNode, error) {
	var (
		err    error
		result []*EpisodeNode
	)
	for _, v := range nodes {
		if v == nil || v.Link == nil {
			continue
		}
		if err = func() error {
			time.Sleep(100 * time.Millisecond)
			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: v.Link,
			})
			if err != nil {
				x.log.Error("cannot get episode page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse episode page", "error", err)
				return err
			}

			card := doc.Find(".card-body")
			if card == nil {
				return errs.ErrNotFound
			}

			href, ok := card.Find("a").Attr("href")
			if !ok {
				return errs.ErrNotFound
			}

			link, err := url.Parse(href)
			if err != nil {
				return err
			}

			track := "sub"
			title := card.Find("h3").Text()
			if !strings.Contains(strings.ToLower(title), " sub ita") {
				track = "dub"
			}

			result = append(result, &EpisodeNode{
				Number: v.Number,
				Type:   track,
				Link:   link,
			})

			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	return result, nil
}

func (x *animeSaturn) filter(nodes []*EpisodeNode) ([]*EpisodeNode, error) {
	var (
		err    error
		result []*EpisodeNode
	)
	for _, v := range nodes {
		if v == nil || v.Link == nil {
			continue
		}
		if err = func() error {
			time.Sleep(100 * time.Millisecond)

			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: v.Link,
			})
			if err != nil {
				x.log.Error("cannot get the servers page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse the servers page", "error", err)
				return err
			}

			doc.Find("div.dropdown-menu").Find("a").Each(func(_ int, s *goquery.Selection) {
				href, ok := s.Attr("href")
				if !ok {
					return
				}

				link, err := url.Parse(href)
				if err != nil {
					return
				}
				if x.isMovie {
					x.log.Info("found movie episode url", "link", link.Path)
				} else {
					x.log.Info("found episode url", "ep", v, "link", link.Path)
				}

				result = append(result, &EpisodeNode{
					Number: v.Number,
					Type:   v.Type,
					Link:   link,
				})
			})

			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	return result, nil
}

func (x *animeSaturn) scrape(nodes []*EpisodeNode) (*[]*EmbedNode, error) {
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
				Endpoint: v.Link,
			})
			if err != nil {
				x.log.Error("cannot get the iframe page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse the iframe page", "error", err)
				return err
			}

			if src, ok := doc.Find(".embed-container").Find("iframe").Attr("src"); ok {
				if src != "" {
					x.log.Info("found iframe source", "src", src)

					result[i] = &EmbedNode{
						Number: v.Number,
						Videos: []models.AnimeVideo{
							{
								Source:   src,
								Type:     v.Type,
								Quality:  "hd",
								Language: analyze.CleanLanguage("italian"),
							},
						},
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
