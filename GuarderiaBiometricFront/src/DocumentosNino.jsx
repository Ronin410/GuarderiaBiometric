import React, { useState, useEffect, useRef } from 'react';
import api from './axiosConfig';
import {
  FileText, Upload, RefreshCw, Trash2, Loader2, ExternalLink, CheckCircle2,
  Settings2, Plus, Edit3, Check, X,
} from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';

// DocumentosNino — checklist de documentos de inscripción de un niño
// (acta de nacimiento, comprobante de domicilio, etc.), embebido en la
// tarjeta expandida de PanelPerfiles.jsx. Cada tipo permite subir/reemplazar
// (mismo campo cubre ambos casos) y eliminar.
//
// El catálogo de tipos (cuáles documentos existen, y en qué orden) ya NO es
// fijo -- es configurable por guardería (tipos_documento.go en el backend),
// así que este componente también trae "Gestionar tipos": crear, renombrar
// y eliminar, mismo patrón que "Gestionar grupos" en PanelPerfiles.jsx. Vive
// aquí (junto al checklist) y no en un panel de configuración aparte porque
// es donde el staff de verdad necesita verlo para decidir qué le falta
// pedir a cada familia.
const DocumentosNino = ({ ninoId }) => {
  const [documentos, setDocumentos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [subiendoTipo, setSubiendoTipo] = useState(null);
  const [eliminandoTipo, setEliminandoTipo] = useState(null);
  const inputsRef = useRef({});

  const [gestionando, setGestionando] = useState(false);
  const [tipos, setTipos] = useState([]);
  const [nuevoTipoNombre, setNuevoTipoNombre] = useState('');
  const [creandoTipo, setCreandoTipo] = useState(false);
  const [editandoTipoId, setEditandoTipoId] = useState(null);
  const [nombreEdit, setNombreEdit] = useState('');

  const cargar = async () => {
    setLoading(true);
    try {
      const res = await api.get(`/hijos/${ninoId}/documentos`);
      setDocumentos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar documentos:', err);
      mostrarError('No se pudieron cargar los documentos');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, [ninoId]);

  const cargarTipos = async () => {
    try {
      const res = await api.get('/admin/tipos-documento');
      setTipos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar tipos de documento:', err);
      mostrarError('No se pudieron cargar los tipos de documento');
    }
  };

  const abrirGestion = () => {
    setGestionando(true);
    cargarTipos();
  };

  const crearTipo = async () => {
    if (!nuevoTipoNombre.trim()) return;
    setCreandoTipo(true);
    try {
      await api.post('/admin/tipos-documento', { nombre: nuevoTipoNombre.trim() });
      setNuevoTipoNombre('');
      await cargarTipos();
      await cargar();
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
      await cargar();
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
      await cargar();
    } catch (err) {
      console.error('Error al eliminar el tipo de documento:', err);
      mostrarError(err.response?.data?.error || 'No se pudo eliminar el tipo de documento');
    }
  };

  const elegirArchivo = (tipo) => {
    inputsRef.current[tipo]?.click();
  };

  const subirArchivo = async (tipo, file) => {
    if (!file) return;
    if (file.size > 10 * 1024 * 1024) {
      mostrarError('El archivo no puede pesar más de 10 MB');
      return;
    }
    setSubiendoTipo(tipo);
    try {
      const data = new FormData();
      data.append('tipo', tipo);
      data.append('archivo', file);
      await api.post(`/hijos/${ninoId}/documentos`, data);
      cargar();
    } catch (err) {
      console.error('Error al subir documento:', err);
      mostrarError(err.response?.data?.error || 'No se pudo subir el archivo');
    } finally {
      setSubiendoTipo(null);
    }
  };

  const eliminarArchivo = async (tipo, label) => {
    const ok = await confirmar(`Se eliminará "${label}".`, '¿Eliminar documento?');
    if (!ok) return;
    setEliminandoTipo(tipo);
    try {
      await api.delete(`/hijos/${ninoId}/documentos/${tipo}`);
      cargar();
    } catch (err) {
      console.error('Error al eliminar documento:', err);
      mostrarError(err.response?.data?.error || 'No se pudo eliminar el documento');
    } finally {
      setEliminandoTipo(null);
    }
  };

  if (loading) {
    return <div className="py-8 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando documentos...</div>;
  }

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <button
          onClick={() => (gestionando ? setGestionando(false) : abrirGestion())}
          className={`flex items-center gap-1.5 text-[10px] font-black uppercase px-3 py-1.5 rounded-full transition-all ${gestionando ? 'bg-slate-200 text-slate-700' : 'text-slate-400 hover:bg-slate-100'}`}
          title="Elegir qué documentos le pides a las familias"
        >
          <Settings2 size={12} /> {gestionando ? 'Listo' : 'Gestionar tipos'}
        </button>
      </div>

      {gestionando && (
        <div className="bg-slate-50 border border-slate-200 rounded-2xl p-4 space-y-3 animate-in fade-in duration-200">
          <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Documentos que le pides a las familias</p>
          <div className="space-y-1.5">
            {tipos.map((t) => (
              <div key={t.id} className="flex items-center gap-2 bg-white border border-slate-200 rounded-xl px-3 py-2">
                {editandoTipoId === t.id ? (
                  <>
                    <input
                      autoFocus
                      value={nombreEdit}
                      onChange={(e) => setNombreEdit(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && guardarNombreTipo(t)}
                      className="flex-1 bg-slate-50 border border-brand-300 rounded-lg px-2 py-1 text-xs font-bold outline-none"
                    />
                    <button onClick={() => guardarNombreTipo(t)} className="text-emerald-500 p-1"><Check size={14} /></button>
                    <button onClick={() => setEditandoTipoId(null)} className="text-slate-400 p-1"><X size={14} /></button>
                  </>
                ) : (
                  <>
                    <span className="flex-1 text-xs font-bold text-slate-700">{t.nombre}</span>
                    {t.en_uso > 0 && (
                      <span className="text-[9px] font-black text-slate-400 uppercase">{t.en_uso} subido{t.en_uso === 1 ? '' : 's'}</span>
                    )}
                    <button onClick={() => { setEditandoTipoId(t.id); setNombreEdit(t.nombre); }} className="text-slate-300 hover:text-brand-600 p-1" title="Renombrar">
                      <Edit3 size={13} />
                    </button>
                    <button onClick={() => eliminarTipo(t)} className="text-slate-300 hover:text-rose-500 p-1" title="Eliminar tipo">
                      <Trash2 size={13} />
                    </button>
                  </>
                )}
              </div>
            ))}
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={nuevoTipoNombre}
              onChange={(e) => setNuevoTipoNombre(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && crearTipo()}
              placeholder="Nombre del nuevo tipo (ej. Constancia de Estudios)"
              className="flex-1 bg-white border border-slate-200 px-3 py-2 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-xs font-bold"
            />
            <button
              onClick={crearTipo}
              disabled={creandoTipo || !nuevoTipoNombre.trim()}
              className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-white text-[10px] font-black uppercase px-3 py-2 rounded-xl shadow-md transition-all shrink-0"
            >
              {creandoTipo ? <Loader2 className="animate-spin" size={14} /> : <Plus size={14} />} Agregar
            </button>
          </div>
        </div>
      )}

      {documentos.map((doc) => {
        const subido = !!doc.nombre_archivo;
        return (
          <div key={doc.tipo} className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 bg-white border border-slate-200 rounded-xl px-4 py-3">
            <div className="flex items-center gap-2.5 min-w-0">
              {subido ? <CheckCircle2 size={16} className="text-emerald-500 shrink-0" /> : <FileText size={16} className="text-slate-300 shrink-0" />}
              <div className="min-w-0">
                <p className="text-xs font-black uppercase text-slate-700">{doc.nombre}</p>
                <p className="text-[10px] text-slate-400 font-medium truncate">
                  {subido ? doc.nombre_archivo : 'Sin subir'}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {subido && doc.url && (
                <a href={doc.url} target="_blank" rel="noreferrer" className="flex items-center gap-1 text-[10px] font-black uppercase text-brand-600 hover:text-brand-700 px-2 py-1.5">
                  <ExternalLink size={12} /> Ver
                </a>
              )}
              <input
                ref={(el) => (inputsRef.current[doc.tipo] = el)}
                type="file"
                accept="image/*,application/pdf"
                className="hidden"
                onChange={(e) => subirArchivo(doc.tipo, e.target.files?.[0])}
              />
              <button
                onClick={() => elegirArchivo(doc.tipo)}
                disabled={subiendoTipo === doc.tipo}
                className="flex items-center gap-1.5 bg-slate-100 hover:bg-slate-200 disabled:opacity-50 text-slate-600 text-[10px] font-black uppercase px-3 py-1.5 rounded-lg transition-all"
              >
                {subiendoTipo === doc.tipo ? <Loader2 className="animate-spin" size={12} /> : subido ? <RefreshCw size={12} /> : <Upload size={12} />}
                {subido ? 'Reemplazar' : 'Subir'}
              </button>
              {subido && (
                <button
                  onClick={() => eliminarArchivo(doc.tipo, doc.nombre)}
                  disabled={eliminandoTipo === doc.tipo}
                  className="text-rose-400 hover:text-rose-600 disabled:opacity-50 p-1.5"
                  title="Eliminar documento"
                >
                  {eliminandoTipo === doc.tipo ? <Loader2 className="animate-spin" size={14} /> : <Trash2 size={14} />}
                </button>
              )}
            </div>
          </div>
        );
      })}
      {documentos.length === 0 && (
        <div className="py-6 text-center text-slate-400 font-black uppercase tracking-widest text-[10px]">
          Sin tipos de documento configurados -- usa "Gestionar tipos" para agregar el primero
        </div>
      )}
    </div>
  );
};

export default DocumentosNino;
