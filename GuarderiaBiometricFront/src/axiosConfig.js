import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'https://guarderiabiometricback.onrender.com',
  // El JWT vive en una cookie httpOnly (invisible a JS) en vez de en
  // localStorage — sin withCredentials el navegador no la manda en
  // peticiones cross-origin, y frontend/backend viven en orígenes
  // distintos (ver internal/middleware/auth.go en el backend).
  withCredentials: true,
});

// El token CSRF viaja en memoria, no se lee de document.cookie -- cuando
// frontend y backend viven en dominios de verdad distintos (ej. Render:
// pasitos-frontend.onrender.com / pasitos-backend-xxxx.onrender.com, no solo
// puertos distintos de "localhost" como en desarrollo local) la cookie
// biosafe_csrf pertenece al DOMINIO DEL BACKEND -- document.cookie en una
// página servida desde el frontend nunca puede leerla, sea cual sea su
// SameSite (eso solo decide si el navegador ADJUNTA la cookie a la
// petición, no si OTRO dominio puede leerla vía JS). setCsrfToken() la
// guarda aquí a partir de lo que /login y /me regresan en el body de la
// respuesta -- ver App.jsx (hidratarSesion) y auth.go (handleLogin/handleMe).
let csrfTokenEnMemoria = null;

export function setCsrfToken(token) {
  csrfTokenEnMemoria = token || null;
}

const METODOS_MUTABLES = ['post', 'put', 'patch', 'delete'];

// INTERCEPTOR DE PETICIÓN: en peticiones que modifican datos, reenvía el
// token CSRF guardado en memoria. GET no lo necesita (no muta nada, y así
// no penaliza cache/prefetch).
api.interceptors.request.use((config) => {
  if (METODOS_MUTABLES.includes((config.method || '').toLowerCase()) && csrfTokenEnMemoria) {
    config.headers['X-CSRF-Token'] = csrfTokenEnMemoria;
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
