# TÉRMINOS Y CONDICIONES DE SERVICIO DE PASITOS

Fecha de versión: 30/08/2026 | Versión: 1.0

> Este documento es el borrador entregado por el representante legal de
> Pasitos, guardado aquí tal cual para que quede versionado junto con el
> código. La misma redacción se publica en la app en `/terminos` (ver
> `GuarderiaBiometricFront/src/TerminosCondiciones.jsx`) -- si este archivo
> cambia, esa página debe actualizarse también. Ver la nota de revisión
> jurídica al final.

## 1. Objeto y aceptación

Los presentes Términos y Condiciones de Servicio (los "Términos") regulan el acceso y uso de Pasitos, una plataforma de software como servicio (SaaS) destinada a guarderías, estancias infantiles y centros de cuidado infantil (el "Cliente").

Pasitos permite, según los módulos contratados, administrar expedientes, asistencia, entradas y salidas, reconocimiento facial de tutores, bitácoras diarias, comunicación entre familias y personal, notificaciones, circulares, encuestas, calendario, pedidos de comedor, pagos y administración del personal.

Al contratar, activar una cuenta, aceptar electrónicamente estos Términos o utilizar la Plataforma, el Cliente manifiesta que cuenta con facultades suficientes para obligarse y acepta estos Términos.

## 2. Definiciones

"Plataforma" significa el software Pasitos, sus interfaces, paneles administrativos, módulos, APIs y servicios relacionados.

"Cliente" significa la guardería, estancia infantil o centro de cuidado que contrata Pasitos.

"Usuario" significa cualquier persona autorizada por el Cliente para acceder a la Plataforma, incluyendo administradores, personal y tutores/familias cuando corresponda.

"Datos del Cliente" significa la información, archivos, fotografías, documentos, mensajes, registros, expedientes y demás información que el Cliente o sus Usuarios introduzcan o generen en la Plataforma.

"Datos Personales", "Responsable" y "Encargado" tendrán el significado previsto en la legislación mexicana aplicable.

## 3. Naturaleza del servicio

Pasitos proporciona una herramienta tecnológica para facilitar la administración y comunicación del Cliente. Pasitos no opera la guardería, no presta servicios de cuidado infantil y no sustituye al personal responsable de supervisar, identificar, autorizar o entregar a una niña o niño.

Las funciones de reconocimiento facial, asistencia y control de entradas y salidas son herramientas de apoyo y no deberán considerarse, por sí mismas, un mecanismo infalible de identificación o autorización.

El Cliente deberá mantener procedimientos internos de seguridad, supervisión humana y verificación alternativa cuando sea necesario.

## 4. Cuenta del Cliente y Usuarios

El Cliente será responsable de proporcionar información verdadera y actualizada, mantener la confidencialidad de sus credenciales, administrar permisos, retirar accesos no autorizados y utilizar la Plataforma conforme a la ley.

Las cuentas de personal podrán contar con permisos por área y, para determinadas funciones, con un PIN adicional de seguridad.

El Cliente será responsable de las acciones realizadas desde las cuentas que haya autorizado, salvo que exista evidencia de un acceso no autorizado imputable a Pasitos.

## 5. Reconocimiento facial y control de entradas y salidas

Cuando el Cliente habilite el reconocimiento facial, podrá utilizarse una plantilla facial del adulto/tutor autorizado para facilitar su identificación al recoger a una niña o niño.

Pasitos no procesa el rostro de las niñas o niños para el reconocimiento facial. La funcionalidad está diseñada para el enrolamiento del adulto/tutor autorizado.

La plantilla facial se genera mediante AWS Rekognition. Debido a la naturaleza sensible de los datos biométricos, el Cliente deberá obtener y conservar los consentimientos y autorizaciones legalmente necesarios y proporcionar el aviso de privacidad aplicable.

