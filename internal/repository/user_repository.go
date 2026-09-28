package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"restapi-test/internal/model"
)

var (
	ErrUsernameTaken = errors.New("username sudah dipakai")
	ErrInvalidCreds  = errors.New("username atau password salah")
	ErrTokenNotFound = errors.New("refresh token tidak valid")
	ErrTokenExpired  = errors.New("refresh token kedaluwarsa")
	ErrTokenRevoked  = errors.New("refresh token sudah dicabut")
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUsernameTaken
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "username = ?", username).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, ErrInvalidCreds
		}
		return model.User{}, err
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, err
	}
	return user, nil
}

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Store(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *TokenRepository) FindValid(ctx context.Context, hash string) (model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.db.WithContext(ctx).First(&token, "token_hash = ?", hash).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.RefreshToken{}, ErrTokenNotFound
		}
		return model.RefreshToken{}, err
	}
	if token.RevokedAt != nil {
		return model.RefreshToken{}, ErrTokenRevoked
	}
	if time.Now().After(token.ExpiresAt) {
		return model.RefreshToken{}, ErrTokenExpired
	}
	return token, nil
}

func (r *TokenRepository) Revoke(ctx context.Context, hash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", now).Error
}

func (r *TokenRepository) RevokeAllForUser(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func isUniqueViolation(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate key value") || strings.Contains(msg, "23505")
}
