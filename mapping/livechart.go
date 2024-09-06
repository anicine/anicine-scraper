package mapping

import (
	"context"
	"errors"
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

type liveChart struct {
	id   int
	info *models.AnimeInfo
	ctx  context.Context
}

func LiveChart(ctx context.Context, info *models.AnimeInfo, id int) (*models.AnimeResource, error) {
	x := liveChart{
		id:   id,
		info: info,
		ctx:  ctx,
	}

	var (
		resource *models.AnimeResource
		err      error
	)

	if x.id != 0 {
		resource, err = x.check()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
		if resource != nil {
			return resource, nil
		}
	}

	found, err := x.query()
	if err != nil {
		return nil, err
	}
	if found {
		resource, err = x.check()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
		}
		if resource != nil {
			return resource, nil
		}
	}

	return nil, errs.ErrNotFound
}

func (x *liveChart) query() (bool, error) {
	body, err := client.Do(x.ctx, &client.Args{
		Method: http.MethodGet,
		Endpoint: &url.URL{
			Scheme:   "https",
			Host:     "www.livechart.me",
			Path:     "/search",
			RawQuery: "q=" + x.info.Query,
		},
	})
	if err != nil {
		return false, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return false, err
	}

	var found bool
	block := doc.Find(".anime-list")
	if block != nil {
		block.Find("li").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if premiere, ok := s.Attr("data-premiere"); ok {
				unix, err := strconv.ParseInt(premiere, 10, 64)
				if err != nil {
					return true
				}

				date := time.Unix(int64(unix), 0)
				if date.Year() == x.info.SD.Year && date.Month() == time.Month(x.info.SD.Month) {
					if data, ok := s.Attr("data-anime-id"); ok {
						x.id, err = strconv.Atoi(data)
						if err != nil {
							return true
						}
						found = true
						return false
					}
				}
			}
			return true
		})
	}

	return found, nil
}

func (x *liveChart) crawl(doc *goquery.Document) *models.AnimeResource {
	if doc == nil {
		return nil
	}

	resource := new(models.AnimeResource)
	if href, ok := doc.Find(".lc-btn-myanimelist").Attr("href"); ok {
		if id := analyze.ExtractNum(analyze.ExtractAnimePath(href + `/`)); id != 0 {
			resource.Mal = id
		}
	}

	if resource.Mal != x.info.MalID {
		return nil
	}

	resource.LiveChart = int64(x.id)

	if href, ok := doc.Find(".lc-btn-anilist").Attr("href"); ok {
		if id := analyze.ExtractNum(analyze.ExtractAnimePath(href + `/`)); id != 0 {
			resource.AniList = id
		}
	}

	if href, ok := doc.Find(".lc-btn-anidb").Attr("href"); ok {
		link, _ := strings.CutPrefix(href, "https://anidb.net/")
		link, _ = strings.CutPrefix(link, "http://anidb.net/")
		if len(link) > 0 {
			link = strings.Split(link, "/")[0]
			if id := analyze.ExtractNum(link); id != 0 {
				resource.AniDB = id
			} else {
				link, _ = strings.CutSuffix(href, "/")
				if len(link) > 0 {
					paths := strings.Split(link, "/")
					link = paths[len(paths)-1]
					if id = analyze.ExtractNum(link); id != 0 {
						resource.AniDB = id
					}
				}
			}
		}
	}

	if href, ok := doc.Find(".lc-btn-anisearch").Attr("href"); ok {
		if id := analyze.ExtractNum(analyze.ExtractAnimePath(href + `/`)); id != 0 {
			resource.AniSearch = int64(id)
		}
	}

	if href, ok := doc.Find(".lc-btn-kitsu").Attr("href"); ok {
		if id := analyze.ExtractNum(analyze.ExtractAnimePath(href + `/`)); id != 0 {
			resource.Kitsu = strconv.Itoa(id)
		}
	}

	if href, ok := doc.Find(".lc-btn-animeplanet").Attr("href"); ok {
		resource.AnimePlanet = analyze.ExtractAnimePath(href + `/`)
	}

	return resource
}

func (x *liveChart) check() (*models.AnimeResource, error) {
	if x.id == 0 {
		return nil, errs.ErrBadData
	}

	body, err := client.Do(x.ctx, &client.Args{
		Method: http.MethodGet,
		Endpoint: &url.URL{
			Scheme: "https",
			Host:   "www.livechart.me",
			Path:   "/anime/" + strconv.Itoa(x.id),
		},
		Headers: map[string]string{
			"Authority": "www.livechart.me",
		},
	})
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	block := doc.Find(".lc-poster-col")
	if block != nil {
		var found bool
		block.Find(".text-sm").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if s != nil {
				hover := s.Find(".link-hover")
				if hover != nil {
					path, ok := hover.Attr("href")
					if ok {
						if strings.Contains(path, "date") {
							if len(hover.Text()) > 0 {
								date, err := time.Parse("January 02, 2006", hover.Text())
								if err != nil {
									date, err = time.Parse(time.DateOnly, strings.TrimPrefix(path, "/schedule?date="))
									if err != nil {
										return true
									}
								}

								if date.Year() == x.info.SD.Year && date.Month() == time.Month(x.info.SD.Month) {
									found = true
									return false
								}
							}
						}
					}
				}
			}
			return true
		})
		if found {
			return x.crawl(doc), nil
		}
	}

	return nil, errs.ErrNotFound
}
