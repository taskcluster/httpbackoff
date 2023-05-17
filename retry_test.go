package httpbackoff

// See test_setup_test.go for test setup...

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Stub server to send three 5xx failure status code responses
// before finally sending a 200 resp. Make sure the retry
// library retries until it gets the 200 resp.
func TestRetry5xx(t *testing.T) {

	handler.QueueResponse(500)
	handler.QueueResponse(501)
	handler.QueueResponse(502)
	handler.QueueResponse(200)
	handler.QueueResponse(502)

	// defer clean up in case we have t.Fatalf calls
	defer handler.ClearResponseQueue()

	resp, _, err := testClient.Get("http://localhost:50849/TestRetry5xx")

	if err != nil {
		t.Fatalf("%v\n", err)
	}

	if statusCode := resp.StatusCode; statusCode != 200 {
		t.Errorf("API retry logic broken - expected response code 200, but received code %v...\n", statusCode)
	}
}

// Check that when retries run out, the last temporary
// error is returned, even if htat was a 500.
func TestRetry5xxAndFail(t *testing.T) {

	testClient.BackOffSettings.InitialInterval = 10 * time.Millisecond

	handler.QueueResponse(500)
	handler.QueueResponse(500)
	handler.QueueResponse(500)
	handler.QueueResponse(500)
	handler.QueueResponse(500)

	// defer clean up in case we have t.Fatalf calls
	defer handler.ClearResponseQueue()

	resp, _, err := testClient.Get("http://localhost:50849/TestRetry5xx")

	if assert.Error(t, err) {
		herr, ok := err.(BadHttpResponseCode)
		require.True(t, ok)
		require.Equal(t, herr.HttpResponseCode, 500)
	}

	if statusCode := resp.StatusCode; statusCode != 500 {
		t.Errorf("API retry logic broken - expected response code 500, but received code %v...\n", statusCode)
	}
}

// Want to make sure 4xx errors (except 429) are not retried...
func TestRetry4xx(t *testing.T) {
	handler.QueueResponse(409)
	handler.QueueResponse(200)

	// defer clean up in case we have t.Fatalf calls
	defer handler.ClearResponseQueue()

	resp, _, err := testClient.Get("http://localhost:50849/TestRetry4xx")

	// NB: this is == not != since we *want* an error
	if err == nil {
		t.Errorf("Was expecting Get call to return an error, due to 409 status code\n")
	}

	if statusCode := resp.StatusCode; statusCode != 409 {
		t.Errorf("API retry logic broken - expected response code 409, but received code %v...\n", statusCode)
	}
}

// Want to make sure 429 is retried...
func TestRetry429(t *testing.T) {
	handler.QueueResponse(429)
	handler.QueueResponse(200)

	// defer clean up in case we have t.Fatalf calls
	defer handler.ClearResponseQueue()

	resp, attempts, err := testClient.Get("http://localhost:50849/TestRetry429")

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	if statusCode := resp.StatusCode; statusCode != 200 {
		t.Errorf("API retry logic broken - expected response code 200, but received code %v...\n", statusCode)
	}

	if attempts != 2 {
		t.Errorf("Was expecting 2 retry attempts, but had %v...\n", attempts)
	}
}

// Test network failures get retried
func TestNetworkFailure(t *testing.T) {

	// bad port
	_, attempts, err := testClient.Get("http://localhost:40849/TestNetworkFailure")

	// NB: this is == not != since we *want* an error
	if err == nil {
		t.Errorf("Was expecting Get call to return an error, due to 409 status code\n")
	}

	if attempts < 4 {
		t.Errorf("Was expecting at least 4 retry attempts, but were only %v...\n", attempts)
	}
}
