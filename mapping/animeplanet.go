package mapping

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/internal/shared"
	"github.com/anicine/anicine-scraper/models"
)

func AnimePlanet(ctx context.Context, info *models.AnimeInfo) (*models.AnimeResource, error) {
	query := url.QueryEscape(strings.ReplaceAll(info.Query, "-", " "))
	body, err := client.Do(ctx, &client.Args{
		Method: http.MethodGet,
		Endpoint: &url.URL{
			Scheme:   "https",
			Host:     "www.anime-planet.com",
			Path:     "/anime/all",
			RawQuery: fmt.Sprintf("name=%s&year=%d&to_year=%d", query, info.SD.Year, info.SD.Year),
		},
	})
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	var (
		found    bool
		resource = new(models.AnimeResource)
	)

	doc.Find("ul.cardGrid").Find("li").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if title, ok := s.Attr("title"); ok && title != "" {
			block, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(title)))
			if err != nil {
				return true
			}

			nameTxt := block.Find(".theme-font")
			if nameTxt == nil {
				return true
			}

			year, err := strconv.Atoi(block.Find(".iconYear").Text())
			if err != nil {
				return true
			}

			if year == info.SD.Year {
				sim := shared.TextAdvancedSimilarity(info.Query, nameTxt.Text())
				if sim > 90 {
					if data, ok := s.Attr("data-id"); ok {
						if data != "" {
							resource.AnimePlanet = data
							found = true
							return false
						}
					}
				}
			}
		}
		return true
	})
	if found {
		return resource, nil
	}

	return nil, errs.ErrNotFound
}