El Cliente deberá contar con un procedimiento alternativo cuando el reconocimiento falle, el tutor no esté enrolado, exista discrepancia en la autorización, haya una emergencia o la Plataforma no esté disponible.

El reconocimiento facial no sustituye la obligación del Cliente de verificar que la persona que recoge al menor esté autorizada.

## 6. Responsabilidades del Cliente respecto de niñas, niños y familias

El Cliente determina qué datos solicita, las finalidades del tratamiento, las personas autorizadas para recoger a cada niña o niño, los permisos del personal, los documentos de inscripción, la información de bitácoras y el contenido y vigencia de su Aviso de Privacidad.

El Cliente deberá evitar introducir en campos libres información que no sea necesaria y deberá establecer controles internos para información de salud, alergias, medicamentos u otra información especialmente sensible.

## 7. Protección de datos personales

Para los datos personales que el Cliente introduzca o gestione mediante Pasitos, las Partes reconocen, en principio, la siguiente distribución: el Cliente actúa como Responsable, pues determina las finalidades y decisiones esenciales del tratamiento; Pasitos actúa como Encargado, pues trata los datos por cuenta del Cliente y conforme a sus instrucciones.

Esta distribución deberá documentarse y complementarse mediante un Acuerdo de Encargo de Tratamiento de Datos.

Pasitos no utilizará los Datos Personales del Cliente para fines propios incompatibles con las instrucciones del Cliente ni venderá los datos personales de las familias del Cliente.

El Cliente seguirá siendo responsable de contar con una base jurídica válida, avisos de privacidad, consentimientos y demás autorizaciones necesarias para el tratamiento que decida realizar.

## 8. Aviso de Privacidad del Cliente

Cada guardería deberá proporcionar su propio Aviso de Privacidad. La Plataforma permite cargarlo, publicarlo y mantener versiones.

Antes de enrolar el rostro de un tutor nuevo, la Plataforma exige mostrar el Aviso de Privacidad vigente y registrar la aceptación correspondiente. La Plataforma registra, según la funcionalidad implementada, la versión aceptada, el nombre del tutor y la IP asociada.

Pasitos proporciona la funcionalidad tecnológica, pero no garantiza que el Aviso de Privacidad redactado por el Cliente cumpla por sí mismo con todas las obligaciones legales aplicables.

## 9. Derechos ARCO

Las solicitudes relacionadas con los datos personales tratados por cuenta del Cliente deberán dirigirse al Cliente, en su calidad de Responsable, mediante los mecanismos indicados en su Aviso de Privacidad.

La Plataforma cuenta con funciones para que el personal autorizado gestione solicitudes de acceso, rectificación y cancelación conforme a las instrucciones del Cliente.

El Cliente será responsable de resolver jurídicamente las solicitudes ARCO y de instruir a Pasitos sobre las acciones que deban ejecutarse.

## 10. Subencargados y proveedores tecnológicos

Para prestar el servicio, Pasitos podrá utilizar proveedores tecnológicos y subencargados, incluyendo AWS/Rekognition, AWS S3, Render, un proveedor de PostgreSQL, Stripe cuando se habiliten pagos y Web Push/VAPID.

Región AWS Rekognition: US-EAST-1.

Proveedor y región de PostgreSQL: Render Oregon (US West).

Región de Render: Oregon (US West).

Cuando un proveedor implique una transferencia internacional de datos o comunicación a un tercero, dicha operación deberá documentarse y comunicarse conforme a la legislación aplicable y al Aviso de Privacidad correspondiente.

## 11. Seguridad de la información

Pasitos implementa medidas técnicas y organizativas destinadas a reducir riesgos, incluyendo: contraseñas protegidas mediante bcrypt; cookies httpOnly y JWT; protección CSRF; PIN adicional; permisos por área; aislamiento entre Clientes; almacenamiento privado; URLs firmadas para determinados archivos; rate limiting; bitácoras de auditoría; HTTPS/TLS; y controles de conexiones a la base de datos.

