import React, { useEffect, useState } from 'react';
import api from './axiosConfig';
import { 
  Utensils, Moon, Camera, Clock, 
  ChevronLeft, Heart, CheckCircle2, 
  AlertCircle, ShieldCheck, Coffee,
  Baby, Info, Calendar as CalendarIcon,
  X // Importamos el icono de cerrar
} from 'lucide-react';

const VistaPadreDetalle = ({ hijoId, nombreHijo, onVolver }) => {
  const [reporte, setReporte] = useState(null);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState("");
  const [fechaSeleccionada, setFechaSeleccionada] = useState(new Date().toISOString().split('T')[0]);
  
  // ESTADO PARA LA FOTO EN GRANDE
  const [fotoSeleccionada, setFotoSeleccionada] = useState(null);

  const fetchDetalle = async (fecha) => {
    try {
      setLoading(true);
      const res = await api.get(`/seguimiento/${hijoId}?fecha=${fecha}`);
      setReporte(res.data);
      setErrorMsg("");
    } catch (err) {
      console.error("Error al obtener el reporte", err);
      setReporte(null);
      setErrorMsg(err.response?.data?.error || "No hay reporte para esta fecha.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDetalle(fechaSeleccionada);
  }, [hijoId, fechaSeleccionada]);

  const handleCambioFecha = (e) => {
    setFechaSeleccionada(e.target.value);
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center p-10 text-center">
        <div className="space-y-4">
          <div className="w-12 h-12 border-4 border-violet-600 border-t-transparent rounded-full animate-spin mx-auto"></div>
          <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Buscando información...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 pb-20 animate-in fade-in duration-500">
      
      {/* MODAL DE FOTO EN GRANDE */}
      {fotoSeleccionada && (
        <div 
          className="fixed inset-0 z-[100] bg-slate-900/90 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in zoom-in duration-200"
          onClick={() => setFotoSeleccionada(null)}
        >
          <button 
            className="absolute top-6 right-6 text-white bg-white/10 p-3 rounded-full hover:bg-white/20 transition-all"
            onClick={() => setFotoSeleccionada(null)}
          >
            <X size={24} />
          </button>
          <img 
            src={fotoSeleccionada} 
            alt="Detalle" 
            className="max-w-full max-h-[85vh] rounded-3xl shadow-2xl object-contain"
            onClick={(e) => e.stopPropagation()} // Evita que el modal se cierre al tocar la imagen
          />
        </div>
      )}

      {/* HEADER CON SELECTOR DE FECHA */}
      <div className="bg-white p-6 pb-8 rounded-b-[3rem] shadow-sm border-b border-slate-100 sticky top-0 z-30">
        <button 
          onClick={onVolver}
          className="flex items-center gap-2 text-slate-400 font-black uppercase text-[10px] tracking-widest mb-6 hover:text-violet-600 transition-colors"
        >
          <ChevronLeft size={16} /> Volver
        </button>

        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="text-3xl font-black text-slate-900 uppercase tracking-tighter">{nombreHijo}</h2>
            <div className="bg-violet-600 p-3 rounded-2xl text-white shadow-lg shadow-violet-200">
              <Heart size={20} fill="currentColor" />
            </div>
          </div>

          <div className="relative">
            <div className="absolute inset-y-0 left-4 flex items-center pointer-events-none text-violet-600">
              <CalendarIcon size={16} />
            </div>
            <input 
              type="date" 
              value={fechaSeleccionada}
              onChange={handleCambioFecha}
              max={new Date().toISOString().split('T')[0]}
              className="w-full bg-slate-50 border-none rounded-2xl py-3 pl-12 pr-4 text-sm font-bold text-slate-700 focus:ring-2 focus:ring-violet-500 transition-all uppercase"
            />
          </div>
        </div>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-4">
        {errorMsg ? (
          <div className="bg-white p-12 rounded-[3rem] border-2 border-dashed border-slate-200 text-center space-y-4">
            <div className="bg-slate-50 w-16 h-16 rounded-full flex items-center justify-center mx-auto text-slate-300">
                <Clock size={32} />
            </div>
            <p className="font-black text-slate-400 uppercase text-[10px] tracking-widest">{errorMsg}</p>
          </div>
        ) : (
          <>
            {/* ... SECCIONES DE ALIMENTACIÓN, SUEÑO Y ESFÍNTER (Igual que antes) ... */}
            <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100">
              <div className="flex items-center gap-2 mb-6">
                <div className="p-2 bg-amber-100 text-amber-600 rounded-lg"><Utensils size={18} /></div>
                <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Alimentación</h3>
              </div>
              <div className="space-y-4">
                <ComidaItem label="Desayuno" valor={reporte.desayuno} />
                <ComidaItem label="Comida" valor={reporte.comida} />
                <ComidaItem label="Merienda" valor={reporte.merienda} />
              </div>
            </div>

            <div className={`rounded-[2.5rem] p-6 border transition-all ${reporte.durmio ? 'bg-violet-600 border-violet-500 text-white shadow-lg shadow-violet-200' : 'bg-white border-slate-100 text-slate-400'}`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className={`p-3 rounded-2xl ${reporte.durmio ? 'bg-violet-500' : 'bg-slate-50 text-slate-300'}`}><Moon size={24} /></div>
                  <div>
                    <p className="font-black uppercase text-xs tracking-widest">Descanso</p>
                    <p className="text-[10px] font-bold uppercase opacity-80">{reporte.durmio ? 'Siesta completada' : 'No reportan siesta'}</p>
                  </div>
                </div>
                {reporte.durmio && <CheckCircle2 size={24} />}
              </div>
            </div>

            <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100 flex items-center gap-4">
              <div className="p-3 bg-blue-50 text-blue-500 rounded-2xl"><Baby size={24} /></div>
              <div>
                <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Esfínter</p>
                <p className="text-sm font-bold text-slate-700">{reporte.esfinter || 'Sin datos'}</p>
              </div>
            </div>

            {/* SECCIÓN DE FOTOS ACTUALIZADA CON CLIC */}
            {reporte.fotos?.length > 0 && (
              <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100">
                <div className="flex items-center gap-2 mb-4">
                  <div className="p-2 bg-rose-100 text-rose-600 rounded-lg"><Camera size={18} /></div>
                  <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Fotos de la fecha</h3>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  {reporte.fotos.map((url, index) => (
                    <div 
                      key={index} 
                      className="aspect-square rounded-3xl overflow-hidden border border-slate-100 shadow-inner bg-slate-50 cursor-zoom-in group relative"
                      onClick={() => setFotoSeleccionada(url)}
                    >
                      <img src={url} alt="Evidencia" className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110" />
                      <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors flex items-center justify-center">
                         <Camera className="text-white opacity-0 group-hover:opacity-100" size={24} />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* OBSERVACIONES */}
            {reporte.observaciones && (
              <div className="bg-slate-900 text-white rounded-[2.5rem] p-7 shadow-xl">
                <div className="flex items-center gap-2 mb-3 text-violet-400">
                  <Info size={18} />
                  <h3 className="font-black uppercase text-[10px] tracking-[0.2em]">Nota de la Maestra</h3>
                </div>
                <p className="text-sm leading-relaxed italic opacity-90">"{reporte.observaciones}"</p>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

const ComidaItem = ({ label, valor }) => (
  <div className="flex gap-4">
    <div className="flex flex-col items-center">
      <div className={`w-3 h-3 rounded-full ${valor ? 'bg-emerald-500' : 'bg-slate-200'}`}></div>
      <div className="w-0.5 h-full bg-slate-100 mt-1"></div>
    </div>
    <div className="pb-2">
      <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">{label}</p>
      <p className="text-sm font-bold text-slate-700">{valor || 'Pendiente'}</p>
    </div>
  </div>
);

export default VistaPadreDetalle;