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
//  res, attempts, err := httpbackoff.New().Get("http://www.google.com/robots.txt")
//
// The variable attempts stores the number of http calls that were made (one
// plus the number of retries).
package httpbackoff

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/cenkalti/backoff"
)

type ExponentialBackOff backoff.ExponentialBackOff

func New() *ExponentialBackOff {
	x := ExponentialBackOff(*backoff.NewExponentialBackOff())
	return &x
}

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
// (resp *http.Response, tempError error, permError error). Errors that should
// cause retries should be returned as tempError. Permanent errors that should
// not result in retries should be returned as permError. Retries are performed
// using the exponential backoff algorithm from the github.com/cenkalti/backoff
// package. Retry automatically treats HTTP 5xx status codes as a temporary
// error, and any other non-2xx HTTP status codes as a permanent error. Thus
// httpCall function does not need to handle the HTTP status code of resp,
// since Retry will take care of it.
//
// Concurrent use of this library method is supported.
func (backOffSettings *ExponentialBackOff) Retry(httpCall func() (resp *http.Response, tempError error, permError error)) (*http.Response, int, error) {
	var tempError, permError error
	var response *http.Response
	attempts := 0
	doHttpCall := func() error {
		response, tempError, permError = httpCall()
		attempts += 1
		if tempError != nil {
			return tempError
		}
		if permError != nil {
			return nil
		}
		// this is a no-op
		raw, readErr := httputil.DumpResponse(response, true)
		out := ""
		if readErr == nil {
			out = string(raw)
		}
		// now check if http response code is such that we should retry [500, 600)...
		if respCode := response.StatusCode; respCode/100 == 5 {
			return BadHttpResponseCode{
				HttpResponseCode: respCode,
				Message:          "(Intermittent) HTTP response code " + strconv.Itoa(respCode) + "\n" + out,
			}
		}
		// now check http response code is ok [200, 300)...
		if respCode := response.StatusCode; respCode/100 != 2 {
			permError = BadHttpResponseCode{
				HttpResponseCode: respCode,
				Message:          "(Permanent) HTTP response code " + strconv.Itoa(respCode) + "\n" + out,
			}
			return nil
		}
		return nil
	}

	// Make HTTP API calls using an exponential backoff algorithm...
	b := backoff.ExponentialBackOff(*backOffSettings)
	backoff.RetryNotify(doHttpCall, &b, func(err error, wait time.Duration) {
		log.Printf("Error: %s", err)
	})

	switch {
	case permError != nil:
		return response, attempts, permError
	case tempError != nil:
		return response, attempts, tempError
	default:
		return response, attempts, nil
	}
}

// Retry wrapper for http://golang.org/pkg/net/http/#Get where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) Get(url string) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := http.Get(url)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Head where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) Head(url string) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := http.Head(url)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Post where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) Post(url string, bodyType string, body io.Reader) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := http.Post(url, bodyType, body)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#PostForm where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) PostForm(url string, data url.Values) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := http.PostForm(url, data)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#ReadResponse where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) ReadResponse(r *bufio.Reader, req *http.Request) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := http.ReadResponse(r, req)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Do where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) ClientDo(c *http.Client, req *http.Request) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := c.Do(req)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Get where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) ClientGet(c *http.Client, url string) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := c.Get(url)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Head where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) ClientHead(c *http.Client, url string) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := c.Head(url)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.Post where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) ClientPost(c *http.Client, url string, bodyType string, body io.Reader) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := c.Post(url, bodyType, body)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Client.PostForm where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) ClientPostForm(c *http.Client, url string, data url.Values) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := c.PostForm(url, data)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}

// Retry wrapper for http://golang.org/pkg/net/http/#Transport.RoundTrip where attempts is the number of http calls made (one plus number of retries).
func (backOffSettings *ExponentialBackOff) RoundTrip(t *http.Transport, req *http.Request) (resp *http.Response, attempts int, err error) {
	return backOffSettings.Retry(func() (*http.Response, error, error) {
		resp, err := t.RoundTrip(req)
		// assume all errors should result in a retry
		return resp, err, nil
	})
}
