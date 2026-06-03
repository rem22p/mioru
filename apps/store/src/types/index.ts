// ── API response types (match backend JSON shapes) ──

export interface Category {
  id: number;
  parent_id: number | null;
  name: string;
  slug: string;
  criteria: string[];
  sort_order: number;
  cover_image?: string | null;
  children?: Category[];
}

export interface SizeChartRow {
  label: string;
  chest?: number;
  waist?: number;
  hips?: number;
  length?: number;
  foot_length?: number;
  wrist?: number;
  sort_order: number;
}

export interface ProductImage {
  id: number;
  url: string;
  sort_order: number;
}

export interface Product {
  id: number;
  slug: string;
  category_id: number;
  category_name: string;
  brand: string;
  name: string;
  price: number;
  color: string;
  model: string;
  fit: string;
  material: string;
  care: string[];
  description: string;
  xp_reward: number;
  in_stock: boolean;
  status: string;
  stock_quantity: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  sizes: string[];
  size_chart: SizeChartRow[];
  images: ProductImage[];
}

/** Size chart transformed for the SizeChartModal table */
export interface SizeChart {
  unit: "cm";
  columns: { key: string; label: string }[];
  rows: Record<string, string | number | undefined>[];
}

export interface Review {
  id: string;
  author: string;
  avatar?: string;
  rating: number;
  date: string;
  text: string;
  size: string;
  helpful: number;
}

export interface CartItem {
  product: Product;
  size: string;
  quantity: number;
}

export interface AvatarParams {
  gender: "male" | "female";
  height: number;
  weight: number;
  fatPercentage: number;
  musclePercentage: number;
}

export interface User {
  id: string;
  name: string;
  email: string | null;
  firstName: string;
  lastName: string;
  phone: string;
  avatarParams: AvatarParams;
  xpBalance: number;
  vipLevel: number;
}

export interface DeliveryInfo {
  name: string;
  phone: string;
  email: string;
  city: string;
  street: string;
  house: string;
  apartment?: string;
  paymentMethod: "card" | "cod" | "sbp";
}

export interface Order {
  id: string;
  userId: string;
  items: CartItem[];
  total: number;
  status: "pending" | "processing" | "shipped" | "delivered" | "cancelled";
  createdAt: string;
  deliveryInfo: DeliveryInfo;
}

// ── Helpers to transform API data into UI shapes ──

const SIZE_CHART_COLUMN_LABELS: Record<string, string> = {
  chest: "Обхват груди, см",
  waist: "Обхват талии, см",
  hips: "Обхват бёдер, см",
  length: "Длина изделия, см",
  foot_length: "Длина стопы, см",
  wrist: "Обхват запястья, см",
};

const MEASUREMENT_KEYS = [
  "chest",
  "waist",
  "hips",
  "length",
  "foot_length",
  "wrist",
] as const;

/**
 * Transforms backend SizeChartRow[] into the UI SizeChart format.
 */
export function toSizeChart(rows: SizeChartRow[]): SizeChart | null {
  if (!rows || rows.length === 0) return null;

  // Detect which measurement columns have at least one non-null value
  const activeKeys = MEASUREMENT_KEYS.filter((key) =>
    rows.some((r) => r[key] != null),
  );

  const columns: { key: string; label: string }[] = [
    { key: "size", label: "Размер" },
    ...activeKeys.map((key) => ({
      key,
      label: SIZE_CHART_COLUMN_LABELS[key] || key,
    })),
  ];

  const uiRows = rows.map((r) => {
    const row: Record<string, string | number | undefined> = {
      size: r.label,
    };
    for (const key of activeKeys) {
      row[key] = r[key];
    }
    return row;
  });

  return { unit: "cm", columns, rows: uiRows };
}
