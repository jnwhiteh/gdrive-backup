package main

import (
	"testing"

	"code.google.com/p/google-api-go-client/googleapi"
)

func TestIsRateLimitingError(t *testing.T) {
	type testCase struct {
		data     *googleapi.Error
		expected bool
	}

	testCases := []testCase{
		testCase{
			data: &googleapi.Error{
				Code:    403,
				Message: "Rate Limit Exceeded",
				Body: `{
 "error": {
  "errors": [
   {
    "domain": "usageLimits",
    "reason": "rateLimitExceeded",
    "message": "Rate Limit Exceeded"
   }
  ],
  "code": 403,
  "message": "Rate Limit Exceeded"
 }
}`},
			expected: true,
		},
		testCase{
			data: &googleapi.Error{
				Code:    403,
				Message: "Rate Limit Exceeded",
				Body: `{
 "error": {
  "errors": [
   {
    "domain": "usageLimits",
    "reason": "userRateLimitExceeded",
    "message": "Rate Limit Exceeded"
   }
  ],
  "code": 403,
  "message": "Rate Limit Exceeded"
 }
}`},
			expected: true,
		},
		testCase{
			data: &googleapi.Error{
				Code:    401,
				Message: "Invalid Credentials",
				Body: `{
 "error": {
  "errors": [
   {
    "domain": "global",
    "reason": "authError",
    "message": "Invalid Credentials",
    "locationType": "header",
    "location": "Authorization",
   }
  ],
  "code": 401,
  "message": "Invalid Credentials"
 }
}`},
			expected: false,
		},
	}

	for idx, testCase := range testCases {
		result := isRateLimitingError(testCase.data)
		if result != testCase.expected {
			t.Errorf("Failed to handle rateLimitExceeded in case %d", idx)
		}
	}
}
