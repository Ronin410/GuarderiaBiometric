import React from 'react';
import { Link } from 'react-router-dom';
import { ShieldCheck, ArrowLeft, Mail, Phone, MapPin } from 'lucide-react';

// AvisoPrivacidadPasitos -- "¿se necesitará cambiar o agregar algo a los
// términos y condiciones?" por el chat de soporte: los papás/staff con
// cuenta ya están cubiertos por el Aviso de Privacidad de SU guardería
// para lo que pasa DENTRO de la plataforma, pero un prospecto que escribe
// por el chat de soporte sin cuenta no tiene ningún aviso que lo ampare --
// este es ESE aviso, propio de Pasitos, distinto del de cada guardería
// (ver la cláusula 9 Bis en TerminosCondiciones.jsx / TERMINOS_Y_
// CONDICIONES.md). Misma redacción que AVISO_PRIVACIDAD_PASITOS.md en la
// raíz del repo -- si el texto cambia ahí, debe actualizarse aquí también.
const AvisoPrivacidadPasitos = () => (
  <div className="min-h-screen bg-paper text-ink">
    <header className="sticky top-0 z-30 bg-paper/90 backdrop-blur-sm border-b border-slate-200">
      <div className="max-w-3xl mx-auto px-5 sm:px-8 h-16 flex items-center justify-between">
        <Link to="/" className="flex items-center gap-2.5">
          <div className="bg-brand-600 w-8 h-8 rounded-xl flex items-center justify-center shrink-0">
            <ShieldCheck size={17} className="text-white" />
          </div>
          <span className="font-black uppercase text-sm tracking-tight">Pasitos</span>
        </Link>
        <Link to="/" className="flex items-center gap-1.5 text-slate-400 hover:text-brand-600 font-bold text-xs uppercase tracking-widest transition-colors">
          <ArrowLeft size={14} /> Inicio
        </Link>
      </div>
    </header>

    <main className="max-w-3xl mx-auto px-5 sm:px-8 py-12">
      <p className="text-[11px] font-black uppercase text-brand-600 tracking-widest mb-2">Documento legal</p>
      <h1 className="text-3xl sm:text-4xl font-black tracking-tight mb-3" style={{ fontFamily: 'Lora, serif' }}>
        Aviso de Privacidad de Pasitos
      </h1>
      <p className="text-xs text-slate-400 font-bold mb-2">Para prospectos y usuarios del chat de soporte</p>
      <p className="text-xs text-slate-400 font-bold mb-6">Fecha de versión: 30/08/2026 &nbsp;|&nbsp; Versión: 1.0</p>

      <div className="bg-amber-50 border border-amber-200 rounded-2xl p-5 mb-10">
        <p className="text-sm text-amber-900 leading-relaxed">
          <strong>Este aviso es distinto del Aviso de Privacidad de cada guardería</strong> (el que cada guardería sube y publica en su propia cuenta de Pasitos, y que se le muestra a los tutores al enrolar su rostro).
          Ese otro aviso sigue rigiendo los datos que una guardería trata de sus propias familias <em>dentro</em> de la plataforma. Este documento rige únicamente la comunicación <strong>directa</strong> con Pasitos, por ejemplo a través del chat de soporte.
        </p>
      </div>

      <section className="mb-8">
        <h2 className="text-lg font-black text-slate-900 mb-3">1. Responsable del tratamiento</h2>
        <p className="text-sm text-slate-600 leading-relaxed"><strong>Alejandro Bueno Mendoza</strong>, operando bajo el nombre comercial "Pasitos".</p>
      </section>

      <section className="mb-8">
        <h2 className="text-lg font-black text-slate-900 mb-3">2. ¿A quién aplica este aviso?</h2>
        <p className="text-sm text-slate-600 leading-relaxed">A cualquier persona que se comunique directamente con Pasitos, fuera de la relación entre tu guardería y Pasitos, por ejemplo:</p>
        <ul className="mt-3 space-y-1.5 list-disc list-inside">
          <li className="text-sm text-slate-600 leading-relaxed"><strong>Un prospecto:</strong> alguien interesado en Pasitos que todavía no tiene cuenta ni relación con ninguna guardería (por ejemplo, quien escribe por el chat de soporte de la página de presentación, o llena el formulario público de alta de guardería).</li>
          <li className="text-sm text-slate-600 leading-relaxed"><strong>Un papá/tutor o miembro del staff/admin</strong> de una guardería que ya usa Pasitos, cuando le escribe directo a Pasitos por el chat de soporte, en vez de comunicarse con su guardería.</li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="text-lg font-black text-slate-900 mb-3">3. Datos personales que recabamos</h2>
        <ul className="space-y-1.5 list-disc list-inside">
          <li className="text-sm text-slate-600 leading-relaxed"><strong>Nombre</strong> (el que nos proporcionas al escribirnos).</li>
          <li className="text-sm text-slate-600 leading-relaxed"><strong>Correo electrónico</strong>, cuando lo proporcionas (siempre en el caso de un prospecto que llena el formulario inicial del chat de soporte).</li>
          <li className="text-sm text-slate-600 leading-relaxed"><strong>El contenido de tus mensajes</strong> en la conversación de soporte.</li>
          <li className="text-sm text-slate-600 leading-relaxed">Si ya tienes cuenta en una guardería, también asociamos tu mensaje con <strong>tu guardería y tu rol</strong>, para poder darte mejor seguimiento -- esto lo determina tu sesión automáticamente, no lo escribes tú.</li>
          <li className="text-sm text-slate-600 leading-relaxed">Un <strong>identificador técnico</strong> (token) que, si escribes como prospecto sin cuenta, se guarda en el almacenamiento de tu navegador para reconocer tu conversación en visitas futuras sin pedirte una cuenta. No es una cookie de rastreo publicitario ni se comparte con terceros de publicidad; solo identifica tu conversación en nuestro propio servidor.</li>
        </ul>
        <p className="text-sm text-slate-600 leading-relaxed mt-3">No te pedimos, y te pedimos evitar compartirnos por este medio, datos sensibles (salud, biométricos, etc.) -- este chat es para soporte y dudas comerciales, no para gestionar expedientes de niñas y niños.</p>
      </section>

      <section className="mb-8">
        <h2 className="text-lg font-black text-slate-900 mb-3">4. Finalidades del tratamiento</h2>
        <p className="text-sm text-slate-600 leading-relaxed font-bold">Finalidades primarias (necesarias para atenderte):</p>
        <ul className="mt-2 space-y-1.5 list-disc list-inside">
          <li className="text-sm text-slate-600 leading-relaxed">Responder tus dudas y darte soporte técnico o comercial.</li>
          <li className="text-sm text-slate-600 leading-relaxed">Dar seguimiento a solicitudes de alta de guardería nueva.</li>
          <li className="text-sm text-slate-600 leading-relaxed">Identificar y resolver problemas del servicio que reportes.</li>
        </ul>
        <p className="text-sm text-slate-600 leading-relaxed font-bold mt-4">Finalidades secundarias (puedes oponerte sin que afecte el soporte que recibas):</p>
        <ul className="mt-2 space-y-1.5 list-disc list-inside">
          <li className="text-sm text-slate-600 leading-relaxed">Contactarte con información sobre Pasitos que pueda interesarte.</li>
        </ul>
        <p className="text-sm text-slate-600 leading-relaxed mt-3">Si no deseas que tratemos tus datos para la finalidad secundaria, puedes decírnoslo por cualquier medio de contacto de este aviso; dejaremos de usarlos para ese fin sin que esto afecte el soporte primario.</p>
      </section>

      <section className="mb-8">
        <h2 className="text-lg font-black text-slate-900 mb-3">5. Transferencias y proveedores tecnológicos</h2>
        <p className="text-sm text-slate-600 leading-relaxed">
          No vendemos ni compartimos tus datos con terceros para fines de publicidad ajenos a Pasitos. Para operar el chat de soporte usamos los mismos proveedores tecnológicos que el resto de la plataforma: hosting (Render) y base de datos (PostgreSQL, alojada en Render, Oregon (US West)). Las notificaciones que recibimos cuando escribes usan el mismo mecanismo Web Push/VAPID que el resto de la app, sin un proveedor externo de notificaciones.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="text-lg font-black text-slate-900 mb-3">6. Conservación</h2>
        <p className="text-sm text-slate-600 leading-relaxed">
          Conservamos tu conversación mientras sea razonablemente necesaria para darte soporte y por el tiempo que la legislación aplicable exija conservar registros de comunicaciones comerciales, salvo que ejerzas tu derecho de cancelación conforme a la sección siguiente.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="text-lg font-black text-slate-900 mb-3">7. Derechos ARCO y cómo ejercerlos</h2>
        <p className="text-sm text-slate-600 leading-relaxed">
          Puedes acceder, rectificar, cancelar tus datos personales, u oponerte a su tratamiento (derechos ARCO), enviando tu solicitud al correo de privacidad abajo, identificándote y describiendo la solicitud. Atenderemos tu solicitud dentro de los plazos que marca la legislación mexicana aplicable.
        </p>
      </section>

      <section>
        <h2 className="text-lg font-black text-slate-900 mb-3">8. Cambios a este aviso</h2>
        <p className="text-sm text-slate-600 leading-relaxed">
          Podemos actualizar este aviso por cambios en la plataforma, la legislación aplicable, o nuestras prácticas de tratamiento de datos. La versión vigente siempre está disponible en esta misma página.
        </p>
      </section>
    </main>

    <footer className="max-w-3xl mx-auto px-5 sm:px-8 py-9 border-t border-slate-200 space-y-2">
      <p className="text-xs font-bold text-slate-500 flex items-center gap-2"><Mail size={13} className="text-brand-500 shrink-0" /> alejandrobuenomendoza@gmail.com</p>
      <p className="text-xs font-bold text-slate-500 flex items-center gap-2"><Phone size={13} className="text-brand-500 shrink-0" /> 667 498 3913</p>
      <p className="text-xs font-bold text-slate-500 flex items-center gap-2"><MapPin size={13} className="text-brand-500 shrink-0" /> Culiacán Rosales, Sinaloa, México</p>
    </footer>
  </div>
);

export default AvisoPrivacidadPasitos;
