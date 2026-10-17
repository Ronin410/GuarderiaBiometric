import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, ActivityIndicator, Alert, Platform } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import DateTimePicker from '@react-native-community/datetimepicker';
import api from '../../api/client';
import { hoyLocal } from '../../utils/fecha';
import { color, radius, sombra } from '../../theme';

const VACIO = { fecha: '', desayuno: true, comida: true, merienda: true, notas: '' };
const COMIDAS = [
  { key: 'desayuno', label: 'Desayuno' },
  { key: 'comida', label: 'Comida' },
  { key: 'merienda', label: 'Merienda' },
];

// Equivalente RN de la pestaña "Comedor" de VistaPadreDetalle.jsx: por
// defecto el niño come las tres comidas, aquí se avisan las excepciones
// (no desayuna, alergias, instrucciones especiales) día por día.
export default function TabComedor({ hijoId }) {
  const [pedidos, setPedidos] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [form, setForm] = useState(VACIO);
  const [mostrarPicker, setMostrarPicker] = useState(false);
  const [guardando, setGuardando] = useState(false);
  const [restableciendoFecha, setRestableciendoFecha] = useState(null);

  const cargar = useCallback(async () => {
    try {
      const res = await api.get(`/padre/hijos/${hijoId}/pedidos-comedor`);
      setPedidos(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al obtener los pedidos de comedor', err);
      setPedidos([]);
    } finally {
      setCargando(false);
    }
  }, [hijoId]);

  useEffect(() => { cargar(); }, [cargar]);

  const guardar = async () => {
    if (!form.fecha) {
      Alert.alert('Elige la fecha');
      return;
    }
    if (form.desayuno && form.comida && form.merienda && !form.notas.trim()) {
      Alert.alert('Desmarca al menos una comida o agrega una nota -- si tu hijo come normal ese día no hace falta registrar nada');
      return;
    }
    setGuardando(true);
    try {
      await api.put(`/padre/hijos/${hijoId}/pedidos-comedor/${form.fecha}`, {
        desayuno: form.desayuno, comida: form.comida, merienda: form.merienda, notas: form.notas,
      });
      setForm(VACIO);
      await cargar();
    } catch (err) {
      console.error('Error al guardar el pedido de comedor', err);
      Alert.alert('No se pudo guardar', err.response?.data?.error || 'Inténtalo de nuevo');
    } finally {
      setGuardando(false);
    }
  };

  const restablecer = (pedido) => {
    Alert.alert('¿Restablecer?', `Se volverá al comedor normal (las tres comidas) del ${pedido.fecha}.`, [
      { text: 'No', style: 'cancel' },
      {
        text: 'Sí, restablecer', onPress: async () => {
          setRestableciendoFecha(pedido.fecha);
          try {
            await api.put(`/padre/hijos/${hijoId}/pedidos-comedor/${pedido.fecha}`, { desayuno: true, comida: true, merienda: true, notas: '' });
            await cargar();
          } catch (err) {
            console.error('Error al restablecer el pedido de comedor', err);
            Alert.alert('No se pudo restablecer');
          } finally {
            setRestableciendoFecha(null);
          }
        },
      },
    ]);
  };

  const alCambiarFecha = (evento, fechaElegida) => {
    if (Platform.OS === 'android') {
      setMostrarPicker(false);
      if (evento.type === 'dismissed' || !fechaElegida) return;
    }
    if (!fechaElegida) return;
    setForm((f) => ({ ...f, fecha: fechaElegida.toLocaleDateString('en-CA') }));
  };

  return (
    <ScrollView style={styles.pantalla} contentContainerStyle={styles.contenido}>
      <View style={styles.tarjeta}>
        <View style={styles.encabezado}>
          <View style={styles.iconoRedondo}><Ionicons name="restaurant" size={18} color={color.brand600} /></View>
          <Text style={styles.tituloTarjeta}>Pedidos de Comedor</Text>
        </View>
        <Text style={styles.descripcion}>
          Por defecto tu hijo come las tres comidas del día. Avísale a la guardería solo cuando algo cambie.
        </Text>

        <TouchableOpacity style={styles.campoFecha} onPress={() => setMostrarPicker(true)}>
          <Text style={styles.labelFecha}>Fecha</Text>
          <Text style={styles.valorFecha}>{form.fecha || 'Elegir'}</Text>
        </TouchableOpacity>

        {mostrarPicker && (
          <View>
            <DateTimePicker
              value={new Date(`${form.fecha || hoyLocal()}T00:00:00`)}
              mode="date"
              display={Platform.OS === 'ios' ? 'spinner' : 'default'}
              minimumDate={new Date(`${hoyLocal()}T00:00:00`)}
              onChange={alCambiarFecha}
            />
            {Platform.OS === 'ios' && (
              <TouchableOpacity style={styles.botonListo} onPress={() => setMostrarPicker(false)}>
                <Text style={styles.botonListoTexto}>Listo</Text>
              </TouchableOpacity>
            )}
          </View>
        )}

        <View style={styles.filaChecks}>
          {COMIDAS.map((c) => (
            <TouchableOpacity key={c.key} style={styles.check} onPress={() => setForm((f) => ({ ...f, [c.key]: !f[c.key] }))}>
              <View style={[styles.checkbox, form[c.key] && styles.checkboxMarcado]}>
                {form[c.key] && <Ionicons name="checkmark" size={12} color={color.white} />}
              </View>
              <Text style={styles.checkLabel}>{c.label}</Text>
            </TouchableOpacity>
          ))}
        </View>

        <TextInput
          style={styles.input}
          placeholder="Notas (opcional), ej. Alergia a los cacahuates"
          placeholderTextColor={color.slate400}
          value={form.notas}
          onChangeText={(notas) => setForm((f) => ({ ...f, notas }))}
        />

        <TouchableOpacity style={[styles.boton, guardando && styles.botonDeshabilitado]} onPress={guardar} disabled={guardando}>
          {guardando ? <ActivityIndicator color={color.white} size="small" /> : <Ionicons name="add" size={16} color={color.white} />}
          <Text style={styles.botonTexto}>Avisar</Text>
        </TouchableOpacity>
      </View>

      <Text style={styles.seccionLabel}>Excepciones registradas</Text>

      {cargando ? (
        <ActivityIndicator color={color.brand600} />
      ) : pedidos.length === 0 ? (
        <View style={styles.vacio}><Text style={styles.vacioTexto}>Sin excepciones, come normal todos los días</Text></View>
      ) : (
        pedidos.map((p) => {
          const faltantes = [!p.desayuno && 'Desayuno', !p.comida && 'Comida', !p.merienda && 'Merienda'].filter(Boolean);
          return (
            <View key={p.id} style={styles.filaPedido}>
              <View style={{ flex: 1 }}>
                <Text style={styles.fechaPedido}>{p.fecha}</Text>
                {faltantes.length > 0 && <Text style={styles.faltantes}>No come: {faltantes.join(', ')}</Text>}
                {!!p.notas && <Text style={styles.notasPedido}>{p.notas}</Text>}
              </View>
              <TouchableOpacity onPress={() => restablecer(p)} disabled={restableciendoFecha === p.fecha} style={styles.botonRestablecer}>
                {restableciendoFecha === p.fecha
                  ? <ActivityIndicator size="small" color={color.brand600} />
                  : <Ionicons name="refresh" size={18} color={color.brand600} />}
              </TouchableOpacity>
            </View>
          );
        })
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
  campoFecha: { backgroundColor: color.slate50, borderWidth: 1, borderColor: color.slate200, borderRadius: radius.sm, padding: 12 },
  labelFecha: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase' },
  valorFecha: { fontSize: 12, fontWeight: '700', color: color.slate800, marginTop: 2 },
  botonListo: { alignSelf: 'flex-end', paddingVertical: 8, paddingHorizontal: 4 },
  botonListoTexto: { color: color.brand600, fontWeight: '900', fontSize: 12, textTransform: 'uppercase' },
  filaChecks: { flexDirection: 'row', gap: 20 },
  check: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  checkbox: { width: 18, height: 18, borderRadius: 5, borderWidth: 2, borderColor: color.slate300, alignItems: 'center', justifyContent: 'center' },
  checkboxMarcado: { backgroundColor: color.brand600, borderColor: color.brand600 },
  checkLabel: { fontSize: 11, fontWeight: '700', color: color.slate600 },
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
  seccionLabel: { fontSize: 10, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 1, marginLeft: 4, marginTop: 4 },
  vacio: { backgroundColor: color.white, borderWidth: 2, borderStyle: 'dashed', borderColor: color.slate200, borderRadius: radius.lg, padding: 24, alignItems: 'center' },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 11, textTransform: 'uppercase', textAlign: 'center' },
  filaPedido: {
    flexDirection: 'row', alignItems: 'center', gap: 10, backgroundColor: color.white,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.md, padding: 16,
  },
  fechaPedido: { fontSize: 13, fontWeight: '900', color: color.slate800 },
  faltantes: { fontSize: 10, color: color.rose500, fontWeight: '700', marginTop: 2 },
  notasPedido: { fontSize: 10, color: color.slate400, fontWeight: '700', marginTop: 2 },
  botonRestablecer: { padding: 8 },
});
