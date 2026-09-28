package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"restapi-test/internal/repository"
	"restapi-test/internal/response"
	"restapi-test/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerResponse struct {
	Message string                `json:"message"`
	Data    response.UserResponse `json:"data"`
}

type loginResponse struct {
	AuthenticationToken string                `json:"authentication_token"`
	RefreshToken        string                `json:"refresh_token"`
	TokenType           string                `json:"token_type" example:"Bearer"`
	ExpiresIn           int64                 `json:"expires_in" example:"900"`
	User                response.UserResponse `json:"user"`
}

// Register god user
// @Summary Register user baru
// @Description Membuat akun baru. Rate limit 3 permintaan per 60 detik.
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body response.RegisterRequest true "data register"
// @Success 201 {object} registerResponse
// @Failure 400 {object} response.ErrorPayload
// @Failure 409 {object} response.ErrorPayload
// @Failure 429 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req response.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeValidation, "data register tidak valid", response.ValidationDetails(err))
		return
	}

	user, err := h.auth.Register(c.Request.Context(),
		strings.TrimSpace(req.Username), req.Password, req.PasswordConfirmation)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeValidation, "data register tidak valid", map[string]any{
				"username": "3-30 karakter, hanya huruf, angka, titik, underscore, atau strip",
			})
		case errors.Is(err, service.ErrPasswordMismatch):
			response.Fail(c, http.StatusBadRequest, response.CodeValidation, "password_confirmation tidak cocok", map[string]any{
				"password_confirmation": "harus sama dengan password",
			})
		case errors.Is(err, repository.ErrUsernameTaken):
			response.Fail(c, http.StatusConflict, response.CodeConflict, "username sudah dipakai", map[string]any{
				"username": req.Username,
			})
		default:
			response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "gagal membuat akun", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, registerResponse{Message: "registrasi berhasil", Data: user})
}

// Login god user
// @Summary Login dan ambil token
// @Description Mengembalikan authentication_token dan refresh_token. Rate limit 3 permintaan per 60 detik.
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body response.LoginRequest true "kredensial login"
// @Success 200 {object} loginResponse
// @Failure 400 {object} response.ErrorPayload
// @Failure 401 {object} response.ErrorPayload
// @Failure 429 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req response.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeValidation, "data login tidak valid", response.ValidationDetails(err))
		return
	}

	pair, user, err := h.auth.Login(c.Request.Context(), strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidCreds) {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "username atau password salah", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "gagal login", nil)
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AuthenticationToken: pair.AuthenticationToken,
		RefreshToken:        pair.RefreshToken,
		TokenType:           pair.TokenType,
		ExpiresIn:           pair.ExpiresIn,
		User:                user,
	})
}

// Refresh god token
// @Summary Tukar refresh token
// @Description Refresh token lama dicabut dan digantikan pair baru (rotasi).
// @Tags auth
// @Accept json
// @Produce json
// @Param token body response.RefreshRequest true "refresh token"
// @Success 200 {object} response.TokenResponse
// @Failure 400 {object} response.ErrorPayload
// @Failure 401 {object} response.ErrorPayload
// @Failure 429 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req response.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeValidation, "refresh_token wajib diisi", response.ValidationDetails(err))
		return
	}

	pair, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, response.CodeTokenInvalid, err.Error(), nil)
		return
	}

	response.OK(c, response.TokenResponse{
		AuthenticationToken: pair.AuthenticationToken,
		RefreshToken:        pair.RefreshToken,
		TokenType:           pair.TokenType,
		ExpiresIn:           pair.ExpiresIn,
	})
}

// Logout god user
// @Summary Cabut refresh token
// @Description Refresh token dicabut sehingga tidak bisa dipakai lagi.
// @Tags auth
// @Accept json
// @Produce json
// @Param token body response.RefreshRequest true "refresh token"
// @Success 200 {object} response.MessagePayload
// @Failure 400 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req response.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeValidation, "refresh_token wajib diisi", response.ValidationDetails(err))
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "gagal logout", nil)
		return
	}
	response.OK(c, gin.H{"message": "logout berhasil"})
}
