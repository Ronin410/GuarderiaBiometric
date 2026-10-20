import React, { useRef, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, ActivityIndicator, Alert } from 'react-native';
import { CameraView, useCameraPermissions } from 'expo-camera';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { color, radius, sombra } from '../theme';

// Equivalente RN de EscanearQRKioscoPadre.jsx (web): el papá apunta la
// cámara de SU PROPIO celular al QR anónimo de un solo uso que muestra la
// tablet de Recepción (modo "Autoservicio") y confirma entrada/salida de
// sus hijos ahí mismo, sin esperar turno. El código no identifica a nadie
// -- la identidad sale siempre de la sesión de quien escanea (ver
// codigos_kiosco.go).
export default function EscanearQRScreen() {
  const [permiso, solicitarPermiso] = useCameraPermissions();
  const escaneandoRef = useRef(false);

  const [verificando, setVerificando] = useState(false);
  const [errorCodigo, setErrorCodigo] = useState('');
  const [resultado, setResultado] = useState(null); // { codigo_kiosco_id, hijos: [...] }
  const [expandido, setExpandido] = useState(null);
  const [formHijo, setFormHijo] = useState({});
  const [confirmando, setConfirmando] = useState(null);
  const [confirmados, setConfirmados] = useState([]);

  const canjear = async ({ data: token }) => {
    if (escaneandoRef.current || resultado) return;
    escaneandoRef.current = true;
    setVerificando(true);
    setErrorCodigo('');
    try {
      const res = await api.post('/padre/codigo-kiosco/canjear', { token });
      setResultado(res.data);
      setConfirmados([]);
      setExpandido(null);
      setFormHijo({});
    } catch (err) {
      setErrorCodigo(err.response?.data?.mensaje || err.response?.data?.error || 'Código no válido');
      // Se deja reintentar en 2s -- el mismo criterio que la versión web
      // (evita que un solo frame borroso bloquee el lector para siempre,
      // pero tampoco reintenta en cada frame mientras el QR sigue en cuadro).
      setTimeout(() => { escaneandoRef.current = false; }, 2000);
    } finally {
      setVerificando(false);
    }
  };

  const confirmar = async (hijo) => {
    const datos = formHijo[hijo.id] || {};
    setConfirmando(hijo.id);
    try {
      await api.post('/padre/confirmar-asistencia', {
        hijo_id: hijo.id,
        codigo_kiosco_id: resultado.codigo_kiosco_id,
        aseado: datos.aseado || false,
        reporte_golpe: datos.golpe || false,
        observaciones: datos.observaciones || '',
      });
      setConfirmados((prev) => [...prev, hijo.id]);
      setExpandido(null);
    } catch (err) {
      console.error('Error al confirmar asistencia', err);
      Alert.alert('No se pudo confirmar', err.response?.data?.error || 'Inténtalo de nuevo');
    } finally {
      setConfirmando(null);
    }
  };

  const volverAEscanear = () => {
    setResultado(null);
    setErrorCodigo('');
    setConfirmados([]);
    setExpandido(null);
    setFormHijo({});
    escaneandoRef.current = false;
  };

  const actualizarForm = (hijoId, campo, valor) => {
    setFormHijo((prev) => ({ ...prev, [hijoId]: { ...prev[hijoId], [campo]: valor } }));
  };

  if (!permiso) {
    return <View style={styles.pantalla} />;
  }

  if (!permiso.granted) {
    return (
      <View style={[styles.pantalla, styles.centrado]}>
        <Ionicons name="camera-outline" size={40} color={color.slate300} />
        <Text style={styles.permisoTitulo}>Se necesita la cámara</Text>
        <Text style={styles.permisoTexto}>
          Para escanear el QR de la tablet de recepción, Pasitos necesita acceso a tu cámara.
        </Text>
        <TouchableOpacity style={styles.boton} onPress={solicitarPermiso}>
          <Text style={styles.botonTexto}>Dar permiso</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (resultado) {
    return (
      <ScrollView style={styles.pantalla} contentContainerStyle={styles.contenido}>
        <View style={styles.bannerVerificado}>
          <Ionicons name="checkmark-circle" size={20} color={color.emerald600} />
          <Text style={styles.bannerTexto}>Código verificado. Elige a quién le das entrada o salida.</Text>
        </View>

        {resultado.hijos.map((h) => {
          const hijoId = h.id || h.hijo_id;
          const yaConfirmado = confirmados.includes(hijoId);
          const esSalida = h.ultimo_estado === 'ENTRADA';
          const abierto = expandido === hijoId;
          const datos = formHijo[hijoId] || {};
          return (
            <View key={hijoId} style={styles.tarjetaHijo}>
              <View style={styles.filaHijo}>
                <View style={styles.filaHijoTexto}>
                  <Text style={styles.nombreHijo}>{h.nombre_niño || h.nombre}</Text>
                  <Text style={styles.estadoHijo}>
                    {h.ultimo_estado === 'AUSENTE' ? 'Sin registro hoy' : `Último: ${h.ultimo_estado}`}
                  </Text>
                </View>
                {yaConfirmado ? (
                  <View style={styles.listoFila}>
                    <Ionicons name="checkmark-circle" size={16} color={color.emerald600} />
                    <Text style={styles.listoTexto}>Listo</Text>
                  </View>
                ) : (
                  <TouchableOpacity
                    style={[styles.botonMovimiento, esSalida ? styles.botonSalida : styles.botonEntrada, abierto && styles.botonCerrar]}
                    onPress={() => setExpandido(abierto ? null : hijoId)}
                  >
                    <Text style={[styles.botonMovimientoTexto, abierto && styles.botonCerrarTexto]}>
                      {abierto ? 'Cerrar' : esSalida ? 'Salida' : 'Entrada'}
                    </Text>
                  </TouchableOpacity>
                )}
              </View>

              {!yaConfirmado && abierto && (
                <View style={styles.formHijo}>
                  {!esSalida && (
                    <View style={styles.filaToggles}>
                      <TouchableOpacity
                        style={[styles.toggle, datos.aseado && styles.toggleAseadoActivo]}
                        onPress={() => actualizarForm(hijoId, 'aseado', !datos.aseado)}
                      >
                        <Text style={[styles.toggleTexto, datos.aseado && styles.toggleTextoActivo]}>
                          {datos.aseado ? 'Aseado ✓' : '¿Aseado?'}
                        </Text>
                      </TouchableOpacity>
                      <TouchableOpacity
                        style={[styles.toggle, datos.golpe && styles.toggleGolpeActivo]}
                        onPress={() => actualizarForm(hijoId, 'golpe', !datos.golpe)}
                      >
                        <Text style={[styles.toggleTexto, datos.golpe && styles.toggleTextoActivo]}>
                          {datos.golpe ? 'Golpes !' : '¿Golpes?'}
                        </Text>
                      </TouchableOpacity>
                    </View>
                  )}
                  <TextInput
                    style={styles.inputObs}
                    placeholder="Observaciones para la bitácora (opcional)..."
                    placeholderTextColor={color.slate400}
                    value={datos.observaciones || ''}
                    onChangeText={(v) => actualizarForm(hijoId, 'observaciones', v)}
                    multiline
                  />
                  <TouchableOpacity
                    style={[styles.botonConfirmar, esSalida ? styles.botonSalida : styles.botonEntrada]}
                    onPress={() => confirmar(h)}
                    disabled={confirmando === hijoId}
                  >
                    {confirmando === hijoId ? (
                      <ActivityIndicator color={color.white} size="small" />
                    ) : (
                      <Text style={styles.botonMovimientoTexto}>Confirmar {esSalida ? 'salida' : 'entrada'}</Text>
                    )}
                  </TouchableOpacity>
                </View>
              )}
            </View>
          );
        })}

        <TouchableOpacity style={styles.botonOtro} onPress={volverAEscanear}>
          <Text style={styles.botonOtroTexto}>Escanear otro código</Text>
        </TouchableOpacity>
      </ScrollView>
    );
  }

  return (
    <View style={styles.pantalla}>
      <CameraView
        style={StyleSheet.absoluteFillObject}
        facing="back"
        barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
        onBarcodeScanned={canjear}
      />
      <View style={styles.overlay}>
        <View style={styles.recuadro} />
        <Text style={styles.instruccion}>
          {verificando ? 'Verificando...' : 'Apunta al QR de la tablet de recepción'}
        </Text>
        {!!errorCodigo && (
          <View style={styles.errorCaja}>
            <Text style={styles.errorTexto}>{errorCodigo}</Text>
          </View>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate900 },
  centrado: { alignItems: 'center', justifyContent: 'center', padding: 32, gap: 12 },
  permisoTitulo: { fontSize: 15, fontWeight: '900', color: color.slate900, textTransform: 'uppercase', marginTop: 4 },
  permisoTexto: { fontSize: 12, color: color.slate500, fontWeight: '600', textAlign: 'center', lineHeight: 18 },
  contenido: { padding: 20, gap: 12, paddingBottom: 40, backgroundColor: color.slate50, flexGrow: 1 },
  bannerVerificado: { flexDirection: 'row', alignItems: 'center', gap: 10, backgroundColor: color.emerald50, borderWidth: 1, borderColor: color.emerald200, borderRadius: radius.md, padding: 16 },
  bannerTexto: { flex: 1, fontSize: 12, fontWeight: '800', color: color.emerald700 },
  tarjetaHijo: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.md, padding: 18, gap: 4, ...sombra.sm },
  filaHijo: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: 10 },
  filaHijoTexto: { flex: 1 },
  nombreHijo: { fontSize: 15, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  estadoHijo: { fontSize: 9, color: color.slate400, fontWeight: '800', textTransform: 'uppercase', marginTop: 2 },
  listoFila: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  listoTexto: { fontSize: 10, fontWeight: '900', color: color.emerald600, textTransform: 'uppercase' },
  botonMovimiento: { paddingHorizontal: 18, paddingVertical: 10, borderRadius: radius.sm },
  botonEntrada: { backgroundColor: color.emerald500 },
  botonSalida: { backgroundColor: color.amber500 },
  botonCerrar: { backgroundColor: color.slate100 },
  botonMovimientoTexto: { color: color.white, fontWeight: '900', fontSize: 11, textTransform: 'uppercase' },
  botonCerrarTexto: { color: color.slate500 },
  formHijo: { marginTop: 14, paddingTop: 14, borderTopWidth: 1, borderTopColor: color.slate100, gap: 10 },
  filaToggles: { flexDirection: 'row', gap: 8 },
  toggle: { flex: 1, paddingVertical: 12, borderRadius: radius.sm, borderWidth: 1, borderColor: color.slate200, alignItems: 'center' },
  toggleAseadoActivo: { backgroundColor: color.emerald500, borderColor: color.emerald500 },
  toggleGolpeActivo: { backgroundColor: color.rose500, borderColor: color.rose500 },
  toggleTexto: { fontSize: 10, fontWeight: '900', color: color.slate400, textTransform: 'uppercase' },
  toggleTextoActivo: { color: color.white },
  inputObs: {
    backgroundColor: color.slate50, borderWidth: 1, borderColor: color.slate200, borderRadius: radius.sm,
    paddingHorizontal: 14, paddingVertical: 12, fontSize: 12, fontWeight: '600', color: color.slate900, minHeight: 50, textAlignVertical: 'top',
  },
  botonConfirmar: { paddingVertical: 14, borderRadius: radius.sm, alignItems: 'center' },
  botonOtro: { paddingVertical: 16, alignItems: 'center' },
  botonOtroTexto: { color: color.slate400, fontWeight: '900', fontSize: 11, textTransform: 'uppercase', letterSpacing: 0.5 },
  overlay: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, gap: 20 },
  recuadro: { width: 240, height: 240, borderWidth: 3, borderColor: color.white, borderRadius: radius.md, opacity: 0.9 },
  instruccion: { color: color.white, fontWeight: '900', fontSize: 11, textTransform: 'uppercase', textAlign: 'center', letterSpacing: 0.5 },
  errorCaja: { backgroundColor: 'rgba(244,63,94,0.15)', borderWidth: 1, borderColor: color.rose500, borderRadius: radius.sm, padding: 12, maxWidth: 280 },
  errorTexto: { color: color.white, fontWeight: '700', fontSize: 11, textAlign: 'center' },
  boton: { backgroundColor: color.brand600, borderRadius: radius.sm, paddingVertical: 14, paddingHorizontal: 24 },
  botonTexto: { color: color.white, fontWeight: '900', fontSize: 12, textTransform: 'uppercase' },
});
