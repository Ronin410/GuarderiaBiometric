import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import {
  Search, IdCard, Eye, EyeOff, Edit3, Check, X, Loader2,
  Cake, MapPin, Phone, Wallet, Users
} from 'lucide-react';

const PanelPerfiles = () => {
  const [ninos, setNinos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [busqueda, setBusqueda] = useState('');
  const [verBajas, setVerBajas] = useState(false);

  const [editandoId, setEditandoId] = useState(null);
  const [form, setForm] = useState({});
  const [guardando, setGuardando] = useState(false);

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

  useEffect(() => { cargarNinos(); }, []);

  const iniciarEdicion = (nino) => {
    setEditandoId(nino.id);
    setForm({
      fecha_nacimiento: nino.fecha_nacimiento || '',
      direccion: nino.direccion || '',
      contacto_emergencia_nombre: nino.contacto_emergencia_nombre || '',
      contacto_emergencia_telefono: nino.contacto_emergencia_telefono || '',
      colegiatura_mensual: nino.colegiatura_mensual || 0,
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
      });
      setEditandoId(null);
      cargarNinos();
    } catch (err) {
      console.error('Error al guardar perfil:', err);
      alert('❌ No se pudo guardar el perfil');
    } finally {
      setGuardando(false);
    }
  };

  const ninosFiltrados = ninos
    .filter(n => verBajas ? true : n.activo)
    .filter(n => n.nombre.toLowerCase().includes(busqueda.toLowerCase()));

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-violet-100 p-3 rounded-2xl text-violet-600"><IdCard size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Perfiles de Alumnos</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Expediente administrativo</p>
            </div>
          </div>
          <button
            onClick={() => setVerBajas(!verBajas)}
            className="flex items-center gap-2 text-[9px] font-black uppercase bg-slate-100 hover:bg-slate-200 px-3 py-2 rounded-full transition-all text-slate-600"
          >
            {verBajas ? <EyeOff size={12} /> : <Eye size={12} />}
            {verBajas ? 'Ocultar Bajas' : 'Ver Bajas'}
          </button>
        </div>

        <div className="relative mb-8">
          <Search className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
          <input
            type="text"
            placeholder="Buscar niño por nombre..."
            value={busqueda}
            onChange={(e) => setBusqueda(e.target.value)}
            className="w-full bg-slate-50 border border-slate-200 pl-14 pr-6 py-4 rounded-2xl outline-none focus:ring-2 focus:ring-violet-500 font-medium text-slate-900"
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
                    <p className="font-black text-lg uppercase tracking-tight text-slate-900">{nino.nombre}</p>
                    <p className="text-[10px] text-violet-500 font-bold uppercase flex items-center gap-1 mt-1">
                      <Users size={12} /> {nino.tutores || 'Sin tutor vinculado'}
                    </p>
                  </div>
                  {editandoId !== nino.id && (
                    <button
                      onClick={() => iniciarEdicion(nino)}
                      className="flex items-center gap-2 bg-violet-600 hover:bg-violet-700 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95"
                    >
                      <Edit3 size={14} /> Editar Perfil
                    </button>
                  )}
                </div>

                {editandoId === nino.id ? (
                  <div className="mt-6 pt-6 border-t border-slate-200 grid sm:grid-cols-2 gap-4">
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Fecha de nacimiento</label>
                      <input type="date" value={form.fecha_nacimiento} onChange={(e) => setForm({ ...form, fecha_nacimiento: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-bold" />
                    </div>
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Colegiatura mensual (MXN)</label>
                      <input type="number" min="0" step="0.01" value={form.colegiatura_mensual} onChange={(e) => setForm({ ...form, colegiatura_mensual: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-bold" />
                    </div>
                    <div className="sm:col-span-2">
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Dirección</label>
                      <textarea rows={2} value={form.direccion} onChange={(e) => setForm({ ...form, direccion: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-medium resize-none" />
                    </div>
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Contacto de emergencia</label>
                      <input type="text" placeholder="Nombre" value={form.contacto_emergencia_nombre} onChange={(e) => setForm({ ...form, contacto_emergencia_nombre: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-medium" />
                    </div>
                    <div>
                      <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Teléfono de emergencia</label>
                      <input type="tel" placeholder="Teléfono" value={form.contacto_emergencia_telefono} onChange={(e) => setForm({ ...form, contacto_emergencia_telefono: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-violet-500 text-sm font-medium" />
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
                    <span className="flex items-center gap-1.5"><Cake size={14} className="text-violet-400" /> {nino.fecha_nacimiento || 'Sin fecha de nacimiento'}</span>
                    <span className="flex items-center gap-1.5"><MapPin size={14} className="text-violet-400" /> {nino.direccion || 'Sin dirección'}</span>
                    <span className="flex items-center gap-1.5"><Phone size={14} className="text-violet-400" /> {nino.contacto_emergencia_nombre ? `${nino.contacto_emergencia_nombre} · ${nino.contacto_emergencia_telefono || 's/n'}` : 'Sin contacto de emergencia'}</span>
                    <span className="flex items-center gap-1.5"><Wallet size={14} className="text-violet-400" /> ${Number(nino.colegiatura_mensual || 0).toLocaleString('es-MX', { minimumFractionDigits: 2 })} / mes</span>
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
