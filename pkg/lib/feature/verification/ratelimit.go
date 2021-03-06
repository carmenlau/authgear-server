package verification

import (
	"fmt"
	"time"

	"github.com/authgear/authgear-server/pkg/lib/ratelimit"
)

// TODO(rate-limit): allow configuration of bucket size & reset period

func GenerateRateLimitBucket(userID string) ratelimit.Bucket {
	return ratelimit.Bucket{
		Key:         fmt.Sprintf("verification-generate-code:%s", userID),
		Size:        10,
		ResetPeriod: 1 * time.Minute,
	}
}

func VerifyRateLimitBucket(ip string) ratelimit.Bucket {
	return ratelimit.Bucket{
		Key:         fmt.Sprintf("verification-verify-code:%s", ip),
		Size:        10,
		ResetPeriod: 1 * time.Minute,
	}
}
