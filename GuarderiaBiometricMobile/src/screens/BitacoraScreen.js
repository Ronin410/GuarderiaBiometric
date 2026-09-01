import React, { useCallback, useEffect, useState } from 'react';
import {
  View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator,
  Image, Modal, Dimensions,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { hoyLocal, sumarDias, formatoLargo } from '../utils/fecha';
import { color, radius, sombra } from '../theme';

const COLOR_COMIDA = { desayuno: color.emerald500, comida: color.amber500, merienda: color.rose500 };

// Equivalente RN de components/ReporteDiario.jsx + el selector de fecha de
// VistaPadreDetalle.jsx (web) -- por ahora solo la pestaña "Hoy" (bitácora
// del día), que es la función central que ya describe el propio dashboard
// ("las bitácoras se actualizan en tiempo real"). Expediente/pagos/
// ausencias/comedor/galería quedan para la siguiente pasada, ver
// API_MOVIL.md.
export default function BitacoraScreen({ route, navigation }) {
  const { hijoId, nombreHijo } = route.params;
  const [fecha, setFecha] = useState(hoyLocal());
  const [reporte, setReporte] = useState(null);
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  const [fotoAmpliada, setFotoAmpliada] = useState(null);

  useEffect(() => {
    navigation.setOptions({ title: nombreHijo });
  }, [navigation, nombreHijo]);

  const cargar = useCallback(async () => {
    setCargando(true);
    try {
      const res = await api.get(`/seguimiento/${hijoId}`, { params: { fecha } });
      setReporte(res.data);
      setError('');
    } catch (err) {
      setReporte(null);
      setError(err.response?.data?.error || 'No hay reporte para esta fecha.');
    } finally {
      setCargando(false);
    }
  }, [hijoId, fecha]);

  useEffect(() => { cargar(); }, [cargar]);

  const esHoy = fecha === hoyLocal();

  return (
    <ScrollView style={styles.pantalla} contentContainerStyle={styles.contenido}>
      <View style={styles.selectorFecha}>
        <TouchableOpacity style={styles.flechaFecha} onPress={() => setFecha((f) => sumarDias(f, -1))}>
          <Ionicons name="chevron-back" size={20} color={color.brand600} />
        </TouchableOpacity>
        <Text style={styles.textoFecha}>{esHoy ? 'Hoy' : formatoLargo(fecha)}</Text>
        <TouchableOpacity
          style={[styles.flechaFecha, esHoy && styles.flechaFechaDeshabilitada]}
          onPress={() => !esHoy && setFecha((f) => sumarDias(f, 1))}
          disabled={esHoy}
        >
          <Ionicons name="chevron-forward" size={20} color={esHoy ? color.slate300 : color.brand600} />
        </TouchableOpacity>
      </View>

      {cargando ? (
        <ActivityIndicator color={color.brand600} style={{ marginTop: 40 }} />
      ) : error ? (
        <View style={styles.vacio}>
          <Ionicons name="time-outline" size={32} color={color.slate300} />
          <Text style={styles.vacioTexto}>{error}</Text>
        </View>
      ) : (
        <>
          <View style={styles.filaEntradaSalida}>
            <View style={[styles.cajaEntradaSalida, reporte.hora_entrada && styles.cajaEntrada]}>
              <View style={[styles.iconoEntradaSalida, reporte.hora_entrada && styles.iconoEntradaActivo]}>
                <Ionicons name="log-in" size={16} color={reporte.hora_entrada ? color.white : color.slate300} />
              </View>
              <Text style={styles.labelEntradaSalida}>Entrada</Text>
              <Text style={[styles.valorEntradaSalida, reporte.hora_entrada ? { color: color.emerald700 } : styles.valorVacio]}>
                {reporte.hora_entrada || 'Sin registro'}
              </Text>
            </View>
            <View style={[styles.cajaEntradaSalida, reporte.hora_salida && styles.cajaSalida]}>
              <View style={[styles.iconoEntradaSalida, reporte.hora_salida && styles.iconoSalidaActivo]}>
                <Ionicons name="log-out" size={16} color={reporte.hora_salida ? color.white : color.slate300} />
              </View>
              <Text style={styles.labelEntradaSalida}>Salida</Text>
              <Text style={[styles.valorEntradaSalida, reporte.hora_salida ? { color: color.amber700 } : styles.valorVacio]}>
                {reporte.hora_salida || (reporte.hora_entrada ? 'Sigue aquí' : 'Sin registro')}
              </Text>
            </View>
          </View>

          <View style={styles.tarjeta}>
            <View style={styles.tarjetaEncabezado}>
              <View style={[styles.iconoRedondo, { backgroundColor: color.amber50 }]}>
                <Ionicons name="restaurant" size={18} color={color.amber600} />
              </View>
              <Text style={styles.tarjetaTitulo}>Alimentación</Text>
            </View>
            <ItemComida label="Desayuno" valor={reporte.desayuno} punto={COLOR_COMIDA.desayuno} />
            <ItemComida label="Comida" valor={reporte.comida} punto={COLOR_COMIDA.comida} />
            <ItemComida label="Merienda" valor={reporte.merienda} punto={COLOR_COMIDA.merienda} />
          </View>

          <View style={[styles.tarjeta, styles.tarjetaDescanso, reporte.durmio && styles.tarjetaDescansoActiva]}>
            <View style={styles.filaDescanso}>
              <View style={styles.filaDescansoIzq}>
                <View style={[styles.iconoDescanso, reporte.durmio && styles.iconoDescansoActivo]}>
                  <Ionicons name="moon" size={22} color={reporte.durmio ? color.white : color.slate300} />
                </View>
                <View>
                  <Text style={[styles.descansoTitulo, reporte.durmio && { color: color.white }]}>Descanso</Text>
                  <Text style={[styles.descansoSubtitulo, reporte.durmio && { color: color.white }]}>
                    {reporte.durmio ? 'Tomó su siesta' : 'No hubo siesta'}
                  </Text>
                </View>
              </View>
              {reporte.durmio && <Ionicons name="checkmark-circle" size={26} color={color.white} />}
            </View>
          </View>

          <View style={[styles.tarjeta, styles.filaSimple]}>
            <View style={[styles.iconoRedondo, { backgroundColor: color.emerald50 }]}>
              <Ionicons name="body" size={22} color={color.emerald600} />
            </View>
            <View style={{ flex: 1 }}>
              <Text style={styles.esfinterLabel}>Control de Esfínter</Text>
              <Text style={styles.esfinterValor}>{reporte.esfinter || 'Sin reporte'}</Text>
            </View>
          </View>

          {Array.isArray(reporte.fotos) && reporte.fotos.length > 0 && (
            <View style={styles.tarjeta}>
              <View style={styles.tarjetaEncabezado}>
                <View style={[styles.iconoRedondo, { backgroundColor: color.rose50 }]}>
                  <Ionicons name="camera" size={18} color={color.rose600} />
                </View>
                <Text style={styles.tarjetaTitulo}>Evidencia del día</Text>
              </View>
              <View style={styles.grillaFotos}>
                {reporte.fotos.map((url, idx) => (
                  <TouchableOpacity key={idx} style={styles.fotoMini} onPress={() => setFotoAmpliada(url)}>
                    <Image source={{ uri: url }} style={styles.fotoMiniImg} />
                  </TouchableOpacity>
                ))}
              </View>
            </View>
          )}

          {!!reporte.observaciones && (
            <View style={styles.tarjetaObservaciones}>
              <Text style={styles.observacionesTitulo}>Nota de la Maestra</Text>
              <Text style={styles.observacionesTexto}>"{reporte.observaciones}"</Text>
            </View>
          )}
        </>
      )}

      <Modal visible={!!fotoAmpliada} transparent animationType="fade" onRequestClose={() => setFotoAmpliada(null)}>
        <TouchableOpacity style={styles.modalFondo} activeOpacity={1} onPress={() => setFotoAmpliada(null)}>
          {!!fotoAmpliada && <Image source={{ uri: fotoAmpliada }} style={styles.modalImagen} resizeMode="contain" />}
        </TouchableOpacity>
      </Modal>
    </ScrollView>
  );
}

const ItemComida = ({ label, valor, punto }) => (
  <View style={styles.filaComida}>
    <View style={[styles.puntoComida, { backgroundColor: valor ? punto : color.slate200 }]} />
    <View style={{ flex: 1 }}>
      <Text style={styles.labelComida}>{label}</Text>
      <Text style={[styles.valorComida, !valor && styles.valorVacioItalic]}>{valor || 'Pendiente'}</Text>
    </View>
  </View>
);

const anchoPantalla = Dimensions.get('window').width;
const anchoFoto = (anchoPantalla - 40 - 40 - 12) / 2; // padding pantalla + padding tarjeta + gap

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, paddingBottom: 40, gap: 12 },
  selectorFecha: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: color.white,
    borderRadius: radius.lg, padding: 8, paddingHorizontal: 16, ...sombra.sm,
  },
  flechaFecha: { padding: 8 },
  flechaFechaDeshabilitada: { opacity: 0.4 },
  textoFecha: { fontWeight: '900', color: color.slate700, fontSize: 13, textTransform: 'capitalize' },
  vacio: { backgroundColor: color.white, borderWidth: 2, borderStyle: 'dashed', borderColor: color.slate200, borderRadius: radius.xl, padding: 40, alignItems: 'center', gap: 12, marginTop: 8 },
  vacioTexto: { color: color.slate400, fontWeight: '800', fontSize: 11, textTransform: 'uppercase', textAlign: 'center' },
  filaEntradaSalida: { flexDirection: 'row', gap: 12 },
  cajaEntradaSalida: { flex: 1, backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 18 },
  cajaEntrada: { backgroundColor: color.emerald50, borderColor: color.emerald100 },
  cajaSalida: { backgroundColor: color.amber50, borderColor: '#fde68a' },
  iconoEntradaSalida: { width: 36, height: 36, borderRadius: radius.sm, backgroundColor: color.slate50, alignItems: 'center', justifyContent: 'center', marginBottom: 10 },
  iconoEntradaActivo: { backgroundColor: color.emerald500 },
  iconoSalidaActivo: { backgroundColor: color.amber500 },
  labelEntradaSalida: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 0.5 },
  valorEntradaSalida: { fontSize: 16, fontWeight: '900', marginTop: 2 },
  valorVacio: { color: color.slate300, fontStyle: 'italic', fontSize: 13 },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, ...sombra.sm },
  tarjetaEncabezado: { flexDirection: 'row', alignItems: 'center', gap: 12, marginBottom: 16 },
  iconoRedondo: { padding: 8, borderRadius: radius.sm },
  tarjetaTitulo: { fontSize: 11, fontWeight: '900', color: color.slate900, textTransform: 'uppercase', letterSpacing: 0.5 },
  filaComida: { flexDirection: 'row', alignItems: 'center', gap: 14, marginBottom: 14 },
  puntoComida: { width: 12, height: 12, borderRadius: 6 },
  labelComida: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 2 },
  valorComida: { fontSize: 13, fontWeight: '900', color: color.slate800, textTransform: 'uppercase' },
  valorVacioItalic: { color: color.slate300, fontStyle: 'italic' },
  tarjetaDescanso: { backgroundColor: color.white },
  tarjetaDescansoActiva: { backgroundColor: color.brand600, borderColor: color.brand500 },
  filaDescanso: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  filaDescansoIzq: { flexDirection: 'row', alignItems: 'center', gap: 14 },
  iconoDescanso: { width: 48, height: 48, borderRadius: radius.md, backgroundColor: color.slate50, alignItems: 'center', justifyContent: 'center' },
  iconoDescansoActivo: { backgroundColor: 'rgba(255,255,255,0.2)' },
  descansoTitulo: { fontSize: 12, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 0.5 },
  descansoSubtitulo: { fontSize: 10, fontWeight: '700', color: color.slate400, textTransform: 'uppercase', marginTop: 2 },
  filaSimple: { flexDirection: 'row', alignItems: 'center', gap: 16 },
  esfinterLabel: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 4 },
  esfinterValor: { fontSize: 14, fontWeight: '700', color: color.slate800 },
  grillaFotos: { flexDirection: 'row', flexWrap: 'wrap', gap: 12 },
  fotoMini: { width: anchoFoto, height: anchoFoto, borderRadius: radius.md, overflow: 'hidden', backgroundColor: color.slate100 },
  fotoMiniImg: { width: '100%', height: '100%' },
  tarjetaObservaciones: { backgroundColor: color.slate900, borderRadius: radius.lg, padding: 24 },
  observacionesTitulo: { fontSize: 10, fontWeight: '900', color: color.brand300, textTransform: 'uppercase', letterSpacing: 1.5, marginBottom: 12 },
  observacionesTexto: { fontSize: 14, color: color.white, fontStyle: 'italic', lineHeight: 20 },
  modalFondo: { flex: 1, backgroundColor: 'rgba(15,23,42,0.92)', alignItems: 'center', justifyContent: 'center' },
  modalImagen: { width: '92%', height: '80%' },
});
