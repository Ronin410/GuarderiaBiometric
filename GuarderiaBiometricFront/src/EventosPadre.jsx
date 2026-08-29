import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import { ChevronLeft, CalendarDays } from 'lucide-react';
import { mostrarError } from './utils/alertas';

// Mismo catálogo/colores que ya usa PanelCalendario.jsx del lado de
// staff -- para que un papá vea el mismo tipo de evento con el mismo color
// que quien lo publicó.
const TIPOS = {
  evento: { label: 'Evento', color: 'bg-brand-100 text-brand-700 border-brand-200' },
  suspension: { label: 'Suspensión de clases', color: 'bg-rose-100 text-rose-700 border-rose-200' },
  vacaciones: { label: 'Vacaciones', color: 'bg-amber-100 text-amber-700 border-amber-200' },
  junta: { label: 'Junta de padres', color: 'bg-emerald-100 text-emerald-700 border-emerald-200' },
};

// EventosPadre -- vista de pantalla completa (mismo patrón que
// CircularesPadre/EncuestasPadre/ChatPadre: swap desde DashboardPadre) con
// TODOS los eventos del calendario escolar, no solo los 3 próximos que ya
// se alcanzaban a ver en el inicio. Reutiliza /padre/calendario tal cual --
// ya regresa hoy hasta 90 días adelante ordenados por fecha, así que aquí
// no hace falta filtrar ni truncar nada, solo no cortar a los primeros 3.
const EventosPadre = ({ onVolver }) => {
  const [eventos, setEventos] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const cargar = async () => {
      setLoading(true);
      try {
        const res = await api.get('/padre/calendario');
        setEventos(Array.isArray(res.data) ? res.data : []);
      } catch (err) {
        console.error('Error al cargar el calendario', err);
        mostrarError('No se pudo cargar el calendario escolar');
      } finally {
        setLoading(false);
      }
    };
    cargar();
  }, []);

  const formatoFecha = (iso) => {
    try {
      const [anio, mes, dia] = iso.split('-').map(Number);
      return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', year: 'numeric' });
    } catch {
      return iso;
    }
  };

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
          <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter">Eventos</h2>
          <div className="bg-brand-600 p-3 rounded-2xl text-white shadow-lg shadow-brand-200">
            <CalendarDays size={20} />
          </div>
        </div>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-4">
        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : eventos.length === 0 ? (
          <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center">
            <CalendarDays size={40} className="mx-auto text-slate-200 mb-4" />
            <p className="text-slate-400 font-bold uppercase text-[10px]">No hay eventos próximos en el calendario</p>
          </div>
        ) : (
          eventos.map((ev) => {
            const info = TIPOS[ev.tipo] || TIPOS.evento;
            return (
              <div key={ev.id} className="bg-white rounded-[2rem] p-6 shadow-sm border border-slate-100 space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <p className="font-black text-slate-900 uppercase text-sm">{ev.titulo}</p>
                  <span className={`shrink-0 text-[9px] font-black px-2.5 py-1 rounded-lg border uppercase ${info.color}`}>{info.label}</span>
                </div>
                <p className="text-[9px] text-brand-500 font-bold uppercase">
                  {formatoFecha(ev.fecha_inicio)}
                  {ev.fecha_fin && ev.fecha_fin !== ev.fecha_inicio ? ` – ${formatoFecha(ev.fecha_fin)}` : ''}
                </p>
                {ev.descripcion && (
                  <p className="text-xs text-slate-600 font-medium whitespace-pre-wrap">{ev.descripcion}</p>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

export default EventosPadre;
