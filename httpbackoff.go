package httpbackoff

import (
	"bufio"
	"github.com/cenkalti/backoff"
	D "github.com/tj/go-debug"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	// These defaults can be replaced by a library user at runtime...
	BackOffSettings *backoff.ExponentialBackOff = backoff.NewExponentialBackOff()

	// Used for logging based on DEBUG environment variable
	// See github.com/tj/go-debug
	debug = D.Debug("httpbackoff")
)

type BadHttpResponseCode struct {
	HttpResponseCode int
	Message          string
}

func (err BadHttpResponseCode) Error() string {
	return err.Message
}

func Retry(httpCall func() (resp *http.Response, err error)) (*http.Response, int, error) {
	var err error
	var response *http.Response
	attempts := 0
	doHttpCall := func() error {
		response, err = httpCall()
		attempts += 1
		if err != nil {
			return err
		}

		// now check if http response code is such that we should retry [500, 600)...
		if respCode := response.StatusCode; respCode/100 == 5 {
			return BadHttpResponseCode{
				HttpResponseCode: respCode,
				Message:          "(Intermittent) HTTP response code " + strconv.Itoa(respCode),
			}
		}

		return nil
	}

	// Make HTTP API calls using an exponential backoff algorithm...
	err = backoff.RetryNotify(doHttpCall, BackOffSettings, func(err error, wait time.Duration) {
		debug("Error: %s", err)
	})

	if err != nil {
		return response, attempts, err
	}

	// now check http response code is ok [200, 300)...
	if respCode := response.StatusCode; respCode/100 != 2 {
		err = BadHttpResponseCode{
			HttpResponseCode: respCode,
			Message:          "(Permanent) HTTP response code " + strconv.Itoa(respCode),
		}
	}
	return response, attempts, err
}

// Retry wrapper for https://golang.org/pkg/net/http/#Get
func Get(url string) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return http.Get(url) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Head
func Head(url string) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return http.Head(url) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Post
func Post(url string, bodyType string, body io.Reader) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return http.Post(url, bodyType, body) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#PostForm
func PostForm(url string, data url.Values) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return http.PostForm(url, data) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#ReadResponse
func ReadResponse(r *bufio.Reader, req *http.Request) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return http.ReadResponse(r, req) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Client.Do
func ClientDo(c *http.Client, req *http.Request) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return c.Do(req) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Client.Get
func ClientGet(c *http.Client, url string) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return c.Get(url) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Client.Head
func ClientHead(c *http.Client, url string) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return c.Head(url) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Client.Post
func ClientPost(c *http.Client, url string, bodyType string, body io.Reader) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return c.Post(url, bodyType, body) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Client.PostForm
func ClientPostForm(c *http.Client, url string, data url.Values) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return c.PostForm(url, data) })
}

// Retry wrapper for https://golang.org/pkg/net/http/#Transport.RoundTrip
func RoundTrip(t *http.Transport, req *http.Request) (*http.Response, int, error) {
	return Retry(func() (*http.Response, error) { return t.RoundTrip(req) })
}
