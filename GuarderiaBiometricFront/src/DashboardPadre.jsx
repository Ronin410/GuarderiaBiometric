import React, { useEffect, useState } from 'react';
import api from './axiosConfig';
import {
  User,
  ChevronRight,
  Heart,
  LogOut,
  Baby,
  Bell,
  BellRing,
  Wallet,
  CheckCircle2,
  Clock,
  XCircle,
  UtensilsCrossed,
  Megaphone,
  MessageCircle,
  CalendarDays,
  ClipboardCheck,
  AlertTriangle,
  BookOpen
} from 'lucide-react';
import VistaPadreDetalle from './VistaPadreDetalle';
import ChatPadre from './ChatPadre';
import EncuestasPadre from './EncuestasPadre';
import CircularesPadre from './CircularesPadre';
import EventosPadre from './EventosPadre';
import MenuPadre from './MenuPadre';
import { suscribirseAPush, desuscribirseDePush, suscripcionActiva, pushSoportado } from './utils/push';
import { hoyLocal } from './utils/fecha';
import InstalarApp from './components/InstalarApp';
import { mostrarExito, mostrarAviso } from './utils/alertas';

const formatoFechaEvento = (iso) => {
  try {
    const [anio, mes, dia] = iso.split('-').map(Number);
    return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { day: 'numeric', month: 'short' });
  } catch {
    return iso;
  }
};

const formatoFechaCircular = (iso) => {
  try {
    return new Date(iso).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' });
  } catch {
    return iso;
  }
};

const ESTADO_PAGO_INFO = {
  pagado: { label: 'Pagado', color: 'bg-emerald-100 text-emerald-700 border-emerald-200', icon: CheckCircle2 },
  parcial: { label: 'Parcial', color: 'bg-amber-100 text-amber-700 border-amber-200', icon: Clock },
  pendiente: { label: 'Pendiente', color: 'bg-slate-100 text-slate-500 border-slate-200', icon: Clock },
  vencido: { label: 'Vencido', color: 'bg-rose-100 text-rose-700 border-rose-200', icon: XCircle },
};

