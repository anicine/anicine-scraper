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

	"github.com/PuerkitoBio/goquery"
	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/internal/shared"
	"github.com/anicine/anicine-scraper/models"
)

type animeRco struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func AnimeRco(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := animeRco{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      animeRcoLog,
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

func (x *animeRco) confirm(page *url.URL) (*url.URL, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: page,
	})
	if err != nil {
		x.log.Error("cannot get the anime season page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime season page", "error", err)
		return nil, err
	}

	var found bool
	doc.Find(".media-btns").Find("a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if href, ok := s.Attr("href"); ok {
			if strings.Contains(href, strconv.Itoa(x.info.MalID)) {
				found = true
				x.log.Info("anime page url", "page", page.String())
				return false
			}
		}
		return true
	})
	if found {
		return page, nil
	}

	return nil, errs.ErrNotFound
}

func (x *animeRco) check(page *url.URL) (*url.URL, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: page,
	})
	if err != nil {
		x.log.Error("cannot check the anime page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime page", "error", err)
		return nil, err
	}

	var (
		link  *url.URL
		found bool
	)
	doc.Find(".details-side").Find(".episodes-lists").Find("li").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if href, ok := s.Find("a").Attr("href"); ok {
			page, err = url.Parse(href)
			if err != nil {
				return true
			}

			time.Sleep(100 * time.Millisecond)
			link, err = x.confirm(page)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return false
				}
			} else {
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

func (x *animeRco) search() (*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "ww3.animerco.org",
		Path:     "/",
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
	doc.Find(".row").Find(".info").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		anime := strings.ToLower(strings.TrimSpace(s.Find(".anime-type").Text()))
		if anime != x.info.Type {
			return true
		}

		year, err := analyze.ExtractYear(s.Find(".anime-aired").Text())
		if err != nil {
			return true
		}

		title := s.Find("h3").Text()
		if x.isMovie {
			if x.info.SD.Year != year {
				return true
			}

			if href, ok := s.Find("a").Attr("href"); ok {
				link, err = url.Parse(href)
				if err != nil {
					return true
				}
				found = true
				x.log.Info("anime page url", "page", link.String())
				return false
			}
		} else {
			if x.info.SD.Year < year {
				return true
			}
			if shared.TextAdvancedSimilarity(x.info.Query, analyze.CleanTitle(title)) < 40 {
				return true
			}
			if href, ok := s.Find("a").Attr("href"); ok {
				page, err := url.Parse(href)
				if err != nil {
					return true
				}

				time.Sleep(100 * time.Millisecond)
				link, err = x.check(page)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return false
					}
				} else {
					found = true
					return false
				}
			}
		}

		return true
	})
	if found {
		return link, nil
	}
	if errors.Is(err, context.Canceled) {
		return nil, err
	}

	return nil, errs.ErrNotFound
}

func (x *animeRco) fetch(endpoint *url.URL) ([]*EpisodeNode, error) {
	if x.isMovie {
		x.log.Info("found movie episode url", "link", endpoint.Path)
		return []*EpisodeNode{
			{
				Number: -1,
				Link:   endpoint,
			},
		}, nil
	}

	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
	})
	if err != nil {
		x.log.Error("cannot fetch the anime page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime page", "error", err)
		return nil, err
	}

	nodes := make([]*EpisodeNode, len(x.episodes))
	doc.Find("ul.episodes-lists").Find("li").Each(func(_ int, s *goquery.Selection) {
		if num, ok := s.Attr("data-number"); ok {
			ep, err := strconv.Atoi(strings.TrimSpace(num))
			if err != nil {
				x.log.Error("cannot parse the episode number", "error", err)
				return
			}
			for i, v := range x.episodes {
				if v == ep {
					if href, ok := s.Find("a").Attr("href"); ok {
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
		}
	})

	return nodes, nil
}

func (x *animeRco) extract(ep *EpisodeNode) (*EmbedNode, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: ep.Link,
	})
	if err != nil {
		x.log.Error("cannot get the anime episode page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime page", "error", err)
		return nil, err
	}

	videos, err := x.video(doc, ep.Link)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
	}

	downloads, err := x.download(doc)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
	}

	return &EmbedNode{
		Number:   ep.Number,
		Videos:   videos,
		Download: downloads,
	}, nil
}

