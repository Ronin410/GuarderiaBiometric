package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/applog"
	"biometrico/internal/middleware"
)

// plataforma.go -- panorama de las guarderías YA aprobadas (a diferencia de
// solicitudes.go, que solo maneja las solicitudes pendientes de
// aprobación). "Quiero ver qué guarderías están registradas, cuántos papás
// y cuántos niños hay registrados, y más información relevante" -- para
// que el dueño de la plataforma tenga un panorama de adopción sin tener
// que entrar a cada guardería una por una.
type GuarderiaPlataforma struct {
	ID         int     `json:"id"`
	Nombre     string  `json:"nombre"`
	Slug       string  `json:"slug"`
	Direccion  *string `json:"direccion"`
	CreadoEn   string  `json:"creado_en"`
	TotalNinos int     `json:"total_ninos"`
	TotalPapas int     `json:"total_papas"`
	// TotalAdmin/TotalMaestras en vez de un solo TotalStaff -- "cuántas
	// maestras tienen" se preguntó explícito, y el proyecto solo tiene dos
	// roles de personal (admin/staff, ver personal.go): admin es
	// prácticamente siempre la dirección, así que separar deja "maestras"
	// como un número que de verdad responde esa pregunta.
	TotalAdmin    int     `json:"total_admin"`
	TotalMaestras int     `json:"total_maestras"`
	UltimoAcceso  *string `json:"ultimo_acceso"`
	// "Quiero una forma de bloquear el acceso a una guardería... eso lo
	// haré manualmente" -- ver handleBloquearGuarderia/
	// handleDesbloquearGuarderia más abajo.
	Bloqueada   bool    `json:"bloqueada"`
	BloqueadaEn *string `json:"bloqueada_en"`
}

func (s *Server) registrarRutasPlataformaGuarderias(r *gin.Engine) {
	platform := middleware.RequirePlatformKey(s.PlatformAdminKey)
	r.GET("/plataforma/guarderias", platform, s.handleListarGuarderiasPlataforma)
	r.PUT("/plataforma/guarderias/:id", platform, s.handleActualizarGuarderiaPlataforma)
	r.POST("/plataforma/guarderias/:id/bloquear", platform, s.handleBloquearGuarderia)
	r.POST("/plataforma/guarderias/:id/desbloquear", platform, s.handleDesbloquearGuarderia)
	r.POST("/plataforma/guarderias/:id/reiniciar-reconocimiento-facial", platform, s.handleReiniciarReconocimientoFacial)
}

