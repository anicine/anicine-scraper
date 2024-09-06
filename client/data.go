package client

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

const UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

var (
	logger = slog.Default().WithGroup("[HTTP]")
	proxy  = http.DefaultClient
)

func SetProxy(client *http.Client) {
	if client != nil {
		proxy = client
	}
}

type Args struct {
	Proxy    bool
	Method   string
	Endpoint *url.URL
	Headers  map[string]string
	Body     io.Reader
	cookies  []*http.Cookie
}

// return the cookies of the response after the Args get passed to the 'Do' function.
func (x *Args) Cookies() []*http.Cookie {
	return x.cookies
}