const DashboardPadre = ({ padreId, nombreUsuario, alCerrarSesion }) => {
  const [hijos, setHijos] = useState([]);
  const [hijoSeleccionado, setHijoSeleccionado] = useState(null);
  const [mostrarChat, setMostrarChat] = useState(false);
  const [mostrarEncuestas, setMostrarEncuestas] = useState(false);
  const [mostrarCirculares, setMostrarCirculares] = useState(false);
  const [mostrarEventos, setMostrarEventos] = useState(false);
  const [mostrarMenu, setMostrarMenu] = useState(false);
  const [loading, setLoading] = useState(true);
  const usuarioNombre = nombreUsuario || 'Familia';
  const [pagos, setPagos] = useState([]);
  const [menuHoy, setMenuHoy] = useState(null);
  const [circulares, setCirculares] = useState([]);
  const [eventos, setEventos] = useState([]);
  const [notifEstado, setNotifEstado] = useState('default');

  // Stripe redirige aquí (a la raíz de la app) con ?pago_colegiatura=exito
  // o =cancelado tras el Checkout -- ver success_url/cancel_url en
  // pagos_online.go. Se limpia el query param con replaceState para que un
  // refresh no vuelva a mostrar el aviso.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const resultado = params.get('pago_colegiatura');
    if (!resultado) return;
    params.delete('pago_colegiatura');
    const nuevaURL = window.location.pathname + (params.toString() ? `?${params}` : '');
    window.history.replaceState({}, '', nuevaURL);

    if (resultado === 'exito') {
      mostrarExito('Tu pago se procesó correctamente. Puede tardar unos segundos en verse reflejado aquí.');
    } else if (resultado === 'cancelado') {
      mostrarAviso('No se completó el pago. Puedes intentarlo de nuevo cuando quieras.', 'Pago cancelado');
    }
  }, []);

  useEffect(() => {
    const cargarDatosIniciales = async () => {
      try {
        setLoading(true);

        // 1. Obtener hijos del padre autenticado. Usamos siempre el comodín "0":
        // el backend lo resuelve con el user_id del token, y solo permite pasar
        // un ID explícito distinto de "0" a cuentas admin/staff (ver /padre/:id/hijos
        // en main.go) — si mandáramos aquí el padreId real, un papá recibiría 403.
        // Ya incluye el expediente extendido (fecha de nacimiento, dirección, etc.).
        const resHijos = await api.get('/padre/0/hijos');

        const hijosFormateados = (resHijos.data || []).map(h => ({
          id: h.id,
          nombre: h.nombre_niño || h.nombre || "Sin nombre",
          activo: h.activo,
          fechaNacimiento: h.fecha_nacimiento,
          direccion: h.direccion,
          contactoEmergenciaNombre: h.contacto_emergencia_nombre,
          contactoEmergenciaTelefono: h.contacto_emergencia_telefono,
        }));

        setHijos(hijosFormateados);

        // 2. Estado de pago del mes actual (solo de mis hijos)
        try {
          const resPagos = await api.get('/padre/mis-pagos');
          setPagos(Array.isArray(resPagos.data) ? resPagos.data : []);
        } catch (errPagos) {
          console.error("Error al cargar el estado de pagos", errPagos);
        }

        // 3. Menú de hoy (si el staff ya lo cargó)
        try {
          const hoy = hoyLocal();
          const resMenu = await api.get('/padre/menu-semanal', { params: { inicio: hoy, fin: hoy } });
          const dia = (resMenu.data || [])[0];
          if (dia && (dia.desayuno || dia.comida || dia.merienda)) setMenuHoy(dia);
        } catch (errMenu) {
          console.error("Error al cargar el menú del día", errMenu);
        }

        // 4. Circulares -- el inicio solo necesita saber si hay alguna y
        // cuál es la más reciente, para el aviso "hay una circular nueva".
        // El listado completo (y el marcar-como-leída, que debe pasar
        // cuando el aviso de verdad se muestra completo) viven en
        // CircularesPadre.jsx.
        try {
          const resCirculares = await api.get('/padre/circulares');
          setCirculares(Array.isArray(resCirculares.data) ? resCirculares.data : []);
        } catch (errCirculares) {
          console.error("Error al cargar circulares", errCirculares);
        }

        // 5. Próximos eventos del calendario escolar
        try {
          const resEventos = await api.get('/padre/calendario');
          setEventos((Array.isArray(resEventos.data) ? resEventos.data : []).slice(0, 3));
        } catch (errEventos) {
          console.error("Error al cargar el calendario", errEventos);
        }

      } catch (err) {
        console.error("Error al cargar el dashboard de padre", err);
      } finally {
        setLoading(false);
      }
    };
    cargarDatosIniciales();
  }, [padreId]);

  // Aparte del resto (no debe frenar la carga del dashboard): revisa si ya hay
  // una suscripción push activa en este navegador.
  useEffect(() => {
    suscripcionActiva().then((activa) => setNotifEstado(activa ? 'granted' : 'default'));
  }, []);

  const handleActivarNotificaciones = async () => {
    setNotifEstado('activando');
    try {
      const ok = await suscribirseAPush(api);
      setNotifEstado(ok ? 'granted' : 'default');
      if (!ok) {
        alert('No se pudieron activar las notificaciones. Revisa los permisos de notificaciones de tu navegador (en iPhone, agrega esta página a tu pantalla de inicio primero).');
      }
    } catch (err) {
      console.error('Error al activar notificaciones', err);
      setNotifEstado('default');
      alert(err.message || 'No se pudieron activar las notificaciones. Inténtalo de nuevo.');
    }
  };

  // "Quiero que las notificaciones las pueda cancelar el papá con el
  // mismo botón" -- mismo botón que activa, ahora también desactiva
  // cuando ya están prendidas.
  const handleDesactivarNotificaciones = async () => {
    setNotifEstado('desactivando');
    try {
      const ok = await desuscribirseDePush(api);
      setNotifEstado(ok ? 'default' : 'granted');
      if (!ok) {
        alert('No se pudieron desactivar las notificaciones. Inténtalo de nuevo.');
      }
    } catch (err) {
      console.error('Error al desactivar notificaciones', err);
      setNotifEstado('granted');
      alert('No se pudieron desactivar las notificaciones. Inténtalo de nuevo.');
    }
  };

  const handleLogout = () => {
    if (typeof alCerrarSesion === 'function') {
      alCerrarSesion();
    } else {
      window.location.href = '/';
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-paper flex flex-col items-center justify-center p-10 text-center">
        <div className="w-16 h-16 border-4 border-brand-200 border-t-brand-600 rounded-full animate-spin mb-4"></div>
        <p className="font-black text-slate-400 uppercase tracking-[0.2em] text-xs">Sincronizando Familia...</p>
      </div>
    );
  }

  // Navegación a la vista de detalle
  if (mostrarChat) {
    return <ChatPadre onVolver={() => setMostrarChat(false)} />;
  }

  if (mostrarEncuestas) {
    return <EncuestasPadre onVolver={() => setMostrarEncuestas(false)} />;
  }

  if (mostrarCirculares) {
    return <CircularesPadre onVolver={() => setMostrarCirculares(false)} />;
  }

  if (mostrarEventos) {
    return <EventosPadre onVolver={() => setMostrarEventos(false)} />;
  }

  if (mostrarMenu) {
    return <MenuPadre onVolver={() => setMostrarMenu(false)} />;
  }

  if (hijoSeleccionado) {
    return (
      <VistaPadreDetalle
        hijoId={hijoSeleccionado.id}
        nombreHijo={hijoSeleccionado.nombre}
        expediente={hijoSeleccionado}
        onVolver={() => setHijoSeleccionado(null)}
      />
    );
  }

  return (
    <div className="min-h-screen bg-paper pb-10">
      {/* NAVBAR */}
      <div className="bg-white px-6 py-4 flex justify-between items-center border-b border-slate-100 sticky top-0 z-10 shadow-sm">
        <div className="flex items-center gap-2">
          <img src="/dinos/logo-pasitos.png" alt="" className="h-8 w-auto shrink-0" />
          <span className="font-black text-slate-900 uppercase tracking-tighter text-sm">PASITOS <span className="text-brand-600">FAMILIA</span></span>
        </div>
        <div className="flex items-center gap-1">
          <a
            href="/manual-papa.html" target="_blank" rel="noopener noreferrer"
            className="p-2 text-slate-400 hover:text-brand-600 transition-colors"
            title="Manual"
          >
            <BookOpen size={20} />
          </a>
          <button onClick={handleLogout} className="p-2 text-slate-400 hover:text-rose-500 transition-colors">
            <LogOut size={20} />
          </button>
        </div>
      </div>

      <div className="max-w-md mx-auto p-6 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
        
        {/* SALUDO - Ahora dinámico */}
        <div className="space-y-2 text-center pt-4">
          <h2 className="text-4xl font-black text-slate-900 uppercase tracking-tighter leading-none">
            Hola, <br/><span className="text-brand-600">{usuarioNombre}</span>
          </h2>
          <p className="text-slate-400 font-bold uppercase text-[10px] tracking-[0.15em]">
            ¿De quién deseas ver el reporte hoy?
          </p>
        </div>

        <InstalarApp />

        {/* INFO CARD */}
        <div className="bg-brand-50 border border-brand-100 p-4 rounded-[2rem] flex items-center gap-4">
          <div className="bg-white p-3 rounded-2xl text-brand-600 shadow-sm">
            <Bell size={20} />
          </div>
          <p className="text-[10px] font-black text-brand-800 uppercase leading-tight tracking-tight">
            Las bitácoras se actualizan en tiempo real por las maestras.
          </p>
        </div>

        {/* CHAT CON LA GUARDERÍA */}
        <button
          onClick={() => setMostrarChat(true)}
          className="w-full bg-white p-4 rounded-[2rem] border border-slate-100 shadow-sm flex items-center gap-4 hover:shadow-md transition-all active:scale-[0.98]"
        >
          <div className="bg-dino-naranja-suave p-3 rounded-2xl text-dino-naranja shrink-0"><MessageCircle size={20} /></div>
          <div className="flex-1 text-left">
            <p className="text-[11px] font-black text-slate-900 uppercase leading-tight">Chat con la guardería</p>
            <p className="text-[9px] text-slate-400 font-bold uppercase tracking-wide">Mensajes directos, sin WhatsApp</p>
          </div>
          <ChevronRight size={20} className="text-slate-300 shrink-0" />
        </button>

        {/* ENCUESTAS */}
        <button
          onClick={() => setMostrarEncuestas(true)}
          className="w-full bg-white p-4 rounded-[2rem] border border-slate-100 shadow-sm flex items-center gap-4 hover:shadow-md transition-all active:scale-[0.98]"
        >
          <div className="bg-dino-morado-suave p-3 rounded-2xl text-dino-morado shrink-0"><ClipboardCheck size={20} /></div>
          <div className="flex-1 text-left">
            <p className="text-[11px] font-black text-slate-900 uppercase leading-tight">Encuestas</p>
            <p className="text-[9px] text-slate-400 font-bold uppercase tracking-wide">Comparte tu opinión con la guardería</p>
          </div>
          <ChevronRight size={20} className="text-slate-300 shrink-0" />
        </button>

        {/* EVENTOS -- "quiero que los eventos también estén en un menú
            aparte", mismo criterio que Chat/Encuestas: entrada fija, no
            condicionada a si hay algo próximo (a diferencia del teaser de
            "Próximos eventos" más abajo, que sí depende de eso). */}
        <button
          onClick={() => setMostrarEventos(true)}
          className="w-full bg-white p-4 rounded-[2rem] border border-slate-100 shadow-sm flex items-center gap-4 hover:shadow-md transition-all active:scale-[0.98]"
        >
          <div className="bg-dino-verde-suave p-3 rounded-2xl text-dino-verde shrink-0"><CalendarDays size={20} /></div>
          <div className="flex-1 text-left">
            <p className="text-[11px] font-black text-slate-900 uppercase leading-tight">Eventos</p>
            <p className="text-[9px] text-slate-400 font-bold uppercase tracking-wide">Calendario escolar de la guardería</p>
          </div>
          <ChevronRight size={20} className="text-slate-300 shrink-0" />
        </button>

        {/* MENÚ SEMANAL -- "quiero que puedan ver también qué comieron en
            días anteriores o posteriores": entrada fija, igual que Chat/
            Encuestas/Eventos, para llegar a la semana completa aunque hoy
            todavía no tenga menú capturado (el teaser "Menú de hoy" de
            abajo solo aparece cuando SÍ hay algo hoy). */}
        <button
          onClick={() => setMostrarMenu(true)}
          className="w-full bg-white p-4 rounded-[2rem] border border-slate-100 shadow-sm flex items-center gap-4 hover:shadow-md transition-all active:scale-[0.98]"
        >
          <div className="bg-dino-amarillo-suave p-3 rounded-2xl text-dino-amarillo shrink-0"><UtensilsCrossed size={20} /></div>
          <div className="flex-1 text-left">
            <p className="text-[11px] font-black text-slate-900 uppercase leading-tight">Menú semanal</p>
            <p className="text-[9px] text-slate-400 font-bold uppercase tracking-wide">Días anteriores y posteriores</p>
          </div>
          <ChevronRight size={20} className="text-slate-300 shrink-0" />
        </button>

        {/* NOTIFICACIONES PUSH */}
        {pushSoportado() && (
          <div className="bg-white border border-slate-100 p-4 rounded-[2rem] flex items-center gap-4 shadow-sm">
            <div className={`p-3 rounded-2xl ${notifEstado === 'granted' ? 'bg-emerald-50 text-emerald-600' : 'bg-slate-50 text-slate-400'}`}>
              <BellRing size={20} />
            </div>
            <div className="flex-1">
              <p className="text-[11px] font-black text-slate-900 uppercase leading-tight">
                {notifEstado === 'granted' ? 'Notificaciones activas' : 'Activar notificaciones'}
              </p>
              <p className="text-[9px] text-slate-400 font-bold uppercase tracking-wide">
                Entradas, salidas y bitácora al instante
              </p>
            </div>
            <button
              onClick={notifEstado === 'granted' ? handleDesactivarNotificaciones : handleActivarNotificaciones}
              disabled={notifEstado === 'activando' || notifEstado === 'desactivando'}
              className={`text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95 disabled:opacity-50 ${
                notifEstado === 'granted' ? 'bg-slate-100 hover:bg-slate-200 text-slate-500' : 'bg-brand-600 hover:bg-brand-700 text-white'
              }`}
            >
              {notifEstado === 'activando' || notifEstado === 'desactivando' ? '...' : notifEstado === 'granted' ? 'Desactivar' : 'Activar'}
            </button>
          </div>
        )}

        {/* ESTADO DE PAGOS DEL MES */}
        {pagos.length > 0 && (
          <div className="space-y-3">
            <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-widest ml-2">Pagos de este mes</h3>
            {pagos.map((p) => {
              const info = ESTADO_PAGO_INFO[p.estado] || ESTADO_PAGO_INFO.pendiente;
              const Icono = info.icon;
              return (
                <div key={p.hijo_id} className="bg-white p-4 rounded-2xl border border-slate-100 flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="bg-slate-50 p-2.5 rounded-xl text-slate-400"><Wallet size={16} /></div>
                    <p className="font-bold text-sm text-slate-700">{p.nombre}</p>
                  </div>
                  <span className={`flex items-center gap-1 text-[9px] font-black px-2.5 py-1.5 rounded-lg border uppercase ${info.color}`}>
                    <Icono size={11} /> {info.label}
                  </span>
                </div>
              );
            })}
            {/* Deuda de meses anteriores al actual -- separado del estado
                de arriba a propósito: un niño puede aparecer "Pagado" este
                mes y aun así deber de un mes pasado que quedó a medias. */}
            {pagos.some((p) => p.deuda_acumulada > 0) && (
              <div className="bg-rose-50 border border-rose-200 rounded-2xl p-4 flex items-start gap-3">
                <AlertTriangle size={18} className="text-rose-500 shrink-0 mt-0.5" />
                <p className="text-xs font-bold text-rose-600 leading-snug">
                  {pagos.filter((p) => p.deuda_acumulada > 0).map((p) => (
                    <span key={p.hijo_id} className="block">
                      {p.nombre}: debe ${Number(p.deuda_acumulada).toLocaleString('es-MX', { minimumFractionDigits: 2 })} de meses anteriores
                    </span>
                  ))}
                </p>
              </div>
            )}
          </div>
        )}

        {/* CIRCULARES -- el inicio muestra el contenido completo del aviso
            más reciente (antes solo mostraba el título como "hay una
            circular nueva"); las anteriores se ven tocando "Ver avisos
            anteriores", que manda al listado completo de
            CircularesPadre.jsx. */}
        {circulares.length > 0 && (
          <div className="bg-white rounded-[2rem] border border-slate-100 shadow-sm overflow-hidden">
            <button
              onClick={() => setMostrarCirculares(true)}
              className="w-full text-left p-5 space-y-2 hover:bg-slate-50 transition-all active:scale-[0.99]"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-brand-600">
                  <Megaphone size={16} />
                  <span className="text-[9px] font-black uppercase tracking-widest">Último aviso</span>
                </div>
                <ChevronRight size={18} className="text-slate-300 shrink-0" />
              </div>
              <p className="font-black text-slate-900 uppercase text-sm leading-tight">{circulares[0].titulo}</p>
              <p className="text-[9px] text-slate-400 font-bold uppercase tracking-wide">{formatoFechaCircular(circulares[0].creado_en)}</p>
              <p className="text-xs text-slate-600 font-medium whitespace-pre-wrap line-clamp-3">{circulares[0].contenido}</p>
              {circulares[0].imagen_url && (
                <img src={circulares[0].imagen_url} alt={circulares[0].titulo} className="mt-2 w-full max-h-48 rounded-2xl border border-slate-100 object-cover" />
              )}
            </button>
            {circulares.length > 1 && (
              <button
                onClick={() => setMostrarCirculares(true)}
                className="w-full text-center py-2.5 text-[9px] font-black uppercase tracking-widest text-brand-500 border-t border-slate-100 hover:bg-slate-50 transition-colors"
              >
                Ver avisos anteriores
              </button>
            )}
          </div>
        )}

        {/* PRÓXIMOS EVENTOS DEL CALENDARIO -- adelanto de lo que hay en el
            menú "Eventos" de arriba; tocar cualquiera manda al listado
            completo, igual que el resto de la app. */}
        {eventos.length > 0 && (
          <div className="space-y-3">
            <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-widest ml-2">Próximos eventos</h3>
            {eventos.map((ev) => (
              <button
                key={ev.id}
                onClick={() => setMostrarEventos(true)}
                className="w-full bg-white p-4 rounded-2xl border border-slate-100 flex items-center gap-3 text-left hover:shadow-md transition-all active:scale-[0.98]"
              >
                <div className="bg-slate-50 p-2.5 rounded-xl text-brand-500 shrink-0"><CalendarDays size={16} /></div>
                <div className="min-w-0">
                  <p className="font-black text-sm text-slate-900 truncate">{ev.titulo}</p>
                  <p className="text-[10px] text-slate-400 font-bold uppercase">{formatoFechaEvento(ev.fecha_inicio)}{ev.fecha_fin && ev.fecha_fin !== ev.fecha_inicio ? ` – ${formatoFechaEvento(ev.fecha_fin)}` : ''}</p>
                </div>
              </button>
            ))}
          </div>
        )}

        {/* MENÚ DE HOY -- clickeable hacia la misma semana completa que el
            botón fijo de arriba, para quien lo toca esperando ver otros
            días directo desde aquí. */}
        {menuHoy && (
          <button
            onClick={() => setMostrarMenu(true)}
            className="w-full bg-white p-5 rounded-[2rem] border border-slate-100 shadow-sm space-y-2 text-left hover:shadow-md transition-all active:scale-[0.98]"
          >
            <div className="flex items-center justify-between">
              <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-widest flex items-center gap-2">
                <UtensilsCrossed size={14} className="text-brand-500" /> Menú de hoy
              </h3>
              <ChevronRight size={16} className="text-slate-300 shrink-0" />
            </div>
            {menuHoy.desayuno && <p className="text-xs font-bold text-slate-700"><span className="text-brand-500">Desayuno:</span> {menuHoy.desayuno}</p>}
            {menuHoy.comida && <p className="text-xs font-bold text-slate-700"><span className="text-brand-500">Comida:</span> {menuHoy.comida}</p>}
            {menuHoy.merienda && <p className="text-xs font-bold text-slate-700"><span className="text-brand-500">Merienda:</span> {menuHoy.merienda}</p>}
          </button>
        )}

        {/* LISTADO DE NIÑOS */}
        <div className="space-y-4">
          {hijos.length === 0 ? (
            <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center">
              <Baby size={40} className="mx-auto text-slate-200 mb-4" />
              <p className="text-slate-400 font-bold uppercase text-[10px]">No se encontraron niños vinculados.</p>
            </div>
          ) : (
            hijos.map((hijo) => (
              <button
                key={hijo.id}
                onClick={() => setHijoSeleccionado(hijo)}
                className="w-full bg-white p-6 rounded-[2.5rem] border border-slate-100 shadow-md hover:shadow-xl transition-all flex items-center justify-between group active:scale-95"
              >
                <div className="flex items-center gap-5">
                  <div className="w-16 h-16 bg-slate-50 rounded-[1.5rem] flex items-center justify-center text-slate-300 group-hover:bg-brand-100 group-hover:text-brand-600 transition-all border border-slate-100">
                    <User size={32} />
                  </div>

                  <div className="text-left">
                    <h4 className="font-black text-slate-900 uppercase text-lg tracking-tight group-hover:text-brand-600 transition-colors">
                      {hijo.nombre}
                    </h4>
                    <div className="flex items-center gap-1 text-brand-500">
                      <Heart size={10} fill="currentColor" />
                      <p className="text-[9px] font-black uppercase tracking-widest">Ver Bitácora Diaria</p>
                    </div>
                  </div>
                </div>

                <div className="bg-slate-50 p-3 rounded-2xl text-slate-300 group-hover:bg-brand-600 group-hover:text-white transition-all shadow-sm">
                  <ChevronRight size={20} />
                </div>
              </button>
            ))
          )}
        </div>

        {/* FOOTER */}
        <div className="text-center pt-8">
            <p className="text-[8px] font-black text-slate-300 uppercase tracking-[0.4em]">Protegido por Pasitos</p>
        </div>
      </div>
    </div>
  );
};

export default DashboardPadre;