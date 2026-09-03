import React, { useEffect, useState } from 'react';
import {
  View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, ActivityIndicator, Alert, RefreshControl,
  KeyboardAvoidingView, Platform,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { color, radius, sombra } from '../theme';

// Equivalente RN de EncuestasPadre.jsx en la web: solo se puede responder
// una vez (el backend rechaza un segundo envío con 400, ver
// handleResponderEncuesta en encuestas.go), y ya respondida se deja el
// mismo formulario pero deshabilitado y con lo que el papá puso
// (respuesta_padre) en vez de ocultarlo -- así puede ver qué contestó sin
// poder cambiarlo.
export default function EncuestasScreen() {
  const [encuestas, setEncuestas] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [refrescando, setRefrescando] = useState(false);
  const [respuestas, setRespuestas] = useState({}); // { [encuestaId]: { [preguntaId]: valor } }
  const [enviando, setEnviando] = useState(null);

  const cargar = async () => {
    try {
      const res = await api.get('/padre/encuestas');
      setEncuestas(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar encuestas:', err);
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

  const setRespuesta = (encuestaId, preguntaId, valor) => {
    setRespuestas((prev) => ({
      ...prev,
      [encuestaId]: { ...(prev[encuestaId] || {}), [preguntaId]: valor },
    }));
  };

  const enviar = async (encuesta) => {
    const propias = respuestas[encuesta.id] || {};
    const cuerpo = encuesta.preguntas
      .map((p) => ({ pregunta_id: p.id, respuesta: (propias[p.id] || '').trim() }))
      .filter((r) => r.respuesta);

    if (cuerpo.length === 0) {
      Alert.alert('Responde al menos una pregunta');
      return;
    }

    setEnviando(encuesta.id);
    try {
      await api.post(`/padre/encuestas/${encuesta.id}/respuestas`, { respuestas: cuerpo });
      await cargar();
    } catch (err) {
      console.error('Error al enviar respuestas:', err);
      Alert.alert('No se pudo enviar', err.response?.data?.error || 'Inténtalo de nuevo');
    } finally {
      setEnviando(null);
    }
  };

  if (cargando) {
    return (
      <View style={styles.centro}><ActivityIndicator color={color.brand600} /></View>
    );
  }

  if (encuestas.length === 0) {
    return (
      <ScrollView
        contentContainerStyle={styles.centro}
        refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
      >
        <Ionicons name="clipboard-outline" size={36} color={color.slate200} />
        <Text style={styles.vacioTexto}>No hay encuestas activas por ahora</Text>
      </ScrollView>
    );
  }

  return (
    <KeyboardAvoidingView
      style={styles.pantalla}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      keyboardVerticalOffset={Platform.OS === 'ios' ? 90 : 0}
    >
    <ScrollView
      contentContainerStyle={styles.contenido}
      refreshControl={<RefreshControl refreshing={refrescando} onRefresh={refrescar} tintColor={color.brand600} />}
    >
      {encuestas.map((enc) => (
        <View key={enc.id} style={styles.tarjeta}>
          <Text style={styles.titulo}>{enc.titulo}</Text>
          {!!enc.descripcion && <Text style={styles.descripcion}>{enc.descripcion}</Text>}

          {enc.ya_respondida && (
            <View style={styles.avisoRespondida}>
              <Ionicons name="checkmark-circle" size={16} color={color.emerald600} />
              <Text style={styles.avisoRespondidaTexto}>Ya respondiste, ¡gracias!</Text>
            </View>
          )}

          <View style={styles.preguntas}>
            {enc.preguntas.map((p) => (
              <View key={p.id}>
                <Text style={styles.pregunta}>{p.texto}</Text>
                {p.tipo === 'opcion_multiple' ? (
                  <View style={styles.opciones}>
                    {(p.opciones || []).map((opcion) => {
                      const elegida = enc.ya_respondida
                        ? p.respuesta_padre === opcion
                        : (respuestas[enc.id] || {})[p.id] === opcion;
                      return (
                        <TouchableOpacity
                          key={opcion}
                          style={[styles.opcion, enc.ya_respondida && styles.opcionDeshabilitada]}
                          activeOpacity={enc.ya_respondida ? 1 : 0.7}
                          onPress={() => !enc.ya_respondida && setRespuesta(enc.id, p.id, opcion)}
                        >
                          <View style={[styles.radio, elegida && styles.radioElegido]}>
                            {elegida && <View style={styles.radioPunto} />}
                          </View>
                          <Text style={styles.opcionTexto}>{opcion}</Text>
                        </TouchableOpacity>
                      );
                    })}
                  </View>
                ) : (
                  <TextInput
                    style={[styles.textarea, enc.ya_respondida && styles.textareaDeshabilitada]}
                    multiline
                    numberOfLines={2}
                    editable={!enc.ya_respondida}
                    placeholder="Tu respuesta..."
                    placeholderTextColor={color.slate400}
                    value={enc.ya_respondida ? (p.respuesta_padre || '') : ((respuestas[enc.id] || {})[p.id] || '')}
                    onChangeText={(texto) => setRespuesta(enc.id, p.id, texto)}
                  />
                )}
              </View>
            ))}
          </View>

          {!enc.ya_respondida && (
            <TouchableOpacity
              style={[styles.botonEnviar, enviando === enc.id && styles.botonEnviarDeshabilitado]}
              onPress={() => enviar(enc)}
              disabled={enviando === enc.id}
            >
              {enviando === enc.id
                ? <ActivityIndicator color={color.white} size="small" />
                : <Ionicons name="send" size={16} color={color.white} />}
              <Text style={styles.botonEnviarTexto}>Enviar respuestas</Text>
            </TouchableOpacity>
          )}
        </View>
      ))}
    </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 16, paddingBottom: 40 },
  centro: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32, gap: 12 },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 12, textTransform: 'uppercase', textAlign: 'center' },
  tarjeta: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 20, gap: 16, ...sombra.sm },
  titulo: { fontSize: 14, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  descripcion: { fontSize: 12, color: color.slate500, fontWeight: '600', marginTop: -8 },
  avisoRespondida: {
    flexDirection: 'row', alignItems: 'center', gap: 8, backgroundColor: color.emerald50,
    borderWidth: 1, borderColor: color.emerald100, borderRadius: radius.sm, padding: 12,
  },
  avisoRespondidaTexto: { color: color.emerald600, fontSize: 11, fontWeight: '900', textTransform: 'uppercase' },
  preguntas: { gap: 16 },
  pregunta: { fontSize: 12, fontWeight: '700', color: color.slate700, marginBottom: 8 },
  opciones: { gap: 6 },
  opcion: {
    flexDirection: 'row', alignItems: 'center', gap: 10, backgroundColor: color.slate50,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.sm, padding: 12,
  },
  opcionDeshabilitada: { backgroundColor: color.slate100, opacity: 0.7 },
  radio: { width: 18, height: 18, borderRadius: 9, borderWidth: 2, borderColor: color.slate300, alignItems: 'center', justifyContent: 'center' },
  radioElegido: { borderColor: color.brand600 },
  radioPunto: { width: 9, height: 9, borderRadius: 5, backgroundColor: color.brand600 },
  opcionTexto: { fontSize: 12, fontWeight: '600', color: color.slate600 },
  textarea: {
    backgroundColor: color.slate50, borderWidth: 1, borderColor: color.slate200, borderRadius: radius.sm,
    padding: 12, fontSize: 12, fontWeight: '600', color: color.slate900, minHeight: 60, textAlignVertical: 'top',
  },
  textareaDeshabilitada: { backgroundColor: color.slate100, color: color.slate400 },
  botonEnviar: {
    flexDirection: 'row', alignItems: 'center', justifyContent: 'center', gap: 8,
    backgroundColor: color.brand600, borderRadius: radius.sm, paddingVertical: 14,
  },
  botonEnviarDeshabilitado: { opacity: 0.5 },
  botonEnviarTexto: { color: color.white, fontWeight: '900', fontSize: 12, textTransform: 'uppercase', letterSpacing: 0.5 },
});
