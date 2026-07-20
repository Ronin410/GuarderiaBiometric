import React, { useEffect, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import axios from 'axios';
import { 
  Utensils, Moon, Camera, Clock, Heart, CheckCircle2, 
  ShieldCheck, Baby, Info, Calendar as CalendarIcon, X 
} from 'lucide-react';

const API_URL = 'https://guarderiabiometricback.onrender.com';

const ReportePublico = () => {
  const { token } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Estados
  const [reporte, setReporte] = useState(null);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState("");
  const [fotoSeleccionada, setFotoSeleccionada] = useState(null);
  
  // Si la URL no trae fecha, usamos la de hoy en formato Local
  const fechaUrl = searchParams.get("fecha") || new Date().toISOString().split('T')[0];
  const [fechaSeleccionada, setFechaSeleccionada] = useState(fechaUrl);

  const fetchPublico = async (fecha) => {
    try {
      setLoading(true);
      // Nota: Usamos la ruta /publico/ que creamos en Go
      const res = await axios.get(`${API_URL}/publico/seguimiento/${token}?fecha=${fecha}`);
      setReporte(res.data);
      setErrorMsg("");
    } catch (err) {
      console.error("Error al obtener reporte:", err);
      setReporte(null);
      setErrorMsg(err.response?.data?.error || "No hay reporte disponible para esta fecha.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) fetchPublico(fechaSeleccionada);
  }, [token, fechaSeleccionada]);

  const handleCambioFecha = (e) => {
    const nuevaFecha = e.target.value;
    setFechaSeleccionada(nuevaFecha);
    setSearchParams({ fecha: nuevaFecha });
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center p-10">
        <div className="text-center space-y-4">
          <div className="w-12 h-12 border-4 border-violet-600 border-t-transparent rounded-full animate-spin mx-auto"></div>
          <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Cargando Seguimiento...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f8fafc] pb-20 animate-in fade-in duration-500">
      
      {/* MODAL FOTO GRANDE */}
      {fotoSeleccionada && (
        <div 
          className="fixed inset-0 z-[100] bg-slate-900/95 backdrop-blur-md flex items-center justify-center p-4"
          onClick={() => setFotoSeleccionada(null)}
        >
          <button className="absolute top-6 right-6 text-white bg-white/10 p-3 rounded-full"><X size={24} /></button>
          <img src={fotoSeleccionada} alt="Preview" className="max-w-full max-h-[85vh] rounded-3xl shadow-2xl object-contain" />
        </div>
      )}

      {/* HEADER PÚBLICO */}
      <div className="bg-white p-6 pb-10 rounded-b-[3.5rem] shadow-sm border-b border-slate-100 sticky top-0 z-30">
        <div className="max-w-md mx-auto space-y-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="bg-violet-600 p-2.5 rounded-xl shadow-lg shadow-violet-200">
                <ShieldCheck size={24} className="text-white" />
              </div>
              <div>
                <h1 className="text-xl font-black text-slate-900 uppercase tracking-tighter leading-none">BioSafe</h1>
                <p className="text-[9px] font-bold text-violet-500 uppercase tracking-widest">Reporte Diario</p>
              </div>
            </div>
            <div className="text-rose-500 bg-rose-50 p-2 rounded-full"><Heart size={20} fill="currentColor" /></div>
          </div>

          <div className="space-y-4">
            <h2 className="text-3xl font-black text-slate-900 uppercase tracking-tighter">
              {reporte?.hijo_nombre || "Alumno"}
            </h2>
            <div className="relative">
              <CalendarIcon className="absolute left-4 top-1/2 -translate-y-1/2 text-violet-600" size={18} />
              <input 
                type="date" 
                value={fechaSeleccionada}
                onChange={handleCambioFecha}
                className="w-full bg-slate-50 border-none rounded-2xl py-4 pl-12 pr-4 text-sm font-black text-slate-700 focus:ring-2 focus:ring-violet-500 transition-all uppercase"
              />
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-md mx-auto p-4 mt-4 space-y-4">
        {errorMsg ? (
          <div className="bg-white p-12 rounded-[2.5rem] border-2 border-dashed border-slate-200 text-center space-y-4">
            <div className="bg-slate-50 w-16 h-16 rounded-full flex items-center justify-center mx-auto text-slate-300">
              <Clock size={32} />
            </div>
            <p className="font-black text-slate-400 uppercase text-[10px] tracking-widest leading-relaxed">
              {errorMsg}
            </p>
          </div>
        ) : (
          <>
            {/* ALIMENTACIÓN */}
            <div className="bg-white rounded-[2.5rem] p-7 shadow-sm border border-slate-100">
              <div className="flex items-center gap-3 mb-6">
                <div className="p-2 bg-amber-100 text-amber-600 rounded-xl"><Utensils size={20} /></div>
                <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Alimentación</h3>
              </div>
              <div className="space-y-5">
                <ItemReporte label="Desayuno" valor={reporte.desayuno} color="bg-blue-500" />
                <ItemReporte label="Comida" valor={reporte.comida} color="bg-orange-500" />
                <ItemReporte label="Merienda" valor={reporte.merienda} color="bg-pink-500" />
              </div>
            </div>

            {/* DESCANSO */}
            <div className={`rounded-[2.5rem] p-7 border transition-all ${reporte.durmio ? 'bg-violet-600 border-violet-500 text-white shadow-xl shadow-violet-100' : 'bg-white border-slate-100 text-slate-400'}`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className={`p-3 rounded-2xl ${reporte.durmio ? 'bg-white/20' : 'bg-slate-50 text-slate-300'}`}><Moon size={28} /></div>
                  <div>
                    <p className="font-black uppercase text-xs tracking-widest">Descanso</p>
                    <p className="text-[10px] font-bold uppercase opacity-80">{reporte.durmio ? 'Tomó su siesta' : 'No hubo siesta'}</p>
                  </div>
                </div>
                {reporte.durmio && <CheckCircle2 size={28} />}
              </div>
            </div>

            {/* ESFÍNTER */}
            <div className="bg-white rounded-[2.5rem] p-7 shadow-sm border border-slate-100 flex items-center gap-4">
              <div className="p-3 bg-emerald-50 text-emerald-600 rounded-2xl"><Baby size={28} /></div>
              <div>
                <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Control de Esfínter</p>
                <p className="text-base font-bold text-slate-800">{reporte.esfinter || 'Sin reporte'}</p>
              </div>
            </div>

            {/* FOTOS DE AWS */}
            {reporte.fotos?.length > 0 && (
              <div className="bg-white rounded-[2.5rem] p-7 shadow-sm border border-slate-100">
                <div className="flex items-center gap-3 mb-5">
                  <div className="p-2 bg-rose-100 text-rose-600 rounded-xl"><Camera size={20} /></div>
                  <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Evidencia del Día</h3>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  {reporte.fotos.map((url, idx) => (
                    <div 
                      key={idx} 
                      className="aspect-square rounded-[2rem] overflow-hidden bg-slate-100 border border-slate-100 cursor-zoom-in"
                      onClick={() => setFotoSeleccionada(url)}
                    >
                      <img src={url} alt="Evidencia" className="w-full h-full object-cover" />
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* NOTAS DE LA MAESTRA */}
            {reporte.observaciones && (
              <div className="bg-slate-900 text-white rounded-[2.5rem] p-8 shadow-2xl relative overflow-hidden">
                <div className="absolute top-0 right-0 p-4 opacity-10"><Info size={80} /></div>
                <div className="relative z-10">
                  <div className="flex items-center gap-2 mb-4 text-violet-400">
                    <h3 className="font-black uppercase text-[10px] tracking-[0.2em]">Observaciones Diarias</h3>
                  </div>
                  <p className="text-base leading-relaxed italic font-medium opacity-95">"{reporte.observaciones}"</p>
                </div>
              </div>
            )}
          </>
        )}
      </div>
      
      {/* FOOTER */}
      <footer className="text-center py-10">
        <p className="text-[9px] font-black text-slate-300 uppercase tracking-[0.3em]">BioSafe Security System</p>
      </footer>
    </div>
  );
};

const ItemReporte = ({ label, valor, color }) => (
  <div className="flex gap-5">
    <div className="flex flex-col items-center">
      <div className={`w-3.5 h-3.5 rounded-full border-2 border-white shadow-sm ${valor ? color : 'bg-slate-200'}`}></div>
      <div className="w-0.5 h-full bg-slate-100 mt-1"></div>
    </div>
    <div className="pb-2">
      <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest mb-0.5">{label}</p>
      <p className={`text-sm font-black uppercase ${valor ? 'text-slate-800' : 'text-slate-300 italic'}`}>
        {valor || 'Pendiente'}
      </p>
    </div>
  </div>
);

export default ReportePublico;