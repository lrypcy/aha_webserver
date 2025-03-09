package database

import (
	"errors"
	"fmt"

	"github.com/lrypcy/aha_webserver/internal/model"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	kMysql = "mysql"
	kPostGreSql = "postgresql"
	kSqlite     = "sqlite"
)

var db *gorm.DB

func initMysql() error {
	dbname := viper.GetString("database.dbname")
	// First connect without dbname to create database if needed
	createDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
	)
	
	tempDB, err := gorm.Open(mysql.Open(createDSN), &gorm.Config{})
	if err == nil {
		tempDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbname))
	}

	// Then connect with dbname
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		dbname,
	)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	
	// 添加自动迁移
	err = db.AutoMigrate(
		&model.Job{},
		&model.Task{},
		&model.User{},
	)
	return err
}

func initPostGreSql() error {
	dbname := viper.GetString("database.dbname")
	// First connect to default database to create target db
	createDSN := fmt.Sprintf("host=%s user=%s password=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		viper.GetString("database.host"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetInt("database.port"),
	)
	
	tempDB, err := gorm.Open(postgres.Open(createDSN), &gorm.Config{})
	if err == nil {
		tempDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
	}

	// Then connect to the created database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		viper.GetString("database.host"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		dbname,
		viper.GetInt("database.port"),
	)
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	
	// 调整迁移顺序，先创建 Job 表再创建 Task 表
	if err := db.AutoMigrate(
		&model.Job{},  // 必须放在 Task 前面
		&model.Task{},
		&model.User{},
	); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}
	return nil
}

func initSqlite() error {
	fmt.Println("init sqlite")
	dsn := viper.GetString("database.dbname")
	fmt.Println(dsn)
	var err error
	db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	db.AutoMigrate(
		&model.Job{},   // 移到 Task 前面
		&model.Task{},
		&model.User{},
	)
	return err
}

func InitDB() {
	var err error
	switch database_type := viper.GetString("database.type"); database_type {
	case kMysql:
		err = initMysql()
	case kPostGreSql:
		err = initPostGreSql()
	case kSqlite:
		err = initSqlite()
	default:
		err = errors.New("Unsupported database")
	}
	if err != nil {
		panic("fail to connect database")
	}
}

func DB() *gorm.DB {return db}