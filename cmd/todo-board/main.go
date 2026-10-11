package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Yanami-2K/todo-board/internal/seed"
	"github.com/Yanami-2K/todo-board/internal/server"
	"github.com/Yanami-2K/todo-board/internal/store"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		fs := flag.NewFlagSet("seed", flag.ExitOnError)
		force := fs.Bool("force", false, "wipe existing agents, todos, projects and inbox items first")
		_ = fs.Parse(os.Args[2:])
		st := openStore()
		defer st.Close()
		cfg := loadConfig()
		if _, err := server.EnsureOwner(st, cfg); err != nil {
			log.Fatal(err)
		}
		if err := seed.Run(st, *force); err != nil {
			log.Fatal(err)
		}
		return
	}
	cfg := loadConfig()
	st := openStore()
	defer st.Close()
	generated, err := server.EnsureOwner(st, cfg)
	if err != nil {
		log.Fatal(err)
	}
	if generated != "" {
		log.Printf("todo-board: 没有设置 OWNER_PASSWORD，已生成主人密码：%s", generated)
		log.Printf("todo-board: 这个密码只显示这一次。想自己定密码，设置 OWNER_PASSWORD 后重启。")
	}
	if cfg.IntakeMode == "token" && cfg.IntakeToken == "" {
		log.Fatal("INTAKE_MODE=token 时必须设置 INTAKE_TOKEN")
	}
	addr := env("ADDR", ":8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(cfg, st).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("todo-board: 听在 %s ，数据在 %s", addr, dbPath())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func openStore() *store.Store {
	st, err := store.Open(dbPath())
	if err != nil {
		log.Fatal(err)
	}
	return st
}

func dbPath() string {
	return filepath.Join(env("DATA_DIR", "data"), "todo.db")
}

func loadConfig() server.Config {
	tz := env("TZ", "Asia/Shanghai")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		if tz == "Asia/Shanghai" || tz == "Asia/Chongqing" {
			loc = time.FixedZone("CST", 8*3600)
		} else {
			log.Printf("todo-board: 时区 %s 无法加载，改用本地时区", tz)
			loc = time.Local
		}
	}
	mode := env("INTAKE_MODE", "open")
	if mode != "open" && mode != "token" {
		log.Fatal("INTAKE_MODE 只能是 open 或 token")
	}
	return server.Config{
		OwnerPassword:  os.Getenv("OWNER_PASSWORD"),
		OwnerName:      os.Getenv("OWNER_NAME"),
		IntakeIntro:    os.Getenv("INTAKE_INTRO"),
		IntakeHomepage: os.Getenv("INTAKE_HOMEPAGE"),
		IntakeMode:     mode,
		IntakeToken:    os.Getenv("INTAKE_TOKEN"),
		Location:       loc,
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
