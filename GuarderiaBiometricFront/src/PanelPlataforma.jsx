import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { KeyRound, CheckCircle2, XCircle, Building2, RefreshCw, LogOut, Users, Baby, MapPin, Clock, ClipboardList, LifeBuoy } from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import PanelSoportePlataforma from './PanelSoportePlataforma';

const API_URL = import.meta.env.VITE_API_URL || 'https://guarderiabiometricback.onrender.com';
const STORAGE_KEY = 'pasitos_platform_key';

// PanelPlataforma es la vista (solo para el dueño de la plataforma, no para
// una guardería en particular) desde donde se revisan y aprueban/rechazan
// las solicitudes de alta de guardería nueva -- protegida por
// PLATFORM_ADMIN_KEY, NO por el login normal de guarderías (ver
// middleware.RequirePlatformKey en el backend). La llave se guarda solo en
// sessionStorage de este navegador, nunca en el backend ni en una cookie.
const PanelPlataforma = () => {
  const [key, setKey] = useState(() => sessionStorage.getItem(STORAGE_KEY) || '');
  const [keyInput, setKeyInput] = useState('');
  const [vista, setVista] = useState('solicitudes'); // solicitudes | guarderias | soporte
  const [solicitudes, setSolicitudes] = useState(null);
  const [guarderias, setGuarderias] = useState(null);
  const [cargando, setCargando] = useState(false);
  const [cargandoGuarderias, setCargandoGuarderias] = useState(false);
  const [errorKey, setErrorKey] = useState('');
  const [noLeidosSoporte, setNoLeidosSoporte] = useState(0);

  const cabeceras = { headers: { 'X-Platform-Key': key } };

  const cargar = async (llave = key) => {
    setCargando(true);
    setErrorKey('');
    try {
      const res = await axios.get(`${API_URL}/plataforma/solicitudes`, { headers: { 'X-Platform-Key': llave } });
      setSolicitudes(res.data || []);
      sessionStorage.setItem(STORAGE_KEY, llave);
      setKey(llave);
    } catch (err) {
      if (err.response?.status === 401) {
        setErrorKey('Llave inválida');
        sessionStorage.removeItem(STORAGE_KEY);
        setKey('');
      } else {
        mostrarError('No se pudo cargar la lista de solicitudes.');
      }
    } finally {
      setCargando(false);
    }
  };

  // cargarGuarderias -- panorama de las guarderías YA aprobadas (a
  // diferencia de "solicitudes", que son las que todavía esperan
  // revisión): cuántos papás y niños tiene registrados cada una, y cuándo
  // fue la última vez que alguien de ahí entró de verdad.
  const cargarGuarderias = async (llave = key) => {
    setCargandoGuarderias(true);
    try {
      const res = await axios.get(`${API_URL}/plataforma/guarderias`, { headers: { 'X-Platform-Key': llave } });
      setGuarderias(res.data || []);
    } catch (err) {
      if (err.response?.status !== 401) {
        mostrarError('No se pudo cargar la lista de guarderías.');
      }
    } finally {
      setCargandoGuarderias(false);
    }
  };

  useEffect(() => {
    if (key) cargar(key);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (key && vista === 'guarderias' && guarderias === null) cargarGuarderias(key);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [vista, key]);

  // Badge de "Soporte" en la pestaña -- se consulta aparte del inbox
  // completo (PanelSoportePlataforma) para que el número se vea aunque
  // Alejandro esté parado en "Solicitudes" o "Guarderías".
  useEffect(() => {
    if (!key) return undefined;
    const consultar = () => {
      axios.get(`${API_URL}/plataforma/soporte/no-leidos`, { headers: { 'X-Platform-Key': key } })
        .then((res) => setNoLeidosSoporte(res.data?.no_leidos || 0))
        .catch(() => {});
    };
    consultar();
    const intervalo = setInterval(consultar, 15000);
    return () => clearInterval(intervalo);
  }, [key]);

  const formatoFecha = (iso) => {
    if (!iso) return null;
    try {
      return new Date(iso).toLocaleDateString('es-MX', { day: 'numeric', month: 'short', year: 'numeric' });
    } catch {
      return iso;
    }
  };

  const aprobar = async (s) => {
    const ok = await confirmar(`¿Crear la guardería "${s.nombre_guarderia}" con el usuario "${s.username_deseado}"?`, 'Aprobar solicitud');
    if (!ok) return;
    try {
      await axios.post(`${API_URL}/plataforma/solicitudes/${s.id}/aprobar`, {}, cabeceras);
      mostrarExito('Guardería creada. Ya puede iniciar sesión.');
      cargar();
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo aprobar la solicitud.');
    }
  };

  const rechazar = async (s) => {
    const ok = await confirmar(`¿Rechazar la solicitud de "${s.nombre_guarderia}"?`, 'Rechazar solicitud');
    if (!ok) return;
    try {
      await axios.post(`${API_URL}/plataforma/solicitudes/${s.id}/rechazar`, {}, cabeceras);
      mostrarExito('Solicitud rechazada.');
      cargar();
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo rechazar la solicitud.');
    }
  };

  const cerrarSesionPlataforma = () => {
    sessionStorage.removeItem(STORAGE_KEY);
    setKey('');
    setSolicitudes(null);
    setGuarderias(null);
    setVista('solicitudes');
  };

  if (!key) {
    return (
      <div className="min-h-screen bg-paper flex items-center justify-center p-4">
        <form
          onSubmit={(e) => { e.preventDefault(); cargar(keyInput.trim()); }}
          className="bg-white border border-slate-200 p-8 rounded-[2.5rem] w-full max-w-sm shadow-xl text-center"
        >
          <div className="inline-flex bg-forest p-4 rounded-3xl shadow-lg mb-4"><KeyRound size={32} className="text-white" /></div>
          <h1 className="text-xl font-black text-slate-900 mb-1">Plataforma</h1>
          <p className="text-slate-400 text-xs mb-6">Llave de administración, no tu usuario de guardería</p>
          <input
            type="password" autoFocus value={keyInput} onChange={(e) => setKeyInput(e.target.value)}
            className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-center outline-none focus:ring-2 focus:ring-brand-500 transition-all mb-3"
            placeholder="PLATFORM_ADMIN_KEY"
          />
          {errorKey && <p className="text-rose-600 text-xs font-bold mb-3">{errorKey}</p>}
          <button type="submit" disabled={cargando} className="w-full bg-forest hover:bg-forest-light text-white font-black py-3.5 rounded-2xl uppercase text-sm shadow-lg transition-all disabled:opacity-50">
            {cargando ? 'Verificando...' : 'Entrar'}
          </button>
        </form>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-paper p-4 sm:p-8">
      <div className="max-w-3xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="bg-forest p-2.5 rounded-xl"><Building2 size={20} className="text-white" /></div>
            <h1 className="text-xl font-black text-slate-900">
              {vista === 'solicitudes' ? 'Solicitudes de guardería' : vista === 'guarderias' ? 'Guarderías registradas' : 'Chat de soporte'}
            </h1>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => (vista === 'solicitudes' ? cargar() : vista === 'guarderias' ? cargarGuarderias() : null)}
              className="p-2.5 bg-white border border-slate-200 rounded-xl text-slate-500 hover:text-brand-600" title="Actualizar"
            >
              <RefreshCw size={18} className={(vista === 'solicitudes' ? cargando : vista === 'guarderias' ? cargandoGuarderias : false) ? 'animate-spin' : ''} />
            </button>
            <button onClick={cerrarSesionPlataforma} className="p-2.5 bg-white border border-slate-200 rounded-xl text-rose-500 hover:bg-rose-50" title="Salir">
              <LogOut size={18} />
            </button>
          </div>
        </div>

        {/* PESTAÑAS -- "Solicitudes" es lo que ya existía (altas pendientes
            de revisar); "Guarderías" es el panorama de las que ya están
            dadas de alta; "Soporte" es el inbox del chat de soporte (papás,
            staff/admin y prospectos, ver PanelSoportePlataforma). */}
        <div className="flex bg-white border border-slate-200 p-1.5 rounded-2xl w-fit flex-wrap">
          <button
            onClick={() => setVista('solicitudes')}
            className={`px-4 py-2 rounded-xl flex items-center gap-2 font-black text-xs uppercase transition-all ${vista === 'solicitudes' ? 'bg-forest text-white shadow-sm' : 'text-slate-400'}`}
          >
            <ClipboardList size={15} /> Solicitudes
          </button>
          <button
            onClick={() => setVista('guarderias')}
            className={`px-4 py-2 rounded-xl flex items-center gap-2 font-black text-xs uppercase transition-all ${vista === 'guarderias' ? 'bg-forest text-white shadow-sm' : 'text-slate-400'}`}
          >
            <Building2 size={15} /> Guarderías
          </button>
          <button
            onClick={() => setVista('soporte')}
            className={`relative px-4 py-2 rounded-xl flex items-center gap-2 font-black text-xs uppercase transition-all ${vista === 'soporte' ? 'bg-forest text-white shadow-sm' : 'text-slate-400'}`}
          >
            <LifeBuoy size={15} /> Soporte
            {noLeidosSoporte > 0 && (
              <span className="bg-rose-500 text-white text-[9px] font-black w-4 h-4 rounded-full flex items-center justify-center">
                {noLeidosSoporte > 9 ? '9+' : noLeidosSoporte}
              </span>
            )}
          </button>
        </div>

        {vista === 'soporte' ? (
          <PanelSoportePlataforma apiUrl={API_URL} platformKey={key} />
        ) : vista === 'guarderias' ? (
          <div className="space-y-3">
            {cargandoGuarderias && guarderias === null ? (
              <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
                Cargando...
              </div>
            ) : guarderias?.length === 0 ? (
              <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
                Todavía no hay ninguna guardería dada de alta
              </div>
            ) : (
              guarderias?.map((g) => (
                <div key={g.id} className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm space-y-4">
                  <div className="flex items-start justify-between gap-4 flex-wrap">
                    <div>
                      <p className="font-black text-slate-900 text-lg">{g.nombre}</p>
                      <p className="text-[10px] font-black uppercase text-brand-600 tracking-widest mt-1">{g.slug}</p>
                      {g.direccion && (
                        <p className="text-xs text-slate-400 flex items-center gap-1.5 mt-2">
                          <MapPin size={12} className="shrink-0" /> {g.direccion}
                        </p>
                      )}
                    </div>
                    <div className="text-right shrink-0">
                      <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest">Dada de alta</p>
                      <p className="text-xs font-bold text-slate-600">{formatoFecha(g.creado_en)}</p>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    <div className="bg-slate-50 rounded-xl p-3 text-center">
                      <Baby size={16} className="mx-auto text-brand-500 mb-1" />
                      <p className="font-black text-slate-900 text-lg leading-none">{g.total_ninos}</p>
                      <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest mt-1">Niños</p>
                    </div>
                    <div className="bg-slate-50 rounded-xl p-3 text-center">
                      <Users size={16} className="mx-auto text-brand-500 mb-1" />
                      <p className="font-black text-slate-900 text-lg leading-none">{g.total_papas}</p>
                      <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest mt-1">Papás</p>
                    </div>
                    <div className="bg-slate-50 rounded-xl p-3 text-center">
                      <Users size={16} className="mx-auto text-slate-400 mb-1" />
                      <p className="font-black text-slate-900 text-lg leading-none">{g.total_staff}</p>
                      <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest mt-1">Staff/Admin</p>
                    </div>
                    <div className="bg-slate-50 rounded-xl p-3 text-center">
                      <Clock size={16} className="mx-auto text-slate-400 mb-1" />
                      <p className="font-black text-slate-900 text-[13px] leading-tight mt-0.5">{g.ultimo_acceso ? formatoFecha(g.ultimo_acceso) : 'Sin accesos'}</p>
                      <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest mt-1">Último acceso</p>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        ) : (
          <>
        {solicitudes?.length === 0 && (
          <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
            No hay solicitudes pendientes
          </div>
        )}

        <div className="space-y-3">
          {solicitudes?.map((s) => (
            <div key={s.id} className="bg-white p-6 rounded-[2rem] border border-slate-200 shadow-sm">
              <div className="flex items-start justify-between gap-4 flex-wrap">
                <div>
                  <p className="font-black text-slate-900 text-lg">{s.nombre_guarderia}</p>
                  {s.direccion && <p className="text-xs text-slate-400">{s.direccion}</p>}
                  <p className="text-xs text-slate-500 mt-2">{s.nombre_contacto} · {s.email_contacto}{s.telefono_contacto ? ` · ${s.telefono_contacto}` : ''}</p>
                  <p className="text-[10px] font-black uppercase text-brand-600 tracking-widest mt-2">Usuario: {s.username_deseado}</p>
                </div>
                <div className="flex gap-2 shrink-0">
                  <button onClick={() => rechazar(s)} className="px-4 py-2.5 rounded-xl bg-rose-50 text-rose-600 font-bold text-xs uppercase flex items-center gap-2 hover:bg-rose-100 transition-all">
                    <XCircle size={16} /> Rechazar
                  </button>
                  <button onClick={() => aprobar(s)} className="px-4 py-2.5 rounded-xl bg-emerald-500 text-white font-bold text-xs uppercase flex items-center gap-2 hover:bg-emerald-600 transition-all shadow-md">
                    <CheckCircle2 size={16} /> Aprobar
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
          </>
        )}
      </div>
    </div>
  );
};

export default PanelPlataforma;
