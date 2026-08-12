import React, { useState, useEffect } from 'react';
import api from './../axiosConfig';
import { ArrowLeft, Printer, ShieldCheck } from 'lucide-react';
import { mostrarError } from './../utils/alertas';

// ReciboPago -- vista imprimible de un pago ya registrado. Se monta a pantalla
// completa (reemplaza todo el contenido del panel que lo usa, mismo patrón que
// PanelReportes.jsx) para que el @media print global (index.css) que oculta
// header/nav/botones no deje residuos de otra pantalla mezclados en el papel.
//
// rutaBase permite reusar el mismo componente desde el panel de staff
// (/pagos/:id/recibo) y desde el portal del padre (/padre/pagos/:id/recibo),
// que exponen el mismo recibo pero con distinta verificación de permisos del
// lado del backend.
const ReciboPago = ({ pagoId, rutaBase = '/pagos', onVolver }) => {
  const [recibo, setRecibo] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const cargar = async () => {
      setLoading(true);
      try {
        const res = await api.get(`${rutaBase}/${pagoId}/recibo`);
        setRecibo(res.data);
      } catch (err) {
        console.error('Error al cargar el recibo:', err);
        mostrarError('No se pudo cargar el recibo');
      } finally {
        setLoading(false);
      }
    };
    cargar();
  }, [pagoId, rutaBase]);

  if (loading) {
    return <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando recibo...</div>;
  }
  if (!recibo) {
    return (
      <div className="py-20 text-center space-y-4">
        <p className="text-slate-400 font-black uppercase tracking-widest text-xs">No se pudo cargar el recibo</p>
        <button onClick={onVolver} className="no-print text-brand-600 font-black uppercase text-xs">Volver</button>
      </div>
    );
  }

  return (
    <div className="animate-in fade-in duration-500 max-w-xl mx-auto">
      <div className="no-print flex items-center justify-between mb-6">
        <button onClick={onVolver} className="flex items-center gap-2 text-brand-600 font-black uppercase text-xs tracking-widest hover:opacity-70 transition-all">
          <ArrowLeft size={16} /> Volver
        </button>
        <button onClick={() => window.print()} className="bg-brand-600 hover:bg-brand-700 text-white font-black uppercase text-xs px-5 py-3 rounded-xl shadow-md flex items-center gap-2 transition-all active:scale-95">
          <Printer size={16} /> Imprimir / Guardar PDF
        </button>
      </div>

      <div className="bg-white p-8 sm:p-10 rounded-[2.5rem] border border-slate-200 shadow-xl print:shadow-none print:border-0 print:rounded-none">
        <div className="flex items-center justify-between border-b border-dashed border-slate-200 pb-6 mb-6">
          <div className="flex items-center gap-3">
            <div className="bg-brand-600 p-2 rounded-xl"><ShieldCheck size={20} className="text-white" /></div>
            <div>
              <p className="font-black uppercase text-slate-900 leading-none">{recibo.guarderia_nombre}</p>
              {recibo.guarderia_direccion && <p className="text-[10px] text-slate-400 font-bold mt-1">{recibo.guarderia_direccion}</p>}
            </div>
          </div>
          <div className="text-right">
            <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Recibo</p>
            <p className="font-black text-slate-900">{recibo.folio}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm mb-6">
          <div>
            <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Alumno</p>
            <p className="font-bold text-slate-800">{recibo.nino_nombre}</p>
          </div>
          <div>
            <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Fecha de pago</p>
            <p className="font-bold text-slate-800">{recibo.fecha_pago}</p>
          </div>
          <div>
            <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Concepto</p>
            <p className="font-bold text-slate-800">{recibo.concepto}</p>
          </div>
          <div>
            <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Periodo</p>
            <p className="font-bold text-slate-800">{recibo.periodo}</p>
          </div>
          <div>
            <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Método de pago</p>
            <p className="font-bold text-slate-800 capitalize">{recibo.metodo_pago}</p>
          </div>
          {recibo.observaciones && (
            <div className="col-span-2">
              <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Observaciones</p>
              <p className="font-bold text-slate-800">{recibo.observaciones}</p>
            </div>
          )}
        </div>

        <div className="bg-brand-50 border border-brand-100 rounded-2xl p-6 flex items-center justify-between print:bg-white print:border-slate-300">
          <p className="text-[10px] font-black text-brand-600 uppercase tracking-widest">Total pagado</p>
          <p className="text-2xl font-black text-brand-700">${Number(recibo.monto).toLocaleString('es-MX', { minimumFractionDigits: 2 })}</p>
        </div>

        <p className="text-center text-[9px] text-slate-300 font-bold uppercase tracking-widest mt-8">Generado por BioSafe</p>
      </div>
    </div>
  );
};

export default ReciboPago;
