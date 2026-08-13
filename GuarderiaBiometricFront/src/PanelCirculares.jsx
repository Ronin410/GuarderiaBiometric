import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import { Megaphone, Plus, X, Send, Loader2, Trash2, CalendarClock, Eye, ChevronDown, ChevronUp, CheckCheck } from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';

const FORM_VACIO = { titulo: '', contenido: '' };

// PanelCirculares — "avisos que el admin o staff pueden mandar a todos los
// padres de la guardería". Al publicar, el backend dispara una notificación
// push a todos los tutores suscritos (reutiliza la misma infraestructura ya
// usada para entradas/salidas/bitácora).
const PanelCirculares = () => {
  const [circulares, setCirculares] = useState([]);
  const [loading, setLoading] = useState(true);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState(FORM_VACIO);
  const [publicando, setPublicando] = useState(false);
  const [eliminandoId, setEliminandoId] = useState(null);
  const [detalleAbierto, setDetalleAbierto] = useState(null);
  const [lecturas, setLecturas] = useState([]);
  const [cargandoLecturas, setCargandoLecturas] = useState(false);

  const cargar = async () => {
    setLoading(true);
    try {
      const res = await api.get('/circulares');
      setCirculares(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar circulares:', err);
      mostrarError('No se pudieron cargar las circulares');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const publicar = async () => {
    if (!form.titulo.trim() || !form.contenido.trim()) {
      mostrarError('El título y el contenido son obligatorios');
      return;
    }
    setPublicando(true);
    try {
      await api.post('/circulares', form);
      mostrarExito('La circular se publicó y se notificó a los tutores suscritos');
      setForm(FORM_VACIO);
      setMostrarForm(false);
      cargar();
    } catch (err) {
      console.error('Error al publicar circular:', err);
      mostrarError(err.response?.data?.error || 'No se pudo publicar la circular');
    } finally {
      setPublicando(false);
    }
  };

  const eliminar = async (cir) => {
    const ok = await confirmar(`Se eliminará "${cir.titulo}". Ya no se podrá ver en el portal de los padres.`, '¿Eliminar circular?');
    if (!ok) return;
    setEliminandoId(cir.id);
    try {
      await api.delete(`/circulares/${cir.id}`);
      cargar();
    } catch (err) {
      console.error('Error al eliminar circular:', err);
      mostrarError('No se pudo eliminar la circular');
    } finally {
      setEliminandoId(null);
    }
  };

  const alternarDetalle = async (cir) => {
    if (detalleAbierto === cir.id) {
      setDetalleAbierto(null);
      return;
    }
    setDetalleAbierto(cir.id);
    setCargandoLecturas(true);
    try {
      const res = await api.get(`/circulares/${cir.id}/lecturas`);
      setLecturas(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar el detalle de lecturas:', err);
      mostrarError('No se pudo cargar el detalle de lecturas');
      setLecturas([]);
    } finally {
      setCargandoLecturas(false);
    }
  };

  const formatoFecha = (iso) => {
    try {
      return new Date(iso).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch {
      return iso;
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><Megaphone size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Circulares</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Avisos para todos los padres</p>
            </div>
          </div>
          <button
            onClick={() => { setMostrarForm(!mostrarForm); setForm(FORM_VACIO); }}
            className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95"
          >
            {mostrarForm ? <X size={14} /> : <Plus size={14} />}
            {mostrarForm ? 'Cancelar' : 'Nueva Circular'}
          </button>
        </div>

        {mostrarForm && (
          <div className="mb-8 p-6 rounded-[2rem] border border-dashed border-brand-300 bg-brand-50/40 space-y-4">
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Título</label>
              <input
                type="text" value={form.titulo} onChange={(e) => setForm({ ...form, titulo: e.target.value })}
                placeholder="ej. Suspensión de clases el viernes"
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold"
              />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Contenido</label>
              <textarea
                rows={4} value={form.contenido} onChange={(e) => setForm({ ...form, contenido: e.target.value })}
                placeholder="Escribe el aviso completo..."
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium resize-y"
              />
            </div>
            <div className="flex justify-end">
              <button
                onClick={publicar}
                disabled={publicando}
                className="flex items-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-3 rounded-xl shadow-md transition-all active:scale-95"
              >
                {publicando ? <Loader2 className="animate-spin" size={16} /> : <Send size={16} />} Publicar y Notificar
              </button>
            </div>
          </div>
        )}

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : circulares.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin circulares publicadas</div>
        ) : (
          <div className="space-y-4">
            {circulares.map((cir) => (
              <div key={cir.id} className="p-6 rounded-[2rem] border border-slate-100 bg-slate-50">
                <div className="flex justify-between items-start gap-4">
                  <div className="min-w-0">
                    <p className="font-black text-lg uppercase tracking-tight text-slate-900">{cir.titulo}</p>
                    <p className="text-[10px] text-brand-500 font-bold uppercase flex items-center gap-1 mt-1">
                      <CalendarClock size={12} /> {formatoFecha(cir.creado_en)}
                    </p>
                  </div>
                  <button
                    onClick={() => eliminar(cir)}
                    disabled={eliminandoId === cir.id}
                    className="text-rose-400 hover:text-rose-600 disabled:opacity-50 p-2 shrink-0"
                    title="Eliminar circular"
                  >
                    {eliminandoId === cir.id ? <Loader2 className="animate-spin" size={16} /> : <Trash2 size={16} />}
                  </button>
                </div>
                <p className="mt-3 text-sm text-slate-600 font-medium whitespace-pre-wrap">{cir.contenido}</p>

                <button
                  onClick={() => alternarDetalle(cir)}
                  className="mt-4 flex items-center gap-1.5 text-[10px] font-black uppercase text-slate-500 hover:text-brand-600 transition-colors"
                >
                  <Eye size={12} />
                  {cir.leido_por} de {cir.total_familias} familias la han leído
                  {detalleAbierto === cir.id ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                </button>

                {detalleAbierto === cir.id && (
                  <div className="mt-3 pt-3 border-t border-dashed border-slate-200">
                    {cargandoLecturas ? (
                      <p className="text-[10px] font-black text-slate-400 uppercase">Cargando...</p>
                    ) : lecturas.length === 0 ? (
                      <p className="text-[10px] font-black text-slate-400 uppercase">Ninguna familia la ha leído todavía</p>
                    ) : (
                      <ul className="space-y-1.5">
                        {lecturas.map((l, i) => (
                          <li key={i} className="flex items-center gap-2 text-xs font-bold text-slate-600">
                            <CheckCheck size={13} className="text-emerald-500 shrink-0" />
                            {l.nombre}
                            <span className="text-slate-300 font-medium">· {formatoFecha(l.leido_en)}</span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelCirculares;
