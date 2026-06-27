import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

export function useScrollToTop() {
  const { pathname } = useLocation();

  useEffect(() => {
    // requestAnimationFrame ensures the scroll happens after
    // the browser has painted the new page content, including
    // lazy-loaded components that may push the fold.
    const raf = requestAnimationFrame(() => {
      window.scrollTo(0, 0);
    });
    return () => cancelAnimationFrame(raf);
  }, [pathname]);
}
