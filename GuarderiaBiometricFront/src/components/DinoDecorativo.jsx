import React from 'react';

// DinoDecorativo -- el dinosaurio de la marca acompañando el título de un
// apartado. Es puro adorno, por eso va con alt vacío y aria-hidden: un
// lector de pantalla no gana nada anunciándolo, el nombre de la sección ya
// está en el encabezado de al lado.
//
// La posición y el tamaño los pone quien lo usa (`className`), porque en
// unos lugares va en el flujo del encabezado y en otros absoluto sobre una
// tarjeta. `espejo` voltea la ilustración en horizontal: los dinos
// disponibles son pocos (ver public/dinos/) y sin eso el mismo personaje se
// vería calcado al pasar de una pantalla a otra.
const DinoDecorativo = ({ src, className = '', espejo = false }) => (
  <img
    src={src}
    alt=""
    aria-hidden="true"
    className={`pointer-events-none select-none ${espejo ? 'scale-x-[-1] ' : ''}${className}`}
  />
);

export default DinoDecorativo;
