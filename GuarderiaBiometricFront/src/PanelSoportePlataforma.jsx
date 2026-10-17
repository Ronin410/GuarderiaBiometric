import React, { useState, useEffect, useCallback, useRef } from 'react';
import axios from 'axios';
import { ChevronLeft, Send, Loader2, LifeBuoy, Baby, Users, Mail } from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { fechaLocal, separadorFecha } from './utils/fecha';

const INTERVALO_POLLING_MS = 8000;

const ETIQUETA_TIPO = {
  papa: { texto: 'Papá', Icon: Baby, color: 'bg-brand-100 text-brand-700' },
  staff: { texto: 'Staff / Admin', Icon: Users, color: 'bg-forest/10 text-forest' },
  prospecto: { texto: 'Prospecto', Icon: Mail, color: 'bg-amber-100 text-amber-700' },
};

// PanelSoportePlataforma -- pestaña "Soporte" dentro de /plataforma (ver
// PanelPlataforma.jsx): el inbox donde Alejandro ve y responde el chat de
// soporte de papás, staff/admin de cualquier guardería, y prospectos sin
// cuenta (ver chat_soporte.go en el backend). Misma autenticación que el
// resto de /plataforma: header X-Platform-Key, no el login normal.
const PanelSoportePlataforma = ({ apiUrl, platformKey }) => {
  const [conversaciones, setConversaciones] = useState(null);
  const [cargando, setCargando] = useState(false);
  const [conversacionActiva, setConversacionActiva] = useState(null);

  const cabeceras = { headers: { 'X-Platform-Key': platformKey } };

  const cargarConversaciones = useCallback(async (mostrarLoading) => {
    if (mostrarLoading) setCargando(true);
    try {
      const res = await axios.get(`${apiUrl}/plataforma/soporte/conversaciones`, cabeceras);
      setConversaciones(Array.isArray(res.data) ? res.data : []);
    } catch {
      if (mostrarLoading) mostrarError('No se pudo cargar el inbox de soporte.');
    } finally {
      if (mostrarLoading) setCargando(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiUrl, platformKey]);

  useEffect(() => {
    cargarConversaciones(true);
    const intervalo = setInterval(() => cargarConversaciones(false), INTERVALO_POLLING_MS);
    return () => clearInterval(intervalo);
  }, [cargarConversaciones]);

  if (conversacionActiva) {
    return (
      <HiloSoportePlataforma
        apiUrl={apiUrl}
        platformKey={platformKey}
        conversacion={conversacionActiva}
        onVolver={() => { setConversacionActiva(null); cargarConversaciones(false); }}
      />
    );
  }

  const formatoFecha = (iso) => {
    if (!iso) return '';
    try {
      return new Date(iso).toLocaleString('es-MX', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
    } catch {
      return iso;
    }
  };

  return (
    <div className="space-y-3">
      {cargando && conversaciones === null ? (
        <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
          Cargando...
        </div>
      ) : conversaciones?.length === 0 ? (
        <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
          Todavía no hay conversaciones de soporte
        </div>
      ) : (
        conversaciones?.map((conv) => {
          const etiqueta = ETIQUETA_TIPO[conv.tipo] || ETIQUETA_TIPO.prospecto;
          return (
            <button
              key={conv.id}
              onClick={() => setConversacionActiva(conv)}
              className="w-full text-left bg-white p-5 rounded-[2rem] border border-slate-200 shadow-sm hover:shadow-md transition-all flex items-start gap-4"
            >
              <div className={`p-2.5 rounded-xl shrink-0 ${etiqueta.color}`}><etiqueta.Icon size={18} /></div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <p className="font-black text-slate-900 text-sm truncate">{conv.nombre}</p>
                  <span className="text-[9px] font-black uppercase text-slate-400 tracking-widest shrink-0">{formatoFecha(conv.actualizado_en)}</span>
                </div>
                <p className="text-[10px] font-black uppercase text-slate-400 tracking-widest mt-0.5">
                  {etiqueta.texto}{conv.guarderia_nombre ? ` · ${conv.guarderia_nombre}` : ''}{conv.email ? ` · ${conv.email}` : ''}
                </p>
                <p className="text-xs text-slate-500 mt-1.5 truncate">{conv.ultimo_mensaje}</p>
              </div>
              {conv.no_leidos > 0 && (
                <span className="bg-rose-500 text-white text-[10px] font-black w-5 h-5 rounded-full flex items-center justify-center shrink-0">
                  {conv.no_leidos > 9 ? '9+' : conv.no_leidos}
                </span>
              )}
            </button>
          );
        })
      )}
    </div>
  );
};

// HiloSoportePlataforma -- el hilo de mensajes de UNA conversación,
// separado del inbox por el mismo motivo que HiloChat en ChatPadre.jsx: no
// arrastrar estado (texto a medio escribir) entre una conversación y otra.
const HiloSoportePlataforma = ({ apiUrl, platformKey, conversacion, onVolver }) => {
  const [mensajes, setMensajes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [texto, setTexto] = useState('');
  const [enviando, setEnviando] = useState(false);
  const finRef = useRef(null);

  const cabeceras = { headers: { 'X-Platform-Key': platformKey } };

  const cargarMensajes = useCallback(async (mostrarLoading) => {
    if (mostrarLoading) setLoading(true);
    try {
      const res = await axios.get(`${apiUrl}/plataforma/soporte/${conversacion.id}/mensajes`, cabeceras);
      setMensajes(Array.isArray(res.data) ? res.data : []);
    } catch {
      if (mostrarLoading) mostrarError('No se pudo cargar la conversación');
    } finally {
      if (mostrarLoading) setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiUrl, platformKey, conversacion.id]);

  useEffect(() => {
    cargarMensajes(true);
    const intervalo = setInterval(() => cargarMensajes(false), INTERVALO_POLLING_MS);
    return () => clearInterval(intervalo);
  }, [cargarMensajes]);

  useEffect(() => {
    finRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [mensajes]);

  const enviar = async () => {
    const contenido = texto.trim();
    if (!contenido) return;
    setEnviando(true);
    try {
      await axios.post(`${apiUrl}/plataforma/soporte/${conversacion.id}/mensajes`, { contenido }, cabeceras);
      setTexto('');
      await cargarMensajes(false);
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo enviar la respuesta');
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

  const etiqueta = ETIQUETA_TIPO[conversacion.tipo] || ETIQUETA_TIPO.prospecto;

  return (
    <div className="bg-white rounded-[2.5rem] border border-slate-200 shadow-sm overflow-hidden flex flex-col h-[70vh]">
      <div className="p-5 border-b border-slate-100 shrink-0">
        <button onClick={onVolver} className="flex items-center gap-2 text-slate-400 font-black uppercase text-[10px] tracking-widest mb-3 hover:text-brand-600 transition-colors">
          <ChevronLeft size={16} /> Inbox
        </button>
        <div className="flex items-center gap-3">
          <div className={`p-2.5 rounded-xl shrink-0 ${etiqueta.color}`}><etiqueta.Icon size={18} /></div>
          <div className="min-w-0">
            <p className="font-black text-slate-900 truncate">{conversacion.nombre}</p>
            <p className="text-[10px] font-black uppercase text-slate-400 tracking-widest">
              {etiqueta.texto}{conversacion.guarderia_nombre ? ` · ${conversacion.guarderia_nombre}` : ''}{conversacion.email ? ` · ${conversacion.email}` : ''}
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-5 space-y-3 bg-slate-50">
        {loading ? (
          <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : mensajes.length === 0 ? (
          <div className="py-10 text-center text-slate-400 font-bold text-xs">Todavía no hay mensajes en esta conversación.</div>
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
                    avisarte a ti -- esta etiqueta distingue esas respuestas (se ven
                    como "tuyas", autor_rol = 'plataforma') de las que escribiste en
                    persona. Ver ia_soporte.go en el backend. */}
                {m.es_mio && m.generado_por_ia && (
                  <span className="text-[9px] font-black uppercase tracking-widest text-forest/70 mr-1">🤖 Respondió el asistente</span>
                )}
                <div className={`max-w-[75%] px-4 py-2.5 rounded-2xl ${m.es_mio ? 'bg-forest text-white rounded-br-md' : 'bg-white border border-slate-200 text-slate-700 rounded-bl-md shadow-sm'}`}>
                  <p className="text-sm font-medium whitespace-pre-wrap">{m.contenido}</p>
                  <p className={`text-[9px] font-bold uppercase mt-1 ${m.es_mio ? 'text-white/60' : 'text-slate-400'}`}>{formatoHora(m.creado_en)}</p>
                </div>
              </div>
            </React.Fragment>
          ))
        )}
        <div ref={finRef} />
      </div>

      <div className="p-3 border-t border-slate-100 shrink-0">
        <div className="bg-slate-50 border border-slate-200 rounded-2xl p-1.5 flex items-end gap-2">
          <textarea
            rows={1} value={texto} onChange={(e) => setTexto(e.target.value)} onKeyDown={alPresionarTecla}
            placeholder="Escribe tu respuesta..." className="flex-1 bg-transparent p-2.5 outline-none text-sm font-medium resize-none"
          />
          <button
            onClick={enviar} disabled={enviando || !texto.trim()}
            className="bg-forest hover:bg-forest-light disabled:opacity-50 text-white p-2.5 rounded-xl shadow-md transition-all active:scale-95 shrink-0"
          >
            {enviando ? <Loader2 className="animate-spin" size={16} /> : <Send size={16} />}
          </button>
        </div>
      </div>
    </div>
  );
};

export default PanelSoportePlataforma;
