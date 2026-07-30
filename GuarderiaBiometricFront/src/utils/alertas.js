import Swal from 'sweetalert2';
import withReactContent from 'sweetalert2-react-content';

// Se exporta también la instancia base, para las pantallas que necesitan un
// Swal.fire() con contenido JSX o colores dinámicos propios (ej. VistaBitacora.jsx),
// en vez de crear su propio withReactContent(Swal) por separado en cada archivo.
export const MySwal = withReactContent(Swal);

const COLOR_CONFIRMAR = '#7c3aed'; // violet-600, mismo acento del resto de la app
const COLOR_CANCELAR = '#64748b'; // slate-500

export function mostrarExito(mensaje, titulo = '¡Listo!') {
  return MySwal.fire({
    icon: 'success',
    title: titulo,
    text: mensaje,
    confirmButtonColor: COLOR_CONFIRMAR,
  });
}

export function mostrarError(mensaje, titulo = 'Ups...') {
  return MySwal.fire({
    icon: 'error',
    title: titulo,
    text: mensaje,
    confirmButtonColor: COLOR_CONFIRMAR,
  });
}

export function mostrarAviso(mensaje, titulo = 'Aviso') {
  return MySwal.fire({
    icon: 'warning',
    title: titulo,
    text: mensaje,
    confirmButtonColor: COLOR_CONFIRMAR,
  });
}

// confirmar reemplaza a window.confirm(): devuelve true/false según lo que elija el usuario.
export async function confirmar(mensaje, titulo = '¿Estás seguro?') {
  const result = await MySwal.fire({
    icon: 'question',
    title: titulo,
    text: mensaje,
    showCancelButton: true,
    confirmButtonText: 'Sí, continuar',
    cancelButtonText: 'Cancelar',
    confirmButtonColor: COLOR_CONFIRMAR,
    cancelButtonColor: COLOR_CANCELAR,
  });
  return result.isConfirmed;
}
