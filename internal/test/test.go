package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"example.com/voting-system/api/pb"
	"example.com/voting-system/internal/grpcserver"
	"example.com/voting-system/internal/httpserver"
	"example.com/voting-system/internal/pkg/logging"
)

var testTopics = []string{"Golang", "Kubernetes", "Rust"}

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

type inMemoryVoteClient struct {
	mu      sync.Mutex
	results map[string]int64
}

func newInMemoryVoteClient() *inMemoryVoteClient {
	results := make(map[string]int64, len(testTopics))
	for _, topic := range testTopics {
		results[topic] = 0
	}
	return &inMemoryVoteClient{results: results}
}

func (c *inMemoryVoteClient) GetResults(_ context.Context, _ *pb.Empty, _ ...grpc.CallOption) (*pb.VoteResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cloned := make(map[string]int64, len(c.results))
	for k, v := range c.results {
		cloned[k] = v
	}
	return &pb.VoteResponse{Results: cloned}, nil
}

func (c *inMemoryVoteClient) CastVote(_ context.Context, in *pb.VoteRequest, _ ...grpc.CallOption) (*pb.VoteResponse, error) {
	topic := strings.TrimSpace(in.GetTopicName())
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.results[topic]; !ok {
		return nil, status.Errorf(codes.InvalidArgument, "topic_name %q is not allowed", topic)
	}
	c.results[topic]++
	cloned := make(map[string]int64, len(c.results))
	for k, v := range c.results {
		cloned[k] = v
	}
	return &pb.VoteResponse{Results: cloned}, nil
}

func newHTTPHandlerForTest(t *testing.T, voteClient httpserver.VoteClient) http.Handler {
	t.Helper()
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatalf("write temp index file failed: %v", err)
	}
	server, err := httpserver.NewServer(voteClient, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err != nil {
		t.Fatalf("create http server failed: %v", err)
	}
	return server.NewHandler(webDir)
}

func newGRPCServiceForTest(t *testing.T) (*grpcserver.VoteService, *redis.Client) {
	t.Helper()
	miniRedis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	t.Cleanup(miniRedis.Close)

	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	service, err := grpcserver.NewVoteService(redisClient, "voting:topics", testTopics, logging.NewLogger("test-grpcserver", "error"))
	if err != nil {
		t.Fatalf("create grpc service failed: %v", err)
	}
	return service, redisClient
}

type httpVoteResp struct {
	Results map[string]int64 `json:"results"`
}

type httpErrorResp struct {
	Error string `json:"error"`
}

func decodeHTTPVoteResponse(t *testing.T, raw string) httpVoteResp {
	t.Helper()
	var resp httpVoteResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("decode vote response failed: %v, body=%s", err, raw)
	}
	return resp
}

func decodeHTTPErrorResponse(t *testing.T, raw string) httpErrorResp {
	t.Helper()
	var resp httpErrorResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("decode error response failed: %v, body=%s", err, raw)
	}
	return resp
}

func testHTTPLoadConfigSuccess(t *testing.T) {
	t.Log("开始执行 HTTP 配置加载成功测试")
	cfgPath := filepath.Join(t.TempDir(), "httpserver.yaml")
	content := `http:
  listen_addr: ":18080"
grpc:
  target: "dns:///127.0.0.1:50051"
server:
  request_timeout: "2s"
web:
  dir: "web"
logging:
  level: "debug"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := httpserver.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.HTTP.ListenAddr != ":18080" {
		t.Fatalf("unexpected listen addr: %s", cfg.HTTP.ListenAddr)
	}
	if cfg.RequestTimeout() != 2*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.RequestTimeout())
	}
	t.Logf("测试通过：listen_addr=%s timeout=%s", cfg.HTTP.ListenAddr, cfg.RequestTimeout())
}

func testHTTPLoadConfigValidationFailure(t *testing.T) {
	t.Log("开始执行 HTTP 配置校验失败测试")
	cfgPath := filepath.Join(t.TempDir(), "httpserver.yaml")
	content := `http:
  listen_addr: ":8080"
grpc:
  target: ""
web:
  dir: "web"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	if _, err := httpserver.LoadConfig(cfgPath); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	t.Log("测试通过：空 grpc.target 被正确拦截")
}

