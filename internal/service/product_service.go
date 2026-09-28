package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/datatypes"

	"restapi-test/internal/cache"
	"restapi-test/internal/model"
	"restapi-test/internal/repository"
	"restapi-test/internal/response"
)

type ProductInput struct {
	Title       string
	Price       float64
	Description string
	Category    string
	Images      []string
}

type Actor struct {
	ID       uint
	Username string
}

type ProductService struct {
	repo    *repository.ProductRepository
	cache   *cache.Cache
	ttlList time.Duration
	ttlItem time.Duration
}

func NewProductService(repo *repository.ProductRepository, c *cache.Cache, ttlList, ttlItem time.Duration) *ProductService {
	return &ProductService{repo: repo, cache: c, ttlList: ttlList, ttlItem: ttlItem}
}

func (s *ProductService) List(ctx context.Context, filter repository.ProductFilter) ([]response.ProductResponse, response.Meta, error) {
	key := s.cache.ListKey(ctx,
		"search="+filter.Search,
		"category="+filter.Category,
		"page="+strconv.Itoa(filter.Page),
		"limit="+strconv.Itoa(filter.Limit),
	)

	var cached struct {
		Data []response.ProductResponse `json:"data"`
		Meta response.Meta              `json:"meta"`
	}
	if s.cache.GetJSON(ctx, key, &cached) {
		return cached.Data, cached.Meta, nil
	}

	products, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, response.Meta{}, err
	}

	items := make([]response.ProductResponse, 0, len(products))
	for _, p := range products {
		items = append(items, toProductResponse(p))
	}

	meta := buildMeta(total, filter.Page, filter.Limit)
	s.cache.SetJSON(ctx, key, struct {
		Data []response.ProductResponse `json:"data"`
		Meta response.Meta              `json:"meta"`
	}{Data: items, Meta: meta}, s.ttlList)

	return items, meta, nil
}

func (s *ProductService) GetByID(ctx context.Context, id uint64) (response.ProductResponse, error) {
	key := s.cache.ItemKey(ctx, id)

	var cached response.ProductResponse
	if s.cache.GetJSON(ctx, key, &cached) {
		return cached, nil
	}

	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return response.ProductResponse{}, err
	}

	out := toProductResponse(product)
	s.cache.SetJSON(ctx, key, out, s.ttlItem)
	return out, nil
}

func (s *ProductService) GetByIDFresh(ctx context.Context, id uint64) (response.ProductResponse, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return response.ProductResponse{}, err
	}
	return toProductResponse(product), nil
}

func (s *ProductService) Create(ctx context.Context, input ProductInput, actor Actor) (response.ProductResponse, error) {
	now := time.Now()
	product := model.Product{
		Title:       input.Title,
		Price:       input.Price,
		Description: input.Description,
		Category:    input.Category,
		Images:      datatypes.JSONSlice[string](input.Images),
		CreatedAt:   now,
		CreatedBy:   actor.Username,
		CreatedByID: actor.ID,
		UpdatedAt:   now,
		UpdatedBy:   actor.Username,
		UpdatedByID: actor.ID,
	}
	if err := s.repo.Create(ctx, &product); err != nil {
		return response.ProductResponse{}, err
	}
	s.cache.BumpVersion(ctx)
	return toProductResponse(product), nil
}

func (s *ProductService) Update(ctx context.Context, id uint64, input ProductInput, actor Actor) (response.ProductResponse, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return response.ProductResponse{}, err
	}

	product.Title = input.Title
	product.Price = input.Price
	product.Description = input.Description
	product.Category = input.Category
	product.Images = datatypes.JSONSlice[string](input.Images)
	product.UpdatedAt = time.Now()
	product.UpdatedBy = actor.Username
	product.UpdatedByID = actor.ID

	if err := s.repo.Update(ctx, &product); err != nil {
		return response.ProductResponse{}, err
	}
	s.cache.BumpVersion(ctx)
	return toProductResponse(product), nil
}

func (s *ProductService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.cache.BumpVersion(ctx)
	return nil
}

func buildMeta(total int64, page, limit int) response.Meta {
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return response.Meta{
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}

func toProductResponse(p model.Product) response.ProductResponse {
	images := make([]string, 0, len(p.Images))
	images = append(images, p.Images...)
	return response.ProductResponse{
		ID:          p.ID,
		Title:       p.Title,
		Price:       p.Price,
		Description: p.Description,
		Category:    p.Category,
		Images:      images,
		CreatedAt:   p.CreatedAt.Format("2006-01-02 15:04:05"),
		CreatedBy:   p.CreatedBy,
		CreatedByID: p.CreatedByID,
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02 15:04:05"),
		UpdatedBy:   p.UpdatedBy,
		UpdatedByID: p.UpdatedByID,
	}
}

var ErrProductNotFound = errors.New("produk tidak ditemukan")
