// This package provides exponential backoff support for making HTTP requests.
//
// It uses the github.com/cenkalti/backoff algorithm.
//
// Network failures and HTTP 5xx status codes qualify for retries.
//
// HTTP calls that return HTTP 4xx status codes do not get retried.
//
// If the last HTTP request made does not result in a 2xx HTTP status code, an
// error is returned, together with the data.
//
// The backoff settings can be configured via the global package variable
// BackOffSettings.
//
// There are several utility methods that wrap the standard net/http package
// calls.
//
// Any function that takes no arguments and returns (*http.Response, error) can
// be retried using this library's Retry function.
//
// The methods in this library should be able to run concurrently in multiple
// go routines.
//
// Example Usage
//
// Consider this trivial HTTP GET request:
//
//  res, err := http.Get("http://www.google.com/robots.txt")
//
// This can be rewritten as follows, enabling automatic retries:
//
//  res, attempts, err := httpbackoff.Get("http://www.google.com/robots.txt")
//
// The variable attempts will store the number of http calls that were made
// (one plus the number of retries).
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

// Any non 2xx HTTP status code is considered a bad response code, and will
// result in a BadHttpResponseCode.
type BadHttpResponseCode struct {
	HttpResponseCode int
	Message          string
}

// Returns an error message for this bad HTTP response code
func (err BadHttpResponseCode) Error() string {
	return err.Message
}

// Retry is the core library method for retrying http calls.
//
// httpCall should be a function that performs the http operation, and returns
// (*http.Response, error). This will be retried using the
// github.com/cenkalti/backoff package, whereby network failures and/or 5xx
// HTTP status codes will qualify for retries. 4xx HTTP status codes will be
// considered permanent failures and not retried.
//
// Concurrent use of this library method is supported.
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

// Retry wrapper for http://golang.org/pkg/net/http/#Get where attempts is the number of http calls made (one plus number of retries).
func Get(url string) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return http.Get(url) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Head where attempts is the number of http calls made (one plus number of retries).
func Head(url string) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return http.Head(url) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Post where attempts is the number of http calls made (one plus number of retries).
func Post(url string, bodyType string, body io.Reader) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return http.Post(url, bodyType, body) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#PostForm where attempts is the number of http calls made (one plus number of retries).
func PostForm(url string, data url.Values) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return http.PostForm(url, data) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#ReadResponse where attempts is the number of http calls made (one plus number of retries).
func ReadResponse(r *bufio.Reader, req *http.Request) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return http.ReadResponse(r, req) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Do where attempts is the number of http calls made (one plus number of retries).
func ClientDo(c *http.Client, req *http.Request) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return c.Do(req) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Get where attempts is the number of http calls made (one plus number of retries).
func ClientGet(c *http.Client, url string) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return c.Get(url) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Head where attempts is the number of http calls made (one plus number of retries).
func ClientHead(c *http.Client, url string) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return c.Head(url) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Post where attempts is the number of http calls made (one plus number of retries).
func ClientPost(c *http.Client, url string, bodyType string, body io.Reader) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return c.Post(url, bodyType, body) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.PostForm where attempts is the number of http calls made (one plus number of retries).
func ClientPostForm(c *http.Client, url string, data url.Values) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return c.PostForm(url, data) })
}

// Retry wrapper for http://golang.org/pkg/net/http/#Transport.RoundTrip where attempts is the number of http calls made (one plus number of retries).
func RoundTrip(t *http.Transport, req *http.Request) (resp *http.Response, attempts int, err error) {
	return Retry(func() (*http.Response, error) { return t.RoundTrip(req) })
}
