package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type client struct {
	baseURL       string
	httpClient    *http.Client
	username      string
	password      string
	sessionCookie *http.Cookie
	semaphore     chan struct{}
}

func newClient(httpClient *http.Client, url, username, password string, maxConnections int) *client {
	return &client{
		baseURL:    url,
		httpClient: httpClient,
		username:   username,
		password:   password,
		semaphore:  make(chan struct{}, maxConnections),
	}
}

func (c *client) init() error {
	// get the session cookie
	req, err := http.NewRequest("GET", c.baseURL, nil)
	if err != nil {
		return err
	}

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	for _, cookie := range res.Cookies() {
		if strings.Contains(cookie.Name, "JSESSIONID") {
			c.sessionCookie = cookie
			break
		}
	}

	// fetch
	if err := c.doGet(context.Background(), spacePath, new(spaceResponse)); err != nil {
		return err
	}
	return nil
}

func (c *client) doGet(ctx context.Context, url string, v interface{}) error {
	req, err := createGetRequest(c.baseURL+url, c.username, c.password, c.sessionCookie)
	if err != nil {
		return err
	}

	select {
	case c.semaphore <- struct{}{}:
		break
	case <-ctx.Done():
		return ctx.Err()
	}

	res, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		<-c.semaphore
		return err
	}
	defer func() {
		res.Body.Close()
		<-c.semaphore
	}()

	// Clear invalid token if unauthorized
	if res.StatusCode == http.StatusUnauthorized {
		c.sessionCookie = nil
		return APIError{
			URL:        url,
			StatusCode: res.StatusCode,
			Title:      res.Status,
		}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return APIError{
			URL:        url,
			StatusCode: res.StatusCode,
			Title:      res.Status,
		}
	}
	if res.StatusCode == http.StatusNoContent {
		return APIError{
			URL:        url,
			StatusCode: res.StatusCode,
			Title:      res.Status,
		}
	}
	if err = json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

type APIError struct {
	URL         string
	StatusCode  int
	Title       string
	Description string
}

func (e APIError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("[%s] %s: %s", e.URL, e.Title, e.Description)
	}
	return fmt.Sprintf("[%s] %s", e.URL, e.Title)
}

func createGetRequest(url, username, password string, sessionCookie *http.Cookie) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}
	if sessionCookie != nil {
		req.AddCookie(sessionCookie)
	}
	req.Header.Add("Accept", "application/json")
	return req, nil
}

func (c *client) getAllSpaces(ctx context.Context) (spaceResp *spaceResponse, err error) {
	spaceResp = new(spaceResponse)
	err = c.doGet(ctx, spacePath, spaceResp)
	return spaceResp, err
}