Estas medidas pueden evolucionar conforme al desarrollo de la Plataforma.

Pasitos no declara contar con certificaciones ISO 27001, SOC 2 u otras certificaciones de seguridad de terceros salvo que se establezca expresamente por escrito.

Ningún sistema conectado a Internet puede garantizar seguridad absoluta.

## 12. Incidentes y vulneraciones

Si Pasitos detecta una vulneración de seguridad que afecte datos personales tratados por cuenta del Cliente, notificará al Cliente sin dilación indebida y proporcionará, en la medida disponible y legalmente permitida, información razonable para que el Cliente pueda cumplir sus obligaciones.

El Cliente será responsable de determinar las comunicaciones que deban realizarse a titulares o autoridades, salvo que la legislación o el acuerdo entre las Partes establezca otra distribución.

## 13. Disponibilidad del servicio

Pasitos tiene como objetivo mantener la Plataforma disponible de manera continua, las veinticuatro (24) horas del día, los siete (7) días de la semana, salvo los periodos de mantenimiento programado, actualizaciones, interrupciones ocasionadas por terceros o eventos fuera del control razonable de Pasitos.

Los mantenimientos y actualizaciones programados que puedan afectar la disponibilidad de la Plataforma se procurarán realizar preferentemente durante horarios nocturnos, con el objetivo de minimizar cualquier afectación a las operaciones del Cliente. Cuando resulte razonablemente posible, Pasitos notificará previamente a las guarderías sobre dichos mantenimientos mediante los medios de comunicación disponibles en la Plataforma.

En caso de presentarse una falla, interrupción, incidente de seguridad o cualquier otra circunstancia que afecte inesperadamente la disponibilidad de la Plataforma, Pasitos realizará esfuerzos razonables para detectar, atender y restablecer el servicio en el menor tiempo razonablemente posible, de acuerdo con la naturaleza y gravedad del incidente.

Las interrupciones ocasionadas por fallas de proveedores de infraestructura, servicios de Internet, servicios de terceros, acontecimientos de fuerza mayor, ataques informáticos, actos de autoridad u otras circunstancias que se encuentren fuera del control razonable de Pasitos no se considerarán incumplimientos imputables a Pasitos, sin perjuicio de las obligaciones de atención y recuperación que correspondan.

La disponibilidad de la Plataforma no constituye una garantía de funcionamiento ininterrumpido o libre de errores, y el Cliente reconoce que ningún servicio tecnológico conectado a Internet puede garantizar disponibilidad absoluta.

En caso de que Pasitos establezca posteriormente un Acuerdo de Nivel de Servicio (SLA) con porcentajes específicos de disponibilidad, tiempos de respuesta o tiempos de recuperación, dicho SLA prevalecerá sobre esta cláusula en lo que expresamente regule.

## 14. Mantenimiento y modificaciones

Pasitos podrá realizar mantenimiento preventivo, correctivo y evolutivo y podrá modificar, agregar o retirar funcionalidades cuando sea necesario por razones legales, de seguridad, interoperabilidad, proveedores o evolución del servicio, sujeto a los derechos contractuales del Cliente y a la legislación aplicable.

## 15. Propiedad intelectual

La Plataforma, código fuente, arquitectura, interfaces, diseño, marcas, logotipos, documentación, algoritmos, procesos y demás elementos desarrollados por Pasitos son propiedad de Pasitos o de sus licenciantes.

El Cliente recibe únicamente una licencia limitada, no exclusiva y no transferible para utilizar la Plataforma durante la vigencia del servicio.

El Cliente no podrá copiar, modificar, descompilar, desensamblar o realizar ingeniería inversa, salvo cuando una disposición legal imperativa lo permita; sublicenciar o revender sin autorización; eliminar avisos de propiedad intelectual; extraer masivamente información; ni acceder a componentes no públicos.

