# MIORU — Виртуальная примерка одежды

3D-аватар, онлайн-примерка и e-commerce для одежды и аксессуаров.

## 🏗 Структура

```
mioru/
├── apps/
│   └── store/          — Интернет-магазин (React 19 + Vite + shadcn/ui)
├── backend/
│   └── api/            — Go API (REST + WebSocket + JWT)
├── packages/
│   └── shared/         — Общие типы и утилиты
└── docs/               — Архитектура и документация
```

## 🔧 Технологии

- **Frontend:** React 19, Vite, Tailwind CSS v4, shadcn/ui, Three.js, Framer Motion
- **State:** Zustand
- **3D:** React Three Fiber, GLTF/GLB, DRACO
- **Backend:** Go, Redis, JWT, WebSocket
- **Tests:** Vitest, Playwright, Testing Library
- **i18n:** 3 языка (Русский, English, Română)
- **Email:** Resend API

## 🚀 Быстрый старт

```bash
git clone https://github.com/<user>/mioru.git
cd mioru

# Магазин
cd apps/store && npm install && npm run dev

# Бэкенд (требуется Redis)
cd backend/api && go run ./cmd/server
```

## 📦 Планы

- [x] Процедурный аватар + GLB-загрузчик
- [x] SEO (Helmet + OpenGraph)
- [x] Тёмная/светлая тема
- [x] i18n (RU/EN/RO)
- [ ] Supabase/PostgreSQL интеграция
- [ ] Система заказов и оплаты
- [ ] Примерка одежды на аватаре
- [ ] PWA-приложение
- [ ] AI-рекомендации размеров

## 📄 Лицензия

MIT
