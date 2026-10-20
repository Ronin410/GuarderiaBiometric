// El JWT ahora vive en una cookie httpOnly (invisible a JavaScript), así
// que ya no hay token que decodificar en el cliente: la fecha de
// expiración la manda el backend directamente (campo expires_at de
// /login y /me, en segundos unix), y este helper solo hace la resta.
export function segundosHastaExpirar(expiraEnUnix) {
  if (!expiraEnUnix) return null;
  return Math.floor(expiraEnUnix - Date.now() / 1000);
}
