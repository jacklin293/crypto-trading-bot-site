package db

import (
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	GormDB *gorm.DB
}

func NewDB(dsn string) (db *DB, err error) {
	gormDB, err := gorm.Open(gormMysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return
	}
	db = &DB{
		GormDB: gormDB,
	}
	return
}
