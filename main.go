package main

import (
	_ "embed"
	"encoding/json"
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
	Dsn  string `yaml:"dsn"`
	Port int    `yaml:"port"`
}

func getCookie(r *http.Request, key string) string {
	mlcokie, _ := r.Cookie(key)
	if mlcokie != nil {
		return mlcokie.Value
	}
	return ""
}

func setCookieHandler(w http.ResponseWriter, r *http.Request) {
	mlcokie, _ := r.Cookie("ml_token")
	token := ""
	if mlcokie != nil {
		token = mlcokie.Value
	}
	_, err := uuid.Parse(token)
	if err != nil {
		fmt.Println("Error parsing UUID:", err)
		token = ""
	}
	if token == "" {
		token = uuid.New().String()
	}
	cookie := &http.Cookie{
		Name:  "ml_token",
		Value: token,
		Path:  "/",
	}
	http.SetCookie(w, cookie)
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
	dbclint, err := db.New(conf.Dsn)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /setcookie", func(w http.ResponseWriter, r *http.Request) {
		nuid := uuid.New()
		cookie := &http.Cookie{
			Name:  "ml_token",
			Value: nuid.String(),
			Path:  "/",
		}
		http.SetCookie(w, cookie)
		fmt.Fprintln(w, "Cookie has been set")
	})

	mux.HandleFunc("GET /todo/", func(w http.ResponseWriter, r *http.Request) {
		setCookieHandler(w, r)
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
		setCookieHandler(w, r)
		fmt.Fprint(w, item)
	})

	mux.HandleFunc("GET /api/todo/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			return
		}
		fmt.Println("GET Parsed UUID:", parsedUUID)
		token := getCookie(r, "ml_token")
		key := r.Header.Get("Ml-Key")
		code := r.Header.Get("Ml-Code")
		fmt.Println(token, key, code)
		data, err := dbclint.GetAllowed(db.Data{UID: id, Token: token, Key: key, Code: code})
		if err != nil {
			fmt.Println(err)
			if err.Error() == db.NEEDCODE {
				fmt.Fprint(w, `{"code":-2,"msg":"need code or key"}`)
				return
			}
			if err.Error() == db.NEEDKEY {
				fmt.Fprint(w, `{"code":-3,"msg":"need key"}`)
				return
			}
			fmt.Fprint(w, `{"code":-1,"msg":"db err"}`)
			return
		}
		if data.Content == "" {
			data.Content = `{"todo":[],"inProgress":[],"done":[],"trash":[],"setting":[]}`
		}
		b, _ := json.Marshal(data)
		fmt.Fprint(w, string(b))
	})

	mux.HandleFunc("POST /api/todo/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"uid err"}`)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("io ReadAll:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"body err"}`)
			return
		}
		token := getCookie(r, "ml_token")
		key := r.Header.Get("ML_KEY")
		code := r.Header.Get("ML_CODE")
		if err := dbclint.SaveOrUpdate(db.Data{UID: id, Content: string(body), Token: token, Key: key, Code: code}); err != nil {
			fmt.Println("db save:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"db err"}`)
			return
		}
		fmt.Println("POST Parsed UUID:", parsedUUID)
		fmt.Fprint(w, `{"code":0,"msg":"ok"}`)
	})

	mux.HandleFunc("POST /api/key/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"uid err"}`)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("io ReadAll:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"body err"}`)
			return
		}
		token := getCookie(r, "ml_token")
		key := r.Header.Get("ML_KEY")
		code := r.Header.Get("ML_CODE")
		if err := dbclint.SetKey(db.Data{UID: id, Token: token, Key: key, Code: code}, string(body)); err != nil {
			fmt.Println("db save:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"db err"}`)
			return
		}
		fmt.Println("POST Parsed UUID:", parsedUUID)
		fmt.Fprint(w, `{"code":0,"msg":"ok"}`)
	})
	mux.HandleFunc("POST /api/code/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"uid err"}`)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("io ReadAll:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"body err"}`)
			return
		}
		token := getCookie(r, "ml_token")
		key := r.Header.Get("ML_KEY")
		code := r.Header.Get("ML_CODE")
		if err := dbclint.SetCode(db.Data{UID: id, Token: token, Key: key, Code: code}, string(body)); err != nil {
			fmt.Println("db save:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"db err"}`)
			return
		}
		fmt.Println("POST Parsed UUID:", parsedUUID)
		fmt.Fprint(w, `{"code":0,"msg":"ok"}`)
	})
	mux.HandleFunc("POST /api/style/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// 解析 UUID 字符串
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing UUID:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"uid err"}`)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("io ReadAll:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"body err"}`)
			return
		}
		token := getCookie(r, "ml_token")
		key := r.Header.Get("ML_KEY")
		code := r.Header.Get("ML_CODE")
		if err := dbclint.SetStyle(db.Data{UID: id, Token: token, Key: key, Code: code}, string(body)); err != nil {
			fmt.Println("db save:", err)
			fmt.Fprint(w, `{"code":-1,"msg":"db err"}`)
			return
		}
		fmt.Println("POST Parsed UUID:", parsedUUID)
		fmt.Fprint(w, `{"code":0,"msg":"ok"}`)
	})
	fmt.Println("0.0.0.0:", conf.Port)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", conf.Port), mux)

}
