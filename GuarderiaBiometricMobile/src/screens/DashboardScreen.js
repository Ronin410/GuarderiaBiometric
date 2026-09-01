import React, { useCallback, useEffect, useState } from 'react';
import {
  View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator, RefreshControl,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { useAuth } from '../context/AuthContext';
import { color, radius, sombra } from '../theme';

// Equivalente a la parte de arriba de DashboardPadre.jsx en la web: saludo,
// aviso de "las bitácoras se actualizan en tiempo real", entradas fijas
// (Chat/Encuestas/Eventos/Menú -- por ahora "Próximamente", ver
// API_MOVIL.md) y el listado de niños para entrar a su bitácora de hoy.
const ENTRADAS = [
  { key: 'chat', icon: 'chatbubble-ellipses', titulo: 'Chat con la guardería', subtitulo: 'Mensajes directos, sin WhatsApp' },
  { key: 'encuestas', icon: 'clipboard', titulo: 'Encuestas', subtitulo: 'Comparte tu opinión con la guardería' },
  { key: 'eventos', icon: 'calendar', titulo: 'Eventos', subtitulo: 'Calendario escolar de la guardería' },
  { key: 'menu', icon: 'restaurant', titulo: 'Menú semanal', subtitulo: 'Días anteriores y posteriores' },
];

export default function DashboardScreen({ navigation }) {
  const { usuario, cerrarSesion } = useAuth();
  const [hijos, setHijos] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [refrescando, setRefrescando] = useState(false);

  const cargarHijos = useCallback(async () => {
    try {
      // "0" == el backend lo resuelve con el user_id del propio token (ver
      // el mismo comentario en DashboardPadre.jsx de la web).
      const res = await api.get('/padre/0/hijos');
      const lista = (res.data || []).map((h) => ({
        id: h.id,
        nombre: h.nombre_niño || h.nombre || 'Sin nombre',
      }));
      setHijos(lista);
    } catch (err) {
      console.error('Error al cargar los hijos:', err);
    } finally {
      setCargando(false);
      setRefrescando(false);
    }
  }, []);

  useEffect(() => { cargarHijos(); }, [cargarHijos]);

  const refrescar = () => {
    setRefrescando(true);
    cargarHijos();
  };

  return (
    <ScrollView
      style={styles.pantalla}
      contentContainerStyle={styles.contenido}
      refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
    >
      <View style={styles.encabezado}>
        <Text style={styles.saludo}>Hola,</Text>
        <Text style={styles.nombreUsuario}>{usuario?.username}</Text>
        <Text style={styles.pregunta}>¿DE QUIÉN DESEAS VER EL REPORTE HOY?</Text>
      </View>

      <View style={styles.avisoTiempoReal}>
        <View style={styles.avisoIcono}><Ionicons name="notifications" size={18} color={color.brand600} /></View>
        <Text style={styles.avisoTexto}>Las bitácoras se actualizan en tiempo real por las maestras.</Text>
      </View>

      {ENTRADAS.map((entrada) => (
        <TouchableOpacity
          key={entrada.key}
          style={styles.tarjetaFila}
          activeOpacity={0.7}
          onPress={() => navigation.navigate('Proximamente', { titulo: entrada.titulo })}
        >
          <View style={styles.iconoRedondo}><Ionicons name={entrada.icon} size={20} color={color.brand600} /></View>
          <View style={styles.filaTexto}>
            <Text style={styles.filaTitulo}>{entrada.titulo}</Text>
            <Text style={styles.filaSubtitulo}>{entrada.subtitulo}</Text>
          </View>
          <Ionicons name="chevron-forward" size={20} color={color.slate300} />
        </TouchableOpacity>
      ))}

      <Text style={styles.seccionLabel}>Niños</Text>

      {cargando ? (
        <ActivityIndicator color={color.brand600} style={{ marginTop: 24 }} />
      ) : hijos.length === 0 ? (
        <View style={styles.vacio}>
          <Ionicons name="body" size={36} color={color.slate200} />
          <Text style={styles.vacioTexto}>No se encontraron niños vinculados.</Text>
        </View>
      ) : (
        hijos.map((hijo) => (
          <TouchableOpacity
            key={hijo.id}
            style={styles.tarjetaHijo}
            activeOpacity={0.8}
            onPress={() => navigation.navigate('Bitacora', { hijoId: hijo.id, nombreHijo: hijo.nombre })}
          >
            <View style={styles.avatarHijo}><Ionicons name="person" size={28} color={color.slate400} /></View>
            <View style={styles.filaTexto}>
              <Text style={styles.nombreHijo}>{hijo.nombre}</Text>
              <View style={styles.verBitacoraFila}>
                <Ionicons name="heart" size={10} color={color.brand500} />
                <Text style={styles.verBitacora}>Ver bitácora diaria</Text>
              </View>
            </View>
            <View style={styles.flechaHijo}><Ionicons name="chevron-forward" size={20} color={color.slate300} /></View>
          </TouchableOpacity>
        ))
      )}

      <TouchableOpacity style={styles.salir} onPress={cerrarSesion}>
        <Ionicons name="log-out-outline" size={16} color={color.slate400} />
        <Text style={styles.salirTexto}>Cerrar sesión</Text>
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, paddingBottom: 40, gap: 12 },
  encabezado: { alignItems: 'center', paddingVertical: 12, gap: 2 },
  saludo: { fontSize: 30, fontWeight: '900', color: color.slate900, lineHeight: 32 },
  nombreUsuario: { fontSize: 22, fontWeight: '900', color: color.brand600 },
  pregunta: { fontSize: 10, fontWeight: '800', color: color.slate400, letterSpacing: 1, marginTop: 8 },
  avisoTiempoReal: {
    flexDirection: 'row', alignItems: 'center', gap: 12, backgroundColor: color.brand50,
    borderWidth: 1, borderColor: color.brand100, borderRadius: radius.lg, padding: 16,
  },
  avisoIcono: { backgroundColor: color.white, padding: 10, borderRadius: radius.sm, ...sombra.sm },
  avisoTexto: { flex: 1, fontSize: 11, fontWeight: '800', color: color.brand800, textTransform: 'uppercase' },
  tarjetaFila: {
    flexDirection: 'row', alignItems: 'center', gap: 14, backgroundColor: color.white,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 16, ...sombra.sm,
  },
  iconoRedondo: { backgroundColor: color.brand100, padding: 10, borderRadius: radius.sm },
  filaTexto: { flex: 1 },
  filaTitulo: { fontSize: 12, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  filaSubtitulo: { fontSize: 9, fontWeight: '700', color: color.slate400, textTransform: 'uppercase', marginTop: 2 },
  seccionLabel: { fontSize: 11, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 1, marginTop: 8, marginLeft: 4 },
  vacio: { backgroundColor: color.white, borderWidth: 2, borderStyle: 'dashed', borderColor: color.slate200, borderRadius: radius.lg, padding: 32, alignItems: 'center', gap: 12 },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 11, textTransform: 'uppercase' },
  tarjetaHijo: {
    flexDirection: 'row', alignItems: 'center', gap: 16, backgroundColor: color.white,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 18, ...sombra.md,
  },
  avatarHijo: { width: 56, height: 56, borderRadius: radius.md, backgroundColor: color.slate50, alignItems: 'center', justifyContent: 'center', borderWidth: 1, borderColor: color.slate100 },
  nombreHijo: { fontSize: 17, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  verBitacoraFila: { flexDirection: 'row', alignItems: 'center', gap: 4, marginTop: 2 },
  verBitacora: { fontSize: 9, fontWeight: '900', color: color.brand500, textTransform: 'uppercase', letterSpacing: 0.5 },
  flechaHijo: { backgroundColor: color.slate50, padding: 10, borderRadius: radius.sm },
  salir: { flexDirection: 'row', alignItems: 'center', justifyContent: 'center', gap: 6, paddingVertical: 20 },
  salirTexto: { color: color.slate400, fontWeight: '800', fontSize: 11, textTransform: 'uppercase' },
});