Los Datos del Cliente seguirán perteneciendo al Cliente o a sus respectivos titulares, según corresponda. El uso de la Plataforma no transfiere a Pasitos la propiedad de dichos datos.

## 16. Contenido proporcionado por el Cliente

El Cliente conserva la responsabilidad sobre fotografías, documentos, mensajes, expedientes, textos, nombres, logotipos y demás contenido que cargue. El Cliente declara contar con las facultades y autorizaciones necesarias y autoriza a Pasitos a almacenarlo, procesarlo y mostrarlo exclusivamente en la medida necesaria para prestar el servicio.

## 17. Uso aceptable

El Cliente y sus Usuarios no podrán utilizar la Plataforma para actividades ilícitas; vulnerar derechos de niñas, niños, familias, trabajadores o terceros; introducir malware; acceder a cuentas o datos de otros Clientes; realizar pruebas de penetración sin autorización; sobrecargar deliberadamente la infraestructura; extraer masivamente información mediante mecanismos no autorizados; suplantar identidades; o utilizar la Plataforma para fines distintos de los contratados.

Pasitos podrá suspender temporalmente una cuenta cuando exista una amenaza razonable para la seguridad de la Plataforma o de terceros.

## 18. Suspensión y terminación

El Cliente podrá cancelar conforme a la política contratada. Pasitos podrá suspender o terminar el servicio ante incumplimiento grave, falta de pago, uso ilícito, riesgo de seguridad u otra causa prevista contractualmente.

Periodo de conservación después de la terminación: 90 días.

Periodo para exportar datos: 30 días.

Antes de la eliminación definitiva, Pasitos deberá proporcionar al Cliente, cuando corresponda, un mecanismo razonable para exportar sus Datos del Cliente.

Al concluir el periodo aplicable, Pasitos eliminará o devolverá los datos conforme al Acuerdo de Encargo, salvo aquellos que deban conservarse por obligación legal o para atender responsabilidades.

## 19. Continuidad del servicio y cierre de operaciones

Si Pasitos deja de operar, se buscará implementar un proceso razonable de transición para que el Cliente pueda obtener sus Datos del Cliente y migrarlos a otro proveedor.

Procedimiento de continuidad y transición: en caso de que Pasitos deje de prestar de manera definitiva los servicios de la Plataforma, Pasitos procurará notificar al Cliente con una anticipación razonable, cuando las circunstancias lo permitan. El Cliente contará con un periodo de 30 días naturales para realizar la exportación de sus Datos del Cliente mediante los mecanismos disponibles en la Plataforma o mediante un mecanismo de exportación proporcionado por Pasitos.

La exportación podrá incluir, en la medida en que técnicamente se encuentre disponible, información de expedientes, tutores, niñas y niños, asistencia, bitácoras, fotografías, documentos, mensajes y demás información almacenada por cuenta del Cliente.

Pasitos procurará proporcionar los datos en formatos estructurados y de uso común que permitan su migración a otro sistema. Una vez concluido el periodo de transición, los datos serán tratados conforme a las disposiciones de conservación y eliminación establecidas en estos Términos y en el Acuerdo de Encargo de Tratamiento de Datos, salvo aquellos que deban conservarse por obligación legal.

## 20. Limitación de responsabilidad

Pasitos será responsable únicamente por los daños que legalmente sean imputables a su actuación y dentro de los límites válidos conforme a la legislación aplicable.

Límite contractual de responsabilidad: la responsabilidad total y acumulada de Pasitos derivada de o relacionada con la prestación de los servicios, independientemente de la forma de la acción (ya sea por responsabilidad contractual, extracontractual, negligencia o cualquier otra), se limitará a la cantidad equivalente al total de las contraprestaciones efectivamente pagadas por el Cliente a Pasitos durante los tres (3) meses previos a la ocurrencia del evento que dio origen a la reclamación (o bien, en caso de encontrarse en periodo de prueba o piloto gratuito, a un monto máximo de $5,000.00 MXN). Cualquier limitación deberá excluir los supuestos en que legalmente no sea posible limitar o excluir responsabilidad.

