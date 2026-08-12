import React, { useState, useEffect, useRef } from 'react';
import api from './axiosConfig';
import {
  MessageCircle, ArrowLeft, Send, Loader2, User, CircleDot,
} from 'lucide-react';
import { mostrarError } from './utils/alertas';

const INTERVALO_POLLING_MS = 5000;

// PanelChat -- inbox de staff para el "Chat privado padres↔maestros". No hay
// asignación de un maestro por niño en el modelo actual, así que es una sola
// conversación compartida por familia: cualquier staff/admin puede leer y
// responder a cualquiera (mismo criterio que Circulares, en espejo: allá es
// un solo emisor hacia todos los padres, aquí son todos los que atienden
// hacia un padre a la vez).
const PanelChat = () => {
  const [conversaciones, setConversaciones] = useState([]);
  const [loading, setLoading] = useState(true);
  const [seleccionada, setSeleccionada] = useState(null);

  const cargarConversaciones = async () => {
    try {
      const res = await api.get('/chat/conversaciones');
      setConversaciones(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar conversaciones:', err);
      mostrarError('No se pudieron cargar las conversaciones');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargarConversaciones(); }, []);

  if (seleccionada) {
    return (
      <HiloChat
        padreId={seleccionada.padre_id}
        nombre={seleccionada.nombre}
        onVolver={() => { setSeleccionada(null); cargarConversaciones(); }}
      />
    );
  }

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex items-center gap-4 mb-8">
          <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><MessageCircle size={28} /></div>
          <div>
            <h3 className="text-xl font-black uppercase text-slate-900">Chat con Familias</h3>
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Mensajes privados, sin apps de terceros</p>
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : conversaciones.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Ninguna familia ha escrito todavía</div>
        ) : (
          <div className="space-y-3">
            {conversaciones.map((c) => (
              <button
                key={c.padre_id}
                onClick={() => setSeleccionada(c)}
                className="w-full text-left bg-slate-50 border border-slate-100 hover:border-brand-300 hover:shadow-md p-5 rounded-[1.75rem] transition-all active:scale-[0.98] flex items-center gap-4"
              >
                <div className="bg-white p-3 rounded-2xl text-slate-300 border border-slate-100 shrink-0"><User size={22} /></div>
                <div className="min-w-0 flex-1">
                  <p className="font-black uppercase text-sm text-slate-900 truncate">{c.nombre}</p>
                  <p className="text-xs text-slate-500 font-medium truncate">{c.ultimo_mensaje}</p>
                </div>
                {c.no_leidos > 0 && (
                  <span className="shrink-0 flex items-center gap-1 bg-brand-600 text-white text-[10px] font-black px-2.5 py-1 rounded-full">
                    <CircleDot size={10} /> {c.no_leidos}
                  </span>
                )}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

const HiloChat = ({ padreId, nombre, onVolver }) => {
  const [mensajes, setMensajes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [texto, setTexto] = useState('');
  const [enviando, setEnviando] = useState(false);
  const finRef = useRef(null);

  const cargarMensajes = async (mostrarLoading) => {
    if (mostrarLoading) setLoading(true);
    try {
      const res = await api.get(`/chat/${padreId}/mensajes`);
      setMensajes(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar mensajes:', err);
      if (mostrarLoading) mostrarError('No se pudo cargar la conversación');
    } finally {
      if (mostrarLoading) setLoading(false);
    }
  };

  useEffect(() => {
    cargarMensajes(true);
    const intervalo = setInterval(() => cargarMensajes(false), INTERVALO_POLLING_MS);
    return () => clearInterval(intervalo);
  }, [padreId]);

  useEffect(() => {
    finRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [mensajes]);

  const enviar = async () => {
    const contenido = texto.trim();
    if (!contenido) return;
    setEnviando(true);
    try {
      await api.post(`/chat/${padreId}/mensajes`, { contenido });
      setTexto('');
      await cargarMensajes(false);
    } catch (err) {
      console.error('Error al enviar mensaje:', err);
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
    <div className="animate-in fade-in duration-500">
      <button onClick={onVolver} className="mb-6 flex items-center gap-2 text-brand-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
        <ArrowLeft size={16} /> Volver a conversaciones
      </button>

      <div className="bg-white rounded-[2.5rem] border border-slate-200 shadow-xl flex flex-col h-[75vh] overflow-hidden">
        <div className="p-6 border-b border-slate-100 flex items-center gap-4 shrink-0">
          <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><User size={22} /></div>
          <h3 className="text-lg font-black uppercase text-slate-900">{nombre}</h3>
        </div>

        <div className="flex-1 overflow-y-auto p-6 space-y-3 bg-slate-50">
          {loading ? (
            <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
          ) : mensajes.length === 0 ? (
            <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Todavía no hay mensajes</div>
          ) : (
            mensajes.map((m) => (
              <div key={m.id} className={`flex ${m.es_mio ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[75%] px-4 py-3 rounded-2xl ${m.es_mio ? 'bg-brand-600 text-white rounded-br-md' : 'bg-white border border-slate-100 text-slate-700 rounded-bl-md'}`}>
                  <p className="text-sm font-medium whitespace-pre-wrap">{m.contenido}</p>
                  <p className={`text-[9px] font-bold uppercase mt-1 ${m.es_mio ? 'text-brand-100' : 'text-slate-400'}`}>{formatoHora(m.creado_en)}</p>
                </div>
              </div>
            ))
          )}
          <div ref={finRef} />
        </div>

        <div className="p-4 border-t border-slate-100 flex items-end gap-3 shrink-0">
          <textarea
            rows={1}
            value={texto}
            onChange={(e) => setTexto(e.target.value)}
            onKeyDown={alPresionarTecla}
            placeholder="Escribe un mensaje..."
            className="flex-1 bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium resize-none"
          />
          <button
            onClick={enviar}
            disabled={enviando || !texto.trim()}
            className="bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white p-3.5 rounded-xl shadow-md transition-all active:scale-95 shrink-0"
          >
            {enviando ? <Loader2 className="animate-spin" size={18} /> : <Send size={18} />}
          </button>
        </div>
      </div>
    </div>
  );
};

export default PanelChat;
