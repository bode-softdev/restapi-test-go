package response

import (
	"errors"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationDetailsRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var req ProductRequest
	err := binding.Validator.ValidateStruct(&req)
	require.Error(t, err)

	details := ValidationDetails(err)
	assert.Contains(t, details, "title")
	assert.Equal(t, "wajib diisi", details["title"])
	assert.Contains(t, details, "price")
	assert.Contains(t, details, "category")
	assert.Contains(t, details, "images")
}

func TestValidationDetailsTagMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name  string
		value ProductRequest
		field string
		want  string
	}{
		{
			name:  "title kosong",
			value: ProductRequest{Price: 10, Category: "Clothes", Images: []string{"https://a.test/1.png"}},
			field: "title",
			want:  "wajib diisi",
		},
		{
			name:  "price nol dianggap kosong",
			value: ProductRequest{Title: "x", Category: "Clothes", Images: []string{"https://a.test/1.png"}},
			field: "price",
			want:  "wajib diisi",
		},
		{
			name:  "price negatif ditolak gt",
			value: ProductRequest{Title: "x", Price: -5, Category: "Clothes", Images: []string{"https://a.test/1.png"}},
			field: "price",
			want:  "harus lebih besar dari 0",
		},
		{
			name:  "images kosong dianggap kosong",
			value: ProductRequest{Title: "x", Price: 10, Category: "Clothes"},
			field: "images",
			want:  "wajib diisi",
		},
		{
			name:  "judul terlalu panjang",
			value: ProductRequest{Title: strings.Repeat("a", 200), Price: 10, Category: "Clothes", Images: []string{"https://a.test/1.png"}},
			field: "title",
			want:  "maksimal panjang 191",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := binding.Validator.ValidateStruct(&tc.value)
			require.Error(t, err)
			details := ValidationDetails(err)
			assert.Equal(t, tc.want, details[tc.field])
		})
	}
}

func TestValidationDetailsDiveReportsIndexedField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	value := ProductRequest{
		Title: "x", Price: 10, Category: "Clothes",
		Images: []string{"https://a.test/1.png", "bukan-url"},
	}
	err := binding.Validator.ValidateStruct(&value)
	require.Error(t, err)

	details := ValidationDetails(err)
	assert.Contains(t, details, "images[1]")
	assert.Equal(t, "harus berupa URL yang valid", details["images[1]"])
}

func TestValidationDetailsBadJSON(t *testing.T) {
	details := ValidationDetails(errors.New("unexpected EOF"))
	assert.Contains(t, details, "body")
}

func TestFailAlwaysIncludesErrorKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := newRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = newRequest("GET", "/x")

	Fail(c, 404, CodeNotFound, "tidak ditemukan", nil)

	assert.Equal(t, 404, rec.Code)
	assert.JSONEq(t, `{"error":{"code":"NOT_FOUND","message":"tidak ditemukan"}}`, rec.Body.String())
}

func TestFailIncludesDetailsWhenPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := newRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = newRequest("GET", "/x")

	Fail(c, 400, CodeValidation, "gagal", map[string]any{"title": "wajib diisi"})

	assert.JSONEq(t, `{"error":{"code":"VALIDATION_ERROR","message":"gagal","details":{"title":"wajib diisi"}}}`, rec.Body.String())
}
