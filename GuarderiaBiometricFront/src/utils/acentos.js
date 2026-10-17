// acentos.js -- de qué color es cada área de la plataforma.
//
// La navegación ya venía agrupada por áreas (Alumnos, Día a día,
// Administración, Sistema); esto le pone a cada una uno de los colores de
// los dinosaurios de la marca y lo deja en UN solo lugar, para que el
// sidebar y el panel que se abre siempre coincidan. Antes el acento vivía
// escrito a mano en el arreglo del sidebar de App.jsx, y el contenido era
// teal sin importar en qué sección estuvieras.
//
// Las clases van escritas completas y no armadas por interpolación:
// Tailwind v4 escanea el código en busca de nombres literales, y algo como
// `bg-dino-${tono}-suave` no llega a generarse nunca.

export const ACENTOS = {
  teal: {
    fondo: 'bg-brand-100',
    fondoSuave: 'bg-brand-50',
    texto: 'text-brand-600',
    // Relleno fuerte, para barras y subrayados.
    solido: 'bg-brand-600',
    borde: 'border-brand-200',
    // Versión clara: los tonos fuertes no contrastan sobre el sidebar
    // navy (--color-forest).
    claro: 'text-brand-300',
    dino: '/dinos/logo-pasitos.png',
  },
  verde: {
    fondo: 'bg-dino-verde-suave',
    fondoSuave: 'bg-dino-verde-suave',
    texto: 'text-dino-verde',
    solido: 'bg-dino-verde',
    borde: 'border-dino-verde/25',
    claro: 'text-dino-verde-claro',
    dino: '/dinos/dino-verde.png',
  },
  naranja: {
    fondo: 'bg-dino-naranja-suave',
    fondoSuave: 'bg-dino-naranja-suave',
    texto: 'text-dino-naranja',
    solido: 'bg-dino-naranja',
    borde: 'border-dino-naranja/25',
    claro: 'text-dino-naranja-claro',
    dino: '/dinos/dino-naranja.png',
  },
  morado: {
    fondo: 'bg-dino-morado-suave',
    fondoSuave: 'bg-dino-morado-suave',
    texto: 'text-dino-morado',
    solido: 'bg-dino-morado',
    borde: 'border-dino-morado/25',
    claro: 'text-dino-morado-claro',
    dino: '/dinos/dino-morado.png',
  },
  amarillo: {
    fondo: 'bg-dino-amarillo-suave',
    fondoSuave: 'bg-dino-amarillo-suave',
    texto: 'text-dino-amarillo',
    solido: 'bg-dino-amarillo',
    borde: 'border-dino-amarillo/30',
    claro: 'text-dino-amarillo-claro',
    dino: '/dinos/dino-amarillo.png',
  },
};

// Qué área le toca a cada tab. Mismos grupos que seccionesNav en App.jsx:
// si algún día se mueve un tab de sección, hay que moverlo también aquí
// para que el color del menú y el del contenido no se separen.
export const AREA_DE_ACENTO = {
  identificar: 'teal',
  registrar: 'teal',

  admin: 'verde',
  perfiles: 'verde',

  bitacora: 'naranja',
  menu: 'naranja',
  circulares: 'naranja',
  chat: 'naranja',
  ausencias: 'naranja',
  calendario: 'naranja',
  comedor: 'naranja',
  encuestas: 'naranja',

  reportes: 'amarillo',
  pagos: 'amarillo',
  estadisticas: 'amarillo',

  configuracion: 'morado',
  personal: 'morado',
  horarios: 'morado',
};

export const acentoDeTab = (tab) => ACENTOS[AREA_DE_ACENTO[tab]] ?? ACENTOS.teal;
