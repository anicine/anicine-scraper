package analyze

import (
	"errors"
	"strconv"
	"strings"

	"github.com/anicine/anicine-scraper/internal/errs"
	"github.com/anicine/anicine-scraper/models"
)

func ExtractYear(input string) (int, error) {
	if input == "" {
		return 0, errs.ErrBadData
	}

	matches := yearExp.FindStringSubmatch(input)

	if len(matches) < 2 {
		return 0, errors.New("no four digits found")
	}

	year, err := strconv.ParseInt(matches[1], 0, 0)
	if err != nil {
		return 0, errors.New("cannot make four digits string a int")
	}

	return int(year), nil
}

func ExtractAid(input string) (int, error) {
	if input == "" {
		return 0, errs.ErrBadData
	}

	// Find the first match in the string
	match := aidExp.FindStringSubmatch(input)
	if len(match) < 2 {
		return 0, errs.ErrNotFound
	}

	// Convert the aid number to an integer
	aid, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}

	return aid, nil
}

func ExtractNum(input string) int {
	if input == "" {
		return 0
	}

	match := numExp.FindStringSubmatch(input)
	if len(match) < 2 {
		return 0
	}

	number, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}

	return number
}

func ExtractIntsWithRanges(input string) []int {
	var extractedInts []int
	encountered := make(map[int]bool) // Keep track of encountered numbers

	// Find all non-overlapping matches
	for {
		match := intsExp.FindStringSubmatch(input)
		if match == nil {
			break
		}

		if match[1] != "" { // Check if it's a range match
			start, _ := strconv.Atoi(match[1])
			end, _ := strconv.Atoi(match[2])

			// Append range of integers (inclusive) if not encountered before
			for i := start; i <= end; i++ {
				if !encountered[i] {
					extractedInts = append(extractedInts, i)
					encountered[i] = true
				}
			}
		} else { // Single digit
			num, _ := strconv.Atoi(match[3])

			// Append single digit if not encountered before
			if !encountered[num] {
				extractedInts = append(extractedInts, num)
				encountered[num] = true
			}
		}

		// Update input to exclude the matched part for further iterations
		input = input[len(match[0]):]
	}

	return extractedInts
}

func ExtractAnimeLinks(input string) []models.AnimeLink {
	if input == "" {
		return nil
	}
	matches := linksExp.FindAllStringSubmatch(input, -1)

	var links []models.AnimeLink
	for _, match := range matches {
		links = append(links, models.AnimeLink{
			Site: match[1],
			URL:  match[2],
		})
	}

	return links
}

func ExtractEngChars(input string) string {
	if input == "" {
		return input
	}

	return engExp.ReplaceAllString(input, "")
}

func ExtractContentRating(i1 string) (string, string) {
	if i1 == "" {
		return "", ""
	}

	i1 = overviewExp.ReplaceAllString(i1, "")
	i1 = strings.TrimSpace(strings.ToLower(i1))

	if strings.Contains(i1, "pg-13") {
		return TVPG3, MPAA3
	}
	if strings.Contains(i1, "pg-") {
		return TVPG2, MPAA2
	}
	if strings.Contains(i1, "g-") {
		return TVPG1, MPAA1
	}
	if strings.Contains(i1, "r-") {
		return TVPG4, MPAA4
	}
	if strings.Contains(i1, "r+") || strings.Contains(i1, "rx") {
		return TVPG5, MPAA5
	}

	return "", ""
}

func ExtractAnimePath(i1 string) string {
	if i1 == "" {
		return ""
	}

	match := pathExp.FindStringSubmatch(i1)
	if len(match) > 1 {
		return match[1]
	}

	return ""
}
