package scrape

import (
	"context"
	"errors"
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

type gogoAnime struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	headers  map[string]string
	log      *slog.Logger
	ctx      context.Context
}

func GogoAnime(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := gogoAnime{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      gogoAnimeLog,
		ctx:      ctx,
		headers: map[string]string{
			"Accept":          "*/*",
			"Accept-Language": "en",
		},
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

func (x *gogoAnime) search() ([]*url.URL, error) {
	args := &client.Args{
		Proxy: true,
		Endpoint: &url.URL{
			Scheme:   "https",
			Host:     "ajax.gogocdn.net",
			Path:     "/site/loadAjaxSearch",
			RawQuery: "keyword=" + analyze.CleanQuery(x.info.Query) + "&id=-1&link_web=https://anitaku.pe/",
		},
		Method: http.MethodGet,
	}

	body, err := client.Do(x.ctx, args)
	if err != nil {
		x.log.Error("cannot query anime data", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse anime data", "error", err)
		return nil, err
	}

	var queries []*url.URL
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if strings.Contains(analyze.CleanTitle(s.Text()), x.info.Query) {
			if href, ok := s.Attr("href"); ok {
				href = strings.ReplaceAll(strings.ReplaceAll(href, `\/`, "/"), `\"`, ``)
				link, err := url.Parse(href)
				if err != nil {
					x.log.Error("cannot parse page url", "error", err)
					return
				}
				queries = append(queries, link)
			}
		}
	})
	if len(queries) == 0 {
		return nil, errs.ErrNotFound
	}

	return queries, nil
}

func (x *gogoAnime) filter(queries []*url.URL) ([]*url.URL, error) {
	var (
		err   error
		links []*url.URL
	)
	for _, v := range queries {
		if v == nil {
			continue
		}
		if err = func() error {
			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: v,
			})
			if err != nil {
				x.log.Error("cannot get query page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse query page", "error", err)
				return err
			}

			var (
				show bool
				date bool
			)
			doc.Find(".anime_info_body_bg").Find("p").Each(func(_ int, s *goquery.Selection) {
				info := strings.ToLower(s.Find("span").Text())
				if strings.Contains(info, "type") {
					anime := strings.ToLower(s.Text())
					if x.info.Type == "tv" {
						if strings.Contains(anime, "tv") || strings.Contains(anime, "serie") || strings.Contains(anime, "anime") {
							show = true
						}
					} else {
						if strings.Contains(anime, x.info.Type) {
							show = true
						}
					}
					return
				}
				if strings.Contains(info, "released") {
					if strings.Contains(strings.ToLower(s.Text()), strconv.Itoa(x.info.SD.Year)) {
						date = true
					}
					return
				}
			})
			if !(date && show) {
				return errs.ErrNotFound
			}

			var (
				id   string
				path string
			)

			doc.Find(".anime_info_episodes").Find("input").Each(func(_ int, s *goquery.Selection) {
				txt, ok := s.Attr("id")
				if ok {
					if strings.Contains(txt, "id") {
						if v, k := s.Attr("value"); k {
							id = strings.TrimSpace(v)
						}
					} else if strings.Contains(txt, "alias") {
						if v, k := s.Attr("value"); k {
							path = strings.TrimSpace(v)
						}
					}
				}
			})
			if id != "" && path != "" {
				links = append(links, &url.URL{
					Scheme:   "https",
					Host:     "ajax.gogocdn.net",
					Path:     "/ajax/load-list-episode",
					RawQuery: "ep_start=0&ep_end=9999&id=" + id + "&default_ep=0&alias=" + path,
				})
				return nil
			}
			return errs.ErrNotFound
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
	}

	if len(links) == 0 {
		return nil, errs.ErrNotFound
	}

	return links, nil
}

func (x *gogoAnime) fetch(endpoints []*url.URL) ([]*EpisodeNode, error) {
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
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: v,
				Headers:  x.headers,
			})
			if err != nil {
				x.log.Error("cannot get anime data", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse anime data", "error", err)
				return err
			}

			var track string
			doc.Find("li").Each(func(i int, s *goquery.Selection) {
				href, ok := s.Find("a").Attr("href")
				if !ok {
					return
				}

				link, err := url.Parse(strings.ReplaceAll("https://anitaku.so"+href, " ", ""))
				if err != nil {
					return
				}

				track = "sub"
				if strings.Contains(strings.ToLower(s.Find(".cate").Text()), "dub") {
					track = "dub"
				}

				if x.isMovie {
					x.log.Info("found movie episode url", "link", link.Path)
					nodes = append(nodes, &EpisodeNode{
						Number: -1,
						Type:   track,
						Link:   link,
					})
				} else {
					ep := analyze.ExtractNum(s.Find(".name").Text())
					if ep == 0 {
						return
					}

					for _, z := range x.episodes {
						if ep == z {
							x.log.Info("found episode url", "ep", ep, "link", link.Path)
							nodes = append(nodes, &EpisodeNode{
								Number: z,
								Type:   track,
								Link:   link,
							})
							break
						}
					}
				}
			})

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

func (x *gogoAnime) scrape(nodes []*EpisodeNode) (*[]*EmbedNode, error) {
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
				x.log.Error("cannot get episode page", "error", err)
				return err
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot parse episode page", "error", err)
				return err
			}

			var videos []models.AnimeVideo
			doc.Find(".anime_video_body").Find("li").Each(func(_ int, s *goquery.Selection) {
				if source, ok := s.Find("a").Attr("data-video"); ok {
					if source != "" {
						x.log.Info("found iframe source", "src", source)
						videos = append(videos, models.AnimeVideo{
							Source:   source,
							Type:     v.Type,
							Quality:  "hd",
							Language: analyze.CleanLanguage("english"),
						})
					}
				}
			})

			if len(videos) != 0 {
				result[i] = &EmbedNode{
					Number: v.Number,
					Videos: videos,
				}
				return nil
			}

			return errs.ErrNotFound
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
