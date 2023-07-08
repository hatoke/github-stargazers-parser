package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

var (
	outputFile *csv.Writer
	wg         sync.WaitGroup
	mu         sync.Mutex
)

func main() {
	csvTitles := []string{"Username", "Mail", "Country"}
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading environment variables file")
	}
	repoUrl := os.Getenv("REPO_URL")
	repoUrl = repoUrl + "/stargazers"

	createCsvFile()
	if err != nil {
		fmt.Println(err)
		return
	}
	writeNewRowCsvFile(csvTitles)
	visitStragazersPage(repoUrl)
}

func visitStragazersPage(repoUrl string) {
	git := colly.NewCollector()
	hasNextPage := false
	page := 1

	git.Limit(&colly.LimitRule{
		RandomDelay: 10 * time.Second,
	})

	git.OnHTML(".paginate-container .pagination", func(e *colly.HTMLElement) {
		if e != nil {
			hasNextPage = true
			page++
		} else {
			fmt.Println(e)
			hasNextPage = false
		}
	})

	git.OnHTML("ol li", func(e *colly.HTMLElement) {
		e.ForEach("h3 a[href]", func(_ int, a *colly.HTMLElement) {
			wg.Add(1)
			profileName := a.Attr("href")
			go visitProfile(profileName)
		})
	})

	git.OnError(func(r *colly.Response, err error) {
		fmt.Println(err)
	})

	git.OnScraped(func(r *colly.Response) {
		wg.Wait()
		if hasNextPage {
			fmt.Println("nex page")
			git.Visit(repoUrl + "?page=" + strconv.Itoa(page))
		}
	})

	git.Visit(repoUrl)
	git.Wait()
}

func visitProfile(profileUrl string) {
	var data []string

	gitProfile := colly.NewCollector()

	gitProfile.OnHTML(".p-name", func(e *colly.HTMLElement) {
		name := strings.TrimSpace(e.Text)
		data = append(data, name)
	})

	gitProfile.OnHTML(".vcard-details [itemprop=email]", func(e *colly.HTMLElement) {
		mail := strings.TrimSpace(e.Text)
		data = append(data, mail)
	})

	gitProfile.OnHTML(".vcard-details [itemprop=homeLocation]", func(e *colly.HTMLElement) {
		location := strings.TrimSpace(e.Text)
		data = append(data, location)
	})

	gitProfile.OnHTML(".vcard-details [itemprop=url]", func(e *colly.HTMLElement) {
		website := strings.TrimSpace(e.Text)
		data = append(data, website)
	})

	gitProfile.OnError(func(r *colly.Response, err error) {
		fmt.Println(err)
	})

	gitProfile.OnScraped(func(r *colly.Response) {
		fmt.Println("new data ", data)
		writeNewRowCsvFile(data)
		wg.Done()
	})

	gitProfile.Visit("https://github.com" + profileUrl)
}

func createCsvFile() (*os.File, error) {
	file, err := os.Create("output.csv")
	if err != nil {
		return nil, fmt.Errorf("an error occurred while creating the file: %v", err)
	}
	outputFile = csv.NewWriter(file)

	return file, nil
}

func writeNewRowCsvFile(row []string) {
	mu.Lock()
	outputFile.Write(row)
	outputFile.Flush()
	mu.Unlock()
}
