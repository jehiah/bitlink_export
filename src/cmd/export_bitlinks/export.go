package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Fetcher struct {
	endpoint    string
	accessToken string
	offset      int64
	err         error
	links       []*Bitlink
}

type Bitlink struct {
	Bitlink string `json:"link"`
	URL     string `json:"long_url"`
	Title   string `json:"title"`
	Notes   string `json:"notes"`
	Created int64  `json:"created_at"`
}

func (b *Bitlink) CSV() (c []string) {
	c = append(c, b.Bitlink, b.URL, b.Title, b.Notes)
	c = append(c, time.Unix(b.Created, 0).Format(time.RFC3339), fmt.Sprintf("%d", b.Created))
	return
}

type ApiResponse struct {
	StatusCode int    `json:"status_code"`
	StatusTxt  string `json:"status_txt"`
	Data       struct {
		LinkHistory []*Bitlink `json:"link_history"`
	} `json:"data"`
}

func (f *Fetcher) Fetch() (ok bool) {
	params := url.Values{
		"access_token": []string{f.accessToken},
		"limit":        []string{"100"},
		"offset":       []string{fmt.Sprintf("%d", f.offset)},
		"archived":     []string{"both"},
		"private":      []string{"both"},
	}

	var req *http.Request
	req, f.err = http.NewRequest("GET", f.endpoint+"/v3/user/link_history?"+params.Encode(), nil)
	if f.err != nil {
		return
	}
	var resp *http.Response
	resp, f.err = http.DefaultClient.Do(req)
	if f.err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		f.err = fmt.Errorf("got HTTP %d", resp.StatusCode)
		return
	}
	var body []byte
	body, f.err = ioutil.ReadAll(resp.Body)
	if f.err != nil {
		return
	}
	var data ApiResponse
	f.err = json.Unmarshal(body, &data)
	if f.err != nil {
		return
	}
	if data.StatusCode != 200 {
		f.err = fmt.Errorf("got API response %d %s", data.StatusCode, data.StatusTxt)
		return
	}
	f.links = data.Data.LinkHistory
	if len(f.links) > 0 {
		return true
	}
	return false
}

func (f *Fetcher) Bitlinks() []*Bitlink {
	return f.links
}
func (f *Fetcher) Error() error {
	return f.err
}

func main() {
	accessToken := flag.String("access-token", "", "Bitly OAuth Access Token - https://bitly.com/a/oauth_apps")
	endpoint := flag.String("api", "https://api-ssl.bitly.com", "Bitly API Endpoint")
	flag.Parse()

	if *accessToken == "" {
		log.Fatalf("-access-token required")
	}

	fetcher := &Fetcher{
		accessToken: *accessToken,
		endpoint:    *endpoint,
	}

	output := csv.NewWriter(os.Stdout)
	output.Write([]string{"bitlink", "long_url", "title", "notes", "created", "created_ts"})
	for fetcher.Fetch() {
		for _, l := range fetcher.Bitlinks() {
			output.Write(l.CSV())
		}
	}
	if err := fetcher.Error(); err != nil {
		log.Printf("Error: %s", err)
	}

}
