package scrape

import (
	"context"
	"errors"
	"fmt"
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

var (
	shaRExp  = regexp.MustCompile(`window\.location\s*=\s*"([^"]+)"`)
	shaEpExp = regexp.MustCompile(`حلقة\s*(\d+)`)
)

type shahidAnime struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func ShahidAnime(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := shahidAnime{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      shahidAnimeLog,
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

	return x.scrape(nodes)
}

func (x *shahidAnime) search() (*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "shahiid-anime.net",
		Path:     "/",
		RawQuery: "s=" + analyze.CleanQuery(x.info.Title),
	}

	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
	})
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
	doc.Find("#main").Find(".one-poster").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		title := s.Find("h2").Text()
		if x.isMovie {
			if !strings.Contains(title, "فيلم") {
				return true
			}
			if shared.TextAdvancedSimilarity(analyze.CleanTitle(analyze.ExtractEngChars(title)), x.info.Title) < 90 {
				return true
			}
		}

		href, ok := s.Find("a").Attr("href")
		if !ok {
			return true
		}

		link, err = url.Parse(href)
		if err != nil {
			return true
		}

		if !x.isMovie {
			if strings.Contains(href, "/seasons/") {
				if analyze.CleanTitle(analyze.ExtractEngChars(title)) == x.info.Title {
					found = true
					return false
				}
				return true
			}
		}

		link, err = x.check(link)
		if err != nil {
			return true
		}
		if link != nil {
			found = true
			return false
		}

		return true
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
	}
	if found {
		return link, nil
	}

	return nil, errs.ErrNotFound
}

func (x *shahidAnime) check(endpoint *url.URL) (*url.URL, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
	})
	if err != nil {
		x.log.Error("cannot check the anime", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the anime page", "error", err)
		return nil, err
	}

	var found bool
	doc.Find(".container").Find("span").EachWithBreak(func(i int, s *goquery.Selection) bool {
		txt := s.Text()
		if strings.Contains(txt, "سنة") {
			if strings.Contains(txt, strconv.Itoa(x.info.SD.Year)) {
				found = true
				return false
			}
		}
		return true
	})
	if !found {
		return nil, errs.ErrNotFound
	}
	if x.isMovie {
		return endpoint, nil
	}

	found = false
	var link *url.URL
	doc.Find(".page-box").Find(".one-poster").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, ok := s.Find("a").Attr("href")
		if !ok {
			return true
		}

		link, err = url.Parse(href)
		if err != nil {
			return true
		}

		if shared.TextAdvancedSimilarity(analyze.CleanTitle(analyze.ExtractEngChars(s.Find("h2").Text())), x.info.Title) > 90 {
			found = true
			return false
		}

		return true
	})
	if found {
		return link, nil
	}

	return nil, errs.ErrNotFound
}

func (x *shahidAnime) fetch(endpoint *url.URL) ([]*EpisodeNode, error) {
	if x.isMovie {
		return []*EpisodeNode{
			{
				Number: -1,
				Link:   endpoint,
			},
		}, nil
	}

	link, err := x.redirect(endpoint)
	if err != nil {
		x.log.Error("cannot get the redirect link", "error", err)
		return nil, err
	}

	endpoint, err = url.Parse(link)
	if err != nil {
		x.log.Error("cannot parse the redirect link", "error", err)
		return nil, err
	}

	var nodes []*EpisodeNode
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Endpoint: endpoint,
		Method:   http.MethodGet,
	})
	if err != nil {
		x.log.Error("cannot get anime data", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse anime data", "error", err)
		return nil, err
	}

	doc.Find(".page-box").Find("nav").Each(func(_ int, s *goquery.Selection) {
		ep := s.Find("p").Text()
		for _, v := range x.episodes {
			if x.episode(ep) == v {
				if href, ok := s.Find("p").Find("a").Attr("href"); ok {
					link, err := url.Parse(href)
					if err != nil {
						x.log.Error("cannot parse the episode link", "error", err)
						return
					}
					nodes = append(nodes, &EpisodeNode{
						Number: v,
						Link:   link,
					})
				}
				return
			}
		}
	})

	if len(nodes) == 0 {
		return nil, errs.ErrNotFound
	}

	return nodes, nil
}

func (x *shahidAnime) episode(text string) int {
	match := shaEpExp.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}

	ep, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}

	return ep
}

func (x *shahidAnime) redirect(endpoint *url.URL) (string, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Endpoint: endpoint,
		Method:   http.MethodGet,
	})
	if err != nil {
		x.log.Error("cannot get anime data page", "error", err)
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse anime data page", "error", err)
		return "", err
	}

	html, err := doc.Html()
	if err != nil {
		x.log.Error("cannot show page html", "error", err)
		return "", err
	}

	match := shaRExp.FindStringSubmatch(html)
	if len(match) < 2 {
		return "", errs.ErrNotFound
	}

	return match[1], nil
}

