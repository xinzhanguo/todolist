package main

import (
	_ "embed"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/xinzhanguo/todolist/db"
	"gopkg.in/yaml.v2"
)

//go:embed home/index.html
var home string

//go:embed item/index.html
var item string

type Conf struct {
	dsn  string `yaml:"dsn"`
	port int    `yaml:"port"`
}

func main() {
	c := flag.String("c", "env.yaml", "conf")

	flag.Parse()
	var conf Conf
	cby, err := os.ReadFile(*c)
	if err != nil {
		log.Fatal(err)
	}
	if err := yaml.Unmarshal(cby, &conf); err != nil {
		log.Fatal(err)
	}
	dbclint, err := db.New(conf.dsn)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		encoded := "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/" +
			"9hAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAlUlEQVR4" +
			"nGNgGBRgVsWCc0sbF90nBc8um38ObsDKpkX3zy5c/p8" +
			"UDNJDfQNubd/2/8npo2AFDw7u/X91/UYw/ejYQTgG8XEa" +
			"cHPr1v8Xlq2mngseIdk8sC54AA0Pgi64tX0bZS44S240rm" +
			"ha+HRT18L/ILx30iIwhvFxia1sWvgUbsCCqgVPVzXO+0oKXl" +
			"A1H2HAgAIAg3T4GQQicf0AAAAASUVORK5CYII="
		decoded, _ := base64.RawStdEncoding.DecodeString(encoded)
		fmt.Fprint(w, string(decoded))
	})

	mux.HandleFunc("GET /todo/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, home)
	})

	mux.HandleFunc("GET /todo/new", func(w http.ResponseWriter, r *http.Request) {
		newUUID := uuid.New()
		http.Redirect(w, r, "/todo/"+newUUID.String(), http.StatusFound)
	})

	mux.HandleFunc("GET /todo/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			return
		}
		fmt.Println("Parsed UUID:", parsedUUID)
		fmt.Fprint(w, item)
	})

	mux.HandleFunc("GET /api/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			return
		}

		fmt.Println("GET Parsed UUID:", parsedUUID)
		content, err := dbclint.Query(id)
		if err != nil {
			return
		}
		if content == "" {
			content = `{"todo":[],"inProgress":[],"done":[],"trash":[]}`
		}
		fmt.Fprint(w, content)
	})

	mux.HandleFunc("POST /api/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			fmt.Fprint(w, "{\"code\":-1,\"msg\":\"uid err\"}")
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("io ReadAll:", err)

			fmt.Fprint(w, "{\"code\":-1,\"msg\":\"body err\"}")
			return
		}
		if err := dbclint.Save(id, string(body)); err != nil {
			fmt.Println("db save:", err)
			fmt.Fprint(w, "{\"code\":-1,\"msg\":\"db err\"}")
			return
		}
		fmt.Println("POST Parsed UUID:", parsedUUID)
		fmt.Fprint(w, "{\"code\":0,\"msg\":\"ok\"}")
	})

	http.ListenAndServe(fmt.Sprintf("localhost:%d", conf.port), mux)

}
