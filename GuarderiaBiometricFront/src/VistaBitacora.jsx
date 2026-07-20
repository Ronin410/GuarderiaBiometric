import React, { useEffect, useState } from 'react';
import api from './axiosConfig'; 
import { 
  Calendar, User, CheckCircle, ShieldAlert, Clock, 
  Search, RefreshCw, MessageSquare
} from 'lucide-react';
import Swal from 'sweetalert2';
import withReactContent from 'sweetalert2-react-content';
import FormularioBitacora from './FormularioBitacora';

const MySwal = withReactContent(Swal);

const VistaBitacora = () => {
  const getFechaLocalCuliacan = () => {
    const fecha = new Date();
    const offset = fecha.getTimezoneOffset() * 60000;
    return new Date(fecha - offset).toISOString().split('T')[0];
  };

  const [niños, setNiños] = useState([]);
  const [loading, setLoading] = useState(true);
  const [fechaFiltro, setFechaFiltro] = useState(getFechaLocalCuliacan());
  const [busqueda, setBusqueda] = useState("");
  const [niñoSeleccionado, setNiñoSeleccionado] = useState(null);

  const fetchEstatus = async () => {
    setLoading(true);
    try {
      const res = await api.get('/bitacora', { params: { fecha: fechaFiltro } });
      setNiños(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error("Error al obtener estatus", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchEstatus(); }, [fechaFiltro]);

  const formatearHora = (fechaStr) => {
    if (!fechaStr || fechaStr === '--:--') return '--:--';
    try {
      const isoStr = fechaStr.includes(' ') ? fechaStr.replace(' ', 'T') : fechaStr;
      const date = new Date(isoStr);
      return isNaN(date.getTime()) ? fechaStr : date.toLocaleTimeString('es-MX', { hour: '2d-digit', minute: '2d-digit', hour12: true });
    } catch (e) { return fechaStr; }
  };

  const handleCambiarEstatus = async (niño) => {
    let nuevoEstatus = (!niño.estatus || niño.estatus === 'AUSENTE' || niño.estatus === 'SALIDA') ? 'ENTRADA' : 'SALIDA';
    let colorConfirmacion = nuevoEstatus === 'ENTRADA' ? '#10b981' : '#f97316';

    const result = await MySwal.fire({
      title: <p className="text-xl font-black text-slate-900 uppercase tracking-tighter">¿Cambiar Estatus?</p>,
      html: <p className="text-slate-600 text-sm uppercase font-bold">Actualizar a {nuevoEstatus} para {niño.hijo}</p>,
      icon: 'question',
      showCancelButton: true,
      confirmButtonText: 'SÍ, ACTUALIZAR',
      confirmButtonColor: colorConfirmacion,
      cancelButtonColor: '#64748b',
    });

    if (result.isConfirmed) {
      try {
        await api.post('/admin/forzar-estatus', { hijo_id: niño.id, tipo_movimiento: nuevoEstatus });
        fetchEstatus(); 
      } catch (err) { console.error(err); }
    }
  };

  const filtrados = niños.filter(n => (n.hijo || "").toLowerCase().includes(busqueda.toLowerCase()));

  const getBadgeStyle = (estatus) => {
    if (estatus === 'ENTRADA') return 'bg-emerald-500 text-white border-emerald-400 shadow-emerald-200/50';
    if (estatus === 'SALIDA') return 'bg-orange-500 text-white border-orange-400 shadow-orange-200/50';
    return 'bg-slate-100 text-slate-500 border-slate-200';
  };

  if (niñoSeleccionado) {
    return (
      <div className="min-h-screen bg-slate-50/50 p-8">
        <div className="max-w-4xl mx-auto">
          <button onClick={() => { setNiñoSeleccionado(null); fetchEstatus(); }} className="mb-6 flex items-center gap-2 text-violet-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
            ← Volver a la lista
          </button>
          {/* PASAMOS TODA LA DATA DEL NIÑO AL FORMULARIO */}
          <FormularioBitacora 
            niñoId={niñoSeleccionado.id} 
            nombreNiño={niñoSeleccionado.hijo} 
            datosEntrada={niñoSeleccionado} // Enviamos aseado, golpe y obs_asistencia
            onCerrar={() => { setNiñoSeleccionado(null); fetchEstatus(); }} 
          />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50/50 -m-4 p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        
        {/* CABECERA */}
        <div className="flex flex-col md:flex-row gap-6 items-center justify-between bg-white p-8 rounded-[2.5rem] border border-slate-200 shadow-xl shadow-slate-200/60">
          <div className="flex flex-col gap-1">
            <h3 className="text-3xl font-black text-slate-900 uppercase tracking-tighter">Bitácora</h3>
            <div className="h-1 w-20 bg-violet-600 rounded-full"></div>
          </div>

          <div className="flex flex-col md:flex-row gap-4 w-full md:w-auto">
            <div className="relative flex items-center bg-slate-50 rounded-2xl border border-slate-200 px-4">
              <Calendar size={20} className="text-slate-400" />
              <input type="date" value={fechaFiltro} onChange={(e) => setFechaFiltro(e.target.value)} className="bg-transparent p-4 text-slate-900 outline-none font-bold text-sm" />
            </div>
            <div className="relative flex-1 md:w-80">
              <Search className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-400" size={20}/>
              <input type="text" placeholder="Buscar niño..." value={busqueda} onChange={(e) => setBusqueda(e.target.value)} className="w-full bg-slate-100 border border-slate-200 pl-14 pr-6 py-4 rounded-2xl outline-none focus:ring-2 focus:ring-violet-500 font-medium" />
            </div>
            <button onClick={fetchEstatus} className="p-4 bg-violet-600 text-white rounded-2xl hover:bg-violet-700 shadow-lg">
              <RefreshCw size={24} className={loading ? "animate-spin" : ""} />
            </button>
          </div>
        </div>

        {/* GRID DE NIÑOS */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {loading ? (
            <div className="col-span-full py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sincronizando...</div>
          ) : (
            filtrados.map(niño => {
              const puedeLlenar = niño.estatus === 'ENTRADA' || niño.estatus === 'SALIDA';

              return (
                <div 
                  key={niño.id} 
                  onClick={() => puedeLlenar && setNiñoSeleccionado(niño)}
                  className={`bg-white border border-slate-100 p-6 rounded-[2.5rem] shadow-md transition-all group flex flex-col justify-between border-b-4 
                    ${puedeLlenar ? 'hover:shadow-xl hover:border-b-violet-500 cursor-pointer active:scale-[0.98]' : 'cursor-default opacity-60'}`}
                >
                  <div>
                    <div className="flex items-start gap-4 mb-4">
                      <div className={`p-4 rounded-2xl transition-all shrink-0 bg-slate-100 text-slate-600 ${puedeLlenar ? 'group-hover:bg-violet-600 group-hover:text-white' : ''}`}>
                        <User size={24} />
                      </div>
                      <div className="flex-1 min-w-0 self-center">
                        <h4 className="font-black text-slate-900 text-base uppercase tracking-tight leading-tight">
                          {niño.hijo}
                        </h4>
                      </div>
                    </div>

                    <button
                      onClick={(e) => { e.stopPropagation(); handleCambiarEstatus(niño); }}
                      className={`w-full py-3 rounded-2xl text-[11px] font-black border uppercase tracking-[0.2em] mb-4 transition-all active:scale-95 shadow-lg ${getBadgeStyle(niño.estatus)}`}
                    >
                      {(!niño.estatus || niño.estatus === 'AUSENTE') ? '🏠 En Casa' : niño.estatus}
                    </button>

                    {/* INDICADORES DE ENTRADA (ESTADO FÍSICO) */}
                    <div className="grid grid-cols-2 gap-3 mb-4">
                      <div className={`py-3 rounded-2xl flex flex-col items-center gap-1 border-2 transition-colors ${niño.aseado ? 'bg-blue-50 border-blue-200 text-blue-500' : 'bg-rose-50 border-rose-500 text-rose-600 animate-pulse'}`}>
                        <CheckCircle size={18}/>
                        <span className="text-[9px] font-black uppercase">{niño.aseado ? 'Limpio' : 'No Aseado'}</span>
                      </div>
                      <div className={`py-3 rounded-2xl flex flex-col items-center gap-1 border-2 transition-colors ${niño.golpe ? 'bg-amber-50 border-amber-500 text-amber-600 animate-pulse' : 'bg-slate-50 border-slate-100 text-slate-300'}`}>
                        <ShieldAlert size={18}/>
                        <span className="text-[9px] font-black uppercase">{niño.golpe ? 'Trae Golpe' : 'Normal'}</span>
                      </div>
                    </div>

                    {/* VISTA PREVIA DE OBSERVACIÓN DE PAPÁS */}
                    {niño.obs_asistencia && (
                      <div className="bg-slate-50 p-3 rounded-xl mb-4 border border-dashed border-slate-200">
                        <p className="text-[10px] text-slate-400 font-black uppercase flex items-center gap-1 mb-1">
                          <MessageSquare size={10} /> Obs. Papá:
                        </p>
                        <p className="text-[10px] text-slate-600 font-bold line-clamp-2 italic leading-tight">
                          "{niño.obs_asistencia}"
                        </p>
                      </div>
                    )}
                  </div>

                  <div className="pt-4 border-t border-slate-100 flex items-center justify-center gap-2 mt-auto">
                    <Clock size={14} className="text-violet-500"/>
                    <span className="text-xs font-bold text-slate-500">{formatearHora(niño.fecha_hora)}</span>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};

export default VistaBitacora;