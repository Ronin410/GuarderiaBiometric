import React, { useEffect, useState } from 'react';
import { Smartphone, X, Copy, Check } from 'lucide-react';
import { obtenerPromptDiferido, suscribirsePrompt, detectarPlataforma } from '../utils/pwaInstall';

const CLAVE_CERRADO = 'pasitos_instalar_cerrado';

// Ícono de "Compartir" de iOS (cuadrado con flecha hacia arriba) dibujado a
// mano en vez de un ícono genérico de lucide -- el punto es que la persona
// lo reconozca de un vistazo como EL MISMO ícono que va a ver abajo en
// Safari, no uno parecido.
// size no es un atributo real de <svg> (a diferencia de los íconos de
// lucide-react, que sí lo traducen a width/height internamente) -- sin
// mapearlo a mano, el SVG queda sin tamaño explícito y el navegador lo
// renderiza gigante por default.
const IconoCompartirIOS = ({ size = 24, ...props }) => (
  <svg viewBox="0 0 24 24" width={size} height={size} fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
    <path d="M12 15V3" /><path d="m7 8 5-5 5 5" />
    <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-7" />
  </svg>
);

// Ícono de "Agregar a pantalla de inicio" (un + dentro de un cuadrado) --
// mismo criterio: reconocible contra el renglón real del menú de Safari.
const IconoAgregarInicio = ({ size = 24, ...props }) => (
  <svg viewBox="0 0 24 24" width={size} height={size} fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
    <rect x="3" y="3" width="18" height="18" rx="4" /><path d="M12 8v8M8 12h8" />
  </svg>
);

// Ícono del menú de Chrome (tres puntitos verticales) -- mismo criterio que
// los de iOS: que se reconozca contra el botón real del navegador.
const IconoMenuAndroid = ({ size = 24, ...props }) => (
  <svg viewBox="0 0 24 24" width={size} height={size} fill="currentColor" {...props}>
    <circle cx="12" cy="5" r="2" /><circle cx="12" cy="12" r="2" /><circle cx="12" cy="19" r="2" />
  </svg>
);

