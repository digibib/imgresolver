package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	query = `{
	"_source": ["image"],
	"query": {
		"term": {
			"%s": "%s"
		}
	}
}`
)

type resolver struct {
	es string
}

func (srv resolver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(r.URL.Path, "/")

	if len(paths) != 3 {
		http.NotFound(w, r)
		return
	}

	var q string

	switch strings.ToLower(paths[1]) {
	//case "work", "workuri", "workid":
	//case "publication", "publicationuri", "publicationid", "pub", "puburi", "pubid"
	case "isbn":
		q = fmt.Sprintf(query, "isbn", paths[2])
	case "recordid", "tnr", "titlenr", "biblionr", "biblionumber":
		q = fmt.Sprintf(query, "recordId", paths[2])
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	resp, err := http.Post(srv.es, "application/json", bytes.NewBufferString(q))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	var es esResponse
	if err := json.NewDecoder(resp.Body).Decode(&es); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if es.Hits.Total == 0 {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		// TODO or 1-pixel invisible gif
		return
	}

	for _, hit := range es.Hits.Hits {
		if imgURL := hit.Source.Image; imgURL != "" {
			imgBytes, err := http.Get(imgURL)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				// TODO or 1-pixel invisible gif
			}
			io.Copy(w, imgBytes.Body)
			w.Header().Set("Content-Type", "image/jpeg")
			imgBytes.Body.Close()
			return
		}
	}

	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	// TODO or 1-pixel invisible gif
}

func main() {
	httpAddr := flag.String("http", ":7001", "HTTP serve address")
	esAddr := flag.String("es", "http://elasticsearch:9200", "Elasticsearch address")

	flag.Parse()

	log.Fatal(http.ListenAndServe(*httpAddr, resolver{es: *esAddr + "/search/publication/_search"}))
}

type esResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total int `json:"total"`
		Hits  []struct {
			Index  string `json:"_index"`
			Type   string `json:"_type"`
			ID     string `json:"_id"`
			Parent string `json:"_parent"`
			Source struct {
				Image string `json:"image"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
