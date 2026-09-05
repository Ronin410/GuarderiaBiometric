import React, { useState, useEffect, useCallback } from 'react';
import api from './axiosConfig';
import { ChevronLeft, ChevronRight, UtensilsCrossed, Coffee, Soup, Cookie } from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { hoyLocal, lunesDeLaSemana, diasHabilesDeLaSemana } from './utils/fecha';
import { ACENTOS } from './utils/acentos';
import DinoDecorativo from './components/DinoDecorativo';

// Mismo color con el que este acceso aparece en el tablero del papá
// (DashboardPadre), definido en utils/acentos.js.
const acento = ACENTOS.amarillo;

const NOMBRES_DIA = ['Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes'];

// MenuPadre -- "me gusta que les aparezca el menú de hoy, pero quiero que
// puedan ver también qué comieron en días anteriores o posteriores": el
// dashboard solo mostraba el día de hoy (menuHoy); esto es la vista de
// pantalla completa (mismo patrón que ChatPadre/EncuestasPadre/
// CircularesPadre/EventosPadre) con la semana entera, navegable
// atrás/adelante -- misma UI que el PanelMenu del staff, pero de solo
// lectura y contra /padre/menu-semanal (mismo endpoint que ya usa
// DashboardPadre para el menú de hoy, solo que pidiendo la semana
// completa en vez de un solo día).
const MenuPadre = ({ onVolver }) => {
  const [lunes, setLunes] = useState(lunesDeLaSemana(hoyLocal()));
  const [porFecha, setPorFecha] = useState({});
  const [loading, setLoading] = useState(true);

  const hoy = hoyLocal();
  const dias = diasHabilesDeLaSemana(lunes);

  const cargarSemana = useCallback(async () => {
    setLoading(true);
    try {
      const diasSemana = diasHabilesDeLaSemana(lunes);
      const res = await api.get('/padre/menu-semanal', { params: { inicio: diasSemana[0], fin: diasSemana[4] } });
      const mapa = {};
      (Array.isArray(res.data) ? res.data : []).forEach((d) => {
        mapa[d.fecha] = { desayuno: d.desayuno || '', comida: d.comida || '', merienda: d.merienda || '' };
      });
      setPorFecha(mapa);
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

  const formatoLargo = (fechaISO) => {
    const d = new Date(`${fechaISO}T00:00:00`);
    return d.toLocaleDateString('es-MX', { day: 'numeric', month: 'short', timeZone: 'UTC' });
  };

  return (
    <div className="min-h-screen bg-slate-50 pb-10 animate-in fade-in duration-500">
      <div className="bg-white p-6 pb-8 rounded-b-[3rem] shadow-sm border-b border-slate-100 sticky top-0 z-30">
        <button
          onClick={onVolver}
          className="flex items-center gap-2 text-slate-400 font-black uppercase text-[10px] tracking-widest mb-6 hover:text-brand-600 transition-colors"
        >
          <ChevronLeft size={16} /> Volver
        </button>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3 min-w-0">
            <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter">Menú de la semana</h2>
            <DinoDecorativo src="/dinos/dino-amarillo.png" className="h-11 w-auto shrink-0" />
          </div>
          <div className={`${acento.solido} p-3 rounded-2xl text-white shadow-lg`}>
            <UtensilsCrossed size={20} />
          </div>
        </div>

        <div className="flex items-center gap-2 bg-slate-50 border border-slate-200 rounded-2xl px-2 py-2 mt-5 w-fit">
          <button onClick={() => cambiarSemana(-1)} className="p-2 rounded-xl hover:bg-white text-slate-500 transition-all"><ChevronLeft size={18} /></button>
          <span className="text-xs font-black uppercase text-slate-700 px-2">
            {formatoLargo(dias[0])} al {formatoLargo(dias[4])}
          </span>
          <button onClick={() => cambiarSemana(1)} className="p-2 rounded-xl hover:bg-white text-slate-500 transition-all"><ChevronRight size={18} /></button>
        </div>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-3">
        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : (
          dias.map((fecha, i) => {
            const dia = porFecha[fecha];
            const esHoy = fecha === hoy;
            const sinNada = !dia || (!dia.desayuno && !dia.comida && !dia.merienda);
            return (
              <div
                key={fecha}
                className={`bg-white p-5 rounded-[2rem] border shadow-sm ${esHoy ? 'border-brand-300 ring-2 ring-brand-100' : 'border-slate-100'}`}
              >
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <p className="font-black uppercase text-slate-900 text-sm">{NOMBRES_DIA[i]}</p>
                    <p className="text-[10px] text-brand-500 font-bold uppercase">{formatoLargo(fecha)}</p>
                  </div>
                  {esHoy && (
                    <span className="text-[9px] font-black uppercase text-white bg-brand-600 px-2.5 py-1 rounded-full tracking-widest">Hoy</span>
                  )}
                </div>

                {sinNada ? (
                  <p className="text-[11px] text-slate-400 font-bold uppercase tracking-wide">Sin menú capturado</p>
                ) : (
                  <div className="space-y-1.5">
                    {dia.desayuno && (
                      <p className="text-xs font-bold text-slate-700 flex items-start gap-1.5">
                        <Coffee size={13} className="text-brand-500 shrink-0 mt-0.5" /> <span><span className="text-brand-500">Desayuno:</span> {dia.desayuno}</span>
                      </p>
                    )}
                    {dia.comida && (
                      <p className="text-xs font-bold text-slate-700 flex items-start gap-1.5">
                        <Soup size={13} className="text-brand-500 shrink-0 mt-0.5" /> <span><span className="text-brand-500">Comida:</span> {dia.comida}</span>
                      </p>
                    )}
                    {dia.merienda && (
                      <p className="text-xs font-bold text-slate-700 flex items-start gap-1.5">
                        <Cookie size={13} className="text-brand-500 shrink-0 mt-0.5" /> <span><span className="text-brand-500">Merienda:</span> {dia.merienda}</span>
                      </p>
                    )}
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

export default MenuPadre;
