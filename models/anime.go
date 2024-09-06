package models

import "time"

type AnimeInfo struct {
	Title string
	Query string
	Type  string
	MalID int
	SD    AnimeDate
	ED    AnimeDate
}

type AnimeID struct {
	Mal     int   `json:"Mal"`
	AniList int   `json:"Anilist"`
	AniDB   int   `json:"Anidb"`
	TVDBID  int64 `json:"TVDbID"`
	TMDBID  int64 `json:"TMDbID"`
}

type AnimeLink struct {
	Site string `json:"Site"`
	URL  string `json:"Link"`
}

type AnimeTag string

type AnimeGenre string

type AnimeCompany struct {
	ID   AnimeID `json:"ID"`
	Name string  `json:"name"`
}

type AnimeTitles struct {
	Original []string `json:"Original"`
	English  []string `json:"English"`
	Synonyms []string `json:"Synonyms"`
}

type AnimeDate struct {
	Year  int `json:"Year"`
	Month int `json:"Month"`
	Day   int `json:"Day"`
}

func (v AnimeDate) IsZero() bool {
	return (v.Day == 0 && v.Month == 0 && v.Year == 0) || (v.Day == 1 && v.Month == 1 && v.Year == 1)
}

func (v *AnimeDate) Max(i *AnimeDate) {
	if i == nil {
		return
	}
	if i.Year > v.Year || (i.Year == v.Year && (i.Month > v.Month || (i.Month == v.Month && i.Day > v.Day))) {
		v.Year = i.Year
		v.Month = i.Month
		v.Day = i.Day
	}
}

type AnimePeriod struct {
	Season string `json:"Season"`
	Year   int    `json:"Year"`
}

type AnimeRelation struct {
	Nature string `json:"Nature"`
	Nodes  []struct {
		ID     AnimeID `json:"ID"`
		Name   string  `json:"Name"`
		Format string  `json:"Format"`
		Type   string  `json:"Type"`
	} `json:"Nodes"`
}

type AnimeImage struct {
	Height    int    `json:"Height,omitempty"`
	Width     int    `json:"Width,omitempty"`
	Image     string `json:"Image"`
	Thumbnail string `json:"Thumbnail"`
}

type AnimeTrailer struct {
	IsOfficial bool   `json:"IsOfficial"`
	HostName   string `json:"HostName"`
	HostKey    string `json:"HostKey"`
}

type AnimeName struct {
	Full        string   `json:"Full"`
	Native      string   `json:"Native"`
	Alternative []string `json:"Alternative"`
	Spoilers    []string `json:"Spoilers,omitempty"`
	Translate   []struct {
		Iso693_1 string `json:"Iso693_1"`
		Name     string `json:"Name"`
	} `json:"Translate,omitempty"`
}

type AnimeVoiceActor struct {
	ID          AnimeID      `json:"ID"`
	Name        AnimeName    `json:"Name"`
	Language    Language     `json:"Language"`
	Images      []AnimeImage `json:"Images"`
	SocialMedia []AnimeLink  `json:"SocialMedia"`
	Age         int          `json:"Age"`
	Gender      string       `json:"Gender"`
	DateOfBirth AnimeDate    `json:"DateOfBirth"`
	DateOfDeath AnimeDate    `json:"DateOfDeath"`
	Home        string       `json:"Home"`
}

type AnimeCharacter struct {
	ID          AnimeID      `json:"ID"`
	Name        AnimeName    `json:"Name"`
	Images      []AnimeImage `json:"Images"`
	Description struct {
		Mal     string `json:"Mal"`
		AniList string `json:"AniList"`
	} `json:"Description,omitempty"`
	MetaData    []MetaData        `json:"MetaData,omitempty"`
	InitialAge  int               `json:"Age"`
	Gender      string            `json:"Gender"`
	DateOfBirth AnimeDate         `json:"DateOfBirth"`
	Role        string            `json:"Role"`
	VoiceActor  []AnimeVoiceActor `json:"VoiceActor"`
}

type AnimeResource struct {
	Mal         int    `json:"Mal"`
	AniList     int    `json:"AniList"`
	AniDB       int    `json:"AniDB"`
	Kitsu       string `json:"Kitsu"`
	TVDBID      int64  `json:"TVDBMovieID,omitempty"`
	TMDBID      int64  `json:"TMDBID"`
	IMDBID      string `json:"IMDBID"`
	AniSearch   int64  `json:"AniSearch"`
	LiveChart   int64  `json:"LiveChart"`
	NotifyMoe   string `json:"NotifyMoe"`
	AnimePlanet string `json:"AnimePlanet"`
	WikiData    string `json:"WikiData"`
}

