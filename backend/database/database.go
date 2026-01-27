package database

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"stratus/config"
	"stratus/models"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {
	var err error
	DB, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("Database connected successfully")
	return nil
}

func Migrate() error {
	log.Println("Running database migrations...")
	err := DB.AutoMigrate(
		&models.User{},
		&models.File{},
		&models.FileVersion{},
		&models.Activity{},
	)
	if err != nil {
		return err
	}
	log.Println("Database migrations completed")
	return nil
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
