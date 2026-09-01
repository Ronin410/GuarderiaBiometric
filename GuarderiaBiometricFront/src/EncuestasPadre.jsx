import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import { ChevronLeft, ClipboardCheck, Send, Loader2, CheckCircle2 } from 'lucide-react';
import { mostrarError, mostrarExito } from './utils/alertas';

// EncuestasPadre -- vista de pantalla completa (mismo patrón que ChatPadre:
// swap desde DashboardPadre) para responder las "Encuestas para familias"
// del PDF de referencia.
const EncuestasPadre = ({ onVolver }) => {
  const [encuestas, setEncuestas] = useState([]);
  const [loading, setLoading] = useState(true);
  const [respuestas, setRespuestas] = useState({}); // { [encuestaId]: { [preguntaId]: valor } }
  const [enviando, setEnviando] = useState(null); // id de la encuesta que se está enviando

  const cargar = async () => {
    setLoading(true);
    try {
      const res = await api.get('/padre/encuestas');
      setEncuestas(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar encuestas:', err);
      mostrarError('No se pudieron cargar las encuestas');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const setRespuesta = (encuestaId, preguntaId, valor) => {
    setRespuestas((prev) => ({
      ...prev,
      [encuestaId]: { ...(prev[encuestaId] || {}), [preguntaId]: valor },
    }));
  };

  const enviar = async (encuesta) => {
    const propias = respuestas[encuesta.id] || {};
    const cuerpo = encuesta.preguntas
      .map((p) => ({ pregunta_id: p.id, respuesta: (propias[p.id] || '').trim() }))
      .filter((r) => r.respuesta);

    if (cuerpo.length === 0) {
      mostrarError('Responde al menos una pregunta');
      return;
    }

    setEnviando(encuesta.id);
    try {
      await api.post(`/padre/encuestas/${encuesta.id}/respuestas`, { respuestas: cuerpo });
      mostrarExito('¡Gracias por responder!');
      cargar();
    } catch (err) {
      console.error('Error al enviar respuestas:', err);
      mostrarError(err.response?.data?.error || 'No se pudieron enviar las respuestas');
    } finally {
      setEnviando(null);
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
          <h2 className="text-2xl font-black text-slate-900 uppercase tracking-tighter">Encuestas</h2>
          <div className="bg-brand-600 p-3 rounded-2xl text-white shadow-lg shadow-brand-200">
            <ClipboardCheck size={20} />
          </div>
        </div>
      </div>

      <div className="max-w-md mx-auto p-4 space-y-4">
        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : encuestas.length === 0 ? (
          <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center">
            <ClipboardCheck size={40} className="mx-auto text-slate-200 mb-4" />
            <p className="text-slate-400 font-bold uppercase text-[10px]">No hay encuestas activas por ahora</p>
          </div>
        ) : (
          encuestas.map((enc) => (
            <div key={enc.id} className="bg-white rounded-[2rem] p-6 shadow-sm border border-slate-100 space-y-4">
              <div>
                <p className="font-black text-slate-900 uppercase text-sm">{enc.titulo}</p>
                {enc.descripcion && <p className="text-xs text-slate-500 font-medium mt-1">{enc.descripcion}</p>}
              </div>

              {enc.ya_respondida && (
                <div className="flex items-center gap-2 text-emerald-600 bg-emerald-50 border border-emerald-100 p-3 rounded-xl">
                  <CheckCircle2 size={16} /> <span className="text-xs font-black uppercase">Ya respondiste, ¡gracias!</span>
                </div>
              )}

              {/* Una vez respondida se deja el mismo formulario pero
                  deshabilitado y con lo que el papá puso (respuesta_padre
                  del backend), en vez de ocultarlo -- así puede ver qué
                  contestó sin poder cambiarlo. */}
              <div className="space-y-4">
                {enc.preguntas.map((p) => (
                  <div key={p.id}>
                    <p className="text-xs font-bold text-slate-700 mb-2">{p.texto}</p>
                    {p.tipo === 'opcion_multiple' ? (
                      <div className="space-y-1.5">
                        {(p.opciones || []).map((opcion) => (
                          <label
                            key={opcion}
                            className={`flex items-center gap-2 text-xs font-medium border p-2.5 rounded-xl ${
                              enc.ya_respondida
                                ? 'bg-slate-100 border-slate-100 text-slate-400 cursor-not-allowed'
                                : 'bg-slate-50 border-slate-100 text-slate-600 cursor-pointer'
                            }`}
                          >
                            <input
                              type="radio" name={`pregunta-${p.id}`} value={opcion}
                              checked={enc.ya_respondida ? p.respuesta_padre === opcion : (respuestas[enc.id] || {})[p.id] === opcion}
                              disabled={enc.ya_respondida}
                              onChange={() => setRespuesta(enc.id, p.id, opcion)}
                              className="w-4 h-4 accent-brand-600 disabled:opacity-50"
                            />
                            {opcion}
                          </label>
                        ))}
                      </div>
                    ) : (
                      <textarea
                        rows={2}
                        value={enc.ya_respondida ? (p.respuesta_padre || '') : (respuestas[enc.id] || {})[p.id] || ''}
                        disabled={enc.ya_respondida}
                        onChange={(e) => setRespuesta(enc.id, p.id, e.target.value)}
                        placeholder="Tu respuesta..."
                        className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-medium resize-y disabled:opacity-60 disabled:cursor-not-allowed"
                      />
                    )}
                  </div>
                ))}
              </div>

              {!enc.ya_respondida && (
                <button
                  onClick={() => enviar(enc)}
                  disabled={enviando === enc.id}
                  className="w-full flex items-center justify-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-3 rounded-xl shadow-md transition-all active:scale-95"
                >
                  {enviando === enc.id ? <Loader2 className="animate-spin" size={16} /> : <Send size={16} />} Enviar respuestas
                </button>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default EncuestasPadre;
