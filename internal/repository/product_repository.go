package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"restapi-test/internal/model"
)

type ProductFilter struct {
	Search   string
	Category string
	Page     int
	Limit    int
}

func (f ProductFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

var ErrNotFound = errors.New("record not found")

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) List(ctx context.Context, filter ProductFilter) ([]model.Product, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Product{})

	if s := strings.TrimSpace(filter.Search); s != "" {
		escaped := escapeLike(s)
		query = query.Where(`title ILIKE ? ESCAPE '\'`, "%"+escaped+"%")
	}
	if c := strings.TrimSpace(filter.Category); c != "" {
		query = query.Where(`category ILIKE ? ESCAPE '\'`, escapeLike(c))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []model.Product
	err := query.Order("id ASC").Limit(filter.Limit).Offset(filter.Offset()).Find(&products).Error
	if err != nil {
		return nil, 0, err
	}
	if products == nil {
		products = []model.Product{}
	}
	return products, total, nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id uint64) (model.Product, error) {
	var product model.Product
	err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Product{}, ErrNotFound
		}
		return model.Product{}, err
	}
	return product, nil
}

func (r *ProductRepository) Create(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *ProductRepository) Update(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *ProductRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&model.Product{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}
