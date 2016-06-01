package mailchimp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Client manages communication with the Mailchimp API.
type Client struct {
	client  *http.Client
	baseURL *url.URL
	dc      string
	apiKey  string
}

// ErrorResponse ...
type ErrorResponse struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// Error ...
func (e ErrorResponse) Error() string {
	return fmt.Sprintf("Error %d %s (%s)", e.Status, e.Title, e.Detail)
}

// NewClient returns a new Mailchimp API client.  If a nil httpClient is
// provided, http.DefaultClient will be used. The apiKey must be in the format xyz-us11.
func NewClient(apiKey string, httpClient *http.Client) (ClientInterface, error) {
	if len(strings.Split(apiKey, "-")) != 2 {
		return nil, errors.New("Mailchimp API Key must be formatted like: xyz-zys")
	}
	dc := strings.Split(apiKey, "-")[1] // data center
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, err := url.Parse(fmt.Sprintf("https://%s.api.mailchimp.com/3.0", dc))
	if err != nil {
		return nil, err
	}
	return &Client{
		client:  httpClient,
		baseURL: baseURL,
		apiKey:  apiKey,
		dc:      dc,
	}, nil
}

// GetBaseURL ...
func (c *Client) GetBaseURL() *url.URL {
	return c.baseURL
}

// SetBaseURL ...
func (c *Client) SetBaseURL(baseURL *url.URL) {
	c.baseURL = baseURL
}

// Subscribe ...
func (c *Client) Subscribe(email string, listID string) (interface{}, error) {
	data := &map[string]string{
		"email_address": email,
		"status":        "subscribed",
	}
	return c.do(
		"POST",
		fmt.Sprintf("/lists/%s/members/", listID),
		data,
	)
}

func (c *Client) do(method string, path string, body interface{}) (interface{}, error) {
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	apiURL := fmt.Sprintf("%s%s", c.GetBaseURL(), path)

	req, err := http.NewRequest(method, apiURL, buf)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var v interface{}
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func checkResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := new(ErrorResponse)
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse)
	}
	return errorResponse
}