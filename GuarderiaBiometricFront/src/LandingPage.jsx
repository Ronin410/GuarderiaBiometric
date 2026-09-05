import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  ShieldCheck, ArrowUpRight, Check, ScanEye, ClipboardList, BellRing,
  MessageCircle, Megaphone, ClipboardCheck, CalendarDays, CalendarOff,
  UtensilsCrossed, Wallet, UserCog, BarChart3, Lock, Clock3,
} from 'lucide-react';
import SoporteChat from './SoporteChat';

// LandingPage -- "pon como main page la página de presentación, me gustó,
// ponla como mi página principal". Antes "/" mandaba directo al login;
// ahora "/" muestra esto (el mismo contenido que se armó como presentación,
// ya adaptado a componentes reales de la app en vez de vivir solo como un
// artifact aparte) y el login/kiosco vive en /panel/identificar, alcanzable
// desde el botón "Iniciar sesión" de aquí.
//
// El CTA principal manda a /registro-guarderia -- el formulario público de
// alta que YA existe (solicitudes.go/RegistroGuarderia.jsx), no a un correo:
// así una guardería interesada puede pedir acceso de verdad sin salir de la
// app, en vez de depender de que alguien conteste un mailto.
const FUNCIONES = [
  {
    key: 'dia',
    tono: 'verde',
    label: 'Día a día',
    items: [
      { Icon: ScanEye, titulo: 'Reconocimiento facial', texto: 'Entradas y salidas del tutor en segundos, sin listas en papel ni tarjetas que se pierden.' },
      { Icon: ClipboardList, titulo: 'Bitácora digital', texto: 'Comidas, siestas, actividades y notas del día, visibles para la familia al instante.' },
      { Icon: BellRing, titulo: 'Notificaciones push', texto: 'Aviso automático de cada entrada, salida y actualización de bitácora, sin que nadie escriba nada.' },
    ],
  },
  {
    key: 'comunicacion',
    tono: 'naranja',
    label: 'Comunicación',
    items: [
      { Icon: MessageCircle, titulo: 'Chat directo', texto: 'Cada familia habla directo con la maestra o el administrador que le corresponde, sin grupos de WhatsApp.' },
      { Icon: Megaphone, titulo: 'Circulares', texto: 'Avisos oficiales de la guardería para todas las familias, con acuse de recibido.' },
      { Icon: ClipboardCheck, titulo: 'Encuestas', texto: 'Recoge la opinión de las familias sobre el servicio, directo desde la plataforma.' },
    ],
  },
  {
    key: 'organizacion',
    tono: 'morado',
    label: 'Organización',
    items: [
      { Icon: CalendarDays, titulo: 'Calendario escolar', texto: 'Eventos, días festivos y actividades especiales, siempre visibles para las familias.' },
      { Icon: CalendarOff, titulo: 'Ausencias avisadas', texto: 'Los papás avisan con anticipación si su hijo no asistirá, y el staff lo ve al momento.' },
      { Icon: UtensilsCrossed, titulo: 'Menú y comedor', texto: 'Menú semanal y pedidos especiales de comedor por niño, sin notas sueltas en la mochila.' },
    ],
  },
  {
    key: 'administracion',
    tono: 'amarillo',
    label: 'Administración',
    items: [
      { Icon: Wallet, titulo: 'Pagos y colegiaturas', texto: 'Estado de cuenta claro para cada familia y control de pagos para la dirección.' },
      { Icon: UserCog, titulo: 'Personal y horarios', texto: 'Turnos, horarios y permisos de acceso por área para cada miembro del equipo.' },
      { Icon: BarChart3, titulo: 'Reportes y estadísticas', texto: 'Visibilidad para la dirección: asistencia, pagos y actividad del centro en un solo lugar.' },
    ],
  },
];

