import React, { useState, useEffect, useRef } from 'react';
import api from './axiosConfig';
import { ChevronLeft, Send, Loader2, MessageCircle, Paperclip, X, FileText, Download, ShieldCheck, User } from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { fechaLocal, separadorFecha } from './utils/fecha';

const INTERVALO_POLLING_MS = 5000;

// ChatPadre -- vista de pantalla completa (mismo patrón que
// VistaPadreDetalle: swap desde DashboardPadre, no una ruta ni un modal)
// para el chat privado. "Quiero que al papá le aparezcan los staff o
// administradores de la guardería... para escoger con quién hablar" -- ya
// no es una sola conversación con "la guardería" en general, así que esto
// primero muestra un selector de contactos y solo entra al hilo de
// mensajes una vez que el papá elige con quién.
const ChatPadre = ({ onVolver }) => {
  const [contactos, setContactos] = useState(null);
  const [cargandoContactos, setCargandoContactos] = useState(true);
  const [contacto, setContacto] = useState(null);

  useEffect(() => {
    api.get('/padre/chat/contactos')
      .then((res) => setContactos(Array.isArray(res.data) ? res.data : []))
      .catch((err) => {
        console.error('Error al cargar el staff de la guardería:', err);
        mostrarError('No se pudo cargar la lista de contactos');
        setContactos([]);
      })
      .finally(() => setCargandoContactos(false));
  }, []);

  if (contacto) {
    return <HiloChat contacto={contacto} onVolver={() => setContacto(null)} />;
  }

  return (
    <div className="min-h-screen bg-slate-50 pb-10 animate-in fade-in duration-500">
      <div className="bg-white p-6 pb-8 rounded-b-[3rem] shadow-sm border-b border-slate-100 sticky top-0 z-30">
        <button
          onClick={onVolver}
          className="flex items-center gap-2 text-slate-400 font-black uppercase text-[10px] tracking-widest mb-6 hover:text-brand-600 transition-colors"
        >
          <ChevronLeft size={16} /> Volver
        </button>
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter">Chat</h2>
          <div className="bg-brand-600 p-3 rounded-2xl text-white shadow-lg shadow-brand-200">
            <MessageCircle size={20} />
          </div>
        </div>
        <p className="text-xs text-slate-400 font-bold mt-3">¿Con quién quieres hablar?</p>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-3">
        {cargandoContactos ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : contactos.length === 0 ? (
          <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center">
            <MessageCircle size={40} className="mx-auto text-slate-200 mb-4" />
            <p className="text-slate-400 font-bold uppercase text-[10px]">Tu guardería todavía no tiene personal disponible para chat</p>
          </div>
        ) : (
          contactos.map((ct) => (
            <button
              key={ct.id}
              onClick={() => setContacto(ct)}
              className="w-full bg-white p-4 rounded-[2rem] border border-slate-100 shadow-sm flex items-center gap-4 hover:shadow-md transition-all active:scale-[0.98]"
            >
              <div className="bg-brand-100 p-3 rounded-2xl text-brand-600 shrink-0">
                {ct.rol === 'admin' ? <ShieldCheck size={20} /> : <User size={20} />}
              </div>
              <div className="flex-1 text-left min-w-0">
                <p className="text-[13px] font-black text-slate-900 uppercase leading-tight truncate">{ct.nombre}</p>
                <p className="text-[9px] text-slate-400 font-bold uppercase tracking-wide">{ct.rol === 'admin' ? 'Administración' : 'Personal'}</p>
              </div>
            </button>
          ))
        )}
      </div>
    </div>
  );
};

// HiloChat -- el hilo de mensajes con UN contacto en particular. Antes era
// todo ChatPadre; se separa para que el selector de arriba pueda montarlo y
// desmontarlo por contacto sin arrastrar el estado de una conversación a la
// siguiente (mensajes, texto a medio escribir, adjunto elegido, etc.).
const HiloChat = ({ contacto, onVolver }) => {
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
      const res = await api.get(`/padre/chat/${contacto.id}`);
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
  }, [contacto.id]);

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
      await api.post(`/padre/chat/${contacto.id}`, data);
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
    <div className="min-h-screen bg-slate-50 pb-6 animate-in fade-in duration-500 flex flex-col">
      {/* HEADER */}
      <div className="bg-white p-6 pb-8 rounded-b-[3rem] shadow-sm border-b border-slate-100 sticky top-0 z-30 shrink-0">
        <button
          onClick={onVolver}
          className="flex items-center gap-2 text-slate-400 font-black uppercase text-[10px] tracking-widest mb-6 hover:text-brand-600 transition-colors"
        >
          <ChevronLeft size={16} /> Contactos
        </button>

        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter truncate">{contacto.nombre}</h2>
          <div className="bg-brand-600 p-3 rounded-2xl text-white shadow-lg shadow-brand-200 shrink-0">
            <MessageCircle size={20} />
          </div>
        </div>
      </div>

      {/* HILO DE MENSAJES */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-3 max-w-md w-full mx-auto">
        {loading ? (
          <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : mensajes.length === 0 ? (
          <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">
            Todavía no hay mensajes.<br />Escribe el primero abajo.
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
              <div className={`flex ${m.es_mio ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[80%] px-4 py-3 rounded-2xl ${m.es_mio ? 'bg-brand-600 text-white rounded-br-md' : 'bg-white border border-slate-100 text-slate-700 rounded-bl-md shadow-sm'}`}>
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

      {/* BARRA DE ENVÍO */}
      <div className="sticky bottom-0 bg-slate-50 pt-2 px-4 sm:px-6 pb-4 max-w-md w-full mx-auto shrink-0">
        {archivo && (
          <div className="bg-white border border-slate-200 rounded-2xl shadow-md p-3 mb-2 flex items-center gap-3">
            {previewArchivo ? (
              <img src={previewArchivo} alt="preview" className="w-12 h-12 rounded-lg object-cover shrink-0" />
            ) : (
              <div className="w-12 h-12 rounded-lg bg-slate-100 text-slate-400 flex items-center justify-center shrink-0"><FileText size={20} /></div>
            )}
            <span className="text-xs font-bold text-slate-600 truncate flex-1">{archivo.name}</span>
            <button onClick={quitarArchivo} className="text-slate-400 hover:text-rose-500 p-1 shrink-0"><X size={16} /></button>
          </div>
        )}
        <div className="bg-white border border-slate-200 rounded-2xl shadow-md p-2 flex items-end gap-2">
          <input
            ref={inputArchivoRef}
            type="file"
            accept="image/*,application/pdf"
            className="hidden"
            onChange={(e) => elegirArchivo(e.target.files?.[0])}
          />
          <button
            onClick={() => inputArchivoRef.current?.click()}
            className="text-slate-400 hover:text-brand-600 p-2.5 rounded-xl shrink-0 transition-colors"
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
            className="flex-1 bg-transparent p-2 outline-none text-sm font-medium resize-none"
          />
          <button
            onClick={enviar}
            disabled={enviando || (!texto.trim() && !archivo)}
            className="bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white p-3 rounded-xl shadow-md transition-all active:scale-95 shrink-0"
          >
            {enviando ? <Loader2 className="animate-spin" size={18} /> : <Send size={18} />}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ChatPadre;
