import React, { useState, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown } from 'lucide-react';

// NavDropdown agrupa varias pestañas del menú principal bajo un solo botón
// (ej. "Alumnos" abre Familia/Perfiles) -- construido a mano porque el
// proyecto no trae ninguna librería de menús (headlessui, radix, etc.).
//
// El panel se monta con un portal a document.body en vez de vivir dentro del
// <nav> -- el <nav> tiene overflow-x-auto (para poder hacer scroll horizontal
// en pantallas angostas), y por la regla de CSS Overflow "si un eje no es
// visible, el otro eje deja de ser visible también", overflow-x-auto
// recortaría verticalmente cualquier panel absolute anidado ahí adentro. El
// portal + position:fixed calculado con getBoundingClientRect lo evita del
// todo, sin importar el overflow de ningún ancestro.
const NavDropdown = ({ label, Icon, items, tabActual, onSeleccionar }) => {
  const [abierto, setAbierto] = useState(false);
  const [posicion, setPosicion] = useState({ top: 0, left: 0 });
  const botonRef = useRef(null);
  const panelRef = useRef(null);

  const grupoActivo = items.some((it) => it.tab === tabActual);

  const alternar = () => {
    if (!abierto && botonRef.current) {
      const rect = botonRef.current.getBoundingClientRect();
      setPosicion({ top: rect.bottom + 8, left: rect.left });
    }
    setAbierto((v) => !v);
  };

  useEffect(() => {
    if (!abierto) return;
    const alHacerClicFuera = (e) => {
      if (
        panelRef.current && !panelRef.current.contains(e.target) &&
        botonRef.current && !botonRef.current.contains(e.target)
      ) {
        setAbierto(false);
      }
    };
    const alPresionarTecla = (e) => {
      if (e.key === 'Escape') setAbierto(false);
    };
    // Nota: a propósito NO se cierra en scroll -- el <nav> que contiene este
    // botón tiene overflow-x-auto (scroll horizontal en pantallas angostas),
    // y un listener de scroll con capture:true también atrapa ese scroll
    // interno del propio nav (ej. el que dispara el navegador al enfocar el
    // botón), cerrando el panel casi al instante. El clic afuera y Escape ya
    // cubren el cierre; una posición ligeramente desactualizada si alguien
    // hace scroll de la página con el panel abierto es un costo aceptable.
    document.addEventListener('mousedown', alHacerClicFuera);
    document.addEventListener('keydown', alPresionarTecla);
    return () => {
      document.removeEventListener('mousedown', alHacerClicFuera);
      document.removeEventListener('keydown', alPresionarTecla);
    };
  }, [abierto]);

  return (
    <>
      <button
        ref={botonRef}
        onClick={alternar}
        className={`px-4 py-2.5 rounded-xl flex items-center gap-2 font-bold transition-all ${grupoActivo ? 'bg-brand-600 text-white shadow-md' : 'text-slate-500 hover:bg-slate-50'}`}
      >
        <Icon size={18} /> {label}
        <ChevronDown size={14} className={`transition-transform ${abierto ? 'rotate-180' : ''}`} />
      </button>

      {abierto && createPortal(
        <div
          ref={panelRef}
          style={{ top: posicion.top, left: posicion.left }}
          className="fixed bg-white border border-slate-200 rounded-2xl shadow-xl p-1.5 min-w-[200px] z-50 animate-in fade-in zoom-in-95 duration-150"
        >
          {items.map(({ tab, label: itemLabel, Icon: ItemIcon }) => (
            <button
              key={tab}
              onClick={() => { onSeleccionar(tab); setAbierto(false); }}
              className={`w-full px-4 py-2.5 rounded-xl flex items-center gap-2.5 font-bold text-sm transition-all text-left ${tab === tabActual ? 'bg-brand-50 text-brand-600' : 'text-slate-600 hover:bg-slate-50'}`}
            >
              <ItemIcon size={16} /> {itemLabel}
            </button>
          ))}
        </div>,
        document.body
      )}
    </>
  );
};

export default NavDropdown;
