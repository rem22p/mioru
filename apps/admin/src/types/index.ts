export interface User {
	id: string;
	username: string;
	first_name: string;
	last_name: string;
	email: string;
	display_name: string;
	avatar_color: string;
	role: string;
}

export interface Workspace {
	id: string;
	label: string;
	icon: string;
	active: boolean;
}

// ── Product types ──

export interface Category {
	id: number;
	parent_id: number | null;
	name: string;
	slug: string;
	criteria: string[];
	children?: Category[];
}

export interface SizeChartEntry {
	label: string;
	chest?: string;
	waist?: string;
	hips?: string;
	length?: string;
	foot_length?: string;
	wrist?: string;
	[key: string]: string | undefined;
}

export interface ProductImage {
	id: string;
	url: string;
	file?: File;
	sort_order: number;
}

export interface Product {
	id: number;
	name: string;
	slug: string;
	description: string;
	brand: string;
	price: number;
	xp_reward: number;
	in_stock: boolean;
	status: string;
	stock_quantity: number;
	category_id: number;
	category_name?: string;
	images: ProductImage[];
	sizes: string[];
	color: string;
	model: string;
	fit: string;
	material: string;
	size_chart: SizeChartEntry[];
	care: string[];
	created_by: string;
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
