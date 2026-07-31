import React from 'react';
import { ShieldCheck, X, Check } from 'lucide-react';

// Se muestra antes de enrolar el rostro de un tutor nuevo: sin esto no hay
// evidencia de que alguien vio y aceptó el Aviso de Privacidad para datos
// biométricos y de menores (LFPDPPP). El texto lo escribe el admin desde el
// panel de Configuración — este componente solo lo presenta y captura la
// aceptación.
const AvisoPrivacidadModal = ({ texto, version, onAceptar, onCancelar }) => {
  return (
    <div className="fixed inset-0 z-[250] flex items-center justify-center p-4 bg-slate-900/70 backdrop-blur-sm">
      <div className="bg-white w-full max-w-lg rounded-[2rem] shadow-2xl flex flex-col max-h-[85vh] overflow-hidden">
        <div className="p-6 sm:p-8 pb-4 border-b border-slate-100 flex items-start gap-4">
          <div className="bg-brand-100 p-3 rounded-2xl text-brand-600 shrink-0"><ShieldCheck size={28} /></div>
          <div>
            <h2 className="text-lg font-black uppercase text-slate-900 leading-tight">Aviso de Privacidad</h2>
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-1">
              Versión {version} · Léelo con el tutor antes de continuar
            </p>
          </div>
          <button onClick={onCancelar} className="ml-auto text-slate-400 hover:text-slate-600 p-1"><X size={24} /></button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 sm:px-8 py-6 custom-scrollbar">
          <p className="whitespace-pre-wrap text-sm text-slate-700 leading-relaxed">{texto}</p>
        </div>

        <div className="p-6 sm:p-8 pt-4 border-t border-slate-100 space-y-3">
          <p className="text-[10px] text-slate-400 text-center">
            Al aceptar, confirmas que el tutor presente leyó este aviso y está de acuerdo.
          </p>
          <div className="flex gap-3">
            <button onClick={onCancelar} className="flex-1 py-4 text-slate-500 font-bold uppercase text-xs rounded-2xl hover:bg-slate-50">
              Cancelar
            </button>
            <button onClick={onAceptar} className="flex-1 flex items-center justify-center gap-2 py-4 bg-brand-600 hover:bg-brand-700 text-white font-black uppercase text-xs rounded-2xl shadow-lg active:scale-95 transition-all">
              <Check size={18} /> Acepto en nombre del tutor presente
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AvisoPrivacidadModal;
