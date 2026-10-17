import React, { useEffect } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { color, radius } from '../theme';

// Placeholder para las secciones que todavía no se portaron a la app
// (Chat, Encuestas, Eventos, Menú semanal -- ya existen y funcionan en la
// web, ver API_MOVIL.md para el orden en que se van a ir agregando aquí).
// Se usa un solo componente para las cuatro en vez de cuatro pantallas
// vacías repetidas.
export default function ProximamenteScreen({ route, navigation }) {
  const { titulo } = route.params;

  useEffect(() => {
    navigation.setOptions({ title: titulo });
  }, [navigation, titulo]);

  return (
    <View style={styles.pantalla}>
      <View style={styles.icono}><Ionicons name="construct" size={32} color={color.brand600} /></View>
      <Text style={styles.titulo}>{titulo}</Text>
      <Text style={styles.texto}>Esta sección todavía no está en la app -- por ahora se sigue viendo en el navegador, igual que siempre.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50, alignItems: 'center', justifyContent: 'center', padding: 32, gap: 12 },
  icono: { backgroundColor: color.brand50, padding: 20, borderRadius: radius.xl, marginBottom: 8 },
  titulo: { fontSize: 16, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  texto: { fontSize: 13, color: color.slate400, textAlign: 'center', fontWeight: '600', lineHeight: 19 },
});