func (x *animeRco) video(doc *goquery.Document, link *url.URL) ([]models.AnimeVideo, error) {
	var (
		err  error
		args = &client.Args{
			Method: http.MethodPost,
			Proxy:  true,
			Endpoint: &url.URL{
				Scheme: link.Scheme,
				Host:   link.Host,
				Path:   "/wp-admin/admin-ajax.php",
			},
			Headers: map[string]string{
				"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
				"Accept":           "*/*",
				"Accept-Language":  "en",
				"origin":           link.Scheme + "://" + link.Host,
				"sec-fetch-site":   "same-origin",
				"sec-fetch-mode":   "cors",
				"X-Requested-With": "XMLHttpRequest",
			},
		}
		videos []models.AnimeVideo
	)
	doc.Find("ul.server-list").Find("li").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		a := s.Find("a")
		if a == nil {
			return true
		}
		typ, o1 := a.Attr("data-type")
		pst, o2 := a.Attr("data-post")
		num, o3 := a.Attr("data-nume")
		if o1 && o2 && o3 {
			args.Body = strings.NewReader(fmt.Sprintf("action=player_ajax&post=%s&nume=%s&type=%s", pst, num, typ))
			time.Sleep(100 * time.Millisecond)

			body, err := client.Do(x.ctx, args)
			if err != nil {
				x.log.Error("cannot get the iframe data", "error", err)
				return !errors.Is(err, context.Canceled)
			}

			var embed map[string]any
			err = json.NewDecoder(body).Decode(&embed)
			if err != nil {
				x.log.Error("cannot parse the iframe data", "error", err)
				return true
			}

			if src, ok := embed["embed_url"].(string); ok {
				if strings.Contains(strings.ToLower(src), "<iframe") {
					html, err := goquery.NewDocumentFromReader(strings.NewReader(src))
					if err != nil {
						return true
					}
					src = ""
					source, ok := html.Find("iframe").Attr("src")
					if ok && source != "" {
						src = strings.ReplaceAll(source, `/\`, "/")
						src = strings.ReplaceAll(src, "https:", "")
						src = strings.ReplaceAll(src, "http:", "")
						src = strings.Replace(src, "//", "https://", 1)
					}
				}

				if src != "" {
					x.log.Info("found iframe source", "src", src)
					videos = append(videos, models.AnimeVideo{
						Source:   strings.TrimSpace(src),
						Type:     "sub",
						Language: analyze.CleanLanguage("arabic"),
						Quality:  "hd",
					})
				}
			}
		}
		return true
	})
	if errors.Is(err, context.Canceled) {
		return nil, err
	}

	return videos, nil
}

func (x *animeRco) download(doc *goquery.Document) ([]models.AnimeVideo, error) {
	var (
		err       error
		downloads []models.AnimeVideo
	)
	doc.Find("#download").Find("tbody").Find("tr").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if href, ok := s.Find("a").Attr("href"); ok {
			endpoint, err := url.Parse(href)
			if err != nil {
				x.log.Error("cannot parse the redirect url", "error", err)
				return true
			}

			time.Sleep(100 * time.Millisecond)

			body, err := client.Do(x.ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: endpoint,
			})
			if err != nil {
				x.log.Error("cannot follow the redirect url", "error", err)
				return !errors.Is(err, context.Canceled)
			}

			doc, err := goquery.NewDocumentFromReader(body)
			if err != nil {
				x.log.Error("cannot follow the redirect page", "error", err)
				return true
			}

			if key, ok := doc.Find("#link").Attr("data-url"); ok {
				data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key))
				if err != nil {
					x.log.Error("cannot get the decoded string of the link", "error", err)
					return true
				}

				src := string(data)
				if strings.Contains(src, "http") {

					x.log.Info("found download url", "url", src)

					downloads = append(downloads, models.AnimeVideo{
						Source:   strings.TrimSpace(src),
						Type:     "sub",
						Language: analyze.CleanLanguage("arabic"),
						Quality:  "fhd",
					})
				}
			}
		}
		return true
	})
	if errors.Is(err, context.Canceled) {
		return nil, err
	}

	return downloads, nil
}

func (x *animeRco) scrape(nodes ...*EpisodeNode) (*[]*EmbedNode, error) {
	result := make([]*EmbedNode, len(nodes))
	for i, v := range nodes {
		if v == nil || v.Link == nil {
			continue
		}
		time.Sleep(100 * time.Millisecond)

		data, err := x.extract(v)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			continue
		}

		result[i] = data
	}

	return &result, nil
}
