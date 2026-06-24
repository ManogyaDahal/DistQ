package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	client := redisclient.New(cfg)
	defer client.Close()

	ctx := context.Background()

	// 1. Check all keys in redis
	keys, err := client.Redis.Keys(ctx, "distq:*").Result()
	if err != nil {
		fmt.Printf("failed to list keys: %v\n", err)
	} else {
		fmt.Printf("All distq keys in Redis:\n")
		for _, k := range keys {
			t, _ := client.Redis.Type(ctx, k).Result()
			fmt.Printf("  Key: %s, Type: %s\n", k, t)
		}
	}

	// 2. Dump distq:scheduled members
	scheduled, err := client.Redis.ZRangeWithScores(ctx, redisclient.KeyScheduled, 0, -1).Result()
	if err != nil {
		fmt.Printf("failed to get scheduled tasks: %v\n", err)
	} else {
		fmt.Printf("Scheduled tasks in ZSET:\n")
		for _, z := range scheduled {
			fmt.Printf("  Member: %s, Score: %f\n", z.Member, z.Score)
		}
	}
}
