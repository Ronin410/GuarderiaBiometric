import React, { useState, useRef, useEffect } from 'react';
import jsQR from 'jsqr';
import Webcam from 'react-webcam';
import api from './axiosConfig';
import { ChevronLeft, QrCode, RefreshCw, AlertCircle, CheckCircle2, LogIn, LogOut } from 'lucide-react';
import { mostrarError } from './utils/alertas';
import { ACENTOS } from './utils/acentos';
import DinoDecorativo from './components/DinoDecorativo';

// Mismo acento que Identificar/Registrar/Código de acceso: esto es, en el
// fondo, otra forma de identificarse en el kiosco.
const acento = ACENTOS.teal;

const videoConstraints = { facingMode: 'environment' };

// EscanearQRKioscoPadre -- "que lleguen, escaneen el código QR desde su
// aplicación y le den entrada a su hijo desde ahí". El QR lo genera la
// TABLET (ver el modo "Autoservicio" de Recepción en App.jsx) y es
// anónimo -- no identifica a nadie. Al escanearlo con la cámara del propio
// celular, el backend regresa los hijos de ESTE papá (nunca los de otro,
// eso sale de su sesión) y desde aquí elige a cuál confirmarle entrada o
// salida, sin pasar por el kiosco ni esperar a que lo atiendan.
const EscanearQRKioscoPadre = ({ onVolver }) => {
  const webcamRef = useRef(null);
  const canvasRef = useRef(null);
  const escaneandoRef = useRef(false);

  const [camaraLista, setCamaraLista] = useState(false);
  const [camaraError, setCamaraError] = useState(false);
  const [verificando, setVerificando] = useState(false);
  const [errorCodigo, setErrorCodigo] = useState('');
  const [resultado, setResultado] = useState(null); // { codigo_kiosco_id, hijos: [...] }
  const [confirmando, setConfirmando] = useState(null); // hijo_id en curso
  const [confirmados, setConfirmados] = useState([]); // hijo_ids ya confirmados en esta sesión de escaneo

  // Escanea un frame cada 400ms mientras no haya un resultado ya mostrado
  // en pantalla -- igual que el modo "Código" del kiosco (App.jsx), pero
  // aquí apuntando a la tablet en vez de al teléfono de alguien más.
  useEffect(() => {
    if (resultado) return undefined;

    const intervalo = setInterval(() => {
      if (escaneandoRef.current) return;
      const video = webcamRef.current?.video;
      const canvas = canvasRef.current;
      if (!video || !canvas || video.readyState !== video.HAVE_ENOUGH_DATA) return;

      const w = video.videoWidth;
      const h = video.videoHeight;
      if (!w || !h) return;

      canvas.width = w;
      canvas.height = h;
      const ctx = canvas.getContext('2d');
      ctx.drawImage(video, 0, 0, w, h);
      let datos;
      try {
        datos = ctx.getImageData(0, 0, w, h);
      } catch {
        return;
      }

      const leido = jsQR(datos.data, w, h);
      if (!leido?.data) return;

      escaneandoRef.current = true;
      canjear(leido.data).finally(() => {
        setTimeout(() => { escaneandoRef.current = false; }, 2000);
      });
    }, 400);

    return () => clearInterval(intervalo);
  }, [resultado]);

  const canjear = async (token) => {
    setVerificando(true);
    setErrorCodigo('');
    try {
      const res = await api.post('/padre/codigo-kiosco/canjear', { token });
      setResultado(res.data);
      setConfirmados([]);
    } catch (err) {
      setErrorCodigo(err.response?.data?.mensaje || err.response?.data?.error || 'Código no válido');
    } finally {
      setVerificando(false);
    }
  };

  const confirmar = async (hijo) => {
    const hijoId = hijo.id || hijo.hijo_id;
    setConfirmando(hijoId);
    try {
      await api.post('/padre/confirmar-asistencia', {
        hijo_id: hijoId,
        codigo_kiosco_id: resultado.codigo_kiosco_id,
      });
      setConfirmados((prev) => [...prev, hijoId]);
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo confirmar');
    } finally {
      setConfirmando(null);
    }
  };

  const volverAEscanear = () => {
    setResultado(null);
    setErrorCodigo('');
    setConfirmados([]);
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
            <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter">Entrada rápida</h2>
            <DinoDecorativo src="/dinos/dino-verde-saludo.png" className="h-11 w-auto shrink-0" espejo />
          </div>
          <div className={`${acento.solido} p-3 rounded-2xl text-white shadow-lg`}>
            <QrCode size={20} />
          </div>
        </div>
        <p className="text-xs text-slate-500 font-medium mt-3 leading-relaxed">
          Escanea el QR que está en pantalla en la tablet de Recepción para darle entrada o salida a tu
          hijo directo desde aquí, sin esperar turno.
        </p>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-4">
        {!resultado ? (
          <>
            <div className="relative rounded-[2.5rem] overflow-hidden border-4 border-white bg-slate-200 shadow-xl aspect-square max-w-xs mx-auto">
              <Webcam
                ref={webcamRef}
                audio={false}
                videoConstraints={videoConstraints}
                className="absolute inset-0 w-full h-full object-cover"
                onUserMedia={() => { setCamaraLista(true); setCamaraError(false); }}
                onUserMediaError={(err) => { console.error('No se pudo activar la cámara:', err); setCamaraError(true); }}
              />
              <canvas ref={canvasRef} className="hidden" />
              {!camaraLista && (
                <div className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-slate-800/90 px-6 text-center">
                  {camaraError ? (
                    <>
                      <AlertCircle className="text-rose-300" size={28} />
                      <p className="text-white text-[10px] font-black uppercase tracking-widest">No se pudo activar la cámara</p>
                    </>
                  ) : (
                    <>
                      <RefreshCw className="animate-spin text-white" size={28} />
                      <p className="text-white text-[9px] font-black uppercase tracking-widest">Iniciando cámara...</p>
                    </>
                  )}
                </div>
              )}
              {camaraLista && (
                <div className="absolute inset-x-0 bottom-0 bg-slate-900/60 py-2 text-center">
                  <p className="text-white text-[9px] font-black uppercase tracking-widest">
                    {verificando ? 'Verificando...' : 'Apunta al QR de la tablet'}
                  </p>
                </div>
              )}
            </div>

            {errorCodigo && (
              <div className="bg-rose-50 border border-rose-200 rounded-2xl p-4 text-center">
                <p className="text-rose-600 text-xs font-bold">{errorCodigo}</p>
              </div>
            )}
          </>
        ) : (
          <div className="space-y-3">
            <div className="bg-emerald-50 border border-emerald-200 rounded-2xl p-4 flex items-center gap-3">
              <CheckCircle2 className="text-emerald-600 shrink-0" size={20} />
              <p className="text-emerald-700 text-xs font-bold">Código verificado. Elige a quién le das entrada o salida.</p>
            </div>

            {resultado.hijos.map((h) => {
              const hijoId = h.id || h.hijo_id;
              const yaConfirmado = confirmados.includes(hijoId);
              const esSalida = h.ultimo_estado === 'ENTRADA';
              return (
                <div key={hijoId} className="bg-white p-5 rounded-2xl border border-slate-100 shadow-sm flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="font-black text-sm text-slate-900 truncate">{h.nombre_niño || h.nombre}</p>
                    <p className="text-[10px] text-slate-400 font-bold uppercase">
                      {h.ultimo_estado === 'AUSENTE' ? 'Sin registro hoy' : `Último: ${h.ultimo_estado}`}
                    </p>
                  </div>
                  {yaConfirmado ? (
                    <span className="flex items-center gap-1.5 text-[10px] font-black text-emerald-600 uppercase shrink-0">
                      <CheckCircle2 size={14} /> Listo
                    </span>
                  ) : (
                    <button
                      onClick={() => confirmar(h)}
                      disabled={confirmando === hijoId}
                      className={`shrink-0 flex items-center gap-1.5 px-4 py-2.5 rounded-xl text-[11px] font-black uppercase transition-all active:scale-95 disabled:opacity-50 ${
                        esSalida ? 'bg-amber-500 text-white' : 'bg-emerald-500 text-white'
                      }`}
                    >
                      {esSalida ? <LogOut size={14} /> : <LogIn size={14} />}
                      {confirmando === hijoId ? '...' : esSalida ? 'Salida' : 'Entrada'}
                    </button>
                  )}
                </div>
              );
            })}

            <button
              onClick={volverAEscanear}
              className="w-full py-3.5 text-slate-400 hover:text-slate-600 font-black uppercase text-[11px] tracking-widest transition-colors"
            >
              Escanear otro código
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default EscanearQRKioscoPadre;
