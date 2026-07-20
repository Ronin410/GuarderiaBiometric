import React, { useState, useEffect, useMemo } from 'react';
import api from './axiosConfig'; 
import { 
  Download, Clock, User, CheckCircle, ShieldAlert, 
  ShieldCheck, ChevronUp, ChevronDown, ArrowUpDown, Moon
} from 'lucide-react';

const PanelReportes = () => {
  const [reportes, setReportes] = useState([]);
  const [loading, setLoading] = useState(false);

  // CORRECCIÓN DE FECHA: Usamos toLocaleDateString para evitar desfases de horario UTC
  const hoyLocal = new Date().toLocaleDateString('en-CA'); 

  const [fechaInicio, setFechaInicio] = useState(hoyLocal);
  const [fechaFin, setFechaFin] = useState(hoyLocal);
  const [busquedaNombre, setBusquedaNombre] = useState(""); 
  const [sortConfig, setSortConfig] = useState({ key: 'fecha', direction: 'desc' });

  const obtenerReportes = async () => {
    setLoading(true);
    try {
      const res = await api.get('/reportes-asistencia', {
        params: { inicio: fechaInicio, fin: fechaFin }
      });
      setReportes(Array.isArray(res.data) ? res.data : []);
    } catch (error) { 
      console.error("Error al obtener reportes:", error); 
    } finally { 
      setLoading(false); 
    }
  };

  useEffect(() => { 
    obtenerReportes(); 
  }, [fechaInicio, fechaFin]);

  const handleSort = (key) => {
    let direction = 'asc';
    if (sortConfig.key === key && sortConfig.direction === 'asc') {
      direction = 'desc';
    }
    setSortConfig({ key, direction });
  };

  const reportesProcesados = useMemo(() => {
    let items = [...reportes].filter(reg => 
      reg.hijo_nombre.toLowerCase().includes(busquedaNombre.toLowerCase())
    );

    items.sort((a, b) => {
      const valA = a[sortConfig.key] || '';
      const valB = b[sortConfig.key] || '';
      if (valA < valB) return sortConfig.direction === 'asc' ? -1 : 1;
      if (valA > valB) return sortConfig.direction === 'asc' ? 1 : -1;
      return 0;
    });
    return items;
  }, [reportes, busquedaNombre, sortConfig]);

  return (
    <div className="min-h-screen bg-[#f8fafc] p-4 md:p-8 font-sans flex flex-col items-center">
      
      <style>{`
        @media print {
          @page { size: landscape; margin: 0.8cm; }
          body { background: white !important; }
          .no-print { display: none !important; }
          .print-container { width: 100% !important; max-width: 100% !important; margin: 0 auto !important; padding: 0 !important; }
          .table-wrapper { border: 1px solid #e2e8f0 !important; box-shadow: none !important; }
          table { width: 100% !important; border-collapse: collapse !important; table-layout: fixed !important; }
          .break-inside-avoid { page-break-inside: avoid; }
        }
        .text-break-custom {
          word-break: break-word;
          overflow-wrap: break-word;
          display: -webkit-box;
          -webkit-line-clamp: 3;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }
      `}</style>

      {/* CABECERA */}
      <div className="print-container w-full max-w-[1400px] mb-6 flex flex-col md:flex-row justify-between items-center px-4">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-violet-600 rounded-2xl text-white shadow-lg">
            <ShieldCheck size={32} />
          </div>
          <div>
            <h1 className="text-2xl font-black text-slate-900 tracking-tighter uppercase leading-none">BIOSAFE</h1>
            <p className="text-[10px] font-bold text-violet-600 uppercase tracking-[0.2em] mt-1">SISTEMA DE CONTROL DIARIO</p>
          </div>
        </div>
        <div className="bg-white px-6 py-3 rounded-2xl border border-slate-200 shadow-sm text-right mt-4 md:mt-0">
          <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-1">Reporte de Asistencia</p>
          <div className="text-sm font-black text-slate-800">{fechaInicio} <span className="text-violet-400">/</span> {fechaFin}</div>
        </div>
      </div>

      {/* FILTROS (no-print) */}
      <div className="no-print w-full max-w-[1400px] mb-6 px-4">
        <div className="bg-white p-4 rounded-[2rem] shadow-sm flex flex-wrap lg:flex-nowrap items-end gap-4 border border-slate-200/60">
          <div className="flex-1 min-w-[140px]">
            <label className="text-[10px] font-black text-slate-400 uppercase ml-2 block mb-1">Inicio</label>
            <input type="date" value={fechaInicio} onChange={(e)=>setFechaInicio(e.target.value)} className="w-full bg-slate-50 border border-slate-200 p-2.5 rounded-xl font-bold text-sm outline-none focus:border-violet-400 transition-colors" />
          </div>
          <div className="flex-1 min-w-[140px]">
            <label className="text-[10px] font-black text-slate-400 uppercase ml-2 block mb-1">Fin</label>
            <input type="date" value={fechaFin} onChange={(e)=>setFechaFin(e.target.value)} className="w-full bg-slate-50 border border-slate-200 p-2.5 rounded-xl font-bold text-sm outline-none focus:border-violet-400 transition-colors" />
          </div>
          <div className="flex-[2] min-w-[200px]">
            <label className="text-[10px] font-black text-slate-400 uppercase ml-2 block mb-1">Filtrar Alumno</label>
            <input type="text" placeholder="BUSCAR POR NOMBRE..." value={busquedaNombre} onChange={(e)=>setBusquedaNombre(e.target.value)} className="w-full bg-slate-50 border border-slate-200 p-2.5 rounded-xl font-bold text-sm outline-none focus:border-violet-400 transition-colors uppercase" />
          </div>
          <button onClick={() => window.print()} className="bg-violet-600 hover:bg-violet-700 text-white font-black px-6 py-3 rounded-xl flex items-center gap-2 shadow-lg text-xs uppercase transition-all active:scale-95">
            <Download size={18}/> Imprimir Reporte
          </button>
        </div>
      </div>

      {/* TABLA PRINCIPAL */}
      <div className="print-container w-full max-w-[1400px] px-4 pb-10">
        <div className="table-wrapper bg-white rounded-[1.5rem] border border-slate-200 shadow-sm overflow-hidden mb-8">
          <table className="w-full border-collapse">
            <thead>
              <tr className="bg-slate-50 text-slate-500 text-[9px] font-black uppercase tracking-widest border-b border-slate-200">
                <th onClick={() => handleSort('fecha')} className="w-[110px] p-4 text-left border-r border-slate-100 cursor-pointer hover:bg-slate-100 group transition-colors">
                  <div className="flex items-center gap-1">
                    FECHA {sortConfig.key === 'fecha' ? (sortConfig.direction === 'asc' ? <ChevronUp size={12}/> : <ChevronDown size={12}/>) : <ArrowUpDown size={10} className="opacity-30 group-hover:opacity-100"/>}
                  </div>
                </th>
                <th onClick={() => handleSort('hijo_nombre')} className="w-[180px] p-4 text-left border-r border-slate-100 cursor-pointer hover:bg-slate-100 group transition-colors">
                  <div className="flex items-center gap-1">
                    ALUMNO {sortConfig.key === 'hijo_nombre' ? (sortConfig.direction === 'asc' ? <ChevronUp size={12}/> : <ChevronDown size={12}/>) : <ArrowUpDown size={10} className="opacity-30 group-hover:opacity-100"/>}
                  </div>
                </th>
                <th className="w-[70px] p-4 text-center border-r border-slate-100 text-[8px]">TIPO</th>
                <th className="w-[100px] p-4 text-center border-r border-slate-100 text-[8px]">ESTADO</th>
                <th className="p-4 text-left text-[8px]">DETALLES DE BITÁCORA</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {reportesProcesados.map((reg, i) => {
                const esSalida = reg.tipo === 'SALIDA';
                const b = reg.bitacora || {};
                return (
                  <tr key={i} className="hover:bg-slate-50/40 break-inside-avoid transition-colors">
                    <td className="p-4 border-r border-slate-50 align-top">
                      <div className="text-[10px] font-bold text-slate-400">{reg.fecha.split(' ')[0]}</div>
                      <div className="text-[11px] font-black text-slate-800 mt-1 flex items-center gap-1">
                        <Clock size={12} className="text-violet-500"/> {reg.fecha.split(' ')[1]}
                      </div>
                    </td>
                    <td className="p-4 border-r border-slate-50 align-top">
                      <div className="font-black text-slate-900 uppercase text-[11px] leading-tight mb-1">{reg.hijo_nombre}</div>
                      <div className="text-[9px] text-violet-500 font-bold uppercase flex items-center gap-1">
                        <User size={10} className="opacity-70"/>{reg.tutor_nombre}
                      </div>
                    </td>
                    <td className="p-4 border-r border-slate-50 align-top text-center">
                      <span className={`text-[9px] font-black px-2 py-1 rounded-lg border ${esSalida ? 'bg-orange-50 text-orange-600 border-orange-100' : 'bg-emerald-50 text-emerald-600 border-emerald-100'}`}>
                        {reg.tipo}
                      </span>
                    </td>
                    <td className="p-4 border-r border-slate-50 align-top">
                      <div className="flex justify-center items-center gap-3 h-full pt-1">
                        <div className="flex flex-col items-center">
                          <CheckCircle size={18} className={reg.aseado ? 'text-emerald-500' : 'text-slate-200'} />
                          <span className="text-[7px] font-black text-slate-400 mt-1 uppercase">Aseo</span>
                        </div>
                        <div className="flex flex-col items-center">
                          <ShieldAlert size={18} className={reg.golpe ? 'text-red-500' : 'text-slate-200'} />
                          <span className="text-[7px] font-black text-slate-400 mt-1 uppercase">Golpe</span>
                        </div>
                      </div>
                    </td>
                    <td className="p-4 align-top">
                      {esSalida ? (
                        <div className="space-y-3">
                          <div className="grid grid-cols-2 lg:grid-cols-5 gap-3"> {/* Ajustado a 5 columnas */}
                            {[
                              { label: 'DESAYUNO', val: b.desayuno, col: 'text-blue-600', bg: 'bg-blue-50/40' },
                              { label: 'COMIDA', val: b.comida, col: 'text-orange-600', bg: 'bg-orange-50/40' },
                              { label: 'MERIENDA', val: b.merienda, col: 'text-pink-600', bg: 'bg-pink-50/40' },
                              { label: 'ESFÍNTER', val: b.esfinter, col: 'text-emerald-600', bg: 'bg-emerald-50/40' },
                              // NUEVA TARJETA: ESTADO DE SUEÑO
                              { 
                                label: 'DORMIÓ', 
                                val: b.durmio ? 'SÍ' : 'NO', 
                                col: b.durmio ? 'text-indigo-600' : 'text-slate-400', 
                                bg: b.durmio ? 'bg-indigo-50/40' : 'bg-slate-50/40' 
                              }
                            ].map(item => (
                              <div key={item.label} className={`${item.bg} p-2.5 rounded-xl border border-slate-100`}>
                                <span className={`text-[8px] font-black ${item.col} block mb-1 tracking-wider`}>{item.label}</span>
                                <p className="text-[10px] font-bold text-slate-700 leading-tight italic text-break-custom">
                                  {item.val || '--'}
                                </p>
                              </div>
                            ))}
                          </div>
                          {b.observaciones && (
                            <div className="p-3 bg-slate-50/80 rounded-xl border border-slate-200 border-dashed">
                              <span className="text-[8px] font-black text-slate-400 block uppercase mb-1 tracking-widest">Observaciones Finales</span>
                              <p className="text-[10px] text-slate-600 font-medium italic leading-relaxed">{b.observaciones}</p>
                            </div>
                          )}
                        </div>
                      ) : (
                        <div className="bg-violet-50/40 p-3 rounded-xl border border-violet-100 border-dashed">
                          <span className="text-[8px] font-black text-violet-600 block mb-1 uppercase tracking-widest">Nota de Ingreso</span>
                          <p className="text-[11px] font-bold text-slate-600 italic">
                            "{reg.obs_asistencia || "Sin novedades reportadas."}"
                          </p>
                        </div>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* SECCIÓN DE FIRMA */}
        <div className="hidden print:flex flex-col items-center mt-16 space-y-4">
          <div className="flex items-center justify-center gap-20">
            <div className="w-[100px] h-[100px] border border-dashed border-slate-200 rounded-full flex items-center justify-center">
              <span className="text-[7px] font-black text-slate-200 uppercase">Sello</span>
            </div>
            <div className="flex flex-col items-center">
              <div className="w-[300px] border-t-2 border-slate-900 mb-2"></div>
              <p className="text-[11px] font-black text-slate-900 uppercase tracking-tighter">Firma de la Dirección</p>
              <p className="text-[8px] font-bold text-slate-400 uppercase tracking-[0.2em]">Responsable de Institución</p>
            </div>
            <div className="w-[100px]"></div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default PanelReportes;