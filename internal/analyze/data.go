package analyze

import "regexp"

const (
	MPAA1 = "G"
	MPAA2 = "PG"
	MPAA3 = "PG-13"
	MPAA4 = "R"
	MPAA5 = "NC-17"
	TVPG1 = "TV-G"
	TVPG2 = "TV-Y7"
	TVPG3 = "TV-PG"
	TVPG4 = "TV-14"
	TVPG5 = "TV-MA"
)

var (
	unicode = [12]string{
		"\u200b",
		"\u200d",
		"\u200e",
		"\u200f",
		"\u00ad",
		"\u200c",
		"\u180e",
		"\u202a",
		"\u202b",
		"\u202d",
		"\u202e",
		"\u00a0",
	}
	chars          = [40]string{"\\", "\"", "\t", "\n", "\f", "\r", "\a", "\v", "\b", ">", "<", "~", ".", ",", "`", "'", ":", ";", "|", "}", "{", "_", "*", "]", "[", "(", ")", "-", "+", "/", "「", "」", "!", "?", "@", "#", "$", "%", "^", "&"}
	seasonTitleExp = regexp.MustCompile(`(?i)\bseason\-\d+\b`)
	ytExp          = []*regexp.Regexp{
		regexp.MustCompile(`(?:youtube\.com\/(?:[^\/\n\s]+\/\S+\/|(?:v|e(?:mbed)?)\/|\S*?[?&]v=)|youtu\.be\/)([a-zA-Z0-9_-]{11})`),
		regexp.MustCompile(`(?:youtube\.com|youtu\.?be)\/watch\?v=([a-zA-Z0-9_\-]+)(&.+)?$`),
		regexp.MustCompile(`(?:youtu\.?be)\/([a-zA-Z0-9_\-]+)$`),
		regexp.MustCompile(`(?:youtu\.?be)\/embed\/([a-zA-Z0-9_\-]+)$`),
		regexp.MustCompile(`^(https?:\/\/)?(www\.)?youtu\.?be\.com\/share\/([a-zA-Z0-9_\-]+)$`),
	}
	overviewExp = regexp.MustCompile(`(?:\[.*?\]|\(.*?\))|(<.*?>)`)
	lineExp     = regexp.MustCompile(`\n`)
	crExp       = regexp.MustCompile(`([A-Z]+-[0-9]+(?:\+)?).*?(\(.*?\))?`)
	aidExp      = regexp.MustCompile(`(?i)aid=(\d+)`)
	yearExp     = regexp.MustCompile(`(\d{4})`)
	numExp      = regexp.MustCompile(`(\d+)`)
	intsExp     = regexp.MustCompile(`(\d+)-(\d+)|(\d+)`)
	linksExp    = regexp.MustCompile(`\[([^\]]+)]\(([^)]+)\)`)
	engExp      = regexp.MustCompile(`[^a-zA-Z0-9\s-]+`)
	pathExp     = regexp.MustCompile(`anime\/([^\/]+)\/`)
)
