package main

import (
    "context"
    "fmt"
    "log"

    "github.com/redis/go-redis/v9"
)

func main() {
    // 1. 初始化背景 Context
    ctx := context.Background()

    // 2. 创建 Redis 客户端连接本地 6379 端口
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // 默认没有密码
        DB:       0,  // 使用默认 DB
    })

    // 3. 测试 Ping
    pong, err := rdb.Ping(ctx).Result()
    if err != nil {
        log.Fatalf("连接 Redis 失败: %v", err)
    }
    fmt.Println("成功连接 Redis，响应:", pong)

    // 4. 预演投票系统的核心逻辑：Hash 结构原子自增 (HINCRBY)
    hashKey := "voting:topics"
    topicName := "Golang"

    // 给 Golang 投 1 票
    fmt.Println("\n--- 模拟用户点击投票 ---")
    newVotes, err := rdb.HIncrBy(ctx, hashKey, topicName, 1).Result()
    if err != nil {
        log.Fatalf("投票失败: %v", err)
    }
    fmt.Printf("投票成功！[%s] 当前总票数: %d\n", topicName, newVotes)

    // 再投 1 票
    newVotes, err = rdb.HIncrBy(ctx, hashKey, topicName, 1).Result()
    if err != nil {
        log.Fatalf("投票失败: %v", err)
    }
    fmt.Printf("又投了一票！[%s] 当前总票数: %d\n", topicName, newVotes)

    // 5. 获取当前所有话题的投票结果 (HGETALL)
    fmt.Println("\n--- 获取所有投票结果 ---")
    results, err := rdb.HGetAll(ctx, hashKey).Result()
    if err != nil {
        log.Fatalf("获取结果失败: %v", err)
    }

    for topic, votes := range results {
        fmt.Printf("话题: %s, 票数: %s\n", topic, votes)
    }
}