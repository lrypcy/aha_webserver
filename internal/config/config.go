// Package config 集中加载并校验服务配置。
// 优先级：环境变量 AHA_* > configs/*.yml > 内置默认值。
package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// EnvPrefix 是所有环境变量的前缀。
const EnvPrefix = "AHA"

// Server 是 HTTP 服务器配置。
type Server struct {
	Port            int
	Env             string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	BodyLimit       int
}

// Database 是数据库连接配置。
type Database struct {
	Type            string
	Host            string
	Port            int
	Username        string
	Password        string
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// JWT 是令牌签发配置。
type JWT struct {
	Secret          string
	Expiration      time.Duration
	Issuer          string
	EphemeralSecret bool // 开发模式下自动生成的临时密钥，重启后失效
}

// Log 是日志配置。
type Log struct{ Level string }

// CORS 是跨域配置，Origins 为逗号分隔列表。
type CORS struct{ Origins string }

// RateLimit 是限流配置。
type RateLimit struct {
	Max        int
	Expiration time.Duration
}

// Config 聚合全部配置。
type Config struct {
	Server    Server
	Database  Database
	JWT       JWT
	Log       Log
	CORS      CORS
	RateLimit RateLimit
}

// Load 读取配置文件并应用环境变量覆盖。
// 配置文件不存在时退化为纯默认值，方便零配置本地启动。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
	}

	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	c := &Config{
		Server: Server{
			Port:            v.GetInt("server.port"),
			Env:             v.GetString("server.env"),
			ReadTimeout:     v.GetDuration("server.read_timeout"),
			WriteTimeout:    v.GetDuration("server.write_timeout"),
			ShutdownTimeout: v.GetDuration("server.shutdown_timeout"),
			BodyLimit:       v.GetInt("server.body_limit"),
		},
		Database: Database{
			Type:            strings.ToLower(v.GetString("database.type")),
			Host:            v.GetString("database.host"),
			Port:            v.GetInt("database.port"),
			Username:        v.GetString("database.username"),
			Password:        v.GetString("database.password"),
			DBName:          v.GetString("database.dbname"),
			MaxOpenConns:    v.GetInt("database.max_open_conns"),
			MaxIdleConns:    v.GetInt("database.max_idle_conns"),
			ConnMaxLifetime: v.GetDuration("database.conn_max_lifetime"),
		},
		JWT: JWT{
			Secret:     v.GetString("jwt.secret"),
			Expiration: v.GetDuration("jwt.expiration"),
			Issuer:     v.GetString("jwt.issuer"),
		},
		Log:       Log{Level: v.GetString("log.level")},
		CORS:      CORS{Origins: v.GetString("cors.origins")},
		RateLimit: RateLimit{Max: v.GetInt("rate_limit.max"), Expiration: v.GetDuration("rate_limit.expiration")},
	}

	// viper 的 AutomaticEnv 在结构体赋值场景下行为不稳定，这里对关键项显式兜底，
	// 保证「环境变量一定优先」这一契约可测试、可预测。
	applyEnvOverrides(c)

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.env", "dev")
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("server.body_limit", 1<<20)
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.username", "postgres")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "aha.db")
	v.SetDefault("database.max_open_conns", 20)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("jwt.secret", "")
	v.SetDefault("jwt.expiration", "24h")
	v.SetDefault("jwt.issuer", "aha_webserver")
	v.SetDefault("log.level", "info")
	v.SetDefault("cors.origins", "*")
	v.SetDefault("rate_limit.max", 100)
	v.SetDefault("rate_limit.expiration", "1m")
}

