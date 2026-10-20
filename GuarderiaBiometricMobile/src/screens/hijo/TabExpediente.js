import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, Linking, TouchableOpacity } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../../api/client';
import { color, radius, sombra } from '../../theme';

// Equivalente RN de la pestaña "Expediente" de VistaPadreDetalle.jsx: los
// datos básicos del niño (que ya llegan en el listado de /padre/0/hijos,
// ver DashboardScreen.js) más los documentos de inscripción, de solo
// lectura -- quien sube/reemplaza documentos sigue siendo el staff.
export default function TabExpediente({ hijoId, expediente }) {
  const [documentos, setDocumentos] = useState([]);
  const [cargando, setCargando] = useState(true);

  useEffect(() => {
    api.get(`/padre/hijos/${hijoId}/documentos`)
      .then((res) => setDocumentos(Array.isArray(res.data) ? res.data : []))
      .catch((err) => {
        console.error('Error al obtener los documentos', err);
        setDocumentos([]);
      })
      .finally(() => setCargando(false));
  }, [hijoId]);

  return (
    <ScrollView style={styles.pantalla} contentContainerStyle={styles.contenido}>
      <View style={styles.tarjeta}>
        <View style={styles.encabezado}>
          <View style={styles.iconoRedondo}><Ionicons name="card" size={18} color={color.brand600} /></View>
          <Text style={styles.tituloTarjeta}>Expediente</Text>
        </View>

        <Dato icono="gift" label="Fecha de nacimiento" valor={expediente?.fechaNacimiento} />
        <Dato icono="location" label="Dirección" valor={expediente?.direccion} />
        <Dato
          icono="call"
          label="Contacto de emergencia"
          valor={expediente?.contactoEmergenciaNombre
            ? `${expediente.contactoEmergenciaNombre} · ${expediente.contactoEmergenciaTelefono || 's/n'}`
            : null}
        />
      </View>

      <View style={styles.tarjeta}>
        <View style={styles.encabezado}>
          <View style={styles.iconoRedondo}><Ionicons name="document-text" size={18} color={color.brand600} /></View>
          <Text style={styles.tituloTarjeta}>Documentos entregados</Text>
        </View>

        {cargando ? (
          <ActivityIndicator color={color.brand600} style={{ marginVertical: 12 }} />
        ) : documentos.length === 0 ? (
          <Text style={styles.vacioTexto}>La guardería todavía no configuró qué documentos pedir</Text>
        ) : (
          <View style={{ gap: 8 }}>
            {documentos.map((d) => {
              const entregado = !!d.nombre_archivo;
              return (
                <View key={d.tipo} style={[styles.filaDocumento, entregado ? styles.filaDocumentoOk : styles.filaDocumentoFalta]}>
                  <Ionicons
                    name={entregado ? 'checkmark-circle' : 'close-circle'}
                    size={18}
                    color={entregado ? color.emerald500 : color.amber500}
                  />
                  <View style={{ flex: 1 }}>
                    <Text style={styles.nombreDocumento}>{d.nombre}</Text>
                    <Text style={[styles.estadoDocumento, { color: entregado ? color.emerald600 : color.amber600 }]}>
                      {entregado ? 'Entregado' : 'Falta entregar'}
                    </Text>
                  </View>
                  {entregado && !!d.url && (
                    <TouchableOpacity onPress={() => Linking.openURL(d.url)} style={styles.verDocumento}>
                      <Ionicons name="open-outline" size={14} color={color.brand600} />
                      <Text style={styles.verDocumentoTexto}>Ver</Text>
                    </TouchableOpacity>
                  )}
                </View>
              );
            })}
          </View>
        )}
      </View>
    </ScrollView>
  );
}

const Dato = ({ icono, label, valor }) => (
  <View style={styles.filaDato}>
    <View style={styles.iconoDato}><Ionicons name={icono} size={18} color={color.slate400} /></View>
    <View style={{ flex: 1 }}>
      <Text style={styles.labelDato}>{label}</Text>
      <Text style={styles.valorDato}>{valor || 'No registrada'}</Text>
    </View>
  </View>
);

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 16, paddingBottom: 40 },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, gap: 16, ...sombra.sm },
  encabezado: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  iconoRedondo: { backgroundColor: color.brand100, padding: 8, borderRadius: radius.sm },
  tituloTarjeta: { fontSize: 11, fontWeight: '900', color: color.slate900, textTransform: 'uppercase', letterSpacing: 0.5 },
  filaDato: { flexDirection: 'row', alignItems: 'center', gap: 14 },
  iconoDato: { backgroundColor: color.slate50, padding: 10, borderRadius: radius.sm },
  labelDato: { fontSize: 9, fontWeight: '900', color: color.slate400, textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 2 },
  valorDato: { fontSize: 13, fontWeight: '700', color: color.slate700 },
  vacioTexto: { fontSize: 10, fontWeight: '800', color: color.slate400, textTransform: 'uppercase', textAlign: 'center', paddingVertical: 12 },
  filaDocumento: { flexDirection: 'row', alignItems: 'center', gap: 10, borderWidth: 1, borderRadius: radius.md, paddingHorizontal: 14, paddingVertical: 12 },
  filaDocumentoOk: { backgroundColor: '#ecfdf580', borderColor: color.emerald100 },
  filaDocumentoFalta: { backgroundColor: '#fffbeb80', borderColor: '#fde68a' },
  nombreDocumento: { fontSize: 11, fontWeight: '900', color: color.slate700, textTransform: 'uppercase' },
  estadoDocumento: { fontSize: 9, fontWeight: '900', textTransform: 'uppercase', marginTop: 1 },
  verDocumento: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  verDocumentoTexto: { fontSize: 10, fontWeight: '900', color: color.brand600, textTransform: 'uppercase' },
});
