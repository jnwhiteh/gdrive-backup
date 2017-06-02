package main

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"
)

type googleAPIError struct {
	Domain  string `json:"domain"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type googleAPIErrorResponse struct {
	Error struct {
		Errors  []googleAPIError `json:"errors"`
		Code    int              `json:"code"`
		Message string           `json:"message"`
	} `json:"error"`
}

func isRateLimitingResponse(body []byte) bool {
	// Attempt to unmarshal the response body
	var errorResponse googleAPIErrorResponse
	err := json.Unmarshal(body, &errorResponse)
	if err == nil && errorResponse.Error.Errors != nil {
		reason := errorResponse.Error.Errors[0].Reason
		return reason == "rateLimitExceeded" || reason == "userRateLimitExceeded"
	}
	return false
}

func getExponentialBackoffDelay(attempt uint, rand *rand.Rand, mutex *sync.Mutex) time.Duration {
	mutex.Lock()
	defer mutex.Unlock()

	randBit := time.Millisecond * time.Duration(rand.Int63n(1001))
	return (time.Second * (1 << (attempt + 1)))
}
