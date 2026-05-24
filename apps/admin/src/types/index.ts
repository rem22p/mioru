export interface User {
  id: string;
  username: string;
  first_name: string;
  last_name: string;
  email: string;
  display_name: string;
  avatar_color: string;
}

export interface Workspace {
  id: string;
  label: string;
  icon: string;
  active: boolean;
}

export interface Note {
  id: string;
  content: string;
  color: string;
  author: string;
  position_x: number;
  position_y: number;
  created_at: string;
  updated_at: string;
}

// ── Product types ──

export interface Category {
  id: number;
  parent_id: number | null;
  name: string;
  slug: string;
  criteria: string[];
}

export interface SizeChartEntry {
  size: string;
  chest?: string;
  waist?: string;
  hips?: string;
  length?: string;
  sleeve?: string;
  [key: string]: string | undefined;
}

export interface ProductImage {
  id: string;
  url: string;
  alt: string;
  sort_order: number;
}

export interface Product {
  id: string;
  name: string;
  slug: string;
  description: string;
  brand: string;
  price: number;
  compare_at_price?: number;
  in_stock: boolean;
  stock_quantity: number;
  category_id: number;
  category?: Category;
  images: ProductImage[];
  sizes: string[];
  color?: string;
  model?: string;
  fit?: string;
  material?: string;
  size_chart: SizeChartEntry[];
  care_instructions: string[];
  created_at: string;
  updated_at: string;
}

export interface ProductFilter {
  search: string;
  category_id: string;
  brand: string;
  sort: string;
  page: number;
  limit: number;
}

export interface ProductsResponse {
  products: Product[];
  total: number;
}
