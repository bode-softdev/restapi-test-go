package database

import (
	"context"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"restapi-test/internal/config"
	"restapi-test/internal/model"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	var db *gorm.DB
	var err error
	attempts := cfg.ConnectAttempts
	if attempts < 1 {
		attempts = 1
	}
	backoff := cfg.ConnectBackoff
	if backoff <= 0 {
		backoff = time.Second
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		db, err = connectOnce(cfg, gormCfg)
		if err == nil {
			break
		}
		log.Printf("database: menunggu postgres (percobaan %d/%d): %v", attempt, attempts, err)
		if attempt == attempts {
			break
		}
		time.Sleep(backoff)
		if backoff < 10*time.Second {
			backoff *= 2
		}
	}
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func connectOnce(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), gormCfg)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{}, &model.Product{}, &model.RefreshToken{})
}

func Seed(db *gorm.DB) error {
	var userCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		seedUser := model.User{Username: "jhon_doe", PasswordHash: string(hash), Role: "user"}
		if err := db.Create(&seedUser).Error; err != nil {
			return err
		}
		log.Printf("seed: user demo dibuat (username=%s password=%s)", seedUser.Username, "supersecret")
	}

	var productCount int64
	if err := db.Model(&model.Product{}).Count(&productCount).Error; err != nil {
		return err
	}
	if productCount == 0 {
		now := time.Now()
		seedProducts := []model.Product{
			{
				Title: "Awesome T-Shirt", Price: 99.99,
				Description: "High-quality cotton t-shirt", Category: "Clothes",
				Images:    datatypes.JSONSlice[string]{"https://placeimg.com/640/480/any"},
				CreatedAt: now, CreatedBy: "Jhon Doe", CreatedByID: 1,
				UpdatedAt: now, UpdatedBy: "Jhon Doe", UpdatedByID: 1,
			},
			{
				Title: "Leather Jacket", Price: 249.50,
				Description: "Warm genuine leather jacket", Category: "Clothes",
				Images:    datatypes.JSONSlice[string]{"https://placeimg.com/640/480/leather"},
				CreatedAt: now, CreatedBy: "Jhon Doe", CreatedByID: 1,
				UpdatedAt: now, UpdatedBy: "Jhon Doe", UpdatedByID: 1,
			},
			{
				Title: "Wireless Mouse", Price: 45.00,
				Description: "Ergonomic 2.4G wireless mouse", Category: "Electronics",
				Images:    datatypes.JSONSlice[string]{"https://placeimg.com/640/480/mouse"},
				CreatedAt: now, CreatedBy: "Jhon Doe", CreatedByID: 1,
				UpdatedAt: now, UpdatedBy: "Jhon Doe", UpdatedByID: 1,
			},
			{
				Title: "Mechanical Keyboard", Price: 129.99,
				Description: "Hot-swappable mechanical keyboard", Category: "Electronics",
				Images:    datatypes.JSONSlice[string]{"https://placeimg.com/640/480/keyboard"},
				CreatedAt: now, CreatedBy: "Jhon Doe", CreatedByID: 1,
				UpdatedAt: now, UpdatedBy: "Jhon Doe", UpdatedByID: 1,
			},
			{
				Title: "Coffee Beans 1kg", Price: 32.75,
				Description: "Single origin arabica coffee beans", Category: "Food",
				Images:    datatypes.JSONSlice[string]{"https://placeimg.com/640/480/coffee"},
				CreatedAt: now, CreatedBy: "Jhon Doe", CreatedByID: 1,
				UpdatedAt: now, UpdatedBy: "Jhon Doe", UpdatedByID: 1,
			},
		}
		if err := db.Create(&seedProducts).Error; err != nil {
			return err
		}
		log.Printf("seed: %d produk demo dibuat", len(seedProducts))
	}
	return nil
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func PingContext(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
