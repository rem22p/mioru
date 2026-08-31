import { useState, useRef, useEffect } from "react";

const CITIES = [
  "Кишинев", "Джурджулешть", "Бельцы", "Дондюшаны", "Тирасполь",
  "Дрокия", "Бендеры", "Дубоссары", "Рыбница", "Думбрава",
  "Бессарабка", "Единец", "Бируинца", "Дэнчень", "Ближний Хутор",
  "Кагул", "Бричаны", "Каинары", "Бубуечь", "Калараш",
  "Будешть", "Каменка", "Быковец", "Кантемир", "Бэчой",
  "Каушаны", "Вадул-луй-Водэ", "Кодру", "Ватра", "Колоница",
  "Вулканешты", "Комрат", "Гидигичь", "Конгаз", "Гиндешты",
  "Корнешты", "Глодяны", "Костешты", "Григориополь", "Кочиеры",
  "Грэтиешть", "Красное", "Днестровск", "Криково", "Криулень",
  "Суклея", "Крузешть", "Сынджера", "Купчинь", "Сынжерей",
  "Леова", "Тараклия", "Липканы", "Твардица", "Леушень",
  "Теленешты", "Маркулешты", "Терновка", "Маяк", "Тохатин",
  "Ниспорены", "Трушень", "Новотираспольский", "Унгень", "Новые Анены",
  "Фалешты", "Окница", "Флорешты", "Орхей", "Фрунзе",
  "Отачь", "Хынчешты", "Парканы", "Чадыр-Лунга", "Резина",
  "Чимишлия", "Первомайск", "Чореску", "Рышканы", "Шолданешты",
  "Слободзея", "Штефан-Водэ", "Сороки", "Яловены", "Страшены",
  "Яргара", "Стэучень",
].sort((a, b) => a.localeCompare(b, "ru"));

interface CityAutocompleteProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  testId?: string;
  /** #83: wires an external <label htmlFor> to the inner input. */
  id?: string;
}

export default function CityAutocomplete({
  value,
  onChange,
  placeholder,
  className = "",
  testId,
  id,
}: CityAutocompleteProps) {
  const [open, setOpen] = useState(false);
  const [highlightIndex, setHighlightIndex] = useState(-1);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = value
    ? CITIES.filter((c) => c.toLowerCase().includes(value.toLowerCase()))
    : CITIES;

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, []);

  const select = (city: string) => {
    onChange(city);
    setOpen(false);
    setHighlightIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        setOpen(true);
        setHighlightIndex(0);
        e.preventDefault();
      }
      return;
    }
    switch (e.key) {
      case "ArrowDown":
        setHighlightIndex((i) => Math.min(i + 1, filtered.length - 1));
        e.preventDefault();
        break;
      case "ArrowUp":
        setHighlightIndex((i) => Math.max(i - 1, 0));
        e.preventDefault();
        break;
      case "Enter":
        if (highlightIndex >= 0 && highlightIndex < filtered.length) {
          select(filtered[highlightIndex]);
        }
        e.preventDefault();
        break;
      case "Escape":
        setOpen(false);
        break;
    }
  };

  return (
    <div ref={wrapperRef} className="relative">
      <input
        ref={inputRef}
        id={id}
        data-testid={testId}
        type="text"
        value={value}
        onChange={(e) => {
          onChange(e.target.value);
          if (!open) setOpen(true);
          setHighlightIndex(-1);
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={handleKeyDown}
        className={className}
        placeholder={placeholder}
        autoComplete="off"
      />
      {open && filtered.length > 0 && (
        <div className="absolute z-50 top-full mt-1 left-0 right-0 max-h-48 overflow-y-auto rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] shadow-lg">
          {filtered.map((city, i) => (
            <button
              key={city}
              type="button"
              onClick={() => select(city)}
              className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                i === highlightIndex
                  ? "bg-[#44944A]/10 text-[#44944A]"
                  : "text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]"
              }`}
            >
              {city}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
