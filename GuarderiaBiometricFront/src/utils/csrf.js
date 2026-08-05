// El JWT vive en una cookie httpOnly, invisible a JavaScript a propósito.
// biosafe_csrf es su contraparte legible: el backend la manda al loguearse
// y el frontend la reenvía en el header X-CSRF-Token en cada petición que
// modifica datos (patrón "double-submit cookie" — ver internal/middleware/
// auth.go en el backend). Sin esto, una cookie de sesión que el navegador
// adjunta solo sería vulnerable a CSRF.
export function leerCookie(nombre) {
  const match = document.cookie.match(new RegExp('(?:^|; )' + nombre + '=([^;]*)'));
  return match ? decodeURIComponent(match[1]) : null;
}
