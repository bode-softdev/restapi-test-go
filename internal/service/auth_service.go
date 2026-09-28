package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"restapi-test/internal/config"
	"restapi-test/internal/model"
	"restapi-test/internal/repository"
	"restapi-test/internal/response"
)

type Claims struct {
	UserID   uint   `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AuthenticationToken string
	RefreshToken        string
	TokenType           string
	ExpiresIn           int64
}

type AuthService struct {
	users  *repository.UserRepository
	tokens *repository.TokenRepository
	cfg    *config.Config
}

func NewAuthService(users *repository.UserRepository, tokens *repository.TokenRepository, cfg *config.Config) *AuthService {
	return &AuthService{users: users, tokens: tokens, cfg: cfg}
}

func (s *AuthService) Register(ctx context.Context, username, password, confirmation string) (response.UserResponse, error) {
	if !usernamePattern.MatchString(username) {
		return response.UserResponse{}, ErrUsernameInvalid
	}
	if password != confirmation {
		return response.UserResponse{}, ErrPasswordMismatch
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BcryptCost)
	if err != nil {
		return response.UserResponse{}, err
	}
	user := model.User{Username: username, PasswordHash: string(hash), Role: "user"}
	if err := s.users.Create(ctx, &user); err != nil {
		return response.UserResponse{}, err
	}
	return response.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (TokenPair, response.UserResponse, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return TokenPair{}, response.UserResponse{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, response.UserResponse{}, repository.ErrInvalidCreds
	}
	pair, err := s.issuePair(ctx, user)
	if err != nil {
		return TokenPair{}, response.UserResponse{}, err
	}
	return pair, response.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (TokenPair, error) {
	record, err := s.tokens.FindValid(ctx, hashToken(rawToken))
	if err != nil {
		return TokenPair{}, err
	}
	user, err := s.users.FindByID(ctx, record.UserID)
	if err != nil {
		return TokenPair{}, repository.ErrTokenNotFound
	}
	if err := s.tokens.Revoke(ctx, record.TokenHash); err != nil {
		return TokenPair{}, err
	}
	return s.issuePair(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	return s.tokens.Revoke(ctx, hashToken(rawToken))
}

func (s *AuthService) ParseAccessToken(rawToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (s *AuthService) issuePair(ctx context.Context, user model.User) (TokenPair, error) {
	now := time.Now()
	expires := now.Add(s.cfg.AccessTokenTTL)

	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			Issuer:    "restapi-test",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return TokenPair{}, err
	}

	refresh, err := randomToken(32)
	if err != nil {
		return TokenPair{}, err
	}
	record := model.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(refresh),
		ExpiresAt: now.Add(s.cfg.RefreshTTL),
		CreatedAt: now,
	}
	if err := s.tokens.Store(ctx, &record); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AuthenticationToken: access,
		RefreshToken:        refresh,
		TokenType:           "Bearer",
		ExpiresIn:           int64(time.Until(expires).Seconds()),
	}, nil
}

var (
	ErrPasswordMismatch = errors.New("password_confirmation tidak cocok")
	ErrTokenInvalid     = errors.New("token tidak valid")
	ErrTokenExpired     = errors.New("token kedaluwarsa")
	ErrUsernameInvalid  = errors.New("username tidak valid")
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,30}$`)

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
