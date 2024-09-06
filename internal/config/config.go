package config

import (
	"bufio"
	"errors"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

var (
	comment = regexp.MustCompile(`\s*#.*$|^\s+|\s+$`).ReplaceAllString
	logger  = slog.Default().WithGroup("[CONFIG]")
)

type Config struct {
	Proxy        string
	CertFile     string
	KeyFile      string
	TMDBKey      string
	TVDBKey      string
	SimklTokens  []string
	FunArtTokens []string
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		config  = new(Config)
		scanner = bufio.NewScanner(file)
	)

	for scanner.Scan() {
		txt := scanner.Text()
		if !strings.Contains(txt, "=") {
			continue
		}

		line := strings.TrimSpace(comment(txt, ""))
		kv := strings.SplitN(line, "=", 2)
		if len(kv) < 1 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(strings.ReplaceAll((strings.ReplaceAll(kv[1], `"`, "")), `'`, ""))

		switch key {
		case "TMDB_KEY":
			if value == "" {
				logger.Warn("no tmdb key value", "key", key)
				return nil, errors.New("you need to set the TMDB API key")
			}
			logger.Info("value was set", "key", key)
			config.TMDBKey = value
		case "TVDB_KEY":
			if value == "" {
				logger.Warn("no tvdb key value", "key", key)
				return nil, errors.New("you need to set the TVDB API key")
			}
			config.TVDBKey = value
		case "PROXY_URL":
			if value == "" {
				logger.Warn("no proxy url value", "key", key)
			} else {
				logger.Info("value was set", "key", key)
				config.Proxy = value
			}
		case "CERT_FILE":
			if value == "" {
				logger.Warn("no certification file value", "key", key)
			} else {
				logger.Info("value was set", "key", key)
				config.CertFile = value
			}
		case "KEY_FILE":
			if value == "" {
				logger.Warn("no certification key file value", "key", key)
			} else {
				logger.Info("value was set", "key", key)
				config.KeyFile = value
			}
		case "SIMKL_TOKENS":
			var tokens []string
			for _, t := range strings.Split(value, ",") {
				tokens = append(tokens, strings.TrimSpace(t))
			}
			if len(tokens) == 0 {
				logger.Warn("no simkl tokens value", "key", key)
				return nil, errors.New("you need to set at least one simkl token")
			}

			logger.Info("value was set", "key", key)
			config.SimklTokens = tokens
		case "FUNART_TOKENS":
			var tokens []string
			for _, t := range strings.Split(value, ",") {
				tokens = append(tokens, strings.TrimSpace(t))
			}
			if len(tokens) == 0 {
				logger.Warn("no funart tokens value", "key", key)
				return nil, errors.New("you need to set at least one funart token")
			}

			logger.Info("value was set", "key", key)
			config.FunArtTokens = tokens
		}
	}

	return config, nil
}
