package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"biometrico/internal/middleware"
)

// diasPermitidosCodigoAcceso son las únicas vigencias que un papá puede
// elegir al generar un código -- "corta, de días, no de horas": alcanza
// para "hoy nada más va mi cuñado" (1) hasta "esta semana me ayuda mi mamá
// con las recogidas" (7), sin dejar un código vivo indefinidamente.
var diasPermitidosCodigoAcceso = map[int]bool{1: true, 3: true, 7: true}

// alfabetoCodigoCorto evita 0/O/1/I/L -- el código corto existe para que
// recepción lo teclee a mano cuando la cámara no logra leer el QR (mala
// luz, brillo de la pantalla de otro celular), así que lo que importa es
// que se pueda dictar y transcribir sin ambigüedad, no la entropía.
const alfabetoCodigoCorto = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func (s *Server) registrarRutasCodigosAcceso(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	r.POST("/padre/codigos-acceso", auth, s.handleGenerarCodigoAcceso)
	r.GET("/padre/codigos-acceso", auth, s.handleListarCodigosAcceso)
	r.DELETE("/padre/codigos-acceso/:id", auth, s.handleRevocarCodigoAcceso)
	// Kiosco/Recepción -- identifica por código en vez de rostro, mismo
	// criterio de autorización que ya tienen /identificar y
	// /confirmar-asistencia (solo Auth(), sin RequireStaff): la UI del
	// papá no expone este flujo, igual que no expone el de reconocimiento
	// facial del kiosco.
	r.POST("/identificar-codigo", auth, s.handleIdentificarPorCodigo)
}

// generarTokenCodigoAcceso produce el valor largo que lleva el QR -- mismo
// patrón que generarCSRFToken/generarTokenProspecto (crypto/rand + hex).
func generarTokenCodigoAcceso() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generarCodigoCorto produce 6 caracteres del alfabeto sin ambigüedades.
// No usa crypto/rand.Int con un módulo chico de forma ingenua (eso sesga
// la distribución salvo que el tamaño del alfabeto divida exacto a 256) --
// rand.Int con un big.Int del tamaño del alfabeto sí es uniforme.
func generarCodigoCorto() (string, error) {
	n := big.NewInt(int64(len(alfabetoCodigoCorto)))
	b := make([]byte, 6)
	for i := range b {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		b[i] = alfabetoCodigoCorto[idx.Int64()]
	}
	return string(b), nil
}

func esCodigoCortoDuplicado(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505" && pqErr.Constraint == "codigos_acceso_codigo_corto_key"
}

