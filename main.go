package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/feeds"

	"github.com/dyatlov/go-htmlinfo/htmlinfo"
	_ "github.com/mattn/go-sqlite3"
)

// NewArticle is the accepted request body for a new article
type NewArticle struct {
	Url string
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World")
	})

	http.HandleFunc("/article", func(w http.ResponseWriter, r *http.Request) {
		article := NewArticle{}
		err := json.NewDecoder(r.Body).Decode(&article)

		if err != nil {
			http.Error(w, "", 400)
			fmt.Printf("An error occured")
			return
		}

		res, err := http.Get(article.Url)

		if err != nil {
			http.Error(w, "Error processing request", 500)
			fmt.Printf("Error requesting article from %s: %s", article.Url, err.Error())
			return
		}

		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)

		if err != nil {
			log.Fatal(err)
		}

		info := htmlinfo.NewHTMLInfo()
		reader := strings.NewReader(string(body))
		err = info.Parse(reader, nil, nil)

		fmt.Fprintln(w, info)

		db, err := sql.Open("sqlite3", "db/graveyard.db")

		if err != nil {
			panic(err)
		}

		createQuery := "create table if not exists article (id integer primary key autoincrement, created_at TEXT, url TEXT, raw_html TEXT, parsed TEXT)"

		_, err = db.Exec(createQuery)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}

		statement, err := db.Prepare("insert into article(created_at, url, raw_html, parsed) values (?,?,?,?)")

		if err != nil {
			panic(err)
		}

		createdAt := time.Now().Format("2006-01-02 03:04:05")
		fmt.Printf("Created at %s\n", createdAt)
		sqlRes, err := statement.Exec(createdAt, article.Url, string(body), info.String())

		numInserts, err := sqlRes.RowsAffected()

		if err != nil {
			panic(err)
		}

		fmt.Printf("Added %d to database\n", numInserts)
	})

	http.HandleFunc("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		db, err := sql.Open("sqlite3", "db/graveyard.db")

		if err != nil {
			panic(err)
		}

		createQuery := "select * from article"

		rows, err := db.Query(createQuery)

		var id int
		var url string
		var createdAt string
		var rawHtml string
		var parsed string

		feed := &feeds.Feed{
			Title:       "My feed",
			Link:        &feeds.Link{Href: "www.google.de"},
			Description: "My personal reading list",
			Created:     time.Now(),
		}

		feed.Items = []*feeds.Item{}

		for rows.Next() {
			err = rows.Scan(&id, &createdAt, &url, &rawHtml, &parsed)
			if err != nil {
				log.Fatal(err)
			}

			var info htmlinfo.HTMLInfo

			createdAt, err := time.Parse("2006-01-02 03:04:05", createdAt)
			if err != nil {
				log.Fatal(err)
			}

			json.NewDecoder(strings.NewReader(parsed)).Decode(&info)

			feed.Items = append(feed.Items, &feeds.Item{
				Id:    strconv.Itoa(id),
				Title: info.Title,
				Link: &feeds.Link{
					Href: url,
				},
				Description: info.MainContent,
				Created:     createdAt,
			})

		}

		rss, err := feed.ToRss()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintln(w, rss)

		rows.Close()

		if err != nil {
			log.Fatal(err)
			panic(err)
		}

	})

	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		panic(err)
	}
}

// add endpoint to post a url to
// --> add url to database
// --> add url to queue that crawles the website and fetches information for a preview
// add endpoint that exposes rss feed
