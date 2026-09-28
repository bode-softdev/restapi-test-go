package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type ErrorPayload struct {
	Error ErrorBody `json:"error"`
}

type Meta struct {
	Total       int64 `json:"total"`
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

type ListPayload struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

type DataPayload struct {
	Data any `json:"data"`
}

type MessagePayload struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ProductResponse struct {
	ID          uint64   `json:"id" example:"1"`
	Title       string   `json:"title" example:"Awesome T-Shirt"`
	Price       float64  `json:"price" example:"99.99"`
	Description string   `json:"description" example:"High-quality cotton t-shirt"`
	Category    string   `json:"category" example:"Clothes"`
	Images      []string `json:"images" example:"https://placeimg.com/640/480/any"`
	CreatedAt   string   `json:"created_at" example:"2025-01-01 15:01:04"`
	CreatedBy   string   `json:"created_by" example:"Jhon Doe"`
	CreatedByID uint     `json:"created_by_id" example:"1"`
	UpdatedAt   string   `json:"updated_at" example:"2025-01-01 15:01:04"`
	UpdatedBy   string   `json:"updated_by" example:"Jhon Doe"`
	UpdatedByID uint     `json:"updated_by_id" example:"1"`
}

type RegisterRequest struct {
	Username             string `json:"username" binding:"required,min=3,max=30" example:"jhon_doe"`
	Password             string `json:"password" binding:"required,min=8,max=72" example:"supersecret"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required" example:"supersecret"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"jhon_doe"`
	Password string `json:"password" binding:"required" example:"supersecret"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"a3f1c0d9e8b7..."`
}

type TokenResponse struct {
	AuthenticationToken string `json:"authentication_token"`
	RefreshToken        string `json:"refresh_token"`
	TokenType           string `json:"token_type" example:"Bearer"`
	ExpiresIn           int64  `json:"expires_in" example:"900"`
}

type UserResponse struct {
	ID        uint   `json:"id" example:"1"`
	Username  string `json:"username" example:"jhon_doe"`
	CreatedAt string `json:"created_at" example:"2025-01-01 15:01:04"`
}

type ProductRequest struct {
	Title       string   `json:"title" binding:"required,max=191" example:"Awesome T-Shirt"`
	Price       float64  `json:"price" binding:"required,gt=0" example:"99.99"`
	Description string   `json:"description" binding:"max=2000" example:"High-quality cotton t-shirt"`
	Category    string   `json:"category" binding:"required,max=100" example:"Clothes"`
	Images      []string `json:"images" binding:"required,min=1,dive,required,url" example:"https://placeimg.com/640/480/any"`
}

type ProductUpdateRequest struct {
	Title       *string   `json:"title" binding:"omitempty,max=191" example:"Awesome T-Shirt"`
	Price       *float64  `json:"price" binding:"omitempty,gt=0" example:"99.99"`
	Description *string   `json:"description" binding:"omitempty,max=2000" example:"High-quality cotton t-shirt"`
	Category    *string   `json:"category" binding:"omitempty,max=100" example:"Clothes"`
	Images      *[]string `json:"images" binding:"omitempty,min=1,dive,required,url" example:"https://placeimg.com/640/480/any"`
}

const (
	CodeValidation   = "VALIDATION_ERROR"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeInternal     = "INTERNAL_ERROR"
	CodeInvalidQuery = "INVALID_QUERY_PARAMETER"
	CodeTokenInvalid = "TOKEN_INVALID"
	CodeTokenExpired = "TOKEN_EXPIRED"
	CodeTokenRevoked = "TOKEN_REVOKED"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, DataPayload{Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, DataPayload{Data: data})
}

func List(c *gin.Context, data any, meta Meta) {
	c.JSON(http.StatusOK, ListPayload{Data: data, Meta: meta})
}

func Deleted(c *gin.Context, message string) {
	c.JSON(http.StatusOK, MessagePayload{Data: nil, Message: message})
}

func Fail(c *gin.Context, status int, code, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	c.AbortWithStatusJSON(status, ErrorPayload{Error: ErrorBody{Code: code, Message: message, Details: details}})
}

func ValidationDetails(err error) map[string]any {
	details := map[string]any{}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		for _, fieldErr := range verrs {
			details[strings.ToLower(fieldErr.Field())] = describe(fieldErr)
		}
		return details
	}
	var jsonErr *json.UnmarshalTypeError
	if errors.As(err, &jsonErr) {
		details[strings.ToLower(jsonErr.Field)] = "tipe data tidak sesuai"
		return details
	}
	details["body"] = "format JSON tidak valid"
	return details
}

func describe(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "wajib diisi"
	case "min":
		return "minimal panjang " + fieldErr.Param()
	case "max":
		return "maksimal panjang " + fieldErr.Param()
	case "gt":
		return "harus lebih besar dari " + fieldErr.Param()
	case "url":
		return "harus berupa URL yang valid"
	case "alphanum":
		return "hanya boleh huruf dan angka"
	default:
		return "tidak valid"
	}
}
