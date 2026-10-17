import React, { useEffect, useState } from 'react';
import {
  View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, ActivityIndicator,
  Alert, Linking, Modal, Share,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../../api/client';
import { hoyLocal } from '../../utils/fecha';
import { color, radius, sombra } from '../../theme';

// Equivalente RN de la pestaña "Pagos" de VistaPadreDetalle.jsx: historial
// de pagos, pago en línea con tarjeta (cuando la guardería lo tenga
// configurado) y el recibo de cada pago (en un modal, con "Compartir" en
// vez del botón "Imprimir" de la web -- ver ReciboPago.jsx).
export default function TabPagos({ hijoId }) {
  const [historial, setHistorial] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [pagosEnLineaHabilitado, setPagosEnLineaHabilitado] = useState(false);
  const [estadoPago, setEstadoPago] = useState(null);
  const [montoAPagar, setMontoAPagar] = useState('');
  const [iniciandoPago, setIniciandoPago] = useState(false);
  const [reciboId, setReciboId] = useState(null);

  useEffect(() => {
    const cargar = async () => {
      try {
        const res = await api.get('/padre/mis-pagos/historial', { params: { hijo_id: hijoId } });
        setHistorial(Array.isArray(res.data) ? res.data : []);
      } catch (err) {
        console.error('Error al obtener el historial de pagos', err);
        setHistorial([]);
      } finally {
        setCargando(false);
      }
    };
    cargar();

    api.get('/pagos-online/config')
      .then((res) => setPagosEnLineaHabilitado(!!res.data?.habilitado))
      .catch(() => setPagosEnLineaHabilitado(false));

    api.get('/padre/mis-pagos')
      .then((res) => {
        const propio = Array.isArray(res.data) ? res.data.find((e) => String(e.hijo_id) === String(hijoId)) : null;
        setEstadoPago(propio || null);
        const saldo = propio ? propio.colegiatura_mensual - propio.total_pagado : 0;
        setMontoAPagar(saldo > 0 ? String(saldo.toFixed(2)) : '');
      })
      .catch(() => setEstadoPago(null));
  }, [hijoId]);

  const pagarEnLinea = async () => {
    const monto = Number(montoAPagar);
    if (!monto || monto <= 0) {
      Alert.alert('Escribe cuánto quieres pagar');
      return;
    }
    setIniciandoPago(true);
    try {
      const periodoActual = hoyLocal().slice(0, 7); // YYYY-MM
      const res = await api.post('/padre/pagos-online/checkout', { hijo_id: String(hijoId), periodo: periodoActual, monto });
      // El checkout de Stripe es una página web -- se abre en el navegador
      // del celular, no dentro de la app, igual que window.location.href
      // en la web.
      await Linking.openURL(res.data.url);
    } catch (err) {
      console.error('Error al iniciar el pago en línea', err);
      Alert.alert('No se pudo iniciar el pago', err.response?.data?.error || 'Inténtalo de nuevo');
    } finally {
      setIniciandoPago(false);
    }
  };

  const saldoPendiente = estadoPago ? estadoPago.colegiatura_mensual - estadoPago.total_pagado : 0;

  return (
    <ScrollView style={styles.pantalla} contentContainerStyle={styles.contenido}>
      {pagosEnLineaHabilitado && estadoPago && saldoPendiente > 0 && (
        <View style={styles.tarjeta}>
          <View style={styles.encabezado}>
            <View style={styles.iconoRedondo}><Ionicons name="card" size={18} color={color.brand600} /></View>
            <Text style={styles.tituloTarjeta}>Pagar colegiatura con tarjeta</Text>
          </View>
          <View style={styles.filaSaldo}>
            <Text style={styles.labelSaldo}>Saldo pendiente de este mes</Text>
            <Text style={styles.valorSaldo}>${Number(saldoPendiente).toLocaleString('es-MX', { minimumFractionDigits: 2 })}</Text>
          </View>
          <View>
            <Text style={styles.labelMonto}>¿Cuánto quieres pagar?</Text>
            <View style={styles.campoMonto}>
              <Text style={styles.simboloMonto}>$</Text>
              <TextInput
                style={styles.inputMonto}
                keyboardType="decimal-pad"
                value={montoAPagar}
                onChangeText={setMontoAPagar}
              />
            </View>
          </View>
          <TouchableOpacity style={[styles.boton, iniciandoPago && styles.botonDeshabilitado]} onPress={pagarEnLinea} disabled={iniciandoPago}>
            {iniciandoPago ? <ActivityIndicator color={color.white} size="small" /> : <Ionicons name="card" size={16} color={color.white} />}
            <Text style={styles.botonTexto}>Pagar ${montoAPagar || '0'} con tarjeta</Text>
          </TouchableOpacity>
        </View>
      )}

      {cargando ? (
        <ActivityIndicator color={color.brand600} />
      ) : historial.length === 0 ? (
        <View style={styles.vacio}>
          <Ionicons name="wallet-outline" size={32} color={color.slate200} />
          <Text style={styles.vacioTexto}>Sin pagos registrados todavía.</Text>
        </View>
      ) : (
        historial.map((p) => (
          <View key={p.id} style={styles.filaPago}>
            <View style={{ flex: 1 }}>
              <Text style={styles.montoPago}>
                ${Number(p.monto).toLocaleString('es-MX', { minimumFractionDigits: 2 })}
                <Text style={styles.conceptoPago}> · {p.concepto}</Text>
              </Text>
              <Text style={styles.detallePago}>{p.periodo} · {p.fecha_pago} · {p.metodo_pago}</Text>
            </View>
            <TouchableOpacity onPress={() => setReciboId(p.id)} style={styles.botonRecibo}>
              <Ionicons name="receipt-outline" size={18} color={color.slate400} />
            </TouchableOpacity>
          </View>
        ))
      )}

      <Modal visible={!!reciboId} animationType="slide" onRequestClose={() => setReciboId(null)}>
        <ModalRecibo pagoId={reciboId} onCerrar={() => setReciboId(null)} />
      </Modal>
    </ScrollView>
  );
}

// ModalRecibo -- equivalente RN de components/ReciboPago.jsx. En vez de
// "Imprimir / Guardar PDF" (window.print(), que no existe en RN) usa el
// Share nativo del celular con el resumen en texto -- generar un PDF de
// verdad (expo-print) queda para cuando haga falta de verdad.
const ModalRecibo = ({ pagoId, onCerrar }) => {
  const [recibo, setRecibo] = useState(null);
  const [cargando, setCargando] = useState(true);

  useEffect(() => {
    if (!pagoId) return;
    setCargando(true);
    api.get(`/padre/pagos/${pagoId}/recibo`)
      .then((res) => setRecibo(res.data))
      .catch((err) => {
        console.error('Error al cargar el recibo:', err);
        setRecibo(null);
      })
      .finally(() => setCargando(false));
  }, [pagoId]);

  const compartir = () => {
    if (!recibo) return;
    Share.share({
      message: `Recibo ${recibo.folio} -- ${recibo.guarderia_nombre}\n` +
        `Alumno: ${recibo.nino_nombre}\nConcepto: ${recibo.concepto}\nPeriodo: ${recibo.periodo}\n` +
        `Fecha de pago: ${recibo.fecha_pago}\nMétodo: ${recibo.metodo_pago}\n` +
        `Total pagado: $${Number(recibo.monto).toLocaleString('es-MX', { minimumFractionDigits: 2 })}`,
    });
  };

  return (
    <View style={styles.pantallaModal}>
      <View style={styles.barraModal}>
        <TouchableOpacity onPress={onCerrar} style={styles.botonCerrarModal}>
          <Ionicons name="close" size={22} color={color.slate500} />
        </TouchableOpacity>
        <Text style={styles.tituloModal}>Recibo</Text>
        <TouchableOpacity onPress={compartir} style={styles.botonCerrarModal}>
          <Ionicons name="share-outline" size={20} color={color.brand600} />
        </TouchableOpacity>
      </View>

      {cargando ? (
        <ActivityIndicator color={color.brand600} style={{ marginTop: 60 }} />
      ) : !recibo ? (
        <Text style={styles.vacioTexto}>No se pudo cargar el recibo</Text>
      ) : (
        <ScrollView contentContainerStyle={styles.contenidoRecibo}>
          <View style={styles.encabezadoRecibo}>
            <View style={styles.iconoGuarderia}><Ionicons name="shield-checkmark" size={20} color={color.white} /></View>
            <View style={{ flex: 1 }}>
              <Text style={styles.nombreGuarderia}>{recibo.guarderia_nombre}</Text>
              {!!recibo.guarderia_direccion && <Text style={styles.direccionGuarderia}>{recibo.guarderia_direccion}</Text>}
            </View>
            <View>
              <Text style={styles.labelFolio}>Recibo</Text>
              <Text style={styles.folio}>{recibo.folio}</Text>
            </View>
          </View>

          <View style={styles.grillaRecibo}>
            <CampoRecibo label="Alumno" valor={recibo.nino_nombre} />
            <CampoRecibo label="Fecha de pago" valor={recibo.fecha_pago} />
            <CampoRecibo label="Concepto" valor={recibo.concepto} />
            <CampoRecibo label="Periodo" valor={recibo.periodo} />
            <CampoRecibo label="Método de pago" valor={recibo.metodo_pago} />
            {!!recibo.observaciones && <CampoRecibo label="Observaciones" valor={recibo.observaciones} />}
          </View>

          <View style={styles.totalRecibo}>
            <Text style={styles.labelTotal}>Total pagado</Text>
            <Text style={styles.valorTotal}>${Number(recibo.monto).toLocaleString('es-MX', { minimumFractionDigits: 2 })}</Text>
          </View>

          <Text style={styles.pieRecibo}>Generado por Pasitos</Text>
        </ScrollView>
      )}
    </View>
  );
};

const CampoRecibo = ({ label, valor }) => (
  <View style={styles.campoRecibo}>
    <Text style={styles.labelCampoRecibo}>{label}</Text>
    <Text style={styles.valorCampoRecibo}>{valor}</Text>
  </View>
);

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 12, paddingBottom: 40 },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, gap: 14, ...sombra.sm },
  encabezado: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  iconoRedondo: { backgroundColor: color.brand100, padding: 8, borderRadius: radius.sm },
  tituloTarjeta: { fontSize: 11, fontWeight: '900', color: color.slate900, textTransform: 'uppercase', letterSpacing: 0.5 },
  filaSaldo: { flexDirection: 'row', justifyContent: 'space-between' },
  labelSaldo: { fontSize: 12, fontWeight: '700', color: color.slate500 },
  valorSaldo: { fontSize: 12, fontWeight: '900', color: color.slate800 },
  labelMonto: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 4 },
  campoMonto: { flexDirection: 'row', alignItems: 'center', backgroundColor: color.slate50, borderRadius: radius.sm, paddingHorizontal: 14 },
  simboloMonto: { fontSize: 18, fontWeight: '900', color: color.slate400 },
  inputMonto: { flex: 1, paddingVertical: 12, paddingLeft: 6, fontSize: 18, fontWeight: '900', color: color.slate800 },
  boton: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'center', gap: 8,
    backgroundColor: color.brand600, borderRadius: radius.sm, paddingVertical: 14,
  },
  botonDeshabilitado: { opacity: 0.5 },
  botonTexto: { color: color.white, fontWeight: '900', fontSize: 12, textTransform: 'uppercase' },
  vacio: { backgroundColor: color.white, borderWidth: 2, borderStyle: 'dashed', borderColor: color.slate200, borderRadius: radius.lg, padding: 32, alignItems: 'center', gap: 12 },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 11, textTransform: 'uppercase', textAlign: 'center' },
  filaPago: {
    flexDirection: 'row', alignItems: 'center', gap: 10, backgroundColor: color.white,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.md, padding: 16,
  },
  montoPago: { fontSize: 14, fontWeight: '900', color: color.slate800 },
  conceptoPago: { fontSize: 12, fontWeight: '700', color: color.slate400 },
  detallePago: { fontSize: 10, color: color.slate400, fontWeight: '700', marginTop: 3, textTransform: 'uppercase' },
  botonRecibo: { padding: 8 },
  // Modal de recibo
  pantallaModal: { flex: 1, backgroundColor: color.slate50 },
  barraModal: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16,
    paddingTop: 56, paddingBottom: 12, backgroundColor: color.white, borderBottomWidth: 1, borderBottomColor: color.slate100,
  },
  botonCerrarModal: { padding: 6, minWidth: 34 },
  tituloModal: { fontSize: 13, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  contenidoRecibo: { padding: 20, gap: 20 },
  encabezadoRecibo: { flexDirection: 'row', alignItems: 'center', gap: 12, borderBottomWidth: 1, borderStyle: 'dashed', borderBottomColor: color.slate200, paddingBottom: 16 },
  iconoGuarderia: { backgroundColor: color.brand600, padding: 10, borderRadius: radius.sm },
  nombreGuarderia: { fontSize: 13, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  direccionGuarderia: { fontSize: 10, color: color.slate400, fontWeight: '700', marginTop: 2 },
  labelFolio: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', textAlign: 'right' },
  folio: { fontSize: 13, fontWeight: '900', color: color.slate900, textAlign: 'right' },
  grillaRecibo: { flexDirection: 'row', flexWrap: 'wrap', gap: 16 },
  campoRecibo: { width: '45%' },
  labelCampoRecibo: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase' },
  valorCampoRecibo: { fontSize: 13, fontWeight: '700', color: color.slate800, marginTop: 2 },
  totalRecibo: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: color.brand50,
    borderWidth: 1, borderColor: color.brand100, borderRadius: radius.lg, padding: 20,
  },
  labelTotal: { fontSize: 10, fontWeight: '900', color: color.brand600, textTransform: 'uppercase', letterSpacing: 0.5 },
  valorTotal: { fontSize: 22, fontWeight: '900', color: color.brand700 },
  pieRecibo: { textAlign: 'center', fontSize: 9, color: color.slate300, fontWeight: '700', textTransform: 'uppercase', marginTop: 12 },
});