type AnimeVideo struct {
	Source   string   `json:"Source"`
	Referer  string   `json:"Referer,omitempty"`
	Type     string   `json:"Type"`
	Quality  string   `json:"Quality"`
	Language Language `json:"Language"`
}

type AnimeEpisodeResources struct {
	Mal         int    `json:"Mal"`
	AniDB       int    `json:"AniDB"`
	TVDBID      int64  `json:"TVDBID"`
	TMDBID      int64  `json:"TMDBID"`
	SimklID     int64  `json:"SimklID"`
	Crunchyroll string `json:"Crunchyroll"`
}

type AnimeEpisode struct {
	EnTitle        string                `json:"EnTitle,omitempty"`
	JpTitle        string                `json:"JpTitle,omitempty"`
	RmTitle        string                `json:"RmTitle,omitempty"`
	Aired          bool                  `json:"Aired"`
	ReleaseTime    time.Time             `json:"ReleaseTime"`
	UnixTime       int64                 `json:"UnixTime"`
	MetaData       []MetaData            `json:"MetaData"`
	Runtime        int                   `json:"Runtime"`
	Filler         bool                  `json:"Filler"`
	Special        bool                  `json:"Special"`
	SeasonNumber   int                   `json:"SeasonNumber"`
	AbsoluteNumber float32               `json:"AbsoluteNumber"`
	Number         float32               `json:"Number"`
	ThumbnailsIMG  AnimeImage            `json:"ThumbnailsIMG"`
	Resources      AnimeEpisodeResources `json:"Resources"`
}

type AnimeThemes struct {
	OP []struct {
		Song    string `json:"Song"`
		Entries []struct {
			Episodes []int `json:"Episodes"`
			Videos   []struct {
				Source     string `json:"Source"`
				Resolution int    `json:"Resolution"`
				Link       string `json:"Link"`
			} `json:"Videos"`
		} `json:"Entries"`
	} `json:"OP"`
	ED []struct {
		Song    string `json:"Song"`
		Entries []struct {
			Episodes []int `json:"Episodes"`
			Videos   []struct {
				Source     string `json:"Source"`
				Resolution int    `json:"Resolution"`
				Link       string `json:"Link"`
			} `json:"Videos"`
		} `json:"Entries"`
	} `json:"ED"`
}

type AnimeSeason struct {
	MetaData    []MetaData     `json:"MetaData"`
	PortraitIMG AnimeImage     `json:"PortraitIMG"`
	Number      int            `json:"Number"`
	TVDBID      int64          `json:"TVDBID"`
	TMDBID      int64          `json:"TMDBID"`
	StartAt     AnimeDate      `json:"StartAt"`
	EndAt       AnimeDate      `json:"EndAt"`
	Posters     []AnimeImage   `json:"Posters"`
	Trailers    []AnimeTrailer `json:"Trailers"`
}

type Anime struct {
	Type            string           `json:"Type"`
	Resources       AnimeResource    `json:"Resources"`
	Titles          AnimeTitles      `json:"Titles"`
	Description     string           `json:"Description,omitempty"`
	MetaData        []MetaData       `json:"MetaData"`
	PortraitIMG     AnimeImage       `json:"PortraitIMG"`
	LandscapeIMG    AnimeImage       `json:"LandscapeIMG"`
	Status          string           `json:"Status"`
	ContentRating   string           `json:"ContentRating"`
	CountryOfOrigin string           `json:"CountryOfOrigin"`
	Period          AnimePeriod      `json:"Period"`
	StartAt         AnimeDate        `json:"StartAt"`
	EndAt           AnimeDate        `json:"EndAt"`
	Studios         []AnimeCompany   `json:"Studios"`
	Genres          []AnimeGenre     `json:"Genres"`
	Producers       []AnimeCompany   `json:"Producers"`
	Licensors       []AnimeCompany   `json:"Licensors"`
	Tags            []AnimeTag       `json:"Tags"`
	Relations       []AnimeRelation  `json:"Relations"`
	Characters      []AnimeCharacter `json:"Characters"`
	InnerSeasons    []AnimeSeason    `json:"InnerSeasons,omitempty"`
	Episodes        []AnimeEpisode   `json:"Episodes,omitempty"`
	Trailers        []AnimeTrailer   `json:"Trailers"`
	External        []AnimeLink      `json:"External"`
	Themes          AnimeThemes      `json:"Themes"`
	Posters         []AnimeImage     `json:"Posters"`
	Backdrops       []AnimeImage     `json:"Backdrops"`
	Logos           []AnimeImage     `json:"Logos"`
	Banners         []AnimeImage     `json:"Banners"`
	Arts            []AnimeImage     `json:"Arts"`
}
