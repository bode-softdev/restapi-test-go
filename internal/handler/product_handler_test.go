package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ctxWithQuery(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/products?"+rawQuery, nil)
	return c
}

func TestParseProductQueryDefaults(t *testing.T) {
	query, err := parseProductQuery(ctxWithQuery(""))
	require.NoError(t, err)
	assert.Equal(t, 1, query.Page)
	assert.Equal(t, 10, query.Limit)
	assert.Empty(t, query.Search)
	assert.Empty(t, query.Category)
}

func TestParseProductQueryFull(t *testing.T) {
	query, err := parseProductQuery(ctxWithQuery("search=shirt&category=Clothes&page=2&limit=25"))
	require.NoError(t, err)
	assert.Equal(t, "shirt", query.Search)
	assert.Equal(t, "Clothes", query.Category)
	assert.Equal(t, 2, query.Page)
	assert.Equal(t, 25, query.Limit)
}

func TestParseProductQueryRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"page bukan angka": "page=abc",
		"page nol":         "page=0",
		"page negatif":     "page=-1",
		"limit nol":        "limit=0",
		"limit kelewat":    "limit=101",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parseProductQuery(ctxWithQuery(raw))
			assert.Error(t, err)
		})
	}
}

func TestParseProductQueryTrimsWhitespace(t *testing.T) {
	query, err := parseProductQuery(ctxWithQuery("search=%20shirt%20&category=%20Food%20"))
	require.NoError(t, err)
	assert.Equal(t, "shirt", query.Search)
	assert.Equal(t, "Food", query.Category)
}

func TestParseID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/products/42", nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	id, ok := parseID(c)
	require.True(t, ok)
	assert.Equal(t, uint64(42), id)
	assert.Equal(t, 200, rec.Code)
}

func TestParseIDInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, raw := range []string{"abc", "0", "-1", "1.5"} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/products/x", nil)
		c.Params = gin.Params{{Key: "id", Value: raw}}

		_, ok := parseID(c)
		assert.False(t, ok, raw)
		assert.Equal(t, 400, rec.Code, raw)
		assert.True(t, strings.Contains(rec.Body.String(), "INVALID_QUERY_PARAMETER"))
	}
}
