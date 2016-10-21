package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

const (
	msgT = "@here そろそろお昼にしませんか？今日のオススメは\n```{{ range . }}{{ .Description }}\n{{ end }}```"
)

var messageTemplate *template.Template

func init() {
	rand.Seed(time.Now().UnixNano())
	messageTemplate = template.Must(template.New("message").Parse(msgT))
}

type Shop struct {
	Description string
}

func shopList(r io.Reader) ([]Shop, error) {
	var list []Shop
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		list = append(list, Shop{Description: scanner.Text()})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func recommend(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	l, err := shopList(f)
	if err != nil {
		return "", err
	}
	if len(l) == 0 {
		return ":thinking_face:", nil
	}
	c := 1
	if len(l) > 10 {
		c = 3
	}
	s := map[int]Shop{}
	for i := 0; i < 100 && len(s) < c; i++ {
		x := rand.Int() % len(l)
		s[x] = l[x]
	}
	var b bytes.Buffer
	if err := messageTemplate.Execute(&b, s); err != nil {
		return "", err
	}
	return b.String(), nil
}

type Payload struct {
	Channel   string `json:"channel"`
	UserName  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
	LinkNames int    `json:"link_names,omitempty"`
}

func post(url string, p Payload) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = http.Post(url, "application/json", bytes.NewReader(b))
	return err
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: command <shop_list> <webhook_url>\n")
}

type Config struct {
	WebHookUrl   string `json:"webhook_url"`
	UserName     string `json:"user_name"`
	Channel      string `json:"channel"`
	ShopListPath string `json:"shop_list"`
	IconEmoji    string `json:"icon_emoji"`
}

func config(file string) (*Config, error) {
	var c Config
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return &c, err
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return &c, err
	}
	if !strings.HasPrefix(c.WebHookUrl, "https") {
		return &c, fmt.Errorf("invalid webhook url:%v", c.ShopListPath)
	}

	return &c, err
}

func main() {
	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}
	c, err := config(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	msg, err := recommend(c.ShopListPath)
	if err != nil {
		log.Fatal(err)
	}
	err = post(c.WebHookUrl, Payload{
		Channel:   c.Channel,
		UserName:  c.UserName,
		Text:      msg,
		IconEmoji: c.IconEmoji,
		LinkNames: 1,
	})
	if err != nil {
		log.Printf("posting error: %v", err)
		return
	}
}
