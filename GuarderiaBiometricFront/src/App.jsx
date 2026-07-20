import React, { useState, useRef, useEffect } from 'react';
import Webcam from 'react-webcam';
import api from './axiosConfig'; 
// Importar componentes de rutas
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { 
  UserPlus, ScanEye, Baby, AlertCircle, Users, Search, 
  ClipboardList, TrendingUp, ShieldCheck, ArrowRightCircle, 
  Lock, LogOut, CheckCircle, KeyRound, RefreshCw, X, Send, Clock, LogOut as LogOutIcon,
  User
} from 'lucide-react';

// Componentes secundarios
import GestionHijos from './GestionHijos';
import VistaBitacora from './VistaBitacora';
import PanelReportes from './PanelReportes';
import DashboardPadre from './DashboardPadre'; 
import ReportePublico from './ReportePublico'; // <-- Tu nueva ruta pública

const videoConstraints = {
  width: { ideal: 720 },
  height: { ideal: 1280 },
  facingMode: "user",
  aspectRatio: 0.75 
};

// --- TODO TU CÓDIGO ACTUAL SE MANTIENE AQUÍ DENTRO ---
function MainApp() {
  // --- ESTADOS DE AUTENTICACIÓN Y SESIÓN ---
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [userRole, setUserRole] = useState(null);
  const [userId, setUserId] = useState(null); 
  const [guarderiaInfo, setGuarderiaInfo] = useState({ nombre: '', slug: '' });
  const [loginUsername, setLoginUsername] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [tipoAcceso, setTipoAcceso] = useState('staff'); 
  
  const [tab, setTab] = useState('identificar');
  const [loading, setLoading] = useState(false);
  const [showAdminPinModal, setShowAdminPinModal] = useState(false);
  const [adminPin, setAdminPin] = useState('');
  const [tabPendiente, setTabPendiente] = useState(null);

  const webcamRef = useRef(null);
  const [nombre, setNombre] = useState(''); 
  const [resultado, setResultado] = useState(null);
  const [seleccionados, setSeleccionados] = useState([]);
  const [formAsistencia, setFormAsistencia] = useState({});

  const [padreSeleccionado, setPadreSeleccionado] = useState(null);
  const [tutoresEncontrados, setTutoresEncontrados] = useState([]);
  const [mostrarModalGestion, setMostrarModalGestion] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (token) {
      setIsLoggedIn(true);
      setUserRole(localStorage.getItem('role'));
      setUserId(localStorage.getItem('userId'));
      setGuarderiaInfo({ 
        nombre: localStorage.getItem('guarderia_nombre') || '', 
        slug: localStorage.getItem('guarderia_slug') || '' 
      });
    }
  }, []);

  useEffect(() => {
    if (tab === 'admin' && isLoggedIn) cargarTodosLosPadres();
  }, [tab, isLoggedIn]);

  const manejarLoginPrincipal = async (e) => {
    if (e) e.preventDefault();
    try {
      const res = await api.post('/login', { 
        username: loginUsername, 
        password: loginPassword,
        tipo: tipoAcceso 
      });

      if (res.data.token) {
        localStorage.setItem('token', res.data.token);
        localStorage.setItem('role', res.data.rol);
        localStorage.setItem('userId', res.data.userId);
        localStorage.setItem('guarderia_nombre', res.data.guarderia_nombre || '');
        localStorage.setItem('guarderia_slug', res.data.guarderia_slug || '');
        
        setIsLoggedIn(true);
        setUserRole(res.data.rol);
        setUserId(res.data.userId);
        setGuarderiaInfo({ 
          nombre: res.data.guarderia_nombre || '', 
          slug: res.data.guarderia_slug || '' 
        });
      }
    } catch (error) { 
      alert("Credenciales incorrectas para el perfil seleccionado"); 
    }
  };

  const cerrarSesion = () => {
    localStorage.clear();
    setIsLoggedIn(false);
    setUserRole(null);
    window.location.reload();
  };

  const cambiarTab = (targetTab) => {
    const tabsProtegidas = ['admin', 'bitacora', 'reportes'];
    if (tabsProtegidas.includes(targetTab) && userRole !== 'admin') {
      setTabPendiente(targetTab);
      setShowAdminPinModal(true);
    } else {
      setTab(targetTab);
      resetearProcesoEscaneo();
    }
  };

  const resetearProcesoEscaneo = () => {
    setResultado(null);
    setNombre('');
    setSeleccionados([]);
    setFormAsistencia({});
  };

  const verificarPinAdmin = async () => {
    try {
      const res = await api.post('/verificar-pin', { pin: adminPin });
      if (res.data.message === "PIN válido") {
        setTab(tabPendiente);
        setShowAdminPinModal(false);
        setAdminPin('');
      }
    } catch (error) { 
      alert("PIN incorrecto"); 
      setAdminPin(''); 
    }
  };

  const procesarRostro = async (endpoint) => {
    if (endpoint === 'registrar' && !nombre.trim()) {
      alert("⚠️ Error: No has escrito un nombre para el registro.");
      return;
    }
    if (!webcamRef.current) return;
    const imageSrc = webcamRef.current.getScreenshot();
    if (!imageSrc) return alert("No se pudo capturar la imagen.");

    setLoading(true);
    const base64Image = imageSrc.split(',')[1];
    try {
      const payload = { 
        imagen: base64Image, 
        collection_id: `guarderia-rostros`, 
        ...(endpoint === 'registrar' && { nombre }) 
      };
      const response = await api.post(`/${endpoint}`, payload);
      
      setResultado({ 
        type: 'success', 
        data: { 
          ...response.data, 
          nombre: endpoint === 'registrar' ? nombre : (response.data.nombre || response.data.padre) 
        } 
      });

      if (endpoint === 'registrar') {
        setPadreSeleccionado({ id: response.data.padre_id, nombre });
        setMostrarModalGestion(true);
      }
    } catch (error) { 
      setResultado({ type: 'error', msg: error.response?.data?.error || 'Rostro no reconocido' }); 
    } finally { 
      setLoading(false); 
    }
  };

  const manejarToggleHijo = (hijo) => {
    const hID = hijo.id || hijo.hijo_id;
    const estado = hijo.ultimo_estado;
    if (estado === "SALIDA") return;

    if (!seleccionados.includes(hID)) {
      if (estado === "ENTRADA") {
        const confirmar = window.confirm(`¿Deseas registrar la SALIDA de ${hijo.nombre_niño || hijo.nombre}?`);
        if (!confirmar) return;
      }
      setSeleccionados([...seleccionados, hID]);
    } else {
      setSeleccionados(seleccionados.filter(id => id !== hID));
    }
  };

  const registrarMultiplesAsistencias = async () => {
    if (seleccionados.length === 0) return;
    setLoading(true);
    try {
      const promesas = seleccionados.map(hijoId => {
        const hijoInfo = resultado.data.hijos.find(h => (h.id || h.hijo_id) === hijoId);
        const datos = formAsistencia[hijoId] || {};
        const esSalida = hijoInfo.ultimo_estado === "ENTRADA";
        
        return api.post('/confirmar-asistencia', {
          padre_id: resultado.data.padre_id || resultado.data.id,
          hijo_id: hijoId,
          aseado: esSalida ? false : (datos.aseado || false),
          reporte_golpe: esSalida ? false : (datos.golpes || false),
          observaciones: datos.observaciones || ""
        });
      });
      await Promise.all(promesas);
      alert("Operación exitosa");
      resetearProcesoEscaneo();
    } catch (error) { 
      alert("Error al registrar"); 
    } finally { 
      setLoading(false); 
    }
  };

  const cargarTodosLosPadres = async (query = '') => {
    try {
      const res = await api.get(`/buscar-padres?q=${query}`);
      setTutoresEncontrados(res.data || []);
    } catch (err) { console.error(err); }
  };

  if (!isLoggedIn) {
    return (
      <div className="min-h-screen bg-slate-100 flex items-center justify-center p-4">
        <div className="bg-white border border-slate-200 p-8 rounded-[2.5rem] w-full max-w-md shadow-xl text-center">
          <div className="inline-block bg-violet-600 p-4 rounded-3xl shadow-lg mb-6">
            <ShieldCheck size={40} className="text-white" />
          </div>
          <h1 className="text-3xl font-black text-slate-900 uppercase mb-2">BioSafe</h1>
          <p className="text-slate-500 font-bold uppercase text-[10px] tracking-widest mb-6">Selecciona tu perfil</p>
          
          <div className="flex bg-slate-100 p-1.5 rounded-2xl mb-8 border border-slate-200">
            <button 
              onClick={() => setTipoAcceso('staff')}
              className={`flex-1 py-2.5 rounded-xl flex items-center justify-center gap-2 font-black text-[10px] uppercase transition-all ${tipoAcceso === 'staff' ? 'bg-white text-violet-600 shadow-sm' : 'text-slate-400'}`}
            >
              <Users size={14}/> Staff / Admin
            </button>
            <button 
              onClick={() => setTipoAcceso('papa')}
              className={`flex-1 py-2.5 rounded-xl flex items-center justify-center gap-2 font-black text-[10px] uppercase transition-all ${tipoAcceso === 'papa' ? 'bg-white text-violet-600 shadow-sm' : 'text-slate-400'}`}
            >
              <Baby size={14}/> Soy Papá
            </button>
          </div>

          <form onSubmit={manejarLoginPrincipal} className="space-y-4">
            <input type="text" value={loginUsername} onChange={(e) => setLoginUsername(e.target.value)} className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-violet-500 transition-all" placeholder={tipoAcceso === 'staff' ? "Usuario" : "Correo electrónico"} />
            <input type="password" value={loginPassword} onChange={(e) => setLoginPassword(e.target.value)} className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-violet-500 transition-all" placeholder="••••••••" />
            <button type="submit" className="w-full bg-violet-600 hover:bg-violet-700 text-white font-black py-4 rounded-2xl uppercase tracking-tighter shadow-lg transition-all active:scale-95">
              Entrar al Panel
            </button>
          </form>
        </div>
      </div>
    );
  }

  if (userRole === 'papa') {
    return <DashboardPadre padreId={userId} alCerrarSesion={cerrarSesion} />;
  }

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 p-4 sm:p-6 lg:p-8">
      <header className="flex flex-col items-center mb-8 border-b border-slate-200 pb-6 gap-6 w-full">
        <div className="flex items-center gap-3">
          <div className="bg-violet-600 p-2 rounded-xl shadow-md"><ShieldCheck size={20} className="text-white" /></div>
          <div className="text-center sm:text-left">
            <h1 className="text-xl font-black uppercase leading-none text-slate-900">BioSafe</h1>
            <p className="text-[9px] text-violet-600 font-bold uppercase tracking-widest">{guarderiaInfo.nombre || 'Kiosk'}</p>
          </div>
        </div>

        <nav className="w-full max-w-full overflow-x-auto no-scrollbar">
          <div className="flex bg-white p-1.5 rounded-2xl border border-slate-200 shadow-sm min-w-max mx-auto">
            <button onClick={() => cambiarTab('identificar')} className={`px-4 py-2.5 rounded-xl flex items-center gap-2 font-bold transition-all ${tab === 'identificar' ? 'bg-violet-600 text-white shadow-md' : 'text-slate-500 hover:bg-slate-50'}`}><ScanEye size={18} /> Kiosco</button>
            <button onClick={() => cambiarTab('registrar')} className={`px-4 py-2.5 rounded-xl flex items-center gap-2 font-bold transition-all ${tab === 'registrar' ? 'bg-violet-600 text-white shadow-md' : 'text-slate-500 hover:bg-slate-50'}`}><UserPlus size={18} /> Registro</button>
            <button onClick={() => cambiarTab('admin')} className={`px-4 py-2.5 rounded-xl flex items-center gap-2 font-bold transition-all ${tab === 'admin' ? 'bg-violet-600 text-white shadow-md' : 'text-slate-500 hover:bg-slate-50'}`}><Users size={18} /> Familia</button>
            <button onClick={() => cambiarTab('bitacora')} className={`px-4 py-2.5 rounded-xl flex items-center gap-2 font-bold transition-all ${tab === 'bitacora' ? 'bg-violet-600 text-white shadow-md' : 'text-slate-500 hover:bg-slate-50'}`}><ClipboardList size={18} /> Bitácora</button>
            <button onClick={() => cambiarTab('reportes')} className={`px-4 py-2.5 rounded-xl flex items-center gap-2 font-bold transition-all ${tab === 'reportes' ? 'bg-emerald-600 text-white shadow-md' : 'text-slate-500 hover:bg-slate-50'}`}><TrendingUp size={18} /> Reportes</button>
            <button onClick={cerrarSesion} className="px-3 py-2 text-rose-500 hover:bg-rose-50 rounded-xl ml-2 border-l border-slate-100"><LogOut size={18} /></button>
          </div>
        </nav>
      </header>

      <main className="max-w-5xl mx-auto">
        {tab === 'reportes' && <PanelReportes />}
        {tab === 'bitacora' && <VistaBitacora />}
        {tab === 'admin' && (
          <div className="animate-in fade-in duration-500">
            <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
              <div className="flex items-center gap-4 mb-6">
                <div className="bg-violet-100 p-3 rounded-2xl text-violet-600"><Search size={28} /></div>
                <h3 className="text-xl font-black uppercase text-slate-900">Directorio de Tutores</h3>
              </div>
              <input type="text" placeholder="Buscar por nombre..." className="w-full bg-slate-50 border border-slate-200 p-5 rounded-2xl focus:ring-2 focus:ring-violet-500 outline-none text-slate-900 mb-6 transition-all" onChange={(e) => cargarTodosLosPadres(e.target.value)} />
              <div className="space-y-3 max-h-[60vh] overflow-y-auto pr-2 custom-scrollbar">
                {tutoresEncontrados.map(tutor => (
                  <button key={tutor.id} onClick={() => { setPadreSeleccionado(tutor); setMostrarModalGestion(true); }} className="w-full bg-slate-50 border border-slate-100 hover:bg-violet-50 p-4 rounded-xl flex justify-between items-center group transition-all text-left">
                    <div>
                      <p className="font-black uppercase text-slate-900">{tutor.nombre}</p>
                      <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">{tutor.hijos?.length || 0} hijos registrados</p>
                    </div>
                    <ArrowRightCircle size={20} className="text-slate-300 group-hover:text-violet-600" />
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}

        {(tab === 'identificar' || tab === 'registrar') && (
          <div className="flex flex-col items-center gap-8 animate-in fade-in duration-500">
            <div className="w-full max-w-md space-y-6">
               {tab === 'registrar' && (
                 <div className="space-y-2">
                   <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Nombre Completo del Tutor</label>
                   <input type="text" placeholder="Ej. Juan Pérez" value={nombre} onChange={(e) => setNombre(e.target.value)} className="w-full bg-white border border-slate-200 p-4 rounded-2xl text-slate-900 focus:ring-2 focus:ring-violet-500 outline-none shadow-sm" />
                 </div>
               )}
               <div className="relative rounded-[3.5rem] overflow-hidden border-8 border-white bg-slate-200 shadow-2xl aspect-[3/4] mx-auto w-full">
                  <Webcam audio={false} ref={webcamRef} screenshotFormat="image/jpeg" videoConstraints={videoConstraints} className="absolute inset-0 w-full h-full object-cover" mirrored={true} />
                  {loading && <div className="absolute inset-0 bg-white/60 backdrop-blur-sm flex items-center justify-center z-20"><RefreshCw className="animate-spin text-violet-600" size={54} /></div>}
               </div>
               <button onClick={() => procesarRostro(tab)} disabled={loading} className="w-full py-6 bg-violet-600 hover:bg-violet-700 text-white rounded-[2rem] font-black uppercase text-xl shadow-lg active:scale-95 transition-all disabled:opacity-50">
                 {loading ? 'Procesando...' : (tab === 'registrar' ? 'Confirmar Registro' : 'Escanear Rostro')}
               </button>
            </div>
            <div className="w-full max-w-md">
               {resultado ? (
                 <div className={`p-6 sm:p-8 rounded-[3rem] border-2 bg-white shadow-xl animate-in zoom-in duration-300 ${resultado.type === 'success' ? 'border-emerald-100' : 'border-rose-100'}`}>
                    <div className="flex items-center gap-4 mb-6">
                      {resultado.type === 'success' ? <CheckCircle className="text-emerald-500" size={28} /> : <AlertCircle className="text-rose-500" size={28} />}
                      <h3 className="text-xl font-black uppercase text-slate-900">{resultado.type === 'success' ? 'Identificado' : 'Aviso'}</h3>
                    </div>
                    {resultado.type === 'success' ? (
                      <div className="space-y-6">
                        <div className="bg-slate-50 p-5 rounded-2xl border border-slate-100">
                          <p className="text-violet-600 text-[10px] font-black uppercase mb-1 tracking-widest">Tutor</p>
                          <p className="text-xl font-bold text-slate-900 uppercase">{resultado.data.nombre}</p>
                        </div>
                        {resultado.data.hijos?.length > 0 ? (
                          <div className="space-y-4">
                            {resultado.data.hijos.map((h, i) => {
                              const hID = h.id || h.hijo_id;
                              const estaSeleccionado = seleccionados.includes(hID);
                              const estado = h.ultimo_estado || "AUSENTE";
                              const yaSalio = estado === "SALIDA";
                              const estaDentro = estado === "ENTRADA";
                              return (
                                <div key={i} className={`p-4 rounded-2xl border transition-all ${yaSalio ? 'opacity-50 bg-slate-100' : estaSeleccionado ? 'bg-violet-50 border-violet-200 shadow-sm' : 'bg-white'}`}>
                                  <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <input type="checkbox" disabled={yaSalio} checked={estaSeleccionado} onChange={() => manejarToggleHijo(h)} className="w-6 h-6 rounded-lg accent-violet-600" />
                                        <span className={`font-bold uppercase text-sm ${yaSalio ? 'line-through' : 'text-slate-700'}`}>{h.nombre_niño || h.nombre}</span>
                                    </div>
                                    <span className={`text-[9px] font-black px-2 py-1 rounded-lg ${estaDentro ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-200 text-slate-500'}`}>{estado}</span>
                                  </div>
                                  {estaSeleccionado && !estaDentro && !yaSalio && (
                                    <div className="mt-4 space-y-3 pt-4 border-t border-violet-100">
                                        <div className="grid grid-cols-2 gap-2">
                                            <button onClick={() => setFormAsistencia({...formAsistencia, [hID]: {...formAsistencia[hID], aseado: !formAsistencia[hID]?.aseado}})} className={`py-3 rounded-xl text-[10px] font-black uppercase border ${formAsistencia[hID]?.aseado ? 'bg-emerald-500 text-white' : 'bg-white text-slate-400'}`}>{formAsistencia[hID]?.aseado ? 'Aseado ✓' : '¿Aseado?'}</button>
                                            <button onClick={() => setFormAsistencia({...formAsistencia, [hID]: {...formAsistencia[hID], golpes: !formAsistencia[hID]?.golpes}})} className={`py-3 rounded-xl text-[10px] font-black uppercase border ${formAsistencia[hID]?.golpes ? 'bg-rose-500 text-white' : 'bg-white text-slate-400'}`}>{formAsistencia[hID]?.golpes ? 'Golpes !' : '¿Golpes?'}</button>
                                        </div>
                                        <input type="text" placeholder="Observaciones..." className="w-full bg-slate-50 border border-slate-200 p-3 rounded-xl text-xs outline-none" onChange={(e) => setFormAsistencia({...formAsistencia, [hID]: {...formAsistencia[hID], observaciones: e.target.value}})} />
                                    </div>
                                  )}
                                  {estaSeleccionado && estaDentro && (
                                    <div className="mt-3 p-3 bg-blue-50 rounded-xl border border-blue-100 flex items-center gap-2">
                                        <LogOutIcon size={14} className="text-blue-600" />
                                        <span className="text-[10px] font-bold text-blue-700 uppercase">Listo para SALIDA</span>
                                    </div>
                                  )}
                                </div>
                              );
                            })}
                            <button onClick={registrarMultiplesAsistencias} disabled={seleccionados.length === 0} className="w-full py-5 bg-slate-900 text-white rounded-2xl font-black uppercase text-sm shadow-lg flex items-center justify-center gap-3 disabled:opacity-50"><Send size={18} /> Confirmar {seleccionados.length} Movimiento(s)</button>
                          </div>
                        ) : (
                          <div className="text-center p-6 bg-amber-50 rounded-2xl border border-amber-100">
                             <Baby size={32} className="mx-auto text-amber-400 mb-2" /><p className="text-sm font-bold text-amber-800">Sin niños asignados</p>
                          </div>
                        )}
                      </div>
                    ) : (
                      <div className="space-y-4">
                        <p className="text-rose-600 font-bold bg-rose-50 p-4 rounded-xl text-center">{resultado.msg}</p>
                        <button onClick={() => setResultado(null)} className="w-full py-3 text-slate-400 font-bold uppercase text-[10px]">Reintentar</button>
                      </div>
                    )}
                 </div>
               ) : (
                 <div className="flex flex-col items-center justify-center text-center p-12 border-2 border-dashed border-slate-300 rounded-[3rem] text-slate-400 bg-white">
                    <ScanEye size={48} className="mb-4 opacity-20" /><p className="font-bold uppercase text-[10px] tracking-widest">Esperando detección...</p>
                 </div>
               )}
            </div>
          </div>
        )}
      </main>

      {mostrarModalGestion && padreSeleccionado && (
        <div className="fixed inset-0 z-[150] flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-md">
          <div className="bg-white w-full max-w-4xl rounded-[2.5rem] shadow-2xl overflow-hidden relative border-t-8 border-t-violet-600 max-h-[90vh] flex flex-col">
            <button onClick={() => { setMostrarModalGestion(false); cargarTodosLosPadres(); }} className="absolute top-6 right-6 text-slate-400 hover:text-slate-600 z-10 p-2"><X size={32} /></button>
            <div className="overflow-y-auto flex-1">
              <GestionHijos padreId={padreSeleccionado.id} nombrePadre={padreSeleccionado.nombre} onFinalizar={() => { setMostrarModalGestion(false); cargarTodosLosPadres(); }} />
            </div>
          </div>
        </div>
      )}

      {showAdminPinModal && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center p-4 bg-slate-900/70 backdrop-blur-sm">
          <div className="bg-white p-8 rounded-[2rem] w-full max-w-sm text-center shadow-2xl">
            <KeyRound size={40} className="mx-auto mb-4 text-amber-500" />
            <h2 className="text-xl font-black text-slate-900 uppercase mb-6">Acceso Protegido</h2>
            <input type="password" value={adminPin} onChange={(e) => setAdminPin(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && verificarPinAdmin()} className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-center text-3xl mb-6 outline-none" autoFocus placeholder="****" />
            <div className="flex gap-2">
              <button onClick={() => setShowAdminPinModal(false)} className="flex-1 py-3 text-slate-500 font-bold uppercase text-xs">Cancelar</button>
              <button onClick={verificarPinAdmin} className="flex-1 bg-amber-500 py-3 rounded-xl font-black text-white uppercase">Validar</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// --- ESTO ES LO ÚNICO NUEVO: EL ENRUTADOR ---
function App() {
  return (
    <Router>
      <Routes>
        {/* Ruta para el reporte del padre (pública) */}
        <Route path="/seguimiento/:token" element={<ReportePublico />} />

        {/* Todas las demás rutas cargan tu app administrativa original */}
        <Route path="/*" element={<MainApp />} />

        {/* Redirección por defecto si algo sale mal */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  );
}

export default App;