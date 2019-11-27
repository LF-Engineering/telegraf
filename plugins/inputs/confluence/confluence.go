package confluence

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/internal/tls"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/selfstat"
)

// measurements
const (
	spacePath   = "/rest/api/space"
	historyPath = "/rest/api/history"
	versionPath = "/rest/api/version"

	measurementSpace   = "confluence_space"
	measurementHistory = "confluence_history"
	measurementVersion = "confluence_version"
)

// Confluence plugin gathers information about various confluence objects
type Confluence struct {
	URL            string            `toml:"url"`
	Username       string            `toml:"username"`
	Password       string            `toml:"password"`
	// HTTP Timeout specified as a string - 3s, 1m, 1h
	HTTPTimeout    internal.Duration `toml:"http_timeout"`
	MaxConnections int               `toml:"max_connections"`

	tls.ClientConfig
	client *client

	//client *client
	RateLimit       selfstat.Stat
	RateLimitErrors selfstat.Stat
	RateRemaining   selfstat.Stat
	semaphore       chan struct{}
}

const sampleConfig = `
	## The confluence url
	url = "https://confluence.linuxfoundation.org"
	# username = "admin"
  	# password = "admin"

	## Set response_timeout
  	http_timeout = "5s"

	## Worker pool for confluence plugin only
  	## Empty this field will use default value 5
  	# max_connections = 5
`

// SampleConfig returns sample configuration for this plugin
func (c *Confluence) SampleConfig() string {
	return sampleConfig
}

// Description implements telegraf.Input interface
func (c *Confluence) Description() string {
	return "Gather confluence data."
}

// Gather implements telegraf.Input interface
func (c *Confluence) Gather(acc telegraf.Accumulator) error {
	if c.client == nil {
		client, err := c.newHTTPClient()
		if err != nil {
			return err
		}
		if err = c.initialize(client); err != nil {
			return err
		}
	}

	c.gatherSpacesData(acc)
	return nil
}

// newHTTPClient instantiates a new http client when called
func (c *Confluence) newHTTPClient() (*http.Client, error) {
	tlsCfg, err := c.ClientConfig.TLSConfig()
	if err != nil {
		return nil, fmt.Errorf("error parse confluence config[%s]: %v", c.URL, err)
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
			MaxIdleConns:    c.MaxConnections,
		},
		Timeout: c.HTTPTimeout.Duration,
	}, nil
}

// separate the client as dependency to use httptest Client for mocking
func (c *Confluence) initialize(client *http.Client) error {
	if c.MaxConnections <= 0 {
		c.MaxConnections = 5
	}

	c.semaphore = make(chan struct{})
	c.client = newClient(client, c.URL, c.Username, c.Password, c.MaxConnections)

	return c.client.init()
}

type space struct {
	Id         int32           `json:"id"`
	Key        string          `json:"key"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Links      spaceLinks      `json:"_links"`
	Expandable spaceExpandable `json:"_expandable"`
}

type spaceLinks struct {
	WebUI string `json:"webui"`
	Self  string `json:"self"`
}

type spaceExpandable struct {
	Metadata    string `json:"metadata"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
}

type spaceResponse struct {
	Results []space `json:"results"`
}

func (c *Confluence) gatherSpaceData(s space, acc telegraf.Accumulator) error {

	tags := map[string]string{}
	if s.Name == "" {
		return fmt.Errorf("error empty space name")
	}

	u, err := url.Parse(c.URL)
	if err != nil {
		return err
	}
	tags["source"] = u.Hostname()
	tags["port"] = u.Port()

	fields := make(map[string]interface{})
	fields["id"] = s.Id
	fields["key"] = s.Key
	fields["name"] = s.Name
	fields["type"] = s.Type
	fields["webui"] = s.Links.WebUI
	fields["self"] = s.Links.Self

	acc.AddFields(measurementSpace, fields, tags)
	return nil
}

func (c *Confluence) gatherSpacesData(acc telegraf.Accumulator) {
	spaceResp, err := c.client.getAllSpaces(context.Background())
	if err != nil {
		acc.AddError(err)
		return
	}
	// get each space's data
	for _, space := range spaceResp.Results {
		err = c.gatherSpaceData(space, acc)
		if err == nil {
			continue
		}
		acc.AddError(err)
	}
}

// wrap the tcp request with doGet
// block tcp request if buffered channel is full
func (c *Confluence) doGet(tcp func() error) error {
	c.semaphore <- struct{}{}
	if err := tcp(); err != nil {
		<-c.semaphore
		return err
	}
	<-c.semaphore
	return nil
}

func init() {
	inputs.Add("confluence", func() telegraf.Input {
		return &Confluence{
			HTTPTimeout:    internal.Duration{Duration: time.Duration(time.Hour)},
			MaxConnections: 5,
		}
	})
}
