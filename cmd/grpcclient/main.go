package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"example.com/voting-system/api/pb"
)

const defaultTimeout = 3 * time.Second

func main() {
	var (
		addr   string
		action string
		topic  string
	)

	flag.StringVar(&addr, "addr", "127.0.0.1:50051", "grpc server address")
	flag.StringVar(&action, "action", "results", "action: results | vote")
	flag.StringVar(&topic, "topic", "", "topic name for vote action")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("dial grpc failed: %v", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("close grpc conn failed: %v", closeErr)
		}
	}()

	client := pb.NewVoteServiceClient(conn)
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "results":
		resp, rpcErr := client.GetResults(ctx, &pb.Empty{})
		if rpcErr != nil {
			log.Fatalf("GetResults failed: %v", rpcErr)
		}
		printResults(resp.GetResults())
	case "vote":
		if strings.TrimSpace(topic) == "" {
			log.Fatalf("-topic is required when -action=vote")
		}
		resp, rpcErr := client.CastVote(ctx, &pb.VoteRequest{TopicName: topic})
		if rpcErr != nil {
			log.Fatalf("CastVote failed: %v", rpcErr)
		}
		printResults(resp.GetResults())
	default:
		log.Fatalf("unknown action %q, expected results or vote", action)
	}
}

func printResults(results map[string]int64) {
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("vote results:")
	for _, key := range keys {
		fmt.Printf("- %s: %d\n", key, results[key])
	}
}
