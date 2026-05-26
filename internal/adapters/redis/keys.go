package redis

import (
	"fmt"
)

const (
	stopWordsKey = "trend:stopwords"
)

func dedupKey(userID, query string) string {
	return fmt.Sprintf("trend:dedup:%s:%s", userID, query)
}
