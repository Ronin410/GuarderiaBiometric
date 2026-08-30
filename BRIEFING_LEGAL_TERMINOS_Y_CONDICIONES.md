# Briefing técnico para el abogado — Términos y Condiciones / Aviso de Privacidad de Pasitos

> Este documento NO son los Términos y Condiciones. Es el insumo técnico que el
> representante legal necesita para redactarlos: qué datos se procesan, cómo,
> dónde viven, quién es responsable de qué, y qué controles ya existen en el
> sistema. Todo lo marcado con **[ ⚠️ COMPLETAR ]** es una decisión de negocio
> o un dato legal que el equipo debe llenar antes de que el abogado pueda
> cerrar el documento.

## 1. Qué es Pasitos y quién contrata qué

Pasitos es una plataforma SaaS (software como servicio) que guarderías,
estancias infantiles y centros de cuidado contratan para digitalizar:
reconocimiento facial de tutores en entradas/salidas, bitácora diaria,
chat privado familia-staff, notificaciones, circulares, encuestas,
calendario, pedidos de comedor, control de pagos, y administración de
personal.

Hay dos relaciones contractuales distintas que necesitan documentos
separados:

1. **Pasitos ↔ la guardería** (el cliente que paga/usa la plataforma):
   Términos de Servicio + un **Acuerdo de Encargo de Tratamiento de Datos**
   (bajo la LFPDPPP, porque Pasitos procesa datos personales -- incluyendo
   datos biométricos y de menores -- **en nombre y por cuenta** de la
   guardería, que es quien decide qué datos recolectar de sus familias).
2. **La guardería ↔ los papás/tutores** (las familias): el "Aviso de
   Privacidad" que cada guardería ya redacta y publica ella misma dentro de
   la plataforma (ver sección 7) -- ese documento es responsabilidad de
   cada guardería, no de Pasitos, pero Pasitos les da la herramienta para
   mostrarlo y capturar la aceptación.

En términos de la LFPDPPP (Ley Federal de Protección de Datos Personales en
Posesión de los Particulares, México):

- **La guardería es el "Responsable"** del tratamiento de los datos de sus
  familias (ella decide qué recolectar y para qué).
- **Pasitos es el "Encargado"**: trata esos datos únicamente siguiendo las
  instrucciones de la guardería, a través del software.

Esta distinción es la base de por qué se necesitan los dos documentos del
punto 1 -- el Encargado necesita un contrato que limite lo que puede hacer
con los datos (no puede usarlos para fines propios, debe borrarlos al
terminar el contrato, debe reportar incidentes de seguridad, etc.).

## 2. Qué datos personales procesa el sistema

### 2.1 De los tutores/padres

- Nombre completo.
- Usuario y contraseña del portal (contraseña guardada con hash bcrypt,
  nunca en texto plano).
- **Datos biométricos: la plantilla facial generada por AWS Rekognition**
  a partir de una foto tomada en el kiosco de la guardería. Esta es
  **la categoría de dato más sensible que maneja el sistema** -- la
  LFPDPPP la trata como "dato sensible" (Art. 3, fracción VI), lo que
  exige consentimiento expreso y por escrito (no basta el consentimiento
  tácito que sirve para datos no sensibles).
- Teléfono/celular (opcional, si la guardería activa notificaciones por
  WhatsApp).
- Dirección, en algunos flujos del expediente del niño.
- Historial de mensajes de chat con el staff.
- Historial de pagos de colegiatura.

### 2.2 De los niños

- Nombre completo, fecha de nacimiento.
- Dirección, contacto de emergencia (nombre y teléfono).
- Colegiatura mensual (dato económico de la familia).
- **Bitácora diaria**: alimentación (desayuno/comida/merienda), si durmió
  siesta, control de esfínteres, y un campo de observaciones libre --
  este último campo, en la práctica, puede llegar a contener información
  de salud si el staff anota algo médico (alergias, medicamentos, etc.),
  aunque el sistema no tiene un campo dedicado a "datos de salud". El
  campo de "notas" de pedidos de comedor también se usa hoy para anotar
  alergias alimentarias.
- Fotos tomadas por el staff durante el día (bitácora), guardadas en un
  bucket privado de AWS S3.
- Documentos de inscripción que el staff suba (acta de nacimiento, cartilla
  de vacunación, etc. -- el catálogo de qué documentos pedir lo configura
  cada guardería).
- Historial de asistencia (entradas/salidas con fecha y hora).
- **El rostro del niño NO se procesa nunca.** Solo se enrola el rostro del
  tutor/adulto que lo recoge -- este es un punto que vale la pena que el
  Aviso de Privacidad deje muy claro, porque es una fuente común de
  confusión de los papás.

Los niños, al ser menores de edad, no pueden dar su propio consentimiento
-- el consentimiento lo da el tutor, y así está implementado (ver sección
7). Además de la LFPDPPP, esto probablemente toca la Ley General de los
Derechos de Niñas, Niños y Adolescentes -- el abogado debe confirmar si
aplican requisitos adicionales de "interés superior de la niñez" al
documento.

