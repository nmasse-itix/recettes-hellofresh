package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/spf13/viper"
	"github.com/studio-b12/gowebdav"
)

type Scrapper struct {
	url             string
	c               *colly.Collector
	dav             *gowebdav.Client
	client          *http.Client
	davFolder       string
	davFolderFormat string
}

func NewScrapper(url, davUrl, davUsername, davPassword, davFolder, davFolderFormat string, httpTimeout time.Duration) (*Scrapper, error) {
	dav := gowebdav.NewClient(davUrl, davUsername, davPassword)
	scrapper := Scrapper{
		url:             url,
		c:               colly.NewCollector(),
		dav:             dav,
		davFolder:       davFolder,
		davFolderFormat: davFolderFormat,
		client: &http.Client{
			Timeout: httpTimeout,
		},
	}

	err := scrapper.dav.Connect()
	if err != nil {
		return nil, err
	}

	return &scrapper, nil
}

func (s *Scrapper) Scrape() []string {
	var url []string

	s.c.OnHTML("div[data-zest] a[href]", func(a *colly.HTMLElement) {
		href := a.Attr("href")
		content := strings.ToLower(a.Text)

		if !strings.HasPrefix(href, "//assets.ctfassets.net/") || !strings.Contains(content, "recette") {
			return
		}

		url = append(url, href)
	})

	s.c.Visit(s.url)

	return url
}

func (s *Scrapper) Download(u, filename string) error {
	davFolder := path.Join(s.davFolder, time.Now().Format(s.davFolderFormat))
	err := s.dav.MkdirAll(davFolder, 0755)
	if err != nil {
		return err
	}

	davFilePath := path.Join(davFolder, filename)
	_, err = s.dav.Stat(davFilePath)
	if err == nil {
		log.Printf("File %s already downloaded!", filename)
		return nil
	}

	resp, err := s.client.Get(u)
	if err != nil {
		return err
	}

	body := resp.Body
	defer body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Wrong status code: %d", resp.StatusCode)
	}

	err = s.dav.WriteStream(davFilePath, body, 0644)
	if err != nil {
		return err
	}

	log.Printf("Downloaded %s", filename)

	return nil
}

func initConfig() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s config.yaml\n", os.Args[0])
		os.Exit(1)
	}

	fd, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("open: %s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
	defer fd.Close()

	viper.SetConfigType("yaml")
	err = viper.ReadConfig(fd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, config := range []string{"Scrapper.URL", "WebDAV.URL", "WebDAV.Username", "WebDAV.Password", "WebDAV.Folder", "WebDAV.FolderFormat"} {
		if viper.GetString(config) == "" {
			fmt.Printf("key %s is missing from configuration file\n", config)
			os.Exit(1)
		}
	}
	viper.SetDefault("Scrapper.Timeout", 60*time.Second)
}

func main() {
	initConfig()

	scrapper, err := NewScrapper(viper.GetString("Scrapper.URL"), viper.GetString("WebDAV.URL"), viper.GetString("WebDAV.Username"), viper.GetString("WebDAV.Password"), viper.GetString("WebDAV.Folder"), viper.GetString("WebDAV.FolderFormat"), viper.GetDuration("Scrapper.Timeout"))
	if err != nil {
		log.Fatal(err)
	}

	urls := scrapper.Scrape()

	fail := false
	atLeastOne := false
	for _, u := range urls {
		parts, err := url.Parse(u)
		if err != nil {
			log.Printf("Cannot parse URL '%s': %s", u, err)
			continue
		}
		parts.Scheme = "https" // scheme is missing
		filename := path.Base(parts.Path)
		err = scrapper.Download(parts.String(), filename)
		if err != nil {
			fail = true
			log.Printf("Cannot download file '%s': %s", filename, err)
		}
		atLeastOne = true
	}

	if fail || !atLeastOne {
		os.Exit(1)
	}

	os.Exit(0)
}
