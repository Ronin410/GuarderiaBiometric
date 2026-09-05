import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, ActivityIndicator } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import api from '../api/client';
import { useAuth } from '../context/AuthContext';
import { color, radius, sombra } from '../theme';

// Equivalente RN del selector de contactos de ChatPadre.jsx en la web:
// "quiero que al papá le aparezcan los staff o administradores de la
// guardería para escoger con quién hablar" -- ya no es una sola
// conversación con "la guardería" en general.
//
// "en el chat de los papás no aparece el nombre de la guardería" -- se
// pone en el título de la pantalla (no en cada tarjeta de contacto, que ya
// tiene su propio nombre) para que quede claro con qué guardería está
// hablando, sobre todo útil mientras se prueba la app contra distintas
// guarderías.
export default function ChatContactosScreen({ navigation }) {
  const { usuario } = useAuth();
  const [contactos, setContactos] = useState(null);
  const [cargando, setCargando] = useState(true);

  useEffect(() => {
    navigation.setOptions({ title: usuario?.guarderiaNombre ? `Chat · ${usuario.guarderiaNombre}` : 'Chat' });
  }, [navigation, usuario?.guarderiaNombre]);

  useEffect(() => {
    api.get('/padre/chat/contactos')
      .then((res) => setContactos(Array.isArray(res.data) ? res.data : []))
      .catch((err) => {
        console.error('Error al cargar el staff de la guardería:', err);
        setContactos([]);
      })
      .finally(() => setCargando(false));
  }, []);

  if (cargando) {
    return (
      <View style={styles.centro}>
        <ActivityIndicator color={color.brand600} />
      </View>
    );
  }

  if (contactos.length === 0) {
    return (
      <View style={styles.centro}>
        <View style={styles.vacio}>
          <Ionicons name="chatbubble-ellipses-outline" size={36} color={color.slate200} />
          <Text style={styles.vacioTexto}>Tu guardería todavía no tiene personal disponible para chat</Text>
        </View>
      </View>
    );
  }

  return (
    <FlatList
      style={styles.pantalla}
      contentContainerStyle={styles.contenido}
      data={contactos}
      keyExtractor={(ct) => String(ct.id)}
      ListHeaderComponent={
        <View style={{ marginBottom: 4 }}>
          <Text style={styles.encabezado}>¿Con quién quieres hablar?</Text>
          {!!usuario?.guarderiaNombre && <Text style={styles.guarderiaTexto}>{usuario.guarderiaNombre}</Text>}
        </View>
      }
      renderItem={({ item: ct }) => (
        <TouchableOpacity
          style={styles.tarjeta}
          activeOpacity={0.7}
          onPress={() => navigation.navigate('ChatHilo', { contactoId: ct.id, nombre: ct.nombre, rol: ct.rol })}
        >
          <View style={styles.icono}>
            <Ionicons name={ct.rol === 'admin' ? 'shield-checkmark' : 'person'} size={20} color={color.brand600} />
          </View>
          <View style={{ flex: 1 }}>
            <Text style={styles.nombre}>{ct.nombre}</Text>
            <Text style={styles.rol}>{ct.rol === 'admin' ? 'Administración' : 'Personal'}</Text>
          </View>
          <Ionicons name="chevron-forward" size={20} color={color.slate300} />
        </TouchableOpacity>
      )}
    />
  );
}

const styles = StyleSheet.create({
  pantalla: { flex: 1, backgroundColor: color.slate50 },
  contenido: { padding: 20, gap: 10 },
  centro: { flex: 1, backgroundColor: color.slate50, alignItems: 'center', justifyContent: 'center', padding: 32 },
  vacio: { alignItems: 'center', gap: 12 },
  vacioTexto: { color: color.slate400, fontWeight: '700', fontSize: 12, textTransform: 'uppercase', textAlign: 'center' },
  encabezado: { fontSize: 12, fontWeight: '700', color: color.slate400 },
  guarderiaTexto: { fontSize: 10, fontWeight: '900', color: color.brand500, textTransform: 'uppercase', letterSpacing: 0.5, marginTop: 2 },
  tarjeta: {
    flexDirection: 'row', alignItems: 'center', gap: 14, backgroundColor: color.white,
    borderWidth: 1, borderColor: color.slate100, borderRadius: radius.lg, padding: 16, ...sombra.sm,
  },
  icono: { backgroundColor: color.brand100, padding: 10, borderRadius: radius.sm },
  nombre: { fontSize: 13, fontWeight: '900', color: color.slate900, textTransform: 'uppercase' },
  rol: { fontSize: 9, fontWeight: '700', color: color.slate400, textTransform: 'uppercase', marginTop: 2 },
});
