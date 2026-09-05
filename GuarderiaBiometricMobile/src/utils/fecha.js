// Subconjunto de GuarderiaBiometricFront/src/utils/fecha.js -- solo lo que
// esta primera versión de la app (bitácora de "Hoy" navegable día por día)
// necesita. Misma zona horaria fija, mismo criterio: todo se calcula en
// America/Mazatlan sin importar en qué zona esté el teléfono del papá, para
// que "hoy" en la app coincida con "hoy" en la guardería.
const ZONA_HORARIA = 'America/Mazatlan';

export function hoyLocal() {
  return new Date().toLocaleDateString('en-CA', { timeZone: ZONA_HORARIA });
}

export function sumarDias(fechaISO, delta) {
  const d = new Date(`${fechaISO}T00:00:00`);
  d.setDate(d.getDate() + delta);
  return d.toLocaleDateString('en-CA');
}

// lunesDeLaSemana/diasHabilesDeLaSemana: mismo par que utils/fecha.js de
// la web -- dada cualquier fecha, el lunes de esa semana, y dado un lunes,
// las 5 fechas lunes-a-viernes de esa semana.
export function lunesDeLaSemana(fechaISO) {
  const d = new Date(`${fechaISO}T00:00:00`);
  const diaSemana = d.getDay(); // 0 = domingo ... 6 = sábado
  const diferencia = diaSemana === 0 ? -6 : 1 - diaSemana;
  d.setDate(d.getDate() + diferencia);
  return d.toLocaleDateString('en-CA', { timeZone: 'UTC' });
}

export function diasHabilesDeLaSemana(lunesISO) {
  const dias = [];
  for (let i = 0; i < 5; i++) {
    const d = new Date(`${lunesISO}T00:00:00`);
    d.setDate(d.getDate() + i);
    dias.push(d.toLocaleDateString('en-CA', { timeZone: 'UTC' }));
  }
  return dias;
}

export function formatoLargo(fechaISO) {
  const [anio, mes, dia] = fechaISO.split('-').map(Number);
  return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', {
    weekday: 'long', day: 'numeric', month: 'long',
  });
}

// fechaLocal/separadorFecha: mismo par que utils/fecha.js de la web -- el
// separador ("Hoy"/"Ayer"/fecha completa) que se pone entre mensajes de
// chat de días distintos.
export function fechaLocal(isoConHora) {
  return new Date(isoConHora).toLocaleDateString('en-CA', { timeZone: ZONA_HORARIA });
}

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

export function formatoHora(isoConHora) {
  try {
    return new Date(isoConHora).toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit', timeZone: ZONA_HORARIA });
  } catch {
    return '';
  }
}
