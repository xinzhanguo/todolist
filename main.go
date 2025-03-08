package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/xinzhanguo/todolist/db"
	"gopkg.in/yaml.v2"
)

var (
	//go:embed home/index.html
	homeHtml string
	//go:embed item/index.html
	itemHtml string
	//go:embed chat/index.html
	chatHtml string
)

type Config struct {
	Dsn  string `yaml:"dsn"`
	Port int    `yaml:"port"`
}

type APIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

const (
	TokenCookieName = "ml_token"
	HeaderKey       = "Ml-Key"
	HeaderCode      = "Ml-Code"
	HeaderVersion   = "Ml-Version"
)

type App struct {
	dbClient *db.Client
	config   *Config
}

type Context struct {
	*http.Request
	Writer  http.ResponseWriter
	Params  map[string]string
	Token   string
	Key     string
	Code    string
	UID     string
	UserID  uuid.UUID
	Version int64
}

func main() {
	confFile := flag.String("c", "env.yaml", "config file path")
	flag.Parse()

	config := loadConfig(*confFile)
	dbClient := initDB(config.Dsn)

	app := &App{
		dbClient: dbClient,
		config:   config,
	}

	router := http.NewServeMux()
	registerRoutes(router, app)

	log.Printf("Starting server on :%d", config.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// 初始化配置
func loadConfig(path string) *Config {
	configBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var conf Config
	if err := yaml.Unmarshal(configBytes, &conf); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
	return &conf
}

// 初始化数据库
func initDB(dsn string) *db.Client {
	client, err := db.New(dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	return client
}

// 路由注册
func registerRoutes(mux *http.ServeMux, app *App) {
	// Static handlers
	mux.HandleFunc("GET /todo/", htmlMiddleware(app.handleTodoHome))
	mux.HandleFunc("GET /todo/new", htmlMiddleware(app.handleNewBoard))
	mux.HandleFunc("GET /todo/{id}", htmlMiddleware(app.handleBoard))
	mux.HandleFunc("GET /chat/{id}", htmlMiddleware(app.handleChat))

	// API routes with middleware chain
	apiHandler := chainMiddleware(
		app.uuidValidationMiddleware,
		app.authMiddleware,
	)

	mux.Handle("GET /api/todo/{id}", apiHandler(http.HandlerFunc(app.handleGetTodo)))
	mux.Handle("GET /api/version/{id}", apiHandler(http.HandlerFunc(app.handleGetVersion)))
	mux.Handle("POST /api/todo/{id}", apiHandler(http.HandlerFunc(app.handleSaveTodo)))
	mux.Handle("POST /api/key/{id}", apiHandler(http.HandlerFunc(app.handleSetKey)))
	mux.Handle("POST /api/code/{id}", apiHandler(http.HandlerFunc(app.handleSetCode)))
	mux.Handle("POST /api/style/{id}", apiHandler(http.HandlerFunc(app.handleSetStyle)))
	mux.Handle("GET /api/chat/{id}", apiHandler(http.HandlerFunc(app.handleGetChat)))
	mux.Handle("POST /api/chat/{id}", apiHandler(http.HandlerFunc(app.handleSendChat)))
}

// 中间件链
func chainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// UUID验证中间件
func (app *App) uuidValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := uuid.Parse(id); err != nil {
			app.sendError(w, http.StatusBadRequest, "invalid UUID format")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func htmlMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := getCookie(r, TokenCookieName)
		if _, err := uuid.Parse(token); err != nil {
			token = uuid.New().String()
			setCookie(w, TokenCookieName, token)
		}

		ctx := &Context{
			Request: r,
			Writer:  w,
			Token:   token,
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "ctx", ctx)))
	})
}

// 认证中间件
func (app *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := getCookie(r, TokenCookieName)
		if _, err := uuid.Parse(token); err != nil {
			token = uuid.New().String()
			setCookie(w, TokenCookieName, token)
		}

		ctx := &Context{
			Request: r,
			Writer:  w,
			Token:   token,
			Key:     r.Header.Get(HeaderKey),
			Code:    r.Header.Get(HeaderCode),
			UID:     r.PathValue("id"),
			Params:  make(map[string]string),
		}

		// 提取公共参数
		ctx.Version, _ = strconv.ParseInt(r.Header.Get(HeaderVersion), 10, 64)

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "ctx", ctx)))
	})
}

// 处理函数示例
func (app *App) handleGetTodo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)

	data, err := app.dbClient.GetAllowed(db.Data{
		UID:     ctx.UID,
		Token:   ctx.Token,
		Key:     ctx.Key,
		Code:    ctx.Code,
		Version: ctx.Version,
	})

	if err != nil {
		app.handleDBError(w, err)
		return
	}

	if data.Content == "" {
		data.Content = `{"todo":[],"inProgress":[],"done":[],"trash":[]}`
	}

	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// 公共工具方法
