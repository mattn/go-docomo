package main

import (
	"encoding/json"
	"fmt"
	"github.com/mattn/go-docomo"
	"github.com/mattn/go-lingr"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

type config struct {
	Apikey string      `json:"apikey"`
	Addr   string      `json:"addr"`
	User   docomo.User `json:"user"`
}

var re = regexp.MustCompile(`^これ読んで\s+((?:http|https)://\S+)$`)

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
		if r.Method == "GET" {
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
			text := strings.TrimSpace(m.Text[len(nick)+2:])

			match := re.FindAllStringSubmatch(text, -1)
			if len(match) > 0 && len(match[0]) == 2 {
				u, err := url.Parse(match[0][1])
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				res, err := http.Get(u.String())
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				b, err := ioutil.ReadAll(res.Body)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				ct := res.Header.Get("Content-Type")
				ret, err := c.CharacterRecognition(ct, path.Base(u.Path), b)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				result := []string{}
				for _, word := range ret.Words.Word {
					if word.Text != "" {
						result = append(result, word.Text)
					}
				}
				if len(result) == 0 && ret.Message.Text != "" {
					result = append(result, ret.Message.Text)
				}
				if len(result) == 0 {
					w.Write([]byte(fmt.Sprintf("%s: わかりません", m.Nickname)))
				} else {
					w.Write([]byte(fmt.Sprintf("%s: %s", m.Nickname, strings.Join(result, ", "))))
				}
			} else {
				ret, err := c.Dialogue(text)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(fmt.Sprintf("%s: %s", m.Nickname, ret.Utt)))
			}
		}
	})
	http.ListenAndServe(cfg.Addr, nil)
}
