package weaknetwork

import (
	"time"
)

// WeakNetworkConfig Controller弱网环境配置
type WeakNetworkConfig struct {
	// 弱网支持启用配置
	WeakNetworkEnabled bool `env:"WEAK_NETWORK_ENABLED" default:"true"`
	AdaptiveEnabled    bool `env:"ADAPTIVE_ENABLED" default:"true"`

	// 请求超时配置
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" default:"30s"`
	ReadTimeout    time.Duration `env:"READ_TIMEOUT" default:"10s"`
	WriteTimeout   time.Duration `env:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout    time.Duration `env:"IDLE_TIMEOUT" default:"120s"`

	// 连接池配置
	MaxIdleConns        int  `env:"MAX_IDLE_CONNS" default:"100"`
	MaxIdleConnsPerHost int  `env:"MAX_IDLE_CONNS_PER_HOST" default:"10"`
	MaxConnsPerHost     int  `env:"MAX_CONNS_PER_HOST" default:"50"`
	DisableKeepAlives   bool `env:"DISABLE_KEEP_ALIVES" default:"false"`

	// 限流配置
	RateLimitEnabled bool `env:"RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRPS     int  `env:"RATE_LIMIT_RPS" default:"1000"`
	RateLimitBurst   int  `env:"RATE_LIMIT_BURST" default:"2000"`

	// 熔断器配置
	CircuitBreakerEnabled     bool          `env:"CIRCUIT_BREAKER_ENABLED" default:"true"`
	CircuitBreakerTimeout     time.Duration `env:"CIRCUIT_BREAKER_TIMEOUT" default:"60s"`
	CircuitBreakerMaxRequests int           `env:"CIRCUIT_BREAKER_MAX_REQUESTS" default:"5"`
	CircuitBreakerInterval    time.Duration `env:"CIRCUIT_BREAKER_INTERVAL" default:"10s"`

	// 重试配置
	RetryEnabled     bool          `env:"RETRY_ENABLED" default:"true"`
	RetryMaxAttempts int           `env:"RETRY_MAX_ATTEMPTS" default:"3"`
	RetryBaseDelay   time.Duration `env:"RETRY_BASE_DELAY" default:"100ms"`
	RetryMaxDelay    time.Duration `env:"RETRY_MAX_DELAY" default:"5s"`

	// 健康检查配置
	HealthCheckEnabled  bool          `env:"HEALTH_CHECK_ENABLED" default:"true"`
	HealthCheckInterval time.Duration `env:"HEALTH_CHECK_INTERVAL" default:"30s"`
	HealthCheckTimeout  time.Duration `env:"HEALTH_CHECK_TIMEOUT" default:"5s"`

	// 缓存配置
	CacheEnabled bool          `env:"CACHE_ENABLED" default:"true"`
	CacheTTL     time.Duration `env:"CACHE_TTL" default:"30s"`
	CacheMaxSize int           `env:"CACHE_MAX_SIZE" default:"1000"`

	// 监控配置
	MetricsEnabled  bool          `env:"METRICS_ENABLED" default:"true"`
	MetricsInterval time.Duration `env:"METRICS_INTERVAL" default:"10s"`
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *WeakNetworkConfig {
	return &WeakNetworkConfig{
		RequestTimeout:            30 * time.Second,
		ReadTimeout:               10 * time.Second,
		WriteTimeout:              10 * time.Second,
		IdleTimeout:               120 * time.Second,
		MaxIdleConns:              100,
		MaxIdleConnsPerHost:       10,
		MaxConnsPerHost:           50,
		DisableKeepAlives:         false,
		RateLimitEnabled:          true,
		RateLimitRPS:              1000,
		RateLimitBurst:            2000,
		CircuitBreakerEnabled:     true,
		CircuitBreakerTimeout:     60 * time.Second,
		CircuitBreakerMaxRequests: 5,
		CircuitBreakerInterval:    10 * time.Second,
		RetryEnabled:              true,
		RetryMaxAttempts:          3,
		RetryBaseDelay:            100 * time.Millisecond,
		RetryMaxDelay:             5 * time.Second,
		HealthCheckEnabled:        true,
		HealthCheckInterval:       30 * time.Second,
		HealthCheckTimeout:        5 * time.Second,
		CacheEnabled:              true,
		CacheTTL:                  30 * time.Second,
		CacheMaxSize:              1000,
		MetricsEnabled:            true,
		MetricsInterval:           10 * time.Second,
	}
}

// LoadConfigFromEnv 从环境变量加载配置
func LoadConfigFromEnv() *WeakNetworkConfig {
	config := GetDefaultConfig()

	// 这里可以使用环境变量解析库，如envconfig
	// 为了简化，我们直接返回默认配置
	// 在实际项目中，可以使用 github.com/kelseyhightower/envconfig

	return config
}
