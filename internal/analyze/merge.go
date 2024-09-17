package analyze

import (
	"log/slog"
	"strings"

	"github.com/anicine/anicine-scraper/internal/shared"
	"github.com/anicine/anicine-scraper/models"
)

func MergeAnimeImages(images []models.AnimeImage) []models.AnimeImage {
	var filter = make(map[string]*models.AnimeImage)

	for _, v := range images {
		if v.Image == "" {
			continue
		}

		if x, ok := filter[v.Image]; ok {
			if x.Thumbnail == "" {
				x.Thumbnail = v.Thumbnail
			}

			if x.Height == 0 {
				x.Height = v.Height
			}

			if x.Width == 0 {
				x.Width = v.Width
			}
		} else {
			filter[v.Image] = &v
		}
	}

	var (
		data = make([]models.AnimeImage, len(filter))
		i    int
	)
	for _, v := range filter {
		data[i] = *v
		i++
	}

	return data
}

func mergeAnimeIDs(animeIDs ...models.AnimeID) models.AnimeID {
	merged := models.AnimeID{}
	for _, id := range animeIDs {
		if id.Mal != 0 {
			merged.Mal = id.Mal
		}
		if id.AniList != 0 {
			merged.AniList = id.AniList
		}
		if id.AniDB != 0 {
			merged.AniDB = id.AniDB
		}
		if id.TVDBID != 0 {
			merged.TVDBID = id.TVDBID
		}
		if id.TMDBID != 0 {
			merged.TMDBID = id.TMDBID
		}
	}

	return merged
}

func mergeAnimeCompany(companies []models.AnimeCompany) []models.AnimeCompany {
	var (
		data  = make([]models.AnimeCompany, 0)
		found bool
	)

	for _, v := range companies {
		name := CleanTitle(v.Name)
		if name == "" {
			continue
		}

		for i, x := range data {
			if x.Name == name {
				data[i].ID = mergeAnimeIDs(x.ID, v.ID)
				found = true
				break
			}
			found = false
		}

		if !found {
			data = append(data, models.AnimeCompany{
				ID:   v.ID,
				Name: name,
			})
		}

	}

	return data
}

func mergeAnimeThumbnails(images []models.AnimeImage) models.AnimeImage {
	var (
		data models.AnimeImage
		rnk  uint8
		idx  int
	)

	for i, v := range images {
		if v.Image == "" {
			continue
		}

		if strings.Contains(v.Image, "tmdb") {
			rnk = 1
			idx = i
			break
		} else if strings.Contains(v.Image, "anilist") && rnk > 1 {
			rnk = 2
			idx = i
		} else if strings.Contains(v.Image, "myanimelist") && rnk > 2 {
			rnk = 3
			idx = i
		}
	}

	if rnk != 0 {
		data = images[idx]
	} else {
		var chk int
		for _, v := range images {
			if v.Image == "" {
				continue
			}
			if data.Image == "" {
				data.Image = v.Image
				chk++
			}
			if data.Thumbnail == "" {
				data.Thumbnail = v.Thumbnail
				chk++
			}
			if chk == 2 {
				break
			}
		}
	}

	return data
}

