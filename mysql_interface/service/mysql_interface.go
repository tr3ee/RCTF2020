package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"math/rand"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/pingcap/parser"                     // v3.1.2-0.20200507065358-a5eade012146+incompatible
	_ "github.com/pingcap/tidb/types/parser_driver" // v1.1.0-beta.0.20200520024639-0414aa53c912
)

var isForbidden = [256]bool{}

const forbidden = "\x00\t\n\v\f\r`~!@#$%^&*()_=[]{}\\|:;'\"/?<>,\xa0"

func init() {
	for i := 0; i < len(forbidden); i++ {
		isForbidden[forbidden[i]] = true
	}
}

func allow(payload string) bool {
	if len(payload) < 3 || len(payload) > 128 {
		return false
	}
	for i := 0; i < len(payload); i++ {
		if isForbidden[payload[i]] {
			return false
		}
	}
	if _, _, err := parser.New().Parse(payload, "", ""); err != nil {
		return true
	}
	return false
}

func pow(ctx iris.Context) {
	rwlock.RLock()
	if ctx.FormValue("pow") != secret {
		ctx.StatusCode(iris.StatusForbidden)
		ctx.HTML("Error: <b> Wrong pow </b>")
		rwlock.RUnlock()
		return
	}
	rwlock.RUnlock()
	ctx.Next()
}

func sqlHandler(ctx iris.Context) {
	sql := ctx.FormValue("sql")
	ctx.Values().Set("sql", sql)
	if allow(sql) {
		var result string
		rows, err := db.Query(sql)
		if err != nil {
			ctx.HTML(err.Error())
			return
		}
		defer rows.Close()
		if rows.Next() {
			if err := rows.Scan(&result); err != nil {
				ctx.HTML(err.Error())
				return
			}
			ctx.HTML(result)
			return
		}
		ctx.HTML("Empty set")
		return
	}
	ctx.StatusCode(iris.StatusForbidden)
	ctx.HTML("Error: <b> Not allowed </b>")
}

const (
	prefix  = "RCTF2020_mysql_interface_"
	letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	db        *sql.DB
	rwlock    sync.RWMutex
	secret    = "000"
	challenge = hash(prefix + secret)
)

func hash(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func main() {
	rand.Seed(time.Now().Unix() ^ 0xcafebabe591591)
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			next := ""
			for i := 0; i < 3; i++ {
				next += string(letters[rand.Intn(len(letters))])
			}
			rwlock.Lock()
			secret = next
			challenge = hash(prefix + secret)
			rwlock.Unlock()
		}
	}()
	defer ticker.Stop()

	var err error
	db, err = sql.Open("mysql", "mysql_interface:b41fec9c1bcb194fb2028fa43dd74722@tcp(mysqld:3306)/mysql_interface")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := iris.New()
	logconf := logger.Config{
		Status:             true,
		IP:                 false,
		Method:             true,
		Path:               true,
		MessageContextKeys: []string{"sql"},
		MessageHeaderKeys:  []string{"X-Real-IP", "User-Agent"},
	}
	logconf.AddSkipper(func(ctx context.Context) bool {
		if ctx.Path() == "/" {
			return false
		}
		return true
	})
	app.Use(logger.New(logconf))
	app.HandleDir("/assets", "./assets")
	app.RegisterView(iris.HTML("./templates", ".html"))
	app.Get("/", func(ctx iris.Context) {
		rwlock.RLock()
		ctx.ViewData("prefix", prefix)
		ctx.ViewData("challenge", challenge)
		rwlock.RUnlock()
		ctx.View("index.html")
	})

	app.Post("/", pow, sqlHandler)

	if err := app.Listen(":8081", iris.WithPostMaxMemory(1)); err != nil {
		log.Fatal(err)
	}
}
