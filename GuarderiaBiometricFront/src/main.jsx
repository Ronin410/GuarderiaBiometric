import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'
import ActualizarApp from './components/ActualizarApp.jsx'
// Efecto de import: registra el listener de beforeinstallprompt lo antes
// posible (ver el comentario dentro del archivo) -- InstalarApp.jsx, dentro
// de DashboardPadre, lo lee después, pero para entonces ya no se lo puede
// perder aunque el componente tarde en montarse.
import './utils/pwaInstall.js'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
    {/* Fuera de <App />, montado una sola vez: el aviso de "hay versión
        nueva" debe verse igual en el kiosco, el panel de staff, el portal
        del papá y /plataforma -- no depende de qué ruta esté activa. */}
    <ActualizarApp />
  </StrictMode>,
)
