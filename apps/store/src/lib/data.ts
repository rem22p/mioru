import type { Category, Product, Order, User } from '@/types';

export const categories: Category[] = [
  { id: '1', slug: 'sneakers', name: 'Кроссовки', description: 'Уличная классика и современные силуэты', image: '/images/categories/sneakers.jpg' },
  { id: '2', slug: 'slides', name: 'Тапки', description: 'Комфорт для дома и улицы', image: '/images/categories/slides.jpg' },
  { id: '3', slug: 'tshirts', name: 'Футболки', description: 'Базовые и оверсайз фиты', image: '/images/categories/tshirts.jpg' },
  { id: '4', slug: 'shorts', name: 'Шорты', description: 'Технологичные материалы', image: '/images/categories/shorts.jpg' },
  { id: '5', slug: 'bracelets', name: 'Браслеты', description: 'Минималистичные аксессуары', image: '/images/categories/bracelets.jpg' },
];

const sneakerSizeChart = {
  unit: 'cm' as const,
  columns: [{ key: 'size' as const, label: 'EU' }, { key: 'footLength' as const, label: 'Длина стопы, см' }],
  rows: [
    { size: '40', footLength: 25.5 }, { size: '41', footLength: 26.2 }, { size: '42', footLength: 26.9 },
    { size: '43', footLength: 27.6 }, { size: '44', footLength: 28.3 }, { size: '45', footLength: 29.0 },
  ],
};

const apparelSizeChart = {
  unit: 'cm' as const,
  columns: [
    { key: 'size' as const, label: 'Размер' }, { key: 'chest' as const, label: 'Обхват груди, см' },
    { key: 'waist' as const, label: 'Обхват талии, см' }, { key: 'length' as const, label: 'Длина изделия, см' },
  ],
  rows: [
    { size: 'XS', chest: 92, waist: 76, length: 66 }, { size: 'S', chest: 96, waist: 80, length: 68 },
    { size: 'M', chest: 100, waist: 84, length: 70 }, { size: 'L', chest: 106, waist: 90, length: 72 },
    { size: 'XL', chest: 112, waist: 96, length: 74 }, { size: 'XXL', chest: 118, waist: 102, length: 76 },
  ],
};

const braceletSizeChart = {
  unit: 'cm' as const,
  columns: [{ key: 'size' as const, label: 'Размер' }, { key: 'wrist' as const, label: 'Обхват запястья, см' }],
  rows: [{ size: 'one size', wrist: 16 }],
};

const generateReviews = (productName: string): Product['reviews'] => [
  { id: 'r1', author: 'Алексей К.', rating: 5, date: '2024-12-10', text: `Отличное качество ${productName}. Материал приятный на ощупь, посадка идеальная. Рекомендую!`, size: 'M', helpful: 12 },
  { id: 'r2', author: 'Мария С.', rating: 4, date: '2024-11-28', text: 'Хороший товар за свои деньги. Доставка быстрая. Единственное — размер чуть больше, чем ожидала.', size: 'S', helpful: 8 },
  { id: 'r3', author: 'Дмитрий В.', rating: 5, date: '2024-11-15', text: 'Беру уже второй раз. Качество на высоте, ничего не растянулось после стирки.', size: 'L', helpful: 5 },
];

