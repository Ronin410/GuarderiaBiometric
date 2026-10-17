import React from 'react';
import { useRegisterSW } from 'virtual:pwa-register/react';
import { RefreshCw, X, WifiOff } from 'lucide-react';

// Cada cuánto se revisa si hay una versión nueva mientras la pestaña sigue
// abierta -- una tablet de recepción o el portal de un papá se puede quedar
// abierto días sin recargar solo, así que no basta con revisar una vez al
// entrar. 30 min balancea "verla pronto" contra gastar datos/batería.
const INTERVALO_REVISION_MS = 30 * 60 * 1000;

// ActualizarApp se monta una sola vez, fuera de <App /> (ver main.jsx), así
// que el aviso funciona igual en el kiosco, el panel de staff, el portal
// del papá y /plataforma -- sin importar en qué pantalla esté cada quien.
//
// A propósito NO se recarga sola: ver el comentario de registerType en
// vite.config.js -- un papá a medio formulario o una tablet a medio escaneo
// facial no debe perder lo que estaba haciendo porque llegó una versión
// nueva. En vez de eso, esto muestra un aviso con botón y la persona decide
// cuándo.
const ActualizarApp = () => {
  const {
    offlineReady: [offlineReady, setOfflineReady],
    needRefresh: [needRefresh, setNeedRefresh],
    updateServiceWorker,
  } = useRegisterSW({
    onRegisteredSW(_url, registration) {
      if (!registration) return;
      setInterval(() => registration.update(), INTERVALO_REVISION_MS);
    },
  });

  const cerrar = () => {
    setOfflineReady(false);
    setNeedRefresh(false);
  };

  // updateServiceWorker(true) solo manda el mensaje SKIP_WAITING -- el
  // "true" no recarga nada por sí solo (verificado con un service worker
  // real: el listener que la librería arma para recargar automáticamente NO
  // se disparó de forma confiable con injectManifest). Recargar al
  // confirmar el cambio de controller (controllerchange, el evento real del
  // navegador) sí se comprobó que funciona siempre -- PERO el listener se
  // arma aquí, en el clic, y no de forma global: clients.claim() en sw.js
  // también dispara controllerchange la primerísima vez que un Service
  // Worker nuevo toma control de una pestaña que cargó antes de que
  // existiera (instalación normal, no una actualización) -- un listener
  // global recargaba de más justo en esa primera visita de cualquiera.
  const actualizar = () => {
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.addEventListener('controllerchange', () => window.location.reload(), { once: true });
    }
    updateServiceWorker(true);
  };

  if (!offlineReady && !needRefresh) return null;

  return (
    <div className="fixed bottom-4 inset-x-4 sm:inset-x-auto sm:right-4 sm:left-auto z-[300] sm:max-w-sm animate-in fade-in slide-in-from-bottom-4 duration-300">
      <div className="bg-forest text-white rounded-2xl shadow-2xl p-4 flex items-start gap-3">
        <div className={`p-2 rounded-xl shrink-0 ${needRefresh ? 'bg-brand-600' : 'bg-forest-light'}`}>
          {needRefresh ? <RefreshCw size={18} /> : <WifiOff size={18} />}
        </div>
        <div className="flex-1 min-w-0">
          <p className="font-black text-sm leading-tight">
            {needRefresh ? 'Hay una versión nueva de Pasitos' : 'Pasitos ya está lista para usarse sin internet'}
          </p>
          {needRefresh && (
            <p className="text-white/60 text-xs mt-1">Actualiza cuando puedas -- no perderás lo que estabas haciendo en otras pantallas.</p>
          )}
          {needRefresh && (
            <button
              onClick={actualizar}
              className="mt-3 bg-brand-600 hover:bg-brand-700 text-white text-xs font-black uppercase px-4 py-2 rounded-xl transition-all active:scale-95"
            >
              Actualizar ahora
            </button>
          )}
        </div>
        <button onClick={cerrar} className="text-white/40 hover:text-white p-1 shrink-0" title="Cerrar">
          <X size={16} />
        </button>
      </div>
    </div>
  );
};

export default ActualizarApp;
