import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import {
  UserCog, UserPlus, Edit3, Check, X, Loader2, KeyRound, Lock,
  ShieldCheck, ShieldOff, Crown, Contact, ListChecks,
} from 'lucide-react';
import { mostrarExito, mostrarError, confirmar } from './utils/alertas';

const FORM_NUEVO_VACIO = { username: '', nombre: '', password: '', rol: 'staff', pin: '' };

// Mismo catálogo que AreasPermiso en el backend (personal.go) -- si se
// agrega un área ahí, hay que reflejarla acá para que el admin pueda
// concedérsela desde este panel.
const AREAS_PERMISO = [
  { area: 'familia', label: 'Familia (directorio de tutores)' },
  { area: 'bitacora', label: 'Bitácora' },
  { area: 'reportes', label: 'Reportes' },
  { area: 'perfiles', label: 'Perfiles' },
  { area: 'pagos', label: 'Pagos' },
  { area: 'estadisticas', label: 'Estadísticas' },
  { area: 'configuracion', label: 'Configuración' },
  { area: 'menu', label: 'Menú Semanal' },
  { area: 'circulares', label: 'Circulares' },
];

// PanelPersonal — "Gestión de personal": el admin crea cuentas de staff, les
// configura el PIN administrativo, y puede editarlas o darlas de baja sin
// borrarlas (para no romper la bitácora de accesos, que sí guarda su id).
// Solo llega aquí una cuenta admin: App.jsx no muestra la pestaña a staff, y
// el backend (RequireAdmin) la bloquea igual aunque alguien navegue directo.
const PanelPersonal = ({ usuarioActualId }) => {
  const [personal, setPersonal] = useState([]);
  const [loading, setLoading] = useState(true);

  const [mostrarForm, setMostrarForm] = useState(false);
  const [formNuevo, setFormNuevo] = useState(FORM_NUEVO_VACIO);
  const [creando, setCreando] = useState(false);

  const [editandoId, setEditandoId] = useState(null);
  const [formEdit, setFormEdit] = useState({});
  const [guardandoEdit, setGuardandoEdit] = useState(false);

  const [pinEditId, setPinEditId] = useState(null);
  const [pinValor, setPinValor] = useState('');
  const [guardandoPin, setGuardandoPin] = useState(false);

  const [passEditId, setPassEditId] = useState(null);
  const [passValor, setPassValor] = useState('');
  const [guardandoPass, setGuardandoPass] = useState(false);

  // permisosPersonalizado=false representa "sin personalizar" (permisos:
  // null en el backend, acceso completo tras el PIN de siempre); en true,
  // permisosSeleccion es la lista exacta de áreas concedidas.
  const [permisosEditId, setPermisosEditId] = useState(null);
  const [permisosPersonalizado, setPermisosPersonalizado] = useState(false);
  const [permisosSeleccion, setPermisosSeleccion] = useState([]);
  const [guardandoPermisos, setGuardandoPermisos] = useState(false);

  const cargarPersonal = async () => {
    setLoading(true);
    try {
      const res = await api.get('/admin/personal');
      setPersonal(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar personal:', err);
      mostrarError('No se pudo cargar la lista de personal');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargarPersonal(); }, []);

  const cerrarTodosLosSubformularios = () => {
    setEditandoId(null);
    setPinEditId(null);
    setPassEditId(null);
    setPermisosEditId(null);
  };

  const crearCuenta = async () => {
    if (!formNuevo.username.trim() || formNuevo.password.length < 8) {
      mostrarError('El usuario y una contraseña de al menos 8 caracteres son obligatorios');
      return;
    }
    if (!/^\d{4}$/.test(formNuevo.pin)) {
      mostrarError('El PIN debe ser de exactamente 4 dígitos');
      return;
    }
    setCreando(true);
    try {
      await api.post('/admin/personal', formNuevo);
      mostrarExito('La cuenta fue creada correctamente');
      setFormNuevo(FORM_NUEVO_VACIO);
      setMostrarForm(false);
      cargarPersonal();
    } catch (err) {
      console.error('Error al crear cuenta:', err);
      mostrarError(err.response?.data?.error || 'No se pudo crear la cuenta');
    } finally {
      setCreando(false);
    }
  };

  const iniciarEdicion = (p) => {
    cerrarTodosLosSubformularios();
    setEditandoId(p.id);
    setFormEdit({ nombre: p.nombre || '', rol: p.rol, activo: p.activo });
  };

  const guardarEdicion = async (id) => {
    setGuardandoEdit(true);
    try {
      await api.put(`/admin/personal/${id}`, formEdit);
      mostrarExito('Cambios guardados');
      setEditandoId(null);
      cargarPersonal();
    } catch (err) {
      console.error('Error al actualizar cuenta:', err);
      mostrarError(err.response?.data?.error || 'No se pudo guardar');
    } finally {
      setGuardandoEdit(false);
    }
  };

  const alternarActivo = async (p) => {
    const accion = p.activo ? 'DESACTIVAR' : 'reactivar';
    const ok = await confirmar(
      p.activo
        ? `${p.nombre || p.username} ya no podrá iniciar sesión hasta que la reactives.`
        : `${p.nombre || p.username} podrá volver a iniciar sesión.`,
      `¿${accion.charAt(0).toUpperCase() + accion.slice(1)} cuenta?`,
    );
    if (!ok) return;
    try {
      await api.put(`/admin/personal/${p.id}`, { nombre: p.nombre || '', rol: p.rol, activo: !p.activo });
      mostrarExito(p.activo ? 'Cuenta desactivada' : 'Cuenta reactivada');
      cargarPersonal();
    } catch (err) {
      console.error('Error al cambiar estado de la cuenta:', err);
      mostrarError(err.response?.data?.error || 'No se pudo cambiar el estado');
    }
  };

  const iniciarPin = (p) => {
    cerrarTodosLosSubformularios();
    setPinEditId(p.id);
    setPinValor('');
  };

  const guardarPin = async (id) => {
    if (!/^\d{4}$/.test(pinValor)) {
      mostrarError('El PIN debe ser de exactamente 4 dígitos');
      return;
    }
    setGuardandoPin(true);
    try {
      await api.put(`/admin/personal/${id}/pin`, { pin: pinValor });
      mostrarExito('PIN actualizado');
      setPinEditId(null);
      setPinValor('');
    } catch (err) {
      console.error('Error al cambiar PIN:', err);
      mostrarError(err.response?.data?.error || 'No se pudo cambiar el PIN');
    } finally {
      setGuardandoPin(false);
    }
  };

  const iniciarPassword = (p) => {
    cerrarTodosLosSubformularios();
    setPassEditId(p.id);
    setPassValor('');
  };

  const guardarPassword = async (id) => {
    if (passValor.length < 8) {
      mostrarError('La contraseña debe tener al menos 8 caracteres');
      return;
    }
    setGuardandoPass(true);
    try {
      await api.put(`/admin/personal/${id}/password`, { password: passValor });
      mostrarExito('Contraseña actualizada');
      setPassEditId(null);
      setPassValor('');
    } catch (err) {
      console.error('Error al resetear contraseña:', err);
      mostrarError(err.response?.data?.error || 'No se pudo cambiar la contraseña');
    } finally {
      setGuardandoPass(false);
    }
  };

  const iniciarPermisos = (p) => {
    cerrarTodosLosSubformularios();
    setPermisosEditId(p.id);
    setPermisosPersonalizado(p.permisos !== null && p.permisos !== undefined);
    setPermisosSeleccion(p.permisos || []);
  };

  const alternarArea = (area) => {
    setPermisosSeleccion((prev) => (
      prev.includes(area) ? prev.filter((a) => a !== area) : [...prev, area]
    ));
  };

  const guardarPermisos = async (id) => {
    setGuardandoPermisos(true);
    try {
      await api.put(`/admin/personal/${id}/permisos`, {
        permisos: permisosPersonalizado ? permisosSeleccion : null,
      });
      mostrarExito('Permisos actualizados');
      setPermisosEditId(null);
      cargarPersonal();
    } catch (err) {
      console.error('Error al actualizar permisos:', err);
      mostrarError(err.response?.data?.error || 'No se pudo guardar');
    } finally {
      setGuardandoPermisos(false);
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
          <div className="flex items-center gap-4">
            <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><UserCog size={28} /></div>
            <div>
              <h3 className="text-xl font-black uppercase text-slate-900">Gestión de Personal</h3>
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Cuentas de staff y administradores</p>
            </div>
          </div>
          <button
            onClick={() => { setMostrarForm(!mostrarForm); setFormNuevo(FORM_NUEVO_VACIO); }}
            className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white text-[10px] font-black uppercase px-4 py-2.5 rounded-xl shadow-md transition-all active:scale-95"
          >
            {mostrarForm ? <X size={14} /> : <UserPlus size={14} />}
            {mostrarForm ? 'Cancelar' : 'Crear Cuenta'}
          </button>
        </div>

        {mostrarForm && (
          <div className="mb-8 p-6 rounded-[2rem] border border-dashed border-brand-300 bg-brand-50/40 grid sm:grid-cols-2 gap-4">
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Usuario (para iniciar sesión)</label>
              <input type="text" value={formNuevo.username} onChange={(e) => setFormNuevo({ ...formNuevo, username: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" placeholder="ej. staff_maria" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Nombre para mostrar</label>
              <input type="text" value={formNuevo.nombre} onChange={(e) => setFormNuevo({ ...formNuevo, nombre: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" placeholder="ej. María Pérez" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Contraseña (mín. 8 caracteres)</label>
              <input type="password" value={formNuevo.password} onChange={(e) => setFormNuevo({ ...formNuevo, password: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">PIN administrativo (4 dígitos)</label>
              <input type="text" inputMode="numeric" maxLength={4} value={formNuevo.pin} onChange={(e) => setFormNuevo({ ...formNuevo, pin: e.target.value.replace(/\D/g, '') })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold tracking-widest" placeholder="0000" />
            </div>
            <div>
              <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Rol</label>
              <select value={formNuevo.rol} onChange={(e) => setFormNuevo({ ...formNuevo, rol: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold">
                <option value="staff">Staff</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div className="sm:col-span-2 flex justify-end pt-2">
              <button onClick={crearCuenta} disabled={creando} className="flex items-center gap-2 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase text-xs px-6 py-3 rounded-xl shadow-md transition-all active:scale-95">
                {creando ? <Loader2 className="animate-spin" size={16} /> : <Check size={16} />} Crear
              </button>
            </div>
          </div>
        )}

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : personal.length === 0 ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Sin cuentas registradas</div>
        ) : (
          <div className="space-y-4">
            {personal.map((p) => {
              const esUnoMismo = p.id === usuarioActualId;
              return (
                <div key={p.id} className={`p-6 rounded-[2rem] border transition-all ${!p.activo ? 'bg-slate-100 opacity-60 border-dashed border-slate-300' : 'bg-slate-50 border-slate-100'}`}>
                  <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                    <div>
                      <p className="font-black text-lg uppercase tracking-tight text-slate-900 flex items-center gap-2">
                        {p.nombre || p.username}
                        {p.rol === 'admin' && <Crown size={14} className="text-amber-500" />}
                        {esUnoMismo && <span className="text-[9px] bg-brand-100 text-brand-600 px-2 py-0.5 rounded-full">TÚ</span>}
                      </p>
                      <p className="text-[10px] text-brand-500 font-bold uppercase flex items-center gap-1 mt-1">
                        <Contact size={12} /> @{p.username} · {p.rol} · {p.activo ? 'Activo' : 'Desactivado'}
                        {p.rol === 'staff' && (
                          <span className="ml-1 text-slate-400 normal-case tracking-normal">
                            · {p.permisos === null || p.permisos === undefined
                              ? 'acceso completo (sin personalizar)'
                              : p.permisos.length === 0
                                ? 'sin secciones protegidas concedidas'
                                : `${p.permisos.length} sección(es) concedida(s)`}
                          </span>
                        )}
                      </p>
                    </div>
                    {editandoId !== p.id && (
                      <div className="flex flex-wrap gap-2">
                        <button onClick={() => iniciarPin(p)} className="flex items-center gap-1.5 bg-white border border-slate-200 hover:border-brand-300 text-slate-600 text-[10px] font-black uppercase px-3 py-2 rounded-xl transition-all"><KeyRound size={13} /> PIN</button>
                        <button onClick={() => iniciarPassword(p)} className="flex items-center gap-1.5 bg-white border border-slate-200 hover:border-brand-300 text-slate-600 text-[10px] font-black uppercase px-3 py-2 rounded-xl transition-all"><Lock size={13} /> Contraseña</button>
                        {p.rol === 'staff' && (
                          // No tiene sentido para "admin": esa cuenta siempre tiene
                          // acceso completo sin importar lo que diga permisos.
                          <button onClick={() => iniciarPermisos(p)} className="flex items-center gap-1.5 bg-white border border-slate-200 hover:border-brand-300 text-slate-600 text-[10px] font-black uppercase px-3 py-2 rounded-xl transition-all"><ListChecks size={13} /> Permisos</button>
                        )}
                        <button onClick={() => iniciarEdicion(p)} className="flex items-center gap-1.5 bg-brand-600 hover:bg-brand-700 text-white text-[10px] font-black uppercase px-3 py-2 rounded-xl shadow-md transition-all"><Edit3 size={13} /> Editar</button>
                        {!esUnoMismo && (
                          <button onClick={() => alternarActivo(p)} className={`flex items-center gap-1.5 text-[10px] font-black uppercase px-3 py-2 rounded-xl transition-all ${p.activo ? 'bg-rose-50 text-rose-600 hover:bg-rose-100' : 'bg-emerald-50 text-emerald-600 hover:bg-emerald-100'}`}>
                            {p.activo ? <ShieldOff size={13} /> : <ShieldCheck size={13} />} {p.activo ? 'Desactivar' : 'Reactivar'}
                          </button>
                        )}
                      </div>
                    )}
                  </div>

                  {editandoId === p.id && (
                    <div className="mt-6 pt-6 border-t border-slate-200 grid sm:grid-cols-2 gap-4">
                      <div>
                        <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Nombre para mostrar</label>
                        <input type="text" value={formEdit.nombre} onChange={(e) => setFormEdit({ ...formEdit, nombre: e.target.value })} className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" />
                      </div>
                      <div>
                        <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Rol</label>
                        <select
                          value={formEdit.rol}
                          disabled={esUnoMismo}
                          onChange={(e) => setFormEdit({ ...formEdit, rol: e.target.value })}
                          className="w-full bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold disabled:opacity-50"
                        >
                          <option value="staff">Staff</option>
                          <option value="admin">Admin</option>
                        </select>
                      </div>
                      <div className="sm:col-span-2 flex gap-3 justify-end pt-2">
                        <button onClick={() => setEditandoId(null)} className="px-5 py-3 rounded-xl text-slate-500 font-bold uppercase text-xs hover:bg-slate-100 flex items-center gap-2"><X size={16} /> Cancelar</button>
                        <button onClick={() => guardarEdicion(p.id)} disabled={guardandoEdit} className="px-5 py-3 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white font-black uppercase text-xs shadow-md flex items-center gap-2 disabled:opacity-50">
                          {guardandoEdit ? <Loader2 className="animate-spin" size={16} /> : <Check size={16} />} Guardar
                        </button>
                      </div>
                    </div>
                  )}

                  {pinEditId === p.id && (
                    <div className="mt-6 pt-6 border-t border-slate-200 flex flex-wrap items-end gap-3">
                      <div>
                        <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Nuevo PIN (4 dígitos)</label>
                        <input type="text" inputMode="numeric" maxLength={4} autoFocus value={pinValor} onChange={(e) => setPinValor(e.target.value.replace(/\D/g, ''))} className="w-32 bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold tracking-[0.3em]" placeholder="0000" />
                      </div>
                      <button onClick={() => setPinEditId(null)} className="px-4 py-3 rounded-xl text-slate-500 font-bold uppercase text-xs hover:bg-slate-100 flex items-center gap-2"><X size={16} /></button>
                      <button onClick={() => guardarPin(p.id)} disabled={guardandoPin} className="px-5 py-3 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white font-black uppercase text-xs shadow-md flex items-center gap-2 disabled:opacity-50">
                        {guardandoPin ? <Loader2 className="animate-spin" size={16} /> : <Check size={16} />} Guardar PIN
                      </button>
                    </div>
                  )}

                  {passEditId === p.id && (
                    <div className="mt-6 pt-6 border-t border-slate-200 flex flex-wrap items-end gap-3">
                      <div>
                        <label className="text-[10px] font-black text-slate-400 uppercase ml-1 mb-1 block">Nueva contraseña (mín. 8 caracteres)</label>
                        <input type="password" autoFocus value={passValor} onChange={(e) => setPassValor(e.target.value)} className="w-56 bg-white border border-slate-200 p-3 rounded-xl outline-none focus:ring-2 focus:ring-brand-500 text-sm font-bold" />
                      </div>
                      <button onClick={() => setPassEditId(null)} className="px-4 py-3 rounded-xl text-slate-500 font-bold uppercase text-xs hover:bg-slate-100 flex items-center gap-2"><X size={16} /></button>
                      <button onClick={() => guardarPassword(p.id)} disabled={guardandoPass} className="px-5 py-3 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white font-black uppercase text-xs shadow-md flex items-center gap-2 disabled:opacity-50">
                        {guardandoPass ? <Loader2 className="animate-spin" size={16} /> : <Check size={16} />} Guardar Contraseña
                      </button>
                    </div>
                  )}

                  {permisosEditId === p.id && (
                    <div className="mt-6 pt-6 border-t border-slate-200">
                      <label className="flex items-center gap-2 mb-4 cursor-pointer w-fit">
                        <input
                          type="checkbox"
                          checked={permisosPersonalizado}
                          onChange={(e) => setPermisosPersonalizado(e.target.checked)}
                          className="w-4 h-4 accent-brand-600"
                        />
                        <span className="text-xs font-bold text-slate-600">
                          Personalizar el acceso de esta cuenta (si no, entra a todo tras el PIN, como siempre)
                        </span>
                      </label>

                      {permisosPersonalizado && (
                        <div className="grid sm:grid-cols-3 gap-2 mb-4">
                          {AREAS_PERMISO.map(({ area, label }) => (
                            <label key={area} className="flex items-center gap-2 bg-white border border-slate-200 rounded-xl px-3 py-2 cursor-pointer text-xs font-bold text-slate-600">
                              <input
                                type="checkbox"
                                checked={permisosSeleccion.includes(area)}
                                onChange={() => alternarArea(area)}
                                className="w-4 h-4 accent-brand-600"
                              />
                              {label}
                            </label>
                          ))}
                        </div>
                      )}

                      <div className="flex gap-3 justify-end">
                        <button onClick={() => setPermisosEditId(null)} className="px-4 py-3 rounded-xl text-slate-500 font-bold uppercase text-xs hover:bg-slate-100 flex items-center gap-2"><X size={16} /></button>
                        <button onClick={() => guardarPermisos(p.id)} disabled={guardandoPermisos} className="px-5 py-3 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white font-black uppercase text-xs shadow-md flex items-center gap-2 disabled:opacity-50">
                          {guardandoPermisos ? <Loader2 className="animate-spin" size={16} /> : <Check size={16} />} Guardar Permisos
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelPersonal;
