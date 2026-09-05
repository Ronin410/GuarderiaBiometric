import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import {
  Search, IdCard, Eye, EyeOff, Edit3, Check, X, Loader2,
  Cake, MapPin, Phone, Wallet, Users, LayoutGrid, Plus, Trash2, Settings2,
  FolderOpen, Image as ImageIcon,
} from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import DocumentosNino from './DocumentosNino';
import GaleriaFotos from './components/GaleriaFotos';
import { acentoDeTab } from './utils/acentos';
import DinoDecorativo from './components/DinoDecorativo';

// Color y dino de este apartado -- los define utils/acentos.js para que
// coincidan con los del menú lateral.
const acento = acentoDeTab('perfiles');

const PanelPerfiles = () => {
  const [ninos, setNinos] = useState([]);
  const [grupos, setGrupos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [busqueda, setBusqueda] = useState('');
  const [verBajas, setVerBajas] = useState(false);
  const [filtroGrupo, setFiltroGrupo] = useState('todos');

  const [editandoId, setEditandoId] = useState(null);
  const [form, setForm] = useState({});
  const [guardando, setGuardando] = useState(false);

  const [gestionandoGrupos, setGestionandoGrupos] = useState(false);
  const [nuevoGrupoNombre, setNuevoGrupoNombre] = useState('');
  const [creandoGrupo, setCreandoGrupo] = useState(false);

  const [documentosAbiertosId, setDocumentosAbiertosId] = useState(null);
  const [galeriaAbiertaId, setGaleriaAbiertaId] = useState(null);
  const [fotoSeleccionada, setFotoSeleccionada] = useState(null);

  const cargarNinos = async () => {
    setLoading(true);
    try {
      const res = await api.get('/admin/ninos');
      setNinos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar niños:', err);
    } finally {
      setLoading(false);
    }
  };

  const cargarGrupos = async () => {
    try {
      const res = await api.get('/admin/grupos');
      setGrupos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar grupos:', err);
    }
  };

  useEffect(() => { cargarNinos(); cargarGrupos(); }, []);

  const alternarDocumentos = (ninoId) => {
    setDocumentosAbiertosId(documentosAbiertosId === ninoId ? null : ninoId);
  };

  const alternarGaleria = (ninoId) => {
    setGaleriaAbiertaId(galeriaAbiertaId === ninoId ? null : ninoId);
  };

  const iniciarEdicion = (nino) => {
    setEditandoId(nino.id);
    setForm({
      fecha_nacimiento: nino.fecha_nacimiento || '',
      direccion: nino.direccion || '',
      contacto_emergencia_nombre: nino.contacto_emergencia_nombre || '',
      contacto_emergencia_telefono: nino.contacto_emergencia_telefono || '',
      colegiatura_mensual: nino.colegiatura_mensual || 0,
      grupo_id: nino.grupo_id || '',
    });
  };

  const cancelarEdicion = () => {
    setEditandoId(null);
    setForm({});
  };

  const guardarPerfil = async (id) => {
    setGuardando(true);
    try {
      await api.put(`/hijos/${id}/perfil`, {
        ...form,
        colegiatura_mensual: parseFloat(form.colegiatura_mensual) || 0,
        grupo_id: form.grupo_id === '' ? null : parseInt(form.grupo_id, 10),
      });
      setEditandoId(null);
      cargarNinos();
      cargarGrupos(); // el conteo de cada píldora de grupo depende de a quién se le asignó
    } catch (err) {
      console.error('Error al guardar perfil:', err);
      mostrarError('No se pudo guardar el perfil');
    } finally {
      setGuardando(false);
    }
  };

  const crearGrupo = async () => {
    if (!nuevoGrupoNombre.trim()) return;
    setCreandoGrupo(true);
    try {
      await api.post('/admin/grupos', { nombre: nuevoGrupoNombre.trim() });
      setNuevoGrupoNombre('');
      cargarGrupos();
    } catch (err) {
      console.error('Error al crear grupo:', err);
      mostrarError('No se pudo crear el grupo');
    } finally {
      setCreandoGrupo(false);
    }
  };

  const eliminarGrupo = async (grupo) => {
    if (grupo.ninos_activos > 0) {
      mostrarError(`"${grupo.nombre}" tiene ${grupo.ninos_activos} niño(s) asignado(s). Reasígnalos a otro grupo antes de eliminarlo.`);
      return;
    }
    const ok = await confirmar(`Se eliminará el grupo "${grupo.nombre}".`, '¿Eliminar grupo?');
    if (!ok) return;
    try {
      await api.delete(`/admin/grupos/${grupo.id}`);
      mostrarExito('Grupo eliminado');
      if (filtroGrupo === grupo.id) setFiltroGrupo('todos');
      cargarGrupos();
    } catch (err) {
      console.error('Error al eliminar grupo:', err);
      mostrarError(err.response?.data?.error || 'No se pudo eliminar el grupo');
    }
  };

  const ninosFiltrados = ninos
    .filter(n => verBajas ? true : n.activo)
    .filter(n => n.nombre.toLowerCase().includes(busqueda.toLowerCase()))
    .filter(n => filtroGrupo === 'todos' ? true : n.grupo_id === filtroGrupo);

  return (
    <div className="animate-in fade-in duration-500">
      {/* MODAL DE FOTO EN GRANDE (galería) */}
      {fotoSeleccionada && (
        <div
          className="fixed inset-0 z-[100] bg-slate-900/90 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in zoom-in duration-200"
          onClick={() => setFotoSeleccionada(null)}
        >
          <button
            className="absolute top-6 right-6 text-white bg-white/10 p-3 rounded-full hover:bg-white/20 transition-all"
            onClick={() => setFotoSeleccionada(null)}
          >
            <X size={24} />
          </button>
          <img
            src={fotoSeleccionada}
            alt="Detalle"
            className="max-w-full max-h-[85vh] rounded-3xl shadow-2xl object-contain"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}

      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className={`${acento.fondo} p-3 rounded-2xl ${acento.texto}`}><IdCard size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Perfiles de Alumnos</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Expediente administrativo</p>
            </div>
            <DinoDecorativo src="/dinos/dino-verde.png" className="hidden sm:block h-14 w-auto shrink-0" />
          </div>
          <button
            onClick={() => setVerBajas(!verBajas)}
            className="flex items-center gap-2 text-[9px] font-black uppercase bg-slate-100 hover:bg-slate-200 px-3 py-2 rounded-full transition-all text-slate-600"
          >
            {verBajas ? <EyeOff size={12} /> : <Eye size={12} />}
            {verBajas ? 'Ocultar Bajas' : 'Ver Bajas'}
          </button>
        </div>

        {/* --- Grupos: filtro + gestión (crear/eliminar) --- */}
        <div className="mb-6">
          <div className="flex flex-wrap items-center gap-2">
            <LayoutGrid size={14} className="text-slate-400 mr-1" />
            <button
              onClick={() => setFiltroGrupo('todos')}
              className={`text-[10px] font-black uppercase px-3.5 py-2 rounded-full border transition-all ${filtroGrupo === 'todos' ? 'bg-brand-600 border-brand-600 text-white shadow-sm' : 'bg-white border-slate-200 text-slate-500 hover:border-brand-300'}`}
            >
              Todos
            </button>
            {grupos.map((g) => (
              <div key={g.id} className="flex items-center gap-1">
                <button
                  onClick={() => setFiltroGrupo(g.id)}
                  className={`flex items-center gap-2 text-[10px] font-black uppercase px-3.5 py-2 rounded-full border transition-all ${filtroGrupo === g.id ? 'bg-brand-600 border-brand-600 text-white shadow-sm' : 'bg-white border-slate-200 text-slate-500 hover:border-brand-300'}`}
                >
                  {g.nombre}
                  <span className={`px-1.5 py-0.5 rounded-full text-[9px] ${filtroGrupo === g.id ? 'bg-white/20' : 'bg-slate-100'}`}>{g.ninos_activos}</span>
                </button>
                {gestionandoGrupos && (
                  <button onClick={() => eliminarGrupo(g)} className="text-rose-400 hover:text-rose-600 p-1" title="Eliminar grupo">
                    <Trash2 size={14} />
                  </button>
                )}
              </div>
            ))}
            <button
              onClick={() => setGestionandoGrupos(!gestionandoGrupos)}
              className={`flex items-center gap-1.5 text-[10px] font-black uppercase px-3 py-2 rounded-full transition-all ${gestionandoGrupos ? 'bg-slate-200 text-slate-700' : 'text-slate-400 hover:bg-slate-100'}`}
              title="Gestionar grupos"
            >
              <Settings2 size={14} /> {gestionandoGrupos ? 'Listo' : 'Gestionar'}
            </button>
          </div>
          {gestionandoGrupos && (
            <div className="flex items-center gap-2 mt-3">
              <input
                type="text"
                value={nuevoGrupoNombre}
                onChange={(e) => setNuevoGrupoNombre(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && crearGrupo()}
                placeholder="Nombre del nuevo grupo (ej. Sala Maternal)"
                className="bg-slate-50 border border-slate-200 px-4 py-2.5 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold flex-1 max-w-xs"
              />
              <button
                onClick={crearGrupo}
                disabled={creandoGrupo || !nuevoGrupoNombre.trim()}
                className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all"
              >
                {creandoGrupo ? <Loader2 className="animate-spin" size={14} /> : <Plus size={14} />} Agregar
              </button>
            </div>
          )}
        </div>

        <div className="relative mb-8">
          <Search className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
          <input
            type="text"
            placeholder="Buscar niño por nombre..."
            value={busqueda}
            onChange={(e) => setBusqueda(e.target.value)}
            className="w-full bg-slate-50 border border-slate-200 pl-14 pr-6 py-4 rounded-2xl outline-none focus:ring-2 focus:ring-brand-500 font-medium text-slate-900"
          />
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : ninosFiltrados.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin resultados</div>
        ) : (
          <div className="space-y-4">
            {ninosFiltrados.map((nino) => (
              <div key={nino.id} className={`p-6 rounded-[2rem] border transition-all ${!nino.activo ? 'bg-slate-100 opacity-60 border-dashed border-slate-300' : 'bg-slate-50 border-slate-100'}`}>
                <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                  <div>
                    <p className="font-black text-lg uppercase tracking-tight text-slate-900 flex items-center gap-2">
                      {nino.nombre}
                      {nino.grupo_nombre && (
                        <span className="text-[9px] bg-brand-100 text-brand-600 px-2 py-0.5 rounded-full normal-case font-black">{nino.grupo_nombre}</span>
                      )}
                    </p>
                    <p className="text-[10px] text-brand-500 font-bold uppercase flex items-center gap-1 mt-1">
                      <Users size={12} /> {nino.tutores || 'Sin tutor vinculado'}
                    </p>
                  </div>
                  {editandoId !== nino.id && (
                    <div className="flex gap-2">
                      <button
                        onClick={() => alternarDocumentos(nino.id)}
                        className={`flex items-center gap-2 text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95 ${documentosAbiertosId === nino.id ? 'bg-slate-700 text-white' : 'bg-white border border-slate-200 text-slate-600 hover:border-brand-300'}`}
                      >
                        <FolderOpen size={14} /> Documentos
                      </button>
                      <button
                        onClick={() => alternarGaleria(nino.id)}
                        className={`flex items-center gap-2 text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95 ${galeriaAbiertaId === nino.id ? 'bg-slate-700 text-white' : 'bg-white border border-slate-200 text-slate-600 hover:border-brand-300'}`}
                      >
                        <ImageIcon size={14} /> Galería
                      </button>
                      <button
                        onClick={() => iniciarEdicion(nino)}
                        className="flex items-center gap-2 bg-brand-600 hover:bg-brand-700 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95"
                      >
                        <Edit3 size={14} /> Editar Perfil
                      </button>
                    </div>
                  )}
                </div>

                {documentosAbiertosId === nino.id && (
                  <div className="mt-6 pt-6 border-t border-slate-200">
                    <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-3">Documentos de inscripción</p>
                    <DocumentosNino ninoId={nino.id} />
                  </div>
                )}

                {galeriaAbiertaId === nino.id && (
                  <div className="mt-6 pt-6 border-t border-slate-200">
                    <p className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-3">Galería de fotos</p>
                    <GaleriaFotos hijoId={nino.id} rutaBase="/hijos" onFotoClick={setFotoSeleccionada} />
                  </div>
                )}

                {editandoId === nino.id ? (
                  <div className="mt-6 pt-6 border-t border-slate-200 grid sm:grid-cols-2 gap-4">
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Fecha de nacimiento</label>
                      <input type="date" value={form.fecha_nacimiento} onChange={(e) => setForm({ ...form, fecha_nacimiento: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" />
                    </div>
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Grupo</label>
                      <select value={form.grupo_id} onChange={(e) => setForm({ ...form, grupo_id: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold">
                        <option value="">Sin grupo</option>
                        {grupos.map((g) => <option key={g.id} value={g.id}>{g.nombre}</option>)}
                      </select>
                    </div>
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Colegiatura mensual (MXN)</label>
                      <input type="number" min="0" step="0.01" value={form.colegiatura_mensual} onChange={(e) => setForm({ ...form, colegiatura_mensual: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" />
                    </div>
                    <div className="sm:col-span-2">
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Dirección</label>
                      <textarea rows={2} value={form.direccion} onChange={(e) => setForm({ ...form, direccion: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium resize-none" />
                    </div>
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Contacto de emergencia</label>
                      <input type="text" placeholder="Nombre" value={form.contacto_emergencia_nombre} onChange={(e) => setForm({ ...form, contacto_emergencia_nombre: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium" />
                    </div>
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Teléfono de emergencia</label>
                      <input type="tel" placeholder="Teléfono" value={form.contacto_emergencia_telefono} onChange={(e) => setForm({ ...form, contacto_emergencia_telefono: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-medium" />
                    </div>

                    <div className="sm:col-span-2 flex gap-3 justify-end pt-2">
                      <button onClick={cancelarEdicion} className="px-5 py-3 rounded-xl text-slate-500 font-bold uppercase text-xs hover:bg-slate-100 flex items-center gap-2"><X size={16} /> Cancelar</button>
                      <button onClick={() => guardarPerfil(nino.id)} disabled={guardando} className="px-5 py-3 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white font-black uppercase text-xs shadow-md flex items-center gap-2 disabled:opacity-50">
                        {guardando ? <Loader2 className="animate-spin" size={16} /> : <Check size={16} />} Guardar
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="mt-4 flex flex-wrap gap-4 text-[11px] text-slate-500 font-medium">
                    <span className="flex items-center gap-1.5"><Cake size={14} className="text-brand-400" /> {nino.fecha_nacimiento || 'Sin fecha de nacimiento'}</span>
                    <span className="flex items-center gap-1.5"><MapPin size={14} className="text-brand-400" /> {nino.direccion || 'Sin dirección'}</span>
                    <span className="flex items-center gap-1.5"><Phone size={14} className="text-brand-400" /> {nino.contacto_emergencia_nombre ? `${nino.contacto_emergencia_nombre} · ${nino.contacto_emergencia_telefono || 's/n'}` : 'Sin contacto de emergencia'}</span>
                    <span className="flex items-center gap-1.5"><Wallet size={14} className="text-brand-400" /> ${Number(nino.colegiatura_mensual || 0).toLocaleString('es-MX', { minimumFractionDigits: 2 })} / mes</span>
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

export default PanelPerfiles;
