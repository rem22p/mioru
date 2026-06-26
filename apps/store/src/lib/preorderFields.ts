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
  // Bottoms — height + waist
  "shorts": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "waist", label: "Талия", unit: "см", min: 50, max: 150, placeholder: "80" },
  ],
  "jeans": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "waist", label: "Талия", unit: "см", min: 50, max: 150, placeholder: "80" },
  ],
  "pants": [
    { key: "height", label: "Рост", unit: "см", min: 100, max: 250, placeholder: "170" },
    { key: "waist", label: "Талия", unit: "см", min: 50, max: 150, placeholder: "80" },
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

export default PREORDER_FIELDS;