En ningún caso deberá interpretarse que Pasitos garantiza que el reconocimiento facial será perfecto o que la Plataforma por sí sola impedirá una salida no autorizada de un menor. El Cliente mantiene la responsabilidad de contar con procedimientos humanos de verificación y seguridad.

## 21. Fuerza mayor

Ninguna Parte será responsable por incumplimientos derivados de acontecimientos fuera de su control razonable, incluyendo desastres naturales, fallas generalizadas de telecomunicaciones, actos de autoridad, conflictos laborales, ataques cibernéticos generalizados, interrupciones de proveedores esenciales o hechos equivalentes, sin perjuicio de obligaciones de mitigación y continuidad.

## 22. Confidencialidad

Las Partes deberán mantener confidencial la información no pública que reciban con motivo de la relación contractual. Las obligaciones de confidencialidad sobrevivirán a la terminación cuando por su naturaleza deban permanecer vigentes.

La confidencialidad respecto de datos personales se mantendrá durante y después de la relación contractual conforme a la legislación aplicable y al Acuerdo de Encargo.

## 23. Modificaciones a los Términos

Pasitos podrá actualizar estos Términos por cambios en la Plataforma, legislación, seguridad, proveedores o modelo comercial. Las modificaciones sustanciales deberán comunicarse al Cliente por medios razonables y, cuando sea jurídicamente necesario, requerirán nueva aceptación.

## 24. Cesión

El Cliente no podrá ceder sus derechos u obligaciones sin autorización previa de Pasitos, salvo los casos permitidos por ley.

Pasitos podrá ceder o reorganizar sus derechos y obligaciones como parte de una operación corporativa, fusión, adquisición o reorganización, manteniendo las obligaciones legales aplicables respecto de los datos personales.

## 25. Relación entre las Partes

Estos Términos no crean una relación laboral, sociedad, asociación, mandato, agencia o representación entre Pasitos y el Cliente. Cada Parte será responsable de su personal, obligaciones fiscales, laborales y administrativas.

## 26. Ley aplicable y jurisdicción

Estos Términos se regirán por las leyes de los Estados Unidos Mexicanos.

Estado y ciudad para jurisdicción: Culiacán, Sinaloa.

Las Partes procurarán resolver de buena fe cualquier controversia antes de acudir a tribunales, sin perjuicio de disposiciones imperativas aplicables.

## 27. Documentos integrantes del contrato

Formarán parte integral e inseparable de la relación contractual entre Pasitos y el Cliente, cuando correspondan y según se encuentren vigentes:

- Los presentes Términos y Condiciones de Servicio.
- El Acuerdo de Encargo de Tratamiento de Datos Personales (DPA).
- La orden de servicio, cotización, propuesta comercial o contrato particular aceptado por el Cliente.
- El Acuerdo de Nivel de Servicio (SLA), en caso de haber sido formalizado expresamente por escrito.
- Los anexos técnicos y de proveedores/subencargados (Anexo A y Anexo B).
- Las demás políticas, avisos o anexos técnicos expresamente incorporados.

Orden de prelación: en caso de discrepancia, contradicción o conflicto entre las disposiciones contenidas en los documentos contractuales anteriores, prevalecerá el siguiente orden de jerarquía:

1. El Acuerdo de Encargo de Tratamiento de Datos Personales (DPA) (exclusivamente en lo referente a la protección y tratamiento de datos personales y biométricos).
2. La orden de servicio, cotización o contrato comercial particular (en lo referente a precios, vigencia, volumen de niños/usuarios y condiciones particulares de pago).
3. Los presentes Términos y Condiciones de Servicio.
4. El Acuerdo de Nivel de Servicio (SLA).
5. Los Anexos técnicos y demás políticas incorporadas.

## 28. Integridad contractual