func (app *App) sendHTML(w http.ResponseWriter, status int, html string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write([]byte(html))
}

func (app *App) sendJSON(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func (app *App) sendError(w http.ResponseWriter, status int, message string) {
	app.sendJSON(w, status, APIResponse{
		Code: status,
		Msg:  message,
	})
}

func (app *App) handleDBError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, db.ErrNeedCode):
		app.sendError(w, http.StatusForbidden, "need code or key")
	case errors.Is(err, db.ErrNeedKey):
		app.sendError(w, http.StatusForbidden, "need key")
	case errors.Is(err, db.ErrVersionConflict):
		app.sendError(w, http.StatusConflict, "version conflict")
	default:
		log.Printf("Database error: %v", err)
		app.sendError(w, http.StatusInternalServerError, "database error")
	}
}

// Cookie处理
func setCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:   name,
		Value:  value,
		Path:   "/",
		MaxAge: 3600 * 24 * 30000, // 30000 days
	})
}

func getCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func getToken(w http.ResponseWriter, r *http.Request) string {
	token := getCookie(r, TokenCookieName)
	if token == "" {
		token = uuid.New().String()
		setCookie(w, TokenCookieName, token)
	}
	return token
}

func (app *App) handleTodoHome(w http.ResponseWriter, r *http.Request) {
	app.sendHTML(w, http.StatusOK, homeHtml)
}

func (app *App) handleNewBoard(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/todo/"+uuid.New().String(), http.StatusFound)
}

func (app *App) handleBoard(w http.ResponseWriter, r *http.Request) {
	app.sendHTML(w, http.StatusOK, itemHtml)
}

func (app *App) handleChat(w http.ResponseWriter, r *http.Request) {
	app.sendHTML(w, http.StatusOK, chatHtml)
}

func (app *App) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)

	data, err := app.dbClient.GetVersion(db.Data{
		UID:     ctx.Params["id"],
		Token:   ctx.Token,
		Key:     ctx.Key,
		Code:    ctx.Code,
		Version: ctx.Version,
	})

	if err != nil {
		app.handleDBError(w, err)
		return
	}

	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func (app *App) handleSaveTodo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("io ReadAll:", err)
		app.sendJSON(w, http.StatusOK, APIResponse{
			Code: -4,
			Msg:  "body err",
		})
		return
	}
	if err := app.dbClient.SaveOrUpdate(db.Data{
		UID:     ctx.UID,
		Content: string(body),
		Token:   ctx.Token,
		Key:     ctx.Key,
		Code:    ctx.Code,
		Version: ctx.Version,
	}); err != nil {
		app.handleDBError(w, err)
		return
	}
	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

func (app *App) handleSetKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("io ReadAll:", err)
		app.sendJSON(w, http.StatusOK, APIResponse{
			Code: -4,
			Msg:  "body err",
		})
		return
	}
	if err := app.dbClient.SetKey(db.Data{
		UID:     ctx.UID,
		Token:   ctx.Token,
		Key:     ctx.Key,
		Code:    ctx.Code,
		Version: ctx.Version,
	}, string(body)); err != nil {
		app.handleDBError(w, err)
		return
	}
	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

func (app *App) handleSetCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("io ReadAll:", err)
		app.sendJSON(w, http.StatusOK, APIResponse{
			Code: -4,
			Msg:  "body err",
		})
		return
	}
	if err := app.dbClient.SetCode(db.Data{
		UID:     ctx.UID,
		Token:   ctx.Token,
		Key:     ctx.Key,
		Code:    ctx.Code,
		Version: ctx.Version,
	}, string(body)); err != nil {
		app.handleDBError(w, err)
		return
	}
	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

func (app *App) handleSetStyle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("io ReadAll:", err)
		app.sendJSON(w, http.StatusOK, APIResponse{
			Code: -4,
			Msg:  "body err",
		})
		return
	}
	if err := app.dbClient.SetStyle(db.Data{
		UID:     ctx.UID,
		Token:   ctx.Token,
		Key:     ctx.Key,
		Code:    ctx.Code,
		Version: ctx.Version,
	}, string(body)); err != nil {
		app.handleDBError(w, err)
		return
	}
	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

func (app *App) handleGetChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)
	data, err := app.dbClient.GetChat(ctx.UID, ctx.Token)
	if err != nil {
		app.handleDBError(w, err)
		return
	}
	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func (app *App) handleSendChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("ctx").(*Context)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("io ReadAll:", err)
		app.sendJSON(w, http.StatusOK, APIResponse{
			Code: -4,
			Msg:  "body err",
		})
		return
	}
	if err := app.dbClient.SendChat(db.Chat{
		UID:     ctx.UID,
		Token:   ctx.Token,
		Content: string(body),
		Creator: ctx.Token,
	}); err != nil {
		app.handleDBError(w, err)
		return
	}
	app.sendJSON(w, http.StatusOK, APIResponse{
		Code: 0,
		Msg:  "success",
	})
}
