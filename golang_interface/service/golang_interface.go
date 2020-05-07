package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/kataras/iris/v12"
)

func checkAndRun(filename string) ([]byte, error) {
	// check
	file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.AllErrors)
	if err != nil {
		return nil, errors.New("Syntax error")
	}
	if len(file.Imports) > 0 {
		return nil, errors.New("Imports are not allowed")
	}

	temp, err := ioutil.TempFile("/home/ctf/builds", "go-build*")
	if err != nil {
		return nil, errors.New("Failed to create temporary file while building")
	}
	temp.Close()
	defer os.Remove(temp.Name())
	// build
	buildcmd := exec.Command("timeout", "-k1", "5", "go", "build", "-buildmode=pie", "-o", temp.Name(), filename)
	if _, err := buildcmd.CombinedOutput(); err != nil {
		return []byte(fmt.Sprintf("Error while building: probably timeout, try again (%s)", err.Error())), nil
	}
	// run
	cmd := exec.Command("timeout", "-k1", "1", "chroot", "--userspec=1000:1000", "/home/ctf", "./builds/"+filepath.Base(temp.Name()))
	output, err := cmd.CombinedOutput()
	return output, nil
}

func uploadLimit(ctx iris.Context) {
	if ctx.GetContentLength() > maxSize {
		ctx.StatusCode(iris.StatusRequestEntityTooLarge)
		ctx.HTML("Error while uploading: <b> file is too large (>10KB) </b>")
		return
	}
	rwlock.RLock()
	if ctx.FormValue("pow") != secret {
		ctx.StatusCode(iris.StatusForbidden)
		ctx.HTML("Error while uploading: <b> wrong pow </b>")
		rwlock.RUnlock()
		return
	}
	rwlock.RUnlock()
	ctx.Next()
}

func uploadHandler(ctx iris.Context) {
	// Get the file from the request.
	file, _, err := ctx.FormFile("file")
	if err != nil {
		ctx.StatusCode(iris.StatusForbidden)
		ctx.HTML("Error while uploading: <b> need a Go file to run</b>")
		return
	}
	defer file.Close()

	tmpfile, err := ioutil.TempFile("/uploads", "upload_*.go")
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.HTML("Error while uploading: <b> failed to create temporary file</b>")
		return
	}

	if _, err := io.Copy(tmpfile, file); err != nil {
		os.Remove(tmpfile.Name())
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.HTML("Error while uploading: <b> failed to write temporary file</b>")
		return
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.HTML("Error while uploading: <b> failed to close temporary file</b>")
		return
	}

	out, err := checkAndRun(tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		ctx.StatusCode(iris.StatusForbidden)
		ctx.HTML("Error: %s", err.Error())
	}

	ctx.HTML("%s", out)
}

const (
	maxSize = 10 * 1024 // 10KB
	prefix  = "RCTF2020_golang_interface_"
	letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	rwlock    sync.RWMutex
	secret    = "000"
	challenge = "30129e34c41fdc84aa4306029dcdd851d609e40b074c919a233d83e840fe9656"
)

func main() {
	rand.Seed(time.Now().Unix() ^ 0xcafebabe909090)
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			next := ""
			for i := 0; i < 3; i++ {
				next += string(letters[rand.Intn(len(letters))])
			}
			hash := sha256.Sum256([]byte(prefix + next))
			rwlock.Lock()
			secret = next
			challenge = hex.EncodeToString(hash[:])
			rwlock.Unlock()
		}
	}()
	defer ticker.Stop()

	app := iris.New()
	app.HandleDir("/assets", "./assets")
	app.RegisterView(iris.HTML("./templates", ".html"))
	app.Get("/", func(ctx iris.Context) {
		rwlock.RLock()
		ctx.ViewData("prefix", prefix)
		ctx.ViewData("challenge", challenge)
		rwlock.RUnlock()
		ctx.View("index.html")
	})

	app.Post("/", uploadLimit, uploadHandler)

	app.Listen(":8080", iris.WithPostMaxMemory(maxSize))
}
