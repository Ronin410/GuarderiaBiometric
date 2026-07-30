import React, { useEffect, useState } from 'react';
import api from './axiosConfig';
import {
  Clock,
  ChevronLeft, Heart, CheckCircle2,
  Calendar as CalendarIcon,
  X, // Importamos el icono de cerrar
  ClipboardList, IdCard, Wallet,
  Cake, MapPin, Phone, XCircle
} from 'lucide-react';
import { hoyLocal } from './utils/fecha';
import ReporteDiario from './components/ReporteDiario';

const ESTADO_PAGO_INFO = {
  pagado: { label: 'Pagado', color: 'bg-emerald-100 text-emerald-700 border-emerald-200', icon: CheckCircle2 },
  parcial: { label: 'Parcial', color: 'bg-amber-100 text-amber-700 border-amber-200', icon: Clock },
  pendiente: { label: 'Pendiente', color: 'bg-slate-100 text-slate-500 border-slate-200', icon: Clock },
  vencido: { label: 'Vencido', color: 'bg-rose-100 text-rose-700 border-rose-200', icon: XCircle },
};

const VistaPadreDetalle = ({ hijoId, nombreHijo, expediente, onVolver }) => {
  const [vista, setVista] = useState('bitacora'); // bitacora | expediente | pagos

  const [reporte, setReporte] = useState(null);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState("");
  const [fechaSeleccionada, setFechaSeleccionada] = useState(hoyLocal());

  const [historialPagos, setHistorialPagos] = useState([]);
  const [loadingPagos, setLoadingPagos] = useState(false);

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
    if (vista === 'bitacora') fetchDetalle(fechaSeleccionada);
  }, [hijoId, fechaSeleccionada, vista]);

  useEffect(() => {
    if (vista !== 'pagos') return;
    const cargarHistorialPagos = async () => {
      setLoadingPagos(true);
      try {
        const res = await api.get('/padre/mis-pagos/historial', { params: { hijo_id: hijoId } });
        setHistorialPagos(Array.isArray(res.data) ? res.data : []);
      } catch (err) {
        console.error("Error al obtener el historial de pagos", err);
        setHistorialPagos([]);
      } finally {
        setLoadingPagos(false);
      }
    };
    cargarHistorialPagos();
  }, [hijoId, vista]);

  const handleCambioFecha = (e) => {
    setFechaSeleccionada(e.target.value);
  };

  if (loading && vista === 'bitacora') {
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

          {/* SELECTOR DE PESTAÑAS */}
          <div className="flex bg-slate-100 p-1.5 rounded-2xl">
            {[
              { key: 'bitacora', label: 'Hoy', icon: ClipboardList },
              { key: 'expediente', label: 'Expediente', icon: IdCard },
              { key: 'pagos', label: 'Pagos', icon: Wallet },
            ].map((tab) => {
              const Icono = tab.icon;
              return (
                <button
                  key={tab.key}
                  onClick={() => setVista(tab.key)}
                  className={`flex-1 py-2.5 rounded-xl flex items-center justify-center gap-1.5 font-black text-[10px] uppercase transition-all ${vista === tab.key ? 'bg-white text-violet-600 shadow-sm' : 'text-slate-400'}`}
                >
                  <Icono size={13} /> {tab.label}
                </button>
              );
            })}
          </div>

          {vista === 'bitacora' && (
            <div className="relative">
              <div className="absolute inset-y-0 left-4 flex items-center pointer-events-none text-violet-600">
                <CalendarIcon size={16} />
              </div>
              <input
                type="date"
                value={fechaSeleccionada}
                onChange={handleCambioFecha}
                max={hoyLocal()}
                className="w-full bg-slate-50 border-none rounded-2xl py-3 pl-12 pr-4 text-sm font-bold text-slate-700 focus:ring-2 focus:ring-violet-500 transition-all uppercase"
              />
            </div>
          )}
        </div>
      </div>

      {vista === 'expediente' && (
        <div className="max-w-md mx-auto p-4 space-y-4">
          <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100 space-y-5">
            <div className="flex items-center gap-2">
              <div className="p-2 bg-violet-100 text-violet-600 rounded-lg"><IdCard size={18} /></div>
              <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Expediente</h3>
            </div>
            <div className="flex items-center gap-4">
              <div className="p-3 bg-slate-50 text-slate-400 rounded-2xl"><Cake size={20} /></div>
              <div>
                <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Fecha de nacimiento</p>
                <p className="text-sm font-bold text-slate-700">{expediente?.fechaNacimiento || 'No registrada'}</p>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="p-3 bg-slate-50 text-slate-400 rounded-2xl"><MapPin size={20} /></div>
              <div>
                <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Dirección</p>
                <p className="text-sm font-bold text-slate-700">{expediente?.direccion || 'No registrada'}</p>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="p-3 bg-slate-50 text-slate-400 rounded-2xl"><Phone size={20} /></div>
              <div>
                <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Contacto de emergencia</p>
                <p className="text-sm font-bold text-slate-700">
                  {expediente?.contactoEmergenciaNombre
                    ? `${expediente.contactoEmergenciaNombre} · ${expediente.contactoEmergenciaTelefono || 's/n'}`
                    : 'No registrado'}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {vista === 'pagos' && (
        <div className="max-w-md mx-auto p-4 space-y-4">
          {loadingPagos ? (
            <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
          ) : historialPagos.length === 0 ? (
            <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center">
              <Wallet size={40} className="mx-auto text-slate-200 mb-4" />
              <p className="text-slate-400 font-bold uppercase text-[10px]">Sin pagos registrados todavía.</p>
            </div>
          ) : (
            historialPagos.map((p) => (
              <div key={p.id} className="bg-white p-5 rounded-2xl border border-slate-100 flex items-center justify-between">
                <div>
                  <p className="font-black text-sm text-slate-800">${Number(p.monto).toLocaleString('es-MX', { minimumFractionDigits: 2 })} <span className="text-slate-400 font-bold text-xs">· {p.concepto}</span></p>
                  <p className="text-[10px] text-slate-400 font-bold uppercase mt-1">{p.periodo} · {p.fecha_pago} · {p.metodo_pago}</p>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {vista === 'bitacora' && (
      <div className="max-w-md mx-auto p-4 space-y-4">
        {errorMsg ? (
          <div className="bg-white p-12 rounded-[3rem] border-2 border-dashed border-slate-200 text-center space-y-4">
            <div className="bg-slate-50 w-16 h-16 rounded-full flex items-center justify-center mx-auto text-slate-300">
                <Clock size={32} />
            </div>
            <p className="font-black text-slate-400 uppercase text-[10px] tracking-widest">{errorMsg}</p>
          </div>
        ) : (
          <ReporteDiario
            reporte={reporte}
            onFotoClick={setFotoSeleccionada}
            tituloObservaciones="Nota de la Maestra"
          />
        )}
      </div>
      )}
    </div>
  );
};

export default VistaPadreDetalle;