import React from 'react';
import { Utensils, Moon, Camera, CheckCircle2, Baby, Info } from 'lucide-react';

const COLOR_COMIDA = { desayuno: 'bg-blue-500', comida: 'bg-orange-500', merienda: 'bg-pink-500' };

const ItemComida = ({ label, valor, color }) => (
  <div className="flex gap-5">
    <div className="flex flex-col items-center">
      <div className={`w-3.5 h-3.5 rounded-full border-2 border-white shadow-sm ${valor ? color : 'bg-slate-200'}`}></div>
      <div className="w-0.5 h-full bg-slate-100 mt-1"></div>
    </div>
    <div className="pb-2">
      <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest mb-0.5">{label}</p>
      <p className={`text-sm font-black uppercase ${valor ? 'text-slate-800' : 'text-slate-300 italic'}`}>
        {valor || 'Pendiente'}
      </p>
    </div>
  </div>
);

// ReporteDiario es la vista de solo lectura de la bitácora de un día (comidas,
// siesta, esfínter, fotos, observaciones). La usan tanto ReportePublico.jsx
// (link público por WhatsApp, sin login) como VistaPadreDetalle.jsx (portal
// autenticado del papá) — antes cada uno tenía su propia copia casi idéntica
// de este bloque. Solo recibe los datos ya cargados: cada caller sigue
// haciendo su propio fetch (endpoints distintos) y le pasa el resultado aquí.
const ReporteDiario = ({ reporte, onFotoClick, tituloObservaciones = 'Observaciones del día' }) => {
  if (!reporte) return null;

  return (
    <>
      <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100">
        <div className="flex items-center gap-3 mb-6">
          <div className="p-2 bg-amber-100 text-amber-600 rounded-xl"><Utensils size={20} /></div>
          <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Alimentación</h3>
        </div>
        <div className="space-y-5">
          <ItemComida label="Desayuno" valor={reporte.desayuno} color={COLOR_COMIDA.desayuno} />
          <ItemComida label="Comida" valor={reporte.comida} color={COLOR_COMIDA.comida} />
          <ItemComida label="Merienda" valor={reporte.merienda} color={COLOR_COMIDA.merienda} />
        </div>
      </div>

      <div className={`rounded-[2.5rem] p-6 border transition-all ${reporte.durmio ? 'bg-brand-600 border-brand-500 text-white shadow-xl shadow-brand-100' : 'bg-white border-slate-100 text-slate-400'}`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className={`p-3 rounded-2xl ${reporte.durmio ? 'bg-white/20' : 'bg-slate-50 text-slate-300'}`}><Moon size={28} /></div>
            <div>
              <p className="font-black uppercase text-xs tracking-widest">Descanso</p>
              <p className="text-[10px] font-bold uppercase opacity-80">{reporte.durmio ? 'Tomó su siesta' : 'No hubo siesta'}</p>
            </div>
          </div>
          {reporte.durmio && <CheckCircle2 size={28} />}
        </div>
      </div>

      <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100 flex items-center gap-4">
        <div className="p-3 bg-emerald-50 text-emerald-600 rounded-2xl"><Baby size={28} /></div>
        <div>
          <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Control de Esfínter</p>
          <p className="text-base font-bold text-slate-800">{reporte.esfinter || 'Sin reporte'}</p>
        </div>
      </div>

      {reporte.fotos?.length > 0 && (
        <div className="bg-white rounded-[2.5rem] p-6 shadow-sm border border-slate-100">
          <div className="flex items-center gap-3 mb-5">
            <div className="p-2 bg-rose-100 text-rose-600 rounded-xl"><Camera size={20} /></div>
            <h3 className="font-black text-slate-900 uppercase text-xs tracking-widest">Evidencia del Día</h3>
          </div>
          <div className="grid grid-cols-2 gap-3">
            {reporte.fotos.map((url, idx) => (
              <div
                key={idx}
                className="aspect-square rounded-[2rem] overflow-hidden bg-slate-100 border border-slate-100 cursor-zoom-in"
                onClick={() => onFotoClick?.(url)}
              >
                <img src={url} alt="Evidencia" className="w-full h-full object-cover" />
              </div>
            ))}
          </div>
        </div>
      )}

      {reporte.observaciones && (
        <div className="bg-slate-900 text-white rounded-[2.5rem] p-8 shadow-2xl relative overflow-hidden">
          <div className="absolute top-0 right-0 p-4 opacity-10"><Info size={80} /></div>
          <div className="relative z-10">
            <div className="flex items-center gap-2 mb-4 text-brand-400">
              <h3 className="font-black uppercase text-[10px] tracking-[0.2em]">{tituloObservaciones}</h3>
            </div>
            <p className="text-base leading-relaxed italic font-medium opacity-95">"{reporte.observaciones}"</p>
          </div>
        </div>
      )}
    </>
  );
};

export default ReporteDiario;
