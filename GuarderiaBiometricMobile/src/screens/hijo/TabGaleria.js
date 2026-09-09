import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator, Image, Modal, Dimensions } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../../api/client';
import { color, radius } from '../../theme';

const formatoFecha = (iso) => {
  try {
    const [anio, mes, dia] = iso.split('-').map(Number);
    return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', year: 'numeric' });
  } catch {
    return iso;
  }
};

const anchoPantalla = Dimensions.get('window').width;
const anchoFoto = (anchoPantalla - 40 - 40 - 16) / 3; // padding pantalla + padding tarjeta + 2 gaps

// Equivalente RN de components/GaleriaFotos.jsx: todas las fotos que ya se
// suben día a día desde la bitácora, agrupadas por fecha.
export default function TabGaleria({ hijoId }) {
  const [fotos, setFotos] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [fotoAmpliada, setFotoAmpliada] = useState(null);

  useEffect(() => {
    api.get(`/padre/hijos/${hijoId}/galeria`)
      .then((res) => setFotos(Array.isArray(res.data) ? res.data : []))
      .catch((err) => {
        console.error('Error al cargar la galería:', err);
        setFotos([]);
      })
      .finally(() => setCargando(false));
  }, [hijoId]);

  if (cargando) {
    return <View style={styles.centro}><ActivityIndicator color={color.brand600} /></View>;
  }

  if (fotos.length === 0) {
    return (
      <View style={styles.centro}>
        <Ionicons name="image-outline" size={36} color={color.slate200} />
        <Text style={styles.vacioTexto}>Sin fotos todavía</Text>
      </View>
    );
  }

  const porFecha = fotos.reduce((acc, f) => {
    (acc[f.fecha] = acc[f.fecha] || []).push(f);
    return acc;
  }, {});

  return (
    <ScrollView style={styles.pantalla} contentContainerStyle={styles.contenido}>
      {Object.keys(porFecha).map((fecha) => (
        <View key={fecha} style={{ gap: 8 }}>
          <Text style={styles.fecha}>{formatoFecha(fecha)}</Text>
          <View style={styles.grilla}>
            {porFecha[fecha].map((f) => (
              <TouchableOpacity key={f.id} style={styles.foto} onPress={() => setFotoAmpliada(f.url)}>
                <Image source={{ uri: f.url }} style={styles.fotoImg} />
              </TouchableOpacity>
            ))}
          </View>
        </View>
      ))}

      <Modal visible={!!fotoAmpliada} transparent animationType="fade" onRequestClose={() => setFotoAmpliada(null)}>
        <TouchableOpacity style={styles.modalFondo} activeOpacity={1} onPress={() => setFotoAmpliada(null)}>
          {!!fotoAmpliada && <Image source={{ uri: fotoAmpliada }} style={styles.modalImagen} resizeMode="contain" />}
        </TouchableOpacity>
      </Modal>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 20, paddingBottom: 40 },
  centro: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32, gap: 12 },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 12, textTransform: 'uppercase' },
  fecha: { fontSize: 10, fontWeight: '900', color: color.brand500, textTransform: 'uppercase', letterSpacing: 0.5, marginLeft: 2 },
  grilla: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  foto: { width: anchoFoto, height: anchoFoto, borderRadius: radius.sm, overflow: 'hidden', backgroundColor: color.slate100 },
  fotoImg: { width: '100%', height: '100%' },
  modalFondo: { flex: 1, backgroundColor: 'rgba(15,23,42,0.92)', alignItems: 'center', justifyContent: 'center' },
  modalImagen: { width: '92%', height: '80%' },
});
