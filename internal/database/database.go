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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.dbname"),
	)
	fmt.Println(dsn)
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	// TODO
	// sqlDB, err := db.DB()
	// sqlDB.SetMaxOpenConns(10) // 设置最大打开的连接数
	// sqlDB.SetMaxIdleConns(5)  // 设置连接池中的最大闲置连接数
	// sqlDB.SetConnMaxLifetime(time.Hour) // 设置连接的最大可复用时间
	return err
}

func initPostGreSql() error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		viper.GetString("database.host"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.dbname"),
		viper.GetInt("database.port"),
	)
	fmt.Println(dsn)
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	return err
}

func initSqlite() error {
	fmt.Println("init sqlite")
	dsn := viper.GetString("database.dbname")
	fmt.Println(dsn)
	var err error
	db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	db.AutoMigrate(&model.Task{})
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