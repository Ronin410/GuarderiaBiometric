import React, { useEffect, useMemo, useState } from 'react';
import axios from 'axios';
import { KeyRound, CheckCircle2, XCircle, Building2, RefreshCw, LogOut, Users, Baby, MapPin, Clock, ClipboardList, LifeBuoy, BellRing, Lock, Unlock, Search, Pencil, Save, X, GraduationCap, ShieldCheck, ScanFace } from 'lucide-react';
import { mostrarError, mostrarExito, confirmar } from './utils/alertas';
import { suscribirseAPush, desuscribirseDePush, suscripcionActiva, pushSoportado } from './utils/push';
import PanelSoportePlataforma from './PanelSoportePlataforma';

// Ruta de suscripción push de esta sesión -- distinta de la de un papá/
// staff (/push/suscribir): /plataforma se autentica con X-Platform-Key, no
// con el JWT normal, así que necesita su propio endpoint (ver
// registrarRutasPushPlataforma en el backend).
const RUTA_PUSH_PLATAFORMA = '/plataforma/push/suscribir';

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
  const [notifEstado, setNotifEstado] = useState('default'); // default | granted | activando | desactivando
  // Buscador de la pestaña "Guarderías" -- filtra en memoria sobre lo que
  // ya se cargó, no vuelve a pedirle nada al backend (la lista completa de
  // guarderías no es tan grande como para justificar un endpoint de
  // búsqueda aparte).
  const [busquedaGuarderia, setBusquedaGuarderia] = useState('');
  // Edición de nombre/dirección -- "algo de configuración, como cambiar el
  // nombre": editandoId distingue "ninguna tarjeta en edición" (null) de
  // "esta tarjeta sí", y los otros dos guardan el borrador mientras se
  // edita, sin tocar guarderias hasta que se guarda de verdad.
  const [editandoId, setEditandoId] = useState(null);
  const [editNombre, setEditNombre] = useState('');
  const [editDireccion, setEditDireccion] = useState('');
  const [guardandoEdicion, setGuardandoEdicion] = useState(false);

  const cabeceras = { headers: { 'X-Platform-Key': key } };

  // Instancia de axios con la llave de plataforma ya puesta -- suscribirseAPush/
  // desuscribirseDePush (utils/push.js) piden un objeto con .post/.delete
  // como el axios instance normal (api.js), no uno con headers sueltos por
  // llamada como el resto de este archivo.
  const apiPlataforma = useMemo(() => axios.create({ baseURL: API_URL, headers: { 'X-Platform-Key': key } }), [key]);

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

  // "Quiero que las notificaciones de los chats sean notificaciones push...
  // que lleguen al iniciar como mi cuenta de admin" -- revisa si este
  // navegador ya tiene una suscripción activa, igual que hace App.jsx para
  // el papá/staff.
  useEffect(() => {
    if (!key) return;
    suscripcionActiva().then((activa) => setNotifEstado(activa ? 'granted' : 'default'));
  }, [key]);

  const handleActivarNotificaciones = async () => {
    setNotifEstado('activando');
    try {
      const ok = await suscribirseAPush(apiPlataforma, RUTA_PUSH_PLATAFORMA);
      setNotifEstado(ok ? 'granted' : 'default');
      if (!ok) mostrarError('No se pudieron activar las notificaciones. Revisa los permisos de notificaciones de tu navegador.');
    } catch (err) {
      console.error('Error al activar notificaciones', err);
      setNotifEstado('default');
      mostrarError(err.message || 'No se pudieron activar las notificaciones. Inténtalo de nuevo.');
    }
  };

  const handleDesactivarNotificaciones = async () => {
    setNotifEstado('desactivando');
    try {
      const ok = await desuscribirseDePush(apiPlataforma, RUTA_PUSH_PLATAFORMA);
      setNotifEstado(ok ? 'default' : 'granted');
      if (!ok) mostrarError('No se pudieron desactivar las notificaciones. Inténtalo de nuevo.');
    } catch (err) {
      console.error('Error al desactivar notificaciones', err);
      setNotifEstado('granted');
      mostrarError('No se pudieron desactivar las notificaciones. Inténtalo de nuevo.');
    }
  };

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

  // "Quiero una forma de bloquear el acceso a una guardería, ya que si no
  // pagan manualmente quiero quitarles el acceso dándoles tiempo para que
  // paguen, pero eso lo haré manualmente" -- actualiza la lista en
  // memoria en vez de recargar todo el listado, para que el cambio se vea
  // al instante.
  const bloquearGuarderia = async (g) => {
    const ok = await confirmar(`¿Bloquear el acceso de "${g.nombre}"? Ningún admin, staff o papá de esa guardería podrá iniciar sesión hasta que la desbloquees.`, 'Bloquear guardería');
    if (!ok) return;
    try {
      await axios.post(`${API_URL}/plataforma/guarderias/${g.id}/bloquear`, {}, cabeceras);
      mostrarExito(`"${g.nombre}" quedó bloqueada.`);
      setGuarderias((prev) => prev.map((x) => (x.id === g.id ? { ...x, bloqueada: true, bloqueada_en: new Date().toISOString() } : x)));
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo bloquear la guardería.');
    }
  };

  // reiniciarReconocimientoFacial -- rescate para "creé una guardería
  // nueva y ya me reconoce de otra": guarderia_id es un SERIAL reciclable,
  // así que si una guardería anterior se borró de la base sin borrar
  // también su colección de Rekognition en AWS, el número puede
  // reaparecer con rostros viejos ya adentro. DESTRUCTIVO -- borra TODOS
  // los rostros ya enrolados en esta guardería en AWS, así que el
  // confirm() es deliberadamente explícito sobre eso.
  const reiniciarReconocimientoFacial = async (g) => {
    const ok = await confirmar(
      `Esto borra TODOS los rostros ya enrolados en "${g.nombre}" (quedarían sin poder entrar hasta volver a registrarse). Úsalo solo si esta guardería nunca ha logrado completar un registro.`,
      '¿Reiniciar el reconocimiento facial?'
    );
    if (!ok) return;
    try {
      await axios.post(`${API_URL}/plataforma/guarderias/${g.id}/reiniciar-reconocimiento-facial`, {}, cabeceras);
      mostrarExito(`Reconocimiento facial de "${g.nombre}" reiniciado.`);
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo reiniciar el reconocimiento facial.');
    }
  };

  const desbloquearGuarderia = async (g) => {
    const ok = await confirmar(`¿Devolver el acceso a "${g.nombre}"?`, 'Desbloquear guardería');
    if (!ok) return;
    try {
      await axios.post(`${API_URL}/plataforma/guarderias/${g.id}/desbloquear`, {}, cabeceras);
      mostrarExito(`"${g.nombre}" ya puede volver a iniciar sesión.`);
      setGuarderias((prev) => prev.map((x) => (x.id === g.id ? { ...x, bloqueada: false, bloqueada_en: null } : x)));
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo desbloquear la guardería.');
    }
  };

  const empezarEdicion = (g) => {
    setEditandoId(g.id);
    setEditNombre(g.nombre);
    setEditDireccion(g.direccion || '');
  };

  const cancelarEdicion = () => setEditandoId(null);

  const guardarEdicion = async (g) => {
    const nombre = editNombre.trim();
    if (!nombre) {
      mostrarError('El nombre no puede quedar vacío.');
      return;
    }
    setGuardandoEdicion(true);
    try {
      await axios.put(`${API_URL}/plataforma/guarderias/${g.id}`, { nombre, direccion: editDireccion.trim() }, cabeceras);
      setGuarderias((prev) => prev.map((x) => (x.id === g.id ? { ...x, nombre, direccion: editDireccion.trim() || null } : x)));
      setEditandoId(null);
      mostrarExito('Guardería actualizada.');
    } catch (err) {
      mostrarError(err.response?.data?.error || 'No se pudo actualizar la guardería.');
    } finally {
      setGuardandoEdicion(false);
    }
  };

  // guarderiasFiltradas -- el buscador filtra sobre lo ya cargado en
  // memoria, no vuelve a pedirle nada al backend.
  const guarderiasFiltradas = useMemo(() => {
    if (!guarderias) return guarderias;
    const q = busquedaGuarderia.trim().toLowerCase();
    if (!q) return guarderias;
    return guarderias.filter((g) => g.nombre.toLowerCase().includes(q) || g.slug.toLowerCase().includes(q));
  }, [guarderias, busquedaGuarderia]);

  // resumenGuarderias -- "quisiera saber estadísticas y todo eso": un
  // panorama agregado de TODA la plataforma arriba de la lista, en vez de
  // tener que sumar a mano guardería por guardería. Se calcula sobre
  // `guarderias` (todas), no sobre el filtro del buscador -- el resumen
  // siempre es el total real, buscar no debe hacerlo parecer más chico.
  const resumenGuarderias = useMemo(() => {
    if (!guarderias || guarderias.length === 0) return null;
    return guarderias.reduce(
      (acc, g) => ({
        total: acc.total + 1,
        bloqueadas: acc.bloqueadas + (g.bloqueada ? 1 : 0),
        ninos: acc.ninos + g.total_ninos,
        papas: acc.papas + g.total_papas,
        admin: acc.admin + g.total_admin,
        maestras: acc.maestras + g.total_maestras,
      }),
      { total: 0, bloqueadas: 0, ninos: 0, papas: 0, admin: 0, maestras: 0 }
    );
  }, [guarderias]);

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
            {pushSoportado() && (
              <button
                onClick={notifEstado === 'granted' ? handleDesactivarNotificaciones : handleActivarNotificaciones}
                disabled={notifEstado === 'activando' || notifEstado === 'desactivando'}
                className={`p-2.5 bg-white border border-slate-200 rounded-xl transition-all disabled:opacity-50 ${notifEstado === 'granted' ? 'text-emerald-500' : 'text-slate-500 hover:text-brand-600'}`}
                title={notifEstado === 'granted' ? 'Notificaciones activas' : 'Activar notificaciones'}
              >
                <BellRing size={18} />
              </button>
            )}
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
            {/* RESUMEN AGREGADO -- "quisiera saber estadísticas y todo eso"
                de toda la plataforma de un vistazo, no guardería por
                guardería. */}
            {resumenGuarderias && (
              <div className="grid grid-cols-3 sm:grid-cols-6 gap-2">
                <div className="bg-white border border-slate-200 rounded-2xl p-3 text-center">
                  <p className="font-black text-slate-900 text-xl leading-none">{resumenGuarderias.total}</p>
                  <p className="text-[8px] font-black uppercase text-slate-400 tracking-widest mt-1">Guarderías</p>
                </div>
                <div className="bg-white border border-slate-200 rounded-2xl p-3 text-center">
                  <p className={`font-black text-xl leading-none ${resumenGuarderias.bloqueadas > 0 ? 'text-rose-600' : 'text-slate-900'}`}>{resumenGuarderias.bloqueadas}</p>
                  <p className="text-[8px] font-black uppercase text-slate-400 tracking-widest mt-1">Bloqueadas</p>
                </div>
                <div className="bg-white border border-slate-200 rounded-2xl p-3 text-center">
                  <p className="font-black text-slate-900 text-xl leading-none">{resumenGuarderias.ninos}</p>
                  <p className="text-[8px] font-black uppercase text-slate-400 tracking-widest mt-1">Niños</p>
                </div>
                <div className="bg-white border border-slate-200 rounded-2xl p-3 text-center">
                  <p className="font-black text-slate-900 text-xl leading-none">{resumenGuarderias.papas}</p>
                  <p className="text-[8px] font-black uppercase text-slate-400 tracking-widest mt-1">Papás</p>
                </div>
                <div className="bg-white border border-slate-200 rounded-2xl p-3 text-center">
                  <p className="font-black text-slate-900 text-xl leading-none">{resumenGuarderias.admin}</p>
                  <p className="text-[8px] font-black uppercase text-slate-400 tracking-widest mt-1">Admins</p>
                </div>
                <div className="bg-white border border-slate-200 rounded-2xl p-3 text-center">
                  <p className="font-black text-slate-900 text-xl leading-none">{resumenGuarderias.maestras}</p>
                  <p className="text-[8px] font-black uppercase text-slate-400 tracking-widest mt-1">Maestras</p>
                </div>
              </div>
            )}

            {guarderias?.length > 0 && (
              <div className="relative">
                <Search size={16} className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-300" />
                <input
                  type="text"
                  value={busquedaGuarderia}
                  onChange={(e) => setBusquedaGuarderia(e.target.value)}
                  placeholder="Buscar por nombre..."
                  className="w-full bg-white border border-slate-200 rounded-2xl py-3 pl-11 pr-4 text-sm outline-none focus:ring-2 focus:ring-brand-500 transition-all"
                />
              </div>
            )}

            {cargandoGuarderias && guarderias === null ? (
              <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
                Cargando...
              </div>
            ) : guarderias?.length === 0 ? (
              <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
                Todavía no hay ninguna guardería dada de alta
              </div>
            ) : guarderiasFiltradas?.length === 0 ? (
              <div className="bg-white p-10 rounded-[2.5rem] border border-dashed border-slate-200 text-center text-slate-400 font-bold uppercase text-xs tracking-widest">
                Ninguna guardería coincide con "{busquedaGuarderia}"
              </div>
            ) : (
              guarderiasFiltradas?.map((g) => (
                <div key={g.id} className={`bg-white p-6 rounded-[2rem] border shadow-sm space-y-4 ${g.bloqueada ? 'border-rose-200' : 'border-slate-200'}`}>
                  <div className="flex items-start justify-between gap-4 flex-wrap">
                    <div className="flex-1 min-w-0">
                      {editandoId === g.id ? (
                        <div className="space-y-2 max-w-sm">
                          <input
                            type="text"
                            value={editNombre}
                            onChange={(e) => setEditNombre(e.target.value)}
                            placeholder="Nombre"
                            autoFocus
                            className="w-full bg-slate-50 border border-slate-200 rounded-xl p-2.5 text-sm font-black outline-none focus:ring-2 focus:ring-brand-500"
                          />
                          <input
                            type="text"
                            value={editDireccion}
                            onChange={(e) => setEditDireccion(e.target.value)}
                            placeholder="Dirección (opcional)"
                            className="w-full bg-slate-50 border border-slate-200 rounded-xl p-2.5 text-xs outline-none focus:ring-2 focus:ring-brand-500"
                          />
                          <div className="flex gap-2">
                            <button
                              onClick={() => guardarEdicion(g)}
                              disabled={guardandoEdicion}
                              className="px-3 py-2 rounded-xl bg-emerald-500 text-white font-bold text-[11px] uppercase flex items-center gap-1.5 hover:bg-emerald-600 transition-all disabled:opacity-50"
                            >
                              <Save size={13} /> Guardar
                            </button>
                            <button
                              onClick={cancelarEdicion}
                              className="px-3 py-2 rounded-xl bg-slate-100 text-slate-500 font-bold text-[11px] uppercase flex items-center gap-1.5 hover:bg-slate-200 transition-all"
                            >
                              <X size={13} /> Cancelar
                            </button>
                          </div>
                        </div>
                      ) : (
                        <>
                          <div className="flex items-center gap-2 flex-wrap">
                            <p className="font-black text-slate-900 text-lg">{g.nombre}</p>
                            {g.bloqueada && (
                              <span className="bg-rose-100 text-rose-700 text-[9px] font-black uppercase tracking-widest px-2 py-1 rounded-full flex items-center gap-1">
                                <Lock size={10} /> Bloqueada
                              </span>
                            )}
                            <button
                              onClick={() => empezarEdicion(g)}
                              className="text-slate-300 hover:text-brand-600 transition-colors"
                              title="Editar nombre/dirección"
                            >
                              <Pencil size={13} />
                            </button>
                          </div>
                          <p className="text-[10px] font-black uppercase text-brand-600 tracking-widest mt-1">#{g.id} · {g.slug}</p>
                          {g.direccion && (
                            <p className="text-xs text-slate-400 flex items-center gap-1.5 mt-2">
                              <MapPin size={12} className="shrink-0" /> {g.direccion}
                            </p>
                          )}
                          {g.bloqueada && g.bloqueada_en && (
                            <p className="text-[10px] text-rose-500 font-bold mt-1.5">Bloqueada desde {formatoFecha(g.bloqueada_en)}</p>
                          )}
                        </>
                      )}
                    </div>
                    <div className="flex flex-col items-end gap-2 shrink-0">
                      <div className="text-right">
                        <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest">Dada de alta</p>
                        <p className="text-xs font-bold text-slate-600">{formatoFecha(g.creado_en)}</p>
                      </div>
                      {g.bloqueada ? (
                        <button
                          onClick={() => desbloquearGuarderia(g)}
                          className="px-3.5 py-2 rounded-xl bg-emerald-50 text-emerald-700 font-bold text-[11px] uppercase flex items-center gap-1.5 hover:bg-emerald-100 transition-all"
                        >
                          <Unlock size={13} /> Desbloquear
                        </button>
                      ) : (
                        <button
                          onClick={() => bloquearGuarderia(g)}
                          className="px-3.5 py-2 rounded-xl bg-rose-50 text-rose-600 font-bold text-[11px] uppercase flex items-center gap-1.5 hover:bg-rose-100 transition-all"
                        >
                          <Lock size={13} /> Bloquear
                        </button>
                      )}
                      <button
                        onClick={() => reiniciarReconocimientoFacial(g)}
                        className="px-3.5 py-2 rounded-xl bg-amber-50 text-amber-700 font-bold text-[11px] uppercase flex items-center gap-1.5 hover:bg-amber-100 transition-all"
                        title="Borra todos los rostros ya enrolados en esta guardería -- úsalo solo si nunca ha logrado completar un registro"
                      >
                        <ScanFace size={13} /> Reiniciar facial
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
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
                      <ShieldCheck size={16} className="mx-auto text-slate-400 mb-1" />
                      <p className="font-black text-slate-900 text-lg leading-none">{g.total_admin}</p>
                      <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest mt-1">Admin</p>
                    </div>
                    <div className="bg-slate-50 rounded-xl p-3 text-center">
                      <GraduationCap size={16} className="mx-auto text-slate-400 mb-1" />
                      <p className="font-black text-slate-900 text-lg leading-none">{g.total_maestras}</p>
                      <p className="text-[9px] font-black uppercase text-slate-400 tracking-widest mt-1">Maestras</p>
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
