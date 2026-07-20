import axios from 'axios';

const api = axios.create({
  //baseURL: 'http://localhost:8099', // O tu URL de Render
  baseURL: 'https://guarderiabiometricback.onrender.com', // Usa variable de entorno o fallback
});

// INTERCEPTOR DE PETICIÓN: Para enviar el token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token'); // Asegúrate que se llame 'token'
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
}, (error) => {
  return Promise.reject(error);
});

// INTERCEPTOR DE RESPUESTA: Para manejar el 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401 && !error.config.url.includes('/verificar-pin')) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;