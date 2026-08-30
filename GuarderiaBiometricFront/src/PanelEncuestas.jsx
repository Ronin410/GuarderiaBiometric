import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import {
  ClipboardCheck, Plus, X, Send, Loader2, Trash2, ArrowLeft, Lock, CheckCircle2, ListChecks,
} from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';

const PREGUNTA_VACIA = { texto: '', tipo: 'opcion_multiple', opciones: ['', ''] };
const FORM_VACIO = { titulo: '', descripcion: '', preguntas: [{ ...PREGUNTA_VACIA }] };

// PanelEncuestas -- "Encuestas para familias" del PDF de referencia (lado
// staff): crear cuestionarios (opción múltiple o texto libre) y ver los
// resultados agregados conforme van respondiendo los padres.
const PanelEncuestas = () => {
  const [encuestas, setEncuestas] = useState([]);
  const [loading, setLoading] = useState(true);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState(FORM_VACIO);
  const [publicando, setPublicando] = useState(false);
  const [seleccionada, setSeleccionada] = useState(null);

  const cargar = async () => {
    setLoading(true);
    try {
      const res = await api.get('/encuestas');
      setEncuestas(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar encuestas:', err);
      mostrarError('No se pudieron cargar las encuestas');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const actualizarPregunta = (i, campo, valor) => {
    const preguntas = [...form.preguntas];
    preguntas[i] = { ...preguntas[i], [campo]: valor };
    setForm({ ...form, preguntas });
  };

  const actualizarOpcion = (i, j, valor) => {
    const preguntas = [...form.preguntas];
    const opciones = [...preguntas[i].opciones];
    opciones[j] = valor;
    preguntas[i] = { ...preguntas[i], opciones };
    setForm({ ...form, preguntas });
  };

  const agregarOpcion = (i) => {
    const preguntas = [...form.preguntas];
    preguntas[i] = { ...preguntas[i], opciones: [...preguntas[i].opciones, ''] };
    setForm({ ...form, preguntas });
  };

  const quitarOpcion = (i, j) => {
    const preguntas = [...form.preguntas];
    preguntas[i] = { ...preguntas[i], opciones: preguntas[i].opciones.filter((_, k) => k !== j) };
    setForm({ ...form, preguntas });
  };

  const agregarPregunta = () => setForm({ ...form, preguntas: [...form.preguntas, { ...PREGUNTA_VACIA }] });
  const quitarPregunta = (i) => setForm({ ...form, preguntas: form.preguntas.filter((_, k) => k !== i) });

  const publicar = async () => {
    if (!form.titulo.trim()) {
      mostrarError('El título es obligatorio');
      return;
    }
    for (const p of form.preguntas) {
      if (!p.texto.trim()) {
        mostrarError('Todas las preguntas necesitan texto');
        return;
      }
      if (p.tipo === 'opcion_multiple' && p.opciones.filter((o) => o.trim()).length < 2) {
        mostrarError('Cada pregunta de opción múltiple necesita al menos 2 opciones');
        return;
      }
    }
    setPublicando(true);
    try {
      await api.post('/encuestas', {
        titulo: form.titulo,
        descripcion: form.descripcion,
        preguntas: form.preguntas.map((p) => ({
          texto: p.texto,
          tipo: p.tipo,
          opciones: p.tipo === 'opcion_multiple' ? p.opciones.filter((o) => o.trim()) : undefined,
        })),
      });
      mostrarExito('La encuesta se publicó y se notificó a los tutores suscritos');
      setForm(FORM_VACIO);
      setMostrarForm(false);
      cargar();
    } catch (err) {
      console.error('Error al publicar la encuesta:', err);
      mostrarError(err.response?.data?.error || 'No se pudo publicar la encuesta');
    } finally {
      setPublicando(false);
    }
  };

  if (seleccionada) {
    return <DetalleEncuesta id={seleccionada} onVolver={() => { setSeleccionada(null); cargar(); }} />;
  }

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><ClipboardCheck size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Encuestas</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Pregúntale algo a todas las familias</p>
            </div>
          </div>
          <button
            onClick={() => { setMostrarForm(!mostrarForm); setForm(FORM_VACIO); }}
            className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95"
          >
            {mostrarForm ? <X size={14} /> : <Plus size={14} />}
            {mostrarForm ? 'Cancelar' : 'Nueva Encuesta'}
          </button>
        </div>

        {mostrarForm && (
          <div className="mb-8 p-6 rounded-[2rem] border border-dashed border-brand-300 bg-brand-50/40 space-y-5">
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Título</label>
              <input
                type="text" value={form.titulo} onChange={(e) => setForm({ ...form, titulo: e.target.value })}
                placeholder="ej. Posada navideña"
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold"
              />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Descripción (opcional)</label>
              <textarea
                rows={2} value={form.descripcion} onChange={(e) => setForm({ ...form, descripcion: e.target.value })}
                placeholder="Contexto adicional para las familias..."
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium resize-y"
              />
            </div>

            <div className="space-y-4">
              {form.preguntas.map((p, i) => (
                <div key={i} className="bg-white border border-slate-200 rounded-2xl p-4 space-y-3">
                  <div className="flex items-center gap-2">
                    <input
                      type="text" value={p.texto} onChange={(e) => actualizarPregunta(i, 'texto', e.target.value)}
                      placeholder={`Pregunta ${i + 1}`}
                      className="flex-1 bg-slate-50 border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold"
                    />
                    <select
                      value={p.tipo} onChange={(e) => actualizarPregunta(i, 'tipo', e.target.value)}
                      className="bg-slate-50 border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold"
                    >
                      <option value="opcion_multiple">Opción múltiple</option>
                      <option value="texto_libre">Texto libre</option>
                    </select>
                    {form.preguntas.length > 1 && (
                      <button onClick={() => quitarPregunta(i)} className="text-rose-400 hover:text-rose-600 p-1.5 shrink-0" title="Quitar pregunta">
                        <Trash2 size={16} />
                      </button>
                    )}
                  </div>

                  {p.tipo === 'opcion_multiple' && (
                    <div className="pl-4 space-y-2 border-l-2 border-slate-100">
                      {p.opciones.map((o, j) => (
                        <div key={j} className="flex items-center gap-2">
                          <input
                            type="text" value={o} onChange={(e) => actualizarOpcion(i, j, e.target.value)}
                            placeholder={`Opción ${j + 1}`}
                            className="flex-1 bg-slate-50 border border-slate-200 p-2 rounded-lg outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium"
                          />
                          {p.opciones.length > 2 && (
                            <button onClick={() => quitarOpcion(i, j)} className="text-slate-300 hover:text-rose-500 p-1 shrink-0"><X size={14} /></button>
                          )}
                        </div>
                      ))}
                      <button onClick={() => agregarOpcion(i)} className="flex items-center gap-1 text-[10px] font-black text-brand-600 uppercase hover:opacity-70">
                        <Plus size={12} /> Agregar opción
                      </button>
                    </div>
                  )}
                </div>
              ))}
              <button onClick={agregarPregunta} className="flex items-center gap-2 text-xs font-black text-brand-600 uppercase hover:opacity-70">
                <Plus size={14} /> Agregar pregunta
              </button>
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
        ) : encuestas.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin encuestas publicadas</div>
        ) : (
          <div className="space-y-3">
            {encuestas.map((enc) => (
              <button
                key={enc.id}
                onClick={() => setSeleccionada(enc.id)}
                className="w-full text-left bg-slate-50 border border-slate-100 hover:border-brand-300 hover:shadow-md p-5 rounded-[1.75rem] transition-all active:scale-[0.98] flex items-center justify-between gap-4"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    {!enc.activa && <span className="flex items-center gap-1 text-[9px] font-black uppercase text-slate-400 bg-slate-200 px-2 py-0.5 rounded-lg"><Lock size={10} /> Cerrada</span>}
                    <p className="font-black uppercase text-sm text-slate-900 truncate">{enc.titulo}</p>
                  </div>
                  {enc.descripcion && <p className="text-xs text-slate-500 font-medium truncate">{enc.descripcion}</p>}
                </div>
                <span className="shrink-0 flex items-center gap-1.5 bg-brand-100 text-brand-700 text-[10px] font-black px-3 py-1.5 rounded-full">
                  <ListChecks size={12} /> {enc.total_respuestas} de {enc.total_familias}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

const DetalleEncuesta = ({ id, onVolver }) => {
  const [detalle, setDetalle] = useState(null);
  const [loading, setLoading] = useState(true);
  const [cerrando, setCerrando] = useState(false);
  const [eliminando, setEliminando] = useState(false);

  const cargar = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get(`/encuestas/${id}`);
      setDetalle(res.data);
    } catch (err) {
      console.error('Error al cargar el detalle de la encuesta:', err);
      mostrarError('No se pudo cargar la encuesta');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { cargar(); }, [cargar]);

  const cerrarEncuesta = async () => {
    const ok = await confirmar('Los padres ya no podrán responderla.', '¿Cerrar esta encuesta?');
    if (!ok) return;
    setCerrando(true);
    try {
      await api.put(`/encuestas/${id}/cerrar`);
      cargar();
    } catch (err) {
      console.error('Error al cerrar la encuesta:', err);
      mostrarError('No se pudo cerrar la encuesta');
    } finally {
      setCerrando(false);
    }
  };

  const eliminarEncuesta = async () => {
    const ok = await confirmar('Se eliminará la encuesta y todas sus respuestas.', '¿Eliminar encuesta?');
    if (!ok) return;
    setEliminando(true);
    try {
      await api.delete(`/encuestas/${id}`);
      onVolver();
    } catch (err) {
      console.error('Error al eliminar la encuesta:', err);
      mostrarError('No se pudo eliminar la encuesta');
      setEliminando(false);
    }
  };

  if (loading || !detalle) {
    return (
      <div className="animate-in fade-in duration-500">
        <button onClick={onVolver} className="mb-6 flex items-center gap-2 text-brand-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
          <ArrowLeft size={16} /> Volver a encuestas
        </button>
        <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
      </div>
    );
  }

  return (
    <div className="animate-in fade-in duration-500">
      <button onClick={onVolver} className="mb-6 flex items-center gap-2 text-brand-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
        <ArrowLeft size={16} /> Volver a encuestas
      </button>

      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl space-y-8">
        <div className="flex flex-col sm:flex-row justify-between items-start gap-4">
          <div>
            <div className="flex items-center gap-2 mb-1">
              {!detalle.activa && <span className="flex items-center gap-1 text-[9px] font-black uppercase text-slate-400 bg-slate-200 px-2 py-0.5 rounded-lg"><Lock size={10} /> Cerrada</span>}
              <h3 className="text-xl font-black uppercase text-slate-900">{detalle.titulo}</h3>
            </div>
            {detalle.descripcion && <p className="text-sm text-slate-500 font-medium">{detalle.descripcion}</p>}
          </div>
          <div className="flex gap-2 shrink-0">
            {detalle.activa && (
              <button onClick={cerrarEncuesta} disabled={cerrando} className="flex items-center gap-1.5 bg-slate-100 hover:bg-slate-200 disabled:opacity-50 text-slate-600 text-[10px] font-black uppercase px-3 py-2 rounded-xl transition-all">
                {cerrando ? <Loader2 className="animate-spin" size={14} /> : <Lock size={14} />} Cerrar
              </button>
            )}
            <button onClick={eliminarEncuesta} disabled={eliminando} className="flex items-center gap-1.5 bg-rose-50 hover:bg-rose-100 disabled:opacity-50 text-rose-600 text-[10px] font-black uppercase px-3 py-2 rounded-xl transition-all">
              {eliminando ? <Loader2 className="animate-spin" size={14} /> : <Trash2 size={14} />} Eliminar
            </button>
          </div>
        </div>

        <div className="space-y-6">
          {detalle.preguntas.map((p) => {
            const totalRespuestas = p.tipo === 'opcion_multiple'
              ? Object.values(p.conteo_opciones || {}).reduce((a, b) => a + b, 0)
              : (p.respuestas_texto || []).length;
            return (
              <div key={p.id}>
                <p className="font-black text-sm text-slate-900 mb-3">{p.texto}</p>
                {p.tipo === 'opcion_multiple' ? (
                  <div className="space-y-2">
                    {(p.opciones || []).map((opcion) => {
                      const n = (p.conteo_opciones || {})[opcion] || 0;
                      const porcentaje = totalRespuestas > 0 ? Math.round((n / totalRespuestas) * 100) : 0;
                      return (
                        <div key={opcion}>
                          <div className="flex justify-between text-xs font-bold text-slate-600 mb-1">
                            <span>{opcion}</span>
                            <span>{n} ({porcentaje}%)</span>
                          </div>
                          <div className="h-2.5 bg-slate-100 rounded-full overflow-hidden">
                            <div className="h-full bg-brand-500 rounded-full transition-all" style={{ width: `${porcentaje}%` }} />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (p.respuestas_texto || []).length === 0 ? (
                  <p className="text-xs text-slate-400 font-bold uppercase">Sin respuestas todavía</p>
                ) : (
                  <div className="space-y-1.5">
                    {p.respuestas_texto.map((r, i) => (
                      <div key={i} className="flex items-start gap-2 text-sm text-slate-600 font-medium bg-slate-50 p-3 rounded-xl">
                        <CheckCircle2 size={14} className="text-emerald-500 shrink-0 mt-0.5" /> {r}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default PanelEncuestas;
