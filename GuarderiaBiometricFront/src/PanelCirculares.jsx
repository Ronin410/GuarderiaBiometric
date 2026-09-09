import React, { useState, useEffect, useRef } from 'react';
import api from './axiosConfig';
import { Megaphone, Plus, X, Send, Loader2, Trash2, CalendarClock, Eye, ChevronDown, ChevronUp, CheckCheck, Image as ImageIcon } from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import { acentoDeTab } from './utils/acentos';
import DinoDecorativo from './components/DinoDecorativo';
import SelectorGrupos from './components/SelectorGrupos';
import EtiquetaGrupos from './components/EtiquetaGrupos';

// Color y dino de este apartado -- los define utils/acentos.js para que
// coincidan con los del menú lateral.
const acento = acentoDeTab('circulares');

const FORM_VACIO = { titulo: '', contenido: '' };
// Lista vacía = para todas las familias (ver SelectorGrupos y
// destinatarios.go en el backend).
const GRUPOS_VACIO = [];

// PanelCirculares — "avisos que el admin o staff pueden mandar a todos los
// padres de la guardería". Al publicar, el backend dispara una notificación
// push a todos los tutores suscritos (reutiliza la misma infraestructura ya
// usada para entradas/salidas/bitácora).
const PanelCirculares = () => {
  const [circulares, setCirculares] = useState([]);
  const [loading, setLoading] = useState(true);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState(FORM_VACIO);
  const [publicando, setPublicando] = useState(false);
  const [eliminandoId, setEliminandoId] = useState(null);
  const [detalleAbierto, setDetalleAbierto] = useState(null);
  const [lecturas, setLecturas] = useState([]);
  const [cargandoLecturas, setCargandoLecturas] = useState(false);
  const [gruposDestino, setGruposDestino] = useState(GRUPOS_VACIO);
  // Filtro del listado: null = todas; un id = solo las dirigidas a ese grupo.
  const [filtroGrupo, setFiltroGrupo] = useState(null);
  const [imagen, setImagen] = useState(null);
  const [previewImagen, setPreviewImagen] = useState(null);
  const inputImagenRef = useRef(null);

  const cargar = async () => {
    setLoading(true);
    try {
      const res = await api.get('/circulares');
      setCirculares(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar circulares:', err);
      mostrarError('No se pudieron cargar las circulares');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const elegirImagen = (file) => {
    if (!file) return;
    if (file.size > 10 * 1024 * 1024) {
      mostrarError('La imagen no puede pesar más de 10 MB');
      return;
    }
    setImagen(file);
    setPreviewImagen(URL.createObjectURL(file));
  };

  const quitarImagen = () => {
    setImagen(null);
    setPreviewImagen(null);
    if (inputImagenRef.current) inputImagenRef.current.value = '';
  };

  const publicar = async () => {
    if (!form.titulo.trim() || !form.contenido.trim()) {
      mostrarError('El título y el contenido son obligatorios');
      return;
    }
    setPublicando(true);
    try {
      const data = new FormData();
      data.append('titulo', form.titulo.trim());
      data.append('contenido', form.contenido.trim());
      if (imagen) data.append('imagen', imagen);
      // Cada grupo va como un campo "grupos" repetido; el backend lo lee con
      // PostFormArray. Sin ninguno, la circular es para todas las familias.
      gruposDestino.forEach((id) => data.append('grupos', String(id)));
      await api.post('/circulares', data);
      mostrarExito(
        gruposDestino.length === 0
          ? 'La circular se publicó y se notificó a los tutores suscritos'
          : 'La circular se publicó y se notificó solo a las familias de los grupos elegidos'
      );
      setForm(FORM_VACIO);
      setGruposDestino(GRUPOS_VACIO);
      quitarImagen();
      setMostrarForm(false);
      cargar();
    } catch (err) {
      console.error('Error al publicar circular:', err);
      mostrarError(err.response?.data?.error || 'No se pudo publicar la circular');
    } finally {
      setPublicando(false);
    }
  };

  const eliminar = async (cir) => {
    const ok = await confirmar(`Se eliminará "${cir.titulo}". Ya no se podrá ver en el portal de los padres.`, '¿Eliminar circular?');
    if (!ok) return;
    setEliminandoId(cir.id);
    try {
      await api.delete(`/circulares/${cir.id}`);
      cargar();
    } catch (err) {
      console.error('Error al eliminar circular:', err);
      mostrarError('No se pudo eliminar la circular');
    } finally {
      setEliminandoId(null);
    }
  };

  const alternarDetalle = async (cir) => {
    if (detalleAbierto === cir.id) {
      setDetalleAbierto(null);
      return;
    }
    setDetalleAbierto(cir.id);
    setCargandoLecturas(true);
    try {
      const res = await api.get(`/circulares/${cir.id}/lecturas`);
      setLecturas(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar el detalle de lecturas:', err);
      mostrarError('No se pudo cargar el detalle de lecturas');
      setLecturas([]);
    } finally {
      setCargandoLecturas(false);
    }
  };

  // Filtrado en memoria y no con otra llamada a la API: el listado trae 50
  // circulares como mucho, así que pedirle al servidor una consulta nueva
  // por cada clic sería más lento que recorrerlas aquí.
  const circularesVisibles = filtroGrupo === null
    ? circulares
    : circulares.filter((cir) => (cir.grupos || []).some((g) => g.id === filtroGrupo));

  // Solo se ofrecen como filtro los grupos que de verdad aparecen en alguna
  // circular; un desplegable con salones que nunca recibieron ninguna solo
  // lleva a listas vacías.
  const gruposEnUso = [];
  circulares.forEach((cir) => {
    (cir.grupos || []).forEach((g) => {
      if (!gruposEnUso.some((x) => x.id === g.id)) gruposEnUso.push(g);
    });
  });

  const formatoFecha = (iso) => {
    try {
      return new Date(iso).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch {
      return iso;
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className={`${acento.fondo} p-3 rounded-2xl ${acento.texto}`}><Megaphone size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Circulares</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Avisos para todos los padres</p>
            </div>
            <DinoDecorativo src="/dinos/dino-naranja.png" className="hidden sm:block h-14 w-auto shrink-0" />
          </div>
          <button
            onClick={() => { setMostrarForm(!mostrarForm); setForm(FORM_VACIO); setGruposDestino(GRUPOS_VACIO); quitarImagen(); }}
            className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95"
          >
            {mostrarForm ? <X size={14} /> : <Plus size={14} />}
            {mostrarForm ? 'Cancelar' : 'Nueva Circular'}
          </button>
        </div>

        {mostrarForm && (
          <div className="mb-8 p-6 rounded-[2rem] border border-dashed border-brand-300 bg-brand-50/40 space-y-4">
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Título</label>
              <input
                type="text" value={form.titulo} onChange={(e) => setForm({ ...form, titulo: e.target.value })}
                placeholder="ej. Suspensión de clases el viernes"
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold"
              />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Contenido</label>
              <textarea
                rows={4} value={form.contenido} onChange={(e) => setForm({ ...form, contenido: e.target.value })}
                placeholder="Escribe el aviso completo..."
                className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium resize-y"
              />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Imagen (opcional)</label>
              {previewImagen ? (
                <div className="relative inline-block">
                  <img src={previewImagen} alt="preview" className="max-h-40 rounded-xl border border-slate-200" />
                  <button onClick={quitarImagen} className="absolute -top-2 -right-2 bg-rose-500 text-white p-1.5 rounded-full shadow-md"><X size={14} /></button>
                </div>
              ) : (
                <button
                  onClick={() => inputImagenRef.current?.click()}
                  className="flex items-center gap-2 bg-white border border-dashed border-slate-300 hover:border-brand-400 text-slate-500 text-xs font-bold px-4 py-3 rounded-xl transition-all"
                >
                  <ImageIcon size={16} /> Agregar imagen
                </button>
              )}
              <input
                ref={inputImagenRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => elegirImagen(e.target.files?.[0])}
              />
            </div>
            <SelectorGrupos seleccionados={gruposDestino} onChange={setGruposDestino} acento={acento} />

            <div className="flex justify-end">
              <button
                onClick={publicar}
                disabled={publicando}
                className="flex items-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-3 rounded-xl shadow-md transition-all active:scale-95"
              >
                {publicando ? <Loader2 className="animate-spin" size={16} /> : <Send size={16} />} Publicar y Notificar
              </button>
            </div>
          </div>
        )}

        {gruposEnUso.length > 0 && (
          <div className="flex flex-wrap items-center gap-2 mb-5">
            <span className="text-[10px] font-black text-slate-400 uppercase tracking-widest mr-1">Ver:</span>
            <button
              onClick={() => setFiltroGrupo(null)}
              className={`text-[11px] font-black uppercase px-3 py-1.5 rounded-lg border transition-all ${
                filtroGrupo === null ? 'bg-forest text-white border-forest' : 'bg-white text-slate-500 border-slate-200 hover:border-slate-300'
              }`}
            >
              Todas
            </button>
            {gruposEnUso.map((g) => (
              <button
                key={g.id}
                onClick={() => setFiltroGrupo(filtroGrupo === g.id ? null : g.id)}
                className={`text-[11px] font-black uppercase px-3 py-1.5 rounded-lg border transition-all ${
                  filtroGrupo === g.id ? `${acento.fondo} ${acento.texto} ${acento.borde}` : 'bg-white text-slate-500 border-slate-200 hover:border-slate-300'
                }`}
              >
                {g.nombre}
              </button>
            ))}
          </div>
        )}

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : circulares.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin circulares publicadas</div>
        ) : circularesVisibles.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Ninguna circular dirigida a ese grupo</div>
        ) : (
          <div className="space-y-4">
            {circularesVisibles.map((cir) => (
              <div key={cir.id} className="p-6 rounded-[2rem] border border-slate-100 bg-slate-50">
                <div className="flex justify-between items-start gap-4">
                  <div className="min-w-0">
                    <p className="font-black text-lg uppercase tracking-tight text-slate-900">{cir.titulo}</p>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 mt-1">
                      <p className="text-[10px] text-brand-500 font-bold uppercase flex items-center gap-1">
                        <CalendarClock size={12} /> {formatoFecha(cir.creado_en)}
                      </p>
                      <EtiquetaGrupos paraTodos={cir.para_todos} grupos={cir.grupos} acento={acento} />
                    </div>
                  </div>
                  <button
                    onClick={() => eliminar(cir)}
                    disabled={eliminandoId === cir.id}
                    className="text-rose-400 hover:text-rose-600 disabled:opacity-50 p-2 shrink-0"
                    title="Eliminar circular"
                  >
                    {eliminandoId === cir.id ? <Loader2 className="animate-spin" size={16} /> : <Trash2 size={16} />}
                  </button>
                </div>
                <p className="mt-3 text-sm text-slate-600 font-medium whitespace-pre-wrap">{cir.contenido}</p>
                {cir.imagen_url && (
                  <img src={cir.imagen_url} alt={cir.titulo} className="mt-3 max-h-64 rounded-2xl border border-slate-200 object-cover" />
                )}

                <button
                  onClick={() => alternarDetalle(cir)}
                  className="mt-4 flex items-center gap-1.5 text-[10px] font-black uppercase text-slate-500 hover:text-brand-600 transition-colors"
                >
                  <Eye size={12} />
                  {cir.leido_por} de {cir.total_familias} familias la han leído
                  {detalleAbierto === cir.id ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                </button>

                {detalleAbierto === cir.id && (
                  <div className="mt-3 pt-3 border-t border-dashed border-slate-200">
                    {cargandoLecturas ? (
                      <p className="text-[10px] font-black text-slate-400 uppercase">Cargando...</p>
                    ) : lecturas.length === 0 ? (
                      <p className="text-[10px] font-black text-slate-400 uppercase">Ninguna familia la ha leído todavía</p>
                    ) : (
                      <ul className="space-y-1.5">
                        {lecturas.map((l, i) => (
                          <li key={i} className="flex items-center gap-2 text-xs font-bold text-slate-600">
                            <CheckCheck size={13} className="text-emerald-500 shrink-0" />
                            {l.nombre}
                            <span className="text-slate-300 font-medium">· {formatoFecha(l.leido_en)}</span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelCirculares;
