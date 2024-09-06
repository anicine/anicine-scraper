package models

type Country struct {
	Name      string `json:"Name"`
	ShortName string `json:"ShortName"`
	ISO3166_1 string `json:"Iso3166_1"`
}

type Language struct {
	Name     string `json:"name"`
	ISO639_1 string `json:"iso639_1"`
}

type MetaData struct {
	Language Language `json:"Language"`
	Title    string   `json:"Title"`
	OverView string   `json:"OverView"`
}

type Video struct {
	Link     string `json:"link"`
	Referer  string `json:"referer"`
	Type     string `json:"type"`
	Quality  string `json:"quality"`
	Language string `json:"language"`
}
