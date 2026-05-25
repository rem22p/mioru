package store

import (
	"context"
	"encoding/json"
	"fmt"

	"mioru/internal/model"
)

// ── Categories ──

// GetCategories returns all categories organized as a tree.
func (s *PostgresStore) GetCategories(ctx context.Context) ([]model.Category, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, parent_id, name, slug, criteria, sort_order FROM categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var all []model.Category
	for rows.Next() {
		var c model.Category
		var parentID *int
		var criteriaJSON string
		if err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &criteriaJSON, &c.SortOrder); err != nil {
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

	return buildCategoryTree(all), nil
}

// GetCategoriesFlat returns all categories as a flat list with ParentID references.
func (s *PostgresStore) GetCategoriesFlat(ctx context.Context) ([]model.Category, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, parent_id, name, slug, criteria, sort_order FROM categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var all []model.Category
	for rows.Next() {
		var c model.Category
		var parentID *int
		var criteriaJSON string
		if err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &criteriaJSON, &c.SortOrder); err != nil {
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
