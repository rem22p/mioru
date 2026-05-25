package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"mioru/internal/model"
)

// parseProductFromForm extracts a Product from multipart form values.
func parseProductFromForm(r *http.Request) (model.Product, error) {
	p := model.Product{
		Slug:        strings.TrimSpace(r.FormValue("slug")),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Brand:       strings.TrimSpace(r.FormValue("brand")),
		Color:       strings.TrimSpace(r.FormValue("color")),
		Model:       strings.TrimSpace(r.FormValue("model")),
		Fit:         strings.TrimSpace(r.FormValue("fit")),
		Material:    strings.TrimSpace(r.FormValue("material")),
	}

	if v, err := strconv.Atoi(r.FormValue("price")); err == nil {
		p.Price = v
	}
	if v, err := strconv.Atoi(r.FormValue("xp_reward")); err == nil {
		p.XPReward = v
	}
	if v, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64); err == nil {
		p.CategoryID = v
	}
	p.InStock = r.FormValue("in_stock") == "true" || r.FormValue("in_stock") == "1"

	p.Status = strings.TrimSpace(r.FormValue("status"))
	if p.Status == "" {
		if p.InStock {
			p.Status = "in_stock"
		} else {
			p.Status = "none"
		}
	}
	if v, err := strconv.Atoi(r.FormValue("stock_quantity")); err == nil {
		p.StockQty = v
	}

	if p.Slug == "" {
		return p, fmt.Errorf("slug is required")
	}
	if p.Name == "" {
		return p, fmt.Errorf("name is required")
	}
	if p.CategoryID <= 0 {
		return p, fmt.Errorf("category_id is required")
	}

	// Sizes
	if r.Form["sizes[]"] != nil {
		p.Sizes = r.Form["sizes[]"]
	} else if r.Form["sizes"] != nil {
		p.Sizes = r.Form["sizes"]
	}

	// Care instructions
	if r.Form["care[]"] != nil {
		p.Care = r.Form["care[]"]
	}

	// Size chart — parse indexed fields like size_chart[0][label], size_chart[0][chest], etc.
	chartMap := make(map[int]*model.SizeChartRow)
	for key, values := range r.Form {
		if !strings.HasPrefix(key, "size_chart[") {
			continue
		}
		// Parse "size_chart[0][label]" → index=0, field="label"
		rest := strings.TrimPrefix(key, "size_chart[")
		closeBracket := strings.Index(rest, "]")
		if closeBracket < 0 {
			continue
		}
		idx, err := strconv.Atoi(rest[:closeBracket])
		if err != nil {
			continue
		}
		fieldPart := rest[closeBracket+1:]
		if !strings.HasPrefix(fieldPart, "[") || !strings.HasSuffix(fieldPart, "]") {
			continue
		}
		field := fieldPart[1 : len(fieldPart)-1]
		if len(values) == 0 {
			continue
		}
		val := values[0]

		if chartMap[idx] == nil {
			chartMap[idx] = &model.SizeChartRow{}
		}
		row := chartMap[idx]

		switch field {
		case "label":
			row.Label = val
		case "chest":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Chest = &f
			}
		case "waist":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Waist = &f
			}
		case "hips":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Hips = &f
			}
		case "length":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Length = &f
			}
		case "foot_length":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.FootLength = &f
			}
		case "wrist":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Wrist = &f
			}
		}
	}
	for i := 0; ; i++ {
		if row, ok := chartMap[i]; ok {
			p.SizeChart = append(p.SizeChart, *row)
		} else {
			break
		}
	}

	return p, nil
}
