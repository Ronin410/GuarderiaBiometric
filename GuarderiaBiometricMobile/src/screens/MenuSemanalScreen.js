import React, { useCallback, useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { hoyLocal, lunesDeLaSemana, diasHabilesDeLaSemana, sumarDias } from '../utils/fecha';
import { color, radius, sombra } from '../theme';

const NOMBRES_DIA = ['Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes'];

// Equivalente RN de MenuPadre.jsx en la web: el menú de la semana completa
// (el inicio solo adelanta el de hoy), navegable atrás/adelante, contra el
// mismo endpoint /padre/menu-semanal.
export default function MenuSemanalScreen() {
  const [lunes, setLunes] = useState(lunesDeLaSemana(hoyLocal()));
  const [porFecha, setPorFecha] = useState({});
  const [cargando, setCargando] = useState(true);

  const hoy = hoyLocal();
  const dias = diasHabilesDeLaSemana(lunes);

  const cargarSemana = useCallback(async () => {
    setCargando(true);
    try {
      const diasSemana = diasHabilesDeLaSemana(lunes);
      const res = await api.get('/padre/menu-semanal', { params: { inicio: diasSemana[0], fin: diasSemana[4] } });
      const mapa = {};
      (Array.isArray(res.data) ? res.data : []).forEach((d) => {
        mapa[d.fecha] = { desayuno: d.desayuno || '', comida: d.comida || '', merienda: d.merienda || '' };
      });
      setPorFecha(mapa);
    } catch (err) {
      console.error('Error al cargar el menú semanal:', err);
    } finally {
      setCargando(false);
    }
  }, [lunes]);

  useEffect(() => { cargarSemana(); }, [cargarSemana]);

  const cambiarSemana = (delta) => setLunes((l) => sumarDias(l, delta * 7));

  const formatoLargo = (fechaISO) => {
    const d = new Date(`${fechaISO}T00:00:00`);
    return d.toLocaleDateString('es-MX', { day: 'numeric', month: 'short', timeZone: 'UTC' });
  };

  return (
    <View style={styles.pantalla}>
      <View style={styles.selectorSemana}>
        <TouchableOpacity style={styles.flecha} onPress={() => cambiarSemana(-1)}>
          <Ionicons name="chevron-back" size={18} color={color.slate500} />
        </TouchableOpacity>
        <Text style={styles.textoSemana}>{formatoLargo(dias[0])} al {formatoLargo(dias[4])}</Text>
        <TouchableOpacity style={styles.flecha} onPress={() => cambiarSemana(1)}>
          <Ionicons name="chevron-forward" size={18} color={color.slate500} />
        </TouchableOpacity>
      </View>

      {cargando ? (
        <ActivityIndicator color={color.brand600} style={{ marginTop: 40 }} />
      ) : (
        <ScrollView contentContainerStyle={styles.contenido}>
          {dias.map((fecha, i) => {
            const dia = porFecha[fecha];
            const esHoy = fecha === hoy;
            const sinNada = !dia || (!dia.desayuno && !dia.comida && !dia.merienda);
            return (
              <View key={fecha} style={[styles.tarjetaDia, esHoy && styles.tarjetaDiaHoy]}>
                <View style={styles.encabezadoDia}>
                  <View>
                    <Text style={styles.nombreDia}>{NOMBRES_DIA[i]}</Text>
                    <Text style={styles.fechaDia}>{formatoLargo(fecha)}</Text>
                  </View>
                  {esHoy && (
                    <View style={styles.insigniaHoy}><Text style={styles.insigniaHoyTexto}>Hoy</Text></View>
                  )}
                </View>

                {sinNada ? (
                  <Text style={styles.sinMenu}>Sin menú capturado</Text>
                ) : (
                  <View style={styles.comidas}>
                    {!!dia.desayuno && <FilaComida icono="cafe" label="Desayuno" valor={dia.desayuno} />}
                    {!!dia.comida && <FilaComida icono="restaurant" label="Comida" valor={dia.comida} />}
                    {!!dia.merienda && <FilaComida icono="ice-cream" label="Merienda" valor={dia.merienda} />}
                  </View>
                )}
              </View>
            );
          })}
        </ScrollView>
      )}
    </View>
  );
}

const FilaComida = ({ icono, label, valor }) => (
  <View style={styles.filaComida}>
    <Ionicons name={icono} size={13} color={color.brand500} style={{ marginTop: 1 }} />
    <Text style={styles.textoComida}><Text style={styles.labelComida}>{label}: </Text>{valor}</Text>
  </View>
);

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  selectorSemana: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: color.white,
    borderBottomWidth: 1, borderBottomColor: color.slate100, paddingHorizontal: 16, paddingVertical: 12,
  },
  flecha: { padding: 8, backgroundColor: color.slate50, borderRadius: radius.sm },
  textoSemana: { fontSize: 12, fontWeight: '900', color: color.slate700, textTransform: 'uppercase' },
  contenido: { padding: 20, gap: 12, paddingBottom: 40 },
  tarjetaDia: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 18, ...sombra.sm },
  tarjetaDiaHoy: { borderColor: color.brand300, borderWidth: 2 },
  encabezadoDia: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 },
  nombreDia: { fontSize: 13, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  fechaDia: { fontSize: 10, color: color.brand500, fontWeight: '700', textTransform: 'uppercase', marginTop: 1 },
  insigniaHoy: { backgroundColor: color.brand600, borderRadius: radius.full, paddingHorizontal: 10, paddingVertical: 4 },
  insigniaHoyTexto: { color: color.white, fontSize: 9, fontWeight: '900', textTransform: 'uppercase', letterSpacing: 0.5 },
  sinMenu: { fontSize: 11, color: color.slate400, fontWeight: '700', textTransform: 'uppercase' },
  comidas: { gap: 6 },
  filaComida: { flexDirection: 'row', alignItems: 'flex-start', gap: 6 },
  textoComida: { flex: 1, fontSize: 12, fontWeight: '700', color: color.slate700 },
  labelComida: { color: color.brand500 },
});