func mergeAnimeVoiceActor(acts ...[]models.AnimeVoiceActor) []models.AnimeVoiceActor {
	var (
		filter = make(map[string]*models.AnimeVoiceActor)
		name   string
	)

	for _, v := range acts {
		if v == nil {
			continue
		}

		if len(v) > 0 {
			for _, x := range v {
				var actor *models.AnimeVoiceActor
				name = CleanTitle(x.Name.Full)
				if name == "" {
					continue
				}
				for i, y := range filter {
					if y.Language.ISO639_1 == x.Language.ISO639_1 {
						sum := shared.TextSimpleSimilarity(i, name)
						slog.Default().Debug("MERGE-VOICE-ACTOR", "[1]", i, "[2]", name, "SUM", sum)
						if sum > 80 {
							actor = y
							break
						}
					}
				}
				if actor == nil {
					actor = new(models.AnimeVoiceActor)
					filter[name] = actor
				}

				actor.Language = x.Language
				actor.ID = mergeAnimeIDs(actor.ID, x.ID)
				if actor.Name.Full == "" {
					actor.Name.Full = x.Name.Full
				}
				if actor.Name.Native == "" {
					actor.Name.Native = x.Name.Native
				}
				if len(x.Name.Alternative) > 0 {
					actor.Name.Alternative = append(actor.Name.Alternative, x.Name.Alternative...)
				}
				if len(x.Name.Spoilers) > 0 {
					actor.Name.Spoilers = append(actor.Name.Spoilers, x.Name.Spoilers...)
				}
				if len(x.Images) > 0 {
					actor.Images = append(actor.Images, x.Images...)
				}
				if len(x.SocialMedia) > 0 {
					actor.SocialMedia = append(actor.SocialMedia, x.SocialMedia...)
				}
				if len(x.Images) > 0 {
					actor.Images = append(actor.Images, x.Images...)
				}
				if actor.Age == 0 {
					actor.Age = x.Age
				}
				if strings.TrimSpace(actor.Gender) == "" {
					actor.Gender = strings.ReplaceAll(x.Gender, " ", "")
				}
				if actor.DateOfBirth.IsZero() {
					actor.DateOfBirth = x.DateOfBirth
				}
				if actor.DateOfDeath.IsZero() {
					actor.DateOfDeath = x.DateOfDeath
				}
				if strings.TrimSpace(actor.Home) == "" {
					actor.Home = CleanUnicode(strings.TrimSpace(x.Home))
				}
			}
		}
	}

	var (
		actors = make([]models.AnimeVoiceActor, len(filter))
		i      int
	)

	for _, v := range filter {
		actors[i] = *v
		actors[i].Images = MergeAnimeImages(actors[i].Images)
		i++
	}

	return actors
}

func MergeAnimeRelation(anime ...*models.Anime) []models.AnimeRelation {
	var relations = make([]models.AnimeRelation, 0)
	for _, v := range anime {
		if v == nil {
			continue
		}
		if len(v.Relations) > 0 {
			for _, x := range v.Relations {
				var (
					r = new(models.AnimeRelation)
					f bool
				)
				r.Nature = x.Nature
				for i, y := range relations {
					if y.Nature == x.Nature {
						r = &relations[i]
						f = true
					}
				}

				if len(r.Nodes) > 0 {
					for i, z := range r.Nodes {
						for _, a := range x.Nodes {
							if (CleanTitle(z.Name) == CleanTitle(a.Name)) && (z.Type == a.Type) {
								r.Nodes[i].ID = mergeAnimeIDs(z.ID, a.ID)
							} else {
								r.Nodes = append(r.Nodes, a)
							}
						}
					}
				} else {
					r.Nodes = append(r.Nodes, x.Nodes...)
				}

				if !f {
					relations = append(relations, *r)
				}
			}
		}
	}

	return relations
}

func MergeAnimeCharacter(anime ...*models.Anime) []models.AnimeCharacter {
	var (
		filter = make(map[string]*models.AnimeCharacter)
		name   string
	)

	for _, v := range anime {
		if v == nil {
			continue
		}
		if len(v.Characters) > 0 {
			for _, x := range v.Characters {
				var character *models.AnimeCharacter
				name = CleanUnicode(strings.ReplaceAll(x.Name.Native, " ", ""))
				if name == "" {
					continue
				}
				for i, y := range filter {
					if i == name {
						character = y
						break
					}
				}
				if character == nil {
					character = new(models.AnimeCharacter)
					filter[name] = character
				}

				character.ID = mergeAnimeIDs(character.ID, x.ID)
				if character.Name.Full == "" {
					character.Name.Full = x.Name.Full
				}
				if character.Name.Native == "" {
					character.Name.Native = x.Name.Native
				}
				if len(x.Name.Alternative) > 0 {
					character.Name.Alternative = append(character.Name.Alternative, x.Name.Alternative...)
				}
				if len(x.Name.Spoilers) > 0 {
					character.Name.Spoilers = append(character.Name.Spoilers, x.Name.Spoilers...)
				}
				if len(x.Images) > 0 {
					character.Images = append(character.Images, x.Images...)
				}
				if character.Description.Mal == "" {
					character.Description.Mal = x.Description.Mal
				}
				if character.Description.AniList == "" {
					character.Description.AniList = x.Description.AniList
				}
				if character.InitialAge == 0 {
					character.InitialAge = x.InitialAge
				}
				if character.Gender == "" {
					character.Gender = CleanTitle(x.Gender)
				}
				if character.Role == "" {
					character.Role = CleanTitle(x.Role)
				}
				if character.DateOfBirth.IsZero() {
					character.DateOfBirth = x.DateOfBirth
				}

				character.VoiceActor = mergeAnimeVoiceActor(character.VoiceActor, x.VoiceActor)
			}
		}
	}

	var (
		characters = make([]models.AnimeCharacter, len(filter))
		i          int
	)

	for _, v := range filter {
		characters[i] = *v
		characters[i].Images = MergeAnimeImages(characters[i].Images)
		i++
	}

	return characters
}

