import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import TabHoy from './hijo/TabHoy';
import TabExpediente from './hijo/TabExpediente';
import TabPagos from './hijo/TabPagos';
import TabAusencias from './hijo/TabAusencias';
import TabComedor from './hijo/TabComedor';
import TabGaleria from './hijo/TabGaleria';
import { color, radius } from '../theme';

const PESTANAS = [
  { key: 'hoy', label: 'Hoy', icon: 'list' },
  { key: 'expediente', label: 'Expediente', icon: 'card' },
  { key: 'pagos', label: 'Pagos', icon: 'wallet' },
  { key: 'ausencias', label: 'Ausencias', icon: 'calendar-clear' },
  { key: 'comedor', label: 'Comedor', icon: 'restaurant' },
  { key: 'galeria', label: 'Galería', icon: 'images' },
];

// Contenedor de pestañas equivalente a VistaPadreDetalle.jsx en la web:
// cada pestaña vive en su propio archivo dentro de screens/hijo/, este
// componente solo trae el selector y decide cuál mostrar. El selector de
// fecha de la pestaña "Hoy" se queda dentro de TabHoy.js, igual que en la
// web (solo aparece con esa pestaña activa).
export default function BitacoraScreen({ route, navigation }) {
  const { hijoId, nombreHijo, expediente } = route.params;
  const [vista, setVista] = useState('hoy');

  useEffect(() => {
    navigation.setOptions({ title: nombreHijo });
  }, [navigation, nombreHijo]);

  return (
    <View style={styles.pantalla}>
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.pestanas}>
        {PESTANAS.map((p) => (
          <TouchableOpacity
            key={p.key}
            style={[styles.pestana, vista === p.key && styles.pestanaActiva]}
            onPress={() => setVista(p.key)}
          >
            <Ionicons name={p.icon} size={13} color={vista === p.key ? color.brand600 : color.slate400} />
            <Text style={[styles.pestanaTexto, vista === p.key && styles.pestanaTextoActivo]}>{p.label}</Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      <View style={{ flex: 1 }}>
        {vista === 'hoy' && <TabHoy hijoId={hijoId} />}
        {vista === 'expediente' && <TabExpediente hijoId={hijoId} expediente={expediente} />}
        {vista === 'pagos' && <TabPagos hijoId={hijoId} />}
        {vista === 'ausencias' && <TabAusencias hijoId={hijoId} />}
        {vista === 'comedor' && <TabComedor hijoId={hijoId} />}
        {vista === 'galeria' && <TabGaleria hijoId={hijoId} />}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  pestanas: {
    flexDirection: 'row', gap: 6, backgroundColor: color.white, paddingHorizontal: 16, paddingVertical: 10,
    borderBottomWidth: 1, borderBottomColor: color.slate100,
  },
  pestana: {
    flexDirection: 'row', alignItems: 'center', gap: 6, backgroundColor: color.slate100,
    paddingHorizontal: 14, paddingVertical: 9, borderRadius: radius.md,
  },
  pestanaActiva: { backgroundColor: color.brand50 },
  pestanaTexto: { fontSize: 10, fontWeight: '900', color: color.slate400, textTransform: 'uppercase' },
  pestanaTextoActivo: { color: color.brand600 },
});
