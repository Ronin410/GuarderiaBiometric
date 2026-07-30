import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import {
  Wallet, Calendar, Loader2, ArrowLeft, Plus, Trash2,
  CheckCircle2, Clock, XCircle
} from 'lucide-react';
import { hoyLocal } from './utils/fecha';

const CONCEPTOS = ['Colegiatura', 'Inscripción', 'Material', 'Otro'];
const METODOS = ['efectivo', 'transferencia', 'tarjeta', 'otro'];

const ESTADO_INFO = {
  pagado: { label: 'Pagado', color: 'bg-emerald-100 text-emerald-700 border-emerald-200', icon: CheckCircle2 },
  parcial: { label: 'Parcial', color: 'bg-amber-100 text-amber-700 border-amber-200', icon: Clock },
  pendiente: { label: 'Pendiente', color: 'bg-slate-100 text-slate-500 border-slate-200', icon: Clock },
  vencido: { label: 'Vencido', color: 'bg-rose-100 text-rose-700 border-rose-200', icon: XCircle },
};

const PanelPagos = () => {
  const [periodo, setPeriodo] = useState(hoyLocal().slice(0, 7));
  const [estados, setEstados] = useState([]);
  const [loading, setLoading] = useState(true);
  const [ninoSeleccionado, setNinoSeleccionado] = useState(null);

  const cargarEstados = async () => {
    setLoading(true);
    try {
      const res = await api.get('/pagos/estado', { params: { periodo } });
      setEstados(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar estado de pagos:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargarEstados(); }, [periodo]);

  if (ninoSeleccionado) {
    return (
      <DetallePago
        nino={ninoSeleccionado}
        periodo={periodo}
        onVolver={() => { setNinoSeleccionado(null); cargarEstados(); }}
      />
    );
  }

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-violet-100 p-3 rounded-2xl text-violet-600"><Wallet size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Control de Pagos</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Colegiaturas mensuales</p>
            </div>
          </div>
          <div className="relative flex items-center bg-slate-50 rounded-2xl border border-slate-200 px-4">
            <Calendar size={18} className="text-slate-400" />
            <input type="month" value={periodo} onChange={(e) => setPeriodo(e.target.value)} className="bg-transparent p-3 text-slate-900 outline-none font-bold text-sm" />
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : estados.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin alumnos activos</div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {estados.map((e) => {
              const info = ESTADO_INFO[e.estado] || ESTADO_INFO.pendiente;
              const Icono = info.icon;
              const saldo = Math.max(0, (e.colegiatura_mensual || 0) - (e.total_pagado || 0));
              return (
                <button
                  key={e.hijo_id}
                  onClick={() => setNinoSeleccionado(e)}
                  className="text-left bg-slate-50 border border-slate-100 hover:border-violet-300 hover:shadow-md p-5 rounded-[1.75rem] transition-all active:scale-[0.98]"
                >
                  <div className="flex justify-between items-start gap-2 mb-4">
                    <p className="font-black uppercase text-sm text-slate-900 leading-tight">{e.nombre}</p>
                    <span className={`shrink-0 flex items-center gap-1 text-[9px] font-black px-2 py-1 rounded-lg border uppercase ${info.color}`}>
                      <Icono size={11} /> {info.label}
                    </span>
                  </div>
                  <div className="space-y-1 text-[11px] font-bold">
                    <div className="flex justify-between text-slate-400"><span>Colegiatura</span><span className="text-slate-700">${Number(e.colegiatura_mensual || 0).toLocaleString('es-MX', { minimumFractionDigits: 2 })}</span></div>
                    <div className="flex justify-between text-slate-400"><span>Pagado</span><span className="text-emerald-600">${Number(e.total_pagado || 0).toLocaleString('es-MX', { minimumFractionDigits: 2 })}</span></div>
                    {saldo > 0 && <div className="flex justify-between text-slate-400"><span>Saldo</span><span className="text-rose-500">${saldo.toLocaleString('es-MX', { minimumFractionDigits: 2 })}</span></div>}
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

const DetallePago = ({ nino, periodo, onVolver }) => {
  const [historial, setHistorial] = useState([]);
  const [loading, setLoading] = useState(true);
  const [guardando, setGuardando] = useState(false);

  const saldoSugerido = Math.max(0, (nino.colegiatura_mensual || 0) - (nino.total_pagado || 0)) || nino.colegiatura_mensual || 0;

  const [form, setForm] = useState({
    monto: saldoSugerido,
    concepto: 'Colegiatura',
    metodo_pago: 'efectivo',
    fecha_pago: hoyLocal(),
    observaciones: '',
  });

  const cargarHistorial = async () => {
    setLoading(true);
    try {
      const res = await api.get('/pagos', { params: { hijo_id: nino.hijo_id } });
      setHistorial(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar historial:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargarHistorial(); }, []);

  const registrarPago = async () => {
    if (!form.monto || Number(form.monto) <= 0) {
      alert('El monto debe ser mayor a 0');
      return;
    }
    setGuardando(true);
    try {
      await api.post('/pagos', {
        hijo_id: nino.hijo_id,
        periodo,
        monto: Number(form.monto),
        concepto: form.concepto,
        metodo_pago: form.metodo_pago,
        fecha_pago: form.fecha_pago,
        observaciones: form.observaciones,
      });
      setForm({ ...form, monto: 0, observaciones: '' });
      cargarHistorial();
    } catch (err) {
      console.error('Error al registrar pago:', err);
      alert('❌ No se pudo registrar el pago');
    } finally {
      setGuardando(false);
    }
  };

  const eliminarPago = async (id) => {
    if (!window.confirm('¿Eliminar este pago? Esta acción no se puede deshacer.')) return;
    try {
      await api.delete(`/pagos/${id}`);
      cargarHistorial();
    } catch (err) {
      console.error('Error al eliminar pago:', err);
      alert('❌ No se pudo eliminar el pago');
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <button onClick={onVolver} className="mb-6 flex items-center gap-2 text-violet-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
        <ArrowLeft size={16} /> Volver al listado
      </button>

      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl space-y-8">
        <div>
          <h3 className="text-xl font-black uppercase text-slate-900">{nino.nombre}</h3>
          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Periodo {periodo}</p>
        </div>

        {/* FORMULARIO DE REGISTRO */}
        <div className="bg-slate-50 p-6 rounded-[2rem] border border-slate-100">
          <h4 className="text-[10px] font-black text-slate-400 uppercase mb-5 tracking-widest">Registrar nuevo pago</h4>
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Monto (MXN)</label>
              <input type="number" min="0" step="0.01" value={form.monto} onChange={(e) => setForm({ ...form, monto: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-bold" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Fecha de pago</label>
              <input type="date" value={form.fecha_pago} onChange={(e) => setForm({ ...form, fecha_pago: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-bold" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Concepto</label>
              <select value={form.concepto} onChange={(e) => setForm({ ...form, concepto: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-bold">
                {CONCEPTOS.map(c => <option key={c} value={c}>{c}</option>)}
              </select>
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Método de pago</label>
              <select value={form.metodo_pago} onChange={(e) => setForm({ ...form, metodo_pago: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-bold capitalize">
                {METODOS.map(m => <option key={m} value={m}>{m}</option>)}
              </select>
            </div>
            <div className="sm:col-span-2">
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Observaciones (opcional)</label>
              <input type="text" value={form.observaciones} onChange={(e) => setForm({ ...form, observaciones: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-medium" />
            </div>
          </div>
          <button onClick={registrarPago} disabled={guardando} className="w-full mt-5 bg-violet-600 hover:bg-violet-700 disabled:opacity-50 text-white font-black uppercase text-xs py-4 rounded-xl shadow-md flex items-center justify-center gap-2 transition-all active:scale-95">
            {guardando ? <Loader2 className="animate-spin" size={18} /> : <Plus size={18} />} Registrar Pago
          </button>
        </div>

        {/* HISTORIAL */}
        <div>
          <h4 className="text-[10px] font-black text-slate-400 uppercase mb-4 tracking-widest">Historial completo</h4>
          {loading ? (
            <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
          ) : historial.length === 0 ? (
            <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin pagos registrados</div>
          ) : (
            <div className="space-y-3">
              {historial.map((p) => (
                <div key={p.id} className="flex items-center justify-between bg-slate-50 border border-slate-100 p-4 rounded-2xl">
                  <div>
                    <p className="font-black text-sm text-slate-800">${Number(p.monto).toLocaleString('es-MX', { minimumFractionDigits: 2 })} <span className="text-slate-400 font-bold text-xs">· {p.concepto}</span></p>
                    <p className="text-[10px] text-slate-400 font-bold uppercase mt-1">{p.periodo} · {p.fecha_pago} · {p.metodo_pago}{p.observaciones ? ` · ${p.observaciones}` : ''}</p>
                  </div>
                  <button onClick={() => eliminarPago(p.id)} title="Eliminar pago" className="text-slate-300 hover:text-rose-500 hover:bg-rose-50 p-2.5 rounded-xl transition-colors"><Trash2 size={18} /></button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default PanelPagos;
