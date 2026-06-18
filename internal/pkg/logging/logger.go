package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	service  string
	minLevel Level
}

func NewLogger(service string, levelText string) *Logger {
	return &Logger{
		service:  service,
		minLevel: ParseLevel(levelText),
	}
}

func ParseLevel(levelText string) Level {
	switch strings.ToLower(strings.TrimSpace(levelText)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l *Logger) Debug(message string, kv ...any) { l.write(LevelDebug, "DEBUG", message, kv...) }
func (l *Logger) Info(message string, kv ...any)  { l.write(LevelInfo, "INFO", message, kv...) }
func (l *Logger) Warn(message string, kv ...any)  { l.write(LevelWarn, "WARN", message, kv...) }
func (l *Logger) Error(message string, kv ...any) { l.write(LevelError, "ERROR", message, kv...) }

func (l *Logger) write(level Level, levelText string, message string, kv ...any) {
	if l == nil || level < l.minLevel {
		return
	}

	record := map[string]any{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   levelText,
		"service": l.service,
		"msg":     message,
	}
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprintf("field_%d", i)
		if i < len(kv) {
			if k, ok := kv[i].(string); ok && strings.TrimSpace(k) != "" {
				key = k
			}
		}
		if i+1 < len(kv) {
			record[key] = kv[i+1]
		} else {
			record[key] = "<missing>"
		}
	}

	out, err := json.Marshal(record)
	if err != nil {
		log.Printf(`{"ts":"%s","level":"ERROR","service":"%s","msg":"marshal log failed","error":"%v"}`, time.Now().UTC().Format(time.RFC3339Nano), l.service, err)
		return
	}
	log.Print(string(out))
}
