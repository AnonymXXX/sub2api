package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repair/subscriptionkey"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "github.com/lib/pq"
)

type authCacheInvalidator struct {
	cache service.APIKeyCache
}

func (i authCacheInvalidator) InvalidateKey(ctx context.Context, key string) error {
	sum := sha256.Sum256([]byte(key))
	cacheKey := hex.EncodeToString(sum[:])
	deleteErr := i.cache.DeleteAuthCache(ctx, cacheKey)
	publishErr := i.cache.PublishAuthCacheInvalidation(ctx, cacheKey)
	return errors.Join(deleteErr, publishErr)
}

func main() {
	apply := flag.Bool("apply", false, "apply the reported repairs (default is dry-run)")
	expectedCount := flag.Int("expected-count", -1, "exact candidate count required with --apply")
	timeout := flag.Duration("timeout", 2*time.Minute, "overall command timeout")
	flag.Parse()

	cfg, err := config.LoadForBootstrap()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := sql.Open("postgres", cfg.Database.DSNWithTimezone(cfg.Timezone))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	var invalidator subscriptionkey.CacheInvalidator
	var closeRedis func() error
	if *apply {
		redisClient := repository.InitRedis(cfg)
		closeRedis = redisClient.Close
		invalidator = authCacheInvalidator{cache: repository.NewAPIKeyCache(redisClient)}
	}
	if closeRedis != nil {
		defer func() { _ = closeRedis() }()
	}

	options := subscriptionkey.Options{Apply: *apply}
	if *expectedCount >= 0 {
		options.ExpectedCount = expectedCount
	}
	runner := subscriptionkey.NewRunner(subscriptionkey.NewSQLStore(db), invalidator)
	report, runErr := runner.Run(ctx, options)
	if report != nil {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			log.Fatalf("encode report: %v", err)
		}
	}
	if runErr != nil {
		log.Fatal(fmt.Errorf("subscription key repair failed: %w", runErr))
	}
}
