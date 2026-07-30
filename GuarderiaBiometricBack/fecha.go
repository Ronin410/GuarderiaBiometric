package main

import "time"

// zonaMazatlan carga la zona horaria de la guardería (Culiacán/Mazatlán). Si el
// contenedor no tiene la base de datos de zonas horarias del sistema operativo,
// cae a UTC en vez de fallar — igual que ya hacían varios de los handlers antes
// de esta extracción (`loc, _ := time.LoadLocation(...)` ignorando el error).
func zonaMazatlan() *time.Location {
	loc, err := time.LoadLocation("America/Mazatlan")
	if err != nil {
		return time.UTC
	}
	return loc
}

// hoyEnZonaLocal formatea "ahora" como fecha (YYYY-MM-DD) en la zona horaria de
// la guardería, no en UTC. Se extrae como función pura (recibe el instante en vez
// de leer time.Now() internamente) para poder probarla sin depender del reloj real
// ni de una base de datos — la misma clase de bug de "fecha equivocada cerca de
// medianoche" que ya se corrigió en el frontend (ver GuarderiaBiometricFront/src/utils/fecha.js)
// es posible aquí si algún día alguien usa time.Now().Format(...) sin convertir la zona primero.
func hoyEnZonaLocal(ahora time.Time) string {
	return ahora.In(zonaMazatlan()).Format("2006-01-02")
}
