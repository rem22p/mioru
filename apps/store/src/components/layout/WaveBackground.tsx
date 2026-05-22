import { useEffect, useRef } from 'react';

export default function WaveBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animIdRef = useRef<number>(0);
  const isVisibleRef = useRef(true);

  useEffect(() => {
    // Respect prefers-reduced-motion
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReducedMotion) return;

    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let width = window.innerWidth;
    let height = window.innerHeight;
    canvas.width = width;
    canvas.height = height;

    const lines: { x: number; y: number; vx: number; vy: number; alpha: number }[] = [];
    const numLines = 25;

    for (let i = 0; i < numLines; i++) {
      lines.push({
        x: Math.random() * width,
        y: height / 2 + (Math.random() - 0.5) * 200,
        vx: (Math.random() - 0.5) * 0.5,
        vy: (Math.random() - 0.5) * 0.3,
        alpha: Math.random() * 0.3 + 0.1,
      });
    }

    let time = 0;

    const draw = () => {
      if (!isVisibleRef.current) {
        animIdRef.current = requestAnimationFrame(draw);
        return;
      }

      ctx.clearRect(0, 0, width, height);
      time += 0.005;

      lines.forEach((line, i) => {
        ctx.beginPath();
        ctx.strokeStyle = `rgba(114, 158, 132, ${line.alpha})`;
        ctx.lineWidth = 0.5;

        for (let x = 0; x < width; x += 5) {
          const y = line.y +
            Math.sin(x * 0.003 + time + i * 0.5) * 60 +
            Math.sin(x * 0.007 + time * 0.7 + i) * 30 +
            Math.cos(x * 0.001 + time * 0.3) * 40;

          if (x === 0) {
            ctx.moveTo(x, y);
          } else {
            ctx.lineTo(x, y);
          }
        }
        ctx.stroke();

        line.y += line.vy;
        if (line.y < 0 || line.y > height) line.vy *= -1;
      });

      animIdRef.current = requestAnimationFrame(draw);
    };

    draw();

    // Pause when tab is hidden
    const handleVisibilityChange = () => {
      isVisibleRef.current = document.visibilityState === 'visible';
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

    // Throttled resize
    let resizeTimeout: ReturnType<typeof setTimeout>;
    const handleResize = () => {
      clearTimeout(resizeTimeout);
      resizeTimeout = setTimeout(() => {
        width = window.innerWidth;
        height = window.innerHeight;
        canvas.width = width;
        canvas.height = height;
      }, 200);
    };
    window.addEventListener('resize', handleResize, { passive: true });

    return () => {
      cancelAnimationFrame(animIdRef.current);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('resize', handleResize);
      clearTimeout(resizeTimeout);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="fixed inset-0 w-full h-full pointer-events-none"
      style={{ zIndex: 1, opacity: 0.6 }}
    />
  );
}
