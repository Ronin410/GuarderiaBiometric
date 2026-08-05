import axios from 'axios';
import { leerCookie } from './utils/csrf';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'https://guarderiabiometricback.onrender.com',
  // El JWT vive en una cookie httpOnly (invisible a JS) en vez de en
  // localStorage — sin withCredentials el navegador no la manda en
  // peticiones cross-origin, y frontend/backend viven en orígenes
  // distintos (ver internal/middleware/auth.go en el backend).
  withCredentials: true,
});

const METODOS_MUTABLES = ['post', 'put', 'patch', 'delete'];

// INTERCEPTOR DE PETICIÓN: en peticiones que modifican datos, reenvía el
// token CSRF que el backend dejó en una cookie legible por JS. GET no lo
// necesita (no muta nada, y así no penaliza cache/prefetch).
api.interceptors.request.use((config) => {
  if (METODOS_MUTABLES.includes((config.method || '').toLowerCase())) {
    const csrfToken = leerCookie('biosafe_csrf');
    if (csrfToken) {
      config.headers['X-CSRF-Token'] = csrfToken;
    }
  }
  return config;
}, (error) => {
  return Promise.reject(error);
});

// Endpoints donde un 401 significa "credenciales incorrectas" o "todavía no
// hay sesión" (no "la sesión activa expiró") — no deben disparar el
// redirect global. /me es el que la app llama al montar para restaurar la
// sesión: un 401 ahí es simplemente "no estás logueado", parte normal del
// flujo, no un error a mitad de uso.
const RUTAS_SIN_LOGOUT_EN_401 = ['/login', '/verificar-pin', '/me'];

// INTERCEPTOR DE RESPUESTA: Para manejar el 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const url = error.config?.url || '';
    const esRutaExcluida = RUTAS_SIN_LOGOUT_EN_401.some((ruta) => url.includes(ruta));
    if (error.response && error.response.status === 401 && !esRutaExcluida) {
      window.location.href = '/';
    }
    return Promise.reject(error);
  }
);

export default api;
