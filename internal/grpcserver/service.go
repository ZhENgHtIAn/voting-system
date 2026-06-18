package grpcserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"example.com/voting-system/api/pb"
	"example.com/voting-system/internal/pkg/logging"
)

type VoteService struct {
	pb.UnimplementedVoteServiceServer

	redisClient   *redis.Client
	redisHashKey  string
	allowedTopics map[string]struct{}
	logger        *logging.Logger
}

func NewVoteService(redisClient *redis.Client, redisHashKey string, topics []string, logger *logging.Logger) (*VoteService, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if redisHashKey == "" {
		return nil, fmt.Errorf("redis hash key is empty")
	}
	allowedTopics := make(map[string]struct{}, len(topics))
	for _, topic := range topics {
		trimmed := strings.TrimSpace(topic)
		if trimmed == "" {
			return nil, fmt.Errorf("topic contains empty value")
		}
		allowedTopics[trimmed] = struct{}{}
	}
	if len(allowedTopics) != 3 {
		return nil, fmt.Errorf("allowed topics must contain exactly 3 unique topics")
	}

	return &VoteService{
		redisClient:   redisClient,
		redisHashKey:  redisHashKey,
		allowedTopics: allowedTopics,
		logger:        logger,
	}, nil
}

func (s *VoteService) CastVote(ctx context.Context, req *pb.VoteRequest) (*pb.VoteResponse, error) {
	if req == nil {
		s.logger.Warn("cast_vote_invalid_request", "reason", "nil_request")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	topic := strings.TrimSpace(req.GetTopicName())
	if topic == "" {
		s.logger.Warn("cast_vote_invalid_request", "reason", "empty_topic")
		return nil, status.Error(codes.InvalidArgument, "topic_name is required")
	}
	if !s.isAllowedTopic(topic) {
		s.logger.Warn("cast_vote_invalid_request", "reason", "topic_not_allowed", "topic", topic)
		return nil, status.Errorf(codes.InvalidArgument, "topic_name %q is not allowed", topic)
	}

	s.logger.Info("cast_vote_started", "topic", topic)
	if _, err := s.redisClient.HIncrBy(ctx, s.redisHashKey, topic, 1).Result(); err != nil {
		s.logger.Error("cast_vote_increment_failed", "topic", topic, "error", err.Error())
		return nil, status.Errorf(codes.Internal, "increment vote failed: %v", err)
	}

	resp, err := s.buildVoteResponse(ctx)
	if err != nil {
		s.logger.Error("cast_vote_build_response_failed", "topic", topic, "error", err.Error())
		return nil, err
	}
	s.logger.Info("cast_vote_completed", "topic", topic, "votes", resp.GetResults()[topic])
	return resp, nil
}

func (s *VoteService) GetResults(ctx context.Context, _ *pb.Empty) (*pb.VoteResponse, error) {
	s.logger.Debug("get_results_started")
	resp, err := s.buildVoteResponse(ctx)
	if err != nil {
		s.logger.Error("get_results_failed", "error", err.Error())
		return nil, err
	}
	s.logger.Debug("get_results_completed", "topics", len(resp.GetResults()))
	return resp, nil
}

func (s *VoteService) buildVoteResponse(ctx context.Context) (*pb.VoteResponse, error) {
	results := make(map[string]int64, len(s.allowedTopics))
	for topic := range s.allowedTopics {
		results[topic] = 0
	}

	rawResults, err := s.redisClient.HGetAll(ctx, s.redisHashKey).Result()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load results failed: %v", err)
	}

	for topic, votesStr := range rawResults {
		if !s.isAllowedTopic(topic) {
			continue
		}
		votes, parseErr := strconv.ParseInt(votesStr, 10, 64)
		if parseErr != nil {
			return nil, status.Errorf(codes.Internal, "invalid vote value for topic %q: %v", topic, parseErr)
		}
		results[topic] = votes
	}

	return &pb.VoteResponse{Results: results}, nil
}

func (s *VoteService) isAllowedTopic(topic string) bool {
	_, ok := s.allowedTopics[topic]
	return ok
}
