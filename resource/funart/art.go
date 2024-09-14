package funart

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/internal/analyze"
	"github.com/anicine/anicine-scraper/internal/errs"
)

func Movie(ctx context.Context, tmdbid int) (*FunArtMovie, error) {
	var (
		err      error
		endpoint = &url.URL{
			Scheme:   "https",
			Host:     "webservice.fanart.tv",
			Path:     "/v3/movies/" + strconv.Itoa(tmdbid),
			RawQuery: "api_key=" + generate(),
		}
	)

	var art FunArtMovie
	for i := 0; i < length(); i += 1 {
		if err = func() error {
			body, err := client.Do(ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: endpoint,
			})
			if err != nil {
				logger.Warn("retry to query anime art", "TMDB", tmdbid, "error", err)
				endpoint.RawQuery = "api_key=" + generate()
				return err
			}

			err = json.NewDecoder(body).Decode(&art)
			if err != nil {
				logger.Error("cannot parse anime art data", "error", err)
				return err
			}

			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			continue
		}
		break
	}

	if analyze.ExtractNum(art.TmdbID) != tmdbid {
		return nil, errs.ErrBadData
	}

	logger.Info("anime art was added", "TMDB", tmdbid)

	return &art, nil
}

func TV(ctx context.Context, tvdbid int) (*FunArtTv, error) {
	var (
		err      error
		endpoint = &url.URL{
			Scheme:   "https",
			Host:     "webservice.fanart.tv",
			Path:     "/v3/tv/" + strconv.Itoa(tvdbid),
			RawQuery: "api_key=" + generate(),
		}
	)

	var art FunArtTv
	for i := 0; i < length(); i += 1 {
		if err = func() error {
			body, err := client.Do(ctx, &client.Args{
				Proxy:    true,
				Method:   http.MethodGet,
				Endpoint: endpoint,
			})
			if err != nil {
				logger.Warn("retry to query anime art", "TVDB", tvdbid, "error", err)
				endpoint.RawQuery = "api_key=" + generate()
				return err
			}

			err = json.NewDecoder(body).Decode(&art)
			if err != nil {
				logger.Error("cannot parse anime art data", "error", err)
				return err
			}

			return nil
		}(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			continue
		}
		break
	}

	if analyze.ExtractNum(art.TheTvdbID) != tvdbid {
		return nil, errs.ErrBadData
	}

	logger.Info("anime art was added", "TVDB", tvdbid)

	return &art, nil
}
