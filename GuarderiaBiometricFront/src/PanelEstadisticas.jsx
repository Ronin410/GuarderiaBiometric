import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import { BarChart3, CalendarRange, TrendingDown, Clock3 } from 'lucide-react';
import { hoyLocal } from './utils/fecha';

// Construye la fecha a partir de un string "YYYY-MM-DD" anclado a UTC, para no
// depender de la zona horaria del navegador al calcular inicios de semana/mes/año
// (mismo espíritu que src/utils/fecha.js: evitar el desfase que ya causó bugs antes).
const parseISO = (str) => {
  const [y, m, d] = str.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d));
};
const formatISO = (date) => date.toISOString().split('T')[0];

const inicioSemana = (hoyStr) => {
  const hoy = parseISO(hoyStr);
  const diaSemana = hoy.getUTCDay(); // 0=domingo .. 6=sábado
  const offset = diaSemana === 0 ? 6 : diaSemana - 1; // lunes = inicio de semana
  hoy.setUTCDate(hoy.getUTCDate() - offset);
  return formatISO(hoy);
};
const inicioMes = (hoyStr) => hoyStr.slice(0, 8) + '01';
const inicioAnio = (hoyStr) => hoyStr.slice(0, 4) + '-01-01';

const RANGOS = [
  { key: 'semana', label: 'Esta Semana', desde: () => inicioSemana(hoyLocal()) },
  { key: 'mes', label: 'Este Mes', desde: () => inicioMes(hoyLocal()) },
  { key: 'anio', label: 'Este Año', desde: () => inicioAnio(hoyLocal()) },
];

const colorPorcentaje = (pct) => {
  if (pct >= 90) return { barra: 'bg-emerald-500', texto: 'text-emerald-600' };
  if (pct >= 75) return { barra: 'bg-amber-500', texto: 'text-amber-600' };
  return { barra: 'bg-rose-500', texto: 'text-rose-600' };
};

const PanelEstadisticas = () => {
  const [rangoActivo, setRangoActivo] = useState('mes');
  const [desde, setDesde] = useState(inicioMes(hoyLocal()));
  const [hasta, setHasta] = useState(hoyLocal());
  const [resumen, setResumen] = useState([]);
  const [loading, setLoading] = useState(true);

  const aplicarRango = (rango) => {
    setRangoActivo(rango.key);
    setDesde(rango.desde());
    setHasta(hoyLocal());
  };

  const cargarResumen = async () => {
    setLoading(true);
    try {
      const res = await api.get('/reportes/asistencia-resumen', { params: { desde, hasta } });
      setResumen(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar estadísticas:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargarResumen(); }, [desde, hasta]);

  const totalAusencias = resumen.reduce((acc, r) => acc + (r.dias_ausente || 0), 0);
  const totalTardanzas = resumen.reduce((acc, r) => acc + (r.dias_tarde || 0), 0);

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><BarChart3 size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Estadísticas de Asistencia</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Faltas y llegadas tarde por alumno</p>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            {RANGOS.map(r => (
              <button
                key={r.key}
                onClick={() => aplicarRango(r)}
                className={`px-4 py-2.5 rounded-xl font-black text-[10px] uppercase transition-all ${rangoActivo === r.key ? 'bg-brand-600 text-white shadow-md' : 'bg-slate-100 text-slate-500 hover:bg-slate-200'}`}
              >
                {r.label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex flex-wrap items-end gap-4 mb-8 bg-slate-50 p-4 rounded-2xl border border-slate-100">
          <div>
            <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Desde</label>
            <div className="flex items-center bg-white border border-slate-200 rounded-xl px-3">
              <CalendarRange size={16} className="text-slate-400" />
              <input type="date" value={desde} onChange={(e) => { setRangoActivo(null); setDesde(e.target.value); }} className="bg-transparent p-2.5 outline-none font-bold text-sm" />
            </div>
          </div>
          <div>
            <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Hasta</label>
            <div className="flex items-center bg-white border border-slate-200 rounded-xl px-3">
              <CalendarRange size={16} className="text-slate-400" />
              <input type="date" value={hasta} onChange={(e) => { setRangoActivo(null); setHasta(e.target.value); }} className="bg-transparent p-2.5 outline-none font-bold text-sm" />
            </div>
          </div>

          <div className="flex-1" />

          <div className="flex items-center gap-2 bg-rose-50 border border-rose-100 px-4 py-3 rounded-xl">
            <TrendingDown size={16} className="text-rose-500" />
            <span className="text-[10px] font-black text-rose-600 uppercase">{totalAusencias} Faltas Totales</span>
          </div>
          <div className="flex items-center gap-2 bg-amber-50 border border-amber-100 px-4 py-3 rounded-xl">
            <Clock3 size={16} className="text-amber-500" />
            <span className="text-[10px] font-black text-amber-600 uppercase">{totalTardanzas} Tardanzas</span>
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Calculando...</div>
        ) : resumen.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin datos para este rango</div>
        ) : (
          <div className="overflow-x-auto rounded-[1.5rem] border border-slate-200">
            <table className="w-full border-collapse">
              <thead>
                <tr className="bg-slate-50 text-slate-500 text-[9px] font-black uppercase tracking-widest border-b border-slate-200">
                  <th className="p-4 text-left">Alumno</th>
                  <th className="p-4 text-center">Días Hábiles</th>
                  <th className="p-4 text-center">Asistió</th>
                  <th className="p-4 text-center">Ausente</th>
                  <th className="p-4 text-center">Tarde</th>
                  <th className="p-4 text-left w-[220px]">% Asistencia</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {resumen.map((r) => {
                  const pct = Math.round(r.porcentaje_asistencia || 0);
                  const { barra, texto } = colorPorcentaje(pct);
                  return (
                    <tr key={r.hijo_id} className="hover:bg-slate-50/60 transition-colors">
                      <td className="p-4 font-black text-slate-900 uppercase text-[11px]">{r.nombre}</td>
                      <td className="p-4 text-center text-[11px] font-bold text-slate-500">{r.dias_habiles}</td>
                      <td className="p-4 text-center text-[11px] font-bold text-emerald-600">{r.dias_asistio}</td>
                      <td className="p-4 text-center text-[11px] font-bold text-rose-500">{r.dias_ausente}</td>
                      <td className="p-4 text-center text-[11px] font-bold text-amber-500">{r.dias_tarde}</td>
                      <td className="p-4">
                        <div className="flex items-center gap-3">
                          <div className="flex-1 h-2.5 bg-slate-100 rounded-full overflow-hidden">
                            <div className={`h-full ${barra} rounded-full transition-all`} style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
                          </div>
                          <span className={`text-[11px] font-black ${texto} w-10 text-right`}>{pct}%</span>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelEstadisticas;
