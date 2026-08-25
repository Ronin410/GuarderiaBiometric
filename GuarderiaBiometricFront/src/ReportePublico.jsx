import React, { useEffect, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import axios from 'axios';
import {
  Clock, Heart, ShieldCheck, Calendar as CalendarIcon, X
} from 'lucide-react';
import { hoyLocal } from './utils/fecha';
import ReporteDiario from './components/ReporteDiario';

const API_URL = import.meta.env.VITE_API_URL || 'https://guarderiabiometricback.onrender.com';

const ReportePublico = () => {
  const { token } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Estados
  const [reporte, setReporte] = useState(null);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState("");
  const [fotoSeleccionada, setFotoSeleccionada] = useState(null);
  
  // Si la URL no trae fecha, usamos la de hoy en formato Local
  const fechaUrl = searchParams.get("fecha") || hoyLocal();
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
          <div className="w-12 h-12 border-4 border-brand-600 border-t-transparent rounded-full animate-spin mx-auto"></div>
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
              <div className="bg-brand-600 p-2.5 rounded-xl shadow-lg shadow-brand-200">
                <ShieldCheck size={24} className="text-white" />
              </div>
              <div>
                <h1 className="text-xl font-black text-slate-900 uppercase tracking-tighter leading-none">Pasitos</h1>
                <p className="text-[9px] font-bold text-brand-500 uppercase tracking-widest">Reporte Diario</p>
              </div>
            </div>
            <div className="text-rose-500 bg-rose-50 p-2 rounded-full"><Heart size={20} fill="currentColor" /></div>
          </div>

          <div className="space-y-4">
            <h2 className="text-3xl font-black text-slate-900 uppercase tracking-tighter">
              {reporte?.hijo_nombre || "Alumno"}
            </h2>
            <div className="relative">
              <CalendarIcon className="absolute left-4 top-1/2 -translate-y-1/2 text-brand-600" size={18} />
              <input 
                type="date" 
                value={fechaSeleccionada}
                onChange={handleCambioFecha}
                className="w-full bg-slate-50 border-none rounded-2xl py-4 pl-12 pr-4 text-sm font-black text-slate-700 focus:ring-2 focus:ring-brand-500 transition-all uppercase"
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
          <ReporteDiario
            reporte={reporte}
            onFotoClick={setFotoSeleccionada}
            tituloObservaciones="Observaciones Diarias"
          />
        )}
      </div>
      
      {/* FOOTER */}
      <footer className="text-center py-10">
        <p className="text-[9px] font-black text-slate-300 uppercase tracking-[0.3em]">Protegido por Pasitos</p>
      </footer>
    </div>
  );
};

export default ReportePublico;