package config

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	_ "modernc.org/sqlite"
)

type DBConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func OpenDB(cfg DBConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlite", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Protocol int
}

func NewRedis(rcg RedisConfig) (*redis.Client, error) {

	return redis.NewClient(&redis.Options{
		Addr:     rcg.Addr,
		Password: rcg.Password,
		DB:       rcg.DB,
		Protocol: rcg.Protocol,
	}), nil
}