// handleGenerarCodigoAcceso -- "una sección de generar QR" en el portal del
// papá. Exige rol "papa" explícito (a diferencia de otras rutas /padre/*
// de este proyecto que solo confían en que la UI no expone el flujo a
// otros roles): a diferencia de ver un historial de pagos, esto autoriza a
// alguien a llevarse a un niño, así que aquí sí vale la pena no heredar esa
// laxitud.
func (s *Server) handleGenerarCodigoAcceso(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "papa" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Esta función es solo para cuentas de papá/tutor"})
		return
	}
	gID, _ := c.Get("guarderia_id")
	padreID, _ := c.Get("user_id")

	var input struct {
		HijosIDs          []int  `json:"hijos_ids"`
		PersonaAutorizada string `json:"persona_autorizada"`
		Dias              int    `json:"dias"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	input.PersonaAutorizada = strings.TrimSpace(input.PersonaAutorizada)

	if len(input.HijosIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Elige al menos un niño"})
		return
	}
	if len(input.PersonaAutorizada) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Escribe el nombre de quién va a usar este código"})
		return
	}
	if !diasPermitidosCodigoAcceso[input.Dias] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La vigencia debe ser 1, 3 o 7 días"})
		return
	}
	for _, hijoID := range input.HijosIDs {
		if !s.hijoPerteneceAPadre(strconv.Itoa(hijoID), padreID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Uno de los niños no pertenece a tu cuenta"})
			return
		}
	}

	token, err := generarTokenCodigoAcceso()
	if err != nil {
		s.logError(c, "No se pudo generar el token del código de acceso", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código"})
		return
	}
	expiraEn := time.Now().In(zonaMazatlan()).AddDate(0, 0, input.Dias)

	tx, err := s.DB.Begin()
	if err != nil {
		s.logError(c, "No se pudo iniciar la transacción del código de acceso", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código"})
		return
	}
	defer tx.Rollback()

	var codigoID int
	var codigoCorto string
	// Reintenta unas pocas veces si el código corto choca contra uno ya
	// existente -- con 33^6 combinaciones (~1,291 millones) es rarísimo,
	// pero más barato reintentar que devolverle un error al papá por eso.
	for intento := 0; intento < 5; intento++ {
		codigoCorto, err = generarCodigoCorto()
		if err != nil {
			s.logError(c, "No se pudo generar el código corto", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código"})
			return
		}
		err = tx.QueryRow(
			`INSERT INTO codigos_acceso (padre_id, guarderia_id, token, codigo_corto, persona_autorizada, expira_en)
             VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			padreID, gID, token, codigoCorto, input.PersonaAutorizada, expiraEn,
		).Scan(&codigoID)
		if err == nil || !esCodigoCortoDuplicado(err) {
			break
		}
	}
	if err != nil {
		s.logError(c, "No se pudo crear el código de acceso", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código, intenta de nuevo"})
		return
	}

	for _, hijoID := range input.HijosIDs {
		if _, err := tx.Exec(`INSERT INTO codigos_acceso_hijos (codigo_id, hijo_id) VALUES ($1, $2)`, codigoID, hijoID); err != nil {
			s.logError(c, "No se pudo vincular el niño al código de acceso", err, "codigo_id", codigoID, "hijo_id", hijoID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		s.logError(c, "No se pudo confirmar la creación del código de acceso", err, "codigo_id", codigoID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el código"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           codigoID,
		"token":        token,
		"codigo_corto": codigoCorto,
		"expira_en":    expiraEn,
	})
}

// handleListarCodigosAcceso regresa todos los códigos que ha generado este
// papá (activos e históricos -- vencidos o revocados incluidos, para que
// pueda ver qué autorizó en el pasado, no solo lo vigente).
func (s *Server) handleListarCodigosAcceso(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "papa" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Esta función es solo para cuentas de papá/tutor"})
		return
	}
	padreID, _ := c.Get("user_id")

	rows, err := s.DB.Query(`
        SELECT ca.id, ca.persona_autorizada, ca.codigo_corto, ca.expira_en, ca.revocado_en, ca.creado_en,
               COALESCE(string_agg(h.nombre_niño, ', ' ORDER BY h.nombre_niño), '')
        FROM codigos_acceso ca
        LEFT JOIN codigos_acceso_hijos cah ON cah.codigo_id = ca.id
        LEFT JOIN hijos h ON h.id = cah.hijo_id
        WHERE ca.padre_id = $1
        GROUP BY ca.id
        ORDER BY ca.creado_en DESC`, padreID)
	if err != nil {
		s.logError(c, "No se pudieron consultar los códigos de acceso", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudieron consultar los códigos de acceso"})
		return
	}
	defer rows.Close()

	type codigoListado struct {
		ID                int        `json:"id"`
		PersonaAutorizada string     `json:"persona_autorizada"`
		CodigoCorto       string     `json:"codigo_corto"`
		ExpiraEn          time.Time  `json:"expira_en"`
		RevocadoEn        *time.Time `json:"revocado_en,omitempty"`
		CreadoEn          time.Time  `json:"creado_en"`
		Hijos             string     `json:"hijos"`
	}

	codigos := []codigoListado{}
	for rows.Next() {
		var cl codigoListado
		var revocadoEn sql.NullTime
		if err := rows.Scan(&cl.ID, &cl.PersonaAutorizada, &cl.CodigoCorto, &cl.ExpiraEn, &revocadoEn, &cl.CreadoEn, &cl.Hijos); err != nil {
			continue
		}
		if revocadoEn.Valid {
			cl.RevocadoEn = &revocadoEn.Time
		}
		codigos = append(codigos, cl)
	}

	c.JSON(http.StatusOK, codigos)
}

// handleRevocarCodigoAcceso -- "si quiere revisar algo" aplicado aquí
// también: el papá puede cortar un código antes de que venza solo (lo
// compartió por error, cambió de opinión, etc.) sin tener que esperar a
// que expire.
func (s *Server) handleRevocarCodigoAcceso(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "papa" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Esta función es solo para cuentas de papá/tutor"})
		return
	}
	padreID, _ := c.Get("user_id")
	codigoID := c.Param("id")

	result, err := s.DB.Exec(
		`UPDATE codigos_acceso SET revocado_en = NOW() WHERE id = $1 AND padre_id = $2 AND revocado_en IS NULL`,
		codigoID, padreID,
	)
	if err != nil {
		s.logError(c, "No se pudo revocar el código de acceso", err, "codigo_id", codigoID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo revocar el código"})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Código no encontrado o ya estaba revocado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Código revocado"})
}

// handleIdentificarPorCodigo es el equivalente de handleIdentificar (que
// busca por rostro) pero para el QR/código corto -- la pantalla de
// Recepción ofrece los dos modos. Acepta tanto el token largo del QR como
// el código corto tecleado a mano; guarderia_id SIEMPRE se exige igual que
// el token/código, para que un código de otra guardería (aunque alguien lo
// escriba a mano por error) nunca identifique a nadie aquí.
func (s *Server) handleIdentificarPorCodigo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var input struct {
		Codigo string `json:"codigo"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Código requerido"})
		return
	}
	codigo := strings.TrimSpace(input.Codigo)
	if codigo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Código requerido"})
		return
	}

	var codigoID, padreID int
	var nombrePadre, personaAutorizada string
	var expiraEn time.Time
	var revocadoEn sql.NullTime
	err := s.DB.QueryRow(`
        SELECT ca.id, ca.padre_id, p.nombre, ca.persona_autorizada, ca.expira_en, ca.revocado_en
        FROM codigos_acceso ca
        JOIN padres p ON p.id = ca.padre_id
        WHERE (ca.token = $1 OR ca.codigo_corto = $2) AND ca.guarderia_id = $3`,
		codigo, strings.ToUpper(codigo), gID,
	).Scan(&codigoID, &padreID, &nombrePadre, &personaAutorizada, &expiraEn, &revocadoEn)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"mensaje": "Código no válido en esta guardería"})
		return
	} else if err != nil {
		s.logError(c, "Error al consultar el código de acceso", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en base de datos"})
		return
	}

	if revocadoEn.Valid {
		c.JSON(http.StatusGone, gin.H{"mensaje": "Este código fue revocado por el tutor"})
		return
	}
	if time.Now().After(expiraEn) {
		c.JSON(http.StatusGone, gin.H{"mensaje": "Este código ya venció"})
		return
	}

	rows, err := s.DB.Query(`
        SELECT h.id, h.nombre_niño
        FROM codigos_acceso_hijos cah
        JOIN hijos h ON h.id = cah.hijo_id
        WHERE cah.codigo_id = $1 AND h.activo = true`, codigoID)
	if err != nil {
		s.logError(c, "Error al consultar los niños del código de acceso", err, "codigo_id", codigoID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en base de datos"})
		return
	}
	defer rows.Close()

	var hijos []Hijo
	for rows.Next() {
		var h Hijo
		if err := rows.Scan(&h.ID, &h.Nombre); err != nil {
			continue
		}
		hijos = append(hijos, h)
	}

	if len(hijos) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"mensaje": "Este código no tiene niños activos vinculados"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"codigo_acceso_id":   codigoID,
		"padre_id":           padreID,
		"padre":              nombrePadre,
		"persona_autorizada": personaAutorizada,
		"hijos":              hijos,
	})
}

// validarCodigoAccesoParaConfirmar revalida un código de acceso al momento
// de CONFIRMAR el movimiento, no solo al identificar -- nunca hay que
// confiar en que el frontend diga "esto vino de un código válido": entre
// identificar y confirmar pueden pasar varios segundos (recepción revisando
// la bitácora, preguntando algo), tiempo de sobra para que el papá revoque
// el código desde su celular. Regresa el texto a anexar a observaciones.
func (s *Server) validarCodigoAccesoParaConfirmar(codigoAccesoID, hijoID int, gID, padreID any) (string, error) {
	var personaAutorizada string
	var expiraEn time.Time
	var revocadoEn sql.NullTime
	var autorizaEsteHijo bool
	err := s.DB.QueryRow(`
        SELECT ca.persona_autorizada, ca.expira_en, ca.revocado_en,
               EXISTS(SELECT 1 FROM codigos_acceso_hijos WHERE codigo_id = ca.id AND hijo_id = $2)
        FROM codigos_acceso ca
        WHERE ca.id = $1 AND ca.guarderia_id = $3 AND ca.padre_id = $4`,
		codigoAccesoID, hijoID, gID, padreID,
	).Scan(&personaAutorizada, &expiraEn, &revocadoEn, &autorizaEsteHijo)
	if err != nil {
		return "", fmt.Errorf("código de acceso no encontrado: %w", err)
	}
	if revocadoEn.Valid {
		return "", fmt.Errorf("código revocado")
	}
	if time.Now().After(expiraEn) {
		return "", fmt.Errorf("código vencido")
	}
	if !autorizaEsteHijo {
		return "", fmt.Errorf("este código no autoriza a este niño")
	}
	return fmt.Sprintf("Vía código de acceso — autorizado para: %s", personaAutorizada), nil
}
