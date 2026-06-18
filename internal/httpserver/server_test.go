package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/voting-system/api/pb"
	"example.com/voting-system/internal/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockVoteClient struct {
	getResults func(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.VoteResponse, error)
	castVote   func(ctx context.Context, in *pb.VoteRequest, opts ...grpc.CallOption) (*pb.VoteResponse, error)
}

func (m *mockVoteClient) GetResults(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.VoteResponse, error) {
	if m.getResults != nil {
		return m.getResults(ctx, in, opts...)
	}
	return &pb.VoteResponse{Results: map[string]int64{}}, nil
}

func (m *mockVoteClient) CastVote(ctx context.Context, in *pb.VoteRequest, opts ...grpc.CallOption) (*pb.VoteResponse, error) {
	if m.castVote != nil {
		return m.castVote(ctx, in, opts...)
	}
	return &pb.VoteResponse{Results: map[string]int64{}}, nil
}

func TestHandleGetResultsSuccess(t *testing.T) {
	t.Parallel()
	server, err := NewServer(&mockVoteClient{
		getResults: func(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.VoteResponse, error) {
			return &pb.VoteResponse{
				Results: map[string]int64{
					"Golang":     2,
					"Kubernetes": 1,
					"Rust":       0,
				},
			}, nil
		},
	}, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/results", nil)
	rec := httptest.NewRecorder()

	server.NewHandler(".").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"Golang":2`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestHandlePostVoteSuccess(t *testing.T) {
	t.Parallel()
	server, err := NewServer(&mockVoteClient{
		castVote: func(ctx context.Context, in *pb.VoteRequest, opts ...grpc.CallOption) (*pb.VoteResponse, error) {
			if in.GetTopicName() != "Golang" {
				t.Fatalf("unexpected topic: %s", in.GetTopicName())
			}
			return &pb.VoteResponse{
				Results: map[string]int64{
					"Golang":     3,
					"Kubernetes": 1,
					"Rust":       0,
				},
			}, nil
		},
	}, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/vote", strings.NewReader(`{"topic_name":"Golang"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.NewHandler(".").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"Golang":3`) {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestHandlePostVoteBadRequest(t *testing.T) {
	t.Parallel()
	server, err := NewServer(&mockVoteClient{}, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/vote", strings.NewReader(`{"topic_name":`))
	rec := httptest.NewRecorder()

	server.NewHandler(".").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePostVoteInvalidArgumentMappedTo400(t *testing.T) {
	t.Parallel()
	server, err := NewServer(&mockVoteClient{
		castVote: func(ctx context.Context, in *pb.VoteRequest, opts ...grpc.CallOption) (*pb.VoteResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "topic not allowed")
		},
	}, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/vote", strings.NewReader(`{"topic_name":"Java"}`))
	rec := httptest.NewRecorder()

	server.NewHandler(".").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleGetResultsMethodNotAllowed(t *testing.T) {
	t.Parallel()
	server, err := NewServer(&mockVoteClient{}, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/results", nil)
	rec := httptest.NewRecorder()

	server.NewHandler(".").ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewServerNilClient(t *testing.T) {
	t.Parallel()
	_, err := NewServer(nil, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestStatusCodeFromGRPCErrorDefault502(t *testing.T) {
	t.Parallel()
	got := statusCodeFromGRPCError(status.Error(codes.Internal, "boom"))
	if got != http.StatusBadGateway {
		t.Fatalf("unexpected status: got %d want %d", got, http.StatusBadGateway)
	}
}

func TestRequestIDFromHeaderAndContext(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/results", nil)
	req.Header.Set("X-Request-ID", "manual-id")
	if got, want := requestIDFromHeader(req), "manual-id"; got != want {
		t.Fatalf("unexpected request id: got %s want %s", got, want)
	}

	ctx := context.WithValue(context.Background(), requestIDKey, "ctx-id")
	if got, want := requestIDFromContext(ctx), "ctx-id"; got != want {
		t.Fatalf("unexpected context request id: got %s want %s", got, want)
	}
}
