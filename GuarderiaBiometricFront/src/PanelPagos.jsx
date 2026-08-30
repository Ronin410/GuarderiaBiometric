import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import {
  Wallet, Calendar, Loader2, ArrowLeft, Plus, Trash2,
  CheckCircle2, Clock, XCircle, Receipt, BellRing, AlertTriangle
} from 'lucide-react';
import { hoyLocal } from './utils/fecha';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import ReciboPago from './components/ReciboPago';

const CONCEPTOS = ['Colegiatura', 'Inscripción', 'Material', 'Otro'];
const METODOS = ['efectivo', 'transferencia', 'tarjeta', 'otro'];

const ESTADO_INFO = {
  pagado: { label: 'Pagado', color: 'bg-emerald-100 text-emerald-700 border-emerald-200', icon: CheckCircle2 },
  parcial: { label: 'Parcial', color: 'bg-amber-100 text-amber-700 border-amber-200', icon: Clock },
  pendiente: { label: 'Pendiente', color: 'bg-slate-100 text-slate-500 border-slate-200', icon: Clock },
  vencido: { label: 'Vencido', color: 'bg-rose-100 text-rose-700 border-rose-200', icon: XCircle },
};

// Filtro de la grilla — "todos" no es un estado real, es el valor por
// defecto que no filtra nada. El resto coincide exactamente con lo que
// regresa calcularEstadoPago en el backend.
const FILTROS = [
  { valor: 'todos', label: 'Todos' },
  { valor: 'pendiente', label: 'Pendientes' },
  { valor: 'parcial', label: 'Parciales' },
  { valor: 'pagado', label: 'Pagados' },
  { valor: 'vencido', label: 'Vencidos' },
];

