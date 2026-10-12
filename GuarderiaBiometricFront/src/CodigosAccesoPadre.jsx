import React, { useState, useEffect } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import api from './axiosConfig';
import { ChevronLeft, QrCode, Check, X, Ban, Clock } from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import { ACENTOS } from './utils/acentos';
import DinoDecorativo from './components/DinoDecorativo';

// Mismo acento que Identificar/Registrar en el panel de staff (utils/acentos.js):
// este es, en el fondo, otra forma de identificarse en el kiosco.
const acento = ACENTOS.teal;

const OPCIONES_DIAS = [
  { dias: 1, etiqueta: '1 día' },
  { dias: 3, etiqueta: '3 días' },
  { dias: 7, etiqueta: '7 días' },
];

const formatoFecha = (iso) => {
  try {
    return new Date(iso).toLocaleDateString('es-MX', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
  } catch {
    return iso;
  }
};

// estadoDeCodigo -- un código vive en uno de tres estados, y los tres se
// ven distinto en la lista: activo (se puede revocar), revocado (lo cortó
// el propio papá) o vencido (se le acabó el tiempo solo). No son
// excluyentes en la base (revocado_en puede estar lleno Y ya haber vencido
// también), así que revocado gana si los dos aplican -- es la acción que
// alguien tomó a propósito.
const estadoDeCodigo = (codigo) => {
  if (codigo.revocado_en) return 'revocado';
  if (new Date(codigo.expira_en) < new Date()) return 'vencido';
  return 'activo';
};

const ESTADO_INFO = {
  activo: { texto: 'Activo', clase: 'bg-emerald-50 text-emerald-700 border-emerald-200', Icono: Check },
  revocado: { texto: 'Revocado', clase: 'bg-slate-100 text-slate-500 border-slate-200', Icono: Ban },
  vencido: { texto: 'Vencido', clase: 'bg-amber-50 text-amber-600 border-amber-200', Icono: Clock },
};

// CodigosAccesoPadre -- "una sección de generar QR" para dos casos: delegar
// una recogida a alguien sin rostro enrolado, O que el papá mismo entre por
// código en vez de rostro -- el facial sigue siendo obligatorio al
// registrarse (eso no cambia), pero el día a día puede ser molesto para
// quien no le gusta quedarse enfocando, o en sitios con poca luz donde la
// cámara del kiosco no alcanza a leer bien la cara. Mismo patrón de
// pantalla completa que CircularesPadre/EncuestasPadre (swap desde
// DashboardPadre, con onVolver).
const CodigosAccesoPadre = ({ onVolver, nombrePadre }) => {
  const [hijos, setHijos] = useState([]);
  const [codigos, setCodigos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [generando, setGenerando] = useState(false);
  const [codigoNuevo, setCodigoNuevo] = useState(null);

  const [hijosSeleccionados, setHijosSeleccionados] = useState([]);
  // paraMiMismo -- "ya va a ser decisión de él si pone el facial o si
  // quiere entrar por código QR": con esto activado se autoriza al propio
  // papá (persona_autorizada queda con su nombre) y se esconde el campo de
  // texto libre, que es para nombrar a alguien MÁS, no a uno mismo.
  const [paraMiMismo, setParaMiMismo] = useState(true);
  const [personaAutorizada, setPersonaAutorizada] = useState('');
  const [dias, setDias] = useState(3);

  const elegirParaMiMismo = (valor) => {
    setParaMiMismo(valor);
    setPersonaAutorizada(valor ? (nombrePadre || 'El tutor') : '');
  };

  const cargar = async () => {
    setLoading(true);
    try {
      const [resHijos, resCodigos] = await Promise.all([
        api.get('/padre/0/hijos'),
        api.get('/padre/codigos-acceso'),
      ]);
      setHijos((resHijos.data || []).map((h) => ({ id: h.id, nombre: h.nombre_niño || h.nombre || 'Sin nombre' })));
      setCodigos(resCodigos.data || []);
    } catch (err) {
      console.error('Error al cargar códigos de acceso:', err);
      mostrarError('No se pudieron cargar tus códigos de acceso');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    cargar();
    // Arranca en "para mí mismo" con el nombre ya puesto -- es el caso más
    // común (un papá generando su propio código), así que no debería
    // obligar a escribir nada para llegar ahí.
    setPersonaAutorizada(nombrePadre || 'El tutor');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const alternarHijo = (hijoId) => {
    setHijosSeleccionados((prev) =>
      prev.includes(hijoId) ? prev.filter((id) => id !== hijoId) : [...prev, hijoId]
    );
  };

  const generar = async (e) => {
    e.preventDefault();
    if (hijosSeleccionados.length === 0) {
      mostrarError('Elige al menos un niño');
      return;
    }
    if (personaAutorizada.trim().length < 2) {
      mostrarError('Escribe el nombre de quién va a usar este código');
      return;
    }
    setGenerando(true);
    try {
      const res = await api.post('/padre/codigos-acceso', {
        hijos_ids: hijosSeleccionados,
        persona_autorizada: personaAutorizada.trim(),
        dias,
      });
      setCodigoNuevo(res.data);
      setHijosSeleccionados([]);
      elegirParaMiMismo(true);
      await cargar();
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo generar el código');
    } finally {
      setGenerando(false);
    }
  };

  const revocar = async (codigo) => {
    const confirmado = await confirmar(
      `"${codigo.persona_autorizada}" ya no podrá usar este código para recoger a ${codigo.hijos || 'tu(s) hijo(s)'}.`,
      '¿Revocar este código?'
    );
    if (!confirmado) return;
    try {
      await api.delete(`/padre/codigos-acceso/${codigo.id}`);
      mostrarExito('El código quedó revocado');
      if (codigoNuevo?.id === codigo.id) setCodigoNuevo(null);
      await cargar();
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo revocar el código');
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
          <div className="flex items-center gap-3 min-w-0">
            <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter">Código de acceso</h2>
            <DinoDecorativo src="/dinos/dino-verde-saludo.png" className="h-11 w-auto shrink-0" espejo />
          </div>
          <div className={`${acento.solido} p-3 rounded-2xl text-white shadow-lg`}>
            <QrCode size={20} />
          </div>
        </div>
        <p className="text-xs text-slate-500 font-medium mt-3 leading-relaxed">
          Úsalo tú mismo cuando no quieras pasar por reconocimiento facial -- hay lugares donde la cámara
          no alcanza a leer bien la cara, o simplemente no te gusta quedarte enfocando -- o genera uno para
          alguien que tú autorices, sin que tenga que enrolar su rostro. Cualquiera que tenga el código
          puede usarlo mientras esté vigente, así que compártelo solo con la persona que autorizas.
        </p>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-6">
        {/* RESULTADO: el QR recién generado -- se queda visible hasta que
            el papá genere otro o salga de la pantalla, para que le dé
            tiempo de enseñarlo o mandarlo. */}
        {codigoNuevo && (
          <div className={`bg-white rounded-[2rem] p-6 shadow-sm border-2 ${acento.borde} text-center space-y-4`}>
            <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest">Código generado</p>
            <div className="flex justify-center">
              <div className="p-4 bg-white rounded-2xl border border-slate-100">
                <QRCodeSVG value={codigoNuevo.token} size={180} level="M" />
              </div>
            </div>
            <div>
              <p className="text-[9px] font-bold text-slate-400 uppercase tracking-widest mb-1">Código de respaldo</p>
              <p className="text-2xl font-black text-slate-900 tracking-[0.2em]">{codigoNuevo.codigo_corto}</p>
            </div>
            <p className="text-[10px] text-slate-400 font-bold uppercase tracking-wide">
              Válido hasta {formatoFecha(codigoNuevo.expira_en)}
            </p>
            <button
              onClick={() => setCodigoNuevo(null)}
              className="text-[10px] font-black text-slate-400 hover:text-slate-600 uppercase tracking-widest underline underline-offset-2"
            >
              Cerrar
            </button>
          </div>
        )}

        {/* FORMULARIO */}
        <form onSubmit={generar} className={`bg-white rounded-[2rem] p-6 shadow-sm border border-slate-100 space-y-5`}>
          <div>
            <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-3">¿Para qué niño(s)?</p>
            <div className="space-y-2">
              {hijos.map((h) => (
                <label
                  key={h.id}
                  className={`flex items-center gap-3 p-3.5 rounded-2xl border cursor-pointer transition-all ${
                    hijosSeleccionados.includes(h.id) ? `${acento.fondoSuave} ${acento.borde}` : 'bg-slate-50 border-slate-100'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={hijosSeleccionados.includes(h.id)}
                    onChange={() => alternarHijo(h.id)}
                    className="w-4 h-4 accent-brand-600"
                  />
                  <span className="text-sm font-bold text-slate-700">{h.nombre}</span>
                </label>
              ))}
              {hijos.length === 0 && (
                <p className="text-xs text-slate-400 font-medium">No se encontraron niños vinculados a tu cuenta.</p>
              )}
            </div>
          </div>

          <div>
            <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">¿Quién lo va a usar?</p>
            <div className="flex gap-2 mb-3">
              <button
                type="button"
                onClick={() => elegirParaMiMismo(true)}
                className={`flex-1 py-3 rounded-2xl text-xs font-black uppercase transition-all ${
                  paraMiMismo ? `${acento.solido} text-white shadow-md` : 'bg-slate-50 text-slate-500 border border-slate-200'
                }`}
              >
                Yo mismo
              </button>
              <button
                type="button"
                onClick={() => elegirParaMiMismo(false)}
                className={`flex-1 py-3 rounded-2xl text-xs font-black uppercase transition-all ${
                  !paraMiMismo ? `${acento.solido} text-white shadow-md` : 'bg-slate-50 text-slate-500 border border-slate-200'
                }`}
              >
                Otra persona
              </button>
            </div>
            {!paraMiMismo && (
              <input
                type="text"
                value={personaAutorizada}
                onChange={(e) => setPersonaAutorizada(e.target.value)}
                placeholder="Ej. Tía Lupita, mi suegro..."
                maxLength={100}
                autoFocus
                className="w-full bg-slate-50 border border-slate-200 rounded-2xl p-4 text-sm font-medium outline-none focus:border-brand-400"
              />
            )}
            <p className="text-[9px] text-slate-400 font-medium mt-1.5">
              {paraMiMismo
                ? 'Recepción va a ver tu nombre al momento de usar el código, igual que si te reconociera el rostro.'
                : 'Recepción va a ver este nombre al momento de usar el código, para poder confirmar con quién habla.'}
            </p>
          </div>

          <div>
            <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">¿Por cuánto tiempo?</p>
            <div className="flex gap-2">
              {OPCIONES_DIAS.map((op) => (
                <button
                  key={op.dias}
                  type="button"
                  onClick={() => setDias(op.dias)}
                  className={`flex-1 py-3 rounded-2xl text-xs font-black uppercase transition-all ${
                    dias === op.dias ? `${acento.solido} text-white shadow-md` : 'bg-slate-50 text-slate-500 border border-slate-200'
                  }`}
                >
                  {op.etiqueta}
                </button>
              ))}
            </div>
          </div>

          <button
            type="submit"
            disabled={generando}
            className={`w-full py-4 ${acento.solido} text-white font-black uppercase text-xs rounded-2xl shadow-lg active:scale-95 transition-all disabled:opacity-50 flex items-center justify-center gap-2`}
          >
            <QrCode size={16} /> {generando ? 'Generando...' : 'Generar código'}
          </button>
        </form>

        {/* HISTORIAL */}
        <div className="space-y-3">
          <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-widest ml-2">Tus códigos</h3>
          {loading ? (
            <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
          ) : codigos.length === 0 ? (
            <div className="bg-white p-8 rounded-[2rem] border border-dashed border-slate-200 text-center">
              <QrCode size={32} className="mx-auto text-slate-200 mb-3" />
              <p className="text-slate-400 font-bold uppercase text-[10px]">Todavía no has generado ningún código</p>
            </div>
          ) : (
            codigos.map((cod) => {
              const estado = estadoDeCodigo(cod);
              const info = ESTADO_INFO[estado];
              const Icono = info.Icono;
              return (
                <div key={cod.id} className="bg-white p-4 rounded-2xl border border-slate-100 flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="font-black text-sm text-slate-900 truncate">{cod.persona_autorizada}</p>
                    <p className="text-[10px] text-slate-400 font-bold uppercase truncate">{cod.hijos || 'Sin niños vinculados'}</p>
                    <p className="text-[9px] text-slate-400 font-medium mt-0.5">
                      {estado === 'activo' ? `Vence ${formatoFecha(cod.expira_en)}` : `Código: ${cod.codigo_corto}`}
                    </p>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span className={`flex items-center gap-1 text-[9px] font-black px-2.5 py-1.5 rounded-lg border uppercase ${info.clase}`}>
                      <Icono size={11} /> {info.texto}
                    </span>
                    {estado === 'activo' && (
                      <button
                        onClick={() => revocar(cod)}
                        className="p-2 text-slate-300 hover:text-rose-500 transition-colors"
                        title="Revocar"
                      >
                        <X size={16} />
                      </button>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};

export default CodigosAccesoPadre;
