package worker

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"io"
	"net/http"
	"strconv"
)

type Client struct {
	AnalyticsUrl string
	GrafanaUrl   string
	Token        string
	Logger       log.Logger
	Filter       string
}

const contentTypeJson = "application/json"

func (api *Client) Get(endpoint string) ([]byte, error) {
	req, err := api.prepareRequest("GET", endpoint)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if res.StatusCode != http.StatusOK {
		errorMessage := fmt.Sprintf("Request failed\nEndpoint: %s\nStatus: %s", endpoint, strconv.Itoa(res.StatusCode))

		return nil, errors.New(errorMessage)
	}

	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (api *Client) Post(endpoint string, payload []byte) ([]byte, error) {
	req, err := api.prepareRequest("POST", endpoint)
	if err != nil {
		return nil, err
	}

	setContentType(req, contentTypeJson)
	addBodyToRequest(req, payload)

	client := &http.Client{}
	res, err := client.Do(req)
	if res.StatusCode != http.StatusOK {
		errorMessage := fmt.Sprintf("Request failed\nEndpoint: %s\nStatus: %s", endpoint, strconv.Itoa(res.StatusCode))

		return nil, errors.New(errorMessage)
	}
	defer res.Body.Close()

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (api *Client) prepareRequest(method string, endpoint string) (*http.Request, error) {
	req, err := http.NewRequest(method, api.GrafanaUrl+endpoint, nil)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create new HTTP Request\nMethod: %s\nEndpoint: %s", method, endpoint))
	}

	req.Header.Add("Authorization", "Bearer "+api.Token)
	req.Header.Add("Accept", contentTypeJson)

	return req, nil
}

func setContentType(req *http.Request, contentType string) {
	req.Header.Add("Content-Type", contentType)
}

func addBodyToRequest(req *http.Request, payload []byte) {
	req.Body = io.NopCloser(bytes.NewBuffer(payload))
	req.ContentLength = int64(len(payload))
}
