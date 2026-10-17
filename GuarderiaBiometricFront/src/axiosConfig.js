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

// Respaldo de la sesión del papá -- ver el comentario largo en
// middleware.Auth (backend) y hidratarSesion (App.jsx) sobre por qué: la
// cookie de sesión es "cross-site" (frontend y backend en subdominios
// distintos) y el navegador puede purgarla al cerrar del todo la PWA
// instalada y reabrirla, aunque el JWT siga vigente. Guardar el JWT
// TAMBIÉN aquí (solo se recibe en el body de /login y /me cuando
// rol=="papa", nunca para admin/staff) y mandarlo por Authorization: Bearer
// como respaldo evita depender de esa decisión del navegador. Persiste en
// localStorage (sobrevive a que se mate el proceso de la PWA, a diferencia
// de una variable en memoria) y se espeja en memoria para no releer
// localStorage en cada petición.
const CLAVE_TOKEN_PAPA = 'pasitos_sesion_token';
let authTokenEnMemoria = null;
try {
  authTokenEnMemoria = localStorage.getItem(CLAVE_TOKEN_PAPA) || null;
} catch {
  // Storage no disponible (modo privado agresivo, etc.) -- la sesión sigue
  // funcionando con solo la cookie, como antes de este respaldo.
}

export function setAuthToken(token) {
  authTokenEnMemoria = token || null;
  try {
    if (token) {
      localStorage.setItem(CLAVE_TOKEN_PAPA, token);
    } else {
      localStorage.removeItem(CLAVE_TOKEN_PAPA);
    }
  } catch {
    // Igual que arriba -- sin storage persistente, el respaldo solo dura
    // mientras la pestaña siga viva, pero no rompe nada.
  }
}

const METODOS_MUTABLES = ['post', 'put', 'patch', 'delete'];

// INTERCEPTOR DE PETICIÓN: en peticiones que modifican datos, reenvía el
// token CSRF guardado en memoria. GET no lo necesita (no muta nada, y así
// no penaliza cache/prefetch). También manda el respaldo de sesión del
// papá (si existe) en TODAS las peticiones, GET incluido -- es lo que deja
// a /me restaurar la sesión aunque la cookie ya no esté. El backend
// siempre prioriza la cookie cuando ambas llegan, así que esto nunca pisa
// una sesión de staff/admin en el mismo dispositivo.
api.interceptors.request.use((config) => {
  if (METODOS_MUTABLES.includes((config.method || '').toLowerCase()) && csrfTokenEnMemoria) {
    config.headers['X-CSRF-Token'] = csrfTokenEnMemoria;
  }
  if (authTokenEnMemoria) {
    config.headers['Authorization'] = `Bearer ${authTokenEnMemoria}`;
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