func testHTTPRequestTimeoutFallback(t *testing.T) {
	t.Log("开始执行 HTTP 超时默认回退测试")
	cfg := &httpserver.Config{}
	cfg.Server.RequestTimeout = "invalid-duration"
	if cfg.RequestTimeout() != httpserver.DefaultRequestTimeout {
		t.Fatalf("unexpected fallback timeout: %s", cfg.RequestTimeout())
	}
	t.Logf("测试通过：fallback timeout=%s", cfg.RequestTimeout())
}

func testHTTPGetResultsSuccess(t *testing.T) {
	t.Log("开始执行 HTTP GET /api/results 成功测试")
	handler := newHTTPHandlerForTest(t, &mockVoteClient{
		getResults: func(_ context.Context, _ *pb.Empty, _ ...grpc.CallOption) (*pb.VoteResponse, error) {
			return &pb.VoteResponse{
				Results: map[string]int64{"Golang": 2, "Kubernetes": 1, "Rust": 0},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/results", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	resp := decodeHTTPVoteResponse(t, rec.Body.String())
	if resp.Results["Golang"] != 2 {
		t.Fatalf("unexpected Golang votes: %d", resp.Results["Golang"])
	}
	t.Logf("测试通过：响应结果=%v", resp.Results)
}

func testHTTPPostVoteSuccess(t *testing.T) {
	t.Log("开始执行 HTTP POST /api/vote 成功测试")
	handler := newHTTPHandlerForTest(t, &mockVoteClient{
		castVote: func(_ context.Context, in *pb.VoteRequest, _ ...grpc.CallOption) (*pb.VoteResponse, error) {
			if in.GetTopicName() != "Golang" {
				t.Fatalf("unexpected topic: %s", in.GetTopicName())
			}
			return &pb.VoteResponse{
				Results: map[string]int64{"Golang": 3, "Kubernetes": 1, "Rust": 0},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/vote", strings.NewReader(`{"topic_name":"Golang"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	resp := decodeHTTPVoteResponse(t, rec.Body.String())
	if resp.Results["Golang"] != 3 {
		t.Fatalf("unexpected Golang votes: %d", resp.Results["Golang"])
	}
	t.Logf("测试通过：响应结果=%v", resp.Results)
}

func testHTTPPostVoteBadRequest(t *testing.T) {
	t.Log("开始执行 HTTP POST /api/vote 非法请求体测试")
	handler := newHTTPHandlerForTest(t, &mockVoteClient{})
	req := httptest.NewRequest(http.MethodPost, "/api/vote", strings.NewReader(`{"topic_name":`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusBadRequest)
	}
	resp := decodeHTTPErrorResponse(t, rec.Body.String())
	t.Logf("测试通过：状态码=%d 错误信息=%s", rec.Code, resp.Error)
}

func testHTTPPostVoteInvalidArgumentMappedTo400(t *testing.T) {
	t.Log("开始执行 gRPC InvalidArgument -> HTTP 400 映射测试")
	handler := newHTTPHandlerForTest(t, &mockVoteClient{
		castVote: func(_ context.Context, _ *pb.VoteRequest, _ ...grpc.CallOption) (*pb.VoteResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "topic not allowed")
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/vote", strings.NewReader(`{"topic_name":"Java"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusBadRequest)
	}
	t.Logf("测试通过：状态码=%d", rec.Code)
}

func testHTTPGetResultsMethodNotAllowed(t *testing.T) {
	t.Log("开始执行 HTTP /api/results 方法约束测试")
	handler := newHTTPHandlerForTest(t, &mockVoteClient{})
	req := httptest.NewRequest(http.MethodPost, "/api/results", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	t.Logf("测试通过：状态码=%d", rec.Code)
}

func testHTTPNewServerNilClient(t *testing.T) {
	t.Log("开始执行 HTTP NewServer 空客户端校验测试")
	_, err := httpserver.NewServer(nil, time.Second, logging.NewLogger("test-httpserver", "error"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	t.Logf("测试通过：错误=%v", err)
}

func testHTTPRequestIDPropagation(t *testing.T) {
	t.Log("开始执行 HTTP -> gRPC request id 透传测试")
	var capturedRequestID string
	handler := newHTTPHandlerForTest(t, &mockVoteClient{
		getResults: func(ctx context.Context, _ *pb.Empty, _ ...grpc.CallOption) (*pb.VoteResponse, error) {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatalf("outgoing metadata missing")
			}
			values := md.Get("x-request-id")
			if len(values) == 0 {
				t.Fatalf("x-request-id missing in outgoing metadata")
			}
			capturedRequestID = values[0]
			return &pb.VoteResponse{Results: map[string]int64{"Golang": 1, "Kubernetes": 0, "Rust": 0}}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/results", nil)
	req.Header.Set("X-Request-ID", "manual-id-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	if capturedRequestID != "manual-id-001" {
		t.Fatalf("unexpected request id propagation: %s", capturedRequestID)
	}
	t.Logf("测试通过：request_id=%s", capturedRequestID)
}

func testHTTPConcurrentVoteConsistency(t *testing.T) {
	t.Log("开始执行 HTTP 并发投票一致性测试")
	voteClient := newInMemoryVoteClient()
	handler := newHTTPHandlerForTest(t, voteClient)

	const totalVotes = 120
	var wg sync.WaitGroup
	wg.Add(totalVotes)
	for i := 0; i < totalVotes; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/vote", strings.NewReader(`{"topic_name":"Golang"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent vote failed: status=%d body=%s", rec.Code, rec.Body.String())
			}
		}()
	}
	wg.Wait()

	getReq := httptest.NewRequest(http.MethodGet, "/api/results", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("query results failed: status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	resp := decodeHTTPVoteResponse(t, getRec.Body.String())
	if resp.Results["Golang"] != totalVotes {
		t.Fatalf("unexpected final votes: got=%d want=%d", resp.Results["Golang"], totalVotes)
	}
	t.Logf("并发测试通过：Golang 最终票数=%d，总并发请求=%d", resp.Results["Golang"], totalVotes)
}

func testGRPCCastVoteSuccess(t *testing.T) {
	t.Log("开始执行 gRPC CastVote 成功测试")
	service, _ := newGRPCServiceForTest(t)
	resp, err := service.CastVote(context.Background(), &pb.VoteRequest{TopicName: "Golang"})
	if err != nil {
		t.Fatalf("CastVote failed: %v", err)
	}
	if resp.Results["Golang"] != 1 {
		t.Fatalf("unexpected Golang votes: %d", resp.Results["Golang"])
	}
	t.Logf("测试通过：响应结果=%v", resp.Results)
}

func testGRPCCastVoteInvalidTopic(t *testing.T) {
	t.Log("开始执行 gRPC CastVote 非法话题测试")
	service, _ := newGRPCServiceForTest(t)
	_, err := service.CastVote(context.Background(), &pb.VoteRequest{TopicName: "Java"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unexpected code: got %s want %s", status.Code(err), codes.InvalidArgument)
	}
	t.Logf("测试通过：错误码=%s", status.Code(err))
}

func testGRPCCastVoteNilRequest(t *testing.T) {
	t.Log("开始执行 gRPC CastVote 空请求测试")
	service, _ := newGRPCServiceForTest(t)
	_, err := service.CastVote(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unexpected code: got %s want %s", status.Code(err), codes.InvalidArgument)
	}
	t.Logf("测试通过：错误码=%s", status.Code(err))
}

func testGRPCGetResultsDefaultZero(t *testing.T) {
	t.Log("开始执行 gRPC GetResults 默认零值测试")
	service, _ := newGRPCServiceForTest(t)
	resp, err := service.GetResults(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}
	for _, topic := range testTopics {
		if resp.Results[topic] != 0 {
			t.Fatalf("unexpected default vote for %s: %d", topic, resp.Results[topic])
		}
	}
	t.Logf("测试通过：默认结果=%v", resp.Results)
}

func testGRPCConcurrentCastVoteConsistency(t *testing.T) {
	t.Log("开始执行 gRPC 并发 CastVote 一致性测试")
	service, redisClient := newGRPCServiceForTest(t)

	const totalVotes = 240
	var wg sync.WaitGroup
	errCh := make(chan error, totalVotes)

	wg.Add(totalVotes)
	for i := 0; i < totalVotes; i++ {
		go func() {
			defer wg.Done()
			_, err := service.CastVote(context.Background(), &pb.VoteRequest{TopicName: "Rust"})
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent CastVote failed: %v", err)
	}

	resp, err := service.GetResults(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}
	if resp.Results["Rust"] != totalVotes {
		t.Fatalf("unexpected final Rust votes: got=%d want=%d", resp.Results["Rust"], totalVotes)
	}

	raw, err := redisClient.HGet(context.Background(), "voting:topics", "Rust").Int64()
	if err != nil {
		t.Fatalf("read redis rust votes failed: %v", err)
	}
	if raw != totalVotes {
		t.Fatalf("unexpected redis rust votes: got=%d want=%d", raw, totalVotes)
	}
	t.Logf("并发测试通过：Rust 最终票数=%d，总并发请求=%d", raw, totalVotes)
}

func testWebFullChainConcurrentIntegration(t *testing.T) {
	t.Log("开始执行 Web->HTTP->gRPC->Redis 全链路并发集成测试")
	service, _ := newGRPCServiceForTest(t)

	grpcListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen grpc addr failed: %v", err)
	}
	t.Cleanup(func() { _ = grpcListener.Close() })

	grpcSrv := grpc.NewServer()
	pb.RegisterVoteServiceServer(grpcSrv, service)
	t.Cleanup(grpcSrv.Stop)

	go func() {
		if serveErr := grpcSrv.Serve(grpcListener); serveErr != nil {
			t.Logf("grpc server stopped: %v", serveErr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, grpcListener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	grpcClient := pb.NewVoteServiceClient(conn)
	httpHandler := newHTTPHandlerForTest(t, grpcClient)
	httpSrv := httptest.NewServer(httpHandler)
	t.Cleanup(httpSrv.Close)

	const totalVotes = 300
	var wg sync.WaitGroup
	errCh := make(chan error, totalVotes)
	wg.Add(totalVotes)
	for i := 0; i < totalVotes; i++ {
		go func() {
			defer wg.Done()
			body := bytes.NewBufferString(`{"topic_name":"Kubernetes"}`)
			req, reqErr := http.NewRequest(http.MethodPost, httpSrv.URL+"/api/vote", body)
			if reqErr != nil {
				errCh <- reqErr
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, doErr := http.DefaultClient.Do(req)
			if doErr != nil {
				errCh <- doErr
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				raw, _ := io.ReadAll(resp.Body)
				errCh <- status.Errorf(codes.Internal, "http status=%d body=%s", resp.StatusCode, string(raw))
				return
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatalf("full chain concurrent request failed: %v", e)
	}

	resultResp, err := http.Get(httpSrv.URL + "/api/results")
	if err != nil {
		t.Fatalf("query results failed: %v", err)
	}
	defer resultResp.Body.Close()
	if resultResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resultResp.Body)
		t.Fatalf("query results status invalid: %d, body=%s", resultResp.StatusCode, string(raw))
	}
	rawResultBody, err := io.ReadAll(resultResp.Body)
	if err != nil {
		t.Fatalf("read result body failed: %v", err)
	}
	finalResp := decodeHTTPVoteResponse(t, string(rawResultBody))
	if finalResp.Results["Kubernetes"] != totalVotes {
		t.Fatalf("unexpected final Kubernetes votes: got=%d want=%d", finalResp.Results["Kubernetes"], totalVotes)
	}

	t.Logf("全链路并发测试通过：Kubernetes 最终票数=%d，总并发请求=%d", finalResp.Results["Kubernetes"], totalVotes)
}
