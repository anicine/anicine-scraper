package scrape

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/models"
)

type animeDojo struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func AnimeDojo(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := animeDojo{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      slog.Default().WithGroup("[DOJO-ANIME]"),
		ctx:      ctx,
	}

	var err error
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

	return x.scrape(nodes...)
}

func (x *animeDojo) search() ([]*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "animedojo.net",
		Path:     "/search",
		RawQuery: "keyword=" + analyze.CleanQuery(x.info.Query),
	}

	args := &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}

	var checkList []*url.URL
	for {
		select {
		case <-x.ctx.Done():
			return nil, context.Canceled
		default:
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

			doc.Find(".film_list-wrap").Find(".flw-item").Each(func(_ int, s *goquery.Selection) {
				if strings.Contains(s.Find(".fd-infor").Text(), strconv.Itoa(x.info.SD.Year)) {
					if href, ok := s.Find(".film-detail").Find("a").Attr("href"); ok {
						if href != "" {
							checkList = append(checkList, &url.URL{
								Scheme: args.Endpoint.Scheme,
								Host:   args.Endpoint.Host,
								Path:   href,
							})
						}
					}
				}
			})

			var (
				active bool
				page   *url.URL
			)
			doc.Find(".pagination").Find("li").Each(func(_ int, s *goquery.Selection) {
				if active {
					if href, ok := s.Find("a").Attr("href"); ok {
						page, err = url.Parse("https://animedojo.net/search" + href)
						if err != nil {
							return
						}
					}
				}
				active = false
				if s.HasClass("active") {
					active = true
				}
			})
			if page != nil {
				if args.Endpoint.String() != page.String() {
					args.Endpoint = page
					continue
				}
			}
			return checkList, nil
		}
	}
}

func (x *animeDojo) filter(endpoints []*url.URL) ([]*url.URL, error) {
	if endpoints == nil {
		return nil, errs.ErrNoData
	}

	check := func(endpoint *url.URL) (*url.URL, error) {
		body, err := client.Do(x.ctx, &client.Args{
			Proxy:    true,
			Method:   http.MethodGet,
			Endpoint: endpoint,
		})
		if err != nil {
			x.log.Error("cannot get the anime details page", "error", err)
			return nil, err
		}

		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			x.log.Error("cannot parse the anime details page", "error", err)
			return nil, err
		}

		if !x.isMovie {
			if !strings.Contains(doc.Find("div.anisc-info").Text(), strconv.FormatInt(int64(x.info.SD.Year), 10)) {
				return nil, nil
			}
		}

		show := strings.ToLower(doc.Find(".film-stats").Find(".item").Text())
		if show == "" {
			return nil, nil
		}

		if x.isMovie {
			if !strings.Contains(show, "movie") {
				return nil, nil
			}
		} else {
			if strings.Contains(show, "special") {
				return nil, nil
			}
		}

		if href, ok := doc.Find(".film-buttons").Find("a").Attr("href"); ok {
			if !strings.Contains(href, "http") {
				href = endpoint.Scheme + "://" + endpoint.Host + href
			}

			link, err := url.Parse(href)
			if err != nil {
				x.log.Error("cannot parse the anime watch page url", "error", err)
			}

			return link, nil
		}

		return nil, nil
	}

	var links []*url.URL
	for _, v := range endpoints {
		link, err := check(v)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
		if link != nil {
			links = append(links, link)
		}
	}

	return links, nil
}

func (x *animeDojo) fetch(endpoints []*url.URL) ([]*EpisodeNode, error) {
	if endpoints == nil {
		return nil, errs.ErrNoData
	}

	grep := func(endpoint *url.URL) ([]*EpisodeNode, error) {
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

		var nodes []*EpisodeNode
		dec := 0
		doc.Find("#episodes-page-1").Find("a").Each(func(i int, s *goquery.Selection) {
			txt, ok := s.Attr("data-number")
			if !ok {
				return
			}

			number, err := strconv.ParseFloat(txt, 32)
			if err != nil || number == 0 {
				return
			}

			if number != float64(i+1-dec) {
				dec++
				return
			}

			for _, v := range x.episodes {
				if float64(v) == (number + float64(dec)) {
					if href, ok := s.Attr("href"); ok {
						if !strings.Contains(href, "http") {
							href = endpoint.Scheme + "://" + endpoint.Host + href
						}

						link, err := url.Parse(href)
						if err != nil {
							x.log.Error("cannot parse the anime episode url", "ep", v, "error", err)
							continue
						}

						x.log.Info("found episode url", "ep", v, "link", link.Path)

						nodes = append(nodes, &EpisodeNode{
							Number: v,
							Link:   link,
						})
					}
					break
				}
			}
		})

		return nodes, nil
	}

	var nodes []*EpisodeNode
	if x.isMovie {
		for _, v := range endpoints {
			x.log.Info("found movie episode url", "link", v.Path)
			nodes = append(nodes, &EpisodeNode{
				Number: -1,
				Link:   v,
			})
		}
	} else {
		for _, v := range endpoints {
			episodes, err := grep(v)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil, err
				}
			}
			if episodes != nil {
				nodes = append(nodes, episodes...)
			}
		}
	}

	return nodes, nil
}

func (x *animeDojo) scrape(nodes ...*EpisodeNode) (*[]*EmbedNode, error) {
	if len(nodes) == 0 {
		return nil, errs.ErrNoData
	}

	var result = make([]*EmbedNode, len(nodes))
	for i, v := range nodes {
		select {
		case <-x.ctx.Done():
			return nil, context.Canceled
		default:
			if v == nil || v.Link == nil {
				continue
			}
			func() {
				body, err := client.Do(x.ctx, &client.Args{
					Proxy:    true,
					Method:   http.MethodGet,
					Endpoint: v.Link,
				})
				if err != nil {
					return
				}

				doc, err := goquery.NewDocumentFromReader(body)
				if err != nil {
					return
				}

				track := "sub"
				if strings.Contains(v.Link.Path, "-dub-") {
					track = "dub"
				}

				var videos []models.AnimeVideo
				doc.Find("#servers-content").Find(".item").Each(func(_ int, s *goquery.Selection) {
					href, ok := s.Find("a").Attr("href")
					if ok && href != "" {
						x.log.Info("found iframe source", "src", href)
						videos = append(videos, models.AnimeVideo{
							Source:   href,
							Quality:  "hd",
							Referer:  v.Link.String(),
							Type:     track,
							Language: analyze.CleanLanguage("english"),
						})
					}
				})
				result[i] = &EmbedNode{
					Number: v.Number,
					Videos: videos,
				}
			}()
		}
	}

	return &result, nil
}
