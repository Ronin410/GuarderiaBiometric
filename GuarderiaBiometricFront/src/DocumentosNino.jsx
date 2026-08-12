import React, { useState, useEffect, useRef } from 'react';
import api from './axiosConfig';
import { FileText, Upload, RefreshCw, Trash2, Loader2, ExternalLink, CheckCircle2 } from 'lucide-react';
import { mostrarError, confirmar } from './utils/alertas';

// Mismo catálogo y orden que ordenTiposDocumento en el backend
// (internal/server/documentos.go) — las etiquetas legibles solo viven aquí,
// igual que CONCEPTOS para Pagos.
const TIPOS_DOCUMENTO = [
  { valor: 'acta_nacimiento', label: 'Acta de Nacimiento' },
  { valor: 'curp', label: 'CURP' },
  { valor: 'comprobante_domicilio', label: 'Comprobante de Domicilio' },
  { valor: 'cartilla_vacunacion', label: 'Cartilla de Vacunación' },
  { valor: 'identificacion_tutor', label: 'Identificación del Tutor' },
  { valor: 'otro', label: 'Otro' },
];

// DocumentosNino — checklist de documentos de inscripción de un niño
// (acta de nacimiento, comprobante de domicilio, etc.), embebido en la
// tarjeta expandida de PanelPerfiles.jsx. Cada tipo permite subir/reemplazar
// (mismo campo cubre ambos casos) y eliminar.
const DocumentosNino = ({ ninoId }) => {
  const [documentos, setDocumentos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [subiendoTipo, setSubiendoTipo] = useState(null);
  const [eliminandoTipo, setEliminandoTipo] = useState(null);
  const inputsRef = useRef({});

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
    <div className="space-y-2">
      {TIPOS_DOCUMENTO.map(({ valor, label }) => {
        const doc = documentos.find((d) => d.tipo === valor);
        const subido = !!doc?.nombre_archivo;
        return (
          <div key={valor} className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 bg-white border border-slate-200 rounded-xl px-4 py-3">
            <div className="flex items-center gap-2.5 min-w-0">
              {subido ? <CheckCircle2 size={16} className="text-emerald-500 shrink-0" /> : <FileText size={16} className="text-slate-300 shrink-0" />}
              <div className="min-w-0">
                <p className="text-xs font-black uppercase text-slate-700">{label}</p>
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
                ref={(el) => (inputsRef.current[valor] = el)}
                type="file"
                accept="image/*,application/pdf"
                className="hidden"
                onChange={(e) => subirArchivo(valor, e.target.files?.[0])}
              />
              <button
                onClick={() => elegirArchivo(valor)}
                disabled={subiendoTipo === valor}
                className="flex items-center gap-1.5 bg-slate-100 hover:bg-slate-200 disabled:opacity-50 text-slate-600 text-[10px] font-black uppercase px-3 py-1.5 rounded-lg transition-all"
              >
                {subiendoTipo === valor ? <Loader2 className="animate-spin" size={12} /> : subido ? <RefreshCw size={12} /> : <Upload size={12} />}
                {subido ? 'Reemplazar' : 'Subir'}
              </button>
              {subido && (
                <button
                  onClick={() => eliminarArchivo(valor, label)}
                  disabled={eliminandoTipo === valor}
                  className="text-rose-400 hover:text-rose-600 disabled:opacity-50 p-1.5"
                  title="Eliminar documento"
                >
                  {eliminandoTipo === valor ? <Loader2 className="animate-spin" size={14} /> : <Trash2 size={14} />}
                </button>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default DocumentosNino;
