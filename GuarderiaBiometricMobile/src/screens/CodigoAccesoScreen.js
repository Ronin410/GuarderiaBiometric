import React, { useCallback, useEffect, useState } from 'react';
import {
  View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, ActivityIndicator, Alert, RefreshControl,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import QRCode from 'react-native-qrcode-svg';
import api from '../api/client';
import { fechaLocal, hoyLocal } from '../utils/fecha';
import { color, radius, sombra } from '../theme';

// Equivalente RN de CodigosAccesoPadre.jsx (web): el papá genera, nombra y
// revoca sus propios códigos de acceso QR/alfanuméricos -- para entrar él
// mismo sin rostro, o para delegar la entrada/salida en alguien que no está
// enrolado biométricamente (un abuelo, un chofer). Ver codigos_acceso.go en
// el backend para el contrato exacto de cada endpoint.
const OPCIONES_DIAS = [1, 3, 7];

export default function CodigoAccesoScreen() {
  const [hijos, setHijos] = useState([]);
  const [codigos, setCodigos] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [refrescando, setRefrescando] = useState(false);

  const [hijosSeleccionados, setHijosSeleccionados] = useState([]);
  const [personaAutorizada, setPersonaAutorizada] = useState('');
  const [dias, setDias] = useState(1);
  const [generando, setGenerando] = useState(false);
  const [codigoNuevo, setCodigoNuevo] = useState(null); // { token, codigo_corto, expira_en }
  const [revocandoId, setRevocandoId] = useState(null);

  const cargar = useCallback(async () => {
    try {
      const [resHijos, resCodigos] = await Promise.all([
        api.get('/padre/0/hijos'),
        api.get('/padre/codigos-acceso'),
      ]);
      setHijos((resHijos.data || []).map((h) => ({ id: h.id, nombre: h.nombre_niño || h.nombre || 'Sin nombre' })));
      setCodigos(Array.isArray(resCodigos.data) ? resCodigos.data : []);
    } catch (err) {
      console.error('Error al cargar códigos de acceso', err);
    } finally {
      setCargando(false);
      setRefrescando(false);
    }
  }, []);

  useEffect(() => { cargar(); }, [cargar]);

  const refrescar = () => {
    setRefrescando(true);
    cargar();
  };

  const alternarHijo = (hijoId) => {
    setHijosSeleccionados((prev) => (
      prev.includes(hijoId) ? prev.filter((id) => id !== hijoId) : [...prev, hijoId]
    ));
  };

  const generar = async () => {
    if (hijosSeleccionados.length === 0) {
      Alert.alert('Elige al menos un niño');
      return;
    }
    if (personaAutorizada.trim().length < 2) {
      Alert.alert('Escribe el nombre de quién va a usar este código');
      return;
    }
    setGenerando(true);
    try {
      const res = await api.post('/padre/codigos-acceso', {
        hijos_ids: hijosSeleccionados,
        persona_autorizada: personaAutorizada.trim(),
        dias,
      });
      setCodigoNuevo(res.data);
      setPersonaAutorizada('');
      setHijosSeleccionados([]);
      await cargar();
    } catch (err) {
      console.error('Error al generar el código de acceso', err);
      Alert.alert('No se pudo generar', err.response?.data?.error || 'Inténtalo de nuevo');
    } finally {
      setGenerando(false);
    }
  };

  const revocar = (codigo) => {
    Alert.alert('¿Revocar este código?', `Dejará de funcionar de inmediato para ${codigo.persona_autorizada}.`, [
      { text: 'No', style: 'cancel' },
      {
        text: 'Sí, revocar', style: 'destructive', onPress: async () => {
          setRevocandoId(codigo.id);
          try {
            await api.delete(`/padre/codigos-acceso/${codigo.id}`);
            await cargar();
          } catch (err) {
            console.error('Error al revocar el código', err);
            Alert.alert('No se pudo revocar');
          } finally {
            setRevocandoId(null);
          }
        },
      },
    ]);
  };

  const estadoDe = (codigo) => {
    if (codigo.revocado_en) return { texto: 'Revocado', color: color.slate400, fondo: color.slate100 };
    if (fechaLocal(codigo.expira_en) < hoyLocal()) return { texto: 'Vencido', color: color.slate400, fondo: color.slate100 };
    return { texto: 'Vigente', color: color.emerald600, fondo: color.emerald50 };
  };

  return (
    <ScrollView
      style={styles.pantalla}
      contentContainerStyle={styles.contenido}
      refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
    >
      <Text style={styles.descripcionGeneral}>
        Genera un código para entrar sin tu rostro, o para delegarle la entrada/salida a alguien que no está
        registrado en el sistema (un familiar, un chofer). Tú decides a quién, para quién y por cuánto tiempo.
      </Text>

      {codigoNuevo ? (
        <View style={styles.tarjetaGenerado}>
          <View style={styles.encabezado}>
            <Ionicons name="checkmark-circle" size={20} color={color.emerald600} />
            <Text style={styles.tituloGenerado}>Código listo</Text>
          </View>
          <View style={styles.qrContenedor}>
            <QRCode value={codigoNuevo.token} size={180} />
          </View>
          <Text style={styles.codigoCortoLabel}>O dicta este código en recepción:</Text>
          <Text style={styles.codigoCorto}>{codigoNuevo.codigo_corto}</Text>
          <Text style={styles.vigenciaTexto}>Vigente hasta {fechaLocal(codigoNuevo.expira_en)}</Text>
          <TouchableOpacity style={styles.botonCerrarGenerado} onPress={() => setCodigoNuevo(null)}>
            <Text style={styles.botonCerrarGeneradoTexto}>Generar otro código</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <View style={styles.tarjeta}>
          <View style={styles.encabezado}>
            <View style={styles.iconoRedondo}><Ionicons name="qr-code" size={18} color={color.brand600} /></View>
            <Text style={styles.tituloTarjeta}>Generar código nuevo</Text>
          </View>

          <Text style={styles.label}>¿Para cuáles niños?</Text>
          <View style={styles.chips}>
            {hijos.map((h) => {
              const activo = hijosSeleccionados.includes(h.id);
              return (
                <TouchableOpacity
                  key={h.id}
                  style={[styles.chip, activo && styles.chipActivo]}
                  onPress={() => alternarHijo(h.id)}
                >
                  <Text style={[styles.chipTexto, activo && styles.chipTextoActivo]}>{h.nombre}</Text>
                </TouchableOpacity>
              );
            })}
          </View>

          <Text style={styles.label}>¿Quién lo va a usar?</Text>
          <TextInput
            style={styles.input}
            placeholder="Nombre de la persona autorizada"
            placeholderTextColor={color.slate400}
            value={personaAutorizada}
            onChangeText={setPersonaAutorizada}
          />

          <Text style={styles.label}>Vigencia</Text>
          <View style={styles.segmentado}>
            {OPCIONES_DIAS.map((d) => (
              <TouchableOpacity
                key={d}
                style={[styles.segmento, dias === d && styles.segmentoActivo]}
                onPress={() => setDias(d)}
              >
                <Text style={[styles.segmentoTexto, dias === d && styles.segmentoTextoActivo]}>
                  {d} {d === 1 ? 'día' : 'días'}
                </Text>
              </TouchableOpacity>
            ))}
          </View>

          <TouchableOpacity style={[styles.boton, generando && styles.botonDeshabilitado]} onPress={generar} disabled={generando}>
            {generando ? <ActivityIndicator color={color.white} size="small" /> : <Ionicons name="qr-code-outline" size={16} color={color.white} />}
            <Text style={styles.botonTexto}>Generar código</Text>
          </TouchableOpacity>
        </View>
      )}

      <Text style={styles.seccionLabel}>Tus códigos</Text>

      {cargando ? (
        <ActivityIndicator color={color.brand600} />
      ) : codigos.length === 0 ? (
        <View style={styles.vacio}><Text style={styles.vacioTexto}>Todavía no has generado ningún código</Text></View>
      ) : (
        codigos.map((c) => {
          const estado = estadoDe(c);
          const puedeRevocar = !c.revocado_en && fechaLocal(c.expira_en) >= hoyLocal();
          return (
            <View key={c.id} style={styles.filaCodigo}>
              <View style={styles.filaCodigoTexto}>
                <View style={styles.filaCodigoEncabezado}>
                  <Text style={styles.personaAutorizada}>{c.persona_autorizada}</Text>
                  <View style={[styles.badge, { backgroundColor: estado.fondo }]}>
                    <Text style={[styles.badgeTexto, { color: estado.color }]}>{estado.texto}</Text>
                  </View>
                </View>
                <Text style={styles.detalleCodigo}>{c.hijos || 'Sin niños vinculados'}</Text>
                <Text style={styles.detalleCodigo}>Código: {c.codigo_corto} · Vence {fechaLocal(c.expira_en)}</Text>
              </View>
              {puedeRevocar && (
                <TouchableOpacity onPress={() => revocar(c)} disabled={revocandoId === c.id} style={styles.botonRevocar}>
                  {revocandoId === c.id
                    ? <ActivityIndicator size="small" color={color.rose500} />
                    : <Ionicons name="close-circle-outline" size={20} color={color.rose500} />}
                </TouchableOpacity>
              )}
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
  descripcionGeneral: { fontSize: 12, color: color.slate500, fontWeight: '600', lineHeight: 18 },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, gap: 10, ...sombra.sm },
  tarjetaGenerado: { backgroundColor: color.white, borderWidth: 1, borderColor: color.emerald200, borderRadius: radius.lg, padding: 20, gap: 10, alignItems: 'center', ...sombra.sm },
  encabezado: { flexDirection: 'row', alignItems: 'center', gap: 10, alignSelf: 'flex-start' },
  iconoRedondo: { backgroundColor: color.brand100, padding: 8, borderRadius: radius.sm },
  tituloTarjeta: { fontSize: 11, fontWeight: '900', color: color.slate900, textTransform: 'uppercase', letterSpacing: 0.5 },
  tituloGenerado: { fontSize: 13, fontWeight: '900', color: color.emerald700, textTransform: 'uppercase' },
  qrContenedor: { backgroundColor: color.white, padding: 16, borderRadius: radius.md, borderWidth: 1, borderColor: color.slate100, marginTop: 4 },
  codigoCortoLabel: { fontSize: 10, color: color.slate400, fontWeight: '700', textTransform: 'uppercase', marginTop: 8 },
  codigoCorto: { fontSize: 24, fontWeight: '900', color: color.slate900, letterSpacing: 4 },
  vigenciaTexto: { fontSize: 11, color: color.slate400, fontWeight: '700' },
  botonCerrarGenerado: { marginTop: 8, paddingVertical: 10 },
  botonCerrarGeneradoTexto: { color: color.brand600, fontWeight: '900', fontSize: 11, textTransform: 'uppercase' },
  label: { fontSize: 10, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 0.5, marginTop: 4 },
  chips: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { backgroundColor: color.slate50, borderWidth: 1, borderColor: color.slate200, borderRadius: radius.full, paddingVertical: 8, paddingHorizontal: 14 },
  chipActivo: { backgroundColor: color.brand600, borderColor: color.brand600 },
  chipTexto: { fontSize: 11, fontWeight: '800', color: color.slate500, textTransform: 'uppercase' },
  chipTextoActivo: { color: color.white },
  input: {
    backgroundColor: color.slate50, borderWidth: 1, borderColor: color.slate200, borderRadius: radius.sm,
    paddingHorizontal: 14, paddingVertical: 12, fontSize: 12, fontWeight: '600', color: color.slate900,
  },
  segmentado: { flexDirection: 'row', backgroundColor: color.slate100, borderRadius: radius.sm, padding: 4, gap: 4 },
  segmento: { flex: 1, paddingVertical: 10, borderRadius: radius.sm - 4, alignItems: 'center' },
  segmentoActivo: { backgroundColor: color.white, ...sombra.sm },
  segmentoTexto: { fontSize: 11, fontWeight: '800', color: color.slate400, textTransform: 'uppercase' },
  segmentoTextoActivo: { color: color.brand600 },
  boton: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'center', gap: 8,
    backgroundColor: color.brand600, borderRadius: radius.sm, paddingVertical: 14, marginTop: 4,
  },
  botonDeshabilitado: { opacity: 0.5 },
  botonTexto: { color: color.white, fontWeight: '900', fontSize: 12, textTransform: 'uppercase' },
  seccionLabel: { fontSize: 10, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 1, marginLeft: 4, marginTop: 4 },
  vacio: { backgroundColor: color.white, borderWidth: 2, borderStyle: 'dashed', borderColor: color.slate200, borderRadius: radius.lg, padding: 24, alignItems: 'center' },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 11, textTransform: 'uppercase' },
  filaCodigo: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: color.white,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.md, padding: 16, gap: 10,
  },
  filaCodigoTexto: { flex: 1, gap: 2 },
  filaCodigoEncabezado: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  personaAutorizada: { fontSize: 13, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  badge: { borderRadius: radius.full, paddingHorizontal: 8, paddingVertical: 3 },
  badgeTexto: { fontSize: 8, fontWeight: '900', textTransform: 'uppercase' },
  detalleCodigo: { fontSize: 10, color: color.slate400, fontWeight: '700' },
  botonRevocar: { padding: 8 },
});
