import { useThemeStore } from '@/stores/themeStore';
import { useUiStore } from '@/stores/uiStore';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Moon, Sun } from 'lucide-react';

const MIN = 11;
const MAX = 18;

export default function Settings() {
  const { theme, setTheme } = useThemeStore();
  const { scale, setScale } = useUiStore();

  return (
    <div className="max-w-2xl mx-auto p-6 space-y-8">
      <h2 className="text-xl font-semibold">Настройки</h2>

      <div className="space-y-2">
        <Label>Тема</Label>
        <div className="flex gap-2">
          <Button
            variant={theme === 'dark' ? 'default' : 'outline'}
            onClick={() => setTheme('dark')}
          >
            <Moon className="h-4 w-4 mr-2" />
            Тёмная
          </Button>
          <Button
            variant={theme === 'light' ? 'default' : 'outline'}
            onClick={() => setTheme('light')}
          >
            <Sun className="h-4 w-4 mr-2" />
            Светлая
          </Button>
        </div>
      </div>

      <div className="space-y-2">
        <Label>Размер интерфейса</Label>
        <div className="flex items-center gap-4">
          <input
            type="range"
            min={MIN}
            max={MAX}
            value={scale}
            onChange={e => setScale(parseInt(e.target.value))}
            className="flex-1 h-2 bg-[var(--color-bg-secondary)] rounded-lg appearance-none cursor-pointer accent-[var(--color-accent)]"
          />
          <span className="text-sm font-mono w-12 text-right">{scale}px</span>
        </div>
      </div>
    </div>
  );
}
