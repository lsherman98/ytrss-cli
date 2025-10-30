package api

import (
	"bytes"
	"encoding/json"

	"github.com/zalando/go-keyring"
)

const (
	BaseURL     = "http://ytrss.xyz/api/v1"
	serviceName = "ytrss-cli"
)

var apiClient = NewAPIClient(BaseURL)

type Podcast struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type AddUrlRequestBody struct {
	PodcastID string `json:"podcast_id"`
	URL       string `json:"url"`
}

type Job struct {
	Status  string `json:"status"`
	Title   string `json:"title,omitempty"`
	Created string `json:"created,omitempty"`
	Error   string `json:"error,omitempty"`
}

type UsageResponse struct {
	Usage int `json:"usage"`
	Limit int `json:"limit"`
}

type Item struct {
	Status  string `json:"status"`
	Title   string `json:"title,omitempty"`
	Error   string `json:"error,omitempty"`
	Created string `json:"created,omitempty"`
}

func GetApiKey() (string, error) {
	return keyring.Get(serviceName, "api_key")
}

func SetApiKey(apiKey string) error {
	return keyring.Set(serviceName, "api_key", apiKey)
}

func ClearApiKey() error {
	return keyring.Delete(serviceName, "api_key")
}

func ListPodcasts() ([]Podcast, error) {
	var podcasts []Podcast
	err := apiClient.do("GET", "/list-podcasts", nil, &podcasts)
	if err != nil {
		return nil, err
	}
	return podcasts, nil
}

func AddUrlToPodcast(podcastID, url string) (Item, error) {
	requestBody := AddUrlRequestBody{
		PodcastID: podcastID,
		URL:       url,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return Item{}, err
	}

	var item Item
	err = apiClient.do("POST", "/podcasts/add-url", bytes.NewBuffer(jsonBody), &item)
	if err != nil {
		return Item{}, err
	}

	return item, nil
}

func GetPodcastItems(podcastID string) ([]Item, error) {
	var items []Item
	err := apiClient.do("GET", "/get-items/"+podcastID, nil, &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func GetUsage() (*UsageResponse, error) {
	var usageResponse UsageResponse
	err := apiClient.do("GET", "/get-usage", nil, &usageResponse)
	if err != nil {
		return nil, err
	}
	return &usageResponse, nil
}