// handleReiniciarReconocimientoFacial -- herramienta de rescate para el
// caso real que destapó esto: "creé una guardería nueva y ya me reconoce
// de otra" porque guarderia_id es un SERIAL reciclable y ese número ya
// tenía una colección de Rekognition con rostros de una guardería
// anterior que se borró de Postgres sin que nadie borrara también SU
// colección en AWS (son sistemas distintos). Esto vuelve a dejar la
// colección de la guardería genuinamente vacía -- DESTRUCTIVO a propósito:
// cualquier tutor que ya se haya enrolado de verdad en ESTA guardería
// tendría que volver a registrar su rostro después de esto, así que solo
// tiene sentido usarlo en una guardería que, como esta, nunca ha logrado
// que un registro se complete.
func (s *Server) handleReiniciarReconocimientoFacial(c *gin.Context) {
	id := c.Param("id")
	colID := getCollectionID(id)

	if err := s.recrearColeccionRekognitionLimpia(c.Request.Context(), colID); err != nil {
		s.logError(c, "No se pudo reiniciar la colección de Rekognition", err, "guarderia_id", id, "coleccion", colID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo reiniciar el reconocimiento facial"})
		return
	}

	applog.Info("Colección de Rekognition reiniciada desde /plataforma", "guarderia_id", id, "coleccion", colID)
	c.JSON(http.StatusOK, gin.H{"status": "Reconocimiento facial reiniciado. Cualquier rostro enrolado antes en esta guardería ya no será reconocido."})
}

// handleActualizarGuarderiaPlataforma -- a diferencia de
// handleActualizarGuarderia (guarderia.go), que deja al ADMIN de una
// guardería cambiar su propio nombre, esto es el dueño de la plataforma
// corrigiendo nombre/dirección de CUALQUIER guardería sin tener que
// pedirle a esa guardería que entre a hacerlo. No toca slug (se usa como
// identificador estable en URLs/exportaciones) ni plan_suscripcion/
// bloqueada (esos ya tienen su propio control dedicado).
func (s *Server) handleActualizarGuarderiaPlataforma(c *gin.Context) {
	id := c.Param("id")

	var input struct {
		Nombre    string `json:"nombre"`
		Direccion string `json:"direccion"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	nombre := strings.TrimSpace(input.Nombre)
	if nombre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre de la guardería no puede quedar vacío"})
		return
	}
	if len([]rune(nombre)) > maxLongitudNombreGuarderia {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre es demasiado largo (máximo 100 caracteres)"})
		return
	}
	direccion := strings.TrimSpace(input.Direccion)

	res, err := s.DBAuth.Exec(
		`UPDATE guarderias SET nombre = $1, direccion = $2 WHERE id = $3`,
		nombre, direccion, id,
	)
	if err != nil {
		s.logError(c, "No se pudo actualizar la guardería desde /plataforma", err, "guarderia_id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar la guardería"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardería no encontrada"})
		return
	}

	applog.Info("Guardería editada desde /plataforma", "guarderia_id", id)
	c.JSON(http.StatusOK, gin.H{"nombre": nombre, "direccion": direccion})
}

// handleBloquearGuarderia corta el acceso de TODA la guardería (admin,
// staff y papás) -- "si no pagan manualmente quiero quitarles el acceso
// dándoles tiempo para que paguen, pero eso lo haré manualmente": nada
// automático la vuelve a bloquear ni la desbloquea sola, es un interruptor
// que el dueño de la plataforma prende/apaga a mano. Ver el efecto real en
// handleLogin/handleMe (auth.go).
func (s *Server) handleBloquearGuarderia(c *gin.Context) {
	id := c.Param("id")
	res, err := s.DBAuth.Exec(`UPDATE guarderias SET bloqueada = true, bloqueada_en = now() WHERE id = $1`, id)
	if err != nil {
		s.logError(c, "Error al bloquear la guardería", err, "guarderia_id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo bloquear la guardería"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardería no encontrada"})
		return
	}
	applog.Info("Guardería bloqueada desde /plataforma", "guarderia_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "Guardería bloqueada"})
}

func (s *Server) handleDesbloquearGuarderia(c *gin.Context) {
	id := c.Param("id")
	res, err := s.DBAuth.Exec(`UPDATE guarderias SET bloqueada = false, bloqueada_en = NULL WHERE id = $1`, id)
	if err != nil {
		s.logError(c, "Error al desbloquear la guardería", err, "guarderia_id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo desbloquear la guardería"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardería no encontrada"})
		return
	}
	applog.Info("Guardería desbloqueada desde /plataforma", "guarderia_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "Guardería desbloqueada"})
}

// handleListarGuarderiasPlataforma junta datos de DBAuth (guarderias,
// usuarios) y de DB (hijos, logs_acceso) en tres consultas por separado --
// database/sql no puede unir en una sola query dos *sql.DB distintos,
// aunque en este despliegue ambos apunten a la misma base física (mismo
// motivo que el comentario sobre usuario_id en la migración 000015 de
// logs_acceso).
func (s *Server) handleListarGuarderiasPlataforma(c *gin.Context) {
	rows, err := s.DBAuth.Query(`
        SELECT g.id, g.nombre, g.slug, g.direccion, g.created_at, g.bloqueada, g.bloqueada_en,
               COUNT(*) FILTER (WHERE u.rol = 'papa') AS total_papas,
               COUNT(*) FILTER (WHERE u.rol = 'admin' AND u.activo) AS total_admin,
               COUNT(*) FILTER (WHERE u.rol = 'staff' AND u.activo) AS total_maestras
        FROM guarderias g
        LEFT JOIN usuarios u ON u.guarderia_id = g.id
        GROUP BY g.id, g.nombre, g.slug, g.direccion, g.created_at, g.bloqueada, g.bloqueada_en
        ORDER BY g.created_at DESC`)
	if err != nil {
		s.logError(c, "Error al consultar guarderías", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar guarderías"})
		return
	}
	guarderias := []GuarderiaPlataforma{}
	for rows.Next() {
		var g GuarderiaPlataforma
		if err := rows.Scan(&g.ID, &g.Nombre, &g.Slug, &g.Direccion, &g.CreadoEn, &g.Bloqueada, &g.BloqueadaEn, &g.TotalPapas, &g.TotalAdmin, &g.TotalMaestras); err != nil {
			continue
		}
		guarderias = append(guarderias, g)
	}
	rows.Close()

	ninosPorGuarderia := map[int]int{}
	rowsNinos, err := s.DB.Query(`SELECT guarderia_id, COUNT(*) FROM hijos WHERE activo = true GROUP BY guarderia_id`)
	if err != nil {
		s.logError(c, "Error al contar niños por guardería", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al contar niños por guardería"})
		return
	}
	for rowsNinos.Next() {
		var gID, total int
		if err := rowsNinos.Scan(&gID, &total); err != nil {
			continue
		}
		ninosPorGuarderia[gID] = total
	}
	rowsNinos.Close()

	// Última vez que alguien de esa guardería (admin, staff o papá) entró
	// de verdad -- un indicador rápido de si la guardería sigue usando el
	// sistema activamente o quedó abandonada tras el alta.
	ultimoAccesoPorGuarderia := map[int]string{}
	rowsAcceso, err := s.DB.Query(`
        SELECT guarderia_id, MAX(creado_en) FROM logs_acceso
        WHERE evento = 'login_exitoso' AND guarderia_id IS NOT NULL
        GROUP BY guarderia_id`)
	if err != nil {
		s.logError(c, "Error al consultar el último acceso por guardería", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el último acceso por guardería"})
		return
	}
	for rowsAcceso.Next() {
		var gID int
		var ultimo string
		if err := rowsAcceso.Scan(&gID, &ultimo); err != nil {
			continue
		}
		ultimoAccesoPorGuarderia[gID] = ultimo
	}
	rowsAcceso.Close()

	for i := range guarderias {
		guarderias[i].TotalNinos = ninosPorGuarderia[guarderias[i].ID]
		if ultimo, ok := ultimoAccesoPorGuarderia[guarderias[i].ID]; ok {
			guarderias[i].UltimoAcceso = &ultimo
		}
	}

	c.JSON(http.StatusOK, guarderias)
}
