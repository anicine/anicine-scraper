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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/models"
)

var (
	witEPExp = regexp.MustCompile(`var processedEpisodeData\s*=\s*'([^']+)'`).FindStringSubmatch
	witDlExp = regexp.MustCompile(`_d\s*=\s*([^;]*)`).FindStringSubmatch
	witVdExp = regexp.MustCompile(`sU\s*=\s*([^;]*)`).FindStringSubmatch
)

type witAnimeEpisode struct {
	Number string `json:"number,omitempty"`
	Type   string `json:"type,omitempty"`
	URL    string `json:"url,omitempty"`
}

type witAnime struct {
	info     *models.AnimeInfo
	episodes []int
	isMovie  bool
	log      *slog.Logger
	ctx      context.Context
}

func WitAnime(ctx context.Context, info *models.AnimeInfo, episodes []int) (*[]*EmbedNode, error) {
	x := witAnime{
		info:     info,
		episodes: episodes,
		isMovie:  info.Type == "movie",
		log:      witAnimeLog,
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

func (x *witAnime) check(args *client.Args) error {
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

	if href, ok := doc.Find("a.anime-mal").Attr("href"); ok {
		if strings.Contains(href, fmt.Sprint(x.info.MalID)) {
			x.log.Info("found anime page", "url", args.Endpoint.String())
			return nil
		}
	}

	return errs.ErrNotFound
}

func (x *witAnime) search() (*url.URL, error) {
	endpoint := &url.URL{
		Scheme:   "https",
		Host:     "witanime.one",
		Path:     "/",
		RawQuery: "search_param=animes&s=" + analyze.CleanQuery(x.info.Query),
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

	var links []string
	doc.Find(".anime-card-details").Each(func(_ int, s *goquery.Selection) {
		if !strings.Contains(strings.ToLower(s.Find(".anime-card-type").Text()), x.info.Type) {
			return
		}

		if title := s.Find(".anime-card-title"); title != nil {
			if strings.Contains(analyze.CleanTitle(title.Text()), x.info.Query) {
				a := title.Find("a")
				href, ok := a.Attr("href")
				if ok {
					links = append(links, href)
				}
			}
		}
	})

	if len(links) == 0 {
		x.log.Error("there is no results for this anime")
		return nil, errs.ErrNotFound
	}

	args := &client.Args{
		Method: http.MethodGet,
	}

	for _, v := range links {
		args.Endpoint, err = url.Parse(v)
		if err != nil {
			x.log.Error("cannot parse the anime page url", "error", err)
			continue
		}

		if err = x.check(args); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			continue
		}

		return args.Endpoint, nil
	}

	return nil, errs.ErrNotFound
}

func (x *witAnime) decode(data string) ([]witAnimeEpisode, error) {
	parts := strings.Split(data, ".")
	if len(parts) != 2 {
		return nil, errs.ErrBadData
	}

	decodedPart1, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	decodedPart2, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	decodedData := make([]byte, len(decodedPart1))
	for i := 0; i < len(decodedPart1); i++ {
		decodedData[i] = decodedPart1[i] ^ decodedPart2[i%len(decodedPart2)]
	}

	cleanData := strings.ReplaceAll(string(decodedData), `\"`, `"`)

	var decodedJSON []witAnimeEpisode
	err = json.Unmarshal([]byte(cleanData), &decodedJSON)
	if err != nil {
		return nil, err
	}

	return decodedJSON, nil
}

func (x *witAnime) fetch(endpoint *url.URL) ([]*EpisodeNode, error) {
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

	var txt string
	if matches := witEPExp(doc.Text()); len(matches) > 1 {
		txt = matches[1]
	} else {
		return nil, errs.ErrNotFound
	}

	data, err := x.decode(txt)
	if err != nil {
		return nil, err
	}

	if x.isMovie {
		for _, v := range data {
			if strings.Contains(v.Type, "فلم") {
				link, err := url.Parse(strings.ReplaceAll(v.URL, `\\/`, `/`))
				if err != nil {
					continue
				}

				x.log.Info("found movie episode url", "link", link.Path)
				return []*EpisodeNode{
					{
						Number: -1,
						Link:   link,
					},
				}, nil
			}
		}
		return nil, errs.ErrNotFound
	} else {
		var nodes []*EpisodeNode
		for _, v := range data {
			for _, z := range x.episodes {
				num, err := strconv.Atoi(v.Number)
				if err != nil {
					continue
				}
				if num == z {
					tv := strings.Contains(v.Type, "حلقة")
					ova := strings.Contains(v.Type, "أوفا")
					if tv || ova {
						link, err := url.Parse(strings.ReplaceAll(v.URL, `\\/`, `/`))
						if err != nil {
							continue
						}

						x.log.Info("found episode url", "ep", z, "link", link.Path)
						nodes = append(nodes, &EpisodeNode{
							Number: z,
							Link:   link,
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
}

func (x *witAnime) video(text string) []models.AnimeVideo {
	matches := witVdExp(text)
	if len(matches) < 1 {
		return nil
	}

	var data []string
	txt := strings.ReplaceAll(matches[1], `\\/`, `/`)
	err := json.Unmarshal([]byte(txt), &data)
	if err != nil {
		return nil
	}
	if len(data) == 0 {
		return nil
	}

	var (
		source   string
		language = analyze.CleanLanguage("arabic")
		result   []models.AnimeVideo
	)

	for _, v := range data {
		link, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			x.log.Error("cannot encode server source", "error", err)
			continue
		}
		source = string(link)
		if source == "" {
			continue
		}

		x.log.Info("found iframe source", "src", source)

		result = append(result, models.AnimeVideo{
			Source:   source,
			Type:     "sub",
			Quality:  "hd",
			Language: language,
		})
	}

	return result
}

func (x *witAnime) download(text string) []models.AnimeVideo {
	matches := witDlExp(text)
	if len(matches) < 1 {
		return nil
	}

	var data []string
	txt := strings.ReplaceAll(matches[1], `\\/`, `/`)
	err := json.Unmarshal([]byte(txt), &data)
	if err != nil {
		return nil
	}
	if len(data) == 0 {
		return nil
	}

	var (
		source   string
		language = analyze.CleanLanguage("arabic")
		result   []models.AnimeVideo
	)

	for _, v := range data {
		link, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			x.log.Error("cannot encode server source", "error", err)
			continue
		}
		source = string(link)
		if source == "" {
			continue
		}
		result = append(result, models.AnimeVideo{
			Source:   source,
			Type:     "sub",
			Quality:  "hd",
			Language: language,
		})
	}

	return result
}

func (x *witAnime) scrape(nodes []*EpisodeNode) (*[]*EmbedNode, error) {
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
			if err = func() error {
				time.Sleep(100 * time.Millisecond)
				body, err := client.Do(x.ctx, &client.Args{
					Method:   http.MethodGet,
					Endpoint: v.Link,
				})
				if err != nil {
					x.log.Error("cannot load the episode page", "error", err)
					return err
				}

				doc, err := goquery.NewDocumentFromReader(body)
				if err != nil {
					x.log.Error("cannot parse the episode page", "error", err)
					return err
				}

				txt := doc.Text()
				video := x.video(txt)
				download := x.download(txt)

				result[i] = &EmbedNode{
					Number:   v.Number,
					Videos:   video,
					Download: download,
				}
				return nil
			}(); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil, err
				}
			}
		}
	}

	return &result, nil
}
