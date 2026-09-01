import React from 'react';
import { View, ActivityIndicator, StyleSheet } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';

import { AuthProvider, useAuth } from './src/context/AuthContext';
import LoginScreen from './src/screens/LoginScreen';
import DashboardScreen from './src/screens/DashboardScreen';
import BitacoraScreen from './src/screens/BitacoraScreen';
import ChatContactosScreen from './src/screens/ChatContactosScreen';
import ChatHiloScreen from './src/screens/ChatHiloScreen';
import EncuestasScreen from './src/screens/EncuestasScreen';
import CircularesScreen from './src/screens/CircularesScreen';
import EventosScreen from './src/screens/EventosScreen';
import ProximamenteScreen from './src/screens/ProximamenteScreen';
import { color } from './src/theme';

const Stack = createNativeStackNavigator();

// Mismo criterio que MainApp() en App.jsx de la web: mientras no se sabe si
// hay sesión (cargando), loading; sin sesión, login; con sesión, la app de
// verdad -- solo que aquí, en vez de pestañas dentro de una sola pantalla,
// es un stack de React Navigation (Dashboard como raíz, empujando
// Bitácora/Próximamente encima, con el botón "atrás" nativo de cada
// plataforma en vez de un botón "Volver" hecho a mano).
function Root() {
  const { cargando, autenticado } = useAuth();

  if (cargando) {
    return (
      <View style={styles.cargando}>
        <ActivityIndicator size="large" color={color.brand600} />
      </View>
    );
  }

  if (!autenticado) {
    return <LoginScreen />;
  }

  return (
    <Stack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: color.white },
        headerTintColor: color.slate900,
        headerTitleStyle: { fontWeight: '900', fontSize: 14, textTransform: 'uppercase' },
        headerShadowVisible: false,
        contentStyle: { backgroundColor: color.slate50 },
      }}
    >
      <Stack.Screen name="Dashboard" component={DashboardScreen} options={{ headerShown: false }} />
      <Stack.Screen name="Bitacora" component={BitacoraScreen} options={{ title: '' }} />
      <Stack.Screen name="ChatContactos" component={ChatContactosScreen} options={{ title: 'Chat' }} />
      <Stack.Screen name="ChatHilo" component={ChatHiloScreen} options={{ title: '' }} />
      <Stack.Screen name="Encuestas" component={EncuestasScreen} options={{ title: 'Encuestas' }} />
      <Stack.Screen name="Circulares" component={CircularesScreen} options={{ title: 'Avisos' }} />
      <Stack.Screen name="Eventos" component={EventosScreen} options={{ title: 'Eventos' }} />
      <Stack.Screen name="Proximamente" component={ProximamenteScreen} options={{ title: '' }} />
    </Stack.Navigator>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <NavigationContainer>
          <StatusBar style="dark" />
          <Root />
        </NavigationContainer>
      </AuthProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  cargando: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: color.paper },
});
