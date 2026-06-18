package test

import "testing"

func TestHTTPLoadConfigSuccess(t *testing.T) {
	testHTTPLoadConfigSuccess(t)
}

func TestHTTPLoadConfigValidationFailure(t *testing.T) {
	testHTTPLoadConfigValidationFailure(t)
}

func TestHTTPRequestTimeoutFallback(t *testing.T) {
	testHTTPRequestTimeoutFallback(t)
}

func TestHTTPGetResultsSuccess(t *testing.T) {
	testHTTPGetResultsSuccess(t)
}

func TestHTTPPostVoteSuccess(t *testing.T) {
	testHTTPPostVoteSuccess(t)
}

func TestHTTPPostVoteBadRequest(t *testing.T) {
	testHTTPPostVoteBadRequest(t)
}

func TestHTTPPostVoteInvalidArgumentMappedTo400(t *testing.T) {
	testHTTPPostVoteInvalidArgumentMappedTo400(t)
}

func TestHTTPGetResultsMethodNotAllowed(t *testing.T) {
	testHTTPGetResultsMethodNotAllowed(t)
}

func TestHTTPNewServerNilClient(t *testing.T) {
	testHTTPNewServerNilClient(t)
}

func TestHTTPRequestIDPropagation(t *testing.T) {
	testHTTPRequestIDPropagation(t)
}

func TestHTTPConcurrentVoteConsistency(t *testing.T) {
	testHTTPConcurrentVoteConsistency(t)
}

func TestGRPCCastVoteSuccess(t *testing.T) {
	testGRPCCastVoteSuccess(t)
}

func TestGRPCCastVoteInvalidTopic(t *testing.T) {
	testGRPCCastVoteInvalidTopic(t)
}

func TestGRPCCastVoteNilRequest(t *testing.T) {
	testGRPCCastVoteNilRequest(t)
}

func TestGRPCGetResultsDefaultZero(t *testing.T) {
	testGRPCGetResultsDefaultZero(t)
}

func TestGRPCConcurrentCastVoteConsistency(t *testing.T) {
	testGRPCConcurrentCastVoteConsistency(t)
}

func TestWebFullChainConcurrentIntegration(t *testing.T) {
	testWebFullChainConcurrentIntegration(t)
}