const PanelPagos = () => {
  const [periodo, setPeriodo] = useState(hoyLocal().slice(0, 7));
  const [estados, setEstados] = useState([]);
  const [loading, setLoading] = useState(true);
  const [ninoSeleccionado, setNinoSeleccionado] = useState(null);
  const [filtroEstado, setFiltroEstado] = useState('todos');
  const [enviandoRecordatorios, setEnviandoRecordatorios] = useState(false);

  const cargarEstados = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get('/pagos/estado', { params: { periodo } });
      setEstados(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar estado de pagos:', err);
    } finally {
      setLoading(false);
    }
  }, [periodo]);

  useEffect(() => { cargarEstados(); }, [cargarEstados]);

  // Incluye tanto al que debe este periodo como al que ya está al
  // corriente en él pero arrastra deuda vieja de meses anteriores -- mismo
  // criterio que usa el backend en /pagos/recordatorio, para que este
  // contador no le prometa al admin un número distinto del que en verdad
  // se va a notificar.
  const pendientesOVencidos = estados.filter((e) => e.estado === 'pendiente' || e.estado === 'vencido' || e.deuda_acumulada > 0).length;

  const enviarRecordatorios = async () => {
    if (pendientesOVencidos === 0) return;
    const ok = await confirmar(
      `Se notificará por push a los tutores de ${pendientesOVencidos} niño(s) con la colegiatura de ${periodo} pendiente/vencida, o con deuda de meses anteriores.`,
      '¿Enviar recordatorios de pago?',
    );
    if (!ok) return;
    setEnviandoRecordatorios(true);
    try {
      const res = await api.post('/pagos/recordatorio', null, { params: { periodo } });
      mostrarExito(`Se enviaron recordatorios a los tutores de ${res.data.enviados} niño(s).`);
    } catch (err) {
      console.error('Error al enviar recordatorios:', err);
      mostrarError(err.response?.data?.error || 'No se pudieron enviar los recordatorios');
    } finally {
      setEnviandoRecordatorios(false);
    }
  };

  const estadosFiltrados = filtroEstado === 'todos'
    ? estados
    : estados.filter((e) => e.estado === filtroEstado);

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
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><Wallet size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Control de Pagos</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Colegiaturas mensuales</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="relative flex items-center bg-slate-50 rounded-2xl border border-slate-200 px-4">
              <Calendar size={18} className="text-slate-400" />
              <input type="month" value={periodo} onChange={(e) => setPeriodo(e.target.value)} className="bg-transparent p-3 text-slate-900 outline-none font-bold text-sm" />
            </div>
            <button
              onClick={enviarRecordatorios}
              disabled={enviandoRecordatorios || pendientesOVencidos === 0}
              title={pendientesOVencidos === 0 ? 'Nadie pendiente en este periodo' : `Notificar a ${pendientesOVencidos} familia(s)`}
              className="flex items-center gap-2 bg-amber-500 hover:bg-amber-600 disabled:opacity-40 text-white text-[10px] font-black uppercase px-4 py-3 rounded-2xl shadow-md transition-all active:scale-95"
            >
              {enviandoRecordatorios ? <Loader2 className="animate-spin" size={16} /> : <BellRing size={16} />}
              Recordatorios {pendientesOVencidos > 0 ? `(${pendientesOVencidos})` : ''}
            </button>
          </div>
        </div>

        {/* Filtro por estado — los contadores se calculan sobre "estados" (sin
            filtrar) para que no cambien de valor según cuál pill esté activa. */}
        <div className="flex flex-wrap gap-2 mb-6">
          {FILTROS.map((f) => {
            const cantidad = f.valor === 'todos' ? estados.length : estados.filter((e) => e.estado === f.valor).length;
            const activo = filtroEstado === f.valor;
            return (
              <button
                key={f.valor}
                onClick={() => setFiltroEstado(f.valor)}
                className={`flex items-center gap-2 text-[10px] font-black uppercase px-3.5 py-2 rounded-full border transition-all ${activo ? 'bg-brand-600 border-brand-600 text-white shadow-sm' : 'bg-white border-slate-200 text-slate-500 hover:border-brand-300'}`}
              >
                {f.label}
                <span className={`px-1.5 py-0.5 rounded-full text-[9px] ${activo ? 'bg-white/20' : 'bg-slate-100'}`}>{cantidad}</span>
              </button>
            );
          })}
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : estados.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin alumnos activos</div>
        ) : estadosFiltrados.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Nadie en este estado</div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {estadosFiltrados.map((e) => {
              const info = ESTADO_INFO[e.estado] || ESTADO_INFO.pendiente;
              const Icono = info.icon;
              const saldo = Math.max(0, (e.colegiatura_mensual || 0) - (e.total_pagado || 0));
              return (
                <button
                  key={e.hijo_id}
                  onClick={() => setNinoSeleccionado(e)}
                  className="text-left bg-slate-50 border border-slate-100 hover:border-brand-300 hover:shadow-md p-5 rounded-[1.75rem] transition-all active:scale-[0.98]"
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
                  {/* Deuda de meses ANTERIORES al periodo mostrado arriba —
                      independiente del estado de este mes: un niño puede
                      estar "Pagado" en el periodo actual y aun así arrastrar
                      deuda vieja de un mes con pago parcial que nunca se
                      terminó de cubrir (ver el comentario de deuda_acumulada
                      en el backend). Se muestra siempre que exista, sin
                      importar el filtro de estado activo. */}
                  {e.deuda_acumulada > 0 && (
                    <div className="flex items-center gap-2 mt-3 bg-rose-50 border border-rose-200 rounded-xl px-3 py-2">
                      <AlertTriangle size={14} className="text-rose-500 shrink-0" />
                      <span className="text-[10px] font-black uppercase text-rose-600 leading-tight">
                        Debe ${Number(e.deuda_acumulada).toLocaleString('es-MX', { minimumFractionDigits: 2 })} de meses anteriores
                      </span>
                    </div>
                  )}
                  {/* Desglose de los demás conceptos del periodo — aparte de la
                      colegiatura de arriba, que es la única con un monto
                      esperado configurado (colegiatura_mensual). */}
                  <div className="flex flex-wrap gap-1.5 mt-3 pt-3 border-t border-slate-200">
                    {[
                      ['Material', e.total_material],
                      ['Inscripción', e.total_inscripcion],
                      ['Otro', e.total_otro],
                    ].map(([label, monto]) => (
                      <span
                        key={label}
                        className={`text-[9px] font-black px-2 py-1 rounded-lg border uppercase ${monto > 0 ? 'bg-emerald-50 text-emerald-600 border-emerald-100' : 'bg-slate-50 text-slate-400 border-slate-100'}`}
                      >
                        {label} {monto > 0 ? `$${Number(monto).toLocaleString('es-MX', { minimumFractionDigits: 2 })}` : 'sin pago'}
                      </span>
                    ))}
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
  const [reciboId, setReciboId] = useState(null);

  const saldoSugerido = Math.max(0, (nino.colegiatura_mensual || 0) - (nino.total_pagado || 0)) || nino.colegiatura_mensual || 0;

  const [form, setForm] = useState({
    monto: saldoSugerido,
    concepto: 'Colegiatura',
    metodo_pago: 'efectivo',
    fecha_pago: hoyLocal(),
    observaciones: '',
  });

  const cargarHistorial = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get('/pagos', { params: { hijo_id: nino.hijo_id } });
      setHistorial(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar historial:', err);
    } finally {
      setLoading(false);
    }
  }, [nino.hijo_id]);

  useEffect(() => { cargarHistorial(); }, [cargarHistorial]);

  const registrarPago = async () => {
    if (!form.monto || Number(form.monto) <= 0) {
      mostrarError('El monto debe ser mayor a 0');
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
      mostrarError('No se pudo registrar el pago');
    } finally {
      setGuardando(false);
    }
  };

  const eliminarPago = async (id) => {
    const ok = await confirmar('¿Eliminar este pago? Esta acción no se puede deshacer.', 'Eliminar pago');
    if (!ok) return;
    try {
      await api.delete(`/pagos/${id}`);
      cargarHistorial();
    } catch (err) {
      console.error('Error al eliminar pago:', err);
      mostrarError('No se pudo eliminar el pago');
    }
  };

  if (reciboId) {
    return <ReciboPago pagoId={reciboId} rutaBase="/pagos" onVolver={() => setReciboId(null)} />;
  }

  return (
    <div className="animate-in fade-in duration-500">
      <button onClick={onVolver} className="mb-6 flex items-center gap-2 text-brand-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
        <ArrowLeft size={16} /> Volver al listado
      </button>

      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl space-y-8">
        <div>
          <h3 className="text-xl font-black uppercase text-slate-900">{nino.nombre}</h3>
          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Periodo {periodo}</p>
        </div>

        {nino.deuda_acumulada > 0 && (
          <div className="flex items-center gap-3 bg-rose-50 border border-rose-200 rounded-2xl px-5 py-4">
            <AlertTriangle size={20} className="text-rose-500 shrink-0" />
            <p className="text-xs font-bold text-rose-600">
              Además de {periodo}, este niño debe <span className="font-black">${Number(nino.deuda_acumulada).toLocaleString('es-MX', { minimumFractionDigits: 2 })}</span> de colegiatura de meses anteriores. Revisa el historial completo abajo para ver de cuáles.
            </p>
          </div>
        )}

        {/* FORMULARIO DE REGISTRO */}
        <div className="bg-slate-50 p-6 rounded-[2rem] border border-slate-100">
          <h4 className="text-[10px] font-black text-slate-400 uppercase mb-5 tracking-widest">Registrar nuevo pago</h4>
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Monto (MXN)</label>
              <input type="number" min="0" step="0.01" value={form.monto} onChange={(e) => setForm({ ...form, monto: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Fecha de pago</label>
              <input type="date" value={form.fecha_pago} onChange={(e) => setForm({ ...form, fecha_pago: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Concepto</label>
              <select value={form.concepto} onChange={(e) => setForm({ ...form, concepto: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold">
                {CONCEPTOS.map(c => <option key={c} value={c}>{c}</option>)}
              </select>
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Método de pago</label>
              <select value={form.metodo_pago} onChange={(e) => setForm({ ...form, metodo_pago: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold capitalize">
                {METODOS.map(m => <option key={m} value={m}>{m}</option>)}
              </select>
            </div>
            <div className="sm:col-span-2">
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Observaciones (opcional)</label>
              <input type="text" value={form.observaciones} onChange={(e) => setForm({ ...form, observaciones: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium" />
            </div>
          </div>
          <button onClick={registrarPago} disabled={guardando} className="w-full mt-5 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs py-4 rounded-xl shadow-md flex items-center justify-center gap-2 transition-all active:scale-95">
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
                  <div className="flex items-center gap-1">
                    <button onClick={() => setReciboId(p.id)} title="Ver recibo" className="text-slate-300 hover:text-brand-600 hover:bg-brand-50 p-2.5 rounded-xl transition-colors"><Receipt size={18} /></button>
                    <button onClick={() => eliminarPago(p.id)} title="Eliminar pago" className="text-slate-300 hover:text-rose-500 hover:bg-rose-50 p-2.5 rounded-xl transition-colors"><Trash2 size={18} /></button>
                  </div>
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
