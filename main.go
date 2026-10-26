package main

import (
    "context"
    "errors"
    "flag"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"
    "time"

    "ties/addons"

    "github.com/fsnotify/fsnotify"
    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/logger"
    "github.com/kataras/iris/v12/middleware/recover"
)

var (
    libs string
)

func init() {
    flag.StringVar(&libs, "libs", ".", "libraries path")
    flag.Parse()
}

func startApp() *iris.Application {
    app := iris.New()
    app.Use(recover.New())
    app.Use(logger.New())

    addons.Load(app, libs)

    go func() {
        err := app.Listen(":8080",
            iris.WithoutInterruptHandler,
            iris.WithoutServerError(iris.ErrServerClosed),
        )
        if err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Println("listen:", err)
        }
    }()
    return app
}

func stopApp(app *iris.Application) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = app.Shutdown(ctx)
    addons.Free()
}

func main() {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal("Failed to create watcher:", err)
    }
    defer watcher.Close()

    if err := watcher.Add(libs); err != nil {
        log.Fatal("Failed to add path:", err)
    }

    sig := make(chan os.Signal, 1)
    signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

    const debounce = 300 * time.Millisecond
    timer := time.NewTimer(time.Hour)
    timer.Stop()

    app := startApp()

    for {
        select {
        case ev, ok := <-watcher.Events:
            if !ok {
                return
            }
            if filepath.Ext(ev.Name) != ".tie" {
                continue
            }
            if ev.Has(fsnotify.Write | fsnotify.Create | fsnotify.Remove | fsnotify.Rename) {
                log.Printf("Event: %s, File: %s", ev.Op, ev.Name)
                timer.Reset(debounce)
            }
        case err, ok := <-watcher.Errors:
            if !ok {
                return
            }
            log.Println("Error:", err)
        case <-timer.C:
            stopApp(app)
            app = startApp()
        case <-sig:
            stopApp(app)
            return
        }
    }
}