// Tonos de los dinos como lista ORDENADA: cada pestaña de funciones arranca
// en el suyo (`tono`) y las tres tarjetas van tomando los siguientes, así
// cada pestaña enseña tres colores a la vez en vez de tres tarjetas del
// mismo tinte. Las clases van escritas completas porque Tailwind v4 no
// puede generar nombres construidos por interpolación.
const TONOS_DINO = [
  { key: 'verde',    fondo: 'bg-dino-verde-suave',    borde: 'border-dino-verde/25',    texto: 'text-dino-verde',    dino: '/dinos/dino-verde.png' },
  { key: 'naranja',  fondo: 'bg-dino-naranja-suave',  borde: 'border-dino-naranja/25',  texto: 'text-dino-naranja',  dino: '/dinos/dino-naranja.png' },
  { key: 'morado',   fondo: 'bg-dino-morado-suave',   borde: 'border-dino-morado/25',   texto: 'text-dino-morado',   dino: '/dinos/dino-morado.png' },
  { key: 'amarillo', fondo: 'bg-dino-amarillo-suave', borde: 'border-dino-amarillo/30', texto: 'text-dino-amarillo', dino: '/dinos/dino-amarillo.png' },
];

const tonoDeTarjeta = (tonoInicial, i) => {
  const inicio = Math.max(0, TONOS_DINO.findIndex((t) => t.key === tonoInicial));
  return TONOS_DINO[(inicio + i) % TONOS_DINO.length];
};

// Cada beneficio toma el color de uno de los dinos de la marca y lo lleva a
// toda la tarjeta (fondo, borde y número), no solo al cuadrito del número:
// cuatro tarjetas blancas seguidas se veían planas junto al resto de la
// página, que ya es teal de arriba a abajo.
const BENEFICIOS = [
  { n: '01', fondo: 'bg-dino-verde-suave', borde: 'border-dino-verde/25', color: 'text-dino-verde', chip: 'bg-white', dino: '/dinos/dino-verde-saludo.png', titulo: 'Evita el ruido de los grupos', texto: 'La comunicación con cada familia pasa por un canal directo con el staff que le corresponde, no por un grupo donde se mezclan los avisos de todos los niños.' },
  { n: '02', fondo: 'bg-dino-naranja-suave', borde: 'border-dino-naranja/25', color: 'text-dino-naranja', chip: 'bg-white', dino: '/dinos/dino-naranja.png', titulo: 'Nada se pierde ni se olvida', texto: 'Bitácora, mensajes y avisos quedan guardados con fecha y hora -- no dependen del cuaderno de una sola maestra.' },
  { n: '03', fondo: 'bg-dino-morado-suave', borde: 'border-dino-morado/25', color: 'text-dino-morado', chip: 'bg-white', dino: '/dinos/dino-morado.png', titulo: 'Menos tiempo administrativo', texto: 'El equipo deja de repetir el mismo aviso por WhatsApp, de pasar lista a mano o de buscar una foto que alguien pidió hace días.' },
  { n: '04', fondo: 'bg-dino-amarillo-suave', borde: 'border-dino-amarillo/30', color: 'text-dino-amarillo', chip: 'bg-white', dino: '/dinos/dino-amarillo.png', titulo: 'El acceso, solo para quien debe verlo', texto: 'Cada cuenta del staff ve solo lo que le corresponde, y la dirección decide qué áreas puede tocar cada quien.' },
];

