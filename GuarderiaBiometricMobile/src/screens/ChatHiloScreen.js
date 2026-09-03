import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  View, Text, StyleSheet, FlatList, TextInput, TouchableOpacity, ActivityIndicator,
  Image, Modal, KeyboardAvoidingView, Platform, Linking, Alert,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import * as DocumentPicker from 'expo-document-picker';
import api from '../api/client';
import { fechaLocal, separadorFecha, formatoHora } from '../utils/fecha';
import { color, radius, sombra } from '../theme';

const INTERVALO_POLLING_MS = 5000;
const TAMANO_MAX_BYTES = 10 * 1024 * 1024;

// Equivalente RN de HiloChat dentro de ChatPadre.jsx en la web: el hilo de
// mensajes con UN contacto, con el mismo polling cada 5s (no hay
// WebSocket/push en tiempo real todavía, ver API_MOVIL.md) y los mismos
// separadores de fecha que ya usan los 4 chats de la web.
export default function ChatHiloScreen({ route, navigation }) {
  const { contactoId, nombre } = route.params;
  const [mensajes, setMensajes] = useState([]);
  const [cargando, setCargando] = useState(true);
  const [texto, setTexto] = useState('');
  const [archivo, setArchivo] = useState(null); // { uri, name, mimeType, size }
  const [enviando, setEnviando] = useState(false);
  const [fotoAmpliada, setFotoAmpliada] = useState(null);
  const listaRef = useRef(null);

  useEffect(() => {
    navigation.setOptions({ title: nombre });
  }, [navigation, nombre]);

  const cargarMensajes = useCallback(async (mostrarCargando) => {
    if (mostrarCargando) setCargando(true);
    try {
      const res = await api.get(`/padre/chat/${contactoId}`);
      setMensajes(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Error al cargar mensajes:', err);
    } finally {
      if (mostrarCargando) setCargando(false);
    }
  }, [contactoId]);

  useEffect(() => {
    cargarMensajes(true);
    const intervalo = setInterval(() => cargarMensajes(false), INTERVALO_POLLING_MS);
    return () => clearInterval(intervalo);
  }, [cargarMensajes]);

  const elegirArchivo = async () => {
    const res = await DocumentPicker.getDocumentAsync({
      type: ['image/*', 'application/pdf'],
      copyToCacheDirectory: true,
    });
    if (res.canceled || !res.assets?.[0]) return;
    const elegido = res.assets[0];
    if (elegido.size && elegido.size > TAMANO_MAX_BYTES) {
      Alert.alert('Archivo muy grande', 'El archivo no puede pesar más de 10 MB');
      return;
    }
    setArchivo(elegido);
  };

  const enviar = async () => {
    const contenido = texto.trim();
    if (!contenido && !archivo) return;
    setEnviando(true);
    try {
      const data = new FormData();
      data.append('contenido', contenido);
      if (archivo) {
        data.append('archivo', {
          uri: archivo.uri,
          name: archivo.name || 'archivo',
          type: archivo.mimeType || 'application/octet-stream',
        });
      }
      await api.post(`/padre/chat/${contactoId}`, data);
      setTexto('');
      setArchivo(null);
      await cargarMensajes(false);
      setTimeout(() => listaRef.current?.scrollToEnd({ animated: true }), 100);
    } catch (err) {
      console.error('Error al enviar mensaje:', err);
      Alert.alert('No se pudo enviar', err.response?.data?.error || 'Inténtalo de nuevo');
    } finally {
      setEnviando(false);
    }
  };

  const abrirAdjunto = (m) => {
    if (m.adjunto_tipo === 'imagen') {
      setFotoAmpliada(m.adjunto_url);
    } else if (m.adjunto_url) {
      Linking.openURL(m.adjunto_url);
    }
  };

  const renderMensaje = ({ item: m, index }) => {
    const anterior = mensajes[index - 1];
    const mostrarSeparador = index === 0 || fechaLocal(m.creado_en) !== fechaLocal(anterior.creado_en);
    return (
      <>
        {mostrarSeparador && (
          <View style={styles.separadorFila}>
            <Text style={styles.separadorTexto}>{separadorFecha(m.creado_en)}</Text>
          </View>
        )}
        <View style={[styles.filaBurbuja, m.es_mio ? styles.filaBurbujaMia : styles.filaBurbujaSuya]}>
          <View style={[styles.burbuja, m.es_mio ? styles.burbujaMia : styles.burbujaSuya]}>
            {!!m.adjunto_url && (
              <TouchableOpacity onPress={() => abrirAdjunto(m)} style={styles.adjunto}>
                {m.adjunto_tipo === 'imagen' ? (
                  <Image source={{ uri: m.adjunto_url }} style={styles.adjuntoImagen} />
                ) : (
                  <View style={[styles.adjuntoArchivo, m.es_mio && styles.adjuntoArchivoMio]}>
                    <Ionicons name="document-text" size={18} color={m.es_mio ? color.white : color.slate500} />
                    <Text numberOfLines={1} style={[styles.adjuntoNombre, m.es_mio && { color: color.white }]}>
                      {m.adjunto_nombre || 'Archivo'}
                    </Text>
                    <Ionicons name="download" size={14} color={m.es_mio ? color.white : color.slate500} />
                  </View>
                )}
              </TouchableOpacity>
            )}
            {!!m.contenido && (
              <Text style={m.es_mio ? styles.textoMio : styles.textoSuyo}>{m.contenido}</Text>
            )}
            <Text style={m.es_mio ? styles.horaMia : styles.horaSuya}>{formatoHora(m.creado_en)}</Text>
          </View>
        </View>
      </>
    );
  };

  return (
    <KeyboardAvoidingView
      style={styles.pantalla}
      // "mientras escribes no se puede ver lo que vas escribiendo, el
      // textbox se queda abajo" -- en Android, behavior: undefined no
      // ajusta nada y deja que el sistema operativo decida (que en varios
      // equipos no reacomoda la pantalla como se espera); 'height' sí
      // encoge el contenido para que la barra de envío suba junto con el
      // teclado, igual que 'padding' en iOS.
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      keyboardVerticalOffset={Platform.OS === 'ios' ? 90 : 0}
    >
      {cargando ? (
        <View style={styles.centro}><ActivityIndicator color={color.brand600} /></View>
      ) : mensajes.length === 0 ? (
        <View style={styles.centro}>
          <Text style={styles.vacioTexto}>Todavía no hay mensajes.{'\n'}Escribe el primero abajo.</Text>
        </View>
      ) : (
        <FlatList
          ref={listaRef}
          style={styles.listaContenedor}
          data={mensajes}
          keyExtractor={(m) => String(m.id)}
          renderItem={renderMensaje}
          contentContainerStyle={styles.lista}
          onContentSizeChange={() => listaRef.current?.scrollToEnd({ animated: false })}
        />
      )}

      {!!archivo && (
        <View style={styles.previewArchivo}>
          {archivo.mimeType?.startsWith('image/') ? (
            <Image source={{ uri: archivo.uri }} style={styles.previewImagen} />
          ) : (
            <View style={styles.previewIcono}><Ionicons name="document-text" size={20} color={color.slate400} /></View>
          )}
          <Text numberOfLines={1} style={styles.previewNombre}>{archivo.name}</Text>
          <TouchableOpacity onPress={() => setArchivo(null)}>
            <Ionicons name="close" size={18} color={color.slate400} />
          </TouchableOpacity>
        </View>
      )}

      <View style={styles.barraEnvio}>
        <TouchableOpacity style={styles.botonAdjuntar} onPress={elegirArchivo}>
          <Ionicons name="attach" size={22} color={color.slate400} />
        </TouchableOpacity>
        <TextInput
          style={styles.input}
          placeholder="Escribe un mensaje..."
          placeholderTextColor={color.slate400}
          value={texto}
          onChangeText={setTexto}
          multiline
        />
        <TouchableOpacity
          style={[styles.botonEnviar, (enviando || (!texto.trim() && !archivo)) && styles.botonEnviarDeshabilitado]}
          onPress={enviar}
          disabled={enviando || (!texto.trim() && !archivo)}
        >
          {enviando ? <ActivityIndicator color={color.white} size="small" /> : <Ionicons name="send" size={18} color={color.white} />}
        </TouchableOpacity>
      </View>

      <Modal visible={!!fotoAmpliada} transparent animationType="fade" onRequestClose={() => setFotoAmpliada(null)}>
        <TouchableOpacity style={styles.modalFondo} activeOpacity={1} onPress={() => setFotoAmpliada(null)}>
          {!!fotoAmpliada && <Image source={{ uri: fotoAmpliada }} style={styles.modalImagen} resizeMode="contain" />}
        </TouchableOpacity>
      </Modal>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  centro: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32 },
  vacioTexto: { color: color.slate400, fontWeight: '800', fontSize: 12, textTransform: 'uppercase', textAlign: 'center', lineHeight: 20 },
  // flex: 1 en el FlatList (no solo en su contentContainerStyle) -- sin
  // esto, dentro de KeyboardAvoidingView el hilo podía crecer más de lo
  // que cabe y empujar la barra de envío fuera de la pantalla cuando el
  // teclado está abierto, dejándola invisible en vez de solo encogerse.
  listaContenedor: { flex: 1 },
  lista: { padding: 16, gap: 8 },
  separadorFila: { alignItems: 'center', marginVertical: 6 },
  separadorTexto: {
    backgroundColor: color.white, borderWidth: 1, borderColor: color.slate200, color: color.slate400,
    fontSize: 9, fontWeight: '900', textTransform: 'uppercase', letterSpacing: 0.5,
    paddingHorizontal: 12, paddingVertical: 6, borderRadius: radius.full, overflow: 'hidden',
  },
  filaBurbuja: { flexDirection: 'row', marginBottom: 6 },
  filaBurbujaMia: { justifyContent: 'flex-end' },
  filaBurbujaSuya: { justifyContent: 'flex-start' },
  burbuja: { maxWidth: '80%', paddingHorizontal: 16, paddingVertical: 12, borderRadius: radius.lg },
  burbujaMia: { backgroundColor: color.brand600, borderBottomRightRadius: 6 },
  burbujaSuya: { backgroundColor: color.white, borderWidth: 1, borderColor: color.slate100, borderBottomLeftRadius: 6, ...sombra.sm },
  textoMio: { color: color.white, fontSize: 14, fontWeight: '600' },
  textoSuyo: { color: color.slate700, fontSize: 14, fontWeight: '600' },
  horaMia: { color: color.brand100, fontSize: 9, fontWeight: '800', textTransform: 'uppercase', marginTop: 4 },
  horaSuya: { color: color.slate400, fontSize: 9, fontWeight: '800', textTransform: 'uppercase', marginTop: 4 },
  adjunto: { marginBottom: 8 },
  adjuntoImagen: { width: 200, height: 160, borderRadius: radius.sm },
  adjuntoArchivo: { flexDirection: 'row', alignItems: 'center', gap: 8, backgroundColor: color.slate50, borderRadius: radius.sm, paddingHorizontal: 12, paddingVertical: 10 },
  adjuntoArchivoMio: { backgroundColor: 'rgba(255,255,255,0.15)' },
  adjuntoNombre: { flex: 1, fontSize: 12, fontWeight: '700', color: color.slate700 },
  previewArchivo: {
    flexDirection: 'row', alignItems: 'center', gap: 10, backgroundColor: color.white, marginHorizontal: 16,
    marginBottom: 4, padding: 10, borderRadius: radius.md, borderWidth: 1, borderColor: color.slate200, ...sombra.sm,
  },
  previewImagen: { width: 40, height: 40, borderRadius: 10 },
  previewIcono: { width: 40, height: 40, borderRadius: 10, backgroundColor: color.slate100, alignItems: 'center', justifyContent: 'center' },
  previewNombre: { flex: 1, fontSize: 12, fontWeight: '700', color: color.slate600 },
  barraEnvio: {
    flexDirection: 'row', alignItems: 'flex-end', gap: 8, backgroundColor: color.white, borderTopWidth: 1,
    borderTopColor: color.slate100, paddingHorizontal: 12, paddingVertical: 10,
  },
  botonAdjuntar: { padding: 8 },
  input: {
    flex: 1, backgroundColor: color.slate50, borderRadius: radius.md, paddingHorizontal: 14, paddingVertical: 10,
    fontSize: 14, color: color.slate900, maxHeight: 100,
  },
  botonEnviar: { backgroundColor: color.brand600, borderRadius: radius.sm, padding: 12 },
  botonEnviarDeshabilitado: { opacity: 0.5 },
  modalFondo: { flex: 1, backgroundColor: 'rgba(15,23,42,0.92)', alignItems: 'center', justifyContent: 'center' },
  modalImagen: { width: '92%', height: '80%' },
});
