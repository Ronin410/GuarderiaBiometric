// Decodifica el payload de un JWT (base64url) sin verificar la firma —
// solo para leer localmente cuándo expira, la verificación real la hace el backend.
export function leerExpiracionToken(token) {
  try {
    const payloadBase64 = token.split('.')[1];
    const payloadJson = atob(payloadBase64.replace(/-/g, '+').replace(/_/g, '/'));
    const payload = JSON.parse(payloadJson);
    if (!payload.exp) return null;
    return payload.exp * 1000; // exp viene en segundos, Date.now() en milisegundos
  } catch {
    return null;
  }
}

export function segundosHastaExpirar(token) {
  const expiraEn = leerExpiracionToken(token);
  if (!expiraEn) return null;
  return Math.floor((expiraEn - Date.now()) / 1000);
}
