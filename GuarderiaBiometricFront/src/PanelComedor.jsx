import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import { UtensilsCrossed, Sunrise, Soup, Cookie, Baby } from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { hoyLocal } from './utils/fecha';

// PanelComedor -- "Pedidos para el comedor o catering" del PDF de
// referencia (lado staff): cuántas porciones de cada comida hay que
// preparar hoy (o cualquier día), descontando las excepciones que avisaron
// los padres, más el detalle de quién tiene alguna nota (alergias,
// instrucciones).
const PanelComedor = () => {
  const [fecha, setFecha] = useState(hoyLocal());
  const [resumen, setResumen] = useState(null);
  const [loading, setLoading] = useState(true);

  const cargar = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get('/pedidos-comedor', { params: { fecha } });
      setResumen(res.data);
    } catch (err) {
      console.error('Error al cargar el resumen del comedor:', err);
      mostrarError('No se pudo cargar el resumen del comedor');
    } finally {
      setLoading(false);
    }
  }, [fecha]);

  useEffect(() => { cargar(); }, [cargar]);

  const formatoFechaLarga = (iso) => {
    try {
      const [anio, mes, dia] = iso.split('-').map(Number);
      return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { weekday: 'long', day: 'numeric', month: 'long' });
    } catch {
      return iso;
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><UtensilsCrossed size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Pedidos de Comedor</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Cuántas porciones preparar hoy</p>
            </div>
          </div>
          <input type="date" value={fecha} onChange={(e) => setFecha(e.target.value)} className="bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold" />
        </div>

        {loading || !resumen ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : (
          <>
            <p className="text-[10px] font-black text-brand-500 uppercase tracking-widest mb-4 ml-1">{formatoFechaLarga(resumen.fecha)} · {resumen.total_ninos} niños activos</p>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
              {[
                { label: 'Desayuno', valor: resumen.resumen.desayuno, icon: Sunrise },
                { label: 'Comida', valor: resumen.resumen.comida, icon: Soup },
                { label: 'Merienda', valor: resumen.resumen.merienda, icon: Cookie },
              ].map((item) => {
                const Icono = item.icon;
                return (
                  <div key={item.label} className="bg-slate-50 border border-slate-100 rounded-[1.75rem] p-5 flex items-center gap-4">
                    <div className="bg-white p-3 rounded-2xl text-brand-500 border border-slate-100"><Icono size={22} /></div>
                    <div>
                      <p className="text-2xl font-black text-slate-900 leading-none">{item.valor}</p>
                      <p className="text-[9px] font-black text-slate-400 uppercase mt-1">{item.label}</p>
                    </div>
                  </div>
                );
              })}
            </div>

            <h4 className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-3 ml-1">Excepciones y notas</h4>
            {resumen.excepciones.length === 0 ? (
              <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Ninguna familia avisó excepciones para este día</div>
            ) : (
              <div className="space-y-2">
                {resumen.excepciones.map((ex) => {
                  const faltantes = [
                    !ex.desayuno && 'Desayuno',
                    !ex.comida && 'Comida',
                    !ex.merienda && 'Merienda',
                  ].filter(Boolean);
                  return (
                    <div key={ex.id} className="flex items-center gap-4 p-4 rounded-2xl border border-slate-100 bg-slate-50">
                      <div className="bg-white p-2.5 rounded-xl text-slate-300 border border-slate-100 shrink-0"><Baby size={18} /></div>
                      <div className="min-w-0">
                        <p className="font-black text-sm text-slate-900">{ex.hijo_nombre}</p>
                        {faltantes.length > 0 && <p className="text-xs text-rose-500 font-bold">No come: {faltantes.join(', ')}</p>}
                        {ex.notas && <p className="text-xs text-slate-500 font-medium">{ex.notas}</p>}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default PanelComedor;
