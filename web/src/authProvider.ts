import type { AuthProvider } from 'react-admin';

const API_URL = '/api';

const authProvider: AuthProvider = {
  login: async ({ username, password }) => {
    const response = await fetch(`${API_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || 'Login failed');
    }
    const { token, user } = await response.json();
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
  },

  logout: async () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
  },

  checkAuth: async () => {
    const token = localStorage.getItem('token');
    if (!token) throw new Error('Not authenticated');
  },

  checkError: async (error) => {
    if (error?.status === 401 || error?.message === '401') {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      throw new Error('Session expired');
    }
  },

  getIdentity: async () => {
    const token = localStorage.getItem('token');
    if (!token) throw new Error('Not authenticated');

    const response = await fetch(`${API_URL}/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!response.ok) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      throw new Error('Session expired');
    }
    const user = await response.json();
    localStorage.setItem('user', JSON.stringify(user));
    return {
      id: user.id,
      fullName: user.display_name || user.username,
      avatar: undefined,
    };
  },

  getPermissions: async () => {
    const stored = localStorage.getItem('user');
    if (stored) {
      const user = JSON.parse(stored);
      return user.role || 'viewer';
    }
    return 'viewer';
  },
};

export default authProvider;
