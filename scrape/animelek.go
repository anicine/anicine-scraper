package scrape

import (
	"context"
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

type animeLek struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func AnimeLek(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := animeLek{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      animeLekLog,
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

func (x *animeLek) check(link *url.URL) (bool, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Method:   http.MethodGet,
		Endpoint: link,
	})
	if err != nil {
		return false, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return false, err
	}

	var found bool
	doc.Find(".anime-container-infos").Find(".full-list-info").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if href, ok := s.Find("a").Attr("href"); ok {
			if strings.Contains(href, strconv.Itoa(x.info.MalID)) {
				found = true
				return false
			}
		}
		return true
	})

	return found, nil
}

func (x *animeLek) search() (*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "animelek.xyz",
		Path:     "/search/",
		RawQuery: "s=" + analyze.CleanQuery(x.info.Query),
	}

	args := &client.Args{
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

	doc.Find(".anime-list-content").Find(".anime-card-title").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		year := s.Find("h4").Text()
		if year == "" {
			return true
		}

		if strings.Contains(year, strconv.Itoa(x.info.SD.Year)) {
			if href, ok := s.Find("a").Attr("href"); ok {
				link, err = url.Parse(href)
				if err != nil {
					return true
				}
				time.Sleep(100 * time.Millisecond)

				found, err = x.check(link)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return false
					}
				}
				if found {
					return false
				}
			}
		}

		return true
	})
	if errors.Is(err, context.Canceled) {
		return nil, err
	}

	if found {
		return link, nil
	}

	return nil, errs.ErrNotFound
}

func (x *animeLek) fetch(endpoint *url.URL) ([]*EpisodeNode, error) {
	body, err := client.Do(x.ctx, &client.Args{
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
		if href, ok := doc.Find(".episodes-card-container").Find("a").Attr("href"); ok {
			link, err := url.Parse(href)
			if err != nil {
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
		doc.Find(".episodes-card-container").Each(func(_ int, s *goquery.Selection) {
			a := s.Find(".ep-card-anime-title-detail").Find("a")
			if a == nil {
				return
			}

			for i, v := range x.episodes {
				if strings.ReplaceAll(a.Text(), " ", "") == fmt.Sprintf("الحلقة%d", v) {
					if href, ok := a.Attr("href"); ok {
						link, err := url.Parse(href)
						if err != nil {
							x.log.Error("cannot parse the episode url", "ep", v, "error", err)
							return
						}

						x.log.Info("found episode url", "ep", v, "link", link.Path)

						nodes[i] = &EpisodeNode{
							Number: v,
							Link:   link,
						}
						break
					}
				}
			}
		})

		return nodes, nil
	}

	return nil, errs.ErrNotFound
}

func (x *animeLek) scrape(nodes ...*EpisodeNode) (*[]*EmbedNode, error) {
	result := make([]*EmbedNode, len(nodes))
	for i, v := range nodes {
		select {
		case <-x.ctx.Done():
			return nil, context.Canceled
		default:
			if v == nil || v.Link == nil {
				continue
			}
			func() {
				time.Sleep(100 * time.Millisecond)
				body, err := client.Do(x.ctx, &client.Args{
					Method:   http.MethodGet,
					Endpoint: v.Link,
				})
				if err != nil {
					x.log.Error("cannot load the episode page", "error", err)
					return
				}

				doc, err := goquery.NewDocumentFromReader(body)
				if err != nil {
					x.log.Error("cannot parse the episode page", "error", err)
					return
				}

				tab := doc.Find(".tab-content")
				if tab == nil {
					return
				}

				var (
					videos    []models.AnimeVideo
					downloads []models.AnimeVideo
				)

				tab.Find("#watch").Find("li").Each(func(_ int, s *goquery.Selection) {
					if src, ok := s.Find("a").Attr("data-ep-url"); ok {
						if src != "" {
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

							x.log.Info("found iframe source", "src", src)

							videos = append(videos, models.AnimeVideo{
								Source:   src,
								Type:     "sub",
								Quality:  quality,
								Language: analyze.CleanLanguage("arabic"),
							})
						}
					}
				})

				tab.Find("#downloads").Find("li").Each(func(_ int, s *goquery.Selection) {
					if src, ok := s.Find("a").Attr("href"); ok {
						if src != "" {
							quality := strings.ToLower(s.Find("small").Text())
							if strings.Contains(quality, "hd") {
								quality = "hd"
							}
							if strings.Contains(quality, "fhd") {
								quality = "fhd"
							}
							if strings.Contains(quality, "sd") || strings.Contains(quality, "ld") {
								quality = "sd"
							}

							x.log.Info("found download url", "url", src)

							downloads = append(downloads, models.AnimeVideo{
								Source:   src,
								Type:     "sub",
								Quality:  quality,
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
			}()
		}
	}

	return &result, nil
}
