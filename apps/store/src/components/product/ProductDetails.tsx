import { useState } from 'react';
import { motion } from 'framer-motion';
import { Package, Truck, RotateCcw, Shirt, Droplets, Info } from 'lucide-react';
import { Product } from '@/types';
import { useTranslation } from 'react-i18next';

interface ProductDetailsProps {
  product: Product;
}

type TabId = 'description' | 'material' | 'delivery';

export default function ProductDetails({ product }: ProductDetailsProps) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<TabId>('description');

  const tabs: { id: TabId; label: string; icon: React.ReactNode }[] = [
    { id: 'description', label: t('product.tabs.description'), icon: <Info className="h-4 w-4" /> },
    { id: 'material', label: t('product.tabs.material'), icon: <Shirt className="h-4 w-4" /> },
    { id: 'delivery', label: t('product.tabs.delivery'), icon: <Truck className="h-4 w-4" /> },
  ];

  return (
    <div className="mt-12">
      {/* Tabs */}
      <div className="flex border-b border-[var(--color-border-custom)]">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`relative flex items-center gap-2 px-6 py-4 text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'text-white'
                : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
            }`}
          >
            {tab.icon}
            {tab.label}
            {activeTab === tab.id && (
              <motion.div
                layoutId="product-details-tab"
                className="absolute bottom-0 left-0 right-0 h-0.5 bg-[#44944A]"
              />
            )}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="py-8">
        {activeTab === 'description' && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <p className="text-[var(--color-text-secondary)] leading-relaxed text-base">{product.description}</p>

            {product.fit && (
              <div className="mt-6 flex items-center gap-3 p-4 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                <div className="w-10 h-10 rounded-lg bg-[#44944A]/10 flex items-center justify-center shrink-0">
                  <Shirt className="h-5 w-5 text-[#44944A]" />
                </div>
                <div>
                  <p className="text-sm font-medium text-[var(--color-text-primary)]">
                    {t('product.fit.title')}: {
                      product.fit === 'slim' ? t('product.fit.slim') :
                      product.fit === 'regular' ? t('product.fit.regular') :
                      product.fit === 'oversized' ? t('product.fit.oversized') :
                      t('product.fit.loose')
                    }
                  </p>
                  <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">
                    {product.fit === 'slim' && 'Плотно облегает фигуру'}
                    {product.fit === 'regular' && 'Классическая посадка'}
                    {product.fit === 'oversized' && 'Увеличенный объём и ширина'}
                    {product.fit === 'loose' && 'Свободный силуэт без объёма'}
                  </p>
                </div>
              </div>
            )}
          </motion.div>
        )}

        {activeTab === 'material' && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="space-y-6"
          >
            <div>
              <h4 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-3">
                Материал
              </h4>
              <p className="text-[var(--color-text-secondary)] leading-relaxed">{product.material}</p>
            </div>

            <div>
              <h4 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-primary)] mb-3">
                Рекомендации по уходу
              </h4>
              <div className="grid gap-3">
                {product.care.map((item, idx) => (
                  <div
                    key={idx}
                    className="flex items-start gap-3 p-4 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]"
                  >
                    <Droplets className="h-4 w-4 text-[#44944A] shrink-0 mt-0.5" />
                    <p className="text-sm text-[var(--color-text-secondary)]">{item}</p>
                  </div>
                ))}
              </div>
            </div>
          </motion.div>
        )}

        {activeTab === 'delivery' && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="space-y-6"
          >
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="p-5 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                <div className="w-10 h-10 rounded-lg bg-[#44944A]/10 flex items-center justify-center mb-4">
                  <Truck className="h-5 w-5 text-[#44944A]" />
                </div>
                <h4 className="text-sm font-semibold text-[var(--color-text-primary)] mb-2">Доставка</h4>
                <ul className="space-y-2 text-sm text-[var(--color-text-secondary)]">
                  <li>• Личная встреча — Тирасполь, бесплатно</li>
                  <li>• Доставка по адресу — Тирасполь/Бендеры, 25 руб</li>
                  <li>• Маршрутка — ПМР + Кишинёв, до 50 руб</li>
                  <li>• Экспресс-почта — ПМР, до 50 руб</li>
                  <li>• Почта Молдовы — все города, до 50 руб</li>
                </ul>
              </div>

              <div className="p-5 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
                <div className="w-10 h-10 rounded-lg bg-[#44944A]/10 flex items-center justify-center mb-4">
                  <RotateCcw className="h-5 w-5 text-[#44944A]" />
                </div>
                <h4 className="text-sm font-semibold text-[var(--color-text-primary)] mb-2">Возврат</h4>
                <ul className="space-y-2 text-sm text-[var(--color-text-secondary)]">
                  <li>• 24 часа на возврат</li>
                  <li>• Возврат возможен только по причине производственного брака</li>
                  <li>• Товар должен быть с бирками и без следов носки</li>
                </ul>
              </div>
            </div>

          </motion.div>
        )}
      </div>
    </div>
  );
}