Estos Términos y los documentos que expresamente formen parte del contrato constituyen el acuerdo entre las Partes respecto del servicio Pasitos. Si alguna disposición fuese declarada inválida, ilegal o inaplicable, las demás permanecerán vigentes en la medida permitida por la ley.

## 29. Datos de contacto

- Proveedor: Alejandro Bueno Mendoza
- RFC: BUMA990820HQ4
- Domicilio: Monte de Piedad Departamento 4356, Fraccionamiento, Valle de Encino, 80197 Culiacán Rosales, Sin.
- Correo contractual: alejandrobuenomendoza@gmail.com
- Correo de privacidad: alejandrobuenomendoza@gmail.com
- Teléfono: 6674983913
- Sitio web: https://pasitos-frontend.onrender.com/

---

## ANEXO A — DATOS Y TRATAMIENTO

### A.1 Categorías principales

Pasitos puede tratar, por cuenta del Cliente, información de tutores/padres, niñas y niños, personal y usuarios administrativos. Puede incluir nombres, credenciales, teléfonos, direcciones, mensajes, pagos, asistencia, fotografías, documentos, registros de bitácora y metadatos técnicos.

La Plataforma puede recibir información que, por el contenido introducido por el Cliente, revele datos de salud, alergias o medicamentos. El Cliente deberá limitar dicha información a lo necesario y aplicar las medidas legales correspondientes.

### A.2 Datos biométricos

Cuando se habilite el reconocimiento facial, se procesa una plantilla facial del adulto/tutor. El rostro de la niña o niño no se procesa para esta finalidad.

### A.3 Infraestructura

| Proveedor | Función | Información | Pendiente |
|---|---|---|---|
| AWS Rekognition | Reconocimiento facial | Plantillas faciales | Región: us-east-1 |
| AWS S3 | Almacenamiento | Fotos/documentos | Región/configuración: us-east-1 |
| PostgreSQL / proveedor | Base de datos | Expedientes, mensajes, pagos, asistencia, cuentas | Proveedor/región/backups: Oregon (US West) |
| Render | Hosting | Backend/frontend | Región: Oregon (US West) |
| Web Push / VAPID | Notificaciones | Suscripciones push | Sin proveedor externo de notificaciones |

### A.4 Medidas técnicas actualmente implementadas

- bcrypt para contraseñas
- Cookies httpOnly y JWT
- CSRF
- PIN administrativo con expiración
- Permisos por área
- Aislamiento multi-tenant
- Almacenamiento privado
- URLs firmadas de corta duración
- Rate limiting
- Auditoría
- HTTPS/TLS

## ANEXO B — DECISIONES PENDIENTES

- Razón social, RFC y domicilio
- Correos y teléfono legales
- Modelo de precios y facturación
- Política de cancelación
- Plazo de exportación de datos
- Plazo de conservación posterior a terminación
- SLA o esquema de mejores esfuerzos
- Límite de responsabilidad
- Procedimiento de continuidad si Pasitos deja de operar
- Jurisdicción
- Regiones de AWS, PostgreSQL y Render
- Backups y cifrado en reposo
- Procedimiento formal de respuesta a incidentes
- Orden de prelación contractual
- Validación específica de biometría y datos de menores

## NOTA DE REVISIÓN JURÍDICA

Este borrador se estructuró con base en el briefing técnico de Pasitos y en la legislación mexicana vigente consultada para esta versión. La Ley Federal de Protección de Datos Personales en Posesión de los Particulares vigente fue publicada como nueva ley el 20 de marzo de 2025 y registra última reforma al 14 de noviembre de 2025.

**Antes de uso comercial, un abogado mexicano deberá validar especialmente** la figura Responsable/Encargado, biometría, datos de niñas, niños y adolescentes, consentimiento, transferencias internacionales, subencargados, conservación y eliminación, derechos ARCO, vulneraciones de seguridad, propiedad intelectual, límites de responsabilidad, jurisdicción y contratación electrónica.
