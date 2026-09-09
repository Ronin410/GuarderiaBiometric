import axios from 'axios';

// Mismo backend que ya usa la web en producción. OJO: el dominio real es
// pasitos-backend-f411.onrender.com -- "guarderiabiometricback.onrender.com"
// (que quedó puesto aquí antes) era solo el valor de ejemplo del fallback
// en axiosConfig.js de la web, no el dominio real; la web en producción lo
// pisa con la variable de entorno VITE_API_URL, que aquí no existe, así
// que hay que ponerlo tal cual.
//
// Ver API_MOVIL.md sobre por qué, a diferencia de la web, aquí no hace
// falta configurar CORS/orígenes en el backend: una app nativa (no un
// WebView) no manda encabezado Origin, así que el middleware de CORS de
// gin-contrib/cors ni siquiera evalúa la petición como "cross-origin".
export const API_URL = 'https://pasitos-backend-f411.onrender.com';

// withCredentials NO se pone aquí a propósito -- es una instrucción para
// que un NAVEGADOR mande cookies en una petición cross-origin (por eso sí
// se usa en axiosConfig.js de la web); en React Native no significa nada
// para el cliente HTTP nativo (que ya manda cookies solo, sin pedírselo,
// ver el comentario de abajo) y en algunas versiones del adaptador XHR de
// React Native poner withCredentials:true en Android hace TRONAR la
// petición entera en vez de ser un no-op inofensivo como en un navegador
// -- si el login nunca llega a responder (error genérico sin
// err.response), esta es la causa más probable.
const api = axios.create({
  baseURL: API_URL,
});

// El JWT vive en una cookie httpOnly que el backend pone en el login --
// a diferencia del WebView de un navegador, el cliente HTTP nativo de
// iOS/Android (sobre el que corre axios aquí) guarda y reenvía las cookies
// del dominio automáticamente entre peticiones, sin que la app tenga que
// leerlas ni gestionarlas a mano.
//
// El token CSRF sí viaja en memoria (igual que en axiosConfig.js de la
// web): no es una cookie httpOnly, así que /login y /me lo regresan en el
// body de la respuesta y esta app lo reenvía en X-CSRF-Token en cada
// petición que modifica datos.
let csrfTokenEnMemoria = null;

export function setCsrfToken(token) {
  csrfTokenEnMemoria = token || null;
}

const METODOS_MUTABLES = ['post', 'put', 'patch', 'delete'];

api.interceptors.request.use((config) => {
  if (METODOS_MUTABLES.includes((config.method || '').toLowerCase()) && csrfTokenEnMemoria) {
    config.headers['X-CSRF-Token'] = csrfTokenEnMemoria;
  }
  return config;
});

export default api;