export const products: Product[] = [
  {
    id: '1', slug: 'midnight-runner', name: 'Midnight Runner',
    description: 'Монохромные кроссовки на вулканизированной подошве. Верх из комбинированных материалов с перфорацией для вентиляции. Усиленный мыс и пятка обеспечивают долговечность.',
    category: categories[0], price: 8900, sizes: ['40', '41', '42', '43', '44', '45'],
    images: ['/images/products/midnight-runner-1.jpg', '/images/products/midnight-runner-2.jpg', '/images/products/midnight-runner-3.jpg'],
    inStock: true, xpReward: 150, createdAt: '2024-12-01',
    material: 'Верх: текстиль, синтетическая кожа. Подкладка: текстиль. Подошва: вулканизированная резина.',
    care: ['Чистить мягкой щёткой', 'Не стирать в машине', 'Сушить при комнатной температуре', 'Использовать водоотталкивающий спрей'],
    sizeChart: sneakerSizeChart, reviews: generateReviews('Midnight Runner'), relatedProductIds: ['6', '2', '7'],
    modelInfo: 'Модель роста 185 см, размер 42', fit: 'regular',
  },
  {
    id: '2', slug: 'cloud-slide', name: 'Cloud Slide',
    description: 'Лёгкие слайды из вспененного материала. Анатомическая стелька повторяет форму стопы. Фактурная подошва с противоскользящим рисунком.',
    category: categories[1], price: 3500, sizes: ['40', '41', '42', '43', '44', '45'],
    images: ['/images/products/cloud-slide-1.jpg', '/images/products/cloud-slide-2.jpg'],
    inStock: true, xpReward: 60, createdAt: '2024-12-05',
    material: 'Верх: EVA-пена. Стелька: вспененный полиуретан с текстильным покрытием. Подошва: резина.',
    care: ['Протирать влажной тканью', 'Не подвергать воздействию высоких температур', 'Хранить в сухом месте'],
    sizeChart: sneakerSizeChart, reviews: generateReviews('Cloud Slide'), relatedProductIds: ['7', '1', '6'],
    modelInfo: 'Модель роста 178 см, размер 43', fit: 'regular',
  },
  {
    id: '3', slug: 'essential-tee-black', name: 'Essential Tee Black',
    description: 'Базовая футболка из плотного хлопка 240 г/м². Прямой крой, усиленная горловина с двойной строчкой. Не деформируется после стирки.',
    category: categories[2], price: 2900, sizes: ['XS', 'S', 'M', 'L', 'XL', 'XXL'],
    images: ['/images/products/essential-tee-black-1.jpg', '/images/products/essential-tee-black-2.jpg', '/images/products/essential-tee-black-3.jpg'],
    inStock: true, xpReward: 50, createdAt: '2024-11-20',
    material: '100% органический хлопок, плотность 240 г/м².',
    care: ['Стирка при 30°C', 'Гладить при низкой температуре', 'Не отбеливать', 'Сушить в расправленном виде'],
    sizeChart: apparelSizeChart, reviews: generateReviews('Essential Tee Black'), relatedProductIds: ['4', '8', '5'],
    modelInfo: 'Модель роста 182 см, размер M', fit: 'regular',
  },
  {
    id: '4', slug: 'essential-tee-white', name: 'Essential Tee White',
    description: 'Базовая футболка из плотного хлопка 240 г/м² в классическом белом цвете. Прямой крой, усиленная горловина с двойной строчкой.',
    category: categories[2], price: 2900, sizes: ['XS', 'S', 'M', 'L', 'XL', 'XXL'],
    images: ['/images/products/essential-tee-white-1.jpg', '/images/products/essential-tee-white-2.jpg'],
    inStock: true, xpReward: 50, createdAt: '2024-11-20',
    material: '100% органический хлопок, плотность 240 г/м².',
    care: ['Стирка при 30°C', 'Гладить при низкой температуре', 'Не отбеливать', 'Сушить в расправленном виде'],
    sizeChart: apparelSizeChart, reviews: generateReviews('Essential Tee White'), relatedProductIds: ['3', '8', '5'],
    modelInfo: 'Модель роста 175 см, размер S', fit: 'regular',
  },
  {
    id: '5', slug: 'tech-shorts', name: 'Tech Shorts',
    description: 'Шорты из нейлоновой ткани с DWR пропиткой. Эластичный пояс со шнурком, боковые карманы на молнии. Лёгкие и быстросохнущие.',
    category: categories[3], price: 4200, sizes: ['XS', 'S', 'M', 'L', 'XL', 'XXL'],
    images: ['/images/products/tech-shorts-1.jpg', '/images/products/tech-shorts-2.jpg'],
    inStock: true, xpReward: 70, createdAt: '2024-12-10',
    material: 'Основа: 100% нейлон с DWR пропиткой. Подкладка: сетчатый полиэстер.',
    care: ['Стирка при 30°C', 'Не использовать кондиционер', 'Не сушить в машине', 'Гладить без пара'],
    sizeChart: apparelSizeChart, reviews: generateReviews('Tech Shorts'), relatedProductIds: ['9', '3', '4'],
    modelInfo: 'Модель роста 188 см, размер L', fit: 'regular',
  },
  {
    id: '6', slug: 'street-runner-red', name: 'Street Runner Red',
    description: 'Кроссовки в цвете "красный оксид". Многослойный верх из замши и текстиля, реактивная амортизация. Отражающие элементы на пятке.',
    category: categories[0], price: 9500, sizes: ['40', '41', '42', '43', '44', '45'],
    images: ['/images/products/street-runner-red-1.jpg', '/images/products/street-runner-red-2.jpg', '/images/products/street-runner-red-3.jpg'],
    inStock: true, xpReward: 160, createdAt: '2024-12-15',
    material: 'Верх: замша, текстиль. Подкладка: текстиль. Подошва: пенный полимер + резина.',
    care: ['Чистить специальным средством для замши', 'Защитный спрей перед первой ноской', 'Сушить с бумагой внутри'],
    sizeChart: sneakerSizeChart, reviews: generateReviews('Street Runner Red'), relatedProductIds: ['1', '2', '7'],
    modelInfo: 'Модель роста 180 см, размер 43', fit: 'regular',
  },
  {
    id: '7', slug: 'bone-slide', name: 'Bone Slide',
    description: 'Минималистичные слайды в цвете кости. Мягкая внутренняя поверхность с эффектом памяти. Универсальный дизайн для дома и улицы.',
    category: categories[1], price: 3200, sizes: ['40', '41', '42', '43', '44', '45'],
    images: ['/images/products/bone-slide-1.jpg', '/images/products/bone-slide-2.jpg'],
    inStock: true, xpReward: 55, createdAt: '2024-12-08',
    material: 'Верх: EVA с мягким покрытием. Стелька: пена с эффектом памяти. Подошва: износостойкая резина.',
    care: ['Протирать влажной тканью', 'Не подвергать воздействию высоких температур', 'Хранить в сухом месте'],
    sizeChart: sneakerSizeChart, reviews: generateReviews('Bone Slide'), relatedProductIds: ['2', '1', '6'],
    modelInfo: 'Модель роста 172 см, размер 41', fit: 'regular',
  },
  {
    id: '8', slug: 'oversized-boxy-tee', name: 'Oversized Boxy Tee',
    description: 'Свободный крой с укороченным рукавом. Тяжёлый хлопок 300 г/м². Объёмный силуэт, отлично сидит с высокой посадкой.',
    category: categories[2], price: 3400, sizes: ['XS', 'S', 'M', 'L', 'XL', 'XXL'],
    images: ['/images/products/oversized-boxy-tee-1.jpg', '/images/products/oversized-boxy-tee-2.jpg'],
    inStock: true, xpReward: 60, createdAt: '2024-11-25',
    material: '100% хлопок, плотность 300 г/м². Предварительная усадка.',
    care: ['Стирка при 30°C', 'Гладить при низкой температуре', 'Не отбеливать', 'Сушить в расправленном виде'],
    sizeChart: apparelSizeChart, reviews: generateReviews('Oversized Boxy Tee'), relatedProductIds: ['3', '4', '5'],
    modelInfo: 'Модель роста 190 см, размер XL', fit: 'oversized',
  },
  {
    id: '9', slug: 'cargo-shorts', name: 'Cargo Shorts',
    description: 'Карго шорты с объёмными карманами. Хлопковая саржа с гарментной варкой. Регулируемый пояс, усиленные швы.',
    category: categories[3], price: 5100, sizes: ['XS', 'S', 'M', 'L', 'XL', 'XXL'],
    images: ['/images/products/cargo-shorts-1.jpg', '/images/products/cargo-shorts-2.jpg', '/images/products/cargo-shorts-3.jpg'],
    inStock: true, xpReward: 85, createdAt: '2024-12-12',
    material: '100% хлопковая саржа, гарментная варка. Фурнитура: металл.',
    care: ['Стирка при 30°C', 'Гладить при средней температуре', 'Не отбеливать', 'Сушить в расправленном виде'],
    sizeChart: apparelSizeChart, reviews: generateReviews('Cargo Shorts'), relatedProductIds: ['5', '3', '4'],
    modelInfo: 'Модель роста 185 см, размер M', fit: 'loose',
  },
  {
    id: '10', slug: 'steel-bracelet', name: 'Steel Bracelet',
    description: 'Минималистичный браслет из нержавеющей стали. Матовая поверхность, регулируемый размер. Гипоаллергенный материал.',
    category: categories[4], price: 1800, sizes: ['one size'],
    images: ['/images/products/steel-bracelet-1.jpg', '/images/products/steel-bracelet-2.jpg'],
    inStock: true, xpReward: 30, createdAt: '2024-12-01',
    material: 'Нержавеющая сталь 316L, матовое покрытие.',
    care: ['Протирать мягкой тканью', 'Избегать контакта с химическими веществами', 'Хранить в фирменной коробке'],
    sizeChart: braceletSizeChart, reviews: generateReviews('Steel Bracelet'), relatedProductIds: [],
    modelInfo: 'Универсальный размер, регулируемая застёжка', fit: 'regular',
  },
];

export const mockUser: User = {
  id: '1', name: 'Иван', email: 'ivan@mioru.store',
  avatarParams: { gender: 'male', height: 180, weight: 75, fatPercentage: 15, musclePercentage: 40 },
  xpBalance: 1240, vipLevel: 2,
};

export const mockOrders: Order[] = [
  {
    id: 'ORD-001', userId: '1',
    items: [
      { product: products[2], size: 'M', quantity: 1 },
      { product: products[4], size: 'L', quantity: 1 },
    ],
    total: 7100, status: 'delivered', createdAt: '2024-12-01',
    deliveryInfo: { name: 'Иван', phone: '+7 (999) 123-45-67', email: 'ivan@mioru.store', city: 'Москва', street: 'Тверская', house: '12', apartment: '45', paymentMethod: 'card' },
  },
  {
    id: 'ORD-002', userId: '1',
    items: [{ product: products[0], size: '42', quantity: 1 }],
    total: 8900, status: 'shipped', createdAt: '2024-12-15',
    deliveryInfo: { name: 'Иван', phone: '+7 (999) 123-45-67', email: 'ivan@mioru.store', city: 'Москва', street: 'Арбат', house: '5', paymentMethod: 'sbp' },
  },
];
