export const WORKSPACES = [
  { id: 'products', label: 'Товары', icon: 'Package', active: true },
  { id: 'orders', label: 'Заказы', icon: 'ShoppingBag', active: true },
  { id: 'users', label: 'Пользователи', icon: 'UsersRound', active: true },
  { id: 'customers', label: 'Клиенты', icon: 'UserCircle', active: true },
  { id: 'telegram', label: 'Telegram', icon: 'Send', active: true },
  { id: 'accounting', label: 'Бухгалтерия', icon: 'Banknote', active: false },
  { id: 'analytics', label: 'Аналитика', icon: 'BarChart3', active: false },
  { id: 'chatbot', label: 'Чат-бот', icon: 'Bot', active: false },
  { id: 'tinder', label: 'Тиндер', icon: 'Flame', active: false },
  { id: 'ideas', label: 'Идеи', icon: 'Lightbulb', active: false },
] as const;

export const AVATAR_COLORS = [
  '#f85149', '#58a6ff', '#3fb950', '#f0883e', '#bc8cff', '#79c0ff', '#f778ba', '#7ee787',
] as const;

// The category tree is owned by the backend (seeded in postgres.go migrate()).
// The admin fetches it via /api/admin/categories into productStore.categories —
// there is intentionally no hardcoded copy here to keep a single source.

export const SIZE_OPTIONS_CLOTHING = ['XXS', 'XS', 'S', 'M', 'L', 'XL', 'XXL', 'XXXL'] as const;
export const SIZE_OPTIONS_SHOES = ['36', '37', '38', '39', '40', '41', '42', '43', '44', '45', '46'] as const;
export const SIZE_OPTIONS_ACCESSORIES = ['One size'] as const;

export const SORT_OPTIONS = [
  { value: 'newest', label: 'Сначала новые' },
  { value: 'oldest', label: 'Сначала старые' },
  { value: 'price_asc', label: 'Цена: по возрастанию' },
  { value: 'price_desc', label: 'Цена: по убыванию' },
  { value: 'name_asc', label: 'Название: А-Я' },
] as const;
