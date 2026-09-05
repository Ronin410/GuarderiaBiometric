import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import { CalendarOff, Baby } from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { hoyLocal } from './utils/fecha';
import { acentoDeTab } from './utils/acentos';
import DinoDecorativo from './components/DinoDecorativo';

// Color y dino de este apartado -- los define utils/acentos.js para que
// coincidan con los del menú lateral.
const acento = acentoDeTab('ausencias');

// PanelAusencias -- "Planificación de ausencias por parte de los padres"
// (lado staff): qué niños ya avisaron que no vienen, en un rango de fechas.
// Por defecto muestra hoy + los próximos 7 días, agrupado por fecha para
// que sea fácil de leer de un vistazo al preparar el día.
const sumarDias = (fechaISO, dias) => {
  const [anio, mes, dia] = fechaISO.split('-').map(Number);
  const d = new Date(anio, mes - 1, dia + dias);
  return d.toISOString().slice(0, 10);
};

const formatoFechaLarga = (iso) => {
  try {
    // new Date('YYYY-MM-DD') se interpreta en UTC; se arma con hora local para
    // evitar que se recorra un día por la zona horaria del navegador.
    const [anio, mes, dia] = iso.split('-').map(Number);
    return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { weekday: 'long', day: 'numeric', month: 'long' });
  } catch {
    return iso;
  }
};

const PanelAusencias = () => {
  const [desde, setDesde] = useState(hoyLocal());
  const [hasta, setHasta] = useState(sumarDias(hoyLocal(), 7));
  const [ausencias, setAusencias] = useState([]);
  const [loading, setLoading] = useState(true);

  const cargar = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get('/ausencias', { params: { desde, hasta } });
      setAusencias(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar ausencias:', err);
      mostrarError('No se pudieron cargar las ausencias avisadas');
    } finally {
      setLoading(false);
    }
  }, [desde, hasta]);

  useEffect(() => { cargar(); }, [cargar]);

  const porFecha = ausencias.reduce((acc, a) => {
    (acc[a.fecha] = acc[a.fecha] || []).push(a);
    return acc;
  }, {});

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className={`${acento.fondo} p-3 rounded-2xl ${acento.texto}`}><CalendarOff size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Ausencias Avisadas</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Quién no viene, avisado por los papás</p>
            </div>
            <DinoDecorativo src="/dinos/dino-verde.png" className="hidden sm:block h-14 w-auto shrink-0" espejo />
          </div>
          <div className="flex items-center gap-2">
            <input type="date" value={desde} onChange={(e) => setDesde(e.target.value)} className="bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
            <span className="text-slate-300 text-xs">–</span>
            <input type="date" value={hasta} min={desde} onChange={(e) => setHasta(e.target.value)} className="bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : Object.keys(porFecha).length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Ninguna familia ha avisado ausencias en este rango</div>
        ) : (
          <div className="space-y-6">
            {Object.keys(porFecha).sort().map((fecha) => (
              <div key={fecha}>
                <p className="text-[10px] font-black text-brand-500 uppercase tracking-widest mb-2 ml-1">{formatoFechaLarga(fecha)}</p>
                <div className="space-y-2">
                  {porFecha[fecha].map((a) => (
                    <div key={a.id} className="flex items-center gap-4 p-4 rounded-2xl border border-slate-100 bg-slate-50">
                      <div className="bg-white p-2.5 rounded-xl text-slate-300 border border-slate-100 shrink-0"><Baby size={18} /></div>
                      <div className="min-w-0">
                        <p className="font-black text-sm text-slate-900">{a.hijo_nombre}</p>
                        {a.motivo && <p className="text-xs text-slate-500 font-medium">{a.motivo}</p>}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelAusencias;
