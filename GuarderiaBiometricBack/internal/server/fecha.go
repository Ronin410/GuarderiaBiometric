package server

import "time"

// zonaMazatlan carga la zona horaria de la guardería (Culiacán/Mazatlán). Si el
// contenedor no tiene la base de datos de zonas horarias del sistema operativo,
// cae a UTC en vez de fallar.
func zonaMazatlan() *time.Location {
	loc, err := time.LoadLocation("America/Mazatlan")
	if err != nil {
		return time.UTC
	}
	return loc
}

// hoyEnZonaLocal formatea "ahora" como fecha (YYYY-MM-DD) en la zona horaria de
// la guardería, no en UTC. Función pura (recibe el instante en vez de leer
// time.Now() internamente) para poder probarla sin depender del reloj real ni
// de una base de datos.
func hoyEnZonaLocal(ahora time.Time) string {
	return ahora.In(zonaMazatlan()).Format("2006-01-02")
}
