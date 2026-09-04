import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Link } from 'react-router-dom';
import api from './axiosConfig';
import { LifeBuoy, X, Send, Loader2, ChevronDown } from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { fechaLocal, separadorFecha } from './utils/fecha';

const INTERVALO_POLLING_MS = 8000;
const TOKEN_PROSPECTO_KEY = 'pasitos_soporte_token';

// SoporteChat -- burbuja flotante de "chat de soporte" con el dueño de la
// plataforma (Alejandro), disponible en dos modos:
//   - "autenticado": para papás y staff/admin ya con sesión (cualquier
//     guardería). Usa /soporte/mis-mensajes -- el backend resuelve quién
//     eres a partir del JWT, una sola conversación continua por cuenta.
//   - "publico": para "posibles nuevos clientes" que todavía no tienen
//     cuenta (visitantes de la página de presentación). La primera vez
//     pide nombre/correo/mensaje, y guarda el token que regresa el backend
//     en localStorage para poder seguir la MISMA conversación en visitas
//     futuras sin necesidad de crear una cuenta.
// Las respuestas de Alejandro se ven desde /plataforma (ver PanelPlataforma).
const SoporteChat = ({ modo }) => {
  const [abierto, setAbierto] = useState(false);
  const [mensajes, setMensajes] = useState([]);
  const [cargando, setCargando] = useState(false);
  const [texto, setTexto] = useState('');
  const [enviando, setEnviando] = useState(false);
  const [noLeidos, setNoLeidos] = useState(0);
  const finRef = useRef(null);

  // Solo aplica al modo público -- null hasta que el prospecto llena el
  // formulario inicial (o si ya lo llenó antes, se recupera de localStorage).
  const [token, setToken] = useState(() => (modo === 'publico' ? localStorage.getItem(TOKEN_PROSPECTO_KEY) : null));
  const [form, setForm] = useState({ nombre: '', email: '', mensaje: '' });
  const [enviandoForm, setEnviandoForm] = useState(false);

  const listoParaChatear = modo === 'autenticado' || !!token;

  const rutaMensajes = modo === 'autenticado' ? '/soporte/mis-mensajes' : `/soporte/prospecto/${token}/mensajes`;

  const cargarMensajes = useCallback(async (mostrarLoading) => {
    if (!listoParaChatear) return;
    if (mostrarLoading) setCargando(true);
    try {
      const res = await api.get(rutaMensajes);
      setMensajes(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar el chat de soporte:', err);
      // Un token de prospecto que ya no existe (ej. limpiado en el backend)
      // no debe dejar el widget en un error permanente -- se olvida el
      // token guardado y se vuelve a mostrar el formulario inicial.
      if (modo === 'publico' && err.response?.status === 404) {
        localStorage.removeItem(TOKEN_PROSPECTO_KEY);
        setToken(null);
      }
    } finally {
      if (mostrarLoading) setCargando(false);
    }
  }, [rutaMensajes, listoParaChatear, modo]);

  // Badge de no leídos -- solo tiene sentido en modo autenticado (un
  // prospecto no tiene de dónde recibir la notificación fuera del propio
  // widget, así que no hace falta consultarlo si el panel ya está cerrado).
  useEffect(() => {
    if (modo !== 'autenticado') return undefined;
    const consultar = () => {
      api.get('/soporte/no-leidos')
        .then((res) => setNoLeidos(res.data?.no_leidos || 0))
        .catch(() => {});
    };
    consultar();
    const intervalo = setInterval(consultar, INTERVALO_POLLING_MS);
    return () => clearInterval(intervalo);
  }, [modo]);

  useEffect(() => {
    if (!abierto || !listoParaChatear) return undefined;
    cargarMensajes(true);
    const intervalo = setInterval(() => cargarMensajes(false), INTERVALO_POLLING_MS);
    return () => clearInterval(intervalo);
  }, [abierto, listoParaChatear, cargarMensajes]);

  useEffect(() => {
    if (abierto) setNoLeidos(0);
  }, [abierto, mensajes]);

  useEffect(() => {
    finRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [mensajes]);

  const iniciarConversacionProspecto = async (e) => {
    e.preventDefault();
    const nombre = form.nombre.trim();
    const email = form.email.trim();
    const mensaje = form.mensaje.trim();
    if (!nombre || !email || !mensaje) {
      mostrarError('Completa tu nombre, correo y mensaje');
      return;
    }
    setEnviandoForm(true);
    try {
      const res = await api.post('/soporte/prospecto', { nombre, email, mensaje });
      localStorage.setItem(TOKEN_PROSPECTO_KEY, res.data.token);
      setToken(res.data.token);
    } catch (err) {
      console.error('Error al iniciar la conversación de soporte:', err);
      mostrarError(err.response?.data?.error || 'No se pudo enviar tu mensaje, intenta de nuevo');
    } finally {
      setEnviandoForm(false);
    }
  };

  const enviar = async () => {
    const contenido = texto.trim();
    if (!contenido) return;
    setEnviando(true);
    try {
      await api.post(rutaMensajes, { contenido });
      setTexto('');
      await cargarMensajes(false);
    } catch (err) {
      console.error('Error al enviar el mensaje de soporte:', err);
      mostrarError(err.response?.data?.error || 'No se pudo enviar el mensaje');
    } finally {
      setEnviando(false);
    }
  };

  const alPresionarTecla = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      enviar();
    }
  };

  const formatoHora = (iso) => {
    try {
      return new Date(iso).toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit' });
    } catch {
      return '';
    }
  };

  return (
    <>
      {/* BURBUJA FLOTANTE */}
      <button
        onClick={() => setAbierto((v) => !v)}
        className="fixed bottom-5 right-5 z-[310] bg-forest hover:bg-forest-light text-white p-4 rounded-full shadow-xl shadow-forest/30 transition-all active:scale-95 flex items-center justify-center"
        title="Soporte"
      >
        {abierto ? <ChevronDown size={24} /> : <LifeBuoy size={24} />}
        {!abierto && noLeidos > 0 && (
          <span className="absolute -top-1 -right-1 bg-brand-600 text-white text-[10px] font-black w-5 h-5 rounded-full flex items-center justify-center border-2 border-white">
            {noLeidos > 9 ? '9+' : noLeidos}
          </span>
        )}
      </button>

      {/* PANEL */}
      {abierto && (
        <div className="fixed bottom-24 right-5 z-[310] w-[calc(100vw-2.5rem)] max-w-sm h-[min(32rem,70vh)] bg-white rounded-[2rem] shadow-2xl border border-slate-200 flex flex-col overflow-hidden animate-in fade-in slide-in-from-bottom-4 duration-300">
          {/* HEADER */}
          <div className="bg-forest p-5 flex items-center justify-between shrink-0">
            <div className="flex items-center gap-3 min-w-0">
              <div className="bg-white/15 p-2 rounded-xl shrink-0"><LifeBuoy size={18} className="text-white" /></div>
              <div className="min-w-0">
                <p className="text-white font-black uppercase text-sm leading-tight">Soporte</p>
                <p className="text-white/60 text-[10px] font-bold">Te respondemos por aquí</p>
              </div>
            </div>
            <button onClick={() => setAbierto(false)} className="text-white/70 hover:text-white p-1 shrink-0"><X size={20} /></button>
          </div>

          {!listoParaChatear ? (
            // FORMULARIO INICIAL (solo prospectos sin token todavía)
            <form onSubmit={iniciarConversacionProspecto} className="flex-1 overflow-y-auto p-5 space-y-3">
              <p className="text-xs text-slate-500 font-medium mb-2">Cuéntanos quién eres y en qué te ayudamos -- te contestamos por este mismo chat.</p>
              <input
                type="text" value={form.nombre} onChange={(e) => setForm({ ...form, nombre: e.target.value })}
                placeholder="Tu nombre" className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm"
              />
              <input
                type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })}
                placeholder="Tu correo" className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm"
              />
              <textarea
                rows={3} value={form.mensaje} onChange={(e) => setForm({ ...form, mensaje: e.target.value })}
                placeholder="¿En qué te podemos ayudar?" className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm resize-none"
              />
              <button
                type="submit" disabled={enviandoForm}
                className="w-full bg-forest hover:bg-forest-light disabled:opacity-50 text-white font-black py-3 rounded-xl uppercase text-xs tracking-wide transition-all flex items-center justify-center gap-2"
              >
                {enviandoForm ? <Loader2 className="animate-spin" size={16} /> : <Send size={14} />}
                Enviar mensaje
              </button>
              <p className="text-[10px] text-slate-400 text-center">
                Al enviar aceptas nuestro{' '}
                <Link to="/aviso-privacidad-pasitos" target="_blank" rel="noreferrer" className="underline hover:text-brand-600">
                  Aviso de Privacidad
                </Link>.
              </p>
            </form>
          ) : (
            <>
              {/* HILO DE MENSAJES */}
              <div className="flex-1 overflow-y-auto p-4 space-y-3 bg-slate-50">
                {cargando ? (
                  <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
                ) : mensajes.length === 0 ? (
                  <div className="py-10 text-center text-slate-400 font-bold text-xs px-4">
                    Todavía no hay mensajes.<br />Escribe el tuyo abajo y te contestamos pronto.
                  </div>
                ) : (
                  mensajes.map((m, i) => (
                    <React.Fragment key={m.id}>
                      {(i === 0 || fechaLocal(m.creado_en) !== fechaLocal(mensajes[i - 1].creado_en)) && (
                        <div className="flex justify-center py-1">
                          <span className="bg-white border border-slate-200 text-slate-400 text-[9px] font-black uppercase tracking-widest px-3 py-1.5 rounded-full shadow-sm">
                            {separadorFecha(m.creado_en)}
                          </span>
                        </div>
                      )}
                      <div className={`flex flex-col ${m.es_mio ? 'items-end' : 'items-start'}`}>
                        {/* "Chat de soporte con RAG": el asistente contesta solo antes de
                            avisarle al dueño de la plataforma (ver ia_soporte.go en el
                            backend) -- esta etiqueta deja claro que esa respuesta no la
                            escribió una persona todavía, sin que se sienta como el resto
                            del chat humano. */}
                        {!m.es_mio && m.generado_por_ia && (
                          <span className="text-[9px] font-black uppercase tracking-widest text-forest/70 mb-0.5 ml-1">🤖 Respuesta automática</span>
                        )}
                        <div className={`max-w-[85%] px-3.5 py-2.5 rounded-2xl ${m.es_mio ? 'bg-forest text-white rounded-br-md' : 'bg-white border border-slate-200 text-slate-700 rounded-bl-md shadow-sm'}`}>
                          <p className="text-sm font-medium whitespace-pre-wrap">{m.contenido}</p>
                          <p className={`text-[9px] font-bold uppercase mt-1 ${m.es_mio ? 'text-white/60' : 'text-slate-400'}`}>{formatoHora(m.creado_en)}</p>
                        </div>
                      </div>
                    </React.Fragment>
                  ))
                )}
                <div ref={finRef} />
              </div>

              {/* COMPOSER */}
              <div className="p-3 border-t border-slate-100 bg-white shrink-0">
                <div className="bg-slate-50 border border-slate-200 rounded-2xl p-1.5 flex items-end gap-2">
                  <textarea
                    rows={1} value={texto} onChange={(e) => setTexto(e.target.value)} onKeyDown={alPresionarTecla}
                    placeholder="Escribe un mensaje..." className="flex-1 bg-transparent p-2 outline-none text-sm font-medium resize-none"
                  />
                  <button
                    onClick={enviar} disabled={enviando || !texto.trim()}
                    className="bg-forest hover:bg-forest-light disabled:opacity-50 text-white p-2.5 rounded-xl shadow-md transition-all active:scale-95 shrink-0"
                  >
                    {enviando ? <Loader2 className="animate-spin" size={16} /> : <Send size={16} />}
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      )}
    </>
  );
};

export default SoporteChat;
