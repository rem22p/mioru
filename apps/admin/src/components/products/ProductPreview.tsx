import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  X,
  Star,
  Ruler,
  Check,
  ShoppingBag,
  Heart,
  Share2,
  ArrowLeft,
  Info,
  Shirt,
  Truck,
  Droplets,
  Package,
  RotateCcw,
  Sun,
  Moon,
} from "lucide-react";

interface ProductPreviewProps {
  data: {
    name: string;
    price: number;
    description: string;
    brand: string;
    material: string;
    care: string[];
    sizes: string[];
    color: string;
    fit: string;
    categoryName: string;
    images: { url: string; file?: File }[];
    sizeChart: {
      label: string;
      chest?: string;
      waist?: string;
      hips?: string;
      length?: string;
      foot_length?: string;
      wrist?: string;
    }[];
    inStock: boolean;
  };
  onClose: () => void;
}

export default function ProductPreview({ data, onClose }: ProductPreviewProps) {
  const [previewTheme, setPreviewTheme] = useState<"dark" | "light">("dark");
  const [selectedSize, setSelectedSize] = useState("");
  const [activeTab, setActiveTab] = useState<
    "description" | "material" | "delivery"
  >("description");
  const [mainImage, setMainImage] = useState(0);
  const [showSizeChart, setShowSizeChart] = useState(false);

  const isLight = previewTheme === "light";
  const displayPrice = data.price || 0;

  const bg = isLight ? "bg-[#fafafa]" : "bg-[#0a0a0a]";
  const cardBg = isLight ? "bg-white" : "bg-[#161616]";
  const border = isLight ? "border-[#e0e0e0]" : "border-[#222222]";
  const textPrimary = isLight ? "text-[#0a0a0a]" : "text-white";
  const textSecondary = isLight ? "text-[#666666]" : "text-[#888888]";
  const textMuted = isLight ? "text-[#999999]" : "text-[#555555]";
  const accentBg = "bg-[#44944A]";
  const accentBg10 = isLight ? "bg-[#44944A]/10" : "bg-[#44944A]/10";
  const accentText = "text-[#44944A]";
  const accentBorder = "border-[#44944A]";
  const hoverBg = isLight ? "hover:bg-gray-100" : "hover:bg-[#222222]";

  const images = data.images.filter((img) => img.url || img.file);
  const displayImages =
    images.length > 0
      ? images.map((img) =>
          img.file ? URL.createObjectURL(img.file) : img.url,
        )
      : [
          "data:image/svg+xml," +
            encodeURIComponent(
              '<svg xmlns="http://www.w3.org/2000/svg" width="600" height="600" fill="none"><rect width="600" height="600" fill="%23161616"/><text x="300" y="310" text-anchor="middle" fill="%23555" font-family="monospace" font-size="24">No image</text></svg>',
            ),
        ];

  const fitLabels: Record<string, string> = {
    slim: "Облегающий",
    regular: "Стандартный",
    oversized: "Оверсайз",
    loose: "Свободный",
  };

  const fitDescs: Record<string, string> = {
    slim: "Плотно облегает фигуру",
    regular: "Классическая посадка",
    oversized: "Увеличенный объём и ширина",
    loose: "Свободный силуэт без объёма",
  };

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto"
      onClick={onClose}
    >
      {/* Theme toggle */}
      <div
        className="fixed top-4 right-4 z-[60] flex gap-2"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={() => setPreviewTheme("dark")}
          className={`h-10 w-10 rounded-xl flex items-center justify-center transition-all ${
            !isLight ? accentBg + " text-black" : "bg-[#222] text-[#888]"
          }`}
        >
          <Moon className="h-4 w-4" />
        </button>
        <button
          onClick={() => setPreviewTheme("light")}
          className={`h-10 w-10 rounded-xl flex items-center justify-center transition-all ${
            isLight ? "bg-[#0a0a0a] text-white" : "bg-[#333] text-[#888]"
          }`}
        >
          <Sun className="h-4 w-4" />
        </button>
        <button
          onClick={onClose}
          className="h-10 w-10 rounded-xl bg-[#222] text-[#888] flex items-center justify-center hover:text-white transition-colors"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Preview content */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: 20 }}
        onClick={(e) => e.stopPropagation()}
        className={`w-full min-h-screen ${bg}`}
      >
        <div className="px-6 py-24 lg:px-8 max-w-7xl mx-auto">
          {/* Breadcrumb */}
          <div
            className={`inline-flex items-center gap-2 text-sm ${textSecondary} mb-8`}
          >
            <ArrowLeft className="h-4 w-4" />
            Назад в каталог
          </div>

          <div className="grid gap-12 lg:grid-cols-2">
            {/* Left: Gallery */}
            <div>
              <div
                className={`aspect-square rounded-2xl ${cardBg} border ${border} overflow-hidden mb-4`}
              >
                <img
                  src={displayImages[mainImage]}
                  alt={data.name}
                  className="w-full h-full object-cover"
                />
              </div>
              {displayImages.length > 1 && (
                <div className="flex gap-2 overflow-x-auto pb-2">
                  {displayImages.map((url, i) => (
                    <button
                      key={i}
                      onClick={() => setMainImage(i)}
                      className={`flex-shrink-0 w-20 h-20 rounded-xl overflow-hidden border-2 transition-all ${
                        i === mainImage
                          ? accentBorder
                          : "border-transparent opacity-60 hover:opacity-100"
                      }`}
                    >
                      <img
                        src={url}
                        alt=""
                        className="w-full h-full object-cover"
                      />
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Right: Info */}
            <div className="flex flex-col">
              {/* Category */}
              <p
                className={`text-xs font-mono uppercase tracking-[0.3em] ${accentText}`}
              >
                {data.categoryName || "Категория"}
              </p>
              <h1
                className={`mt-3 text-4xl font-bold tracking-tighter ${textPrimary} sm:text-5xl`}
              >
                {data.name || "Название товара"}
              </h1>

              {/* Brand + Price */}
              <div className="mt-6 flex items-center justify-between">
                {data.brand && (
                  <span
                    className={`text-sm ${textMuted} uppercase tracking-wider`}
                  >
                    {data.brand}
                  </span>
                )}
                <p className={`text-3xl font-bold ${accentText}`}>
                  {displayPrice.toLocaleString("ru-RU")} ₽
                </p>
              </div>

              {/* Color */}
              {data.color && (
                <p className={`mt-2 text-sm ${textSecondary}`}>
                  Цвет: <span className={textPrimary}>{data.color}</span>
                </p>
              )}

              {/* Description */}
              {data.description && (
                <p className={`mt-6 ${textSecondary} leading-relaxed text-sm`}>
                  {data.description}
                </p>
              )}

              {/* Sizes */}
              {data.sizes.length > 0 && (
                <div className="mt-8">
                  <div className="flex items-center justify-between mb-4">
                    <h3
                      className={`text-sm font-semibold uppercase tracking-wider ${textPrimary}`}
                    >
                      Размер
                    </h3>
                    <button
                      onClick={() => setShowSizeChart(true)}
                      className={`inline-flex items-center gap-1.5 text-xs ${textSecondary} hover:${accentText} transition-colors`}
                    >
                      <Ruler className="h-3.5 w-3.5" />
                      Таблица размеров
                    </button>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    {data.sizes.map((size) => (
                      <button
                        key={size}
                        onClick={() => setSelectedSize(size)}
                        className={`relative rounded-xl border px-5 py-3 text-sm font-medium transition-all ${
                          selectedSize === size
                            ? `${accentBorder} ${accentBg} text-black`
                            : `${border} ${textSecondary} hover:border-gray-500 hover:${textPrimary}`
                        }`}
                      >
                        {size}
                        {selectedSize === size && (
                          <span className="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-black flex items-center justify-center">
                            <Check className="h-3 w-3 text-[#44944A]" />
                          </span>
                        )}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Add to cart */}
              <div className="mt-8 flex gap-3">
                <button
                  disabled={!selectedSize && data.sizes.length > 0}
                  className={`flex flex-1 items-center justify-center gap-2 rounded-xl px-6 py-4 text-sm font-semibold transition-all ${
                    accentBg +
                    " text-black hover:shadow-[0_0_30px_rgba(68,148,74,0.3)]"
                  } disabled:opacity-50 disabled:cursor-not-allowed`}
                >
                  <ShoppingBag className="h-4 w-4" />
                  {data.sizes.length > 0
                    ? selectedSize
                      ? "В корзину"
                      : "Выберите размер"
                    : "В корзину"}
                </button>
                <button
                  className={`flex items-center justify-center gap-2 rounded-xl border ${border} ${textSecondary} px-5 py-4`}
                >
                  <Heart className="h-4 w-4" />
                </button>
                <button
                  className={`flex items-center justify-center gap-2 rounded-xl border ${border} ${textSecondary} px-5 py-4`}
                >
                  <Share2 className="h-4 w-4" />
                </button>
              </div>

              {/* Trust badges */}
              <div className="mt-8 grid grid-cols-3 gap-3">
                {[
                  {
                    icon: <Truck className="h-5 w-5" />,
                    label: "Быстрая доставка",
                  },
                  {
                    icon: <RotateCcw className="h-5 w-5" />,
                    label: "14 дней на возврат",
                  },
                  {
                    icon: <Package className="h-5 w-5" />,
                    label: "Безопасная оплата",
                  },
                ].map((badge, i) => (
                  <div
                    key={i}
                    className={`flex flex-col items-center gap-2 p-4 rounded-xl ${cardBg} border ${border}`}
                  >
                    <div className={accentText}>{badge.icon}</div>
                    <p className={`text-xs ${textSecondary} text-center`}>
                      {badge.label}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Tabs */}
          <div className="mt-12">
            <div className={`flex border-b ${border}`}>
              {[
                {
                  id: "description" as const,
                  label: "Описание",
                  icon: <Info className="h-4 w-4" />,
                },
                {
                  id: "material" as const,
                  label: "Состав и уход",
                  icon: <Shirt className="h-4 w-4" />,
                },
                {
                  id: "delivery" as const,
                  label: "Доставка и возврат",
                  icon: <Truck className="h-4 w-4" />,
                },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`relative flex items-center gap-2 px-6 py-4 text-sm font-medium transition-colors ${
                    activeTab === tab.id
                      ? textPrimary
                      : `${textSecondary} hover:${textPrimary}`
                  }`}
                >
                  {tab.icon}
                  {tab.label}
                  {activeTab === tab.id && (
                    <div
                      className={`absolute bottom-0 left-0 right-0 h-0.5 ${accentBg}`}
                    />
                  )}
                </button>
              ))}
            </div>

            <div className="py-8">
              {activeTab === "description" && (
                <div>
                  <p className={`${textSecondary} leading-relaxed text-base`}>
                    {data.description || "Описание товара..."}
                  </p>
                  {data.fit && (
                    <div
                      className={`mt-6 flex items-center gap-3 p-4 rounded-xl ${cardBg} border ${border}`}
                    >
                      <div
                        className={`w-10 h-10 rounded-lg ${accentBg10} flex items-center justify-center shrink-0`}
                      >
                        <Shirt className={`h-5 w-5 ${accentText}`} />
                      </div>
                      <div>
                        <p className={`text-sm font-medium ${textPrimary}`}>
                          Крой: {fitLabels[data.fit] || data.fit}
                        </p>
                        <p className={`text-xs ${textSecondary} mt-0.5`}>
                          {fitDescs[data.fit] || ""}
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {activeTab === "material" && (
                <div className="space-y-6">
                  {data.material && (
                    <div>
                      <h4
                        className={`text-sm font-semibold uppercase tracking-wider ${textPrimary} mb-3`}
                      >
                        Материал
                      </h4>
                      <p className={`${textSecondary} leading-relaxed`}>
                        {data.material}
                      </p>
                    </div>
                  )}
                  {data.care.length > 0 && data.care[0] !== "" && (
                    <div>
                      <h4
                        className={`text-sm font-semibold uppercase tracking-wider ${textPrimary} mb-3`}
                      >
                        Рекомендации по уходу
                      </h4>
                      <div className="grid gap-3">
                        {data.care
                          .filter((c) => c.trim())
                          .map((item, idx) => (
                            <div
                              key={idx}
                              className={`flex items-start gap-3 p-4 rounded-xl ${cardBg} border ${border}`}
                            >
                              <Droplets
                                className={`h-4 w-4 ${accentText} shrink-0 mt-0.5`}
                              />
                              <p className={`text-sm ${textSecondary}`}>
                                {item}
                              </p>
                            </div>
                          ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {activeTab === "delivery" && (
                <div className="space-y-6">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div
                      className={`p-5 rounded-xl ${cardBg} border ${border}`}
                    >
                      <div
                        className={`w-10 h-10 rounded-lg ${accentBg10} flex items-center justify-center mb-4`}
                      >
                        <Truck className={`h-5 w-5 ${accentText}`} />
                      </div>
                      <h4
                        className={`text-sm font-semibold ${textPrimary} mb-2`}
                      >
                        Доставка
                      </h4>
                      <ul className={`space-y-2 text-sm ${textSecondary}`}>
                        <li>• СДЭК — 1–3 дня, от 350 ₽</li>
                        <li>• Почта России — 3–7 дней, от 250 ₽</li>
                        <li>• Курьер по Москве — день в день, 500 ₽</li>
                        <li>• Самовывоз — бесплатно</li>
                      </ul>
                    </div>
                    <div
                      className={`p-5 rounded-xl ${cardBg} border ${border}`}
                    >
                      <div
                        className={`w-10 h-10 rounded-lg ${accentBg10} flex items-center justify-center mb-4`}
                      >
                        <RotateCcw className={`h-5 w-5 ${accentText}`} />
                      </div>
                      <h4
                        className={`text-sm font-semibold ${textPrimary} mb-2`}
                      >
                        Возврат
                      </h4>
                      <ul className={`space-y-2 text-sm ${textSecondary}`}>
                        <li>• 14 дней на возврат без объяснения причин</li>
                        <li>
                          • Товар должен быть с бирками и без следов носки
                        </li>
                        <li>• Возврат средств в течение 3–5 рабочих дней</li>
                        <li>• Обмен на другой размер — бесплатно</li>
                      </ul>
                    </div>
                  </div>
                  <div className={`p-5 rounded-xl ${cardBg} border ${border}`}>
                    <div className="flex items-start gap-3">
                      <Package
                        className={`h-5 w-5 ${accentText} shrink-0 mt-0.5`}
                      />
                      <div>
                        <h4
                          className={`text-sm font-semibold ${textPrimary} mb-1`}
                        >
                          Упаковка
                        </h4>
                        <p className={`text-sm ${textSecondary}`}>
                          Каждый заказ упакован в фирменную коробку MIORU из
                          переработанного картона.
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </motion.div>

      {/* Size Chart Modal */}
      {showSizeChart && data.sizeChart.length > 0 && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-[70] flex items-center justify-center bg-black/60 backdrop-blur-sm"
          onClick={() => setShowSizeChart(false)}
        >
          <motion.div
            initial={{ scale: 0.95, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            onClick={(e) => e.stopPropagation()}
            className={`w-full max-w-lg mx-4 rounded-2xl ${cardBg} border ${border} shadow-2xl overflow-hidden`}
          >
            <div
              className={`flex items-center justify-between p-6 border-b ${border}`}
            >
              <h2
                className={`text-lg font-bold tracking-tighter ${textPrimary}`}
              >
                Таблица размеров
              </h2>
              <button
                onClick={() => setShowSizeChart(false)}
                className={`h-8 w-8 flex items-center justify-center rounded-lg ${textMuted} hover:${textPrimary} transition-colors`}
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="p-6 overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className={`border-b ${border}`}>
                    <th
                      className={`text-left py-3 px-3 text-xs font-semibold uppercase tracking-wider ${textMuted}`}
                    >
                      Размер
                    </th>
                    {data.sizeChart.some((r) => r.chest) && (
                      <th
                        className={`text-left py-3 px-3 text-xs font-semibold uppercase tracking-wider ${textMuted}`}
                      >
                        Грудь
                      </th>
                    )}
                    {data.sizeChart.some((r) => r.waist) && (
                      <th
                        className={`text-left py-3 px-3 text-xs font-semibold uppercase tracking-wider ${textMuted}`}
                      >
                        Талия
                      </th>
                    )}
                    {data.sizeChart.some((r) => r.hips) && (
                      <th
                        className={`text-left py-3 px-3 text-xs font-semibold uppercase tracking-wider ${textMuted}`}
                      >
                        Бёдра
                      </th>
                    )}
                    {data.sizeChart.some((r) => r.length) && (
                      <th
                        className={`text-left py-3 px-3 text-xs font-semibold uppercase tracking-wider ${textMuted}`}
                      >
                        Длина
                      </th>
                    )}
                    {data.sizeChart.some((r) => r.foot_length) && (
                      <th
                        className={`text-left py-3 px-3 text-xs font-semibold uppercase tracking-wider ${textMuted}`}
                      >
                        Длина стопы (см)
                      </th>
                    )}
                    {data.sizeChart.some((r) => r.wrist) && (
                      <th
                        className={`text-left py-3 px-3 text-xs font-semibold uppercase tracking-wider ${textMuted}`}
                      >
                        Запястье (см)
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {data.sizeChart
                    .filter((r) => r.label.trim())
                    .map((row, i) => (
                      <tr
                        key={i}
                        className={`border-b ${border} last:border-b-0`}
                      >
                        <td className={`py-3 px-3 font-medium ${textPrimary}`}>
                          {row.label}
                        </td>
                        {data.sizeChart.some((r) => r.chest) && (
                          <td
                            className={`py-3 px-3 ${textSecondary} font-mono`}
                          >
                            {row.chest || "-"}
                          </td>
                        )}
                        {data.sizeChart.some((r) => r.waist) && (
                          <td
                            className={`py-3 px-3 ${textSecondary} font-mono`}
                          >
                            {row.waist || "-"}
                          </td>
                        )}
                        {data.sizeChart.some((r) => r.hips) && (
                          <td
                            className={`py-3 px-3 ${textSecondary} font-mono`}
                          >
                            {row.hips || "-"}
                          </td>
                        )}
                        {data.sizeChart.some((r) => r.length) && (
                          <td
                            className={`py-3 px-3 ${textSecondary} font-mono`}
                          >
                            {row.length || "-"}
                          </td>
                        )}
                        {data.sizeChart.some((r) => r.foot_length) && (
                          <td
                            className={`py-3 px-3 ${textSecondary} font-mono`}
                          >
                            {row.foot_length || "-"}
                          </td>
                        )}
                        {data.sizeChart.some((r) => r.wrist) && (
                          <td
                            className={`py-3 px-3 ${textSecondary} font-mono`}
                          >
                            {row.wrist || "-"}
                          </td>
                        )}
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          </motion.div>
        </motion.div>
      )}
    </motion.div>
  );
}
