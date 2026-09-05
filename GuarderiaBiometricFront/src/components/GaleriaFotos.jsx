import React, { useState, useEffect } from 'react';
import api from '../axiosConfig';
import { Image as ImageIcon } from 'lucide-react';
import { mostrarError } from '../utils/alertas';

const formatoFecha = (iso) => {
  try {
    const [anio, mes, dia] = iso.split('-').map(Number);
    return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', year: 'numeric' });
  } catch {
    return iso;
  }
};

// GaleriaFotos -- "Galería de fotos" del PDF de referencia: junta en una
// sola vista todas las fotos que ya se suben día a día desde la bitácora,
// agrupadas por fecha, en vez de tener que revisar reporte por reporte.
// Compartido entre PanelPerfiles.jsx (staff, rutaBase="/hijos") y
// VistaPadreDetalle.jsx (padre, rutaBase="/padre/hijos") -- mismo criterio
// que ReciboPago.jsx.
const GaleriaFotos = ({ hijoId, rutaBase, onFotoClick }) => {
  const [fotos, setFotos] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const cargar = async () => {
      setLoading(true);
      try {
        const res = await api.get(`${rutaBase}/${hijoId}/galeria`);
        setFotos(Array.isArray(res.data) ? res.data : []);
      } catch (err) {
        console.error('Error al cargar la galería:', err);
        mostrarError('No se pudo cargar la galería de fotos');
      } finally {
        setLoading(false);
      }
    };
    cargar();
  }, [hijoId, rutaBase]);

  if (loading) {
    return <div className="py-8 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>;
  }
  if (fotos.length === 0) {
    return (
      <div className="py-10 text-center">
        <ImageIcon size={32} className="mx-auto text-slate-200 mb-3" />
        <p className="text-slate-400 font-bold uppercase text-[10px]">Sin fotos todavía</p>
      </div>
    );
  }

  const porFecha = fotos.reduce((acc, f) => {
    (acc[f.fecha] = acc[f.fecha] || []).push(f);
    return acc;
  }, {});

  return (
    <div className="space-y-6">
      {Object.keys(porFecha).map((fecha) => (
        <div key={fecha}>
          <p className="text-[10px] font-black text-brand-500 uppercase tracking-widest mb-2 ml-1">{formatoFecha(fecha)}</p>
          <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
            {porFecha[fecha].map((f) => (
              <button
                key={f.id}
                onClick={() => onFotoClick?.(f.url)}
                className="aspect-square rounded-xl overflow-hidden border border-slate-100 bg-slate-50 hover:ring-2 hover:ring-brand-400 transition-all"
              >
                <img src={f.url} alt="" className="w-full h-full object-cover" loading="lazy" />
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

export default GaleriaFotos;
