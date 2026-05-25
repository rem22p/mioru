-- Baseline schema. Uses CREATE TABLE IF NOT EXISTS so an existing database
-- (e.g. the production VPS, whose tables predate tern) adopts this migration as
-- a no-op and is recorded at version 1 without manual intervention. Later
-- migrations (002+) run forward from this known baseline and may use plain DDL.
-- Irreversible: no down section — the baseline is never rolled back.

CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	email TEXT UNIQUE NOT NULL,
	hashed_password TEXT NOT NULL,
	first_name TEXT NOT NULL DEFAULT '',
	last_name TEXT NOT NULL DEFAULT '',
	display_name TEXT NOT NULL DEFAULT '',
	avatar_color TEXT NOT NULL DEFAULT '#f85149',
	role TEXT NOT NULL DEFAULT 'admin',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customers (
	id SERIAL PRIMARY KEY,
	email TEXT UNIQUE NOT NULL,
	hashed_password TEXT NOT NULL,
	first_name TEXT NOT NULL DEFAULT '',
	last_name TEXT NOT NULL DEFAULT '',
	phone TEXT NOT NULL DEFAULT '',
	avatar_color TEXT NOT NULL DEFAULT '#44944A',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
	id SERIAL PRIMARY KEY,
	parent_id INTEGER REFERENCES categories(id),
	name TEXT NOT NULL,
	slug TEXT NOT NULL,
	criteria TEXT NOT NULL DEFAULT '[]',
	sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS products (
	id SERIAL PRIMARY KEY,
	slug TEXT UNIQUE NOT NULL,
	category_id INTEGER REFERENCES categories(id),
	brand TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	price INTEGER NOT NULL DEFAULT 0,
	color TEXT NOT NULL DEFAULT '',
	model TEXT NOT NULL DEFAULT '',
	fit TEXT NOT NULL DEFAULT '',
	material TEXT NOT NULL DEFAULT '',
	care TEXT NOT NULL DEFAULT '[]',
	description TEXT NOT NULL DEFAULT '',
	xp_reward INTEGER NOT NULL DEFAULT 0,
	in_stock SMALLINT NOT NULL DEFAULT 1,
	status TEXT NOT NULL DEFAULT 'in_stock',
	stock_quantity INTEGER NOT NULL DEFAULT 0,
	created_by TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS product_sizes (
	id SERIAL PRIMARY KEY,
	product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
	size_label TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS size_chart_rows (
	id SERIAL PRIMARY KEY,
	product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
	label TEXT NOT NULL,
	chest REAL,
	waist REAL,
	hips REAL,
	length REAL,
	foot_length REAL,
	wrist REAL,
	sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS product_images (
	id SERIAL PRIMARY KEY,
	product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
	url TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
	token TEXT PRIMARY KEY,
	username TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL
);
