// Zona horaria de referencia de la guardería (misma que usa el backend en Go:
// "America/Mazatlan" para Culiacán). Centralizar aquí evita que cada componente
// calcule "hoy" con una lógica distinta (UTC, offset manual, etc.), lo cual
// puede mostrar la fecha equivocada cerca de la medianoche.
const ZONA_HORARIA = 'America/Mazatlan';

// Devuelve la fecha local de hoy en formato YYYY-MM-DD.
export function hoyLocal() {
  return new Date().toLocaleDateString('en-CA', { timeZone: ZONA_HORARIA });
}

// Dada una fecha YYYY-MM-DD, regresa el lunes de esa misma semana (también en
// YYYY-MM-DD). Se usa para anclar el selector de "semana de" del Menú Semanal
// a un único lunes en vez de dejar que el usuario elija cualquier día suelto.
export function lunesDeLaSemana(fechaISO) {
  const d = new Date(`${fechaISO}T00:00:00`);
  const diaSemana = d.getDay(); // 0 = domingo ... 6 = sábado
  const diferencia = diaSemana === 0 ? -6 : 1 - diaSemana;
  d.setDate(d.getDate() + diferencia);
  return d.toLocaleDateString('en-CA', { timeZone: 'UTC' });
}

// Dado un lunes YYYY-MM-DD, regresa las 5 fechas de esa semana (lunes a
// viernes) como array de strings YYYY-MM-DD.
export function diasHabilesDeLaSemana(lunesISO) {
  const dias = [];
  for (let i = 0; i < 5; i++) {
    const d = new Date(`${lunesISO}T00:00:00`);
    d.setDate(d.getDate() + i);
    dias.push(d.toLocaleDateString('en-CA', { timeZone: 'UTC' }));
  }
  return dias;
}
