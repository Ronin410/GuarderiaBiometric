import React, { useEffect, useState } from 'react';
import api from './axiosConfig';
import {
  Clock,
  ChevronLeft, Heart, CheckCircle2,
  Calendar as CalendarIcon,
  X, // Importamos el icono de cerrar
  ClipboardList, IdCard, Wallet,
  Cake, MapPin, Phone, XCircle, Receipt,
  CalendarOff, Plus, Loader2, Trash2, UtensilsCrossed, RotateCcw, Image as ImageIcon,
  CreditCard, FileText, ExternalLink
} from 'lucide-react';
import { hoyLocal } from './utils/fecha';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import ReporteDiario from './components/ReporteDiario';
import ReciboPago from './components/ReciboPago';
import GaleriaFotos from './components/GaleriaFotos';

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
  const [reciboId, setReciboId] = useState(null);

  // Pagos en línea (Stripe): sigue apagado hasta que la guardería configure
  // sus llaves -- pagosEnLineaHabilitado se queda en false y el botón de
  // "Pagar en línea" ni se intenta renderizar, sin más cambios visibles.
  const [pagosEnLineaHabilitado, setPagosEnLineaHabilitado] = useState(false);
  const [iniciandoPagoEnLinea, setIniciandoPagoEnLinea] = useState(false);
  // estadoPago: saldo pendiente de colegiatura del mes en curso para ESTE
  // hijo (mismo dato que ya se ve en el resumen del inicio, pero filtrado a
  // uno solo). montoAPagar es lo que el papá elige pagar con tarjeta --
  // arranca en el saldo completo pero se puede bajar ("solo tengo 1000 de
  // los 2000"), nunca subir más allá de eso (el backend también lo valida).
  const [estadoPago, setEstadoPago] = useState(null);
  const [montoAPagar, setMontoAPagar] = useState('');

  const [ausencias, setAusencias] = useState([]);
  const [loadingAusencias, setLoadingAusencias] = useState(false);
  const [formAusencia, setFormAusencia] = useState({ fecha_inicio: '', fecha_fin: '', motivo: '' });
  const [guardandoAusencia, setGuardandoAusencia] = useState(false);
  const [cancelandoId, setCancelandoId] = useState(null);

  const [documentos, setDocumentos] = useState([]);
  const [loadingDocumentos, setLoadingDocumentos] = useState(false);

  const [pedidosComedor, setPedidosComedor] = useState([]);
  const [loadingComedor, setLoadingComedor] = useState(false);
  const [formComedor, setFormComedor] = useState({ fecha: '', desayuno: true, comida: true, merienda: true, notas: '' });
  const [guardandoComedor, setGuardandoComedor] = useState(false);
  const [restableciendoFecha, setRestableciendoFecha] = useState(null);

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

  const cargarAusencias = async () => {
    setLoadingAusencias(true);
    try {
      const res = await api.get(`/padre/hijos/${hijoId}/ausencias`);
      setAusencias(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error("Error al obtener las ausencias", err);
      setAusencias([]);
    } finally {
      setLoadingAusencias(false);
    }
  };

  useEffect(() => {
    if (vista === 'ausencias') cargarAusencias();
  }, [hijoId, vista]);

  const cargarDocumentos = async () => {
    setLoadingDocumentos(true);
    try {
      const res = await api.get(`/padre/hijos/${hijoId}/documentos`);
      setDocumentos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error("Error al obtener los documentos", err);
      setDocumentos([]);
    } finally {
      setLoadingDocumentos(false);
    }
  };

  useEffect(() => {
    if (vista === 'expediente') cargarDocumentos();
  }, [hijoId, vista]);

  const reportarAusencia = async () => {
    if (!formAusencia.fecha_inicio) {
      mostrarError('Elige al menos la fecha de inicio');
      return;
    }
    setGuardandoAusencia(true);
    try {
      await api.post(`/padre/hijos/${hijoId}/ausencias`, formAusencia);
      mostrarExito('Le avisamos a la guardería que tu hijo no asistirá esos días');
      setFormAusencia({ fecha_inicio: '', fecha_fin: '', motivo: '' });
      cargarAusencias();
    } catch (err) {
      console.error("Error al reportar la ausencia", err);
      mostrarError(err.response?.data?.error || 'No se pudo reportar la ausencia');
    } finally {
      setGuardandoAusencia(false);
    }
  };

  const cancelarAusencia = async (ausencia) => {
    const ok = await confirmar(`Se cancelará el aviso de ausencia del ${ausencia.fecha}.`, '¿Cancelar ausencia?');
    if (!ok) return;
    setCancelandoId(ausencia.id);
    try {
      await api.delete(`/padre/ausencias/${ausencia.id}`);
      cargarAusencias();
    } catch (err) {
      console.error("Error al cancelar la ausencia", err);
      mostrarError('No se pudo cancelar la ausencia');
    } finally {
      setCancelandoId(null);
    }
  };

  const cargarPedidosComedor = async () => {
    setLoadingComedor(true);
    try {
      const res = await api.get(`/padre/hijos/${hijoId}/pedidos-comedor`);
      setPedidosComedor(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error("Error al obtener los pedidos de comedor", err);
      setPedidosComedor([]);
    } finally {
      setLoadingComedor(false);
    }
  };

  useEffect(() => {
    if (vista === 'comedor') cargarPedidosComedor();
  }, [hijoId, vista]);

  const guardarPedidoComedor = async () => {
    if (!formComedor.fecha) {
      mostrarError('Elige la fecha');
      return;
    }
    if (formComedor.desayuno && formComedor.comida && formComedor.merienda && !formComedor.notas.trim()) {
      mostrarError('Desmarca al menos una comida o agrega una nota -- si tu hijo come normal ese día no hace falta registrar nada');
      return;
    }
    setGuardandoComedor(true);
    try {
      await api.put(`/padre/hijos/${hijoId}/pedidos-comedor/${formComedor.fecha}`, {
        desayuno: formComedor.desayuno, comida: formComedor.comida, merienda: formComedor.merienda, notas: formComedor.notas,
      });
      mostrarExito('Le avisamos a la guardería');
      setFormComedor({ fecha: '', desayuno: true, comida: true, merienda: true, notas: '' });
      cargarPedidosComedor();
    } catch (err) {
      console.error("Error al guardar el pedido de comedor", err);
      mostrarError(err.response?.data?.error || 'No se pudo guardar el aviso');
    } finally {
      setGuardandoComedor(false);
    }
  };

  const restablecerPedidoComedor = async (pedido) => {
    const ok = await confirmar(`Se volverá al comedor normal (las tres comidas) del ${pedido.fecha}.`, '¿Restablecer?');
    if (!ok) return;
    setRestableciendoFecha(pedido.fecha);
    try {
      await api.put(`/padre/hijos/${hijoId}/pedidos-comedor/${pedido.fecha}`, { desayuno: true, comida: true, merienda: true, notas: '' });
      cargarPedidosComedor();
    } catch (err) {
      console.error("Error al restablecer el pedido de comedor", err);
      mostrarError('No se pudo restablecer');
    } finally {
      setRestableciendoFecha(null);
    }
  };

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

    api.get('/pagos-online/config')
      .then((res) => setPagosEnLineaHabilitado(!!res.data?.habilitado))
      .catch(() => setPagosEnLineaHabilitado(false));

    // /padre/mis-pagos trae el estado de TODOS los hijos del papá -- se
    // filtra al que se está viendo aquí. Mismo endpoint que ya usa el
    // resumen de DashboardPadre, así que el saldo siempre coincide.
    api.get('/padre/mis-pagos')
      .then((res) => {
        const propio = Array.isArray(res.data) ? res.data.find((e) => String(e.hijo_id) === String(hijoId)) : null;
        setEstadoPago(propio || null);
        const saldo = propio ? propio.colegiatura_mensual - propio.total_pagado : 0;
        setMontoAPagar(saldo > 0 ? String(saldo.toFixed(2)) : '');
      })
      .catch(() => setEstadoPago(null));
  }, [hijoId, vista]);

  const pagarColegiaturaEnLinea = async () => {
    const monto = Number(montoAPagar);
    if (!monto || monto <= 0) {
      mostrarError('Escribe cuánto quieres pagar');
      return;
    }
    setIniciandoPagoEnLinea(true);
    try {
      const periodoActual = hoyLocal().slice(0, 7); // YYYY-MM
      const res = await api.post('/padre/pagos-online/checkout', { hijo_id: String(hijoId), periodo: periodoActual, monto });
      window.location.href = res.data.url;
    } catch (err) {
      console.error('Error al iniciar el pago en línea', err);
      mostrarError(err.response?.data?.error || 'No se pudo iniciar el pago en línea');
      setIniciandoPagoEnLinea(false);
    }
  };

  const handleCambioFecha = (e) => {
    setFechaSeleccionada(e.target.value);
  };

  if (reciboId) {
    return (
      <div className="min-h-screen bg-slate-50 p-4 sm:p-6">
        <ReciboPago pagoId={reciboId} rutaBase="/padre/pagos" onVolver={() => setReciboId(null)} />
      </div>
    );
  }

  if (loading && vista === 'bitacora') {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center p-10 text-center">
        <div className="space-y-4">
          <div className="w-12 h-12 border-4 border-brand-600 border-t-transparent rounded-full animate-spin mx-auto"></div>
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
          className="flex items-center gap-2 text-slate-400 font-black uppercase text-[10px] tracking-widest mb-6 hover:text-brand-600 transition-colors"
        >
          <ChevronLeft size={16} /> Volver
        </button>

        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="text-3xl font-black text-slate-900 uppercase tracking-tighter">{nombreHijo}</h2>
            <div className="bg-brand-600 p-3 rounded-2xl text-white shadow-lg shadow-brand-200">
              <Heart size={20} fill="currentColor" />
            </div>
          </div>

          {/* SELECTOR DE PESTAÑAS */}
          <div className="flex bg-slate-100 p-1.5 rounded-2xl">
            {[
              { key: 'bitacora', label: 'Hoy', icon: ClipboardList },
              { key: 'expediente', label: 'Expediente', icon: IdCard },
              { key: 'pagos', label: 'Pagos', icon: Wallet },
              { key: 'ausencias', label: 'Ausencias', icon: CalendarOff },
              { key: 'comedor', label: 'Comedor', icon: UtensilsCrossed },
              { key: 'galeria', label: 'Galería', icon: ImageIcon },
            ].map((tab) => {
              const Icono = tab.icon;
              return (
                <button
                  key={tab.key}
                  onClick={() => setVista(tab.key)}
                  className={`flex-1 py-2.5 rounded-xl flex items-center justify-center gap-1.5 font-black text-[10px] uppercase transition-all ${vista === tab.key ? 'bg-white text-brand-600 shadow-sm' : 'text-slate-400'}`}
                >
                  <Icono size={13} /> {tab.label}
                </button>
              );
            })}
          </div>

          {vista === 'bitacora' && (
            <div className="relative">
              <div className="absolute inset-y-0 left-4 flex items-center pointer-events-none text-brand-600">
                <CalendarIcon size={16} />
              </div>
              <input
                type="date"
                value={fechaSeleccionada}
                onChange={handleCambioFecha}
                max={hoyLocal()}
                className="w-full bg-slate-50 border-none rounded-2xl py-3 pl-12 pr-4 text-sm font-bold text-slate-700 focus:ring-2 focus:ring-brand-500 transition-all uppercase"
              />
            </div>
          )}
        </div>
      </div>

      {vista === 'expediente' && (
        <div className="max-w-md mx-auto p-4 space-y-4">
          <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100 space-y-5">
            <div className="flex items-center gap-2">
              <div className="p-2 bg-brand-100 text-brand-600 rounded-lg"><IdCard size={18} /></div>
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

          {/* DOCUMENTOS DE INSCRIPCIÓN -- "Quiero que en la parte de
              expediente los papás puedan ver cuáles son los documentos que
              han entregado a la guardería y cuáles son los que les falta".
              Solo lectura: quien sube/reemplaza documentos sigue siendo el
              staff (DocumentosNino.jsx), esto es nada más para que el papá
              sepa qué falta sin tener que preguntar. */}
          <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100 space-y-4">
            <div className="flex items-center gap-2">
              <div className="p-2 bg-brand-100 text-brand-600 rounded-lg"><FileText size={18} /></div>
              <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Documentos entregados</h3>
            </div>
            {loadingDocumentos ? (
              <div className="py-6 text-center text-slate-400 font-black uppercase tracking-widest text-[10px]">Cargando...</div>
            ) : documentos.length === 0 ? (
              <div className="py-6 text-center text-slate-400 font-black uppercase tracking-widest text-[10px]">
                La guardería todavía no configuró qué documentos pedir
              </div>
            ) : (
              <div className="space-y-2">
                {documentos.map((d) => {
                  const entregado = !!d.nombre_archivo;
                  return (
                    <div key={d.tipo} className={`flex items-center justify-between gap-3 px-4 py-3 rounded-2xl border ${entregado ? 'bg-emerald-50/50 border-emerald-100' : 'bg-amber-50/50 border-amber-100'}`}>
                      <div className="flex items-center gap-3 min-w-0">
                        {entregado ? <CheckCircle2 size={18} className="text-emerald-500 shrink-0" /> : <XCircle size={18} className="text-amber-500 shrink-0" />}
                        <div className="min-w-0">
                          <p className="text-xs font-black uppercase text-slate-700 truncate">{d.nombre}</p>
                          <p className={`text-[9px] font-black uppercase ${entregado ? 'text-emerald-600' : 'text-amber-600'}`}>
                            {entregado ? 'Entregado' : 'Falta entregar'}
                          </p>
                        </div>
                      </div>
                      {entregado && d.url && (
                        <a href={d.url} target="_blank" rel="noreferrer" className="flex items-center gap-1 text-[10px] font-black uppercase text-brand-600 shrink-0 px-2 py-1.5">
                          <ExternalLink size={12} /> Ver
                        </a>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      )}

      {vista === 'pagos' && (
        <div className="max-w-md mx-auto p-4 space-y-4">
          {pagosEnLineaHabilitado && estadoPago && (estadoPago.colegiatura_mensual - estadoPago.total_pagado) > 0 && (
            <div className="bg-white rounded-[2rem] p-5 shadow-sm border border-slate-100 space-y-4">
              <div className="flex items-center gap-2">
                <div className="p-2 bg-brand-100 text-brand-600 rounded-lg"><CreditCard size={18} /></div>
                <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Pagar colegiatura con tarjeta</h3>
              </div>
              <div className="flex items-center justify-between text-xs font-bold text-slate-500">
                <span>Saldo pendiente de este mes</span>
                <span className="text-slate-800">${Number(estadoPago.colegiatura_mensual - estadoPago.total_pagado).toLocaleString('es-MX', { minimumFractionDigits: 2 })}</span>
              </div>
              {/* Monto editable -- el papá puede pagar menos que el saldo
                  completo si así le alcanza; lo que falte se sigue viendo
                  pendiente y entra a la deuda acumulada el próximo mes. */}
              <div>
                <label className="text-[9px] font-black text-slate-400 uppercase tracking-widest">¿Cuánto quieres pagar?</label>
                <div className="relative mt-1">
                  <span className="absolute inset-y-0 left-4 flex items-center text-slate-400 font-black">$</span>
                  <input
                    type="number"
                    min="10"
                    step="0.01"
                    max={estadoPago.colegiatura_mensual - estadoPago.total_pagado}
                    value={montoAPagar}
                    onChange={(e) => setMontoAPagar(e.target.value)}
                    className="w-full bg-slate-50 border-none rounded-2xl py-3 pl-8 pr-4 text-lg font-black text-slate-800 focus:ring-2 focus:ring-brand-500 transition-all"
                  />
                </div>
              </div>
              <button
                onClick={pagarColegiaturaEnLinea}
                disabled={iniciandoPagoEnLinea}
                className="w-full flex items-center justify-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-4 rounded-2xl shadow-md transition-all active:scale-95"
              >
                {iniciandoPagoEnLinea ? <Loader2 className="animate-spin" size={16} /> : <CreditCard size={16} />}
                Pagar ${montoAPagar || '0'} con tarjeta
              </button>
            </div>
          )}
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
                <button onClick={() => setReciboId(p.id)} title="Ver recibo" className="text-slate-300 hover:text-brand-600 hover:bg-brand-50 p-2.5 rounded-xl transition-colors shrink-0"><Receipt size={18} /></button>
              </div>
            ))
          )}
        </div>
      )}

      {vista === 'ausencias' && (
        <div className="max-w-md mx-auto p-4 space-y-4">
          <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100 space-y-4">
            <div className="flex items-center gap-2">
              <div className="p-2 bg-brand-100 text-brand-600 rounded-lg"><CalendarOff size={18} /></div>
              <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Avisar una ausencia</h3>
            </div>
            <p className="text-[11px] text-slate-400 font-medium">Avísale a la guardería con anticipación si tu hijo no va a asistir uno o varios días.</p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Desde</label>
                <input
                  type="date" min={hoyLocal()} value={formAusencia.fecha_inicio}
                  onChange={(e) => setFormAusencia({ ...formAusencia, fecha_inicio: e.target.value })}
                  className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold"
                />
              </div>
              <div>
                <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Hasta (opcional)</label>
                <input
                  type="date" min={formAusencia.fecha_inicio || hoyLocal()} value={formAusencia.fecha_fin}
                  onChange={(e) => setFormAusencia({ ...formAusencia, fecha_fin: e.target.value })}
                  className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold"
                />
              </div>
            </div>
            <div>
              <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Motivo (opcional)</label>
              <input
                type="text" value={formAusencia.motivo}
                onChange={(e) => setFormAusencia({ ...formAusencia, motivo: e.target.value })}
                placeholder="ej. Cita médica"
                className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium"
              />
            </div>
            <button
              onClick={reportarAusencia}
              disabled={guardandoAusencia}
              className="w-full flex items-center justify-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-3 rounded-xl shadow-md transition-all active:scale-95"
            >
              {guardandoAusencia ? <Loader2 className="animate-spin" size={16} /> : <Plus size={16} />} Avisar ausencia
            </button>
          </div>

          <div className="space-y-3">
            <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-widest ml-2">Próximas ausencias avisadas</h3>
            {loadingAusencias ? (
              <div className="py-8 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
            ) : ausencias.length === 0 ? (
              <div className="bg-white p-8 rounded-[2rem] border border-dashed border-slate-200 text-center">
                <p className="text-slate-400 font-bold uppercase text-[10px]">Sin ausencias avisadas</p>
              </div>
            ) : (
              ausencias.map((a) => (
                <div key={a.id} className="bg-white p-4 rounded-2xl border border-slate-100 flex items-center justify-between">
                  <div>
                    <p className="font-black text-sm text-slate-800">{a.fecha}</p>
                    {a.motivo && <p className="text-[10px] text-slate-400 font-bold mt-0.5">{a.motivo}</p>}
                  </div>
                  <button onClick={() => cancelarAusencia(a)} disabled={cancelandoId === a.id} title="Cancelar aviso" className="text-slate-300 hover:text-rose-500 hover:bg-rose-50 disabled:opacity-50 p-2 rounded-xl transition-colors shrink-0">
                    {cancelandoId === a.id ? <Loader2 className="animate-spin" size={16} /> : <Trash2 size={16} />}
                  </button>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {vista === 'comedor' && (
        <div className="max-w-md mx-auto p-4 space-y-4">
          <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100 space-y-4">
            <div className="flex items-center gap-2">
              <div className="p-2 bg-brand-100 text-brand-600 rounded-lg"><UtensilsCrossed size={18} /></div>
              <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Pedidos de Comedor</h3>
            </div>
            <p className="text-[11px] text-slate-400 font-medium">Por defecto tu hijo come las tres comidas del día. Avísale a la guardería solo cuando algo cambie (no desayuna, alergias, instrucciones especiales).</p>

            <div>
              <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Fecha</label>
              <input
                type="date" min={hoyLocal()} value={formComedor.fecha}
                onChange={(e) => setFormComedor({ ...formComedor, fecha: e.target.value })}
                className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold"
              />
            </div>

            <div className="flex gap-4">
              {[
                { key: 'desayuno', label: 'Desayuno' },
                { key: 'comida', label: 'Comida' },
                { key: 'merienda', label: 'Merienda' },
              ].map((comida) => (
                <label key={comida.key} className="flex items-center gap-2 text-xs font-bold text-slate-600">
                  <input
                    type="checkbox" checked={formComedor[comida.key]}
                    onChange={(e) => setFormComedor({ ...formComedor, [comida.key]: e.target.checked })}
                    className="w-4 h-4 accent-brand-600"
                  />
                  {comida.label}
                </label>
              ))}
            </div>

            <div>
              <label className="text-[9px] font-black text-slate-400 uppercase ml-1 mb-1 block">Notas (opcional)</label>
              <input
                type="text" value={formComedor.notas}
                onChange={(e) => setFormComedor({ ...formComedor, notas: e.target.value })}
                placeholder="ej. Alergia a los cacahuates"
                className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium"
              />
            </div>

            <button
              onClick={guardarPedidoComedor}
              disabled={guardandoComedor}
              className="w-full flex items-center justify-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-3 rounded-xl shadow-md transition-all active:scale-95"
            >
              {guardandoComedor ? <Loader2 className="animate-spin" size={16} /> : <Plus size={16} />} Avisar
            </button>
          </div>

          <div className="space-y-3">
            <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-widest ml-2">Excepciones registradas</h3>
            {loadingComedor ? (
              <div className="py-8 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
            ) : pedidosComedor.length === 0 ? (
              <div className="bg-white p-8 rounded-[2rem] border border-dashed border-slate-200 text-center">
                <p className="text-slate-400 font-bold uppercase text-[10px]">Sin excepciones, come normal todos los días</p>
              </div>
            ) : (
              pedidosComedor.map((p) => {
                const faltantes = [
                  !p.desayuno && 'Desayuno',
                  !p.comida && 'Comida',
                  !p.merienda && 'Merienda',
                ].filter(Boolean);
                return (
                  <div key={p.id} className="bg-white p-4 rounded-2xl border border-slate-100 flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-black text-sm text-slate-800">{p.fecha}</p>
                      {faltantes.length > 0 && <p className="text-[10px] text-rose-500 font-bold mt-0.5">No come: {faltantes.join(', ')}</p>}
                      {p.notas && <p className="text-[10px] text-slate-400 font-bold mt-0.5">{p.notas}</p>}
                    </div>
                    <button onClick={() => restablecerPedidoComedor(p)} disabled={restableciendoFecha === p.fecha} title="Restablecer al comedor normal" className="text-slate-300 hover:text-brand-600 hover:bg-brand-50 disabled:opacity-50 p-2 rounded-xl transition-colors shrink-0">
                      {restableciendoFecha === p.fecha ? <Loader2 className="animate-spin" size={16} /> : <RotateCcw size={16} />}
                    </button>
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}

      {vista === 'galeria' && (
        <div className="max-w-md mx-auto p-4">
          <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100">
            <GaleriaFotos hijoId={hijoId} rutaBase="/padre/hijos" onFotoClick={setFotoSeleccionada} />
          </div>
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