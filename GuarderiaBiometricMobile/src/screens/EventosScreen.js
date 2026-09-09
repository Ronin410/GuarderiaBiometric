import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, RefreshControl } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { color, radius, sombra } from '../theme';

// Mismo catálogo de colores que PanelCalendario.jsx (staff) y
// EventosPadre.jsx (web) -- para que un papá vea el mismo tipo de evento
// con el mismo color que quien lo publicó.
const TIPOS = {
  evento: { label: 'Evento', bg: color.brand100, texto: color.brand700, borde: color.brand200 },
  suspension: { label: 'Suspensión de clases', bg: color.rose100, texto: color.rose700, borde: color.rose50 },
  vacaciones: { label: 'Vacaciones', bg: color.amber100, texto: color.amber700, borde: color.amber50 },
  junta: { label: 'Junta de padres', bg: color.emerald100, texto: color.emerald700, borde: color.emerald200 },
};

// Equivalente RN de EventosPadre.jsx: TODOS los eventos del calendario
// escolar (el inicio solo adelanta los próximos 3). /padre/calendario ya
// regresa hoy hasta 90 días adelante ordenados por fecha.
export default function EventosScreen() {
  const [eventos, setEventos] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [refrescando, setRefrescando] = useState(false);

  const cargar = async () => {
    try {
      const res = await api.get('/padre/calendario');
      setEventos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar el calendario', err);
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

  const formatoFecha = (fechaISO) => {
    try {
      const [anio, mes, dia] = fechaISO.split('-').map(Number);
      return new Date(anio, mes - 1, dia).toLocaleDateString('es-MX', { day: 'numeric', month: 'long', year: 'numeric' });
    } catch {
      return fechaISO;
    }
  };

  if (cargando) {
    return <View style={styles.centro}><ActivityIndicator color={color.brand600} /></View>;
  }

  if (eventos.length === 0) {
    return (
      <ScrollView
        contentContainerStyle={styles.centro}
        refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
      >
        <Ionicons name="calendar-outline" size={36} color={color.slate200} />
        <Text style={styles.vacioTexto}>No hay eventos próximos en el calendario</Text>
      </ScrollView>
    );
  }

  return (
    <ScrollView
      style={styles.pantalla}
      contentContainerStyle={styles.contenido}
      refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
    >
      {eventos.map((ev) => {
        const info = TIPOS[ev.tipo] || TIPOS.evento;
        return (
          <View key={ev.id} style={styles.tarjeta}>
            <View style={styles.filaTitulo}>
              <Text style={styles.titulo}>{ev.titulo}</Text>
              <View style={[styles.etiqueta, { backgroundColor: info.bg, borderColor: info.borde }]}>
                <Text style={[styles.etiquetaTexto, { color: info.texto }]}>{info.label}</Text>
              </View>
            </View>
            <Text style={styles.fecha}>
              {formatoFecha(ev.fecha_inicio)}
              {ev.fecha_fin && ev.fecha_fin !== ev.fecha_inicio ? ` – ${formatoFecha(ev.fecha_fin)}` : ''}
            </Text>
            {!!ev.descripcion && <Text style={styles.descripcion}>{ev.descripcion}</Text>}
          </View>
        );
      })}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 14, paddingBottom: 40 },
  centro: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32, gap: 12 },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 12, textTransform: 'uppercase', textAlign: 'center' },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, gap: 8, ...sombra.sm },
  filaTitulo: { flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', gap: 8 },
  titulo: { flex: 1, fontSize: 14, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  etiqueta: { borderWidth: 1, borderRadius: radius.sm, paddingHorizontal: 8, paddingVertical: 4 },
  etiquetaTexto: { fontSize: 9, fontWeight: '900', textTransform: 'uppercase' },
  fecha: { fontSize: 10, color: color.brand500, fontWeight: '700', textTransform: 'uppercase' },
  descripcion: { fontSize: 13, color: color.slate600, fontWeight: '600', lineHeight: 19 },
});
