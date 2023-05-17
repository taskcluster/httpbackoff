package httpbackoff

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
)

var (
	err        error
	handler    *MyHandler
	listener   net.Listener
	server     *http.Server
	testClient *Client
)

// Handler for stubbing http requests from auth API endpoint
type MyHandler struct {
	QueuedResponses []HTTPResponse
}

type HTTPResponse struct {
	body       string
	statusCode int
}

// Fake auth endpoint
func (handler *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(handler.QueuedResponses) == 0 {
		return
	}
	response := handler.QueuedResponses[0]
	handler.QueuedResponses = handler.QueuedResponses[1:]
	w.WriteHeader(response.statusCode)
	_, _ = io.WriteString(w, response.body)
}

func (handler *MyHandler) QueueResponse(statusCode int) {
	body := fmt.Sprintf("This is a *fake* HTTP %d response body for testing purposes", statusCode)
	handler.QueuedResponses = append(handler.QueuedResponses, HTTPResponse{body: body, statusCode: statusCode})
}

func (handler *MyHandler) ClearResponseQueue() {
	handler.QueuedResponses = make([]HTTPResponse, 0)
}

func NewMyHandler() *MyHandler {
	return &MyHandler{QueuedResponses: make([]HTTPResponse, 0)}
}

// Custom setup and tear down
func TestMain(m *testing.M) {

	// Set up appropriate backoff settings to tests run quickly...
	testClient = &Client{
		BackOffSettings: &backoff.ExponentialBackOff{
			InitialInterval:     1 * time.Millisecond,
			RandomizationFactor: 0.2,
			Multiplier:          1.2,
			MaxInterval:         5 * time.Millisecond,
			MaxElapsedTime:      20 * time.Millisecond,
			Clock:               backoff.SystemClock,
		},
	}

	err = startServingHTTP()
	if err != nil {
		fmt.Println("ERROR: unable to serve HTTP for stub auth service!")
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// Start the listener...
	go func() {
		_ = server.Serve(listener)
	}()

	// Run all the tests...
	returnCode := m.Run()

	// Tear down
	err = stopServingHTTP()
	if err != nil {
		fmt.Println("ERROR: Failed to shut down http server")
		fmt.Printf("%v\n", err)
	}
	os.Exit(returnCode)
}

// start up http service
func startServingHTTP() error {
	handler = NewMyHandler()
	server = &http.Server{Addr: "localhost:50849", Handler: handler}
	listener, err = net.Listen("tcp", "localhost:50849")
	return err
}

// bring down http service
func stopServingHTTP() error {
	server.SetKeepAlivesEnabled(false)
	return listener.Close()
}
