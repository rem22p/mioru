export const WORKSPACES = [
  { id: 'products', label: 'Товары', icon: 'Package', active: true },
  { id: 'notes', label: 'Доска заметок', icon: 'StickyNote', active: false },
  { id: 'accounting', label: 'Бухгалтерия', icon: 'Banknote', active: false },
  { id: 'analytics', label: 'Аналитика', icon: 'BarChart3', active: false },
  { id: 'chatbot', label: 'Чат-бот', icon: 'Bot', active: false },
  { id: 'tinder', label: 'Тиндер', icon: 'Flame', active: false },
  { id: 'ideas', label: 'Идеи', icon: 'Lightbulb', active: false },
] as const;

export const NOTE_COLORS = [
  '#ffffff', '#ff6b6b', '#feca57', '#48dbfb', '#ff9ff3', '#54a0ff', '#5f27cd', '#01a3a4',
] as const;

export const AVATAR_COLORS = [
  '#f85149', '#58a6ff', '#3fb950', '#f0883e', '#bc8cff', '#79c0ff', '#f778ba', '#7ee787',
] as const;

export const CATEGORIES = [
  { id: 1, parent_id: null, name: 'Одежда', slug: 'clothing', criteria: ['size', 'brand', 'color'] },
  { id: 2, parent_id: 1, name: 'Футболки / поло', slug: 'tshirts-polo', criteria: [] },
  { id: 3, parent_id: 1, name: 'Рубашки', slug: 'shirts', criteria: [] },
  { id: 4, parent_id: 1, name: 'Свитеры / худи', slug: 'sweaters-hoodies', criteria: [] },
  { id: 5, parent_id: 1, name: 'Верхняя одежда', slug: 'outerwear', criteria: [] },
  { id: 6, parent_id: 1, name: 'Брюки / джинсы', slug: 'pants-jeans', criteria: [] },
  { id: 7, parent_id: 1, name: 'Шорты', slug: 'shorts', criteria: [] },
  { id: 8, parent_id: 1, name: 'Спортивная одежда', slug: 'sportswear', criteria: [] },
  { id: 9, parent_id: 1, name: 'Платья', slug: 'dresses', criteria: [] },
  { id: 10, parent_id: 1, name: 'Юбки', slug: 'skirts', criteria: [] },
  { id: 11, parent_id: null, name: 'Обувь', slug: 'shoes', criteria: ['size', 'brand', 'color'] },
  { id: 12, parent_id: 11, name: 'Кроссовки', slug: 'sneakers', criteria: [] },
  { id: 13, parent_id: 11, name: 'Ботинки', slug: 'boots', criteria: [] },
  { id: 14, parent_id: 11, name: 'Туфли', slug: 'formal-shoes', criteria: [] },
  { id: 15, parent_id: 11, name: 'Сандалии', slug: 'sandals', criteria: [] },
  { id: 16, parent_id: null, name: 'Аксессуары', slug: 'accessories', criteria: ['brand', 'color'] },
  { id: 17, parent_id: 16, name: 'Сумки', slug: 'bags', criteria: [] },
  { id: 18, parent_id: 16, name: 'Головные уборы', slug: 'headwear', criteria: [] },
  { id: 19, parent_id: 16, name: 'Часы', slug: 'watches', criteria: [] },
  { id: 20, parent_id: 16, name: 'Очки', slug: 'glasses', criteria: [] },
  { id: 21, parent_id: 16, name: 'Ремни', slug: 'belts', criteria: [] },
  { id: 22, parent_id: 16, name: 'Украшения', slug: 'jewelry', criteria: [] },
  { id: 23, parent_id: null, name: 'Нижнее бельё', slug: 'underwear', criteria: ['size', 'brand', 'color'] },
  { id: 24, parent_id: 23, name: 'Носки', slug: 'socks', criteria: [] },
] as const;

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
