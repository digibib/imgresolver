package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net/http"
	"strconv"
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
		http.NotFound(w, r)
		return
	}

	resp, err := http.Post(srv.es, "application/json", bytes.NewBufferString(q))
	if err != nil {
		log.Println(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	var es esResponse
	if err := json.NewDecoder(resp.Body).Decode(&es); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if es.Hits.Total == 0 {
		notFound(w, r)
		return
	}

	for _, hit := range es.Hits.Hits {
		if imgURL := hit.Source.Image; imgURL != "" {
			imgBytes, err := http.Get(imgURL)
			if err != nil {
				log.Println(err.Error())
				notFound(w, r)
				return
			}
			io.Copy(w, imgBytes.Body)
			w.Header().Set("Content-Type", "image/jpeg")
			imgBytes.Body.Close()
			return
		}
	}

	notFound(w, r)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("notFoundImage") != "" {
		img := image.NewRGBA(image.Rect(0, 0, 100, 150))
		if rawColor := r.URL.Query().Get("color"); rawColor != "" {
			if len(rawColor) == 6 {
				red, green, blue := uint8(255), uint8(255), uint8(255)
				if c, err := strconv.ParseInt(rawColor[0:2], 16, 32); err == nil {
					red = uint8(c)
				}
				if c, err := strconv.ParseInt(rawColor[2:4], 16, 32); err == nil {
					green = uint8(c)
				}
				if c, err := strconv.ParseInt(rawColor[4:6], 16, 32); err == nil {
					blue = uint8(c)
				}
				clr := color.RGBA{red, green, blue, 255}
				draw.Draw(img, img.Bounds(), &image.Uniform{clr}, image.ZP, draw.Src)
			}
		}
		png.Encode(w, img)
		w.Header().Set("Content-Type", "image/png")
		return
	}
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func main() {
	httpAddr := flag.String("http", ":7001", "HTTP serve address")
	esAddr := flag.String("es", "http://localhost:9200", "Elasticsearch address")

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