// InstalarApp se monta en el portal del papá, en el panel de staff/admin y en
// la pantalla de acceso (el kiosco de la tablet es justo donde más conviene
// tenerla instalada) -- es la
// audiencia pensada para esto: muchos no son usuarios técnicos, así que la
// meta es que instalar la app sea "un toque" en Android (sí se puede, vía
// beforeinstallprompt) y, en iPhone, donde Apple NO permite automatizarlo
// para nada, los pasos más simples y visuales posibles en vez de mandarlos
// a "Configuración" o explicaciones en texto corrido.
//
// En Android hay un tercer caso: beforeinstallprompt no siempre llega (el
// navegador no es Chrome, o Chrome decidió no dispararlo en esa carga). Antes
// eso dejaba la franja invisible y sin ninguna explicación; ahora se cae a
// los pasos manuales del menú del navegador, que es lo mismo que hace el
// botón cuando sí está disponible.
const InstalarApp = () => {
  const [promptDiferido, setPromptDiferido] = useState(obtenerPromptDiferido());
  const [plataforma] = useState(detectarPlataforma);
  const [cerrado, setCerrado] = useState(() => sessionStorage.getItem(CLAVE_CERRADO) === '1');
  // null | 'ios' | 'android' -- qué pasos manuales se están mostrando.
  const [pasosVisibles, setPasosVisibles] = useState(null);
  const [linkCopiado, setLinkCopiado] = useState(false);

  useEffect(() => suscribirsePrompt(setPromptDiferido), []);

  const cerrar = () => {
    sessionStorage.setItem(CLAVE_CERRADO, '1');
    setCerrado(true);
  };

  const instalarAndroid = async () => {
    if (!promptDiferido) return;
    promptDiferido.prompt();
    await promptDiferido.userChoice;
    // Se puede usar una sola vez -- una vez mostrado (lo haya aceptado o
    // no), Chrome no lo vuelve a disparar hasta la próxima carga de página.
    setPromptDiferido(null);
  };

  const copiarLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.origin);
      setLinkCopiado(true);
      setTimeout(() => setLinkCopiado(false), 2500);
    } catch {
      // Sin permiso de portapapeles (raro, pero pasa en algunos navegadores
      // empotrados) -- no hay mucho más que ofrecer aquí que el link visible.
    }
  };

  if (plataforma.yaInstalada || cerrado) return null;
  if (!plataforma.esNavegadorEnApp && !promptDiferido && !plataforma.esIOS && !plataforma.esAndroid) return null;

  return (
    <>
      <div className="bg-forest text-white p-4 rounded-[2rem] flex items-center gap-4 relative">
        <button onClick={cerrar} className="absolute top-3 right-3 text-white/40 hover:text-white p-1" title="Cerrar">
          <X size={16} />
        </button>
        <div className="bg-brand-600 p-3 rounded-2xl shrink-0"><Smartphone size={22} /></div>
        <div className="flex-1 pr-4">
          {plataforma.esNavegadorEnApp ? (
            <>
              <p className="font-black text-sm leading-tight">Para instalarla, ábrela en tu navegador</p>
              <p className="text-white/60 text-xs mt-1 mb-3">Estás dentro de otra app (como WhatsApp) -- copia el link y ábrelo en Safari o Chrome.</p>
              <button onClick={copiarLink} className="bg-brand-600 hover:bg-brand-700 text-white text-xs font-black uppercase px-4 py-2.5 rounded-xl flex items-center gap-2 transition-all active:scale-95">
                {linkCopiado ? <><Check size={14} /> Copiado</> : <><Copy size={14} /> Copiar link</>}
              </button>
            </>
          ) : promptDiferido ? (
            <>
              <p className="font-black text-sm leading-tight">Instala Pasitos en tu teléfono</p>
              <p className="text-white/60 text-xs mt-1 mb-3">Entra más rápido, sin buscar el link cada vez.</p>
              <button onClick={instalarAndroid} className="bg-brand-600 hover:bg-brand-700 text-white text-xs font-black uppercase px-4 py-2.5 rounded-xl transition-all active:scale-95">
                Instalar ahora
              </button>
            </>
          ) : plataforma.esIOS ? (
            <>
              <p className="font-black text-sm leading-tight">Instala Pasitos en tu iPhone</p>
              <p className="text-white/60 text-xs mt-1 mb-3">Entra más rápido, sin buscar el link cada vez.</p>
              <button onClick={() => setPasosVisibles('ios')} className="bg-brand-600 hover:bg-brand-700 text-white text-xs font-black uppercase px-4 py-2.5 rounded-xl transition-all active:scale-95">
                Ver cómo (2 pasos)
              </button>
            </>
          ) : (
            <>
              <p className="font-black text-sm leading-tight">Instala Pasitos en tu teléfono</p>
              <p className="text-white/60 text-xs mt-1 mb-3">Entra más rápido, sin buscar el link cada vez.</p>
              <button onClick={() => setPasosVisibles('android')} className="bg-brand-600 hover:bg-brand-700 text-white text-xs font-black uppercase px-4 py-2.5 rounded-xl transition-all active:scale-95">
                Ver cómo (2 pasos)
              </button>
            </>
          )}
        </div>
      </div>

      {pasosVisibles && (
        <div className="fixed inset-0 z-[300] flex items-end sm:items-center justify-center p-4 bg-slate-900/70 backdrop-blur-sm" onClick={() => setPasosVisibles(null)}>
          <div className="bg-white rounded-[2rem] w-full max-w-sm p-6 shadow-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-black text-slate-900">
                {pasosVisibles === 'ios' ? 'Instalar en tu iPhone' : 'Instalar en tu teléfono'}
              </h3>
              <button onClick={() => setPasosVisibles(null)} className="text-slate-400 hover:text-slate-600 p-1"><X size={22} /></button>
            </div>

            <div className="space-y-4">
              {(pasosVisibles === 'ios'
                ? [
                    { Icono: IconoCompartirIOS, color: 'text-blue-500', texto: 'Toca el ícono de Compartir, abajo en la pantalla' },
                    { Icono: IconoAgregarInicio, color: 'text-slate-500', texto: 'Busca y toca "Agregar a inicio"' },
                  ]
                : [
                    { Icono: IconoMenuAndroid, color: 'text-slate-500', texto: 'Toca el menú del navegador, los tres puntitos de la esquina' },
                    { Icono: IconoAgregarInicio, color: 'text-slate-500', texto: 'Busca y toca "Instalar aplicación" o "Agregar a pantalla principal"' },
                  ]
              ).map((paso, i) => (
                <div key={i} className="flex items-center gap-4 bg-slate-50 rounded-2xl p-4">
                  <div className="bg-white text-brand-600 w-11 h-11 rounded-2xl flex items-center justify-center font-black text-lg shrink-0 shadow-sm">{i + 1}</div>
                  <paso.Icono size={30} className={`${paso.color} shrink-0`} />
                  <span className="text-sm font-bold text-slate-700 leading-snug">{paso.texto}</span>
                </div>
              ))}
            </div>

            <p className="text-center text-xs text-slate-400 font-bold mt-6">
              {pasosVisibles === 'ios'
                ? 'Listo -- aparecerá en tu pantalla de inicio como una app.'
                : 'Listo -- aparecerá en tu pantalla de inicio como una app.'}
            </p>
          </div>
        </div>
      )}
    </>
  );
};

export default InstalarApp;
