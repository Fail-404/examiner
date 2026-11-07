package main

import (
	"os"
	"testing"
	"time"

	"github.com/pavelanni/examiner/internal/llm"
)

// TestLLMReliabilityConfiguration проверяет приоритет CLI, окружения, YAML и значений по умолчанию.
func TestLLMReliabilityConfiguration(t *testing.T) {
	for _, source := range []string{"defaults", "file", "environment", "flags"} {
		t.Run(source, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for _, key := range []string{"EXAMINER_LLM_TIMEOUT", "EXAMINER_LLM_MAX_RETRIES", "EXAMINER_LLM_MAX_CONCURRENCY"} {
				t.Setenv(key, "")
			}
			timeout, retries, concurrency := llm.DefaultTimeout, llm.DefaultMaxRetries, llm.DefaultMaxConcurrency
			if source != "defaults" {
				if err := os.WriteFile("examiner.yaml", []byte("llm-timeout: 7s\nllm-max-retries: 2\nllm-max-concurrency: 6\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				timeout, retries, concurrency = 7*time.Second, 2, 6
			}
			if source == "environment" || source == "flags" {
				t.Setenv("EXAMINER_LLM_TIMEOUT", "3s")
				t.Setenv("EXAMINER_LLM_MAX_RETRIES", "0")
				t.Setenv("EXAMINER_LLM_MAX_CONCURRENCY", "2")
				timeout, retries, concurrency = 3*time.Second, 0, 2
			}
			cmd := serveCmd()
			if source == "flags" {
				if err := cmd.ParseFlags([]string{"--llm-timeout=5s", "--llm-max-retries=1", "--llm-max-concurrency=8"}); err != nil {
					t.Fatal(err)
				}
				timeout, retries, concurrency = 5*time.Second, 1, 8
			}
			v := viperForCmd(cmd)
			if got := v.GetDuration("llm-timeout"); got != timeout {
				t.Errorf("timeout=%v, want %v", got, timeout)
			}
			if got := v.GetInt("llm-max-retries"); got != retries {
				t.Errorf("retries=%d, want %d", got, retries)
			}
			if got := v.GetInt("llm-max-concurrency"); got != concurrency {
				t.Errorf("concurrency=%d, want %d", got, concurrency)
			}
		})
	}
}
