package shared

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/anicine/anicine-scraper/client"
	"github.com/anicine/anicine-scraper/models"
)

var (
	translateLogger = slog.Default().WithGroup("[TRANSLATE]")
)

func Translate(i1, i2 string) []models.MetaData {
	var (
		metadata = make([]models.MetaData, len(Languages))
		wg       sync.WaitGroup
		mx       sync.Mutex
	)

	metadata[0] = models.MetaData{
		Language: Languages[0],
		Title:    i1,
		OverView: i2,
	}

	wg.Add(len(Languages) - 1)
	for i := 1; i < len(Languages); i++ {
		go func(i int, v models.Language) {
			defer wg.Done()

			title, err := gt(
				i1,
				"en",
				v.ISO639_1,
			)
			if err != nil {
				translateLogger.Error("cannot translate the title", "language", v.Name, "title", i1, "error", err)
				return
			}

			overview, err := gt(
				i2,
				"en",
				v.ISO639_1,
			)
			if err != nil {
				translateLogger.Error("cannot translate the overview", "language", v.Name, "overview", i2, "error", err)
				return
			}

			translateLogger.Info("title & overview translated successfully", "language", v.Name)

			defer mx.Unlock()
			mx.Lock()
			metadata[i] = models.MetaData{
				Language: v,
				Title:    title,
				OverView: overview,
			}
		}(i, Languages[i])
	}
	wg.Wait()

	return metadata
}

func gt(txt, from, to string) (string, error) {
	var (
		params   = url.Values{}
		endpoint = &url.URL{
			Scheme: "https",
			Host:   "translate.google.com",
			Path:   "/translate_a/single",
		}
		data = map[string]string{
			"client": "gtx",
			"sl":     from,
			"tl":     to,
			"hl":     to,
			"ie":     "UTF-8",
			"oe":     "UTF-8",
			"otf":    "1",
			"ssel":   "0",
			"tsel":   "0",
			"kc":     "7",
			"q":      txt,
		}
		result string
	)

	for k, v := range data {
		params.Add(k, v)
	}
	for _, v := range []string{"at", "bd", "ex", "ld", "md", "qca", "rw", "rm", "ss", "t"} {
		params.Add("dt", v)
	}

	endpoint.RawQuery = params.Encode()
	body, err := client.Do(context.TODO(), &client.Args{
		Proxy:    true,
		Method:   http.MethodGet,
		Endpoint: endpoint,
	})
	if err != nil {
		return result, err
	}

	var raw []interface{}
	err = json.NewDecoder(body).Decode(&raw)
	if err != nil {
		return result, err
	}

	if len(raw) == 0 {
		return result, err
	}

	for _, obj := range raw[0].([]interface{}) {
		if len(obj.([]interface{})) == 0 {
			break
		}

		t, ok := obj.([]interface{})[0].(string)
		if ok {
			result += t
		}
	}

	return result, nil
}
