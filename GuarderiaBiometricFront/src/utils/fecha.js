// Zona horaria de referencia de la guardería (misma que usa el backend en Go:
// "America/Mazatlan" para Culiacán). Centralizar aquí evita que cada componente
// calcule "hoy" con una lógica distinta (UTC, offset manual, etc.), lo cual
// puede mostrar la fecha equivocada cerca de la medianoche.
const ZONA_HORARIA = 'America/Mazatlan';

// Devuelve la fecha local de hoy en formato YYYY-MM-DD.
export function hoyLocal() {
  return new Date().toLocaleDateString('en-CA', { timeZone: ZONA_HORARIA });
}
