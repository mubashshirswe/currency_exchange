package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/mubashshir3767/currencyExchange/internal/db"
	"github.com/mubashshir3767/currencyExchange/internal/env"
	"github.com/mubashshir3767/currencyExchange/internal/fcm"
	"github.com/mubashshir3767/currencyExchange/internal/notify"
	"github.com/mubashshir3767/currencyExchange/internal/service"
	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/store/cache"
)

func main() {
	// Local: .env fayl; server/Docker: o'zgaruvchilar environment orqali (fayl shart emas)
	if err := godotenv.Load(); err != nil {
		log.Printf("dotenv: %v (using process environment)", err)
	}

	if len(os.Args) > 1 && os.Args[1] == "-print-db-url" {
		fmt.Println(env.PostgresURL())
		return
	}

	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		db: dbConfig{
			addr:         env.PostgresURL(),
			maxOpenConns: env.GetInt("MAX_OPEN_CONNS", 50),
			maxIdleConns: env.GetInt("MAX_IDLE_CONNS", 50),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		redisConfig: redisConfig{
			addr:    env.GetString("REDIS_ADDR", "localhost:6379"),
			pw:      env.GetString("REDIS_PW", ""),
			db:      env.GetInt("REDIS_DB", 0),
			enabled: env.GetBool("REDIS_ENABLED", true),
		},
		env: env.GetString("ENV", "PROD"),
	}

	rdb := cache.NewRedisClient(cfg.redisConfig.addr, cfg.redisConfig.pw, cfg.redisConfig.db)
	log.Println("redis cache connection established")

	db, err := db.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)

	if err != nil {
		log.Fatalf("failed to establish a database connection: %v", err)
	}

	defer db.Close()
	log.Println("DATABASE HAS BEEN SUCCESSFULLY ESTABLISHED")

	store := store.NewStorage(db)

	var delivered notify.DeliveredUser = notify.NoopDeliveredUser{}
	if creds := env.GetString("FIREBASE_CREDENTIALS_PATH", ""); creds != "" {
		n, err := fcm.NewDeliveredNotifier(creds, store)
		if err != nil {
			log.Printf("FCM notifier disabled: %v", err)
		} else {
			delivered = n
		}
	}

	service := service.NewService(store, delivered)
	cacheStore := cache.NewRedisStorage(rdb)

	dedupWindow := time.Duration(env.GetInt("DEDUP_WINDOW_SECONDS", 10)) * time.Second

	app := application{
		config:     cfg,
		store:      store,
		service:    service,
		cacheStore: cacheStore,
		dedup:      newIdempotencyGuard(dedupWindow),
	}

	mux := app.mount()
	log.Fatal(app.run(mux))
}
