import React, { useEffect, useState } from 'react';
import api from './axiosConfig';
import { 
  User, 
  ChevronRight, 
  Heart, 
  LogOut, 
  Baby, 
  Bell,
  LayoutDashboard
} from 'lucide-react';
import VistaPadreDetalle from './VistaPadreDetalle'; 

const DashboardPadre = () => {
  const [hijos, setHijos] = useState([]);
  const [hijoSeleccionado, setHijoSeleccionado] = useState(null);
  const [loading, setLoading] = useState(true);
  const [usuarioNombre, setUsuarioNombre] = useState('');

  useEffect(() => {
    const cargarDatosIniciales = async () => {
      try {
        setLoading(true);
        
        // 1. Obtener hijos desde tu endpoint de Go
        const resHijos = await api.get('/padre/0/hijos');
        
        // CORRECCIÓN DE MAPEO: Aseguramos que 'nombre' tome el valor de 'nombre_niño'
        const hijosFormateados = (resHijos.data || []).map(h => ({
          id: h.id,
          // Intentamos leer nombre_niño (SQL) o nombre (JSON genérico)
          nombre: h.nombre_niño || h.nombre || "Sin nombre", 
          activo: h.activo
        }));

        setHijos(hijosFormateados);
        
        // 2. Recuperar el nombre del usuario logueado
        // Asegúrate de que en tu App.js guardes el nombre al hacer login
        const storedName = localStorage.getItem('username');
        setUsuarioNombre(storedName || 'Familia');

      } catch (err) {
        console.error("Error al cargar el dashboard de padre", err);
      } finally {
        setLoading(false);
      }
    };
    cargarDatosIniciales();
  }, []);

  const handleLogout = () => {
    localStorage.clear();
    window.location.href = '/'; 
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-50 flex flex-col items-center justify-center p-10 text-center">
        <div className="w-16 h-16 border-4 border-violet-200 border-t-violet-600 rounded-full animate-spin mb-4"></div>
        <p className="font-black text-slate-400 uppercase tracking-[0.2em] text-xs">Sincronizando Familia...</p>
      </div>
    );
  }

  // Navegación a la vista de detalle
  if (hijoSeleccionado) {
    return (
      <VistaPadreDetalle 
        hijoId={hijoSeleccionado.id} 
        nombreHijo={hijoSeleccionado.nombre}
        onVolver={() => setHijoSeleccionado(null)} 
      />
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 pb-10">
      {/* NAVBAR */}
      <div className="bg-white px-6 py-4 flex justify-between items-center border-b border-slate-100 sticky top-0 z-10 shadow-sm">
        <div className="flex items-center gap-2">
          <div className="bg-violet-600 p-2 rounded-xl text-white">
            <LayoutDashboard size={18} />
          </div>
          <span className="font-black text-slate-900 uppercase tracking-tighter text-sm">BIOSAFE <span className="text-violet-600">FAMILIA</span></span>
        </div>
        <button onClick={handleLogout} className="p-2 text-slate-400 hover:text-rose-500 transition-colors">
          <LogOut size={20} />
        </button>
      </div>

      <div className="max-w-md mx-auto p-6 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
        
        {/* SALUDO - Ahora dinámico */}
        <div className="space-y-2 text-center pt-4">
          <h2 className="text-4xl font-black text-slate-900 uppercase tracking-tighter leading-none">
            Hola, <br/><span className="text-violet-600">{usuarioNombre}</span>
          </h2>
          <p className="text-slate-400 font-bold uppercase text-[10px] tracking-[0.15em]">
            ¿De quién deseas ver el reporte hoy?
          </p>
        </div>

        {/* INFO CARD */}
        <div className="bg-violet-50 border border-violet-100 p-4 rounded-[2rem] flex items-center gap-4">
          <div className="bg-white p-3 rounded-2xl text-violet-600 shadow-sm">
            <Bell size={20} />
          </div>
          <p className="text-[10px] font-black text-violet-800 uppercase leading-tight tracking-tight">
            Las bitácoras se actualizan en tiempo real por las maestras.
          </p>
        </div>

        {/* LISTADO DE NIÑOS */}
        <div className="space-y-4">
          {hijos.length === 0 ? (
            <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center">
              <Baby size={40} className="mx-auto text-slate-200 mb-4" />
              <p className="text-slate-400 font-bold uppercase text-[10px]">No se encontraron niños vinculados.</p>
            </div>
          ) : (
            hijos.map((hijo) => (
              <button
                key={hijo.id}
                onClick={() => setHijoSeleccionado(hijo)}
                className="w-full bg-white p-6 rounded-[2.5rem] border border-slate-100 shadow-md hover:shadow-xl transition-all flex items-center justify-between group active:scale-95"
              >
                <div className="flex items-center gap-5">
                  <div className="w-16 h-16 bg-slate-50 rounded-[1.5rem] flex items-center justify-center text-slate-300 group-hover:bg-violet-100 group-hover:text-violet-600 transition-all border border-slate-100">
                    <User size={32} />
                  </div>

                  <div className="text-left">
                    <h4 className="font-black text-slate-900 uppercase text-lg tracking-tight group-hover:text-violet-600 transition-colors">
                      {hijo.nombre}
                    </h4>
                    <div className="flex items-center gap-1 text-violet-500">
                      <Heart size={10} fill="currentColor" />
                      <p className="text-[9px] font-black uppercase tracking-widest">Ver Bitácora Diaria</p>
                    </div>
                  </div>
                </div>

                <div className="bg-slate-50 p-3 rounded-2xl text-slate-300 group-hover:bg-violet-600 group-hover:text-white transition-all shadow-sm">
                  <ChevronRight size={20} />
                </div>
              </button>
            ))
          )}
        </div>

        {/* FOOTER */}
        <div className="text-center pt-8">
            <p className="text-[8px] font-black text-slate-300 uppercase tracking-[0.4em]">Protegido por BioSafe System</p>
        </div>
      </div>
    </div>
  );
};

export default DashboardPadre;