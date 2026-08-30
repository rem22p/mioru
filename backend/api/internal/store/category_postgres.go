package store

import (
	"context"
	"encoding/json"
	"fmt"

	"mioru/internal/model"
)

// ── Categories ──

const categoryQuery = `SELECT
		c.id, c.parent_id, c.name, c.slug, c.criteria, c.sort_order,
		(SELECT pi.url
		 FROM products p
		 JOIN product_images pi ON pi.product_id = p.id
		 WHERE p.category_id = c.id
		    OR p.category_id IN (SELECT id FROM categories WHERE parent_id = c.id)
		 ORDER BY p.stock_quantity DESC, pi.sort_order
		 LIMIT 1) AS cover_image,
		(SELECT count(*) FROM products p
		 WHERE p.category_id = c.id
		    OR p.category_id IN (SELECT id FROM categories WHERE parent_id = c.id)) AS products_count
	FROM categories c
	ORDER BY c.sort_order, c.id`

// GetCategories returns all categories organized as a tree.
func (s *PostgresStore) GetCategories(ctx context.Context) ([]model.Category, error) {
	all, err := s.queryCategories(ctx)
	if err != nil {
		return nil, err
	}
	return buildCategoryTree(all), nil
}

// GetCategoriesFlat returns all categories as a flat list with ParentID references.
func (s *PostgresStore) GetCategoriesFlat(ctx context.Context) ([]model.Category, error) {
	return s.queryCategories(ctx)
}

func (s *PostgresStore) queryCategories(ctx context.Context) ([]model.Category, error) {
	rows, err := s.pool.Query(ctx, categoryQuery)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var all []model.Category
	for rows.Next() {
		var c model.Category
		var parentID *int
		var criteriaJSON string
		if err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &criteriaJSON, &c.SortOrder, &c.CoverImage, &c.ProductsCount); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		c.ParentID = parentID
		json.Unmarshal([]byte(criteriaJSON), &c.Criteria)
		if c.Criteria == nil {
			c.Criteria = []string{}
		}
		all = append(all, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return all, nil
}

func buildCategoryTree(cats []model.Category) []model.Category {
	byID := make(map[int]*model.Category)
	for i := range cats {
		cats[i].Children = []model.Category{}
		byID[cats[i].ID] = &cats[i]
	}

	var roots []*model.Category
	for i := range cats {
		c := &cats[i]
		if c.ParentID == nil {
			roots = append(roots, c)
		} else if parent, ok := byID[*c.ParentID]; ok {
			parent.Children = append(parent.Children, *c)
		}
	}

	result := make([]model.Category, len(roots))
	for i, r := range roots {
		cleanEmptyChildren(r)
		result[i] = *r
	}
	return result
}

func cleanEmptyChildren(c *model.Category) {
	for i := range c.Children {
		cleanEmptyChildren(&c.Children[i])
	}
	if len(c.Children) == 0 {
		c.Children = nil
	}
}
