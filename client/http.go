package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/anicine/anicine-scraper/internal/errs"
)

// reader copies the request body into a new buffer for potential reuse.
func reader(body io.Reader) io.Reader {
	if body == nil {
		return nil
	}

	var buf *bytes.Buffer
	switch t := body.(type) {
	case *bytes.Buffer:
		buf = t
	default:
		buf = new(bytes.Buffer)
		_, err := io.Copy(buf, body)
		if err != nil {
			logger.Error("cannot copying request body", "error", err)
			return nil
		}
	}

	return bytes.NewReader(buf.Bytes())
}

// request initializes a new HTTP request with the given arguments.
func request(args *Args) (*http.Request, error) {
	req, err := http.NewRequest(args.Method, args.Endpoint.String(), reader(args.Body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", UserAgent)
	for k, v := range args.Headers {
		req.Header.Add(k, v)
	}
	return req, nil
}

// redirect updates the endpoint if redirection is required.
func redirect(resp *http.Response, args *Args) error {
	newURL, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return errs.ErrNotFound
	}
	args.Endpoint = newURL
	return nil
}

// retry handles retries for specific status codes.
func retry(statusCode int, client **http.Client) time.Duration {
	switch statusCode {
	case http.StatusForbidden:
		if *client == http.DefaultClient {
			return 15 * time.Second
		}
		*client = http.DefaultClient
		return 0
	case http.StatusTooManyRequests:
		if *client == http.DefaultClient {
			return 1 * time.Minute
		}
		return 15 * time.Second
	}
	return 0
}

// clean handles closer for both the request and response bodies.
func clean(req *http.Request, resp *http.Response) {
	if req != nil && req.Body != nil {
		req.Body.Close()
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// agent handles which client should do the request.
func agent(retries int, args *Args) *http.Client {
	if retries == 0 {
		if args.Proxy {
			return proxy
		}
	} else if retries > 7 {
		return http.DefaultClient
	}
	return http.DefaultClient
}

// Do executes the HTTP request with retry and redirection handling.
func Do(ctx context.Context, args *Args) (io.Reader, error) {
	if args.Endpoint == nil {
		return nil, errs.ErrBadData
	}

	var (
		req    *http.Request
		resp   *http.Response
		client *http.Client
		err    error
	)

	// Retry loop for handling failures
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		default:
			client = agent(i, args)
			req, err = request(args)
			if err != nil {
				logger.Error("cannot create request", "link", args.Endpoint, "error", err)
				continue
			}

			resp, err = client.Do(req)
			if err != nil {
				logger.Error("cannot get response", "link", args.Endpoint, "error", err)
				continue
			}
			defer clean(req, resp)

			logger.Info("accepted response", "proxy", args.Proxy, "code", resp.StatusCode, "host", args.Endpoint.Host, "link", args.Endpoint.Path)

			switch resp.StatusCode {
			case http.StatusPreconditionFailed, http.StatusPreconditionRequired:
				return nil, errs.ErrBadData
			case http.StatusOK, http.StatusNotModified:
				args.cookies = resp.Cookies()
				if body := reader(resp.Body); body != nil {
					return body, nil
				}
				return nil, errs.ErrNoData
			case http.StatusNotFound, http.StatusBadRequest:
				return nil, errs.ErrNotFound
			case http.StatusFound, http.StatusMovedPermanently:
				if err = redirect(resp, args); err != nil {
					return nil, err
				}
				continue
			}
			if retryDuration := retry(resp.StatusCode, &client); retryDuration > 0 {
				time.Sleep(retryDuration)
				continue
			}
		}
	}

	logger.Error("failed to complete the operation", "link", args.Endpoint, "error", err)
	return nil, errs.ErrNoData
}
