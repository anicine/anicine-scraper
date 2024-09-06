package analyze

import (
	"strings"

	"github.com/anicine/anicine-scraper/internal/shared"
	"github.com/anicine/anicine-scraper/models"
)

func CleanUnicode(input string) string {
	if input == "" {
		return input
	}

	for _, u := range unicode {
		input = strings.ReplaceAll(input, u, " ")
	}
	input = strings.Join(strings.Fields(input), " ")

	return input
}

func CleanTitle(input string) string {
	if input == "" {
		return input
	}

	input = strings.ToLower(CleanUnicode(input))

	for _, c := range chars {
		input = strings.ReplaceAll(input, c, " ")
	}

	input = strings.ReplaceAll(strings.Join(strings.Fields(input), " "), " ", "-")
	cleanedText := lineExp.ReplaceAllString(strings.ReplaceAll(input, "--", "-"), "")
	return cleanedText
}

func CleanQuery(input string) string {
	return strings.ReplaceAll(input, "-", "+")
}

func CleanOverview(input string) string {
	if input == "" {
		return input
	}

	var data []string
	for _, v := range strings.Split(strings.ReplaceAll(strings.TrimSuffix(input, "\n"), `\n\n`, `\n`), `\n`) {
		v = overviewExp.ReplaceAllString(v, "")
		v = strings.TrimSpace(CleanUnicode(v))
		if v == "" || v == " " || v == "\n" {
			continue
		}

		data = append(data, v)
	}
	return strings.Join(data, "\n")
}

func CleanSeasonTitle(input string) string {
	if input == "" {
		return input
	}

	return seasonTitleExp.ReplaceAllString(input, "")
}

func CleanCountry(input string) string {
	if input == "" {
		return input
	}

	input = CleanTitle(input)
	for _, v := range shared.Countries {
		o0 := shared.TextAdvancedSimilarity(input, v.ISO3166_1)
		o1 := shared.TextAdvancedSimilarity(input, v.Name)
		o2 := shared.TextAdvancedSimilarity(input, v.ShortName)
		if o0 > 90 || o1 > 90 || o2 > 90 {
			return v.ISO3166_1
		}
	}

	return ""
}

func CleanYTKey(input string) string {
	for _, pattern := range ytExp {
		matches := pattern.FindStringSubmatch(input)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	return ""
}

func CleanAnimeContentRating(input string) (string, string) {
	if input == "" {
		return input, input
	}

	matches := crExp.FindStringSubmatch(input)

	var (
		s1 string
		s2 string
	)

	if len(matches) > 0 {
		s1 = strings.TrimSpace(matches[1])
		if len(matches) > 2 && len(matches[2]) > 0 {
			s2 = strings.TrimSpace(matches[2])
		}
	}

	return s1, s2
}

func CleanStrings(input []string) []string {
	unique := make(map[string]struct{})
	result := make([]string, 0)

	for _, item := range input {
		item = CleanUnicode(item)
		if item == "" {
			continue
		}
		if _, found := unique[item]; !found {
			unique[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func CleanTag(input string) string {
	if input == "" {
		return input
	}

	input = strings.ToLower(input)
	tags := []string{
		"maintenance",
		"to episode",
		"moved to",
		"tag",
		"element",
		"setting",
		"themes",
		"deleted",
		"content",
		"-- ",
	}

	for _, t := range tags {
		if strings.Contains(input, t) {
			return ""
		}
	}

	return CleanUnicode(input)
}

func CleanJson(input string) string {
	if input == "" {
		return input
	}

	input = strings.ReplaceAll(input, " ", "")
	if len(input) <= 4 {
		input = ""
	}

	return input
}

func CleanRuntime(input string) string {
	data := map[string]string{
		"hours":   "h ",
		"hour":    "h ",
		"hr":      "h ",
		"minutes": "m ",
		"minute":  "m ",
		"min":     "m ",
		"seconds": "s ",
		"second":  "s ",
		"sec":     "s ",
	}

	input = strings.ToLower(input)
	for k, v := range data {
		input = strings.ReplaceAll(input, k, v)
	}

	return strings.Join(strings.Fields(input), " ")
}

func CleanLanguage(input string) models.Language {
	var data models.Language
	if input == "" {
		return data
	}

	input = CleanTitle(input)
	for _, v := range shared.Languages {
		if input == v.Name {
			return v
		}
	}

	return data
}

func CleanRepetition[T int | int64 | string](input *[]T) *T {
	if input == nil {
		return nil
	}

	var filter map[T]int = make(map[T]int)

	for _, v := range *input {
		if x, ok := filter[v]; ok {
			filter[v] = x + 1
		} else {
			filter[v] = 1
		}
	}

	var (
		length int
		result *T
	)

	for k, v := range filter {
		if v > length {
			length = v
			result = &k
		}
	}

	return result
}
