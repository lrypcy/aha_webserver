// Package database 负责连接、连接池、迁移与健康检查。
package database

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/lrypcy/aha_webserver/internal/config"
	"github.com/lrypcy/aha_webserver/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	DBTypeMySQL    = "mysql"
	DBTypePostgres = "postgresql"
	DBTypeSQLite   = "sqlite"
)

var db *gorm.DB

// Init 建立连接、配置连接池并执行自动迁移。
// 不再自动 CREATE DATABASE：应用账号应遵循最小权限，库需由运维/迁移工具预先创建。
func Init(cfg config.Database) (*gorm.DB, error) {
	dial, err := dialector(cfg)
	if err != nil {
		return nil, err
	}

	gdb, err := gorm.Open(dial, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.Type, err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("acquire sql.DB: %w", err)
	}
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 20
	}
	// sqlite 内存库每个连接都有独立实例，必须限制为单连接。
	if cfg.Type == DBTypeSQLite && strings.Contains(cfg.DBName, "memory") {
		maxOpen = 1
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := migrate(gdb); err != nil {
		return nil, err
	}

	db = gdb
	return gdb, nil
}

// Migrate 导出以便测试与运维脚本复用。
func Migrate(gdb *gorm.DB) error { return migrate(gdb) }

func migrate(gdb *gorm.DB) error {
	// 顺序固定：父表 Job 必须先于引用它的 Task / JobResult。
	if err := gdb.AutoMigrate(
		&model.Job{},
		&model.Task{},
		&model.JobResult{},
		&model.User{},
		&model.SysConfig{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

func dialector(cfg config.Database) (gorm.Dialector, error) {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	switch cfg.Type {
	case DBTypeSQLite:
		return sqlite.Open(cfg.DBName), nil
	case DBTypeMySQL:
		// 用官方驱动构造 DSN，自动转义密码中的 @ : / 等特殊字符。
		c := mysqldriver.NewConfig()
		c.User = cfg.Username
		c.Passwd = cfg.Password
		c.Net = "tcp"
		c.Addr = addr
		c.DBName = cfg.DBName
		c.ParseTime = true
		c.Loc = time.Local
		c.Params = map[string]string{"charset": "utf8mb4"}
		return mysql.Open(c.FormatDSN()), nil
	case DBTypePostgres:
		u := &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(cfg.Username, cfg.Password),
			Host:     addr,
			Path:     "/" + cfg.DBName,
			RawQuery: "sslmode=disable&TimeZone=UTC",
		}
		return postgres.Open(u.String()), nil
	default:
		return nil, fmt.Errorf("unsupported database type %q", cfg.Type)
	}
}

// DB 返回已初始化的连接；Init 之前调用会返回 nil。
func DB() *gorm.DB { return db }

// Close 关闭底层连接池。
func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health 供 /ready 探活使用。
func Health() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
