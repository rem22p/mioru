import { useState, useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import Register from './pages/Register';
import ForgotPassword from './pages/ForgotPassword';
import ResetPassword from './pages/ResetPassword';
import AdminLayout from './components/layout/AdminLayout';

export default function App() {
  const [authed, setAuthed] = useState(!!localStorage.getItem('token'));

  useEffect(() => {
    const check = () => setAuthed(!!localStorage.getItem('token'));
    window.addEventListener('storage', check);
    window.addEventListener('focus', check);
    return () => {
      window.removeEventListener('storage', check);
      window.removeEventListener('focus', check);
    };
  }, []);

  return (
    <Routes>
      <Route path="/login" element={!authed ? <Login /> : <Navigate to="/" />} />
      <Route path="/register" element={!authed ? <Register /> : <Navigate to="/" />} />
      <Route path="/forgot-password" element={!authed ? <ForgotPassword /> : <Navigate to="/" />} />
      <Route path="/reset/:token" element={!authed ? <ResetPassword /> : <Navigate to="/" />} />
      <Route path="/*" element={authed ? <AdminLayout /> : <Navigate to="/login" />} />
    </Routes>
  );
}
