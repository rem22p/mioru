export interface PreorderField {
  key: string;
  label: string;
  unit: string;
  min: number;
  max: number;
  placeholder: string;
}

// Category slug → measurement fields for preorder products.
// Products with status !== "in_stock" show these fields
// instead of the standard size selector.
const PREORDER_FIELDS: Record<string, PreorderField[]> = {
  // Apparel — height + weight
  "tshirts-polo": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  "hoodies-zip": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  "sweatshirts-sweaters": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  "jackets": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  "vests": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  "underwear": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  // Bottoms — height + weight (same as apparel)
  "shorts": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  "jeans": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  "pants": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "weight", label: "Вес", unit: "кг", min: 30, max: 250, placeholder: "70" },
  ],
  // Footwear — foot length
  "sneakers": [
    { key: "foot_length", label: "Длина стопы", unit: "см", min: 18, max: 35, placeholder: "27" },
  ],
  "slides": [
    { key: "foot_length", label: "Длина стопы", unit: "см", min: 18, max: 35, placeholder: "27" },
  ],
  "boots": [
    { key: "foot_length", label: "Длина стопы", unit: "см", min: 18, max: 35, placeholder: "27" },
  ],
  // Headwear — head circumference
  "headwear": [
    { key: "head_circumference", label: "Обхват головы", unit: "см", min: 50, max: 65, placeholder: "57" },
  ],
};

const CATEGORY_NAME_TO_SLUG: Record<string, string> = {
  "Футболки / поло": "tshirts-polo",
  "Шорты": "shorts",
  "Худи / зип-худи": "hoodies-zip",
  "Свитшоты / свитера": "sweatshirts-sweaters",
  "Джинсы": "jeans",
  "Штаны": "pants",
  "Куртки": "jackets",
  "Жилетки": "vests",
  "Нижнее бельё": "underwear",
  "Кроссовки": "sneakers",
  "Тапки": "slides",
  "Ботинки": "boots",
  "Головные уборы": "headwear",
};

export function getPreorderFields(categoryName: string): PreorderField[] {
  const slug = CATEGORY_NAME_TO_SLUG[categoryName] || "";
  return PREORDER_FIELDS[slug] || [];
}

// All known measurement keys → human label (union across all categories)
const ALL_LABELS: Record<string, string> = {};
for (const fields of Object.values(PREORDER_FIELDS)) {
  for (const f of fields) {
    ALL_LABELS[f.key] = f.label;
  }
}

export function getMeasurementLabel(key: string): string {
  return ALL_LABELS[key] || key;
}
