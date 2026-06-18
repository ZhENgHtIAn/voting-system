package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	"example.com/voting-system/api/pb"
	"example.com/voting-system/internal/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type VoteClient interface {
	GetResults(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.VoteResponse, error)
	CastVote(ctx context.Context, in *pb.VoteRequest, opts ...grpc.CallOption) (*pb.VoteResponse, error)
}

type Server struct {
	voteClient     VoteClient
	requestTimeout time.Duration
	logger         *logging.Logger
}

type voteRequest struct {
	TopicName string `json:"topic_name"`
}

type voteResponse struct {
	Results map[string]int64 `json:"results"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type contextKey string

const requestIDKey contextKey = "request_id"

var requestIDSeed uint64

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func NewServer(voteClient VoteClient, requestTimeout time.Duration, logger *logging.Logger) (*Server, error) {
	if voteClient == nil {
		return nil, fmt.Errorf("vote client is nil")
	}
	if requestTimeout <= 0 {
		requestTimeout = DefaultRequestTimeout
	}
	return &Server{
		voteClient:     voteClient,
		requestTimeout: requestTimeout,
		logger:         logger,
	}, nil
}

func (s *Server) NewHandler(webDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/results", s.handleGetResults)
	mux.HandleFunc("/api/vote", s.handlePostVote)

	fileServer := http.FileServer(http.Dir(webDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	return s.loggingMiddleware(mux)
}

func (s *Server) handleGetResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()
	reqID := requestIDFromContext(r.Context())
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", reqID)

	s.logger.Info("http_forward_get_results_started", "request_id", reqID)

	resp, err := s.voteClient.GetResults(ctx, &pb.Empty{})
	if err != nil {
		s.logger.Error("http_forward_get_results_failed", "request_id", reqID, "error", err.Error())
		writeError(w, http.StatusBadGateway, fmt.Sprintf("get results failed: %v", err))
		return
	}

	s.logger.Info("http_forward_get_results_completed", "request_id", reqID, "topics", len(resp.GetResults()))
	writeJSON(w, http.StatusOK, voteResponse{Results: resp.GetResults()})
}

func (s *Server) handlePostVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TopicName == "" {
		writeError(w, http.StatusBadRequest, "topic_name is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()
	reqID := requestIDFromContext(r.Context())
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", reqID)

	s.logger.Info("http_forward_cast_vote_started", "request_id", reqID, "topic", req.TopicName)

	resp, err := s.voteClient.CastVote(ctx, &pb.VoteRequest{
		TopicName: req.TopicName,
	})
	if err != nil {
		s.logger.Error("http_forward_cast_vote_failed", "request_id", reqID, "topic", req.TopicName, "error", err.Error())
		writeError(w, statusCodeFromGRPCError(err), fmt.Sprintf("cast vote failed: %v", err))
		return
	}

	s.logger.Info("http_forward_cast_vote_completed", "request_id", reqID, "topic", req.TopicName, "votes", resp.GetResults()[req.TopicName])
	writeJSON(w, http.StatusOK, voteResponse{Results: resp.GetResults()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func statusCodeFromGRPCError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch status.Code(err) {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := requestIDFromHeader(r)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		r = r.WithContext(ctx)

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		s.logger.Info("http_request_started",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
		next.ServeHTTP(rec, r)
		s.logger.Info("http_request_completed",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func requestIDFromHeader(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	seed := atomic.AddUint64(&requestIDSeed, 1)
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), seed)
}

func requestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok && v != "" {
		return v
	}
	seed := atomic.AddUint64(&requestIDSeed, 1)
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), seed)
}
