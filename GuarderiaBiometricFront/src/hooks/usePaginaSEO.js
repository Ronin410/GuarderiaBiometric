import { useEffect } from 'react';

// usePaginaSEO -- Pasitos es una SPA con un solo index.html (Vite build +
// Render static site reescribe toda ruta a ese mismo archivo, ver routes en
// render.yaml). Eso significa que SIN este hook, Google ve el mismo
// <title>/<meta description> en "/", "/registro-guarderia", "/terminos",
// etc. -- títulos duplicados entre páginas, que es justo lo que penaliza
// que cada una rankee para su propia búsqueda.
//
// Esto NO es tan bueno como pre-renderizar HTML estático por ruta (lo que
// vería un bot que no ejecuta JavaScript, como el que arma la tarjeta de
// WhatsApp al compartir un link), pero Google sí renderiza JavaScript antes
// de indexar -- por eso esto alcanza para el propósito de rankear en
// búsqueda orgánica, aunque no alcance para las vistas previas de redes
// sociales (esas siguen usando el og:title/og:description fijo de
// index.html).
//
// Se restaura al desmontar para que navegar de /registro-guarderia de
// vuelta a "/" no se quede pegado con el título de la página anterior.
export function usePaginaSEO({ titulo, descripcion, ruta }) {
  useEffect(() => {
    const tituloAnterior = document.title;
    const metaDescripcion = document.querySelector('meta[name="description"]');
    const descripcionAnterior = metaDescripcion?.getAttribute('content');
    const linkCanonical = document.querySelector('link[rel="canonical"]');
    const canonicalAnterior = linkCanonical?.getAttribute('href');

    document.title = titulo;
    if (metaDescripcion && descripcion) {
      metaDescripcion.setAttribute('content', descripcion);
    }
    if (linkCanonical && ruta) {
      linkCanonical.setAttribute('href', `https://pasitos-frontend.onrender.com${ruta}`);
    }

    return () => {
      document.title = tituloAnterior;
      if (metaDescripcion && descripcionAnterior) {
        metaDescripcion.setAttribute('content', descripcionAnterior);
      }
      if (linkCanonical && canonicalAnterior) {
        linkCanonical.setAttribute('href', canonicalAnterior);
      }
    };
  }, [titulo, descripcion, ruta]);
}
