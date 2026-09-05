import React, { useState, useRef, useEffect, useCallback } from 'react';
import Webcam from 'react-webcam';
import api, { setCsrfToken } from './axiosConfig';
// Importar componentes de rutas
import { BrowserRouter as Router, Routes, Route, Navigate, useNavigate, useParams, Link } from 'react-router-dom';
import {
  UserPlus, ScanEye, Baby, AlertCircle, Users, Search,
  ClipboardList, TrendingUp, ArrowRightCircle,
  Lock, LogOut, CheckCircle, KeyRound, RefreshCw, X, Send, Clock, LogOut as LogOutIcon,
  User, IdCard, Wallet, BarChart3, ShieldCheck as ShieldCheckIcon, UserCog, UtensilsCrossed,
  CalendarDays, LayoutDashboard, Settings, Megaphone, MessageCircle, CalendarOff, Soup, ClipboardCheck,
  BookOpen, Menu, BellRing
} from 'lucide-react';

// Componentes secundarios
import GestionHijos from './GestionHijos';
import VistaBitacora from './VistaBitacora';
import PanelReportes from './PanelReportes';
import PanelPerfiles from './PanelPerfiles';
import PanelPagos from './PanelPagos';
import PanelEstadisticas from './PanelEstadisticas';
import PanelConfiguracion from './PanelConfiguracion';
import PanelPersonal from './PanelPersonal';
import PanelMenu from './PanelMenu';
import PanelCirculares from './PanelCirculares';
import PanelHorarios from './PanelHorarios';
import PanelChat from './PanelChat';
import PanelAusencias from './PanelAusencias';
import PanelCalendario from './PanelCalendario';
import PanelComedor from './PanelComedor';
import PanelEncuestas from './PanelEncuestas';
import DashboardPadre from './DashboardPadre';
import AvisoPrivacidadModal from './AvisoPrivacidadModal';
import { mostrarError, mostrarExito, mostrarAviso, confirmar as confirmarAccion } from './utils/alertas';
import { segundosHastaExpirar } from './utils/sesion';
import { suscribirseAPush, desuscribirseDePush, suscripcionActiva, pushSoportado } from './utils/push';
import ReportePublico from './ReportePublico'; // <-- Tu nueva ruta pública
import RegistroGuarderia from './RegistroGuarderia';
import PanelPlataforma from './PanelPlataforma';
import LandingPage from './LandingPage';
import TerminosCondiciones from './TerminosCondiciones';
import AvisoPrivacidadPasitos from './AvisoPrivacidadPasitos';
import SoporteChat from './SoporteChat';

const videoConstraints = {
  width: { ideal: 720 },
  height: { ideal: 1280 },
  facingMode: "user",
  aspectRatio: 0.75,
  // "el facial en la tablet necesita 40-50cm para enfocar la cara, se
  // siente muy lejos" -- pedir enfoque continuo explícito en vez de dejar
  // que el navegador arranque con el modo que traiga por default: en
  // tablets Android con cámara autofocus, Chrome a veces la deja fija en
  // el enfoque con el que abrió (normalmente el de distancia "selfie
  // normal", ~40-50cm) si nada pide lo contrario. Va en "advanced" para
  // que, si el navegador/cámara no reconoce focusMode, lo ignore sin
  // tronar el resto de la petición (así se comporta un constraint dentro
  // de advanced que no se puede cumplir, a diferencia de uno en el nivel
  // de arriba). En una tablet con lente de enfoque FIJO de verdad (muchas
  // de las baratas) esto no cambia nada -- ese mínimo de 40-50cm es una
  // limitación física del lente, ningún ajuste de software lo mueve.
  advanced: [{ focusMode: "continuous" }]
};

// "Después de poner el PIN quiero que solo pueda tener acceso unos 30 min o
// si no se lo volverá a pedir cada 30 minutos" -- el PIN del kiosco deja de
// ser válido para siempre una vez tecleado y pasa a tener esta duración,
// igual que la sesión misma (ver pinVerificadoEn más abajo).
const DURACION_PIN_MS = 30 * 60 * 1000;

// Mapea cada pestaña protegida a la clave de área que usa el backend
// (RequireArea) -- "admin" (nombre heredado de la URL, la pestaña se ve
// como "Familia" en el menú) es la única que no comparte literal con su
// área, para no confundirla con el rol "admin".
//
// "configuracion" NO va aquí a propósito -- a diferencia del resto, esa
// pestaña nunca es "personalizable por permisos": el staff no debe poder
// entrar nunca, ni siquiera si un admin se lo intentara conceder (mismo
// criterio que personal/horarios, un poco más abajo). El backend ya lo
// exige con RequireAdmin() en vez de RequireArea (ver privacidad.go y
// tipos_documento.go), esto solo evita ofrecerlo en la UI para empezar.
//
// Vive fuera del componente (no depende de props/estado) para que
// TABS_PROTEGIDAS sea una referencia estable entre renders -- si viviera
// adentro, sería un objeto/arreglo nuevo en cada render y cualquier
// useEffect que lo necesitara en sus dependencias se volvería a disparar
// en cada render sin motivo real.
const AREA_DE_TAB = {
  admin: 'familia', bitacora: 'bitacora', reportes: 'reportes',
  perfiles: 'perfiles', pagos: 'pagos', estadisticas: 'estadisticas',
  menu: 'menu', circulares: 'circulares',
};
const TABS_PROTEGIDAS = Object.keys(AREA_DE_TAB);

