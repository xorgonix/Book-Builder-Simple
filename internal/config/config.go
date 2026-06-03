package config

import (
	"bufio"
	"os"
	"strings"
)

const DefaultPath = "bookbuilder.config"

type Config struct {
	values map[string]string
}

func Load() Config {
	cfg := Config{values: defaultValues()}
	file, err := os.Open(DefaultPath)
	if err != nil {
		return cfg
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = normalizeKey(key)
		if key == "" {
			continue
		}
		cfg.values[key] = strings.TrimSpace(value)
	}
	return cfg
}

func (c Config) Get(key string) string {
	return c.values[normalizeKey(key)]
}

func (c Config) GetEnv(key string) string {
	envKey := strings.ToUpper(normalizeKey(key))
	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		return value
	}
	return c.Get(key)
}

func defaultValues() map[string]string {
	return map[string]string{
		"llm_mock":        "0",
		"local_llm_url":   "http://localhost:1234/v1/chat/completions",
		"local_llm_model": "qwen/qwen3.5-9b",
		"book_author":     "",
		"book_language":   "en",
	}
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, "-", "_")
	return key
}
