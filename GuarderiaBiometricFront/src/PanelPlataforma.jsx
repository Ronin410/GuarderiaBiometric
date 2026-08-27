import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { KeyRound, CheckCircle2, XCircle, Building2, RefreshCw, LogOut } from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';

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
  const [solicitudes, setSolicitudes] = useState(null);
  const [cargando, setCargando] = useState(false);
  const [errorKey, setErrorKey] = useState('');

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

  useEffect(() => {
    if (key) cargar(key);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
            <h1 className="text-xl font-black text-slate-900">Solicitudes de guardería</h1>
          </div>
          <div className="flex gap-2">
            <button onClick={() => cargar()} className="p-2.5 bg-white border border-slate-200 rounded-xl text-slate-500 hover:text-brand-600" title="Actualizar">
              <RefreshCw size={18} className={cargando ? 'animate-spin' : ''} />
            </button>
            <button onClick={cerrarSesionPlataforma} className="p-2.5 bg-white border border-slate-200 rounded-xl text-rose-500 hover:bg-rose-50" title="Salir">
              <LogOut size={18} />
            </button>
          </div>
        </div>

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
      </div>
    </div>
  );
};

export default PanelPlataforma;