func applyEnvOverrides(c *Config) {
	str := func(name string, dst *string) {
		if v, ok := os.LookupEnv(EnvPrefix + "_" + name); ok {
			*dst = v
		}
	}
	integer := func(name string, dst *int) {
		if v, ok := os.LookupEnv(EnvPrefix + "_" + name); ok {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	duration := func(name string, dst *time.Duration) {
		if v, ok := os.LookupEnv(EnvPrefix + "_" + name); ok {
			if d, err := time.ParseDuration(v); err == nil {
				*dst = d
			}
		}
	}

	integer("SERVER_PORT", &c.Server.Port)
	str("SERVER_ENV", &c.Server.Env)
	duration("SERVER_READ_TIMEOUT", &c.Server.ReadTimeout)
	duration("SERVER_WRITE_TIMEOUT", &c.Server.WriteTimeout)
	duration("SERVER_SHUTDOWN_TIMEOUT", &c.Server.ShutdownTimeout)
	integer("SERVER_BODY_LIMIT", &c.Server.BodyLimit)

	str("DB_TYPE", &c.Database.Type)
	c.Database.Type = strings.ToLower(c.Database.Type)
	str("DB_HOST", &c.Database.Host)
	integer("DB_PORT", &c.Database.Port)
	str("DB_USER", &c.Database.Username)
	str("DB_PASSWORD", &c.Database.Password)
	str("DB_NAME", &c.Database.DBName)
	integer("DB_MAX_OPEN_CONNS", &c.Database.MaxOpenConns)
	integer("DB_MAX_IDLE_CONNS", &c.Database.MaxIdleConns)
	duration("DB_CONN_MAX_LIFETIME", &c.Database.ConnMaxLifetime)

	str("JWT_SECRET", &c.JWT.Secret)
	duration("JWT_EXPIRATION", &c.JWT.Expiration)
	str("JWT_ISSUER", &c.JWT.Issuer)
	str("LOG_LEVEL", &c.Log.Level)
	str("CORS_ORIGINS", &c.CORS.Origins)
	integer("RATE_LIMIT_MAX", &c.RateLimit.Max)
	duration("RATE_LIMIT_EXPIRATION", &c.RateLimit.Expiration)
}

// Validate 校验配置并在开发模式下生成临时 JWT 密钥。
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be in [1, 65535], got %d", c.Server.Port)
	}
	switch c.Database.Type {
	case "sqlite", "mysql", "postgresql":
	default:
		return fmt.Errorf("unsupported database.type %q (want sqlite|mysql|postgresql)", c.Database.Type)
	}
	if c.Database.DBName == "" {
		return errors.New("database.dbname must not be empty")
	}
	if c.JWT.Expiration <= 0 {
		return errors.New("jwt.expiration must be positive")
	}
	if c.JWT.Issuer == "" {
		c.JWT.Issuer = "aha_webserver"
	}
	if c.RateLimit.Max <= 0 {
		c.RateLimit.Max = 100
	}
	if c.RateLimit.Expiration <= 0 {
		c.RateLimit.Expiration = time.Minute
	}

	production := strings.EqualFold(c.Server.Env, "production") || strings.EqualFold(c.Server.Env, "prod")
	if production {
		if len(c.JWT.Secret) < 32 {
			return errors.New("AHA_JWT_SECRET must be set to at least 32 characters in production")
		}
		if c.CORS.Origins == "*" {
			return errors.New("AHA_CORS_ORIGINS must be an explicit allow-list in production")
		}
		if c.Database.Type != "sqlite" && c.Database.Password == "" {
			return errors.New("AHA_DB_PASSWORD must be set in production")
		}
	} else if c.JWT.Secret == "" {
		secret, err := randomSecret(32)
		if err != nil {
			return fmt.Errorf("generate ephemeral jwt secret: %w", err)
		}
		c.JWT.Secret = secret
		c.JWT.EphemeralSecret = true
	}
	return nil
}

// IsProduction 报告当前是否为生产环境。
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Server.Env, "production") || strings.EqualFold(c.Server.Env, "prod")
}

// Addr 返回监听地址。
func (c *Config) Addr() string { return fmt.Sprintf(":%d", c.Server.Port) }

func randomSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