// --- TODO TU CÓDIGO ACTUAL SE MANTIENE AQUÍ DENTRO ---
function MainApp() {
  // --- ESTADOS DE AUTENTICACIÓN Y SESIÓN ---
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [userRole, setUserRole] = useState(null);
  const [userId, setUserId] = useState(null);
  const [username, setUsername] = useState('');
  const [guarderiaInfo, setGuarderiaInfo] = useState({ nombre: '', slug: '' });
  // La cookie con el JWT es httpOnly (invisible a JS) — mientras se
  // confirma si ya hay sesión activa (GET /me), se muestra un loading en
  // vez de saltar directo a la pantalla de login.
  const [sesionCargando, setSesionCargando] = useState(true);
  const [expiraEn, setExpiraEn] = useState(null);
  // null = cuenta sin permisos personalizados (comportamiento de siempre:
  // el PIN de la cuenta desbloquea TODAS las pestañas protegidas por
  // igual). Un array (incluso vacío) = la lista exacta de áreas que un
  // admin le concedió a esta cuenta de staff -- ver AreasPermiso en el
  // backend (personal.go). Para "admin" este valor no importa, siempre
  // tiene acceso completo.
  const [permisos, setPermisos] = useState(null);
  const [loginUsername, setLoginUsername] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [tipoAcceso, setTipoAcceso] = useState('staff');
  // Solo controla el drawer del menú lateral en pantallas angostas -- en
  // md+ el sidebar siempre está visible y este estado no se usa.
  const [sidebarAbierto, setSidebarAbierto] = useState(false);
  // "Si llega un mensaje a la guardería, el botón del menú de los mensajes
  // aparecerá un icono de cuántos chats han llegado sin leer" -- chats
  // (no mensajes sueltos) pendientes, se refresca por polling igual que el
  // resto de contadores de este panel.
  const [chatNoLeidos, setChatNoLeidos] = useState(0);
  // "A la guardería también le llegarán notificaciones" -- mismo estado
  // 'default'/'granted'/'activando'/'desactivando' que ya usa DashboardPadre
  // del lado del papá, ahora también para staff/admin.
  const [notifEstado, setNotifEstado] = useState('default');

  // La pestaña activa vive en la URL (/panel/:tab) en vez de en estado local:
  // así sobrevive a un refresh y el botón "atrás" del navegador funciona.
  const { tab: tabDeUrl } = useParams();
  const tab = tabDeUrl || 'identificar';
  const navigate = useNavigate();

  const [loading, setLoading] = useState(false);
  const [showAdminPinModal, setShowAdminPinModal] = useState(false);
  const [adminPin, setAdminPin] = useState('');
  const [tabPendiente, setTabPendiente] = useState(null);
  // pinVerificadoEn guarda CUÁNDO se validó el PIN (null = nunca, o ya
  // venció) en vez de un simple booleano -- así el PIN deja de ser válido
  // para siempre y pasa a tener una duración (DURACION_PIN_MS). Ver
  // pinVigente() y el useEffect que lo vigila más abajo.
  const [pinVerificadoEn, setPinVerificadoEn] = useState(null);

  // true si esta cuenta puede entrar a `tabProtegida` -- admin siempre;
  // staff sin permisos personalizados (permisos === null) también siempre
  // (el PIN es lo que decide, no esto); staff personalizado solo si el
  // área está en su lista. En useCallback (con sus dependencias reales:
  // userRole y permisos) para que sea una referencia estable entre
  // renders y se pueda declarar sin problema en las dependencias de los
  // useEffect que la usan más abajo. Va DESPUÉS de los useState de los que
  // depende (userRole/permisos/pinVerificadoEn ya declarados arriba) --
  // referenciarlos antes de su declaración revienta en tiempo de
  // ejecución ("Cannot access ... before initialization"), aunque
  // compile sin problema.
  const tienePermiso = useCallback((tabProtegida) => {
    if (userRole === 'admin') return true;
    if (permisos === null) return true;
    return permisos.includes(AREA_DE_TAB[tabProtegida]);
  }, [userRole, permisos]);

  // Quita del menú las pestañas protegidas que esta cuenta no tiene
  // concedidas -- solo tiene efecto en cuentas ya personalizadas
  // (permisos !== null); las demás pestañas (no protegidas, o cuentas sin
  // personalizar) se muestran igual que siempre.
  const filtrarProtegidos = (items) => items.filter(({ tab: t }) => !TABS_PROTEGIDAS.includes(t) || tienePermiso(t));

  // true si el PIN sigue vigente (se tecleó hace menos de DURACION_PIN_MS).
  // Solo aplica a cuentas sin personalizar (permisos === null) -- las
  // personalizadas nunca dependen del PIN, ver tienePermiso. Misma razón
  // de useCallback que en tienePermiso.
  const pinVigente = useCallback(
    () => pinVerificadoEn !== null && (Date.now() - pinVerificadoEn < DURACION_PIN_MS),
    [pinVerificadoEn]
  );

  const avisoExpiracionMostrado = useRef(false);
  const webcamRef = useRef(null);
  const resultadoRef = useRef(null);
  // "A veces prende la cámara y a veces no... me cambio a otra parte del
  // menú y regreso y ya no se ve" -- react-webcam sí libera la cámara al
  // desmontar y la vuelve a pedir al montar, pero en tablets Android el
  // hardware de cámara a veces tarda en soltarse (o el getUserMedia
  // simplemente no resuelve) y, como antes no había onUserMedia ni
  // onUserMediaError, esa falla quedaba MUDA: un rectángulo en blanco sin
  // ningún aviso ni forma de reintentar. camaraLista/camaraError hacen ese
  // estado visible; camaraKey fuerza un remount limpio del <Webcam> al
  // reintentar (desmonta el que quedó colgado y pide la cámara de cero).
  const [camaraLista, setCamaraLista] = useState(false);
  const [camaraError, setCamaraError] = useState(false);
  const [camaraKey, setCamaraKey] = useState(0);
  const [nombre, setNombre] = useState('');
  const [resultado, setResultado] = useState(null);
  const [seleccionados, setSeleccionados] = useState([]);
  const [formAsistencia, setFormAsistencia] = useState({});
  // Cuenta del portal del papá, opcional al registrar su rostro -- el tutor
  // está presente en el kiosco en ese momento, así que puede quedar lista de
  // una vez (ver el comentario de handleRegistrar en el backend).
  const [crearCuentaPortal, setCrearCuentaPortal] = useState(false);
  const [usernamePortal, setUsernamePortal] = useState('');
  const [passwordPortal, setPasswordPortal] = useState('');

  const [padreSeleccionado, setPadreSeleccionado] = useState(null);
  const [tutoresEncontrados, setTutoresEncontrados] = useState([]);
  const [mostrarModalGestion, setMostrarModalGestion] = useState(false);

  // Aviso de Privacidad: se consulta una vez y, si el tutor todavía no lo
  // aceptó en esta sesión del kiosco, se le muestra antes de enrolar su
  // rostro. Una vez aceptado no se vuelve a pedir hasta recargar/cerrar sesión.
  const [avisoPrivacidad, setAvisoPrivacidad] = useState(null);
  const [avisoAceptado, setAvisoAceptado] = useState(false);
  const [mostrarModalAviso, setMostrarModalAviso] = useState(false);

  // Restaura la sesión al montar: la cookie httpOnly viaja sola en la
  // petición, pero JS no puede leerla — por eso se le pregunta al backend
  // quién está logueado en vez de leer localStorage como antes.
  useEffect(() => {
    // Limpieza única de lo que haya quedado de sesiones previas a este
    // cambio (el JWT vivía en localStorage) — ya no se usa como mecanismo
    // de sesión, pero no debe quedar tirado ahí.
    localStorage.clear();

    api.get('/me')
      .then((res) => hidratarSesion(res.data))
      .catch(() => { /* sin sesión activa: se queda en la pantalla de login */ })
      .finally(() => setSesionCargando(false));
  }, []);

  const hidratarSesion = (data) => {
    // csrf_token viaja en el body de /login y /me, no en la cookie -- ver
    // el comentario en axiosConfig.js sobre por qué (frontend y backend en
    // dominios de verdad distintos, no solo puertos, cuando esto corre en
    // Render en vez de local).
    setCsrfToken(data.csrf_token);
    setIsLoggedIn(true);
    setUserRole(data.rol);
    setUserId(data.user_id);
    setUsername(data.username || '');
    setGuarderiaInfo({
      nombre: data.guarderia_nombre || '',
      slug: data.guarderia_slug || ''
    });
    setExpiraEn(data.expires_at || null);
    // data.permisos: null = sin personalizar, array = lista exacta de
    // áreas concedidas -- ver el comentario del estado `permisos`.
    setPermisos(data.permisos ?? null);
  };

  useEffect(() => {
    if (tab === 'admin' && isLoggedIn) cargarTodosLosPadres();
  }, [tab, isLoggedIn]);

  // Guard de la pestaña de admin: cambiarTab() ya pedía el PIN al hacer clic en
  // el menú, pero con rutas reales alguien podría escribir /panel/pagos
  // directamente en la URL (o recargar ahí) sin pasar por ese clic. Este efecto
  // cubre ese caso: si la pestaña activa (derivada de la URL) es protegida,
  // regresa al kiosco -- pidiendo el PIN (cuenta sin personalizar) o sin más
  // trámite (cuenta personalizada sin ese permiso: el backend la rechazaría
  // de todos modos, así que no tiene caso pedir un PIN que no cambiaría nada).
  useEffect(() => {
    if (!isLoggedIn || !TABS_PROTEGIDAS.includes(tab) || userRole === 'admin') return;
    if (permisos !== null) {
      if (!tienePermiso(tab)) navigate('/panel/identificar', { replace: true });
      return;
    }
    if (!pinVigente()) {
      setTabPendiente(tab);
      setShowAdminPinModal(true);
      navigate('/panel/identificar', { replace: true });
    }
  }, [tab, userRole, isLoggedIn, permisos, navigate, tienePermiso, pinVigente]);

  // Vigila el vencimiento del PIN mientras siga vigente -- "si no se lo
  // volverá a pedir cada 30 minutos". En vez de solo revisar la fecha la
  // próxima vez que alguien navegue, esto invalida pinVerificadoEn en
  // cuanto se cumplen los 30 min, lo que dispara de nuevo el guard de
  // arriba (mismo efecto que si nunca se hubiera tecleado el PIN): si la
  // pestaña activa es protegida, pide el PIN otra vez ahí mismo.
  useEffect(() => {
    if (pinVerificadoEn === null) return;
    const revisarPin = () => {
      if (Date.now() - pinVerificadoEn >= DURACION_PIN_MS) setPinVerificadoEn(null);
    };
    const intervalo = setInterval(revisarPin, 30000);
    return () => clearInterval(intervalo);
  }, [pinVerificadoEn]);

  const manejarLoginPrincipal = async (e) => {
    if (e) e.preventDefault();
    try {
      const res = await api.post('/login', {
        username: loginUsername,
        password: loginPassword,
        tipo: tipoAcceso
      });
      hidratarSesion(res.data);
    } catch (error) {
      console.error("Error en login:", error);
      // El backend ahora distingue "credenciales inválidas" de "esta cuenta
      // es de otro tipo de perfil" (ver auth.go) -- se muestra tal cual en
      // vez del genérico de antes, que dejaba pensar que la contraseña
      // estaba mal cuando en realidad era la pestaña equivocada.
      mostrarError(error.response?.data?.error || "Credenciales incorrectas para el perfil seleccionado");
    }
  };

  const cerrarSesion = async () => {
    try {
      await api.post('/logout');
    } catch (error) {
      console.error("Error al cerrar sesión:", error);
    }
    setCsrfToken(null);
    setIsLoggedIn(false);
    setUserRole(null);
    window.location.reload();
  };

  // Aviso proactivo de expiración: en vez de esperar a que una petición falle
  // con 401 (interceptor de axiosConfig.js), revisamos cada minuto cuánto
  // falta para expiraEn (que manda el backend en /login y /me) para avisar
  // antes de que la sesión muera a medio trabajo.
  useEffect(() => {
    if (!isLoggedIn) return;

    const revisarExpiracion = () => {
      const restantes = segundosHastaExpirar(expiraEn);
      if (restantes === null) return;

      if (restantes <= 0) {
        mostrarAviso('Tu sesión expiró por inactividad. Vuelve a iniciar sesión.', 'Sesión expirada')
          .then(() => cerrarSesion());
      } else if (restantes <= 300 && !avisoExpiracionMostrado.current) {
        avisoExpiracionMostrado.current = true;
        mostrarAviso('Tu sesión está por expirar en unos minutos. Guarda cualquier cambio pendiente.', 'Sesión por expirar');
      }
    };

    revisarExpiracion();
    const intervalo = setInterval(revisarExpiracion, 60000);
    return () => clearInterval(intervalo);
  }, [isLoggedIn, expiraEn]);

  // Icono de chats sin leer en el menú lateral -- se revisa al entrar y
  // luego por polling, igual que la expiración de sesión de arriba.
  useEffect(() => {
    if (!isLoggedIn) return;

    const revisarNoLeidos = async () => {
      try {
        const res = await api.get('/chat/no-leidos');
        setChatNoLeidos(res.data?.no_leidos || 0);
      } catch (err) {
        console.error('Error al revisar los chats sin leer', err);
      }
    };

    revisarNoLeidos();
    const intervalo = setInterval(revisarNoLeidos, 30000);
    return () => clearInterval(intervalo);
  }, [isLoggedIn]);

  // Aparte del resto (no debe frenar la carga de la sesión): revisa si ya
  // hay una suscripción push activa en este navegador.
  useEffect(() => {
    if (!isLoggedIn) return;
    suscripcionActiva().then((activa) => setNotifEstado(activa ? 'granted' : 'default'));
  }, [isLoggedIn]);

  const handleActivarNotificaciones = async () => {
    setNotifEstado('activando');
    try {
      const ok = await suscribirseAPush(api);
      setNotifEstado(ok ? 'granted' : 'default');
      if (!ok) {
        mostrarError('No se pudieron activar las notificaciones. Revisa los permisos de notificaciones de tu navegador.');
      }
    } catch (err) {
      console.error('Error al activar notificaciones', err);
      setNotifEstado('default');
      mostrarError(err.message || 'No se pudieron activar las notificaciones. Inténtalo de nuevo.');
    }
  };

  const handleDesactivarNotificaciones = async () => {
    setNotifEstado('desactivando');
    try {
      const ok = await desuscribirseDePush(api);
      setNotifEstado(ok ? 'default' : 'granted');
      if (!ok) {
        mostrarError('No se pudieron desactivar las notificaciones. Inténtalo de nuevo.');
      }
    } catch (err) {
      console.error('Error al desactivar notificaciones', err);
      setNotifEstado('granted');
      mostrarError('No se pudieron desactivar las notificaciones. Inténtalo de nuevo.');
    }
  };

  const cambiarTab = (targetTab) => {
    if (TABS_PROTEGIDAS.includes(targetTab) && userRole !== 'admin') {
      if (permisos !== null) {
        // Cuenta personalizada: ya sabemos si puede entrar sin preguntar
        // nada (el menú ya le esconde las que no tiene, pero cubrimos
        // igual una URL escrita a mano).
        if (!tienePermiso(targetTab)) {
          mostrarError('Tu cuenta no tiene permiso para acceder a esta sección. Pide a un administrador que te lo habilite.');
          return;
        }
      } else if (!pinVigente()) {
        setTabPendiente(targetTab);
        setShowAdminPinModal(true);
        return;
      }
    }
    navigate('/panel/' + targetTab);
    resetearProcesoEscaneo();
  };

  const resetearProcesoEscaneo = () => {
    setResultado(null);
    setNombre('');
    setSeleccionados([]);
    setFormAsistencia({});
    setCrearCuentaPortal(false);
    setUsernamePortal('');
    setPasswordPortal('');
  };

  const verificarPinAdmin = async () => {
    try {
      const res = await api.post('/verificar-pin', { pin: adminPin });
      if (res.data.valid) {
        setPinVerificadoEn(Date.now());
        navigate('/panel/' + tabPendiente);
        setShowAdminPinModal(false);
        setAdminPin('');
      }
    } catch (error) {
      console.error("Error al verificar PIN:", error);
      mostrarError("PIN incorrecto");
      setAdminPin('');
    }
  };

  const procesarRostro = async (endpoint) => {
    if (endpoint === 'registrar' && !nombre.trim()) {
      mostrarError("No has escrito un nombre para el registro.");
      return;
    }
    if (endpoint === 'registrar' && crearCuentaPortal) {
      if (usernamePortal.trim().length < 3) {
        mostrarError("El usuario del portal debe tener al menos 3 caracteres.");
        return;
      }
      if (passwordPortal.length < 8) {
        mostrarError("La contraseña del portal debe tener al menos 8 caracteres.");
        return;
      }
    }
    if (endpoint === 'registrar' && !avisoAceptado) {
      await verificarYMostrarAvisoPrivacidad();
      return;
    }
    await capturarYEnviar(endpoint);
  };

  // Consulta (una sola vez por sesión de kiosco) si la guardería ya
  // configuró su Aviso de Privacidad. Sin texto configurado, se bloquea el
  // registro: no hay evidencia de consentimiento posible para mostrar.
  const verificarYMostrarAvisoPrivacidad = async () => {
    try {
      let aviso = avisoPrivacidad;
      if (!aviso) {
        const res = await api.get('/aviso-privacidad');
        aviso = res.data;
        setAvisoPrivacidad(aviso);
      }
      if (!aviso.configurado) {
        mostrarError("El administrador debe configurar el Aviso de Privacidad (pestaña Configuración) antes de poder registrar tutores.");
        return;
      }
      setMostrarModalAviso(true);
    } catch (error) {
      console.error("Error al consultar el Aviso de Privacidad:", error);
      mostrarError("No se pudo cargar el Aviso de Privacidad.");
    }
  };

  const manejarAceptarAvisoPrivacidad = () => {
    setAvisoAceptado(true);
    setMostrarModalAviso(false);
    capturarYEnviar('registrar');
  };

  // "Quiero que la pantalla vaya hacia abajo cuando sale más info o el
  // error" -- el cuadro de la cámara ahora ocupa casi toda la pantalla
  // (ver más abajo), así que el resultado (identificado o el aviso de
  // error) queda fuera de la vista hasta que alguien baje solo; esto lo
  // lleva ahí automáticamente en cuanto aparece.
  useEffect(() => {
    if (resultado) {
      resultadoRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [resultado]);

  // mostrarCamara se queda en true al alternar entre "identificar" y
  // "registrar" (los dos comparten el mismo <Webcam>, nunca se desmonta
  // entre ellos) -- por eso el efecto de abajo usa ESTE booleano como
  // dependencia y no `tab` directo: solo debe reiniciar el estado de la
  // cámara cuando de verdad se sale y se vuelve a esta vista, no en cada
  // clic entre esas dos pestañas.
  const mostrarCamara = tab === 'identificar' || tab === 'registrar';
  useEffect(() => {
    if (!mostrarCamara) return;
    setCamaraLista(false);
    setCamaraError(false);
  }, [mostrarCamara, camaraKey]);

  // Si onUserMedia/onUserMediaError nunca llegan a dispararse (pasa en
  // algunas tablets Android: el hardware de cámara se queda "colgado" sin
  // resolver ni rechazar la promesa de getUserMedia), esto lo trata igual
  // que un error después de un rato en vez de dejarlo en blanco para
  // siempre sin ningún aviso.
  useEffect(() => {
    if (!mostrarCamara) return;
    const limite = setTimeout(() => {
      setCamaraLista((listo) => {
        if (!listo) setCamaraError(true);
        return listo;
      });
    }, 6000);
    return () => clearTimeout(limite);
  }, [mostrarCamara, camaraKey]);

  const reintentarCamara = () => {
    // Cambiar la key de <Webcam> obliga a React a desmontarla y volver a
    // montarla de cero -- suelta cualquier stream que se hubiera quedado a
    // medias y hace un getUserMedia nuevo, en vez de reintentar sobre el
    // mismo componente que ya falló.
    setCamaraKey((k) => k + 1);
  };

  // ajustarEnfoqueCercano -- "el facial necesita 40-50cm para enfocar,
  // ¿se puede hacer algo para que detecte más cerca?" -- el constraint
  // `advanced: [{ focusMode: "continuous" }]` de arriba ya pide enfoque
  // continuo al abrir la cámara, pero algunas tablets Android SÍ traen
  // lente autofocus y exponen un control de distancia manual
  // (focusDistance) que el navegador no usa por default. Aquí, ya con el
  // stream abierto, se checa si el track realmente ofrece esos controles
  // (getCapabilities) y si es así se manda el focusDistance más chico que
  // permita (el valor "min" de su rango = lo más cerca que puede enfocar)
  // -- en vez de solo pedir "continuous" y esperar. Si el navegador o la
  // cámara no exponen nada de esto (la mayoría, sobre todo tablets
  // baratas con lente de enfoque FIJO) simplemente no hay capabilities
  // que ajustar y esto no hace nada -- ahí el mínimo de 40-50cm es una
  // limitación física del lente que ningún ajuste de software puede
  // mover.
  const ajustarEnfoqueCercano = (stream) => {
    try {
      const track = stream?.getVideoTracks?.()[0];
      if (!track || typeof track.getCapabilities !== 'function') return;
      const capacidades = track.getCapabilities();
      if (!capacidades) return;

      const avanzados = [];
      if (capacidades.focusMode?.includes?.('continuous')) {
        avanzados.push({ focusMode: 'continuous' });
      } else if (capacidades.focusMode?.includes?.('manual') && capacidades.focusDistance) {
        avanzados.push({ focusMode: 'manual', focusDistance: capacidades.focusDistance.min });
      }
      if (avanzados.length > 0) {
        track.applyConstraints({ advanced: avanzados }).catch(() => {});
      }
    } catch {
      // Sin soporte de la API de capabilities en este navegador -- se
      // deja la cámara con lo que haya arrancado por default.
    }
  };

  const capturarYEnviar = async (endpoint) => {
    if (!webcamRef.current) return;
    const imageSrc = webcamRef.current.getScreenshot();
    if (!imageSrc) return mostrarError("No se pudo capturar la imagen.");

    setLoading(true);
    const base64Image = imageSrc.split(',')[1];
    try {
      const payload = {
        imagen: base64Image,
        ...(endpoint === 'registrar' && {
          nombre, acepta_aviso: true,
          ...(crearCuentaPortal && { crear_cuenta: true, username: usernamePortal.trim(), password: passwordPortal }),
        })
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
        if (response.data.cuenta_error) {
          mostrarError(response.data.cuenta_error);
        } else if (response.data.cuenta_creada) {
          mostrarExito('Rostro y cuenta del portal creados correctamente.');
        }
        setPadreSeleccionado({ id: response.data.padre_id, nombre });
        setMostrarModalGestion(true);
      }
    } catch (error) { 
      setResultado({ type: 'error', msg: error.response?.data?.error || 'Rostro no reconocido' }); 
    } finally { 
      setLoading(false); 
    }
  };

  const manejarToggleHijo = async (hijo) => {
    const hID = hijo.id || hijo.hijo_id;
    const estado = hijo.ultimo_estado;
    if (estado === "SALIDA") return;

    if (!seleccionados.includes(hID)) {
      if (estado === "ENTRADA") {
        const ok = await confirmarAccion(`¿Deseas registrar la SALIDA de ${hijo.nombre_niño || hijo.nombre}?`, "Confirmar salida");
        if (!ok) return;
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
      mostrarExito("Operación exitosa");
      resetearProcesoEscaneo();
    } catch (error) {
      console.error("Error al registrar asistencia:", error);
      mostrarError("Error al registrar");
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

  // Mientras se confirma (GET /me) si la cookie httpOnly corresponde a una
  // sesión activa, se muestra este loading en vez de saltar directo al
  // formulario de login — evita el parpadeo de "no hay sesión" seguido de
  // "sí la hay" en cada recarga.
  if (sesionCargando) {
    return (
      <div className="min-h-screen bg-paper flex items-center justify-center p-4">
        <RefreshCw className="animate-spin text-brand-600" size={40} />
      </div>
    );
  }

  if (!isLoggedIn) {
    return (
      <div className="min-h-screen bg-paper flex items-center justify-center p-4">
        <div className="bg-white border border-slate-200 p-8 rounded-[2.5rem] w-full max-w-md shadow-xl text-center">
          <img src="/dinos/logo-pasitos.png" alt="Pasitos" className="h-20 w-auto mx-auto mb-4" />
          <h1 className="text-3xl font-black text-ink uppercase mb-2">Pasitos</h1>
          <p className="text-slate-500 font-bold uppercase text-[10px] tracking-widest mb-6">Selecciona tu perfil</p>
          
          <div className="flex bg-slate-100 p-1.5 rounded-2xl mb-8 border border-slate-200">
            <button 
              onClick={() => setTipoAcceso('staff')}
              className={`flex-1 py-2.5 rounded-xl flex items-center justify-center gap-2 font-black text-[10px] uppercase transition-all ${tipoAcceso === 'staff' ? 'bg-white text-brand-600 shadow-sm' : 'text-slate-400'}`}
            >
              <Users size={14}/> Staff / Admin
            </button>
            <button 
              onClick={() => setTipoAcceso('papa')}
              className={`flex-1 py-2.5 rounded-xl flex items-center justify-center gap-2 font-black text-[10px] uppercase transition-all ${tipoAcceso === 'papa' ? 'bg-white text-brand-600 shadow-sm' : 'text-slate-400'}`}
            >
              <Baby size={14}/> Soy Papá
            </button>
          </div>

          <form onSubmit={manejarLoginPrincipal} className="space-y-4">
            <input type="text" value={loginUsername} onChange={(e) => setLoginUsername(e.target.value)} className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all" placeholder={tipoAcceso === 'staff' ? "Usuario" : "Correo electrónico"} />
            <input type="password" value={loginPassword} onChange={(e) => setLoginPassword(e.target.value)} className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all" placeholder="••••••••" />
            <button type="submit" className="w-full bg-brand-600 hover:bg-brand-700 text-white font-black py-4 rounded-2xl uppercase tracking-tighter shadow-lg transition-all active:scale-95">
              Entrar al Panel
            </button>
          </form>

          <Link to="/registro-guarderia" className="block mt-6 text-slate-400 hover:text-brand-600 font-bold text-[11px] uppercase tracking-widest transition-colors">
            ¿Tienes una guardería? Solicita tu alta
          </Link>
        </div>
      </div>
    );
  }

  if (userRole === 'papa') {
    return (
      <>
        <DashboardPadre padreId={userId} nombreUsuario={username} alCerrarSesion={cerrarSesion} />
        <SoporteChat modo="autenticado" />
      </>
    );
  }

  // Estructura del menú lateral: mismas pestañas y el mismo filtrado por
  // permisos (filtrarProtegidos/tienePermiso) que antes vivían repartidos
  // entre los NavDropdown -- solo cambia cómo se presentan (lista fija en
  // vez de menús desplegables).
  const seccionesNav = [
    {
      items: [
        { tab: 'identificar', label: 'Recepción', Icon: ScanEye },
        { tab: 'registrar', label: 'Registro', Icon: UserPlus },
      ],
    },
    {
      label: 'Alumnos',
      items: filtrarProtegidos([
        { tab: 'admin', label: 'Familia', Icon: Users },
        { tab: 'perfiles', label: 'Perfiles', Icon: IdCard },
      ]),
    },
    {
      label: 'Día a día',
      items: filtrarProtegidos([
        { tab: 'bitacora', label: 'Bitácora', Icon: ClipboardList },
        { tab: 'menu', label: 'Menú Semanal', Icon: UtensilsCrossed },
        { tab: 'circulares', label: 'Circulares', Icon: Megaphone },
        { tab: 'chat', label: 'Chat con Familias', Icon: MessageCircle },
        { tab: 'ausencias', label: 'Ausencias Avisadas', Icon: CalendarOff },
        { tab: 'calendario', label: 'Calendario Escolar', Icon: CalendarDays },
        { tab: 'comedor', label: 'Pedidos de Comedor', Icon: Soup },
        { tab: 'encuestas', label: 'Encuestas', Icon: ClipboardCheck },
      ]),
    },
    {
      label: 'Administración',
      items: filtrarProtegidos([
        { tab: 'reportes', label: 'Reportes', Icon: TrendingUp },
        { tab: 'pagos', label: 'Pagos', Icon: Wallet },
        { tab: 'estadisticas', label: 'Estadísticas', Icon: BarChart3 },
      ]),
    },
    {
      label: 'Sistema',
      // Exclusivo del admin -- el staff no debe ver ni entrar aquí nunca,
      // ni siquiera con permisos personalizados (ver el comentario largo de
      // AREA_DE_TAB más arriba).
      items: userRole === 'admin' ? [
        { tab: 'configuracion', label: 'Configuración', Icon: ShieldCheckIcon },
        { tab: 'personal', label: 'Personal', Icon: UserCog },
        { tab: 'horarios', label: 'Horarios de Personal', Icon: Clock },
      ] : [],
    },
  ].filter((seccion) => seccion.items.length > 0);

  const claseItemNav = (activo) =>
    `w-full flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-[13px] font-bold transition-all text-left ${
      activo ? 'bg-forest-light text-white shadow-sm' : 'text-white/70 hover:bg-white/10 hover:text-white'
    }`;

  const contenidoSidebar = (
    <>
      <div className="flex items-center gap-2.5 px-1 shrink-0">
        <img src="/dinos/logo-pasitos.png" alt="" className="h-9 w-auto shrink-0" />
        <span className="text-white font-black uppercase tracking-tight text-base">Pasitos</span>
      </div>

      <div className="flex-1 overflow-y-auto flex flex-col gap-5 mt-6 pr-1 custom-scrollbar-dark">
        {seccionesNav.map((seccion, i) => (
          <div key={i} className="space-y-1">
            {seccion.label && (
              <p className="text-[10px] font-black uppercase tracking-widest text-white/40 px-3.5 mb-1.5">{seccion.label}</p>
            )}
            {seccion.items.map(({ tab: t, label, Icon }) => (
              <button key={t} onClick={() => { cambiarTab(t); setSidebarAbierto(false); }} className={claseItemNav(tab === t)}>
                <Icon size={17} className="shrink-0" /> {label}
                {t === 'chat' && chatNoLeidos > 0 && (
                  <span className="ml-auto bg-rose-500 text-white text-[10px] font-black px-2 py-0.5 rounded-full shrink-0">{chatNoLeidos}</span>
                )}
              </button>
            ))}
          </div>
        ))}
      </div>

      <div className="mt-4 space-y-2 shrink-0">
        {pushSoportado() && (
          <button
            onClick={notifEstado === 'granted' ? handleDesactivarNotificaciones : handleActivarNotificaciones}
            disabled={notifEstado === 'activando' || notifEstado === 'desactivando'}
            className="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-[13px] font-bold text-white/70 hover:bg-white/10 hover:text-white transition-all disabled:opacity-50"
          >
            <BellRing size={17} className={`shrink-0 ${notifEstado === 'granted' ? 'text-emerald-300' : ''}`} />
            {notifEstado === 'activando' || notifEstado === 'desactivando'
              ? 'Un momento...'
              : notifEstado === 'granted' ? 'Notificaciones activas' : 'Activar notificaciones'}
          </button>
        )}
        <a
          href="/manual.html" target="_blank" rel="noopener noreferrer"
          className="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-[13px] font-bold text-white/70 hover:bg-white/10 hover:text-white transition-all"
        ><BookOpen size={17} className="shrink-0" /> Manual</a>
        <button onClick={cerrarSesion} className="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-[13px] font-bold text-rose-300 hover:bg-rose-500/10 hover:text-rose-200 transition-all">
          <LogOut size={17} className="shrink-0" /> Cerrar sesión
        </button>
        <div className="p-3 bg-forest-dark rounded-xl text-white/70 text-[11px] leading-relaxed">
          <p className="font-bold text-white mb-0.5 truncate">{guarderiaInfo.nombre || 'Guardería'}</p>
          Sesión: {username || 'staff'}
        </div>
      </div>
    </>
  );

  return (
    <div className="min-h-screen bg-paper text-ink flex">
      {/* MENÚ LATERAL — escritorio. "En la tablet grande el menú no se
          colapsa" -- md: por sí solo solo mira el ANCHO, y una tablet
          grande en vertical (kiosco real, igual que la chica) tiene más de
          768px de ancho aunque siga siendo un dispositivo angosto y alto,
          no una pantalla de escritorio. md:landscape: exige ADEMÁS
          orientación horizontal, que es la señal real de "esto es una
          pantalla de computadora" -- una tablet en vertical, sin importar
          su ancho, sigue usando el menú móvil de abajo. */}
      <aside className="hidden md:landscape:flex md:landscape:flex-col w-64 shrink-0 bg-forest p-5 sticky top-0 h-screen">
        {contenidoSidebar}
      </aside>

      {/* MENÚ LATERAL — móvil (se abre como cajón encima del contenido) */}
      {sidebarAbierto && (
        <div className="fixed inset-0 z-40 md:landscape:hidden flex">
          <div className="absolute inset-0 bg-slate-900/60" onClick={() => setSidebarAbierto(false)} />
          <aside className="relative w-72 max-w-[80vw] h-full bg-forest p-5 flex flex-col animate-in slide-in-from-left duration-200">
            {contenidoSidebar}
          </aside>
        </div>
      )}

      <div className="flex-1 min-w-0 flex flex-col">
        {/* BARRA SUPERIOR — se ve siempre que el sidebar de escritorio esté
            oculto (mismo criterio md:landscape: de arriba). */}
        <div className="md:landscape:hidden flex items-center justify-between px-4 py-3.5 bg-white border-b border-slate-200 sticky top-0 z-20">
          <button onClick={() => setSidebarAbierto(true)} className="p-1 text-ink" title="Abrir menú"><Menu size={22} /></button>
          <span className="font-black uppercase text-sm text-ink">Pasitos</span>
          <button onClick={cerrarSesion} className="p-1 text-rose-500" title="Cerrar sesión"><LogOut size={20} /></button>
        </div>

        <main className="flex-1 w-full max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 pb-4 sm:pb-6 lg:pb-8 pt-8 sm:pt-10 lg:pt-12">
        {tab === 'reportes' && <PanelReportes guarderiaInfo={guarderiaInfo} />}
        {tab === 'bitacora' && <VistaBitacora />}
        {tab === 'perfiles' && <PanelPerfiles />}
        {tab === 'pagos' && <PanelPagos />}
        {tab === 'estadisticas' && <PanelEstadisticas />}
        {tab === 'menu' && <PanelMenu />}
        {tab === 'circulares' && <PanelCirculares />}
        {tab === 'chat' && <PanelChat usuarioActualId={userId} />}
        {tab === 'ausencias' && <PanelAusencias />}
        {tab === 'calendario' && <PanelCalendario />}
        {tab === 'comedor' && <PanelComedor />}
        {tab === 'encuestas' && <PanelEncuestas />}
        {tab === 'configuracion' && userRole === 'admin' && (
          <PanelConfiguracion
            nombreGuarderia={guarderiaInfo.nombre}
            onNombreActualizado={(nombre) => setGuarderiaInfo((prev) => ({ ...prev, nombre }))}
          />
        )}
        {tab === 'personal' && userRole === 'admin' && <PanelPersonal usuarioActualId={userId} />}
        {tab === 'horarios' && userRole === 'admin' && <PanelHorarios />}
        {tab === 'admin' && (
          <div className="animate-in fade-in duration-500">
            <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
              <div className="flex items-center gap-4 mb-6">
                <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><Search size={28} /></div>
                <h3 className="text-xl font-black uppercase text-slate-900">Directorio de Tutores</h3>
              </div>
              <input type="text" placeholder="Buscar por nombre..." className="w-full bg-slate-50 border border-slate-200 p-5 rounded-2xl focus:ring-2 focus:ring-brand-500 outline-none text-slate-900 mb-6 transition-all" onChange={(e) => cargarTodosLosPadres(e.target.value)} />
              <div className="space-y-3 max-h-[60vh] overflow-y-auto pr-2 custom-scrollbar">
                {tutoresEncontrados.map(tutor => (
                  <button key={tutor.id} onClick={() => { setPadreSeleccionado(tutor); setMostrarModalGestion(true); }} className="w-full bg-slate-50 border border-slate-100 hover:bg-brand-50 p-4 rounded-xl flex justify-between items-center group transition-all text-left">
                    <div>
                      <p className="font-black uppercase text-slate-900">{tutor.nombre}</p>
                      <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">{tutor.hijos?.length || 0} hijos registrados</p>
                    </div>
                    <ArrowRightCircle size={20} className="text-slate-300 group-hover:text-brand-600" />
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}

        {(tab === 'identificar' || tab === 'registrar') && (
          <div className="flex flex-col items-center gap-8 animate-in fade-in duration-500">
            <div className="w-full space-y-6">
               {tab === 'registrar' && (
                 <div className="max-w-md mx-auto space-y-3">
                   <div className="space-y-2">
                     <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Nombre Completo del Tutor</label>
                     <input type="text" placeholder="Ej. Juan Pérez" value={nombre} onChange={(e) => setNombre(e.target.value)} className="w-full bg-white border border-slate-200 p-4 rounded-2xl text-slate-900 focus:ring-2 focus:ring-brand-500 outline-none shadow-sm" />
                   </div>

                   <label className="flex items-center gap-3 bg-white border border-slate-200 p-4 rounded-2xl shadow-sm cursor-pointer">
                     <input type="checkbox" checked={crearCuentaPortal} onChange={(e) => setCrearCuentaPortal(e.target.checked)} className="w-5 h-5 rounded-md accent-brand-600" />
                     <span className="text-xs font-bold text-slate-600">Crear también su cuenta del portal (el tutor está aquí)</span>
                   </label>

                   {crearCuentaPortal && (
                     <div className="grid grid-cols-2 gap-2 animate-in fade-in duration-200">
                       <input type="text" placeholder="Usuario" value={usernamePortal} onChange={(e) => setUsernamePortal(e.target.value)} className="w-full bg-white border border-slate-200 p-4 rounded-2xl text-slate-900 focus:ring-2 focus:ring-brand-500 outline-none shadow-sm" />
                       <input type="password" placeholder="Contraseña" value={passwordPortal} onChange={(e) => setPasswordPortal(e.target.value)} className="w-full bg-white border border-slate-200 p-4 rounded-2xl text-slate-900 focus:ring-2 focus:ring-brand-500 outline-none shadow-sm" />
                     </div>
                   )}
                 </div>
               )}
               {/* "Quiero que abarque toda la pantalla el cuadro del facial"
                   -- un min(100%, Xvh) fijo por ancho de pantalla se rompía
                   con una tablet grande EN VERTICAL: tiene más de 768px de
                   ancho (como una computadora) pero un alto de pantalla
                   modesto, así que el tope en vh la encogía igual que a una
                   computadora -- "en la grande se ve pequeño", aunque siga
                   siendo un kiosco en vertical como la chica. La señal
                   correcta no es el ancho sino la ORIENTACIÓN: en vertical
                   (cualquier tablet, grande o chica) manda w-full, ancho
                   completo. Solo en horizontal (pantalla de computadora de
                   verdad) se topa el alto a ~54vh -- ahí sí, ese mismo
                   ancho completo con este aspecto 3:4 da una caja más alta
                   que la ventana. */}
               <div className="relative rounded-[3.5rem] overflow-hidden border-8 border-white bg-slate-200 shadow-2xl aspect-[3/4] mx-auto w-full landscape:w-[min(100%,54vh)]">
                  <Webcam
                    key={camaraKey}
                    audio={false}
                    ref={webcamRef}
                    screenshotFormat="image/jpeg"
                    videoConstraints={videoConstraints}
                    className="absolute inset-0 w-full h-full object-cover"
                    mirrored={true}
                    onUserMedia={(stream) => { setCamaraLista(true); setCamaraError(false); ajustarEnfoqueCercano(stream); }}
                    onUserMediaError={(err) => { console.error('No se pudo activar la cámara:', err); setCamaraError(true); }}
                  />
                  {/* "A veces prende la cámara y a veces no... me cambio a otra parte
                      del menú y regreso y ya no se ve" -- sin este aviso, una cámara que
                      no arrancó (hardware ocupado, WebView tardado, lo que sea) se veía
                      igual que una que sí -- un rectángulo gris, sin ninguna pista de que
                      algo falló ni forma de reintentar sin recargar toda la página. */}
                  {!camaraLista && (
                    <div className="absolute inset-0 z-30 flex flex-col items-center justify-center gap-4 bg-slate-800/90 px-8 text-center">
                      {camaraError ? (
                        <>
                          <AlertCircle className="text-rose-300" size={36} />
                          <p className="text-white text-xs font-black uppercase tracking-widest">No se pudo activar la cámara</p>
                          <button onClick={reintentarCamara} className="bg-white text-slate-900 text-[11px] font-black uppercase tracking-widest px-6 py-3 rounded-2xl shadow-md active:scale-95 transition-all">
                            Reintentar
                          </button>
                        </>
                      ) : (
                        <>
                          <RefreshCw className="animate-spin text-white" size={32} />
                          <p className="text-white text-[10px] font-black uppercase tracking-widest">Iniciando cámara...</p>
                        </>
                      )}
                    </div>
                  )}
                  {/* Antes había un óvalo guía dibujado encima del video ("Encuadra tu rostro
                      dentro del óvalo"). Nunca fue un requisito real -- Rekognition no lo usa
                      para nada, solo era una ayuda visual -- pero forzaba a los papás a alejarse
                      de la cámara para que la cara completa cupiera adentro ("tengo que estirar
                      todo el brazo"), sin importar qué tan chico se hiciera el óvalo. Se quitó del
                      todo: ahora el video se ve solo, sin guía de encuadre. */}
                  {/* El botón vive DENTRO del cuadro de la cámara (flotando
                      sobre el video, pegado abajo pero respetando el margen
                      del marco blanco) en vez de un bloque aparte debajo --
                      así no le suma más alto a la pantalla. */}
                  <div className="absolute inset-x-0 bottom-0 z-20 p-5 sm:p-7">
                    <button onClick={() => procesarRostro(tab)} disabled={loading} className="w-full py-5 sm:py-6 bg-brand-600/95 hover:bg-brand-700 text-white rounded-[2rem] font-black uppercase text-lg sm:text-xl shadow-lg backdrop-blur-sm active:scale-95 transition-all disabled:opacity-50">
                      {loading ? 'Procesando...' : (tab === 'registrar' ? 'Confirmar Registro' : 'Escanear Rostro')}
                    </button>
                  </div>
                  {loading && <div className="absolute inset-0 bg-white/60 backdrop-blur-sm flex items-center justify-center z-30"><RefreshCw className="animate-spin text-brand-600" size={54} /></div>}
               </div>
            </div>
            <div ref={resultadoRef} className="w-full max-w-md scroll-mt-6">
               {resultado ? (
                 <div className={`p-6 sm:p-8 rounded-[3rem] border-2 bg-white shadow-xl animate-in zoom-in duration-300 ${resultado.type === 'success' ? 'border-emerald-100' : 'border-rose-100'}`}>
                    <div className="flex items-center gap-4 mb-6">
                      {resultado.type === 'success' ? <CheckCircle className="text-emerald-500" size={28} /> : <AlertCircle className="text-rose-500" size={28} />}
                      <h3 className="text-xl font-black uppercase text-slate-900">{resultado.type === 'success' ? 'Identificado' : 'Aviso'}</h3>
                    </div>
                    {resultado.type === 'success' ? (
                      <div className="space-y-6">
                        <div className="bg-slate-50 p-5 rounded-2xl border border-slate-100">
                          <p className="text-brand-600 text-[10px] font-black uppercase mb-1 tracking-widest">Tutor</p>
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
                                <div key={i} className={`p-4 rounded-2xl border transition-all ${yaSalio ? 'opacity-50 bg-slate-100' : estaSeleccionado ? 'bg-brand-50 border-brand-200 shadow-sm' : 'bg-white'}`}>
                                  <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <input type="checkbox" disabled={yaSalio} checked={estaSeleccionado} onChange={() => manejarToggleHijo(h)} className="w-6 h-6 rounded-lg accent-brand-600" />
                                        <span className={`font-bold uppercase text-sm ${yaSalio ? 'line-through' : 'text-slate-700'}`}>{h.nombre_niño || h.nombre}</span>
                                    </div>
                                    <span className={`text-[9px] font-black px-2 py-1 rounded-lg ${estaDentro ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-200 text-slate-500'}`}>{estado}</span>
                                  </div>
                                  {estaSeleccionado && !estaDentro && !yaSalio && (
                                    <div className="mt-4 space-y-3 pt-4 border-t border-brand-100">
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
      </div>

      {mostrarModalGestion && padreSeleccionado && (
        <div className="fixed inset-0 z-[150] flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-md">
          <div className="bg-white w-full max-w-4xl rounded-[2.5rem] shadow-2xl overflow-hidden relative border-t-8 border-t-brand-600 max-h-[90vh] flex flex-col">
            <button onClick={() => { setMostrarModalGestion(false); cargarTodosLosPadres(); }} className="absolute top-4 right-4 sm:top-6 sm:right-6 bg-white/80 backdrop-blur-sm rounded-full text-slate-400 hover:text-slate-600 z-10 p-2"><X size={24} className="sm:hidden" /><X size={32} className="hidden sm:block" /></button>
            <div className="overflow-y-auto overflow-x-hidden flex-1">
              <GestionHijos padreId={padreSeleccionado.id} nombrePadre={padreSeleccionado.nombre} onFinalizar={() => { setMostrarModalGestion(false); cargarTodosLosPadres(); }} />
            </div>
          </div>
        </div>
      )}

      {mostrarModalAviso && avisoPrivacidad && (
        <AvisoPrivacidadModal
          texto={avisoPrivacidad.texto}
          pdfUrl={avisoPrivacidad.pdf_url}
          version={avisoPrivacidad.version}
          onAceptar={manejarAceptarAvisoPrivacidad}
          onCancelar={() => setMostrarModalAviso(false)}
        />
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

      <SoporteChat modo="autenticado" />
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

        {/* Alta de guardería nueva (pública, con aprobación manual -- ver
            solicitudes.go) y revisión de esas solicitudes (protegida por
            PLATFORM_ADMIN_KEY, no por el login normal de guarderías). */}
        <Route path="/registro-guarderia" element={<RegistroGuarderia />} />
        <Route path="/plataforma" element={<PanelPlataforma />} />

        {/* Panel de personal: la pestaña activa vive en la URL, así sobrevive
            a un refresh y el botón "atrás" del navegador funciona. */}
        <Route path="/panel/:tab" element={<MainApp />} />

        {/* "Ponla como mi página principal" -- la página de presentación es
            ahora la entrada real de "/", en vez de mandar directo al login.
            El login/kiosco se queda en /panel/identificar, alcanzable desde
            el botón "Iniciar sesión" de la propia LandingPage. */}
        <Route path="/" element={<LandingPage />} />

        {/* Términos y Condiciones -- pública, sin sesión, para que
            cualquiera (prospecto o guardería ya dada de alta) pueda
            leerlos. Ver TERMINOS_Y_CONDICIONES.md en la raíz del repo,
            misma redacción. */}
        <Route path="/terminos" element={<TerminosCondiciones />} />

        {/* Aviso de Privacidad de Pasitos (distinto del de cada guardería)
            -- rige el chat de soporte con prospectos y con papás/staff que
            escriben directo a Pasitos, ver la cláusula 9 Bis de
            TerminosCondiciones.jsx. */}
        <Route path="/aviso-privacidad-pasitos" element={<AvisoPrivacidadPasitos />} />

        {/* Redirección por defecto si algo sale mal */}
        <Route path="*" element={<Navigate to="/panel/identificar" replace />} />
      </Routes>
    </Router>
  );
}

export default App;