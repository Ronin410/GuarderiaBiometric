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

export function formatoLargo(fechaISO) {
  const [anio, mes, dia] = fechaISO.split('-').map(Number);
  return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', {
    weekday: 'long', day: 'numeric', month: 'long',
  });
}