const LandingPage = () => {
  const [tabActiva, setTabActiva] = useState('dia');
  const funcionesActivas = FUNCIONES.find((f) => f.key === tabActiva) ?? FUNCIONES[0];

  return (
    <>
    <div className="min-h-screen bg-paper text-ink">
      {/* NAV */}
      <header className="sticky top-0 z-30 bg-paper/90 backdrop-blur-sm border-b border-slate-200">
        <div className="max-w-6xl mx-auto px-5 sm:px-8 h-[76px] flex items-center justify-between gap-4">
          <div className="flex items-center gap-2.5">
            <img src="/dinos/logo-pasitos.png" alt="" className="h-11 w-auto shrink-0" />
            <div className="leading-none">
              <span className="block font-black uppercase tracking-tight text-lg text-brand-900">Pasitos</span>
              <span className="hidden sm:block text-[9.5px] font-bold text-brand-600 mt-0.5">Gestión para guarderías</span>
            </div>
          </div>
          <div className="flex items-center gap-2 sm:gap-3">
            <Link
              to="/panel/identificar"
              className="text-[11px] sm:text-xs font-black uppercase tracking-widest px-3 sm:px-5 py-3 rounded-2xl text-ink/70 hover:text-ink hover:bg-white transition-all"
            >
              Iniciar sesión
            </Link>
            <Link
              to="/registro-guarderia"
              className="flex items-center gap-2 bg-forest text-white text-[11px] sm:text-xs font-black uppercase tracking-widest px-4 sm:px-5 py-3 rounded-2xl shadow-md hover:opacity-90 transition-all"
            >
              Solicita tu acceso <ArrowUpRight size={14} />
            </Link>
          </div>
        </div>
      </header>

      {/* HERO */}
      <section className="relative overflow-hidden">
        <div className="absolute -top-40 -right-40 w-[560px] h-[560px] rounded-full bg-brand-100/50 blur-3xl pointer-events-none" />
        <div className="max-w-6xl mx-auto px-5 sm:px-8 pt-14 sm:pt-20 pb-16 sm:pb-24 relative grid lg:grid-cols-[1.1fr_0.9fr] gap-12 items-center">
          <div>
            <p className="text-[11px] sm:text-xs font-black text-brand-600 uppercase tracking-[0.14em] mb-5">
              Gestión para guarderías y centros infantiles
            </p>
            <h1 className="font-black text-ink leading-[1.05] tracking-tight" style={{ fontFamily: "'Lora', Georgia, serif", fontSize: 'clamp(2.4rem, 5.4vw, 4.2rem)' }}>
              Tu guardería,<br />organizada en<br />un solo lugar.
            </h1>
            <p className="mt-6 text-base sm:text-lg text-slate-600 leading-relaxed max-w-md">
              Pasitos reemplaza el cuaderno, el grupo de WhatsApp y la lista de asistencia en papel por una plataforma con reconocimiento facial, bitácora digital y chat directo entre cada familia y el staff.
            </p>
            <div className="mt-9 flex flex-wrap items-center gap-3">
              <Link
                to="/registro-guarderia"
                className="flex items-center gap-2.5 bg-forest text-white text-xs font-black uppercase tracking-widest px-6 py-4 rounded-2xl shadow-lg hover:opacity-90 active:scale-95 transition-all"
              >
                Solicita tu acceso <ArrowUpRight size={16} />
              </Link>
              <a
                href="#funciones"
                className="bg-white border border-slate-200 text-ink text-xs font-black uppercase tracking-widest px-6 py-4 rounded-2xl hover:bg-slate-50 active:scale-95 transition-all"
              >
                Ver funciones
              </a>
            </div>
          </div>

          {/* La escena del hero: el dino que saluda al centro y sus compañeros
              de la familia asomando, sobre el mismo degradado suave de las
              tarjetas de funciones. Cada aviso flotante usa el dino de la
              sección que representa (verde=asistencia, morado=chat). */}
          <div className="relative h-[420px] hidden sm:block">
            <div className="absolute inset-6 rounded-[40px] bg-gradient-to-br from-brand-50 via-white to-dino-amarillo-suave border border-brand-100" />
            <img src="/dinos/dino-verde-saludo.png" alt="" className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 h-[240px] w-auto drop-shadow-sm" />
            <img src="/dinos/dino-amarillo.png" alt="" className="absolute left-[8%] bottom-[12%] h-[86px] w-auto" />
            <img src="/dinos/dino-naranja.png" alt="" className="absolute right-[6%] top-[10%] h-[84px] w-auto" />
            <div className="absolute top-[4%] -left-4 w-[240px] bg-white rounded-2xl shadow-xl p-3 flex items-center gap-3 -rotate-3">
              <img src="/dinos/dino-verde.png" alt="" className="h-10 w-auto shrink-0" />
              <div className="min-w-0">
                <p className="text-[13px] font-black text-ink truncate">Entrada registrada</p>
                <p className="text-[11px] text-slate-400 truncate">Ryan llegó · 8:02 am</p>
              </div>
            </div>
            <div className="absolute bottom-[4%] -right-3 w-[236px] bg-white rounded-2xl shadow-xl p-3 flex items-center gap-3 rotate-3">
              <img src="/dinos/dino-morado.png" alt="" className="h-10 w-auto shrink-0" />
              <div className="min-w-0">
                <p className="text-[13px] font-black text-ink truncate">Nuevo mensaje</p>
                <p className="text-[11px] text-slate-400 truncate">La maestra de Ryan te escribió</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ANTES / DESPUÉS */}
      <section className="max-w-6xl mx-auto px-5 sm:px-8 py-16 sm:py-20">
        <div className="max-w-xl mb-12">
          <p className="text-[11px] font-black text-brand-600 uppercase tracking-[0.14em] mb-4">El día a día, sin fricción</p>
          <h2 className="font-black text-ink" style={{ fontFamily: "'Lora', Georgia, serif", fontSize: 'clamp(1.8rem, 3.4vw, 2.5rem)' }}>
            Menos cuadernos y grupos de WhatsApp. Más control.
          </h2>
          <p className="mt-4 text-slate-600">
            Pasitos está pensado para guarderías y centros de cuidado infantil que quieren profesionalizar la relación con las familias.
          </p>
        </div>

        <div className="grid md:grid-cols-2 gap-5">
          <div className="bg-white border border-slate-200 rounded-[2rem] p-8">
            <span className="inline-block text-[10px] font-black uppercase tracking-widest text-slate-400 bg-slate-100 px-3.5 py-1.5 rounded-full">Antes</span>
            <h3 className="mt-5 text-lg font-bold text-slate-500">Cuadernos y trabajo manual</h3>
            <ul className="mt-5 space-y-3.5">
              {[
                'Asistencia registrada a mano en una lista',
                'Fotos y avisos sueltos en un grupo de WhatsApp',
                'Cuadernos que se pierden, se dañan o llegan tarde',
                'Cada maestra decide cómo y cuándo avisar a los papás',
              ].map((t) => (
                <li key={t} className="flex gap-3 text-sm text-slate-500">
                  <span className="w-1.5 h-0.5 bg-slate-300 mt-2.5 shrink-0" /> {t}
                </li>
              ))}
            </ul>
          </div>

          <div className="relative overflow-hidden bg-forest rounded-[2rem] p-8">
            <img
              src="/dinos/dino-amarillo.png"
              alt=""
              aria-hidden="true"
              className="pointer-events-none select-none absolute right-2 bottom-2 h-24 w-auto"
            />
            <span className="inline-block text-[10px] font-black uppercase tracking-widest text-forest bg-white px-3.5 py-1.5 rounded-full relative z-[1]">Con Pasitos</span>
            <h3 className="mt-5 text-lg font-bold text-white relative z-[1]">Una plataforma para todo el día</h3>
            <ul className="mt-5 space-y-3.5 relative z-[1]">
              {[
                'Entradas y salidas con reconocimiento facial, en segundos',
                'Bitácora del día para cada niño, disponible al instante',
                'Chat privado y directo entre cada familia y el staff',
                'Notificaciones automáticas de entradas, salidas y mensajes',
              ].map((t) => (
                <li key={t} className="flex gap-3 text-sm text-white/85">
                  <Check size={16} className="text-dino-amarillo mt-0.5 shrink-0" /> {t}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      {/* BENEFICIOS NUMERADOS */}
      <section className="max-w-6xl mx-auto px-5 sm:px-8 pb-16 sm:pb-20">
        <div className="grid sm:grid-cols-2 gap-5">
          {BENEFICIOS.map((b) => (
            <div key={b.n} className={`relative overflow-hidden border rounded-[1.75rem] p-7 ${b.fondo} ${b.borde}`}>
              <img
                src={b.dino}
                alt=""
                aria-hidden="true"
                className="pointer-events-none select-none absolute top-5 right-5 h-16 w-auto opacity-80"
              />
              <div className={`w-11 h-11 rounded-xl ${b.chip} ${b.color} flex items-center justify-center font-black text-sm shadow-sm`} style={{ fontFamily: "'Lora', Georgia, serif" }}>
                {b.n}
              </div>
              <h3 className="mt-5 text-[17px] font-bold text-ink relative z-[1]">{b.titulo}</h3>
              <p className="mt-2.5 text-sm text-slate-600 leading-relaxed relative z-[1]">{b.texto}</p>
            </div>
          ))}
        </div>
      </section>

      {/* FUNCIONES */}
      <section id="funciones" className="bg-brand-50/50 py-16 sm:py-20 scroll-mt-4">
        <div className="max-w-6xl mx-auto px-5 sm:px-8">
          <div className="max-w-xl mb-10">
            <p className="text-[11px] font-black text-brand-600 uppercase tracking-[0.14em] mb-4">Todo lo importante, en un solo lugar</p>
            <h2 className="font-black text-ink" style={{ fontFamily: "'Lora', Georgia, serif", fontSize: 'clamp(1.8rem, 3.4vw, 2.5rem)' }}>
              Una plataforma. Cada momento del día.
            </h2>
          </div>

          <div className="flex flex-wrap gap-2.5 mb-9">
            {FUNCIONES.map((f) => (
              <button
                key={f.key}
                onClick={() => setTabActiva(f.key)}
                className={`text-[11px] font-black uppercase tracking-widest px-5 py-3 rounded-xl transition-all ${
                  tabActiva === f.key ? 'bg-forest text-white shadow-sm' : 'bg-white text-slate-500 border border-slate-200 hover:border-brand-300 hover:text-brand-700'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {funcionesActivas.items.map((item, i) => {
              const tono = tonoDeTarjeta(funcionesActivas.tono, i);
              return (
                <div
                  key={item.titulo}
                  className={`relative overflow-hidden border rounded-[1.75rem] p-7 ${tono.fondo} ${tono.borde}`}
                >
                  <img
                    src={tono.dino}
                    alt=""
                    aria-hidden="true"
                    className="pointer-events-none select-none absolute top-3 right-4 h-[74px] w-auto"
                  />
                  <div className="w-12 h-12 rounded-2xl bg-white flex items-center justify-center shadow-sm">
                    <item.Icon size={24} strokeWidth={1.8} className={tono.texto} />
                  </div>
                  <h3 className="mt-4 text-[17px] font-bold text-ink relative z-[1]">{item.titulo}</h3>
                  <p className="mt-2 text-sm text-slate-600 leading-relaxed relative z-[1]">{item.texto}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* SEGURIDAD */}
      <section className="max-w-6xl mx-auto px-5 sm:px-8 py-16 sm:py-20 grid lg:grid-cols-[0.9fr_1.1fr] gap-12 items-center">
        <div>
          <p className="text-[11px] font-black text-brand-600 uppercase tracking-[0.14em] mb-4">Datos sensibles, tratados como tal</p>
          <h2 className="font-black text-ink" style={{ fontFamily: "'Lora', Georgia, serif", fontSize: 'clamp(1.8rem, 3.4vw, 2.3rem)' }}>
            Pensado para manejar información delicada.
          </h2>
          <p className="mt-4 text-slate-600">
            Reconocimiento facial, expedientes y pagos no son datos cualquiera. Pasitos se construyó con eso en mente desde el primer día.
          </p>
        </div>

        <div className="space-y-4">
          {[
            { Icon: ShieldCheck, chip: 'bg-brand-100', icono: 'text-brand-600', texto: <><strong className="text-ink">El reconocimiento facial se usa solo para registrar entradas y salidas</strong> -- no queda como una foto suelta que cualquiera puede reenviar.</> },
            { Icon: Lock, chip: 'bg-dino-morado-suave', icono: 'text-dino-morado', texto: <><strong className="text-ink">Cada cuenta entra con su propia contraseña</strong>, y las secciones más delicadas piden un PIN aparte antes de mostrarse.</> },
            { Icon: Clock3, chip: 'bg-dino-amarillo-suave', icono: 'text-dino-amarillo', texto: <><strong className="text-ink">La sesión se cierra sola</strong> tras un rato de inactividad en las cuentas de staff y administración.</> },
          ].map((item, i) => (
            <div key={i} className="bg-white border border-slate-200 rounded-2xl p-6 flex gap-4">
              <div className={`w-10 h-10 rounded-xl ${item.chip} flex items-center justify-center shrink-0`}>
                <item.Icon size={19} className={item.icono} />
              </div>
              <p className="text-sm text-slate-600 leading-relaxed">{item.texto}</p>
            </div>
          ))}
        </div>
      </section>

      {/* CTA FINAL */}
      <section className="relative overflow-hidden bg-forest">
        <div className="absolute -bottom-32 -left-24 w-[420px] h-[420px] rounded-full bg-brand-600/25 blur-3xl pointer-events-none" />
        <div className="absolute -top-28 right-0 w-[360px] h-[360px] rounded-full bg-dino-morado/25 blur-3xl pointer-events-none" />
        <div className="absolute top-1/3 left-1/3 w-[260px] h-[260px] rounded-full bg-brand-400/15 blur-3xl pointer-events-none" />
        <img
          src="/dinos/dino-morado.png"
          alt=""
          aria-hidden="true"
          className="hidden md:block pointer-events-none select-none absolute left-12 bottom-10 h-24 w-auto"
        />
        <img
          src="/dinos/dino-verde.png"
          alt=""
          aria-hidden="true"
          className="hidden md:block pointer-events-none select-none absolute right-12 top-12 h-20 w-auto"
        />
        <div className="max-w-2xl mx-auto px-5 sm:px-8 py-20 sm:py-24 text-center relative">
          <h2 className="font-black text-white" style={{ fontFamily: "'Lora', Georgia, serif", fontSize: 'clamp(1.9rem, 3.8vw, 2.8rem)' }}>
            ¿Lista para dejar el cuaderno?
          </h2>
          <p className="mt-4 text-white/70">
            Solicita tu acceso y te mostramos cómo funciona Pasitos en tu guardería, paso a paso.
          </p>
          <Link
            to="/registro-guarderia"
            className="mt-8 inline-flex items-center gap-2.5 bg-brand-600 text-white text-xs font-black uppercase tracking-widest px-7 py-4 rounded-2xl shadow-lg hover:bg-brand-700 active:scale-95 transition-all"
          >
            Solicita tu acceso <ArrowUpRight size={16} />
          </Link>
        </div>
      </section>

      {/* FOOTER */}
      <footer className="max-w-6xl mx-auto px-5 sm:px-8 py-9 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-2.5">
          <img src="/dinos/logo-pasitos.png" alt="" className="h-8 w-auto shrink-0" />
          <span className="font-black uppercase text-sm text-brand-900">Pasitos</span>
        </div>
        <div className="flex items-center gap-5">
          <Link to="/terminos" className="text-xs font-bold text-slate-500 hover:text-brand-600 transition-colors">
            Términos y condiciones
          </Link>
          <a href="mailto:alejandrobuenomendoza@gmail.com" className="text-xs font-bold text-slate-500 hover:text-brand-600 transition-colors">
            alejandrobuenomendoza@gmail.com
          </a>
        </div>
      </footer>
    </div>
    {/* "Posibles nuevos clientes" -- para alguien interesado que todavía no
        tiene cuenta, sin depender de que escriban al correo del footer. */}
    <SoporteChat modo="publico" />
    </>
  );
};

export default LandingPage;
