# taskcluster-client.go
[![Build Status](https://secure.travis-ci.org/taskcluster/httpbackoff.png)](http://travis-ci.org/taskcluster/httpbackoff)
[![GoDoc](https://godoc.org/github.com/taskcluster/httpbackoff?status.png)](https://godoc.org/github.com/taskcluster/httpbackoff)

Automatic http retries for intermittent failures, with exponential backoff, based on https://github.com/cenkalti/backoff.

The reason for a separate library, is that this library handles http status codes to know whether to retry or not.
HTTP codes in range 500-599 are retried. Connection failures are also retried. Status codes 400-499 are considered permanent errors and are not retried.

The Retry function wraps any function that returns `(*http.Response, error)`.

For example, the following code that is not using retries:

```go
res, err := http.Get("http://www.google.com/robots.txt")
```

can be rewritten as:

```go
httpbackoff.Retry(func() (*http.Response, error) { return http.Get("http://www.google.com/robots.txt") })
```

The go http package has 11 functions that return `(*http.Reponse, error)`:

* https://golang.org/pkg/net/http/#Client.Do
* https://golang.org/pkg/net/http/#Client.Get
* https://golang.org/pkg/net/http/#Client.Head
* https://golang.org/pkg/net/http/#Client.Post
* https://golang.org/pkg/net/http/#Client.PostForm
* https://golang.org/pkg/net/http/#Get
* https://golang.org/pkg/net/http/#Head
* https://golang.org/pkg/net/http/#Post
* https://golang.org/pkg/net/http/#PostForm
* https://golang.org/pkg/net/http/#ReadResponse
* https://golang.org/pkg/net/http/#Transport.RoundTrip

To simplify using these functions, 11 utility functions have been written that wrap these. Therefore you can simplify this further with:

```go
res, err := httpbackoff.Get("http://www.google.com/robots.txt")
```

## Configuring backoff settings

Do something like this:

```go
import "github.com/cenkalti/backoff"

...
...

httpbackoff.BackOffSettings = &backoff.ExponentialBackOff{
	InitialInterval:     1 * time.Millisecond,
	RandomizationFactor: 0.2,
	Multiplier:          1.2,
	MaxInterval:         5 * time.Millisecond,
	MaxElapsedTime:      20 * time.Millisecond,
	Clock:               backoff.SystemClock,
}
```

Please note these are global settings to the module, so you can't concurrently use different backoff settings.

## Concurrency

As far as I am aware, there is nothing in this library that prevents it from being used concurrently. Please let me know if you find any problems!

## Contributing
Contributions are welcome. Please fork, and issue a Pull Request back with an explanation of your changes.
