import React, { useState } from 'react';
import {
  View, Text, TextInput, TouchableOpacity, StyleSheet, ActivityIndicator,
  KeyboardAvoidingView, Platform, ScrollView,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useAuth } from '../context/AuthContext';
import { color, radius } from '../theme';

// Pantalla de entrada -- equivalente al bloque "!isLoggedIn" de App.jsx en
// la web, pero solo la mitad "Soy Papá": esta app no tiene la parte de
// staff/admin (ver la decisión en API_MOVIL.md), así que no hace falta el
// selector de tipo de acceso.
export default function LoginScreen() {
  const { login, errorSesion } = useAuth();
  const [correo, setCorreo] = useState('');
  const [contrasena, setContrasena] = useState('');
  const [cargando, setCargando] = useState(false);
  const [error, setError] = useState(errorSesion || '');

  const entrar = async () => {
    if (!correo.trim() || !contrasena) {
      setError('Escribe tu correo y tu contraseña');
      return;
    }
    setError('');
    setCargando(true);
    try {
      await login(correo.trim(), contrasena);
    } catch (err) {
      // err.response.data.error es lo que el backend dice cuando SÍ
      // respondió (usuario no existe, contraseña incorrecta, etc.). Si no
      // hay err.response, la petición ni siquiera llegó a tener respuesta
      // (sin internet, DNS, tiempo agotado...) -- se muestra err.message
      // en vez del aviso genérico de siempre, para no tener que adivinar
      // qué pasó si vuelve a fallar.
      setError(err.response?.data?.error || err.message || 'No se pudo iniciar sesión');
    } finally {
      setCargando(false);
    }
  };

  return (
    <KeyboardAvoidingView
      style={styles.pantalla}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        <View style={styles.tarjeta}>
          <View style={styles.logo}>
            <Ionicons name="shield-checkmark" size={32} color={color.white} />
          </View>
          <Text style={styles.titulo}>PASITOS</Text>
          <Text style={styles.subtitulo}>Portal de familias</Text>

          {!!error && (
            <View style={styles.avisoError}>
              <Text style={styles.avisoErrorTexto}>{error}</Text>
            </View>
          )}

          <TextInput
            style={styles.input}
            placeholder="Correo electrónico"
            placeholderTextColor={color.slate400}
            value={correo}
            onChangeText={setCorreo}
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="email-address"
            editable={!cargando}
          />
          <TextInput
            style={styles.input}
            placeholder="Contraseña"
            placeholderTextColor={color.slate400}
            value={contrasena}
            onChangeText={setContrasena}
            secureTextEntry
            editable={!cargando}
            onSubmitEditing={entrar}
          />

          <TouchableOpacity style={styles.boton} onPress={entrar} disabled={cargando} activeOpacity={0.85}>
            {cargando
              ? <ActivityIndicator color={color.white} />
              : <Text style={styles.botonTexto}>Entrar</Text>}
          </TouchableOpacity>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.paper },
  scroll: { flexGrow: 1, justifyContent: 'center', padding: 24 },
  tarjeta: {
    backgroundColor: color.white, borderRadius: radius.lg, padding: 28, alignItems: 'center',
    shadowColor: '#000', shadowOpacity: 0.08, shadowRadius: 20, shadowOffset: { width: 0, height: 8 }, elevation: 6,
  },
  logo: {
    width: 64, height: 64, borderRadius: radius.md, backgroundColor: color.brand600,
    alignItems: 'center', justifyContent: 'center', marginBottom: 16,
  },
  titulo: { fontSize: 26, fontWeight: '900', color: color.slate900, letterSpacing: -0.5 },
  subtitulo: { fontSize: 11, fontWeight: '800', color: color.slate400, textTransform: 'uppercase', letterSpacing: 1, marginTop: 4, marginBottom: 24 },
  input: {
    width: '100%', backgroundColor: color.slate50, borderRadius: radius.sm, paddingHorizontal: 16, paddingVertical: 14,
    fontSize: 15, color: color.slate900, marginBottom: 12,
  },
  boton: {
    width: '100%', backgroundColor: color.brand600, borderRadius: radius.sm, paddingVertical: 16,
    alignItems: 'center', marginTop: 4,
  },
  botonTexto: { color: color.white, fontWeight: '900', fontSize: 13, textTransform: 'uppercase', letterSpacing: 0.5 },
  avisoError: { width: '100%', backgroundColor: color.rose50, borderRadius: radius.sm, padding: 12, marginBottom: 12 },
  avisoErrorTexto: { color: color.rose600, fontSize: 12, fontWeight: '700', textAlign: 'center' },
});
