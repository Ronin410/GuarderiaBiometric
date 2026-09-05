import React from 'react';
import { Users } from 'lucide-react';

// EtiquetaGrupos -- la línea de "a quién se le mandó" en los listados de
// circulares y encuestas. Se muestra siempre, también cuando fue para todas
// las familias: sin esa etiqueta habría que adivinar si una publicación sin
// grupos es que se mandó a todos o que la etiqueta no cargó.
const EtiquetaGrupos = ({ paraTodos, grupos = [], acento }) => {
  if (paraTodos) {
    return (
      <span className="inline-flex items-center gap-1.5 text-[10px] font-black uppercase tracking-wide text-slate-400">
        <Users size={12} /> Todas las familias
      </span>
    );
  }

  // Dirigida pero sin grupos: le pasa a una publicación cuyos salones se
  // borraron después. Ya no le aparece a nadie, y decirlo es más útil que
  // dejar la línea vacía.
  if (grupos.length === 0) {
    return (
      <span className="inline-flex items-center gap-1.5 text-[10px] font-black uppercase tracking-wide text-amber-600">
        <Users size={12} /> Sus grupos ya no existen
      </span>
    );
  }

  return (
    <span className={`inline-flex items-center gap-1.5 text-[10px] font-black uppercase tracking-wide ${acento.texto}`}>
      <Users size={12} /> {grupos.map((g) => g.nombre).join(' · ')}
    </span>
  );
};

export default EtiquetaGrupos;
