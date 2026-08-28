import React, { useState, useEffect, useRef } from 'react';
import api from './axiosConfig';
import {
  ShieldCheck, Save, Loader2, Users, FileText,
  FolderOpen, Plus, Edit3, Check, X, Trash2, FileUp, ExternalLink,
} from 'lucide-react';
import { mostrarExito, mostrarError, confirmar } from './utils/alertas';

const PanelConfiguracion = () => {
  const [texto, setTexto] = useState('');
  const [version, setVersion] = useState('');
  const [configurado, setConfigurado] = useState(false);
  const [totalConsentimientos, setTotalConsentimientos] = useState(null);
  const [loading, setLoading] = useState(true);
  const [guardando, setGuardando] = useState(false);

  // "En el aviso de privacidad también podrán, en lugar de escribir el
  // aviso, poner un archivo PDF" -- son alternativas (el backend borra una
  // al guardar la otra, ver el comentario largo en handleActualizarAviso),
  // así que la interfaz también las presenta como un solo modo a la vez en
  // vez de dos formularios sueltos.
  const [modo, setModo] = useState('texto'); // 'texto' | 'pdf'
  const [pdfUrlActual, setPdfUrlActual] = useState(null);
  const [pdfNuevo, setPdfNuevo] = useState(null);
  const inputPdfRef = useRef(null);

  // Documentos requeridos: la MISMA plantilla para todos los niños de la
  // guardería -- se configura aquí, una sola vez, no niño por niño (ver el
  // comentario largo en tipos_documento.go). DocumentosNino.jsx (en cada
  // tarjeta de niño) solo LEE este catálogo para subir/ver archivos, ya no
  // lo edita.
  const [tipos, setTipos] = useState([]);
  const [loadingTipos, setLoadingTipos] = useState(true);
  const [nuevoTipoNombre, setNuevoTipoNombre] = useState('');
  const [creandoTipo, setCreandoTipo] = useState(false);
  const [editandoTipoId, setEditandoTipoId] = useState(null);
  const [nombreEdit, setNombreEdit] = useState('');

  const cargar = async () => {
    setLoading(true);
    try {
      const [resAviso, resEstadisticas] = await Promise.all([
        api.get('/aviso-privacidad'),
        api.get('/admin/aviso-privacidad/estadisticas'),
      ]);
      setTexto(resAviso.data.texto || '');
      setVersion(resAviso.data.version || '');
      setConfigurado(!!resAviso.data.configurado);
      setPdfUrlActual(resAviso.data.pdf_url || null);
      setModo(resAviso.data.pdf_url ? 'pdf' : 'texto');
      setTotalConsentimientos(resEstadisticas.data.total_consentimientos ?? 0);
    } catch (err) {
      console.error('Error al cargar la configuración de privacidad:', err);
      mostrarError('No se pudo cargar el Aviso de Privacidad');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const cargarTipos = async () => {
    setLoadingTipos(true);
    try {
      const res = await api.get('/admin/tipos-documento');
      setTipos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar tipos de documento:', err);
      mostrarError('No se pudieron cargar los tipos de documento');
    } finally {
      setLoadingTipos(false);
    }
  };

  useEffect(() => { cargarTipos(); }, []);

  const crearTipo = async () => {
    if (!nuevoTipoNombre.trim()) return;
    setCreandoTipo(true);
    try {
      await api.post('/admin/tipos-documento', { nombre: nuevoTipoNombre.trim() });
      setNuevoTipoNombre('');
      await cargarTipos();
    } catch (err) {
      console.error('Error al crear el tipo de documento:', err);
      mostrarError(err.response?.data?.error || 'No se pudo crear el tipo de documento');
    } finally {
      setCreandoTipo(false);
    }
  };

  const guardarNombreTipo = async (tipo) => {
    if (!nombreEdit.trim()) return;
    try {
      await api.put(`/admin/tipos-documento/${tipo.id}`, { nombre: nombreEdit.trim() });
      setEditandoTipoId(null);
      await cargarTipos();
    } catch (err) {
      console.error('Error al renombrar el tipo de documento:', err);
      mostrarError(err.response?.data?.error || 'No se pudo renombrar el tipo de documento');
    }
  };

  const eliminarTipo = async (tipo) => {
    const ok = await confirmar(`Se quitará "${tipo.nombre}" de lo que le pides a las familias.`, '¿Eliminar tipo de documento?');
    if (!ok) return;
    try {
      await api.delete(`/admin/tipos-documento/${tipo.id}`);
      mostrarExito('Tipo de documento eliminado');
      await cargarTipos();
    } catch (err) {
      console.error('Error al eliminar el tipo de documento:', err);
      mostrarError(err.response?.data?.error || 'No se pudo eliminar el tipo de documento');
    }
  };

  const elegirPdf = (file) => {
    if (!file) return;
    if (file.type !== 'application/pdf') {
      mostrarError('El archivo debe ser un PDF');
      return;
    }
    if (file.size > 10 * 1024 * 1024) {
      mostrarError('El PDF no puede pesar más de 10 MB');
      return;
    }
    setPdfNuevo(file);
  };

  const guardar = async () => {
    if (modo === 'texto' && !texto.trim()) {
      mostrarError('El texto del Aviso de Privacidad no puede quedar vacío');
      return;
    }
    if (modo === 'pdf' && !pdfNuevo) {
      mostrarError('Selecciona el PDF del Aviso de Privacidad');
      return;
    }
    setGuardando(true);
    try {
      const data = new FormData();
      if (modo === 'pdf') {
        data.append('pdf', pdfNuevo);
      } else {
        data.append('texto', texto.trim());
      }
      const res = await api.put('/admin/aviso-privacidad', data);
      mostrarExito(`Se guardó una nueva versión (${res.data.version}). Los tutores que ya habían aceptado una versión anterior no necesitan volver a hacerlo, pero cualquier registro nuevo verá este ${modo === 'pdf' ? 'PDF' : 'texto'}.`);
      setPdfNuevo(null);
      if (inputPdfRef.current) inputPdfRef.current.value = '';
      cargar();
    } catch (err) {
      console.error('Error al guardar el Aviso de Privacidad:', err);
      mostrarError(err.response?.data?.error || 'No se pudo guardar el Aviso de Privacidad');
    } finally {
      setGuardando(false);
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex items-center gap-4 mb-6">
          <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><ShieldCheck size={28} /></div>
          <div>
            <h3 className="text-xl font-black uppercase text-slate-900">Aviso de Privacidad</h3>
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Cumplimiento LFPDPPP</p>
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : (
          <div className="space-y-6">
            {!configurado && (
              <div className="bg-amber-50 border border-amber-200 rounded-2xl p-5 text-amber-800 text-sm font-bold">
                Mientras no guardes un texto o un PDF aquí, el kiosco no permitirá registrar nuevos
                tutores. Pide a tu asesor legal el Aviso de Privacidad para datos biométricos y de
                menores, y pégalo o súbelo abajo.
              </div>
            )}

            <div className="flex flex-wrap gap-4">
              <div className="bg-slate-50 border border-slate-100 rounded-2xl px-5 py-4 flex items-center gap-3">
                <FileText size={20} className="text-brand-500" />
                <div>
                  <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Versión vigente</p>
                  <p className="font-black text-slate-900">{version || 'Sin configurar'}</p>
                </div>
              </div>
              <div className="bg-slate-50 border border-slate-100 rounded-2xl px-5 py-4 flex items-center gap-3">
                <Users size={20} className="text-brand-500" />
                <div>
                  <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Consentimientos registrados</p>
                  <p className="font-black text-slate-900">{totalConsentimientos}</p>
                </div>
              </div>
            </div>

            {/* Texto y PDF son alternativas -- guardar uno reemplaza al otro
                (ver el comentario largo en handleActualizarAviso), así que
                la interfaz también obliga a elegir un solo modo a la vez. */}
            <div className="flex gap-2 bg-slate-100 p-1.5 rounded-2xl w-fit">
              <button
                onClick={() => setModo('texto')}
                className={`flex items-center gap-2 text-[10px] font-black uppercase px-4 py-2.5 rounded-xl transition-all ${modo === 'texto' ? 'bg-white text-brand-600 shadow-sm' : 'text-slate-400'}`}
              >
                <FileText size={14} /> Escribir texto
              </button>
              <button
                onClick={() => setModo('pdf')}
                className={`flex items-center gap-2 text-[10px] font-black uppercase px-4 py-2.5 rounded-xl transition-all ${modo === 'pdf' ? 'bg-white text-brand-600 shadow-sm' : 'text-slate-400'}`}
              >
                <FileUp size={14} /> Subir PDF
              </button>
            </div>

            {modo === 'texto' ? (
              <div className="space-y-2">
                <label className="text-[10px] font-black uppercase text-slate-400 ml-2 tracking-widest">
                  Texto del Aviso de Privacidad
                </label>
                <textarea
                  rows={14}
                  value={texto}
                  onChange={(e) => setTexto(e.target.value)}
                  placeholder="Pega aquí el texto completo del Aviso de Privacidad..."
                  className="w-full bg-slate-50 border border-slate-200 p-5 rounded-2xl outline-none focus:ring-2 focus:ring-brand-500 text-slate-900 font-medium resize-y"
                />
                <p className="text-[10px] text-slate-400 ml-2">
                  Al guardar se crea una nueva versión automáticamente. Los tutores lo verán completo,
                  con scroll, antes de registrar su rostro en el kiosco.
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                <label className="text-[10px] font-black uppercase text-slate-400 ml-2 tracking-widest">
                  PDF del Aviso de Privacidad
                </label>
                {pdfUrlActual && !pdfNuevo && (
                  <a
                    href={pdfUrlActual} target="_blank" rel="noreferrer"
                    className="flex items-center gap-2 bg-emerald-50 border border-emerald-100 text-emerald-700 text-sm font-bold px-5 py-4 rounded-2xl w-fit hover:bg-emerald-100 transition-all"
                  >
                    <ExternalLink size={16} /> Ver el PDF vigente ({version})
                  </a>
                )}
                <button
                  onClick={() => inputPdfRef.current?.click()}
                  className="flex items-center gap-2 bg-slate-50 border border-dashed border-slate-300 hover:border-brand-400 text-slate-500 text-sm font-bold px-5 py-4 rounded-2xl transition-all w-fit"
                >
                  <FileUp size={18} />
                  {pdfNuevo ? pdfNuevo.name : (pdfUrlActual ? 'Reemplazar PDF' : 'Elegir PDF')}
                </button>
                <input
                  ref={inputPdfRef}
                  type="file"
                  accept="application/pdf"
                  className="hidden"
                  onChange={(e) => elegirPdf(e.target.files?.[0])}
                />
                <p className="text-[10px] text-slate-400 ml-2">
                  Al guardar se crea una nueva versión automáticamente. Los tutores lo verán completo
                  antes de registrar su rostro en el kiosco.
                </p>
              </div>
            )}

            <button
              onClick={guardar}
              disabled={guardando}
              className="w-full sm:w-auto flex items-center justify-center gap-3 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase px-8 py-4 rounded-2xl shadow-lg transition-all active:scale-95"
            >
              {guardando ? <Loader2 className="animate-spin" size={20} /> : <Save size={20} />}
              {guardando ? 'Guardando...' : 'Guardar Aviso de Privacidad'}
            </button>
          </div>
        )}
      </div>

      {/* DOCUMENTOS REQUERIDOS -- una sola plantilla para toda la
          guardería, no niño por niño: se configura aquí, y cada tarjeta de
          niño (DocumentosNino.jsx en Perfiles) solo sube/ve archivos contra
          este mismo catálogo. */}
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl mt-8">
        <div className="flex items-center gap-4 mb-6">
          <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><FolderOpen size={28} /></div>
          <div>
            <h3 className="text-xl font-black uppercase text-slate-900">Documentos Requeridos</h3>
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Misma lista para todos los niños de la guardería</p>
          </div>
        </div>

        {loadingTipos ? (
          <div className="py-10 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : (
          <div className="space-y-4">
            {tipos.length === 0 ? (
              <div className="py-6 text-center text-slate-400 font-black uppercase tracking-widest text-[10px]">
                Todavía no configuras qué documentos pedirle a las familias
              </div>
            ) : (
              <div className="space-y-2">
                {tipos.map((t) => (
                  <div key={t.id} className="flex items-center gap-3 bg-slate-50 border border-slate-100 rounded-2xl px-4 py-3">
                    {editandoTipoId === t.id ? (
                      <>
                        <input
                          autoFocus
                          value={nombreEdit}
                          onChange={(e) => setNombreEdit(e.target.value)}
                          onKeyDown={(e) => e.key === 'Enter' && guardarNombreTipo(t)}
                          className="flex-1 bg-white border border-brand-300 rounded-xl px-3 py-2 text-sm font-bold outline-none"
                        />
                        <button onClick={() => guardarNombreTipo(t)} className="text-emerald-500 hover:text-emerald-600 p-2"><Check size={16} /></button>
                        <button onClick={() => setEditandoTipoId(null)} className="text-slate-400 hover:text-slate-600 p-2"><X size={16} /></button>
                      </>
                    ) : (
                      <>
                        <span className="flex-1 text-sm font-bold text-slate-700">{t.nombre}</span>
                        {t.en_uso > 0 && (
                          <span className="text-[9px] font-black text-slate-400 uppercase bg-white border border-slate-200 px-2 py-1 rounded-full">
                            {t.en_uso} niño{t.en_uso === 1 ? '' : 's'} ya subió
                          </span>
                        )}
                        <button onClick={() => { setEditandoTipoId(t.id); setNombreEdit(t.nombre); }} className="text-slate-300 hover:text-brand-600 p-2" title="Renombrar">
                          <Edit3 size={15} />
                        </button>
                        <button onClick={() => eliminarTipo(t)} className="text-slate-300 hover:text-rose-500 p-2" title="Eliminar tipo">
                          <Trash2 size={15} />
                        </button>
                      </>
                    )}
                  </div>
                ))}
              </div>
            )}

            <div className="flex items-center gap-3 pt-2">
              <input
                type="text"
                value={nuevoTipoNombre}
                onChange={(e) => setNuevoTipoNombre(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && crearTipo()}
                placeholder="Nombre del nuevo tipo (ej. Constancia de Estudios)"
                className="flex-1 bg-slate-50 border border-slate-200 px-4 py-3 rounded-2xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold"
              />
              <button
                onClick={crearTipo}
                disabled={creandoTipo || !nuevoTipoNombre.trim()}
                className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-white text-xs font-black uppercase px-5 py-3 rounded-2xl shadow-md transition-all active:scale-95 shrink-0"
              >
                {creandoTipo ? <Loader2 className="animate-spin" size={16} /> : <Plus size={16} />} Agregar
              </button>
            </div>
            <p className="text-[10px] text-slate-400 ml-2">
              Se aplica de inmediato a todos los niños -- no hay que configurarlo uno por uno.
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelConfiguracion;