### 2.3 Del personal de la guardería

- Nombre, usuario, contraseña (hash bcrypt), rol (admin/staff), PIN de 4
  dígitos para acciones sensibles.
- Permisos personalizados por área (qué secciones del panel puede tocar
  cada cuenta).
- Horarios/turnos y registro de horas trabajadas, si la guardería usa ese
  módulo.

### 2.4 Metadatos técnicos

- Dirección IP y timestamp de cada evento sensible (login, exportación de
  datos, eliminación de un tutor, etc.) -- se guarda en una bitácora de
  auditoría interna (`logs_acceso`), no visible para las familias.
- Logs de la aplicación (errores del servidor) -- no incluyen contenido de
  mensajes de chat a propósito (ver sección 6).

## 3. Dónde viven los datos (proveedores / "sub-encargados")

El abogado necesita esta lista para la sección de "transferencias" y
"subcontratación" del contrato:

| Proveedor | Qué guarda | Nota |
|---|---|---|
| **AWS Rekognition** | Las plantillas biométricas faciales (no las fotos originales) | Región: **[ ⚠️ COMPLETAR -- confirmar la región de AWS configurada, ej. us-east-1 ]**. Si la región no es México, esto es una **transferencia internacional de datos** y la LFPDPPP exige informarlo en el aviso de privacidad. |
| **AWS S3** | Fotos de bitácora, documentos de inscripción, PDFs de avisos de privacidad | Bucket **privado** (sin acceso público); las fotos se sirven vía URLs firmadas que expiran en 1 hora. |
| **Base de datos PostgreSQL** | Todo lo demás: expedientes, mensajes, pagos, asistencia, cuentas | **[ ⚠️ COMPLETAR -- nombre del proveedor de hosting de la base de datos compartida y su política de backups/ubicación ]**. |
| **Render** (hosting) | Aloja el backend y el frontend de la aplicación | **[ ⚠️ COMPLETAR -- confirmar región de despliegue de Render ]**. |
| **Stripe** (cuando se active) | Datos de pago con tarjeta | Pasitos **nunca** ve ni guarda el número de tarjeta -- eso lo procesa Stripe directamente (Stripe Checkout), y Stripe es quien mantiene el cumplimiento PCI-DSS. Hoy esta función está **desactivada** en producción. |
| **Web Push (VAPID)** | Suscripciones de notificaciones push del navegador | Sin servicio de terceros -- es el protocolo estándar del navegador, sin pasar por un proveedor externo de notificaciones. |

## 4. Seguridad ya implementada (para la cláusula de "medidas de seguridad")

Esto es lo que el sistema YA hace hoy, útil para que el contrato no
prometa algo que no existe y para poder decir con precisión qué
salvaguardas hay:

- Contraseñas con hash `bcrypt` (nunca en texto plano).
- Sesión vía cookie `httpOnly` + JWT, con CSRF token de doble verificación
  en cada petición que modifica datos.
- PIN adicional de 4 dígitos para entrar a secciones administrativas
  sensibles, con expiración a los 30 minutos de inactividad.
- Permisos por área: una cuenta de staff solo ve/toca lo que el admin le
  autorizó explícitamente.
- Multi-tenant a nivel de fila: los datos de cada guardería están
  aislados por `guarderia_id` en cada consulta -- una guardería nunca ve
  datos de otra.
- Bucket de fotos/documentos privado, sin acceso público directo.
- Límite de tasa (rate limiting) en login y verificación de PIN, para
  frenar intentos de fuerza bruta.
- Bitácora de auditoría de accesos sensibles (login, exportación de datos,
  eliminación de una familia, etc.).
- Comunicación cifrada (HTTPS/TLS) en todo momento.
- Límite de conexiones a la base de datos configurado explícitamente (para
  no degradar el servicio bajo carga).

Lo que **NO** existe todavía (para que el contrato no prometa de más, o
para decidir si se agrega antes de firmar clientes más grandes):

- No hay cifrado en reposo (at-rest) declarado explícitamente a nivel de
  aplicación más allá de lo que el proveedor de base de datos/AWS haga por
  defecto -- **[ ⚠️ COMPLETAR -- confirmar con el proveedor de la base de
  datos y con AWS qué cifrado en reposo aplican por defecto ]**.
- No hay un proceso formal de respuesta a incidentes de seguridad
  documentado (solo el que exige la ley: notificar sin dilación al
  Responsable si el Encargado detecta una vulneración).
- No hay certificaciones de seguridad de terceros (ISO 27001, SOC 2, etc.)
  -- normal para una plataforma en etapa piloto, pero el contrato no debe
  insinuar que sí las hay.

## 5. Derechos ARCO (Acceso, Rectificación, Cancelación, Oposición)

Ya implementados en el sistema, del lado del tutor:

- **Acceso**: un tutor puede exportar su expediente completo (sus datos,
  los de sus hijos vinculados, bitácoras y consentimientos) en un archivo
  descargable.
- **Cancelación**: un tutor puede pedir que se elimine su perfil -- esto
  borra su rostro de Rekognition, su cuenta, sus suscripciones de
  notificaciones, y desvincula (sin borrar) su historial de asistencia
  (que se conserva como parte del expediente del niño, no del tutor, por
  si otro tutor o la guardería lo necesita después).