func (x *shahidAnime) movie(link *url.URL) (*EmbedNode, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: link,
	})
	if err != nil {
		x.log.Error("cannot get the movie page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the movie page", "error", err)
		return nil, err
	}

	var videos []models.AnimeVideo
	doc.Find(".movies-servers").Find("li").Each(func(_ int, s *goquery.Selection) {
		frame, ok := s.Find("a").Attr("data-frameserver")
		if !ok {
			return
		}

		html, err := goquery.NewDocumentFromReader(strings.NewReader(frame))
		if err != nil {
			return
		}

		source, ok := html.Find("iframe").Attr("src")
		if !ok {
			return
		}

		if !strings.Contains(source, "http") {
			source = "http" + source
		}

		x.log.Info("found iframe source", "src", source)

		videos = append(videos, models.AnimeVideo{
			Source:   source,
			Type:     "sub",
			Quality:  "hd",
			Language: analyze.CleanLanguage("arabic"),
		})

	})

	var downloads []models.AnimeVideo
	doc.Find(".bar-download-movie").Find("a").Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			if href != "" {
				x.log.Info("found download url", "url", href)
				downloads = append(downloads, models.AnimeVideo{
					Source:   href,
					Type:     "sub",
					Quality:  "hd",
					Language: analyze.CleanLanguage("arabic"),
				})
			}
		}
	})

	return &EmbedNode{
		Number:   -1,
		Videos:   videos,
		Download: downloads,
	}, nil
}

func (x *shahidAnime) tv(ep *EpisodeNode) (*EmbedNode, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: ep.Link,
	})
	if err != nil {
		x.log.Error("cannot get the episode page", "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse the episode page", "error", err)
		return nil, err
	}

	var videos []models.AnimeVideo
	doc.Find(".movies-servers").Find("li").Each(func(_ int, s *goquery.Selection) {
		a := s.Find("a")
		if a == nil {
			return
		}

		post, o1 := a.Attr("data-post")
		frame, o2 := a.Attr("data-frameserver")
		server, o3 := a.Attr("data-serv")
		if !(o1 && o2 && o3) {
			return
		}

		link := &url.URL{
			Scheme:   ep.Link.Scheme,
			Host:     ep.Link.Host,
			Path:     "/wp-admin/admin-ajax.php",
			RawQuery: fmt.Sprintf("action=codecanal_ajax_request&post=%s&frameserver=%s&serv=%s", post, frame, server),
		}

		source, err := x.video(link)
		if err != nil {
			return
		}

		x.log.Info("found iframe source", "src", source)

		videos = append(videos, models.AnimeVideo{
			Source:   source,
			Type:     "sub",
			Quality:  "hd",
			Language: analyze.CleanLanguage("arabic"),
		})
	})

	node := &EmbedNode{
		Number: ep.Number,
		Videos: videos,
	}

	href, ok := doc.Find(".btn-download-eps").Attr("href")
	if !ok {
		return node, nil
	}

	link, err := url.Parse(href)
	if err != nil {
		return node, nil
	}

	download := x.download(link)
	if download != nil {
		node.Download = download
	}

	return node, nil
}

func (x *shahidAnime) scrape(nodes []*EpisodeNode) (*[]*EmbedNode, error) {
	var (
		err    error
		result = make([]*EmbedNode, len(nodes))
	)
	for i, v := range nodes {
		select {
		case <-x.ctx.Done():
			return nil, context.Canceled
		default:
			if v == nil || v.Link == nil {
				continue
			}
			if x.isMovie {
				result[i], err = x.movie(v.Link)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return nil, err
					}
				}
			} else {
				result[i], err = x.tv(v)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return nil, err
					}
				}
			}
		}
	}

	return &result, nil
}

func (x *shahidAnime) video(endpoint *url.URL) (string, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Endpoint: endpoint,
		Method:   http.MethodGet,
		Headers: map[string]string{
			"sec-fetch-site":   "same-origin",
			"x-requested-with": "XMLHttpRequest",
		},
	})
	if err != nil {
		x.log.Error("cannot get iframe data", "error", err)
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse iframe data", "error", err)
		return "", err
	}

	if src, ok := doc.Find("iframe").Attr("src"); ok {
		if !strings.Contains(src, "http") {
			src = "http" + src
		}
		return src, nil
	}

	return "", errs.ErrNotFound
}

func (x *shahidAnime) download(endpoint *url.URL) []models.AnimeVideo {
	body, err := client.Do(x.ctx, &client.Args{
		Proxy:    true,
		Endpoint: endpoint,
		Method:   http.MethodGet,
		Headers: map[string]string{
			"sec-fetch-site":   "same-origin",
			"x-requested-with": "XMLHttpRequest",
		},
	})
	if err != nil {
		x.log.Error("cannot get download page", "error", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		x.log.Error("cannot parse download page", "error", err)
		return nil
	}

	var downloads []models.AnimeVideo
	doc.Find("#metadownloadlink").Find("a").Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			if href != "" {
				x.log.Info("found download url", "url", href)
				downloads = append(downloads, models.AnimeVideo{
					Source:   href,
					Type:     "sub",
					Quality:  "hd",
					Language: analyze.CleanLanguage("arabic"),
				})
			}
		}
	})
	if downloads != nil {
		return downloads
	}

	return nil
}
