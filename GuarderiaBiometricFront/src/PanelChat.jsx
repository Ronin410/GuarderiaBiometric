import React, { useState, useEffect, useRef } from 'react';
import api from './axiosConfig';
import {
  MessageCircle, ArrowLeft, Send, Loader2, User, CircleDot,
  Paperclip, X, FileText, Download, Plus, Search,
} from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { fechaLocal, separadorFecha } from './utils/fecha';

const INTERVALO_POLLING_MS = 5000;

// PanelChat -- inbox de staff para el "Chat privado padres↔guardería".
// "Quiero que al papá le aparezcan los staff o administradores... para
// escoger con quién hablar" -- ya no es una sola conversación compartida
// por familia: cada quien ve y responde solo las conversaciones que un
// papá le dirigió a él/ella. El admin, además, ve TODAS (supervisión) --
// por eso las conversaciones de otros traen personal_nombre, para que sepa
// de quién es cada una; un staff normal solo ve las suyas, así que ese
// dato no le hace falta (viene vacío).
//
// usuarioActualId identifica al staff/admin que tiene la sesión abierta --
// lo necesita el flujo de "Nueva conversación" ("quiero que la guardería
// también escoja con qué papá hablar aunque nunca hayan hablado"): un hilo
// nuevo se abre como HiloChat con personalId = usuarioActualId, sin esperar
// a que exista ya un mensaje previo en /chat/conversaciones.
const PanelChat = ({ usuarioActualId }) => {
  const [conversaciones, setConversaciones] = useState([]);
  const [loading, setLoading] = useState(true);
  const [seleccionada, setSeleccionada] = useState(null);
  const [mostrarSelectorFamilia, setMostrarSelectorFamilia] = useState(false);

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
        personalId={seleccionada.personal_id}
        nombre={seleccionada.nombre}
        onVolver={() => { setSeleccionada(null); cargarConversaciones(); }}
      />
    );
  }

  if (mostrarSelectorFamilia) {
    return (
      <SelectorFamilia
        onElegir={(familia) => {
          setMostrarSelectorFamilia(false);
          setSeleccionada({ padre_id: familia.id, personal_id: usuarioActualId, nombre: familia.nombre });
        }}
        onCancelar={() => setMostrarSelectorFamilia(false)}
      />
    );
  }

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex items-center justify-between gap-4 mb-8">
          <div className="flex items-center gap-4 min-w-0">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600 shrink-0"><MessageCircle size={28} /></div>
            <div className="min-w-0">
              <h3 className="text-xl font-black uppercase text-slate-900">Chat con Familias</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Mensajes privados, sin apps de terceros</p>
            </div>
          </div>
          <button
            onClick={() => setMostrarSelectorFamilia(true)}
            className="shrink-0 flex items-center gap-2 bg-brand-600 hover:bg-brand-700 text-white text-xs font-black uppercase tracking-widest px-4 py-3 rounded-2xl shadow-md transition-all active:scale-95"
          >
            <Plus size={16} /> Nueva conversación
          </button>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : conversaciones.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Ninguna familia ha escrito todavía</div>
        ) : (
          <div className="space-y-3">
            {conversaciones.map((c) => (
              <button
                key={`${c.padre_id}-${c.personal_id}`}
                onClick={() => setSeleccionada(c)}
                className="w-full text-left bg-slate-50 border border-slate-100 hover:border-brand-300 hover:shadow-md p-5 rounded-[1.75rem] transition-all active:scale-[0.98] flex items-center gap-4"
              >
                <div className="bg-white p-3 rounded-2xl text-slate-300 border border-slate-100 shrink-0"><User size={22} /></div>
                <div className="min-w-0 flex-1">
                  <p className="font-black uppercase text-sm text-slate-900 truncate">{c.nombre}</p>
                  <p className="text-xs text-slate-500 font-medium truncate">{c.ultimo_mensaje}</p>
                  {/* Solo el admin recibe personal_nombre (ve conversaciones
                      de todos) -- un staff normal solo ve las suyas, así
                      que aquí siempre viene vacío para él. */}
                  {c.personal_nombre && (
                    <p className="text-[9px] text-brand-500 font-black uppercase tracking-widest mt-0.5">Con {c.personal_nombre}</p>
                  )}
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

// SelectorFamilia -- directorio COMPLETO de familias de la guardería (no
// solo las que ya escribieron), para arrancar una conversación desde cero.
const SelectorFamilia = ({ onElegir, onCancelar }) => {
  const [familias, setFamilias] = useState([]);
  const [loading, setLoading] = useState(true);
  const [busqueda, setBusqueda] = useState('');

  useEffect(() => {
    (async () => {
      try {
        const res = await api.get('/chat/familias');
        setFamilias(Array.isArray(res.data) ? res.data : []);
      } catch (err) {
        console.error('Error al cargar las familias:', err);
        mostrarError('No se pudieron cargar las familias');
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const filtradas = familias.filter((f) => f.nombre.toLowerCase().includes(busqueda.trim().toLowerCase()));

  return (
    <div className="animate-in fade-in duration-500">
      <button onClick={onCancelar} className="mb-6 flex items-center gap-2 text-brand-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
        <ArrowLeft size={16} /> Volver a conversaciones
      </button>

      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <h3 className="text-xl font-black uppercase text-slate-900 mb-1">Elige una familia</h3>
        <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-6">Aunque todavía no hayan escrito</p>

        <div className="relative mb-6">
          <Search size={16} className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-300" />
          <input
            type="text"
            value={busqueda}
            onChange={(e) => setBusqueda(e.target.value)}
            placeholder="Buscar familia..."
            className="w-full bg-slate-50 border border-slate-200 pl-11 pr-4 py-3 rounded-2xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium"
          />
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : filtradas.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Ninguna familia coincide</div>
        ) : (
          <div className="space-y-2 max-h-[55vh] overflow-y-auto pr-1">
            {filtradas.map((f) => (
              <button
                key={f.id}
                onClick={() => onElegir(f)}
                className="w-full text-left bg-slate-50 border border-slate-100 hover:border-brand-300 hover:shadow-md p-4 rounded-2xl transition-all active:scale-[0.98] flex items-center gap-3"
              >
                <div className="bg-white p-2.5 rounded-xl text-slate-300 border border-slate-100 shrink-0"><User size={18} /></div>
                <p className="font-black uppercase text-sm text-slate-900 truncate">{f.nombre}</p>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

const HiloChat = ({ padreId, personalId, nombre, onVolver }) => {
  const [mensajes, setMensajes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [texto, setTexto] = useState('');
  const [enviando, setEnviando] = useState(false);
  const [archivo, setArchivo] = useState(null);
  const [previewArchivo, setPreviewArchivo] = useState(null);
  const finRef = useRef(null);
  const inputArchivoRef = useRef(null);

  const cargarMensajes = async (mostrarLoading) => {
    if (mostrarLoading) setLoading(true);
    try {
      const res = await api.get(`/chat/${padreId}/${personalId}/mensajes`);
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [padreId, personalId]);

  useEffect(() => {
    finRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [mensajes]);

  const elegirArchivo = (file) => {
    if (!file) return;
    if (file.size > 10 * 1024 * 1024) {
      mostrarError('El archivo no puede pesar más de 10 MB');
      return;
    }
    setArchivo(file);
    setPreviewArchivo(file.type.startsWith('image/') ? URL.createObjectURL(file) : null);
  };

  const quitarArchivo = () => {
    setArchivo(null);
    setPreviewArchivo(null);
    if (inputArchivoRef.current) inputArchivoRef.current.value = '';
  };

  const enviar = async () => {
    const contenido = texto.trim();
    if (!contenido && !archivo) return;
    setEnviando(true);
    try {
      const data = new FormData();
      data.append('contenido', contenido);
      if (archivo) data.append('archivo', archivo);
      await api.post(`/chat/${padreId}/${personalId}/mensajes`, data);
      setTexto('');
      quitarArchivo();
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
            mensajes.map((m, i) => (
              <React.Fragment key={m.id}>
                {(i === 0 || fechaLocal(m.creado_en) !== fechaLocal(mensajes[i - 1].creado_en)) && (
                  <div className="flex justify-center py-1">
                    <span className="bg-white border border-slate-200 text-slate-400 text-[9px] font-black uppercase tracking-widest px-3 py-1.5 rounded-full shadow-sm">
                      {separadorFecha(m.creado_en)}
                    </span>
                  </div>
                )}
                <div className={`flex ${m.es_mio ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[75%] px-4 py-3 rounded-2xl ${m.es_mio ? 'bg-brand-600 text-white rounded-br-md' : 'bg-white border border-slate-100 text-slate-700 rounded-bl-md'}`}>
                    {m.adjunto_url && (
                      m.adjunto_tipo === 'imagen' ? (
                        <a href={m.adjunto_url} target="_blank" rel="noreferrer" className="block mb-2">
                          <img src={m.adjunto_url} alt={m.adjunto_nombre || 'imagen'} className="max-w-full max-h-64 rounded-xl object-cover" />
                        </a>
                      ) : (
                        <a
                          href={m.adjunto_url} target="_blank" rel="noreferrer"
                          className={`flex items-center gap-2 mb-2 px-3 py-2 rounded-xl ${m.es_mio ? 'bg-white/10' : 'bg-slate-50'}`}
                        >
                          <FileText size={18} className="shrink-0" />
                          <span className="text-xs font-bold truncate flex-1">{m.adjunto_nombre || 'Archivo'}</span>
                          <Download size={14} className="shrink-0" />
                        </a>
                      )
                    )}
                    {m.contenido && <p className="text-sm font-medium whitespace-pre-wrap">{m.contenido}</p>}
                    <p className={`text-[9px] font-bold uppercase mt-1 ${m.es_mio ? 'text-brand-100' : 'text-slate-400'}`}>{formatoHora(m.creado_en)}</p>
                  </div>
                </div>
              </React.Fragment>
            ))
          )}
          <div ref={finRef} />
        </div>

        {archivo && (
          <div className="px-4 pt-3 shrink-0">
            <div className="bg-slate-50 border border-slate-200 rounded-xl p-3 flex items-center gap-3">
              {previewArchivo ? (
                <img src={previewArchivo} alt="preview" className="w-12 h-12 rounded-lg object-cover shrink-0" />
              ) : (
                <div className="w-12 h-12 rounded-lg bg-white text-slate-400 flex items-center justify-center shrink-0"><FileText size={20} /></div>
              )}
              <span className="text-xs font-bold text-slate-600 truncate flex-1">{archivo.name}</span>
              <button onClick={quitarArchivo} className="text-slate-400 hover:text-rose-500 p-1 shrink-0"><X size={16} /></button>
            </div>
          </div>
        )}

        <div className="p-4 border-t border-slate-100 flex items-end gap-3 shrink-0">
          <input
            ref={inputArchivoRef}
            type="file"
            accept="image/*,application/pdf"
            className="hidden"
            onChange={(e) => elegirArchivo(e.target.files?.[0])}
          />
          <button
            onClick={() => inputArchivoRef.current?.click()}
            className="text-slate-400 hover:text-brand-600 p-3 rounded-xl shrink-0 transition-colors"
            title="Adjuntar imagen o archivo"
          >
            <Paperclip size={20} />
          </button>
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
            disabled={enviando || (!texto.trim() && !archivo)}
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
