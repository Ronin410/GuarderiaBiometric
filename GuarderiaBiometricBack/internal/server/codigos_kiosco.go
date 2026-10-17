package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// codigos_kiosco.go -- "que un administrador genere un código QR... para
// que lleguen, escaneen y le den entrada o salida al niño desde su
// teléfono". A diferencia de codigos_acceso.go (lo genera el papá, dura
// días, autoriza a una persona nombrada), este lo genera la TABLET y es
// anónimo a propósito: no identifica a ningún niño ni tutor, solo prueba
// "alguien lo escaneó parado frente a esta tablet, en este momento". La
// identidad real sale de la sesión del papá que lo escanea desde SU PROPIA
// app, nunca del código -- por eso un papá solo puede terminar viendo y
// confirmando movimientos de SUS PROPIOS hijos, sin importar qué tablet
// mostró el QR.
//
// Vida corta (1 minuto) + un solo uso: entre las dos cosas, ni sirve
// fotografiarlo para usarlo después lejos de la tablet, ni lo pueden usar
// dos papás a la vez con el mismo código.
const vigenciaCodigoKiosco = 60 * time.Second

func (s *Server) registrarRutasCodigosKiosco(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()
	// La tablet genera y vuelve a consultar el mismo código una y otra vez
	// (polling) para saber cuándo mostrar uno nuevo -- ver
	// EscaneoQRKioscoPadre/el modo "Autoservicio" de Recepción en el
	// frontend.
	r.POST("/admin/codigo-kiosco", auth, staff, s.handleGenerarCodigoKiosco)
	r.GET("/admin/codigo-kiosco/:id", auth, staff, s.handleEstadoCodigoKiosco)
	// Papá -- canjea el código con SU propia sesión y confirma el
	// movimiento de SU propio hijo. Exige rol "papa" explícito, igual que
	// codigos_acceso.go: esto registra una entrada/salida real, no es algo
	// de lo que valga la pena heredar la laxitud de otras rutas /padre/*.
	r.POST("/padre/codigo-kiosco/canjear", auth, s.handleCanjearCodigoKiosco)
	r.POST("/padre/confirmar-asistencia", auth, s.handleConfirmarAsistenciaPadre)
}

// handleGenerarCodigoKiosco -- lo llama la tablet, no un papá. Un token
// nuevo cada vez que se llama; el frontend decide cuándo pedir uno (ver
// handleEstadoCodigoKiosco).
func (s *Server) handleGenerarCodigoKiosco(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	token, err := generarTokenCodigoAcceso()
	if err != nil {
		s.logError(c, "No se pudo generar el token del código de kiosco", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código"})
		return
	}
	expiraEn := time.Now().Add(vigenciaCodigoKiosco)

	var id int
	if err := s.DB.QueryRow(
		`INSERT INTO codigos_kiosco (guarderia_id, token, expira_en) VALUES ($1, $2, $3) RETURNING id`,
		gID, token, expiraEn,
	).Scan(&id); err != nil {
		s.logError(c, "No se pudo crear el código de kiosco", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "token": token, "expira_en": expiraEn})
}

// handleEstadoCodigoKiosco -- la tablet lo consulta cada pocos segundos
// mientras el QR sigue en pantalla, para enterarse EN CUANTO alguien lo usa
// (sin esperar a que se cumpla el minuto completo) y pedir uno nuevo de
// inmediato.
func (s *Server) handleEstadoCodigoKiosco(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	id := c.Param("id")

	var expiraEn time.Time
	var usadoEn sql.NullTime
	err := s.DB.QueryRow(
		`SELECT expira_en, usado_en FROM codigos_kiosco WHERE id = $1 AND guarderia_id = $2`,
		id, gID,
	).Scan(&expiraEn, &usadoEn)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Código no encontrado"})
		return
	} else if err != nil {
		s.logError(c, "No se pudo consultar el código de kiosco", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en base de datos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"usado":    usadoEn.Valid,
		"expirado": time.Now().After(expiraEn),
	})
}

