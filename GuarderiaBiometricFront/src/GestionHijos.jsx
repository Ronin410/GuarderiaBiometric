import React, { useState, useEffect } from 'react';
// Cambiamos axios por la instancia personalizada
import api from './axiosConfig'; 
import {
  UserPlus, Search, Baby, Save, X, Edit3,
  Loader2, Check, RotateCcw, Eye, EyeOff, UserX, Link2Off, RefreshCw,
  Download, Trash2
} from 'lucide-react';
import { mostrarExito, mostrarError, confirmar } from './utils/alertas';

const GestionHijos = ({ padreId, nombrePadre, onFinalizar }) => {
  const [hijosRelacionados, setHijosRelacionados] = useState([]);
  const [busqueda, setBusqueda] = useState('');
  const [sugerencias, setSugerencias] = useState([]);
  const [nuevoHijoNombre, setNuevoHijoNombre] = useState('');
  const [loading, setLoading] = useState(false);
  
  const [verBajas, setVerBajas] = useState(false);
  
  const [nombreTutorEdit, setNombreTutorEdit] = useState(nombrePadre);
  const [editandoTutor, setEditandoTutor] = useState(false);

  const [editandoHijoId, setEditandoHijoId] = useState(null);
  const [nombreHijoEdit, setNombreHijoEdit] = useState('');

  const cargarHijosActuales = async () => {
    if (!padreId) return;
    try {
      const res = await api.get(`/padre/${padreId}/hijos`);
      const hijosMapeados = res.data.map(h => ({ 
        id: h.id, 
        nombre_niño: h.nombre || h.nombre_niño, 
        activo: h.activo !== undefined ? h.activo : true,
        esNuevo: false, 
        persistente: true 
      }));
      setHijosRelacionados(hijosMapeados);
    } catch (err) {
      console.error("Error cargando hijos:", err);
    }
  };

  useEffect(() => {
    cargarHijosActuales();
  }, [padreId]);

  useEffect(() => {
    const delayDebounceFn = setTimeout(async () => {
      if (busqueda.trim().length > 2) {
        try {
          const res = await api.get(`/buscar-hijos?q=${busqueda}`);
          setSugerencias(Array.isArray(res.data) ? res.data : []);
        } catch {
          setSugerencias([]);
        }
      } else {
        setSugerencias([]);
      }
    }, 300);
    return () => clearTimeout(delayDebounceFn);
  }, [busqueda]);

  const manejarActualizarTutor = async () => {
    if (!nombreTutorEdit.trim()) return;
    setLoading(true);
    try {
      await api.post(`/actualizar-padre`, {
        id: parseInt(padreId),
        nombre: nombreTutorEdit.trim()
      });
      setEditandoTutor(false);
    } catch (err) {
      console.error("Error al actualizar tutor:", err);
      mostrarError("Error al actualizar tutor");
    } finally {
      setLoading(false);
    }
  };

  const manejarActualizarNombreHijo = async (id) => {
    if (!nombreHijoEdit.trim()) return;
    try {
      await api.put(`/hijos/${id}`, {
        nombre: nombreHijoEdit.trim()
      });
      setEditandoHijoId(null);
      cargarHijosActuales();
    } catch (err) {
      console.error("Error al actualizar el nombre del niño:", err);
      mostrarError("Error al actualizar el nombre del niño");
    }
  };

  const manejarBajaHijo = async (hijo) => {
    const ok = await confirmar(`¿Seguro que quieres DESACTIVAR a ${hijo.nombre_niño}?`, "Dar de baja");
    if (!ok) return;
    try {
      await api.patch(`/hijos/${hijo.id}/desactivar`);
      cargarHijosActuales();
    } catch (err) {
      console.error("Error al procesar la baja:", err);
      mostrarError("Error al procesar la baja");
    }
  };

  const manejarDesvincular = async (hijo) => {
    const ok = await confirmar(`¿Quieres quitar a ${hijo.nombre_niño} de la lista de ${nombreTutorEdit}?`, "Desvincular");
    if (!ok) return;

    setLoading(true);
    try {
      await api.post(`/desvincular-hijo`, {
        padre_id: parseInt(padreId),
        hijo_id: parseInt(hijo.id)
      });
      mostrarExito("Desvinculación exitosa");
      cargarHijosActuales();
    } catch (err) {
      console.error(err);
      mostrarError("Error al desvincular: " + (err.response?.data?.mensaje || "Error desconocido"));
    } finally {
      setLoading(false);
    }
  };

  // Derechos ARCO (LFPDPPP): un padre puede pedir copia de sus datos o que
  // se borren. El borrado solo alcanza al tutor (perfil, rostro, cuenta) —
  // la bitácora/asistencia/pagos de sus hijos se conserva, es su expediente.
  const manejarExportarDatosArco = async () => {
    try {
      const res = await api.get(`/admin/familias/${padreId}/exportar`);
      const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const enlace = document.createElement('a');
      enlace.href = url;
      enlace.download = `expediente_padre_${padreId}.json`;
      document.body.appendChild(enlace);
      enlace.click();
      enlace.remove();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Error al exportar datos del tutor:", err);
      mostrarError("No se pudo exportar la información del tutor");
    }
  };

  const manejarEliminarDatosArco = async () => {
    const ok = await confirmar(
      `Se eliminará el perfil, rostro biométrico y cuenta de ${nombreTutorEdit}. La bitácora, asistencia y pagos de sus hijos NO se borran. Esta acción no se puede deshacer.`,
      "Eliminar datos del tutor"
    );
    if (!ok) return;

    setLoading(true);
    try {
      await api.delete(`/admin/familias/${padreId}`);
      mostrarExito("Los datos del tutor fueron eliminados");
      if (typeof onFinalizar === 'function') {
        onFinalizar();
      }
    } catch (err) {
      console.error("Error al eliminar datos del tutor:", err);
      mostrarError("No se pudo eliminar al tutor");
    } finally {
      setLoading(false);
    }
  };

  const manejarRegenerarToken = async (hijo) => {
    const ok = await confirmar(
      `El enlace de bitácora que ya se compartió por WhatsApp para ${hijo.nombre_niño} dejará de funcionar. ¿Generar uno nuevo?`,
      "Regenerar enlace"
    );
    if (!ok) return;
    try {
      await api.post(`/hijos/${hijo.id}/regenerar-token`);
      mostrarExito("El enlace anterior fue invalidado. El próximo reporte que envíes usará el nuevo.");
    } catch (err) {
      console.error("Error al regenerar el enlace:", err);
      mostrarError("Error al regenerar el enlace");
    }
  };

  const manejarAltaHijo = async (hijo) => {
    const ok = await confirmar(`¿Activar a ${hijo.nombre_niño}?`, "Reactivar alumno");
    if (!ok) return;
    try {
      await api.patch(`/hijos/${hijo.id}/activar`);
      cargarHijosActuales();
    } catch (err) {
      console.error("Error al reactivar:", err);
      mostrarError("Error al reactivar");
    }
  };

  const agregarSugerencia = (hijo) => {
    if (!hijosRelacionados.find(h => h.id === hijo.id)) {
      setHijosRelacionados([...hijosRelacionados, { 
        id: hijo.id,
        nombre_niño: hijo.nombre_niño,
        activo: true,
        esNuevo: false, 
        persistente: false 
      }]);
    }
    setBusqueda('');
    setSugerencias([]);
  };

  const prepararNuevoHijo = () => {
    if (nuevoHijoNombre.trim()) {
      setHijosRelacionados([...hijosRelacionados, { 
        nombre_niño: nuevoHijoNombre.trim(), 
        activo: true,
        esNuevo: true,
        persistente: false,
        id: Date.now() 
      }]);
      setNuevoHijoNombre('');
    }
  };

  const guardarRelaciones = async () => {
    setLoading(true);
    try {
      const nuevosPorGuardar = hijosRelacionados.filter(h => !h.persistente);
      for (const hijo of nuevosPorGuardar) {
        let idHijoFinal = hijo.id;
        if (hijo.esNuevo) {
          const resHijo = await api.post(`/registrar-hijo`, { 
            nombre_niño: hijo.nombre_niño 
          });
          idHijoFinal = resHijo.data.id;
        }
        await api.post(`/vincular-tutor`, { 
          padre_id: parseInt(padreId), 
          hijo_id: idHijoFinal 
        });
      }
      mostrarExito("Cambios guardados");
      if (typeof onFinalizar === 'function') {
        onFinalizar();
      } else {
        cargarHijosActuales();
      }
    } catch (error) {
      console.error("Error al guardar relaciones:", error);
      mostrarError("Error al guardar");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-4 md:p-8 bg-white text-slate-900 max-h-[90vh] overflow-y-auto rounded-[2rem] md:rounded-[2.5rem] relative">
      
      {/* SECCIÓN TUTOR */}
      <div className="flex justify-between items-start mb-10 border-b border-slate-100 pb-8 pr-2 md:pr-12">
        <div className="flex-1 w-full">
          <p className="text-brand-600 font-black uppercase text-[10px] tracking-[0.2em] mb-2">Tutor Registrado</p>
          {editandoTutor ? (
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
              <input 
                autoFocus
                value={nombreTutorEdit}
                onChange={(e) => setNombreTutorEdit(e.target.value)}
                className="bg-slate-50 border-2 border-brand-500 rounded-2xl px-5 py-4 text-xl md:text-2xl font-black uppercase outline-none w-full max-w-md shadow-sm"
              />
              <div className="flex gap-2">
                <button onClick={manejarActualizarTutor} className="flex-1 sm:flex-none bg-emerald-500 p-4 rounded-xl text-white hover:bg-emerald-600 shadow-md flex justify-center"><Check size={24}/></button>
                <button onClick={() => {setEditandoTutor(false); setNombreTutorEdit(nombrePadre);}} className="flex-1 sm:flex-none bg-slate-200 p-4 rounded-xl text-slate-600 hover:bg-slate-300 shadow-sm flex justify-center"><X size={24}/></button>
              </div>
            </div>
          ) : (
            <div className="flex items-center gap-5 group">
              <h2 className="text-2xl md:text-4xl font-black uppercase tracking-tighter text-slate-900 leading-none">{nombreTutorEdit}</h2>
              <button onClick={() => setEditandoTutor(true)} className="p-2.5 bg-brand-50 rounded-xl text-brand-600 hover:bg-brand-100 transition-all md:opacity-0 group-hover:opacity-100">
                <Edit3 size={20} />
              </button>
            </div>
          )}
        </div>

        <div className="flex gap-2 shrink-0">
          <button
            onClick={manejarExportarDatosArco}
            title="Exportar datos (ARCO)"
            className="flex items-center gap-2 text-[9px] font-black uppercase bg-slate-100 hover:bg-slate-200 text-slate-600 px-3 py-2.5 rounded-xl transition-all"
          >
            <Download size={14} /> Exportar
          </button>
          <button
            onClick={manejarEliminarDatosArco}
            title="Eliminar datos del tutor (ARCO)"
            className="flex items-center gap-2 text-[9px] font-black uppercase bg-rose-50 hover:bg-rose-100 text-rose-600 px-3 py-2.5 rounded-xl transition-all"
          >
            <Trash2 size={14} /> Eliminar Tutor
          </button>
        </div>
      </div>

      <div className="grid md:grid-cols-2 gap-8 md:gap-12">
        {/* COLUMNA IZQUIERDA: BÚSQUEDA Y NUEVOS */}
        <div className="space-y-6 md:space-y-8">
          <div className="bg-slate-50 p-6 md:p-7 rounded-[2rem] border border-slate-100 shadow-sm">
            <h4 className="text-[10px] font-black text-slate-400 uppercase mb-5 ml-2 tracking-widest">Vincular de la lista general</h4>
            <div className="relative">
              <Search className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input 
                type="text"
                placeholder="Buscar niño..."
                value={busqueda}
                onChange={(e) => setBusqueda(e.target.value)}
                className="w-full bg-white border border-slate-200 rounded-[1.5rem] py-5 pl-14 pr-6 outline-none focus:ring-2 focus:ring-brand-500 transition-all text-slate-900 font-medium shadow-sm"
              />
              {sugerencias.length > 0 && (
                <div className="absolute z-50 w-full mt-3 bg-white border border-slate-200 rounded-[2rem] overflow-hidden shadow-2xl">
                  {sugerencias.map(s => (
                    <button key={s.id} onClick={() => agregarSugerencia(s)} className="w-full p-5 text-left hover:bg-brand-50 flex justify-between items-center border-b border-slate-50 last:border-0 group transition-colors">
                      <span className="font-bold uppercase text-slate-700 group-hover:text-brand-700">{s.nombre_niño}</span>
                      <UserPlus size={20} className="text-brand-400 group-hover:text-brand-600"/>
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          <div className="bg-slate-50 p-6 md:p-7 rounded-[2rem] border border-slate-100 shadow-sm">
            <h4 className="text-[10px] font-black text-slate-400 uppercase mb-5 ml-2 tracking-widest">Nuevo registro de niño</h4>
            <div className="flex gap-3">
              <input 
                type="text"
                placeholder="Nombre del niño..."
                value={nuevoHijoNombre}
                onChange={(e) => setNuevoHijoNombre(e.target.value)}
                className="flex-1 bg-white border border-slate-200 rounded-[1.5rem] p-5 outline-none focus:ring-2 focus:ring-emerald-500 text-slate-900 font-medium shadow-sm"
              />
              <button onClick={prepararNuevoHijo} className="bg-emerald-500 p-5 rounded-[1.5rem] text-white hover:bg-emerald-600 shadow-lg transition-all active:scale-95"><UserPlus size={24}/></button>
            </div>
          </div>
        </div>

        {/* COLUMNA DERECHA: LISTA DE VINCULADOS */}
        <div className="bg-white p-5 md:p-7 rounded-[2.5rem] border-2 border-slate-50 shadow-inner">
          <div className="flex justify-between items-center mb-8 px-2">
            <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-[0.2em]">Familia vinculada</h3>
            <button 
              onClick={() => setVerBajas(!verBajas)}
              className="flex items-center gap-2 text-[9px] font-black uppercase bg-slate-100 hover:bg-slate-200 px-3 py-1.5 rounded-full transition-all text-slate-600"
            >
              {verBajas ? <EyeOff size={12}/> : <Eye size={12}/>}
              {verBajas ? "Ocultar Bajas" : "Ver Bajas"}
            </button>
          </div>

          <div className="space-y-4">
            {hijosRelacionados
              .filter(h => verBajas ? true : h.activo !== false)
              .map((h) => (
              <div key={h.id} className={`flex flex-col sm:flex-row items-center justify-between p-5 rounded-[1.5rem] border transition-all gap-4 ${!h.activo ? 'bg-slate-100 opacity-60 grayscale border-dashed border-slate-300' : 'bg-slate-50 border-slate-100'}`}>
                
                <div className="flex items-center gap-4 w-full flex-1">
                  <div className={`hidden xs:block p-4 rounded-2xl shadow-sm ${!h.activo ? 'bg-slate-200 text-slate-400' : h.persistente ? 'bg-brand-100 text-brand-600' : 'bg-amber-100 text-amber-600 animate-pulse'}`}>
                    <Baby size={24}/>
                  </div>
                  
                  <div className="flex-1 w-full">
                    {editandoHijoId === h.id ? (
                      <div className="flex flex-col gap-2 w-full">
                        <input 
                          autoFocus
                          value={nombreHijoEdit}
                          onChange={(e) => setNombreHijoEdit(e.target.value)}
                          className="bg-white border-2 border-brand-400 rounded-xl px-4 py-3 text-base font-bold uppercase outline-none w-full shadow-inner"
                        />
                        <div className="flex justify-end gap-2">
                          <button onClick={() => manejarActualizarNombreHijo(h.id)} className="bg-emerald-500 text-white p-3 rounded-xl shadow-sm active:scale-90"><Check size={20}/></button>
                          <button onClick={() => setEditandoHijoId(null)} className="bg-slate-200 text-slate-500 p-3 rounded-xl shadow-sm active:scale-90"><X size={20}/></button>
                        </div>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 group/name">
                        <p className={`font-black text-lg uppercase tracking-tight leading-tight ${!h.activo ? 'text-slate-400 line-through' : 'text-slate-800'}`}>
                          {h.nombre_niño}
                        </p>
                        {h.activo && h.persistente && (
                          <button 
                            onClick={() => { setEditandoHijoId(h.id); setNombreHijoEdit(h.nombre_niño); }}
                            className="md:opacity-0 group-hover/name:opacity-100 text-brand-400 transition-opacity p-1"
                          >
                            <Edit3 size={16} />
                          </button>
                        )}
                      </div>
                    )}
                    
                    <div className="flex items-center gap-2 mt-1">
                       {!h.activo ? (
                         <span className="text-[9px] bg-rose-100 text-rose-600 px-2 py-0.5 rounded-full font-black uppercase">Desactivado</span>
                       ) : (
                         <span className={`text-[9px] px-2 py-0.5 rounded-full font-black uppercase ${h.persistente ? 'bg-brand-100 text-brand-600' : 'bg-amber-100 text-amber-600'}`}>
                           {h.persistente ? 'Activo' : 'Por Guardar'}
                         </span>
                       )}
                    </div>
                  </div>
                </div>
                
                {/* BOTONES DE ACCIÓN */}
                {!editandoHijoId && (
                  <div className="flex gap-2 w-full sm:w-auto justify-end border-t sm:border-t-0 pt-3 sm:pt-0 border-slate-200">
                    {h.persistente && (
                      h.activo ? (
                        <>
                          <button onClick={() => manejarRegenerarToken(h)} title="Regenerar enlace de bitácora" className="flex-1 sm:flex-none text-slate-400 hover:text-brand-600 hover:bg-brand-50 p-3 rounded-xl border border-slate-200 sm:border-none flex justify-center"><RefreshCw size={22}/></button>
                          <button onClick={() => manejarBajaHijo(h)} title="Baja del sistema" className="flex-1 sm:flex-none text-slate-400 hover:text-rose-500 hover:bg-rose-50 p-3 rounded-xl border border-slate-200 sm:border-none flex justify-center"><UserX size={22}/></button>
                          <button onClick={() => manejarDesvincular(h)} title="Desvincular tutor" className="flex-1 sm:flex-none text-slate-400 hover:text-amber-600 hover:bg-amber-50 p-3 rounded-xl border border-slate-200 sm:border-none flex justify-center"><Link2Off size={22}/></button>
                        </>
                      ) : (
                        <button onClick={() => manejarAltaHijo(h)} title="Reactivar Alumno" className="w-full sm:w-auto text-emerald-500 bg-emerald-50 p-3 rounded-xl flex justify-center"><RotateCcw size={22}/></button>
                      )
                    )}
                    {!h.persistente && (
                       <button onClick={() => setHijosRelacionados(prev => prev.filter(item => item.id !== h.id))} className="text-rose-400 p-3 flex justify-center"><X size={22}/></button>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>

          <button 
            onClick={guardarRelaciones} 
            disabled={loading || hijosRelacionados.every(h => h.persistente)}
            className="w-full mt-10 bg-brand-600 hover:bg-brand-700 disabled:bg-slate-100 disabled:text-slate-300 py-6 rounded-[1.5rem] font-black uppercase text-white shadow-xl flex items-center justify-center gap-4 transition-all active:scale-95"
          >
            {loading ? <Loader2 className="animate-spin" size={24}/> : <Save size={24}/>}
            {loading ? 'Guardando...' : 'Guardar Cambios'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default GestionHijos;