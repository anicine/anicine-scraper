package shared

import "github.com/anicine/anicine-scraper/models"

var (
	Countries = [15]models.Country{
		{Name: "japan", ShortName: "jpn", ISO3166_1: "jp"},
		{Name: "united-states", ShortName: "usa", ISO3166_1: "us"},
		{Name: "south-korea", ShortName: "kor", ISO3166_1: "kr"},
		{Name: "china", ShortName: "chn", ISO3166_1: "cn"},
		{Name: "india", ShortName: "ind", ISO3166_1: "in"},
		{Name: "france", ShortName: "fra", ISO3166_1: "fr"},
		{Name: "united-kingdom", ShortName: "gbr", ISO3166_1: "gb"},
		{Name: "germany", ShortName: "deu", ISO3166_1: "de"},
		{Name: "italy", ShortName: "ita", ISO3166_1: "it"},
		{Name: "canada", ShortName: "can", ISO3166_1: "ca"},
		{Name: "australia", ShortName: "aus", ISO3166_1: "au"},
		{Name: "brazil", ShortName: "bra", ISO3166_1: "br"},
		{Name: "mexico", ShortName: "mex", ISO3166_1: "mx"},
		{Name: "spain", ShortName: "esp", ISO3166_1: "es"},
		{Name: "russia", ShortName: "rus", ISO3166_1: "ru"},
	}
	Languages = [13]models.Language{
		{Name: "english", ISO639_1: "en"},
		{Name: "arabic", ISO639_1: "ar"},
		{Name: "japanese", ISO639_1: "ja"},
		{Name: "italian", ISO639_1: "it"},
		{Name: "spanish", ISO639_1: "es"},
		{Name: "portuguese", ISO639_1: "pt"},
		{Name: "russian", ISO639_1: "ru"},
		{Name: "german", ISO639_1: "de"},
		{Name: "korean", ISO639_1: "ko"},
		{Name: "mandarin-chinese", ISO639_1: "zh"},
		{Name: "hindi", ISO639_1: "hi"},
		{Name: "french", ISO639_1: "fr"},
		{Name: "thai", ISO639_1: "th"},
	}
	Seasons = [4]struct {
		Name string
		Zone []uint8
	}{
		{
			Name: "fall",
			Zone: []uint8{9, 10, 11},
		},
		{
			Name: "winter",
			Zone: []uint8{12, 1, 2},
		},
		{
			Name: "spring",
			Zone: []uint8{3, 4, 5},
		},
		{
			Name: "summer",
			Zone: []uint8{6, 7, 8},
		},
	}
)
