import React, { useEffect, useState } from 'react';
import api from '../axiosConfig';
import { Users, Check } from 'lucide-react';

// SelectorGrupos -- "que si alguien quiere hacer una encuesta pueda escoger a
// quién va dirigida, y de la misma forma las circulares".
//
// Lo comparten PanelCirculares y PanelEncuestas porque la decisión es la
// misma en los dos: o va para todas las familias (el comportamiento de
// siempre, y el que queda por default) o para los salones que se marquen.
//
// `seleccionados` es la lista de ids elegidos; vacía significa "para todas",
// así que el componente no necesita un estado aparte para esa opción.
const SelectorGrupos = ({ seleccionados, onChange, acento }) => {
  const [grupos, setGrupos] = useState([]);
  const [cargando, setCargando] = useState(true);

  useEffect(() => {
    let vigente = true;
    api.get('/admin/grupos')
      .then((res) => { if (vigente) setGrupos(Array.isArray(res.data) ? res.data : []); })
      .catch((err) => console.error('Error al cargar los grupos:', err))
      .finally(() => { if (vigente) setCargando(false); });
    return () => { vigente = false; };
  }, []);

  const alternar = (id) => {
    onChange(seleccionados.includes(id) ? seleccionados.filter((g) => g !== id) : [...seleccionados, id]);
  };

  // Una guardería que todavía no creó salones no tiene nada que elegir: en
  // vez de un bloque vacío y confuso, se explica dónde se crean.
  if (!cargando && grupos.length === 0) {
    return (
      <p className="text-[11px] text-slate-400 font-bold">
        Todavía no hay grupos creados, así que esto le llega a todas las familias.
        Los grupos se crean desde Perfiles.
      </p>
    );
  }

  return (
    <div>
      <label className="flex items-center gap-2 text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">
        <Users size={13} /> ¿A quién va dirigida?
      </label>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => onChange([])}
          className={`flex items-center gap-1.5 text-[11px] font-black uppercase px-3.5 py-2 rounded-xl border transition-all ${
            seleccionados.length === 0
              ? 'bg-forest text-white border-forest'
              : 'bg-white text-slate-500 border-slate-200 hover:border-slate-300'
          }`}
        >
          {seleccionados.length === 0 && <Check size={13} />} Todas las familias
        </button>

        {grupos.map((g) => {
          const activo = seleccionados.includes(g.id);
          return (
            <button
              key={g.id}
              type="button"
              onClick={() => alternar(g.id)}
              className={`flex items-center gap-1.5 text-[11px] font-black uppercase px-3.5 py-2 rounded-xl border transition-all ${
                activo
                  ? `${acento.fondo} ${acento.texto} ${acento.borde}`
                  : 'bg-white text-slate-500 border-slate-200 hover:border-slate-300'
              }`}
            >
              {activo && <Check size={13} />} {g.nombre}
              <span className="text-slate-400 font-bold normal-case">({g.ninos_activos})</span>
            </button>
          );
        })}
      </div>

      <p className="text-[10px] text-slate-400 font-bold mt-2">
        {seleccionados.length === 0
          ? 'Se le mostrará a todas las familias de la guardería.'
          : 'Solo la verán las familias con un hijo en los grupos marcados, y solo a ellas les llegará la notificación.'}
      </p>
    </div>
  );
};

export default SelectorGrupos;
