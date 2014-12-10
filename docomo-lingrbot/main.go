package main

import (
	"encoding/json"
	"fmt"
	"github.com/mattn/go-docomo"
	"github.com/mattn/go-lingr"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type config struct {
	Apikey string      `json:"apikey"`
	Addr   string      `json:"addr"`
	User   docomo.User `json:"user"`
}

func main() {
	f, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var cfg config
	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	c := docomo.NewClient(cfg.Apikey, cfg.User)

	nick := cfg.User.Nickname
	http.Handle("/assets/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			f, err := os.Open("index.html")
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			defer f.Close()
			w.Header().Set("Content-Type", "text/html")
			io.Copy(w, f)
			return
		}
		if r.URL.Path != "/lingr" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		status, err := lingr.DecodeStatus(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		for _, event := range status.Events {
			m := event.Message
			if m == nil {
				continue
			}
			if !strings.HasPrefix(m.Text, nick+":") {
				continue
			}
			text := m.Text[len(nick)+2:]
			res, err := c.Conversation(text)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(fmt.Sprintf("%s: %s", m.Nickname, res.Utt)))
		}
	})
	http.ListenAndServe(cfg.Addr, nil)
}
