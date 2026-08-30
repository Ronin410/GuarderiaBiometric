import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import { CalendarDays, Plus, X, Send, Loader2, Trash2 } from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import { hoyLocal } from './utils/fecha';

const FORM_VACIO = { titulo: '', descripcion: '', fecha_inicio: '', fecha_fin: '', tipo: 'evento' };

const TIPOS = {
  evento: { label: 'Evento', color: 'bg-brand-100 text-brand-700 border-brand-200' },
  suspension: { label: 'Suspensión de clases', color: 'bg-rose-100 text-rose-700 border-rose-200' },
  vacaciones: { label: 'Vacaciones', color: 'bg-amber-100 text-amber-700 border-amber-200' },
  junta: { label: 'Junta de padres', color: 'bg-emerald-100 text-emerald-700 border-emerald-200' },
};

const sumarDias = (fechaISO, dias) => {
  const [anio, mes, dia] = fechaISO.split('-').map(Number);
  const d = new Date(anio, mes - 1, dia + dias);
  return d.toISOString().slice(0, 10);
};

const formatoFecha = (iso) => {
  try {
    const [anio, mes, dia] = iso.split('-').map(Number);
    return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', year: 'numeric' });
  } catch {
    return iso;
  }
};

// PanelCalendario -- "Calendario escolar" del PDF de referencia: fechas
// importantes del centro (suspensiones, actividades, juntas, vacaciones)
// que crea/gestiona staff y consultan también los padres.
const PanelCalendario = () => {
  const [eventos, setEventos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState(FORM_VACIO);
  const [guardando, setGuardando] = useState(false);
  const [eliminandoId, setEliminandoId] = useState(null);
  const [desde, setDesde] = useState(hoyLocal());
  const [hasta, setHasta] = useState(sumarDias(hoyLocal(), 90));

  const cargar = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get('/calendario', { params: { desde, hasta } });
      setEventos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar el calendario:', err);
      mostrarError('No se pudo cargar el calendario');
    } finally {
      setLoading(false);
    }
  }, [desde, hasta]);

  useEffect(() => { cargar(); }, [cargar]);

  const publicar = async () => {
    if (!form.titulo.trim() || !form.fecha_inicio) {
      mostrarError('El título y la fecha de inicio son obligatorios');
      return;
    }
    setGuardando(true);
    try {
      await api.post('/calendario', form);
      mostrarExito('El evento se publicó en el calendario');
      setForm(FORM_VACIO);
      setMostrarForm(false);
      cargar();
    } catch (err) {
      console.error('Error al crear el evento:', err);
      mostrarError(err.response?.data?.error || 'No se pudo crear el evento');
    } finally {
      setGuardando(false);
    }
  };

  const eliminar = async (ev) => {
    const ok = await confirmar(`Se eliminará "${ev.titulo}" del calendario.`, '¿Eliminar evento?');
    if (!ok) return;
    setEliminandoId(ev.id);
    try {
      await api.delete(`/calendario/${ev.id}`);
      cargar();
    } catch (err) {
      console.error('Error al eliminar el evento:', err);
      mostrarError('No se pudo eliminar el evento');
    } finally {
      setEliminandoId(null);
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><CalendarDays size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Calendario Escolar</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Fechas importantes del centro</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <input type="date" value={desde} onChange={(e) => setDesde(e.target.value)} className="bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
              <span className="text-slate-300 text-xs">–</span>
              <input type="date" value={hasta} min={desde} onChange={(e) => setHasta(e.target.value)} className="bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
            </div>
            <button
              onClick={() => { setMostrarForm(!mostrarForm); setForm(FORM_VACIO); }}
              className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95 whitespace-nowrap"
            >
              {mostrarForm ? <X size={14} /> : <Plus size={14} />}
              {mostrarForm ? 'Cancelar' : 'Nuevo Evento'}
            </button>
          </div>
        </div>

        {mostrarForm && (
          <div className="mb-8 p-6 rounded-[2rem] border border-dashed border-brand-300 bg-brand-50/40 space-y-4">
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Título</label>
              <input
                type="text" value={form.titulo} onChange={(e) => setForm({ ...form, titulo: e.target.value })}
                placeholder="ej. Suspensión de clases por junta de consejo técnico"
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold"
              />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Desde</label>
                <input type="date" value={form.fecha_inicio} onChange={(e) => setForm({ ...form, fecha_inicio: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
              </div>
              <div>
                <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Hasta (opcional)</label>
                <input type="date" value={form.fecha_fin} min={form.fecha_inicio} onChange={(e) => setForm({ ...form, fecha_fin: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
              </div>
              <div>
                <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Tipo</label>
                <select value={form.tipo} onChange={(e) => setForm({ ...form, tipo: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold">
                  {Object.entries(TIPOS).map(([valor, info]) => <option key={valor} value={valor}>{info.label}</option>)}
                </select>
              </div>
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Descripción (opcional)</label>
              <textarea
                rows={3} value={form.descripcion} onChange={(e) => setForm({ ...form, descripcion: e.target.value })}
                placeholder="Detalles adicionales..."
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium resize-y"
              />
            </div>
            <div className="flex justify-end">
              <button
                onClick={publicar}
                disabled={guardando}
                className="flex items-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-3 rounded-xl shadow-md transition-all active:scale-95"
              >
                {guardando ? <Loader2 className="animate-spin" size={16} /> : <Send size={16} />} Publicar Evento
              </button>
            </div>
          </div>
        )}

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : eventos.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin eventos en este rango</div>
        ) : (
          <div className="space-y-4">
            {eventos.map((ev) => {
              const info = TIPOS[ev.tipo] || TIPOS.evento;
              return (
                <div key={ev.id} className="p-6 rounded-[2rem] border border-slate-100 bg-slate-50">
                  <div className="flex justify-between items-start gap-4">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap mb-1.5">
                        <span className={`text-[9px] font-black uppercase px-2.5 py-1 rounded-lg border ${info.color}`}>{info.label}</span>
                      </div>
                      <p className="font-black text-lg uppercase tracking-tight text-slate-900">{ev.titulo}</p>
                      <p className="text-[10px] text-brand-500 font-bold uppercase mt-1">
                        {formatoFecha(ev.fecha_inicio)}{ev.fecha_fin && ev.fecha_fin !== ev.fecha_inicio ? ` – ${formatoFecha(ev.fecha_fin)}` : ''}
                      </p>
                    </div>
                    <button
                      onClick={() => eliminar(ev)}
                      disabled={eliminandoId === ev.id}
                      className="text-rose-400 hover:text-rose-600 disabled:opacity-50 p-2 shrink-0"
                      title="Eliminar evento"
                    >
                      {eliminandoId === ev.id ? <Loader2 className="animate-spin" size={16} /> : <Trash2 size={16} />}
                    </button>
                  </div>
                  {ev.descripcion && <p className="mt-3 text-sm text-slate-600 font-medium whitespace-pre-wrap">{ev.descripcion}</p>}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelCalendario;
