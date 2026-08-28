import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import { ChevronLeft, Megaphone, CalendarClock } from 'lucide-react';
import { mostrarError } from './utils/alertas';

// CircularesPadre -- vista de pantalla completa (mismo patrón que
// EncuestasPadre/ChatPadre: swap desde DashboardPadre) con TODOS los avisos
// de la guardería, no solo los últimos 3 que se alcanzaban a ver antes en
// el inicio. El inicio ahora solo avisa "hay una circular nueva" y manda
// para acá -- ver el comentario largo en DashboardPadre.jsx.
const CircularesPadre = ({ onVolver }) => {
  const [circulares, setCirculares] = useState([]);
  const [loading, setLoading] = useState(true);

  const cargar = async () => {
    setLoading(true);
    try {
      const res = await api.get('/padre/circulares');
      const todas = Array.isArray(res.data) ? res.data : [];
      setCirculares(todas);
      // Se marcan como leídas las que de verdad se muestran aquí (el
      // listado completo) -- no desde el teaser del inicio, que solo
      // enseña un título, no el aviso completo. Mismo criterio que ya
      // usaba el dashboard antes: el conteo que ve staff debe reflejar
      // avisos vistos de verdad.
      todas.forEach((cir) => {
        api.post(`/padre/circulares/${cir.id}/leido`).catch((err) => {
          console.error('Error al marcar circular como leída', err);
        });
      });
    } catch (err) {
      console.error('Error al cargar circulares:', err);
      mostrarError('No se pudieron cargar los avisos');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const formatoFecha = (iso) => {
    try {
      return new Date(iso).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' });
    } catch {
      return iso;
    }
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
          <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter">Avisos</h2>
          <div className="bg-brand-600 p-3 rounded-2xl text-white shadow-lg shadow-brand-200">
            <Megaphone size={20} />
          </div>
        </div>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-4">
        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : circulares.length === 0 ? (
          <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center">
            <Megaphone size={40} className="mx-auto text-slate-200 mb-4" />
            <p className="text-slate-400 font-bold uppercase text-[10px]">Todavía no hay avisos de la guardería</p>
          </div>
        ) : (
          circulares.map((cir) => (
            <div key={cir.id} className="bg-white rounded-[2rem] p-6 shadow-sm border border-slate-100 space-y-2">
              <p className="font-black text-slate-900 uppercase text-sm">{cir.titulo}</p>
              <p className="text-[9px] text-brand-500 font-bold uppercase flex items-center gap-1">
                <CalendarClock size={11} /> {formatoFecha(cir.creado_en)}
              </p>
              <p className="text-xs text-slate-600 font-medium whitespace-pre-wrap">{cir.contenido}</p>
              {cir.imagen_url && (
                <img src={cir.imagen_url} alt={cir.titulo} className="mt-2 w-full max-h-72 rounded-2xl border border-slate-100 object-cover" />
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default CircularesPadre;
