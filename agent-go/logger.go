package main

import (
	"log"
	"os"
	"strings"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var currentLogLevel LogLevel = LogLevelInfo // 默认INFO级别

// parseLogLevel 解析日志级别字符串
func parseLogLevel(level string) LogLevel {
	level = strings.ToUpper(strings.TrimSpace(level))
	switch level {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	default:
		return LogLevelInfo // 默认INFO
	}
}

// InitLogger 初始化日志级别（从环境变量读取）
func InitLogger() {
	levelStr := os.Getenv("AGENT_LOG_LEVEL")
	if levelStr == "" {
		levelStr = "INFO" // 默认INFO级别
	}
	currentLogLevel = parseLogLevel(levelStr)
	log.Printf("Log level set to: %s", strings.ToUpper(levelStr))
}

// shouldLog 判断是否应该输出日志
func shouldLog(level LogLevel) bool {
	return level >= currentLogLevel
}

// LogDebug 输出DEBUG级别日志
func LogDebug(format string, v ...interface{}) {
	if shouldLog(LogLevelDebug) {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// LogInfo 输出INFO级别日志
func LogInfo(format string, v ...interface{}) {
	if shouldLog(LogLevelInfo) {
		log.Printf("[INFO] "+format, v...)
	}
}

// LogWarn 输出WARN级别日志
func LogWarn(format string, v ...interface{}) {
	if shouldLog(LogLevelWarn) {
		log.Printf("[WARN] "+format, v...)
	}
}

// LogError 输出ERROR级别日志
func LogError(format string, v ...interface{}) {
	if shouldLog(LogLevelError) {
		log.Printf("[ERROR] "+format, v...)
	}
}

// LogFatal 输出FATAL级别日志并退出程序
func LogFatal(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}

// LogPrintf 兼容原有的log.Printf调用（默认INFO级别）
func LogPrintf(format string, v ...interface{}) {
	// 检查是否是DEBUG日志（以"DEBUG:"开头）
	if strings.HasPrefix(format, "DEBUG:") {
		LogDebug(strings.TrimPrefix(format, "DEBUG: "), v...)
		return
	}
	// 检查是否是ERROR日志
	if strings.HasPrefix(format, "ERROR:") {
		LogError(strings.TrimPrefix(format, "ERROR: "), v...)
		return
	}
	// 检查是否是WARNING日志
	if strings.HasPrefix(format, "WARNING:") || strings.HasPrefix(format, "WARN:") {
		LogWarn(strings.TrimPrefix(format, "WARNING: "), v...)
		return
	}
	// 默认INFO级别
	LogInfo(format, v...)
}

// LogPrint 兼容原有的log.Print调用（默认INFO级别）
func LogPrint(v ...interface{}) {
	if shouldLog(LogLevelInfo) {
		log.Print(v...)
	}
}

// LogPrintln 兼容原有的log.Println调用（默认INFO级别）
func LogPrintln(v ...interface{}) {
	if shouldLog(LogLevelInfo) {
		log.Println(v...)
	}
}
