package payload_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/MacroPower/macropower-analytics-panel/server/cacher"
	server "github.com/MacroPower/macropower-analytics-panel/server/payload"
	"github.com/go-kit/kit/log"
)

var (
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	cache  = cacher.NewCache(10, logger)
)

func SendPayload(t *testing.T, url string, r interface{}) {
	requestByte, _ := json.Marshal(r)
	requestReader := bytes.NewReader(requestByte)

	resp, err := http.Post(url, "application/json", requestReader)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("Received non-201 response: %d\n", resp.StatusCode)
	}

	expected := ""
	actual, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if expected != string(actual) {
		t.Errorf("Expected the message '%s'\n", expected)
	}
}

func TestHandler(t *testing.T) {
	handler := server.NewHandler(cache, false, false, logger)
	testserver := httptest.NewServer(handler)
	defer testserver.Close()

	request := server.Payload{}
	request.Type = "start"
	SendPayload(t, testserver.URL, request)
	time.Sleep(100 * time.Millisecond)
}

func TestPayloadLifecycle(t *testing.T) {
	handler := server.NewHandler(cache, false, false, logger)
	testserver := httptest.NewServer(handler)
	defer testserver.Close()

	var request server.Payload

	request = server.Payload{}
	request.UUID = "test"
	request.Type = "start"
	request.TimeOrigin = 1600000000
	SendPayload(t, testserver.URL, request)

	request = server.Payload{}
	request.UUID = "test"
	request.Type = "end"
	request.Time = 1600007200
	SendPayload(t, testserver.URL, request)

	time.Sleep(100 * time.Millisecond)

	p1, exists := cache.Get("test")
	if !exists {
		t.Fatal("Expected cache to contain item for payload")
	}
	p := p1.(server.Payload)
	actual := p.GetDuration(time.Duration(0))
	expected := 2 * time.Hour
	if expected != actual {
		t.Errorf("Expected the duration '%s', got '%s'\n", expected.String(), actual.String())
	}
}

func TestPayloadHeartbeat(t *testing.T) {
	handler := server.NewHandler(cache, false, false, logger)
	testserver := httptest.NewServer(handler)
	defer testserver.Close()

	var request server.Payload

	request = server.Payload{}
	request.UUID = "test"
	request.Type = "heartbeat"
	request.Time = 1600000001
	SendPayload(t, testserver.URL, request)

	request = server.Payload{}
	request.UUID = "test"
	request.Type = "heartbeat"
	request.Time = 1600000000
	SendPayload(t, testserver.URL, request)

	request = server.Payload{}
	request.UUID = "test"
	request.Type = "heartbeat"
	request.Time = 1600007200
	SendPayload(t, testserver.URL, request)

	time.Sleep(100 * time.Millisecond)

	p1, exists := cache.Get("test")
	if !exists {
		t.Fatal("Expected cache to contain item for payload")
	}
	p := p1.(server.Payload)
	actual := p.GetDuration(time.Hour)
	expected := time.Hour + time.Second
	if expected != actual {
		t.Errorf("Expected the duration '%s', got '%s'\n", expected.String(), actual.String())
	}
}
