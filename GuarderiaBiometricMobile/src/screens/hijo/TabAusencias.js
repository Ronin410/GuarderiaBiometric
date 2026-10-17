import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, ActivityIndicator, Alert, Platform } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import DateTimePicker from '@react-native-community/datetimepicker';
import api from '../../api/client';
import { hoyLocal } from '../../utils/fecha';
import { color, radius, sombra } from '../../theme';

// Equivalente RN de la pestaña "Ausencias" de VistaPadreDetalle.jsx:
// avisar con anticipación que el niño no asistirá uno o varios días, y ver/
// cancelar los avisos ya hechos.
export default function TabAusencias({ hijoId }) {
  const [ausencias, setAusencias] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [fechaInicio, setFechaInicio] = useState(null);
  const [fechaFin, setFechaFin] = useState(null);
  const [motivo, setMotivo] = useState('');
  const [mostrarPicker, setMostrarPicker] = useState(null); // 'inicio' | 'fin' | null
  const [guardando, setGuardando] = useState(false);
  const [cancelandoId, setCancelandoId] = useState(null);

  const cargar = useCallback(async () => {
    try {
      const res = await api.get(`/padre/hijos/${hijoId}/ausencias`);
      setAusencias(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al obtener las ausencias', err);
      setAusencias([]);
    } finally {
      setCargando(false);
    }
  }, [hijoId]);

  useEffect(() => { cargar(); }, [cargar]);

  const reportar = async () => {
    if (!fechaInicio) {
      Alert.alert('Elige al menos la fecha de inicio');
      return;
    }
    setGuardando(true);
    try {
      await api.post(`/padre/hijos/${hijoId}/ausencias`, {
        fecha_inicio: fechaInicio,
        fecha_fin: fechaFin || '',
        motivo,
      });
      setFechaInicio(null);
      setFechaFin(null);
      setMotivo('');
      await cargar();
    } catch (err) {
      console.error('Error al reportar la ausencia', err);
      Alert.alert('No se pudo avisar', err.response?.data?.error || 'Inténtalo de nuevo');
    } finally {
      setGuardando(false);
    }
  };

  const cancelar = (ausencia) => {
    Alert.alert('¿Cancelar ausencia?', `Se cancelará el aviso de ausencia del ${ausencia.fecha}.`, [
      { text: 'No', style: 'cancel' },
      {
        text: 'Sí, cancelar', style: 'destructive', onPress: async () => {
          setCancelandoId(ausencia.id);
          try {
            await api.delete(`/padre/ausencias/${ausencia.id}`);
            await cargar();
          } catch (err) {
            console.error('Error al cancelar la ausencia', err);
            Alert.alert('No se pudo cancelar');
          } finally {
            setCancelandoId(null);
          }
        },
      },
    ]);
  };

  // En Android el picker es un diálogo modal nativo que se cierra solo al
  // confirmar/cancelar (un solo onChange con type 'set'/'dismissed'); en
  // iOS es un widget embebido que dispara onChange en cada scroll, así que
  // ahí hay que esperar a que toquen "Listo" para cerrarlo -- ver el botón
  // de abajo.
  const alCambiarFecha = (evento, fechaElegida) => {
    if (Platform.OS === 'android') {
      setMostrarPicker(null);
      if (evento.type === 'dismissed' || !fechaElegida) return;
    }
    if (!fechaElegida) return;
    const iso = fechaElegida.toLocaleDateString('en-CA');
    if (mostrarPicker === 'inicio') setFechaInicio(iso);
    else setFechaFin(iso);
  };

  return (
    <ScrollView style={styles.pantalla} contentContainerStyle={styles.contenido}>
      <View style={styles.tarjeta}>
        <View style={styles.encabezado}>
          <View style={styles.iconoRedondo}><Ionicons name="calendar-clear" size={18} color={color.brand600} /></View>
          <Text style={styles.tituloTarjeta}>Avisar una ausencia</Text>
        </View>
        <Text style={styles.descripcion}>Avísale a la guardería con anticipación si tu hijo no va a asistir uno o varios días.</Text>

        <View style={styles.filaFechas}>
          <TouchableOpacity style={styles.campoFecha} onPress={() => setMostrarPicker('inicio')}>
            <Text style={styles.labelFecha}>Desde</Text>
            <Text style={styles.valorFecha}>{fechaInicio || 'Elegir'}</Text>
          </TouchableOpacity>
          <TouchableOpacity style={styles.campoFecha} onPress={() => setMostrarPicker('fin')}>
            <Text style={styles.labelFecha}>Hasta (opcional)</Text>
            <Text style={styles.valorFecha}>{fechaFin || 'Elegir'}</Text>
          </TouchableOpacity>
        </View>

        {!!mostrarPicker && (
          <View>
            <DateTimePicker
              value={new Date(`${(mostrarPicker === 'inicio' ? fechaInicio : fechaFin) || hoyLocal()}T00:00:00`)}
              mode="date"
              display={Platform.OS === 'ios' ? 'spinner' : 'default'}
              minimumDate={new Date(`${(mostrarPicker === 'inicio' ? hoyLocal() : (fechaInicio || hoyLocal()))}T00:00:00`)}
              onChange={alCambiarFecha}
            />
            {Platform.OS === 'ios' && (
              <TouchableOpacity style={styles.botonListo} onPress={() => setMostrarPicker(null)}>
                <Text style={styles.botonListoTexto}>Listo</Text>
              </TouchableOpacity>
            )}
          </View>
        )}

        <TextInput
          style={styles.input}
          placeholder="Motivo (opcional), ej. Cita médica"
          placeholderTextColor={color.slate400}
          value={motivo}
          onChangeText={setMotivo}
        />

        <TouchableOpacity style={[styles.boton, guardando && styles.botonDeshabilitado]} onPress={reportar} disabled={guardando}>
          {guardando ? <ActivityIndicator color={color.white} size="small" /> : <Ionicons name="add" size={16} color={color.white} />}
          <Text style={styles.botonTexto}>Avisar ausencia</Text>
        </TouchableOpacity>
      </View>

      <Text style={styles.seccionLabel}>Próximas ausencias avisadas</Text>

      {cargando ? (
        <ActivityIndicator color={color.brand600} />
      ) : ausencias.length === 0 ? (
        <View style={styles.vacio}><Text style={styles.vacioTexto}>Sin ausencias avisadas</Text></View>
      ) : (
        ausencias.map((a) => (
          <View key={a.id} style={styles.filaAusencia}>
            <View>
              <Text style={styles.fechaAusencia}>{a.fecha}</Text>
              {!!a.motivo && <Text style={styles.motivoAusencia}>{a.motivo}</Text>}
            </View>
            <TouchableOpacity onPress={() => cancelar(a)} disabled={cancelandoId === a.id} style={styles.botonCancelar}>
              {cancelandoId === a.id
                ? <ActivityIndicator size="small" color={color.rose500} />
                : <Ionicons name="trash-outline" size={18} color={color.rose500} />}
            </TouchableOpacity>
          </View>
        ))
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 12, paddingBottom: 40 },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, gap: 12, ...sombra.sm },
  encabezado: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  iconoRedondo: { backgroundColor: color.brand100, padding: 8, borderRadius: radius.sm },
  tituloTarjeta: { fontSize: 11, fontWeight: '900', color: color.slate900, textTransform: 'uppercase', letterSpacing: 0.5 },
  descripcion: { fontSize: 11, color: color.slate400, fontWeight: '600', marginTop: -6 },
  filaFechas: { flexDirection: 'row', gap: 10 },
  campoFecha: { flex: 1, backgroundColor: color.slate50, borderWidth: 1, borderColor: color.slate200, borderRadius: radius.sm, padding: 12 },
  labelFecha: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase' },
  valorFecha: { fontSize: 12, fontWeight: '700', color: color.slate800, marginTop: 2 },
  input: {
    backgroundColor: color.slate50, borderWidth: 1, borderColor: color.slate200, borderRadius: radius.sm,
    paddingHorizontal: 14, paddingVertical: 12, fontSize: 12, fontWeight: '600', color: color.slate900,
  },
  boton: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'center', gap: 8,
    backgroundColor: color.brand600, borderRadius: radius.sm, paddingVertical: 14,
  },
  botonDeshabilitado: { opacity: 0.5 },
  botonTexto: { color: color.white, fontWeight: '900', fontSize: 12, textTransform: 'uppercase' },
  botonListo: { alignSelf: 'flex-end', paddingVertical: 8, paddingHorizontal: 4 },
  botonListoTexto: { color: color.brand600, fontWeight: '900', fontSize: 12, textTransform: 'uppercase' },
  seccionLabel: { fontSize: 10, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 1, marginLeft: 4, marginTop: 4 },
  vacio: { backgroundColor: color.white, borderWidth: 2, borderStyle: 'dashed', borderColor: color.slate200, borderRadius: radius.lg, padding: 24, alignItems: 'center' },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 11, textTransform: 'uppercase' },
  filaAusencia: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: color.white,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.md, padding: 16,
  },
  fechaAusencia: { fontSize: 13, fontWeight: '900', color: color.slate800 },
  motivoAusencia: { fontSize: 10, color: color.slate400, fontWeight: '700', marginTop: 2 },
  botonCancelar: { padding: 8 },
});
