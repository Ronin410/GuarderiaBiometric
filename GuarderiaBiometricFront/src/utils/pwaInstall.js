// pwaInstall.js: captura el evento beforeinstallprompt (Android/Chrome) lo
// ANTES posible -- si nadie lo escucha en el momento exacto en que el
// navegador lo dispara, se pierde para siempre en esa carga de página. Por
// eso este módulo se importa una sola vez en main.jsx (efecto de import, no
// hace falta usar nada de él ahí) para registrar el listener apenas arranca
// la app, sin depender de que InstalarApp.jsx ya esté montado.
let promptDiferido = null;
const listeners = new Set();

const avisar = () => listeners.forEach((cb) => cb(promptDiferido));

if (typeof window !== 'undefined') {
  window.addEventListener('beforeinstallprompt', (evento) => {
    // Sin este preventDefault(), Chrome muestra su propio mini-banner de
    // instalación además del nuestro -- lo bloqueamos para controlar
    // nosotros cuándo y cómo se pide.
    evento.preventDefault();
    promptDiferido = evento;
    avisar();
  });

  window.addEventListener('appinstalled', () => {
    promptDiferido = null;
    avisar();
  });
}

export function obtenerPromptDiferido() {
  return promptDiferido;
}

// Devuelve una función para des-suscribirse (mismo patrón que useEffect
// espera de vuelta en su cleanup).
export function suscribirsePrompt(callback) {
  listeners.add(callback);
  return () => listeners.delete(callback);
}

// Heurísticas de plataforma -- no hay ninguna API estándar para "puedo
// instalar esto"; iOS Safari en particular nunca dispara beforeinstallprompt
// (Apple no lo soporta), así que ahí no queda otra que detectar la
// plataforma y explicar los pasos a mano.
export function detectarPlataforma() {
  const ua = window.navigator.userAgent;

  // iPadOS 13+ reporta un user-agent de escritorio (Macintosh) por
  // default -- se distingue de una Mac real porque un iPad sí tiene
  // pantalla táctil multi-touch.
  const esIPadComoMac = navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1;
  const esIOS = (/iPad|iPhone|iPod/.test(ua) && !window.MSStream) || esIPadComoMac;
  const esAndroid = /Android/.test(ua);

  const yaInstalada = window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone === true;

  // Navegadores empotrados en otra app (se abre el link desde un mensaje de
  // WhatsApp o una publicación de Facebook/Instagram): ni beforeinstallprompt
  // ni el botón "Compartir" de iOS funcionan igual ahí -- hay que pedir que
  // abran el link en su navegador normal.
  const esNavegadorEnApp = /FBAN|FBAV|Instagram|Line\/|MicroMessenger|\bWhatsApp\b/i.test(ua);

  return { esIOS, esAndroid, yaInstalada, esNavegadorEnApp };
}