func MergeAnimeOverview(anime ...*models.Anime) string {
	var (
		overviews []string
		longest   string
	)

	for _, v := range anime {
		if v != nil {
			overviews = append(overviews, v.Description)
		}
	}

	for _, v := range overviews {
		des := CleanOverview(v)
		if len(des) > len(longest) {
			longest = des
		}
	}

	return longest
}

func MergeAnimeResource(anime ...*models.Anime) models.AnimeResource {
	var (
		Mal         []int
		AniList     []int
		AniDB       []int
		Kitsu       []string
		TVDBID      []int64
		TMDBID      []int64
		IMDBID      []string
		LiveChart   []int64
		AnimePlanet []string
		AniSearch   []int64
		NotifyMoe   []string
		WikiData    []string
	)

	for _, v := range anime {
		if v != nil {
			if v.Resources.Mal != 0 {
				Mal = append(Mal, v.Resources.Mal)
			}
			if v.Resources.AniList != 0 {
				AniList = append(AniList, v.Resources.AniList)
			}
			if v.Resources.AniDB != 0 {
				AniDB = append(AniDB, v.Resources.AniDB)
			}
			if v.Resources.Kitsu != "" {
				Kitsu = append(Kitsu, strings.TrimSpace(v.Resources.Kitsu))
			}
			if v.Resources.TVDBID != 0 {
				TVDBID = append(TVDBID, v.Resources.TVDBID)
			}
			if v.Resources.TMDBID != 0 {
				TMDBID = append(TMDBID, v.Resources.TMDBID)
			}
			if v.Resources.LiveChart != 0 {
				LiveChart = append(LiveChart, v.Resources.LiveChart)
			}
			if v.Resources.IMDBID != "" {
				IMDBID = append(IMDBID, strings.TrimSpace(v.Resources.IMDBID))
			}
			if v.Resources.AnimePlanet != "" {
				AnimePlanet = append(AnimePlanet, strings.TrimSpace(v.Resources.AnimePlanet))
			}
			if v.Resources.AniSearch != 0 {
				AniSearch = append(AniSearch, v.Resources.AniSearch)
			}
			if v.Resources.NotifyMoe != "" {
				NotifyMoe = append(NotifyMoe, strings.TrimSpace(v.Resources.NotifyMoe))
			}
			if v.Resources.WikiData != "" {
				WikiData = append(WikiData, strings.TrimSpace(v.Resources.WikiData))
			}
		}
	}

	var merged models.AnimeResource
	if id := CleanRepetition(&Mal); id != nil {
		merged.Mal = *id
	}
	if id := CleanRepetition(&AniList); id != nil {
		merged.AniList = *id
	}
	if id := CleanRepetition(&AniDB); id != nil {
		merged.AniDB = *id
	}
	if id := CleanRepetition(&Kitsu); id != nil {
		merged.Kitsu = *id
	}
	if id := CleanRepetition(&TVDBID); id != nil {
		merged.TVDBID = *id
	}
	if id := CleanRepetition(&TMDBID); id != nil {
		merged.TMDBID = *id
	}
	if id := CleanRepetition(&LiveChart); id != nil {
		merged.LiveChart = *id
	}
	if id := CleanRepetition(&IMDBID); id != nil {
		merged.IMDBID = *id
	}
	if id := CleanRepetition(&AnimePlanet); id != nil {
		merged.AnimePlanet = *id
	}
	if id := CleanRepetition(&AniSearch); id != nil {
		merged.AniSearch = *id
	}
	if id := CleanRepetition(&NotifyMoe); id != nil {
		merged.NotifyMoe = *id
	}
	if id := CleanRepetition(&WikiData); id != nil {
		merged.WikiData = *id
	}

	return merged
}

func MergeAnimeTags(anime ...*models.Anime) []models.AnimeTag {
	var (
		data []models.AnimeTag          = make([]models.AnimeTag, 0)
		seen map[string]models.AnimeTag = make(map[string]models.AnimeTag)
	)

	for _, v := range anime {
		if v != nil {
			for _, x := range v.Tags {
				name := CleanTitle(CleanTag(string(x)))
				if name == "" {
					continue
				}

				_, ok := seen[name]
				if !ok {
					seen[name] = models.AnimeTag(name)
				}
			}
		}
	}

	for _, tag := range seen {
		data = append(data, tag)
	}

	return data
}

