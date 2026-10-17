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

// fechaLocal: dada una marca de tiempo ISO con hora (ej. "creado_en" de un
// mensaje de chat), regresa su fecha local YYYY-MM-DD en la zona horaria de
// la guardería -- para agrupar mensajes por día sin importar en qué zona
// horaria esté el navegador de quien los lee.
export function fechaLocal(isoConHora) {
  return new Date(isoConHora).toLocaleDateString('en-CA', { timeZone: ZONA_HORARIA });
}

// separadorFecha: el texto del separador que se muestra entre mensajes de
// chat de días distintos -- "Hoy" / "Ayer" / "30 de agosto de 2026". Se usa
// en los 4 chats de la app (ChatPadre, PanelChat, SoporteChat,
// PanelSoportePlataforma) para que quede claro cuándo se mandó cada
// mensaje sin tener que repetir la fecha en cada burbuja.
export function separadorFecha(isoConHora) {
  const fecha = fechaLocal(isoConHora);
  const hoy = hoyLocal();
  if (fecha === hoy) return 'Hoy';

  const d = new Date(`${hoy}T00:00:00`);
  d.setDate(d.getDate() - 1);
  if (fecha === d.toLocaleDateString('en-CA', { timeZone: 'UTC' })) return 'Ayer';

  return new Date(isoConHora).toLocaleDateString('es-MX', {
    day: 'numeric', month: 'long', year: 'numeric', timeZone: ZONA_HORARIA,
  });
}
