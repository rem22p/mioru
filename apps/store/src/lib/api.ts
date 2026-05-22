const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000';

interface ApiError {
  error: string;
}

async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token');
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'Network error' }));
    throw new Error((body as ApiError).error || 'Request failed');
  }
  return res.json();
}

export { api, API_URL };
