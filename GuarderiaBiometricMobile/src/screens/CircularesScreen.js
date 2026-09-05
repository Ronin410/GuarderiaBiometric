import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, Image, RefreshControl } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { color, radius, sombra } from '../theme';

// Equivalente RN de CircularesPadre.jsx en la web: TODOS los avisos de la
// guardería (el inicio solo muestra el más reciente). Se marcan como
// leídos los que de verdad se muestran aquí, mismo criterio que la web.
export default function CircularesScreen() {
  const [circulares, setCirculares] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [refrescando, setRefrescando] = useState(false);

  const cargar = async () => {
    try {
      const res = await api.get('/padre/circulares');
      const todas = Array.isArray(res.data) ? res.data : [];
      setCirculares(todas);
      todas.forEach((cir) => {
        api.post(`/padre/circulares/${cir.id}/leido`).catch((err) => {
          console.error('Error al marcar circular como leída', err);
        });
      });
    } catch (err) {
      console.error('Error al cargar circulares:', err);
    } finally {
      setCargando(false);
      setRefrescando(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const refrescar = () => {
    setRefrescando(true);
    cargar();
  };

  const formatoFecha = (iso) => {
    try {
      return new Date(iso).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' });
    } catch {
      return iso;
    }
  };

  if (cargando) {
    return <View style={styles.centro}><ActivityIndicator color={color.brand600} /></View>;
  }

  if (circulares.length === 0) {
    return (
      <ScrollView
        contentContainerStyle={styles.centro}
        refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
      >
        <Ionicons name="megaphone-outline" size={36} color={color.slate200} />
        <Text style={styles.vacioTexto}>Todavía no hay avisos de la guardería</Text>
      </ScrollView>
    );
  }

  return (
    <ScrollView
      style={styles.pantalla}
      contentContainerStyle={styles.contenido}
      refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
    >
      {circulares.map((cir) => (
        <View key={cir.id} style={styles.tarjeta}>
          <Text style={styles.titulo}>{cir.titulo}</Text>
          <View style={styles.filaFecha}>
            <Ionicons name="time-outline" size={12} color={color.brand500} />
            <Text style={styles.fecha}>{formatoFecha(cir.creado_en)}</Text>
          </View>
          <Text style={styles.contenido}>{cir.contenido}</Text>
          {!!cir.imagen_url && (
            <Image source={{ uri: cir.imagen_url }} style={styles.imagen} resizeMode="cover" />
          )}
        </View>
      ))}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 14, paddingBottom: 40 },
  centro: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32, gap: 12 },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 12, textTransform: 'uppercase', textAlign: 'center' },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, gap: 8, ...sombra.sm },
  titulo: { fontSize: 14, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  filaFecha: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  fecha: { fontSize: 10, color: color.brand500, fontWeight: '700', textTransform: 'uppercase' },
  contenido: { fontSize: 13, color: color.slate600, fontWeight: '600', lineHeight: 19 },
  imagen: { width: '100%', height: 180, borderRadius: radius.md, marginTop: 4, backgroundColor: color.slate100 },
});