func MergeAnimeStudios(anime ...*models.Anime) []models.AnimeCompany {
	var data = make([]models.AnimeCompany, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Studios...)
		}
	}

	return mergeAnimeCompany(data)
}

func MergeAnimeProducers(anime ...*models.Anime) []models.AnimeCompany {
	var data = make([]models.AnimeCompany, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Producers...)
		}
	}

	return mergeAnimeCompany(data)
}

func MergeAnimeLicensors(anime ...*models.Anime) []models.AnimeCompany {
	var data = make([]models.AnimeCompany, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Licensors...)
		}
	}

	return mergeAnimeCompany(data)
}

func MergeAnimeGenres(anime ...*models.Anime) []models.AnimeGenre {
	filter := make(map[string]struct{})

	for _, v := range anime {
		if v != nil {
			for _, x := range v.Genres {
				name := CleanTitle(string(x))
				if name == "" {
					continue
				}

				if _, ok := filter[name]; !ok {
					filter[name] = struct{}{}
				}
			}
		}
	}

	var (
		data = make([]models.AnimeGenre, len(filter))
		i    int
	)

	for v := range filter {
		data[i] = models.AnimeGenre(v)
		i++
	}

	return data
}

func MergeAnimePortraitIMG(anime ...*models.Anime) models.AnimeImage {
	var data []models.AnimeImage

	for _, v := range anime {
		if v != nil {
			data = append(data, v.PortraitIMG)
		}
	}

	return mergeAnimeThumbnails(data)
}

func MergeAnimeLandscapeIMG(anime ...*models.Anime) models.AnimeImage {
	var data []models.AnimeImage
	for _, v := range anime {
		if v != nil {
			data = append(data, v.LandscapeIMG)
		}
	}
	return mergeAnimeThumbnails(data)
}

func MergeAnimeTitle(anime ...*models.Anime) models.AnimeTitles {
	var (
		data models.AnimeTitles
		et   []string
		ot   []string
		st   []string
	)

	for _, v := range anime {
		if v != nil {
			et = append(et, v.Titles.English...)
			ot = append(ot, v.Titles.Original...)
			st = append(st, v.Titles.Synonyms...)
		}
	}

	data.English = append(data.English, CleanStrings(et)...)
	data.Original = append(data.Original, CleanStrings(ot)...)
	data.Synonyms = append(data.Synonyms, CleanStrings(st)...)

	return data
}

func MergeAnimeTrailer(anime ...*models.Anime) []models.AnimeTrailer {
	var (
		data  = make([]models.AnimeTrailer, 0)
		found bool
	)

	for _, v := range anime {
		if v != nil {
			for _, x := range v.Trailers {
				for _, y := range data {
					if y.HostKey == x.HostKey {
						found = true
						break
					}
					found = false
				}
				if !found {
					data = append(data, x)
				}
			}
		}
	}

	return data
}

func MergeAnimeContentRating(anime ...*models.Anime) string {
	var (
		filter []string
		result string
	)

	for _, v := range anime {
		if v != nil {
			txt := CleanTitle(v.ContentRating)
			if txt != "" {
				filter = append(filter, txt)
			}
		}
	}

	result = *CleanRepetition(&filter)

	return result
}

func MergeAnimePosters(anime ...*models.Anime) []models.AnimeImage {
	var data = make([]models.AnimeImage, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Posters...)
		}
	}

	return MergeAnimeImages(data)
}

func MergeAnimeBackdrops(anime ...*models.Anime) []models.AnimeImage {
	var data = make([]models.AnimeImage, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Backdrops...)
		}
	}

	return MergeAnimeImages(data)
}

func MergeAnimeLogos(anime ...*models.Anime) []models.AnimeImage {
	var data = make([]models.AnimeImage, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Logos...)
		}
	}

	return MergeAnimeImages(data)
}

func MergeAnimeBanners(anime ...*models.Anime) []models.AnimeImage {
	var data = make([]models.AnimeImage, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Banners...)
		}
	}

	return MergeAnimeImages(data)
}

func MergeAnimeArts(anime ...*models.Anime) []models.AnimeImage {
	var data = make([]models.AnimeImage, 0)

	for _, v := range anime {
		if v != nil {
			data = append(data, v.Arts...)
		}
	}

	return MergeAnimeImages(data)
}

