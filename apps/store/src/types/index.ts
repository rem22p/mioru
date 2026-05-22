export interface Category {
  id: string;
  slug: string;
  name: string;
  description: string;
  image: string;
}

export interface SizeChartRow {
  size: string;
  chest?: number;
  waist?: number;
  hips?: number;
  length?: number;
  footLength?: number;
  wrist?: number;
}

export interface SizeChart {
  unit: 'cm' | 'inch';
  columns: { key: keyof SizeChartRow; label: string }[];
  rows: SizeChartRow[];
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

export interface Product {
  id: string;
  slug: string;
  name: string;
  description: string;
  category: Category;
  price: number;
  sizes: string[];
  images: string[];
  inStock: boolean;
  xpReward: number;
  createdAt: string;
  material: string;
  care: string[];
  sizeChart: SizeChart;
  reviews: Review[];
  relatedProductIds: string[];
  modelInfo?: string;
  fit?: 'slim' | 'regular' | 'oversized' | 'loose';
}

export interface CartItem {
  product: Product;
  size: string;
  quantity: number;
}

export interface AvatarParams {
  gender: 'male' | 'female';
  height: number;
  weight: number;
  fatPercentage: number;
  musclePercentage: number;
}

export interface User {
  id: string;
  name: string;
  email: string;
  avatarParams: AvatarParams;
  xpBalance: number;
  vipLevel: number;
}

export interface Order {
  id: string;
  userId: string;
  items: CartItem[];
  total: number;
  status: 'pending' | 'processing' | 'shipped' | 'delivered' | 'cancelled';
  createdAt: string;
  deliveryInfo: DeliveryInfo;
}

export interface DeliveryInfo {
  name: string;
  phone: string;
  email: string;
  city: string;
  street: string;
  house: string;
  apartment?: string;
  paymentMethod: 'card' | 'cod' | 'sbp';
}