// handleCanjearCodigoKiosco -- el papá escanea el QR de la tablet con SU
// PROPIO teléfono, ya logueado en su cuenta. Regresa SUS hijos (nunca los
// de nadie más, eso sale de tutor_hijos filtrado por su propio padre_id,
// no del código) con su último estado de hoy, igual que handleIdentificar.
func (s *Server) handleCanjearCodigoKiosco(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "papa" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Esta función es solo para cuentas de papá/tutor"})
		return
	}
	gID, _ := c.Get("guarderia_id")
	padreID, _ := c.Get("user_id")

	var input struct {
		Token string `json:"token"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Código requerido"})
		return
	}
	token := strings.TrimSpace(input.Token)
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Código requerido"})
		return
	}

	var codigoID int
	var expiraEn time.Time
	var usadoEn sql.NullTime
	err := s.DB.QueryRow(
		`SELECT id, expira_en, usado_en FROM codigos_kiosco WHERE token = $1 AND guarderia_id = $2`,
		token, gID,
	).Scan(&codigoID, &expiraEn, &usadoEn)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"mensaje": "Código no válido en esta guardería"})
		return
	} else if err != nil {
		s.logError(c, "Error al consultar el código de kiosco", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en base de datos"})
		return
	}
	if usadoEn.Valid {
		c.JSON(http.StatusGone, gin.H{"mensaje": "Este código ya fue usado. Escanea el que esté en pantalla ahora."})
		return
	}
	if time.Now().After(expiraEn) {
		c.JSON(http.StatusGone, gin.H{"mensaje": "Este código ya venció. Escanea el que esté en pantalla ahora."})
		return
	}

	// Canjear de forma atómica -- WHERE usado_en IS NULL evita que dos
	// papás que escanearon casi al mismo tiempo lo canjeen los dos: el
	// segundo UPDATE afecta 0 filas y se entera aquí mismo.
	res, err := s.DB.Exec(
		`UPDATE codigos_kiosco SET usado_en = NOW(), usado_por = $1 WHERE id = $2 AND usado_en IS NULL`,
		padreID, codigoID,
	)
	if err != nil {
		s.logError(c, "No se pudo canjear el código de kiosco", err, "codigo_id", codigoID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en base de datos"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusGone, gin.H{"mensaje": "Este código ya fue usado. Escanea el que esté en pantalla ahora."})
		return
	}

	rows, err := s.DB.Query(`
        SELECT h.id, h.nombre_niño,
               COALESCE((
                   SELECT tipo_movimiento
                   FROM asistencia
                   WHERE hijo_id = h.id
                     AND guarderia_id = $2
                     AND (fecha_hora AT TIME ZONE 'America/Mazatlan')::date =
                         (CURRENT_TIMESTAMP AT TIME ZONE 'America/Mazatlan')::date
                   ORDER BY fecha_hora DESC
                   LIMIT 1
               ), 'AUSENTE') AS ultimo_estado
        FROM hijos h
        JOIN tutor_hijos th ON th.hijo_id = h.id
        WHERE th.padre_id = $1 AND h.guarderia_id = $2 AND h.activo = true`,
		padreID, gID,
	)
	if err != nil {
		s.logError(c, "Error al consultar tus hijos", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en base de datos"})
		return
	}
	defer rows.Close()

	hijos := []Hijo{}
	for rows.Next() {
		var h Hijo
		if err := rows.Scan(&h.ID, &h.Nombre, &h.UltimoEstado); err != nil {
			continue
		}
		hijos = append(hijos, h)
	}

	if len(hijos) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"mensaje": "No se encontraron niños activos vinculados a tu cuenta"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"codigo_kiosco_id": codigoID,
		"hijos":            hijos,
	})
}

// handleConfirmarAsistenciaPadre -- el paso final: el papá, ya con la
// lista de SUS hijos en pantalla (desde handleCanjearCodigoKiosco), elige
// uno y confirma. padre_id SIEMPRE sale de la sesión, nunca del cuerpo de
// la petición -- a diferencia de /confirmar-asistencia (de uso exclusivo
// de staff, que sí confía en el padre_id que manda porque staff ya
// verificó la identidad con rostro/código antes), aquí el papá llama este
// endpoint DIRECTO desde su propia cuenta, así que confiar en un padre_id
// del body dejaría que cualquiera marcara asistencia a nombre de otro
// tutor con solo cambiar un número.
func (s *Server) handleConfirmarAsistenciaPadre(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "papa" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Esta función es solo para cuentas de papá/tutor"})
		return
	}
	gID, _ := c.Get("guarderia_id")
	padreIDRaw, _ := c.Get("user_id")
	padreID, _ := padreIDRaw.(int)

	var input struct {
		HijoID         int    `json:"hijo_id"`
		CodigoKioscoID int    `json:"codigo_kiosco_id"`
		Aseado         bool   `json:"aseado"`
		ReporteGolpe   bool   `json:"reporte_golpe"`
		Observaciones  string `json:"observaciones"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	if !s.hijoPerteneceAPadre(strconv.Itoa(input.HijoID), padreID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Ese niño no pertenece a tu cuenta"})
		return
	}

	// Revalida que ESTE código de kiosco de verdad lo canjeó ESTE papá --
	// nunca confiar en que el frontend lo diga, mismo criterio que ya usa
	// /confirmar-asistencia con codigo_acceso_id.
	var usadoPor sql.NullInt64
	if err := s.DB.QueryRow(
		`SELECT usado_por FROM codigos_kiosco WHERE id = $1 AND guarderia_id = $2`,
		input.CodigoKioscoID, gID,
	).Scan(&usadoPor); err != nil || !usadoPor.Valid || int(usadoPor.Int64) != padreID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Este código ya no es válido. Vuelve a escanear el QR de la tablet."})
		return
	}

	// Misma zona horaria que el resto del código (ver asistencia.go) -- sin
	// esto, una SALIDA real entrada la noche en Mazatlán se detectaba como
	// si no hubiera habido ENTRADA ese día y volvía a quedar como ENTRADA.
	var ultimoTipo string
	err := s.DB.QueryRow(`
        SELECT tipo_movimiento
        FROM asistencia
        WHERE hijo_id = $1
          AND guarderia_id = $2
          AND (fecha_hora AT TIME ZONE 'America/Mazatlan')::date =
              (CURRENT_TIMESTAMP AT TIME ZONE 'America/Mazatlan')::date
        ORDER BY fecha_hora DESC
        LIMIT 1`, input.HijoID, gID).Scan(&ultimoTipo)

	tipoFinal := "ENTRADA"
	if err == nil {
		if ultimoTipo == "ENTRADA" {
			tipoFinal = "SALIDA"
		} else if ultimoTipo == "SALIDA" {
			tipoFinal = "ENTRADA"
		}
	}

	nota := "Confirmado por el propio tutor desde su app, vía QR del kiosco"
	observaciones := strings.TrimSpace(input.Observaciones)
	if observaciones != "" {
		nota = nota + " — " + observaciones
	}
	// "¿Aseado?" / "¿Golpes?" son preguntas de ENTRADA (cómo llegó el niño),
	// igual que en el kiosco de recepción (App.jsx) -- en SALIDA no aplican
	// y se guardan en false sin importar lo que mande el cliente.
	aseado := tipoFinal == "ENTRADA" && input.Aseado
	reporteGolpe := tipoFinal == "ENTRADA" && input.ReporteGolpe

	_, err = s.DB.Exec(
		`INSERT INTO asistencia (padre_id, hijo_id, aseado, reporte_golpe, observaciones, tipo_movimiento, guarderia_id, codigo_kiosco_id)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		padreID, input.HijoID, aseado, reporteGolpe, nota, tipoFinal, gID, input.CodigoKioscoID,
	)
	if err != nil {
		s.logError(c, "No se pudo guardar la asistencia (confirmación del propio papá)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar"})
		return
	}

	go s.notificarEvento(input.HijoID, tipoFinal, "")

	c.JSON(http.StatusOK, gin.H{"status": "Registro guardado", "tipo": tipoFinal, "hijo_id": input.HijoID})
}