- Estas acciones las ejecuta hoy el **staff/admin de la guardería** desde
  el panel, a petición del tutor -- no hay todavía un formulario de
  autoservicio para que el tutor lo pida directamente sin pasar por la
  guardería. El contrato/aviso debe explicar este flujo (a quién le pide
  el tutor sus derechos ARCO: a la guardería, que es el Responsable).
- **Rectificación**: el staff puede corregir el nombre de un tutor o de un
  niño desde el panel.
- **Oposición**: no hay un mecanismo dedicado distinto de la cancelación
  hoy -- el abogado debe confirmar si esto es aceptable o si se necesita
  algo más granular (ej. oponerse solo a las notificaciones, que sí es
  autoservicio: el propio tutor puede activar/desactivar sus
  notificaciones push desde el portal).

## 6. Qué NO se guarda / minimización de datos ya aplicada

Útil para la cláusula de "principio de minimización":

- El contenido de los mensajes de chat **no se incluye** en las
  notificaciones push que llegan al teléfono (solo dice "tienes un mensaje
  nuevo") -- para que el contenido de una conversación privada no quede
  expuesto en la pantalla de bloqueo del celular.
- Los logs de error del servidor no registran el contenido de mensajes de
  chat, contraseñas, ni el payload de las notificaciones push.
- El rostro del niño nunca se captura ni se procesa -- solo el del adulto
  tutor.

## 7. Cómo funciona hoy el consentimiento (para que el contrato lo describa bien)

1. Cada guardería redacta su propio Aviso de Privacidad (texto libre o un
   PDF que suba) desde su panel de Configuración -- Pasitos no le da una
   plantilla todavía (ver punto 10, sección de "pendientes").
2. Antes de enrolar el rostro de un tutor nuevo, el sistema **obliga** a
   mostrarle el Aviso de Privacidad vigente y a que el staff confirme "el
   tutor presente lo leyó y aceptó" -- sin esto, el sistema no deja
   continuar con el registro biométrico.
3. Cada aceptación queda registrada con: el nombre del tutor en ese
   momento, la versión exacta del aviso que aceptó, y la IP desde la que
   se hizo -- esto es la evidencia de consentimiento que la LFPDPPP exige
   poder demostrar.
4. El aviso tiene control de versiones: si la guardería lo cambia, la
   versión sube y las aceptaciones anteriores quedan ligadas a la versión
   vieja que aceptaron, no a la nueva.

## 8. Términos de servicio -- puntos de negocio que el abogado necesita que el equipo decida

Esto no es información técnica -- son decisiones de negocio que hoy no
están resueltas y que el contrato necesita:

- **[ ⚠️ COMPLETAR ]** Razón social / nombre legal de la empresa que
  presta el servicio, RFC, domicilio fiscal.
- **[ ⚠️ COMPLETAR ]** Modelo de precios una vez termine el piloto
  gratuito (mensual por niño, por guardería, plan único, etc.) y
  condiciones de pago/facturación.
- **[ ⚠️ COMPLETAR ]** Política de cancelación: si una guardería deja de
  pagar o quiere darse de baja, ¿qué pasa con sus datos? ¿Cuánto tiempo se
  conservan antes de borrarse definitivamente? ¿Puede exportar todo antes
  de irse?
- **[ ⚠️ COMPLETAR ]** Nivel de servicio (SLA): hoy no hay ningún
  compromiso de disponibilidad formal. Vale la pena decidir si el
  contrato promete algo (ej. "mejor esfuerzo, sin garantía de
  disponibilidad") en vez de dejarlo implícito.
- **[ ⚠️ COMPLETAR ]** Límite de responsabilidad de Pasitos ante una falla
  del sistema (ej. si el reconocimiento facial falla y un niño sale con
  la persona equivocada por un error del sistema -- el negocio necesita
  decidir qué tanto se puede/quiere limitar esa responsabilidad, dado que
  hay menores de por medio).
- **[ ⚠️ COMPLETAR ]** Qué pasa con los datos si Pasitos como empresa deja
  de operar (cláusula de continuidad/transición de datos a otro proveedor
  o a la guardería directamente).
- **[ ⚠️ COMPLETAR ]** Jurisdicción y ley aplicable para disputas (se
  asume México dado que la ley de datos personales que aplica es la
  LFPDPPP, pero debe quedar explícito).

## 9. Resumen para el abogado en una frase

Pasitos es un **Encargado** que procesa, en nombre de cada guardería,
datos personales de tutores, niños y personal -- incluyendo una categoría
de **dato sensible** (biometría facial de adultos) y datos de **menores de
edad**, con las medidas de seguridad y los mecanismos de consentimiento
descritos arriba, alojados en proveedores de nube (AWS, y un proveedor de
base de datos y de hosting por confirmar) que deben integrarse como
sub-encargados en el contrato -- y hace falta resolver las decisiones de
negocio de la sección 8 antes de poder cerrar el documento final.
