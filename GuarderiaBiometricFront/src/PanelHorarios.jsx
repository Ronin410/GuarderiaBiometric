import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import {
  Clock, ArrowLeft, Save, Loader2, Plus, Trash2, CalendarClock, UserCog,
} from 'lucide-react';
import { hoyLocal } from './utils/fecha';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';

const NOMBRES_DIA = ['Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes', 'Sábado', 'Domingo'];

// PanelHorarios -- "Horarios/turnos y horas trabajadas del personal".
// Exclusivo de admin (mismo nivel que Gestión de Personal): es información
// ligada a nómina. Dos secciones separadas por persona: el TURNO fijo
// semanal (el plan) y el REGISTRO de horas reales trabajadas día a día
// (para nómina, puede diferir del turno por faltas, horas extra, etc.).
const PanelHorarios = () => {
  const [personal, setPersonal] = useState([]);
  const [loading, setLoading] = useState(true);
  const [seleccionado, setSeleccionado] = useState(null);

  const cargarPersonal = async () => {
    setLoading(true);
    try {
      const res = await api.get('/admin/personal');
      setPersonal(Array.isArray(res.data) ? res.data.filter((p) => p.activo) : []);
    } catch (err) {
      console.error('Error al cargar personal:', err);
      mostrarError('No se pudo cargar la lista de personal');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargarPersonal(); }, []);

  if (seleccionado) {
    return <DetalleHorario persona={seleccionado} onVolver={() => setSeleccionado(null)} />;
  }

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex items-center gap-4 mb-8">
          <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><Clock size={28} /></div>
          <div>
            <h3 className="text-xl font-black uppercase text-slate-900">Horarios de Personal</h3>
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Turnos y horas trabajadas</p>
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : personal.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin cuentas de personal activas</div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {personal.map((p) => (
              <button
                key={p.id}
                onClick={() => setSeleccionado(p)}
                className="text-left bg-slate-50 border border-slate-100 hover:border-brand-300 hover:shadow-md p-5 rounded-[1.75rem] transition-all active:scale-[0.98] flex items-center gap-4"
              >
                <div className="bg-white p-3 rounded-2xl text-slate-300 border border-slate-100"><UserCog size={22} /></div>
                <div className="min-w-0">
                  <p className="font-black uppercase text-sm text-slate-900 truncate">{p.nombre || p.username}</p>
                  <p className="text-[9px] text-brand-500 font-bold uppercase mt-0.5">{p.rol}</p>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

const DetalleHorario = ({ persona, onVolver }) => {
  const [turnos, setTurnos] = useState([]);
  const [loadingTurnos, setLoadingTurnos] = useState(true);
  const [guardandoDia, setGuardandoDia] = useState(null);

  const [mes, setMes] = useState(hoyLocal().slice(0, 7));
  const [registros, setRegistros] = useState([]);
  const [loadingHoras, setLoadingHoras] = useState(true);
  const [nuevoRegistro, setNuevoRegistro] = useState({ fecha: hoyLocal(), horas_trabajadas: '', observaciones: '' });
  const [guardandoRegistro, setGuardandoRegistro] = useState(false);

  const cargarTurnos = useCallback(async () => {
    setLoadingTurnos(true);
    try {
      const res = await api.get(`/admin/horarios/${persona.id}`);
      // El backend regresa las horas como "HH:MM:SS" (columna TIME de
      // Postgres); <input type="time"> por defecto solo entiende "HH:MM"
      // (granularidad de minuto) y descarta en silencio cualquier valor con
      // segundos -- se recorta aquí para que el campo se vea poblado.
      const datos = (Array.isArray(res.data) ? res.data : []).map((t) => ({
        ...t,
        hora_entrada: t.hora_entrada ? t.hora_entrada.slice(0, 5) : t.hora_entrada,
        hora_salida: t.hora_salida ? t.hora_salida.slice(0, 5) : t.hora_salida,
      }));
      setTurnos(datos);
    } catch (err) {
      console.error('Error al cargar el turno:', err);
      mostrarError('No se pudo cargar el turno');
    } finally {
      setLoadingTurnos(false);
    }
  }, [persona.id]);

  const cargarHoras = useCallback(async () => {
    setLoadingHoras(true);
    try {
      const [anio, mesNum] = mes.split('-');
      const inicio = `${mes}-01`;
      const fin = new Date(Number(anio), Number(mesNum), 0).toISOString().slice(0, 10);
      const res = await api.get(`/admin/horas/${persona.id}`, { params: { inicio, fin } });
      setRegistros(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar el registro de horas:', err);
      mostrarError('No se pudo cargar el registro de horas');
    } finally {
      setLoadingHoras(false);
    }
  }, [mes, persona.id]);

  useEffect(() => { cargarTurnos(); }, [cargarTurnos]);
  useEffect(() => { cargarHoras(); }, [cargarHoras]);

  const actualizarTurno = (dia, campo, valor) => {
    setTurnos((prev) => prev.map((t) => (t.dia_semana === dia ? { ...t, [campo]: valor || null } : t)));
  };

  const guardarTurno = async (dia) => {
    const t = turnos.find((x) => x.dia_semana === dia);
    setGuardandoDia(dia);
    try {
      await api.put(`/admin/horarios/${persona.id}/${dia}`, {
        hora_entrada: t.hora_entrada || null,
        hora_salida: t.hora_salida || null,
      });
      mostrarExito('Turno guardado');
    } catch (err) {
      console.error('Error al guardar el turno:', err);
      mostrarError('No se pudo guardar el turno');
    } finally {
      setGuardandoDia(null);
    }
  };

  const agregarRegistro = async () => {
    const horas = parseFloat(nuevoRegistro.horas_trabajadas);
    if (!horas || horas <= 0 || horas > 24) {
      mostrarError('Las horas trabajadas deben ser mayores a 0 y hasta 24');
      return;
    }
    setGuardandoRegistro(true);
    try {
      await api.put(`/admin/horas/${persona.id}/${nuevoRegistro.fecha}`, {
        horas_trabajadas: horas,
        observaciones: nuevoRegistro.observaciones,
      });
      setNuevoRegistro({ fecha: hoyLocal(), horas_trabajadas: '', observaciones: '' });
      cargarHoras();
    } catch (err) {
      console.error('Error al guardar el registro:', err);
      mostrarError('No se pudo guardar el registro');
    } finally {
      setGuardandoRegistro(false);
    }
  };

  const eliminarRegistro = async (fecha) => {
    const ok = await confirmar(`Se eliminará el registro del ${fecha}.`, '¿Eliminar registro?');
    if (!ok) return;
    try {
      await api.delete(`/admin/horas/${persona.id}/${fecha}`);
      cargarHoras();
    } catch (err) {
      console.error('Error al eliminar el registro:', err);
      mostrarError('No se pudo eliminar el registro');
    }
  };

  const totalHorasMes = registros.reduce((acc, r) => acc + Number(r.horas_trabajadas || 0), 0);

  return (
    <div className="animate-in fade-in duration-500">
      <button onClick={onVolver} className="mb-6 flex items-center gap-2 text-brand-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
        <ArrowLeft size={16} /> Volver al listado
      </button>

      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl space-y-10">
        <div>
          <h3 className="text-xl font-black uppercase text-slate-900">{persona.nombre || persona.username}</h3>
          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">@{persona.username} · {persona.rol}</p>
        </div>

        {/* TURNO SEMANAL */}
        <div>
          <h4 className="text-[10px] font-black text-slate-400 uppercase mb-4 tracking-widest flex items-center gap-2"><Clock size={14} /> Turno semanal</h4>
          {loadingTurnos ? (
            <div className="py-8 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
              {turnos.map((t) => (
                <div key={t.dia_semana} className="bg-slate-50 border border-slate-100 rounded-2xl p-4 space-y-2">
                  <p className="text-xs font-black uppercase text-slate-700">{NOMBRES_DIA[t.dia_semana]}</p>
                  <div className="flex items-center gap-2">
                    <input type="time" value={t.hora_entrada || ''} onChange={(e) => actualizarTurno(t.dia_semana, 'hora_entrada', e.target.value)} className="flex-1 min-w-0 bg-white border border-slate-200 p-2 rounded-lg outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
                    <span className="text-slate-300 text-xs shrink-0">–</span>
                    <input type="time" value={t.hora_salida || ''} onChange={(e) => actualizarTurno(t.dia_semana, 'hora_salida', e.target.value)} className="flex-1 min-w-0 bg-white border border-slate-200 p-2 rounded-lg outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
                  </div>
                  <button
                    onClick={() => guardarTurno(t.dia_semana)}
                    disabled={guardandoDia === t.dia_semana}
                    className="w-full flex items-center justify-center gap-1.5 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white text-[9px] font-black uppercase px-2 py-2 rounded-lg transition-all"
                  >
                    {guardandoDia === t.dia_semana ? <Loader2 className="animate-spin" size={12} /> : <Save size={12} />} Guardar
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* REGISTRO DE HORAS */}
        <div>
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-4">
            <h4 className="text-[10px] font-black text-slate-400 uppercase tracking-widest flex items-center gap-2"><CalendarClock size={14} /> Registro de horas trabajadas</h4>
            <div className="flex items-center gap-3">
              <input type="month" value={mes} onChange={(e) => setMes(e.target.value)} className="bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
              <span className="text-[10px] font-black text-slate-400 uppercase">Total: <span className="text-slate-700">{totalHorasMes.toFixed(1)} h</span></span>
            </div>
          </div>

          <div className="bg-slate-50 border border-dashed border-slate-200 rounded-2xl p-4 flex flex-wrap items-end gap-3 mb-4">
            <div>
              <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Fecha</label>
              <input type="date" value={nuevoRegistro.fecha} onChange={(e) => setNuevoRegistro({ ...nuevoRegistro, fecha: e.target.value })} className="bg-white border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
            </div>
            <div>
              <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Horas</label>
              <input type="number" min="0.5" max="24" step="0.5" value={nuevoRegistro.horas_trabajadas} onChange={(e) => setNuevoRegistro({ ...nuevoRegistro, horas_trabajadas: e.target.value })} className="w-24 bg-white border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
            </div>
            <div className="flex-1 min-w-[160px]">
              <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Observaciones (opcional)</label>
              <input type="text" value={nuevoRegistro.observaciones} onChange={(e) => setNuevoRegistro({ ...nuevoRegistro, observaciones: e.target.value })} className="w-full bg-white border border-slate-200 p-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium" />
            </div>
            <button onClick={agregarRegistro} disabled={guardandoRegistro} className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all">
              {guardandoRegistro ? <Loader2 className="animate-spin" size={14} /> : <Plus size={14} />} Guardar día
            </button>
          </div>

          {loadingHoras ? (
            <div className="py-8 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
          ) : registros.length === 0 ? (
            <div className="py-8 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin registros este mes</div>
          ) : (
            <div className="space-y-2">
              {registros.map((r) => (
                <div key={r.id} className="flex items-center justify-between bg-slate-50 border border-slate-100 p-3.5 rounded-xl">
                  <div>
                    <p className="font-black text-sm text-slate-800">{r.fecha} <span className="text-brand-600">· {Number(r.horas_trabajadas).toFixed(1)} h</span></p>
                    {r.observaciones && <p className="text-[10px] text-slate-400 font-bold mt-0.5">{r.observaciones}</p>}
                  </div>
                  <button onClick={() => eliminarRegistro(r.fecha)} title="Eliminar registro" className="text-slate-300 hover:text-rose-500 hover:bg-rose-50 p-2 rounded-xl transition-colors"><Trash2 size={16} /></button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default PanelHorarios;
