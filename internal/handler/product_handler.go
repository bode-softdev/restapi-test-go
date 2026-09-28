package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"restapi-test/internal/middleware"
	"restapi-test/internal/repository"
	"restapi-test/internal/response"
	"restapi-test/internal/service"
)

type ProductHandler struct {
	products *service.ProductService
}

func NewProductHandler(products *service.ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

type productQuery struct {
	Search   string
	Category string
	Page     int
	Limit    int
}

func (q productQuery) toFilter() repository.ProductFilter {
	return repository.ProductFilter{
		Search:   q.Search,
		Category: q.Category,
		Page:     q.Page,
		Limit:    q.Limit,
	}
}

// List god products
// @Summary Ambil semua produk
// @Description Mengembalikan daftar produk dengan pencarian, filter kategori, dan pagination
// @Tags products
// @Accept json
// @Produce json
// @Param search query string false "cari produk berdasarkan judul"
// @Param category query string false "filter berdasarkan kategori"
// @Param page query int false "nomor halaman" default(1)
// @Param limit query int false "jumlah item per halaman" default(10)
// @Success 200 {object} response.ListPayload{data=[]response.ProductResponse,meta=response.Meta}
// @Failure 400 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/products [get]
func (h *ProductHandler) List(c *gin.Context) {
	query, err := parseProductQuery(c)
	if err != nil {
		response.Fail(c, 400, response.CodeInvalidQuery, err.Error(), nil)
		return
	}

	items, meta, err := h.products.List(c.Request.Context(), query.toFilter())
	if err != nil {
		response.Fail(c, 500, response.CodeInternal, "gagal mengambil produk", nil)
		return
	}
	response.List(c, items, meta)
}

// Get god product by id
// @Summary Ambil detail satu produk
// @Description Mengembalikan detail produk berdasarkan ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "ID produk"
// @Success 200 {object} response.DataPayload{data=response.ProductResponse}
// @Failure 404 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/products/{id} [get]
func (h *ProductHandler) GetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	product, err := h.products.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Fail(c, 404, response.CodeNotFound, "produk tidak ditemukan", map[string]any{"id": id})
			return
		}
		response.Fail(c, 500, response.CodeInternal, "gagal mengambil produk", nil)
		return
	}
	response.OK(c, product)
}

// Create god product
// @Summary Tambah produk baru
// @Description Endpoint butuh autentikasi. Rate limit 1 permintaan per 5 detik.
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param product body response.ProductRequest true "data produk"
// @Success 201 {object} response.DataPayload{data=response.ProductResponse}
// @Failure 400 {object} response.ErrorPayload
// @Failure 401 {object} response.ErrorPayload
// @Failure 429 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var req response.ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, response.CodeValidation, "data produk tidak valid", response.ValidationDetails(err))
		return
	}

	actor, _ := currentActor(c)
	product, err := h.products.Create(c.Request.Context(), service.ProductInput{
		Title:       strings.TrimSpace(req.Title),
		Price:       req.Price,
		Description: strings.TrimSpace(req.Description),
		Category:    strings.TrimSpace(req.Category),
		Images:      req.Images,
	}, actor)
	if err != nil {
		response.Fail(c, 500, response.CodeInternal, "gagal menyimpan produk", nil)
		return
	}
	response.Created(c, product)
}

// Update god product
// @Summary Edit produk
// @Description Endpoint butuh autentikasi. Rate limit 1 permintaan per 5 detik.
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID produk"
// @Param product body response.ProductUpdateRequest true "data produk"
// @Success 200 {object} response.DataPayload{data=response.ProductResponse}
// @Failure 400 {object} response.ErrorPayload
// @Failure 401 {object} response.ErrorPayload
// @Failure 404 {object} response.ErrorPayload
// @Failure 429 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req response.ProductUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, response.CodeValidation, "data produk tidak valid", response.ValidationDetails(err))
		return
	}

	existing, err := h.products.GetByIDFresh(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Fail(c, 404, response.CodeNotFound, "produk tidak ditemukan", map[string]any{"id": id})
			return
		}
		response.Fail(c, 500, response.CodeInternal, "gagal mengambil produk", nil)
		return
	}

	input := service.ProductInput{
		Title:       existing.Title,
		Price:       existing.Price,
		Description: existing.Description,
		Category:    existing.Category,
		Images:      existing.Images,
	}
	if req.Title != nil {
		input.Title = strings.TrimSpace(*req.Title)
	}
	if req.Price != nil {
		input.Price = *req.Price
	}
	if req.Description != nil {
		input.Description = strings.TrimSpace(*req.Description)
	}
	if req.Category != nil {
		input.Category = strings.TrimSpace(*req.Category)
	}
	if req.Images != nil {
		input.Images = *req.Images
	}

	if strings.TrimSpace(input.Title) == "" {
		response.Fail(c, 400, response.CodeValidation, "data produk tidak valid", map[string]any{"title": "wajib diisi"})
		return
	}
	if input.Price <= 0 {
		response.Fail(c, 400, response.CodeValidation, "data produk tidak valid", map[string]any{"price": "harus lebih besar dari 0"})
		return
	}
	if strings.TrimSpace(input.Category) == "" {
		response.Fail(c, 400, response.CodeValidation, "data produk tidak valid", map[string]any{"category": "wajib diisi"})
		return
	}
	if len(input.Images) == 0 {
		response.Fail(c, 400, response.CodeValidation, "data produk tidak valid", map[string]any{"images": "minimal 1 gambar"})
		return
	}

	actor, _ := currentActor(c)
	product, err := h.products.Update(c.Request.Context(), id, input, actor)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Fail(c, 404, response.CodeNotFound, "produk tidak ditemukan", map[string]any{"id": id})
			return
		}
		response.Fail(c, 500, response.CodeInternal, "gagal memperbarui produk", nil)
		return
	}
	response.OK(c, product)
}

// Delete god product
// @Summary Hapus produk
// @Description Endpoint butuh autentikasi. Rate limit 1 permintaan per 5 detik.
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID produk"
// @Success 200 {object} response.MessagePayload
// @Failure 401 {object} response.ErrorPayload
// @Failure 404 {object} response.ErrorPayload
// @Failure 429 {object} response.ErrorPayload
// @Failure 500 {object} response.ErrorPayload
// @Router /api/products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.products.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Fail(c, 404, response.CodeNotFound, "produk tidak ditemukan", map[string]any{"id": id})
			return
		}
		response.Fail(c, 500, response.CodeInternal, "gagal menghapus produk", nil)
		return
	}
	response.Deleted(c, "produk berhasil dihapus")
}

func parseProductQuery(c *gin.Context) (productQuery, error) {
	query := productQuery{Page: 1, Limit: 10}

	if raw := c.Query("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 {
			return query, errors.New("query page harus angka bulat positif")
		}
		query.Page = page
	}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 {
			return query, errors.New("query limit harus angka bulat positif")
		}
		if limit > 100 {
			return query, errors.New("query limit maksimal 100")
		}
		query.Limit = limit
	}
	query.Search = strings.TrimSpace(c.Query("search"))
	query.Category = strings.TrimSpace(c.Query("category"))

	if len(query.Category) > 100 {
		return query, errors.New("query category maksimal 100 karakter")
	}
	return query, nil
}

func parseID(c *gin.Context) (uint64, bool) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, 400, response.CodeInvalidQuery, "id harus angka bulat positif", map[string]any{"id": raw})
		return 0, false
	}
	return id, true
}

func currentActor(c *gin.Context) (service.Actor, bool) {
	return middleware.CurrentUser(c)
}
