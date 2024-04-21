package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func EnvMustParseDurationSec(key string) time.Duration {
	value := os.Getenv(key)
	seconds, err := strconv.Atoi(value)
	if err != nil {
		Logger.Fatalf("unexpected format of duration env var: key=%v, value=%v, err=%v", key, value, err)
	}
	return time.Duration(seconds) * time.Second
}

func EnvMustParseString(key string) string {
	value := os.Getenv(key)
	if value == "" {
		Logger.Fatalf("empty string found for required env var: key=%v", key)
	}
	return value
}

func EnvMustParseStringArray(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		Logger.Fatalf("empty string found for required env var: key=%v", key)
	}
	return strings.Split(value, ",")
}

func EnvMustParseInt(key string) int64 {
	value := os.Getenv(key)
	if value == "" {
		Logger.Fatalf("empty string found for required env var: key=%v", key)
	}
	integer, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		Logger.Fatalf("failed to parse integer: key=%v, value=%v", key, value)
	}
	return integer
}

func EnvTryParseBool(key string) bool {
	value := os.Getenv(key)
	if strings.ToLower(value) == "true" {
		return true
	} else if strings.ToLower(value) == "false" {
		return false
	}
	return false
}
