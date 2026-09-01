import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import api, { setCsrfToken } from '../api/client';

const AuthContext = createContext(null);

// Mismo patrón que hidratarSesion() en GuarderiaBiometricFront/src/App.jsx:
// la cookie con el JWT es httpOnly (invisible a JS), así que al abrir la
// app no hay forma de saber quién es el usuario sin preguntarle al
// backend -- GET /me hace eso, y de paso renueva la sesión del papá (ver
// el comentario largo en auth.go: handleMe, sesión deslizante de 90 días
// solo para "papa").
//
// Esta app es SOLO para papás (ver AuthProvider más abajo) -- si /me
// regresa una cuenta de staff/admin, se trata igual que "no hay sesión":
// esta app no tiene esas pantallas.
const AuthContext_ = AuthContext;

export function AuthProvider({ children }) {
  const [cargando, setCargando] = useState(true);
  const [usuario, setUsuario] = useState(null); // { userId, username, guarderiaNombre, guarderiaSlug }
  const [errorSesion, setErrorSesion] = useState(null);

  const cargarSesion = useCallback(async () => {
    setCargando(true);
    try {
      const res = await api.get('/me');
      if (res.data?.rol !== 'papa') {
        // Cuenta real pero no es de una familia -- esta app no la atiende.
        setUsuario(null);
        setErrorSesion('Esta cuenta no es de una familia. Esta app es solo para papás.');
        return;
      }
      setCsrfToken(res.data.csrf_token);
      setUsuario({
        userId: res.data.user_id,
        username: res.data.username,
        guarderiaNombre: res.data.guarderia_nombre,
        guarderiaSlug: res.data.guarderia_slug,
      });
      setErrorSesion(null);
    } catch {
      // Sin sesión (o guardería bloqueada, ver el comentario de handleMe
      // en el backend) -- se trata igual que "no hay sesión todavía", el
      // mensaje específico de bloqueo se vuelve a ver si intenta el login.
      setUsuario(null);
    } finally {
      setCargando(false);
    }
  }, []);

  useEffect(() => { cargarSesion(); }, [cargarSesion]);

  const login = useCallback(async (correo, contrasena) => {
    const res = await api.post('/login', { username: correo, password: contrasena, tipo: 'papa' });
    setCsrfToken(res.data.csrf_token);
    setUsuario({
      userId: res.data.user_id,
      username: res.data.username,
      guarderiaNombre: res.data.guarderia_nombre,
      guarderiaSlug: res.data.guarderia_slug,
    });
    setErrorSesion(null);
  }, []);

  const cerrarSesion = useCallback(async () => {
    try {
      await api.post('/logout');
    } catch {
      // Si /logout falla igual se limpia la sesión local -- no tiene caso
      // dejar al papá "atorado" adentro de la app por un error de red.
    }
    setCsrfToken(null);
    setUsuario(null);
  }, []);

  const value = useMemo(() => ({
    cargando, usuario, errorSesion, autenticado: !!usuario, login, cerrarSesion, recargarSesion: cargarSesion,
  }), [cargando, usuario, errorSesion, login, cerrarSesion, cargarSesion]);

  return <AuthContext_.Provider value={value}>{children}</AuthContext_.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth debe usarse dentro de <AuthProvider>');
  return ctx;
}
