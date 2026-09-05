// Mismos tokens de color que GuarderiaBiometricFront/src/index.css
// (--color-brand-*, --color-forest, --color-paper, --color-ink) más la
// paleta slate/emerald/rose/amber estándar de Tailwind que ya se usa en
// todo el frontend web -- se repiten aquí a mano porque React Native no
// lee el CSS del proyecto web, pero son los mismos valores exactos para
// que la app se vea como la misma marca.
export const color = {
  brand50: '#eefafa',
  brand100: '#d3f2f2',
  brand200: '#a9e6e7',
  brand300: '#74d5d7',
  brand400: '#3cc0c3',
  brand500: '#18a8a8',
  brand600: '#0f9294',
  brand700: '#0c777b',
  brand800: '#0a5f66',
  brand900: '#084850',

  forest: '#084850',
  forestLight: '#0c5a63',
  forestDark: '#05323a',
  paper: '#f8fcfc',
  ink: '#0d2a2e',

  // Acentos de los dinosaurios de la marca (mismos valores que
  // --color-dino-* en GuarderiaBiometricFront/src/index.css).
  dinoVerde: '#6f9134',
  dinoVerdeSuave: '#f2faf0',
  dinoNaranja: '#d47a0b',
  dinoNaranjaSuave: '#fff6ea',
  dinoMorado: '#6f6486',
  dinoMoradoSuave: '#f6f3fb',
  dinoAmarillo: '#c79511',
  dinoAmarilloSuave: '#fff9e6',

  slate50: '#f8fafc',
  slate100: '#f1f5f9',
  slate200: '#e2e8f0',
  slate300: '#cbd5e1',
  slate400: '#94a3b8',
  slate500: '#64748b',
  slate700: '#334155',
  slate800: '#1e293b',
  slate900: '#0f172a',

  emerald50: '#ecfdf5',
  emerald100: '#d1fae5',
  emerald200: '#a7f3d0',
  emerald500: '#10b981',
  emerald600: '#059669',
  emerald700: '#047857',

  amber50: '#fffbeb',
  amber100: '#fef3c7',
  amber500: '#f59e0b',
  amber600: '#d97706',
  amber700: '#b45309',

  rose50: '#fff1f2',
  rose100: '#ffe4e6',
  rose500: '#f43f5e',
  rose600: '#e11d48',
  rose700: '#be123c',

  white: '#ffffff',
};

// Radios grandes y sombras suaves -- mismo lenguaje visual del web
// (rounded-[2rem]/[2.5rem], shadow-sm/md/xl de Tailwind).
export const radius = { sm: 16, md: 24, lg: 32, xl: 40, full: 999 };

export const sombra = {
  sm: { shadowColor: '#000', shadowOpacity: 0.05, shadowRadius: 6, shadowOffset: { width: 0, height: 2 }, elevation: 2 },
  md: { shadowColor: '#000', shadowOpacity: 0.08, shadowRadius: 12, shadowOffset: { width: 0, height: 4 }, elevation: 4 },
};
