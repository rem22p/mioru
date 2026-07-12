import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import './index.css';

const savedTheme = localStorage.getItem('ui_theme');
if (savedTheme === 'light') {
  document.documentElement.classList.add('light');
}

const savedScale = localStorage.getItem('ui_scale');
if (savedScale) {
  document.documentElement.style.setProperty('--ui', savedScale + 'px');
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
);
