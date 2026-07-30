import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'https://guarderiabiometricback.onrender.com',
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

// Endpoints donde un 401 significa "credenciales incorrectas" (el usuario
// todavía no tiene sesión), no "la sesión expiró" — no deben disparar el
// logout global, o el mensaje de error nunca le da tiempo de verse antes
// de que la página se recargue.
const RUTAS_SIN_LOGOUT_EN_401 = ['/login', '/verificar-pin'];

// INTERCEPTOR DE RESPUESTA: Para manejar el 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const url = error.config?.url || '';
    const esRutaExcluida = RUTAS_SIN_LOGOUT_EN_401.some((ruta) => url.includes(ruta));
    if (error.response && error.response.status === 401 && !esRutaExcluida) {
      localStorage.clear();
      window.location.href = '/';
    }
    return Promise.reject(error);
  }
);

export default api;