var externals = map[string]string{
	"twitter":          "x",
	"ja.wikipedia":     "ja-wiki",
	"en.wikipedia":     "en-wiki",
	"syoboi":           "syoboi",
	"animenewsnetwork": "anime-news-network",
	"bangumi.tv":       "bangumi",
	"bgm.tv":           "bangumi",
	"crunchyroll":      "crunchyroll",
	"netflix":          "netflix",
	"hulu":             "hulu",
	"bilibili":         "bilibili",
	"primevideo":       "prime-video",
	"hidive":           "hidive",
	"funimation":       "funimation",
	"iqiyi":            "iqiyi",
	"wetv":             "wetv",
}

func MergeAnimeExternals(anime ...*models.Anime) []models.AnimeLink {
	var data []models.AnimeLink
	var found bool
	for _, v := range anime {
		if v != nil {
			for _, x := range v.External {
				found = false
				for d, n := range externals {
					if strings.Contains(x.URL, d) {
						data = append(data, models.AnimeLink{
							Site: n,
							URL:  strings.TrimSpace(x.URL),
						})
						found = true
						break
					}
				}
				if found {
					continue
				}

				t := CleanTitle(x.Site)
				website := strings.Contains(t, "official-website")
				site := strings.Contains(t, "official-site")
				webpage := strings.Contains(t, "official-webpage")
				page := strings.Contains(t, "official-page")

				if website || site || webpage || page {
					data = append(data, models.AnimeLink{
						Site: "official-website",
						URL:  strings.TrimSpace(x.URL),
					})
					continue
				}
			}
		}
	}

	return data
}

func MergeAnimeCountry(anime ...*models.Anime) string {
	var (
		filter = make(map[string]bool)
		code   string
	)

	for _, v := range anime {
		if v != nil {
			code = CleanCountry(v.CountryOfOrigin)
			if code != "" {
				if filter[code] {
					return code
				} else {
					filter[code] = true
				}
			}
		}
	}

	return code
}

func MergeAnimeTypes(anime ...*models.Anime) string {
	var (
		filter = make(map[string]int)
		data   string
	)

	for _, v := range anime {
		if v != nil {
			if v.Type == "" {
				continue
			}
			v.Type = CleanUnicode(v.Type)

			if x, found := filter[v.Type]; found {
				x++
				continue
			}

			filter[v.Type] = 1
		}
	}

	var longest int
	for x, v := range filter {
		if v > longest {
			longest = v
			data = x
		}
	}

	return strings.ToUpper(data)
}

// func mergeAnimeStartDate(anime ...*models.Anime) models.AnimeDate {
// 	var (
// 		filter = make(map[models.AnimeDate]bool)
// 		date   models.AnimeDate
// 	)

// 	for _, v := range anime {
// 		if v != nil {
// 			if !v.StartAt.IsZero() {
// 				if filter[v.StartAt] {
// 					return v.StartAt
// 				} else {
// 					filter[v.StartAt] = true
// 				}
// 			}
// 		}
// 	}

// 	return date
// }

// func mergeAnimeEndDate(anime ...*models.Anime) models.AnimeDate {
// 	var (
// 		filter = make(map[models.AnimeDate]bool)
// 		date   models.AnimeDate
// 	)

// 	for _, v := range anime {
// 		if v != nil {
// 			if !v.EndAt.IsZero() {
// 				if filter[v.EndAt] {
// 					return v.EndAt
// 				} else {
// 					filter[v.EndAt] = true
// 				}
// 			}
// 		}
// 	}

// 	return date
// }

// func mergeAnimePeriod(date models.AnimeDate, anime ...*models.Anime) models.AnimePeriod {
// 	var (
// 		filter []models.AnimePeriod
// 		data   models.AnimePeriod
// 	)

// 	for _, v := range anime {
// 		if v != nil {
// 			if v.Period.Season == "" && v.Period.Year == 0 {
// 				continue
// 			}
// 			filter = append(filter, v.Period)
// 		}
// 	}

// 	for _, v := range filter {
// 		if v.Season != "" {
// 			data.Season = strings.ToLower(v.Season)
// 		}

// 		if v.Year != 0 {
// 			data.Year = v.Year
// 		}

// 		if data.Season != "" && data.Year != 0 {
// 			return data
// 		}
// 	}

// 	if data.Season == "" {
// 		for _, v := range shared.Seasons {
// 			for _, x := range v.Zone {
// 				if date.Month == int(x) {
// 					data.Season = v.Name
// 					return data
// 				}
// 			}

// 		}
// 	}

// 	return data
// }
