import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import { UtensilsCrossed, ChevronLeft, ChevronRight, Save, Loader2, Coffee, Soup, Cookie } from 'lucide-react';
import { mostrarError, mostrarExito } from './utils/alertas';
import { hoyLocal, lunesDeLaSemana, diasHabilesDeLaSemana } from './utils/fecha';

const NOMBRES_DIA = ['Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes'];
const VACIO = { desayuno: '', comida: '', merienda: '' };

// PanelMenu — "un apartado donde el administrador y staff con permisos
// puedan ir cargando el menú de la semana". Una semana = lunes a viernes,
// con guardado día por día (no hace falta llenar toda la semana de una vez).
const PanelMenu = () => {
  const [lunes, setLunes] = useState(lunesDeLaSemana(hoyLocal()));
  const [form, setForm] = useState({});
  const [loading, setLoading] = useState(true);
  const [guardandoFecha, setGuardandoFecha] = useState(null);

  const dias = diasHabilesDeLaSemana(lunes);

  // Recalcula los días DENTRO del callback (en vez de cerrar sobre la
  // constante `dias` de arriba) para que cargarSemana solo dependa de
  // `lunes` -- diasHabilesDeLaSemana(lunes) da un arreglo nuevo en cada
  // render, así que si cargarSemana cerrara sobre `dias` se recrearía en
  // cada render y el useEffect de abajo entraría en un ciclo de recargas.
  const cargarSemana = useCallback(async () => {
    setLoading(true);
    try {
      const diasSemana = diasHabilesDeLaSemana(lunes);
      const res = await api.get('/menu-semanal', { params: { inicio: diasSemana[0], fin: diasSemana[4] } });
      const porFecha = {};
      (Array.isArray(res.data) ? res.data : []).forEach((d) => {
        porFecha[d.fecha] = { desayuno: d.desayuno || '', comida: d.comida || '', merienda: d.merienda || '' };
      });
      const nuevoForm = {};
      diasSemana.forEach((f) => { nuevoForm[f] = porFecha[f] || { ...VACIO }; });
      setForm(nuevoForm);
    } catch (err) {
      console.error('Error al cargar el menú semanal:', err);
      mostrarError('No se pudo cargar el menú de la semana');
    } finally {
      setLoading(false);
    }
  }, [lunes]);

  useEffect(() => { cargarSemana(); }, [cargarSemana]);

  const cambiarSemana = (delta) => {
    const d = new Date(`${lunes}T00:00:00`);
    d.setDate(d.getDate() + delta * 7);
    setLunes(d.toLocaleDateString('en-CA'));
  };

  const actualizarCampo = (fecha, campo, valor) => {
    setForm((prev) => ({ ...prev, [fecha]: { ...prev[fecha], [campo]: valor } }));
  };

  const guardarDia = async (fecha) => {
    setGuardandoFecha(fecha);
    try {
      await api.put(`/menu-semanal/${fecha}`, form[fecha]);
      mostrarExito('Menú del día guardado');
    } catch (err) {
      console.error('Error al guardar el menú:', err);
      mostrarError('No se pudo guardar el menú de ese día');
    } finally {
      setGuardandoFecha(null);
    }
  };

  const formatoLargo = (fechaISO) => {
    const d = new Date(`${fechaISO}T00:00:00`);
    return d.toLocaleDateString('es-MX', { day: 'numeric', month: 'short', timeZone: 'UTC' });
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><UtensilsCrossed size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Menú Semanal</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Desayuno, comida y merienda</p>
            </div>
          </div>
          <div className="flex items-center gap-2 bg-slate-50 border border-slate-200 rounded-2xl px-2 py-2">
            <button onClick={() => cambiarSemana(-1)} className="p-2 rounded-xl hover:bg-white text-slate-500 transition-all"><ChevronLeft size={18} /></button>
            <span className="text-xs font-black uppercase text-slate-700 px-2">
              Semana del {formatoLargo(dias[0])} al {formatoLargo(dias[4])}
            </span>
            <button onClick={() => cambiarSemana(1)} className="p-2 rounded-xl hover:bg-white text-slate-500 transition-all"><ChevronRight size={18} /></button>
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
            {dias.map((fecha, i) => (
              <div key={fecha} className="bg-slate-50 border border-slate-100 rounded-[1.75rem] p-5 flex flex-col gap-3">
                <div>
                  <p className="font-black uppercase text-slate-900">{NOMBRES_DIA[i]}</p>
                  <p className="text-[10px] text-brand-500 font-bold uppercase">{formatoLargo(fecha)}</p>
                </div>

                <div className="space-y-2 flex-1">
                  <label className="text-[9px] font-black text-slate-400 uppercase flex items-center gap-1.5"><Coffee size={11} /> Desayuno</label>
                  <textarea rows={2} value={form[fecha]?.desayuno || ''} onChange={(e) => actualizarCampo(fecha, 'desayuno', e.target.value)} className="w-full bg-white border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium resize-none" />

                  <label className="text-[9px] font-black text-slate-400 uppercase flex items-center gap-1.5"><Soup size={11} /> Comida</label>
                  <textarea rows={2} value={form[fecha]?.comida || ''} onChange={(e) => actualizarCampo(fecha, 'comida', e.target.value)} className="w-full bg-white border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium resize-none" />

                  <label className="text-[9px] font-black text-slate-400 uppercase flex items-center gap-1.5"><Cookie size={11} /> Merienda</label>
                  <textarea rows={2} value={form[fecha]?.merienda || ''} onChange={(e) => actualizarCampo(fecha, 'merienda', e.target.value)} className="w-full bg-white border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium resize-none" />
                </div>

                <button
                  onClick={() => guardarDia(fecha)}
                  disabled={guardandoFecha === fecha}
                  className="flex items-center justify-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white text-[10px] font-black uppercase px-3 py-2.5 rounded-xl shadow-md transition-all active:scale-95"
                >
                  {guardandoFecha === fecha ? <Loader2 className="animate-spin" size={14} /> : <Save size={14} />} Guardar
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelMenu;
