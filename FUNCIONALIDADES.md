# Funcionalidades — Pasitos (Sistema Biométrico para Guarderías)

> Documento de referencia de todo lo que hace la plataforma hoy, organizado por quién lo usa. Última actualización: incluye el módulo de Administración (Perfiles/Pagos/Estadísticas) y las notificaciones push + portal del papá ampliado.

## 1. Identificación biométrica (Kiosco)

- **Registro de tutores por rostro**: se captura la cara con la cámara y se indexa en AWS Rekognition, ligado a esa guardería específica.
- **Identificación facial en la entrada**: al escanear, el sistema reconoce al tutor y muestra automáticamente a todos sus hijos vinculados y el estado del día de cada uno (ausente / entrada / salida).
- **Registro de asistencia**: el tutor confirma entrada o salida por niño (puede marcar varios a la vez si tiene más de un hijo), con casillas de "¿aseado?" y "¿trae golpe?" y un campo de observaciones.
- **Alternancia automática entrada/salida**: el sistema decide solo si el movimiento es entrada o salida según el último registro del día.
- **Cierre automático nocturno**: a las 11pm, cualquier niño que quedó con "entrada" abierta se marca automáticamente como salida.

## 2. Bitácora diaria (personal)

- **Roster del día**: tarjetas con cada niño, su estado actual, si viene aseado, si trae golpe, y la observación que dejó el tutor al entrar.
- **Formulario de bitácora pedagógica**: desayuno, comida, merienda, esfínter, si durmió siesta, observaciones de la maestra y fotos del día.
- **Envío automático por WhatsApp**: al guardar la bitácora, se genera un enlace de WhatsApp pre-escrito hacia el tutor con el resumen del día (el maestro solo confirma el envío, sin costo).
- **Forzar estatus manualmente** (admin): para corregir o registrar entradas/salidas sin pasar por el escaneo facial.

## 3. Reportes de asistencia (personal)

- Tabla de todos los movimientos del rango de fechas elegido, con nombre del niño, tutor, tipo de movimiento y el detalle completo de la bitácora de ese día.
- Filtro por nombre, columnas ordenables, vista imprimible (pensada para reporte físico/firma de dirección).

## 4. Administración — Perfiles

- **Expediente completo por niño**: fecha de nacimiento, dirección, contacto de emergencia (nombre y teléfono), colegiatura mensual asignada.
- Buscador por nombre, alterna entre ver solo activos o incluir bajas.
- Edición inline desde la misma tarjeta del niño.

## 5. Administración — Pagos

- **Registro manual de pagos**: monto, concepto (colegiatura, inscripción, material, otro), método (efectivo, transferencia, tarjeta), fecha y observaciones — sin pasarela de cobro en línea, sin comisiones.
- **Estado de cuenta automático por mes**, calculado contra la colegiatura configurada en el expediente del niño: **Pagado**, **Parcial**, **Pendiente** o **Vencido**.
- Soporta abonos (varios pagos parciales del mismo concepto en el mismo mes).
- Historial completo por niño, con opción de borrar un registro mal capturado.

## 6. Administración — Estadísticas

- Resumen por niño de: días hábiles del periodo, días que asistió, días que faltó, días que llegó tarde (después de las 9:00am) y % de asistencia.
- Rangos rápidos (esta semana / este mes / este año) o rango de fechas manual.
- Ordenado de peor a mejor asistencia, para detectar de un vistazo quién necesita seguimiento.

## 7. Notificaciones push (gratis)

- El papá activa notificaciones desde su portal con un clic (sin costo, sin WhatsApp Business API, sin SMS).
- Avisos automáticos cuando: el niño registra entrada, registra salida, o el personal actualiza su bitácora del día.
- Llegan al navegador/celular del papá aunque no tenga la página abierta (funciona como una app instalada).
- Limitación real: en iPhone, Apple solo lo permite si el papá agregó la página a su pantalla de inicio (Safari normal no soporta push).

## 8. Portal del papá

- Login separado del personal; el papá solo ve a sus propios hijos.
- **Pestaña Bitácora del día**: comida, siesta, esfínter, fotos y observaciones de la maestra, con selector de fecha.
- **Pestaña Expediente**: fecha de nacimiento, dirección y contacto de emergencia registrados (solo lectura).
- **Pestaña Pagos**: historial de pagos de sus hijos y si están al corriente (solo lectura).
- Resumen de estado de pago del mes actual visible directo en el inicio del portal.
- Botón para activar notificaciones push.

## 9. Reporte público (para compartir sin login)

- Cada niño tiene un enlace único (token) que el personal comparte por WhatsApp tras llenar la bitácora.
- Quien abre el enlace ve el reporte del día sin necesidad de crear cuenta ni iniciar sesión.

## 10. Cuentas, roles y seguridad

- Tres roles: **admin**, **staff** y **papá** (padre/tutor).
- Las pestañas administrativas (Familia, Bitácora, Reportes, Perfiles, Pagos, Estadísticas) están protegidas por PIN para el personal que no es admin.
- Las rutas que exponen datos de **todas** las familias (domicilios, contactos de emergencia, pagos) están bloqueadas a nivel de servidor para cualquier cuenta que no sea admin/staff — un papá no puede leerlas ni llamando la API directo.
- Multi-sede: cada guardería tiene sus propios usuarios, niños, tutores y colección de rostros, completamente aislados de otras guarderías que usen la misma plataforma.
- Límite de intentos (rate limiting) en login, identificación facial y verificación de PIN para frenar ataques de fuerza bruta.
- Contraseñas con hash (bcrypt), tokens de sesión firmados (JWT) con expiración de 24 horas.

## 11. App instalable (PWA)

- La plataforma se puede "instalar" desde el navegador (ícono en pantalla de inicio, funciona a pantalla completa).
- Funciona con cámara en el kiosco gracias a HTTPS obligatorio.
- Actualizaciones de la app se aplican solas sin que el usuario tenga que hacer nada.

---

### Resumen por quién lo usa

| Rol | Qué puede hacer |
|---|---|
| **Kiosco / Staff** | Escanear rostros, registrar entrada/salida, llenar bitácora diaria |
| **Admin** | Todo lo anterior + Familia (directorio de tutores), Perfiles, Pagos, Estadísticas, Reportes, sin necesidad de PIN |
| **Staff sin PIN de admin** | Kiosco y Registro libres; el resto de pestañas piden PIN |
| **Papá / Tutor** | Ver a sus hijos, bitácora diaria, expediente, estado de pagos, activar notificaciones — nada de otras familias |
| **Público (sin cuenta)** | Solo el reporte del día de un niño específico, vía enlace compartido |
