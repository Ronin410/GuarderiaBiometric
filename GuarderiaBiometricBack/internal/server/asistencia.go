package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

// Hijo resume el estado de un niño identificado por Rekognition.
type Hijo struct {
	ID           int    `json:"id"`
	Nombre       string `json:"nombre"`
	UltimoEstado string `json:"ultimo_estado"`
	Activo       bool   `json:"activo"`
}

type RespuestaIdentificacion struct {
	PadreID   int     `json:"padre_id"`
	Padre     string  `json:"padre"`
	Confianza float64 `json:"confianza"`
	Hijos     []Hijo  `json:"hijos"`
	Mensaje   string  `json:"mensaje"`
}

type RegistroAsistencia struct {
	PadreID       int    `json:"padre_id"`
	HijoID        int    `json:"hijo_id"`
	Aseado        bool   `json:"aseado"`
	ReporteGolpe  bool   `json:"reporte_golpe"`
	Observaciones string `json:"observaciones"`
	Tipo          string `json:"tipo"`
}

func (s *Server) registrarRutasAsistencia(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	r.POST("/registrar", auth, s.handleRegistrar)
	r.POST("/identificar", auth, s.identificarLimiter.Middleware(), s.handleIdentificar)
	r.POST("/confirmar-asistencia", auth, s.handleConfirmarAsistencia)
	r.POST("/admin/forzar-estatus", auth, s.handleForzarEstatus)
}

func (s *Server) handleRegistrar(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	colID := getCollectionID(gID)

	var input struct {
		Nombre      string `json:"nombre"`
		Imagen      string `json:"imagen"`
		AceptaAviso bool   `json:"acepta_aviso"`
		// Campos opcionales -- si el staff activa "crear cuenta del portal"
		// en el momento (el tutor está presente ahí mismo), esto crea junto
		// con el rostro una cuenta de acceso rol "papa" para VistaPadre/
		// DashboardPadre. Sin ellos, el registro se comporta exactamente
		// igual que antes: solo queda el dato biométrico en `padres`.
		CrearCuenta bool   `json:"crear_cuenta"`
		Username    string `json:"username"`
		Password    string `json:"password"`
	}
	c.BindJSON(&input)

	// El enrolamiento de un rostro (dato biométrico sensible, de un tutor y
	// que además queda ligado a los datos de sus hijos) no puede proceder sin
	// un Aviso de Privacidad vigente ni sin que el tutor lo haya aceptado
	// explícitamente frente al staff que opera el kiosco.
	var textoAviso, versionAviso string
	if err := s.DB.QueryRow(
		"SELECT COALESCE(aviso_privacidad_texto, ''), aviso_privacidad_version FROM guarderias WHERE id = $1",
		gID,
	).Scan(&textoAviso, &versionAviso); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo verificar el Aviso de Privacidad"})
		return
	}
	if strings.TrimSpace(textoAviso) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El administrador debe configurar el Aviso de Privacidad antes de poder registrar tutores."})
		return
	}
	if !input.AceptaAviso {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debes mostrar el Aviso de Privacidad y obtener la aceptación del tutor antes de registrar su rostro."})
		return
	}

	// Validar la cuenta ANTES de tocar Rekognition -- así un usuario/
	// contraseña inválidos no dejan un rostro ya indexado sin una cuenta que
	// lo acompañe (el registro biométrico solo, sin cuenta, sigue siendo
	// válido y es justo el comportamiento de antes; lo que no queremos es
	// fallar A MEDIAS después de ya haber gastado la llamada a Rekognition).
	input.Username = strings.TrimSpace(input.Username)
	var passwordHashCuenta string
	if input.CrearCuenta {
		if len(input.Username) < 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El usuario del portal debe tener al menos 3 caracteres"})
			return
		}
		if len(input.Password) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "La contraseña del portal debe tener al menos 8 caracteres"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
			return
		}
		passwordHashCuenta = string(hash)
	}

	imgBytes, _ := base64.StdEncoding.DecodeString(input.Imagen)

	// 1. Validar duplicados SOLO en la colección de esta guardería
	searchRes, err := s.Rek.SearchFacesByImage(context.TODO(), &rekognition.SearchFacesByImageInput{
		CollectionId:       aws.String(colID),
		FaceMatchThreshold: aws.Float32(90.0),
		Image:              &types.Image{Bytes: imgBytes},
		MaxFaces:           aws.Int32(1),
	})

	if err == nil && len(searchRes.FaceMatches) > 0 {
		c.JSON(409, gin.H{"error": "Esta persona ya está registrada en esta guardería."})
		return
	}

	// 2. Registro en la colección específica
	indexRes, err := s.Rek.IndexFaces(context.TODO(), &rekognition.IndexFacesInput{
		CollectionId:    aws.String(colID),
		ExternalImageId: aws.String(strings.ReplaceAll(input.Nombre, " ", "_")),
		Image:           &types.Image{Bytes: imgBytes},
	})

	if err != nil || len(indexRes.FaceRecords) == 0 {
		c.JSON(500, gin.H{"error": "Error al procesar rostro"})
		return
	}

	faceID := *indexRes.FaceRecords[0].Face.FaceId
	var nuevoPadreID int
	s.DB.QueryRow("INSERT INTO padres (nombre, face_id, guarderia_id) VALUES ($1, $2, $3) RETURNING id",
		input.Nombre, faceID, gID).Scan(&nuevoPadreID)

	_, err = s.DB.Exec(
		`INSERT INTO consentimientos (padre_id, padre_nombre_historico, guarderia_id, version_aviso, ip)
		 VALUES ($1, $2, $3, $4, $5)`,
		nuevoPadreID, input.Nombre, gID, versionAviso, c.ClientIP(),
	)
	if err != nil {
		log.Printf("No se pudo registrar el consentimiento del padre %d: %v", nuevoPadreID, err)
	}

	respuesta := gin.H{"status": "OK", "padre_id": nuevoPadreID}

	// El portal del papá (DashboardPadre.jsx) resuelve "quién soy" con el
	// user_id del JWT usado directo como padre_id (ver el comodín "0" en
	// handleHijosDePadre, y el mismo criterio en pagos.go/recibos.go/
	// pagos_online.go) -- por eso esta cuenta se crea forzando
	// usuarios.id = padres.id, en vez de agregar una columna de relación
	// nueva y tener que tocar las cuatro consultas que ya asumen esa
	// igualdad. setval() empuja la secuencia de usuarios más allá de este id
	// explícito para que el próximo alta de personal (secuencial, sin id
	// explícito) no choque con él.
	if input.CrearCuenta {
		_, errCuenta := s.DBAuth.Exec(
			`INSERT INTO usuarios (id, guarderia_id, username, password_hash, pin_admin, rol, nombre, activo)
             VALUES ($1, $2, $3, $4, '0000', 'papa', $5, true)`,
			nuevoPadreID, gID, input.Username, passwordHashCuenta, input.Nombre,
		)
		if errCuenta != nil {
			if esUsernameDuplicado(errCuenta) {
				respuesta["cuenta_error"] = "El rostro se registró, pero ese nombre de usuario ya existe. Crea la cuenta del portal por separado con otro usuario."
			} else {
				log.Printf("No se pudo crear la cuenta de portal para el padre %d: %v", nuevoPadreID, errCuenta)
				respuesta["cuenta_error"] = "El rostro se registró, pero no se pudo crear la cuenta del portal. Inténtalo de nuevo por separado."
			}
		} else {
			if _, errSeq := s.DBAuth.Exec(`SELECT setval('usuarios_id_seq', (SELECT MAX(id) FROM usuarios))`); errSeq != nil {
				log.Printf("No se pudo reacomodar la secuencia de usuarios tras crear la cuenta del padre %d: %v", nuevoPadreID, errSeq)
			}
			respuesta["cuenta_creada"] = true
		}
	}

	c.JSON(200, respuesta)
}

func (s *Server) handleIdentificar(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	colID := getCollectionID(gID)

	var input struct {
		Imagen string `json:"imagen"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Imagen requerida"})
		return
	}

	imgBytes, _ := base64.StdEncoding.DecodeString(input.Imagen)

	// 1. Identificación facial con Rekognition
	result, err := s.Rek.SearchFacesByImage(context.TODO(), &rekognition.SearchFacesByImageInput{
		CollectionId:       aws.String(colID),
		FaceMatchThreshold: aws.Float32(90.0),
		Image:              &types.Image{Bytes: imgBytes},
		MaxFaces:           aws.Int32(1),
	})

	if err != nil || len(result.FaceMatches) == 0 {
		c.JSON(404, gin.H{"mensaje": "No reconocido en esta sede"})
		return
	}

	faceID := *result.FaceMatches[0].Face.FaceId
	confianza := float64(*result.FaceMatches[0].Similarity)

	// 2. CONSULTA CON ZONA HORARIA CORRECTA (America/Mazatlan para Culiacán)
	// Esta query asegura que "hoy" termine a las 12:00 AM de Culiacán, no de Londres.
	query := `
		SELECT
	        p.id AS padre_id,
			p.nombre AS padre_nombre,
			n.id AS hijo_id,
			n.nombre_niño AS hijo_nombre,
			COALESCE((
				SELECT tipo_movimiento
				FROM asistencia
				WHERE hijo_id = n.id
				AND guarderia_id = $2
				-- Solo un AT TIME ZONE para convertir de UTC a local correctamente
				AND (fecha_hora AT TIME ZONE 'America/Mazatlan')::date =
					(CURRENT_TIMESTAMP AT TIME ZONE 'America/Mazatlan')::date
				ORDER BY fecha_hora DESC
				LIMIT 1
			), 'AUSENTE') as ultimo_estado
		FROM padres p
		INNER JOIN tutor_hijos tn ON p.id = tn.padre_id
		INNER JOIN hijos n ON tn.hijo_id = n.id
		WHERE p.face_id = $1
		AND p.guarderia_id = $2
		AND n.activo = true`

	rows, err := s.DB.Query(query, faceID, gID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error en base de datos: " + err.Error()})
		return
	}
	defer rows.Close()

	var padreID int
	var nombrePadre string
	var hijos []Hijo

	for rows.Next() {
		var hID int
		var hNom string
		var hEst string

		err := rows.Scan(&padreID, &nombrePadre, &hID, &hNom, &hEst)
		if err != nil {
			continue
		}

		hijos = append(hijos, Hijo{
			ID:           hID,
			Nombre:       hNom,
			UltimoEstado: hEst,
		})
	}

	if len(hijos) == 0 {
		c.JSON(404, gin.H{"mensaje": "Padre identificado pero no tiene hijos activos asignados"})
		return
	}

	c.JSON(200, RespuestaIdentificacion{
		PadreID:   padreID,
		Padre:     nombrePadre,
		Confianza: confianza,
		Hijos:     hijos,
	})
}

func (s *Server) handleConfirmarAsistencia(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var req RegistroAsistencia
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Buscamos el último registro de HOY
	var ultimoTipo string
	err := s.DB.QueryRow(`
        SELECT tipo_movimiento
        FROM asistencia
        WHERE hijo_id = $1
          AND guarderia_id = $2
          AND fecha_hora::date = CURRENT_DATE
        ORDER BY fecha_hora DESC
        LIMIT 1`, req.HijoID, gID).Scan(&ultimoTipo)

	// Lógica de decisión:
	// 1. Si no hay registro hoy (err != nil) -> Es una ENTRADA.
	// 2. Si el último fue ENTRADA -> Es una SALIDA.
	// 3. Si el último fue SALIDA -> re-entrada.
	tipoFinal := "ENTRADA"
	if err == nil {
		if ultimoTipo == "ENTRADA" {
			tipoFinal = "SALIDA"
		} else if ultimoTipo == "SALIDA" {
			tipoFinal = "ENTRADA"
		}
	}

	query := `
        INSERT INTO asistencia (padre_id, hijo_id, aseado, reporte_golpe, observaciones, tipo_movimiento, guarderia_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = s.DB.Exec(query, req.PadreID, req.HijoID, req.Aseado, req.ReporteGolpe, req.Observaciones, tipoFinal, gID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar"})
		return
	}

	go s.notificarEvento(req.HijoID, tipoFinal, "")

	c.JSON(http.StatusOK, gin.H{
		"status":  "Registro guardado",
		"tipo":    tipoFinal,
		"hijo_id": req.HijoID,
	})
}

func (s *Server) handleForzarEstatus(c *gin.Context) {
	var req struct {
		HijoID     int    `json:"hijo_id"`
		Movimiento string `json:"tipo_movimiento"`
	}

	gID, _ := c.Get("guarderia_id")

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	location := zonaMazatlan()
	ahora := time.Now().In(location)

	// Buscar el padre_id del ÚLTIMO registro de asistencia de ese niño
	var padreID int
	err := s.DB.QueryRow(`
        SELECT padre_id
        FROM asistencia
        WHERE hijo_id = $1
        ORDER BY fecha_hora DESC
        LIMIT 1`, req.HijoID).Scan(&padreID)

	if err != nil {
		if err == sql.ErrNoRows {
			// FALLBACK: Si no hay historial, buscamos al tutor asignado en tutor_hijos
			err = s.DB.QueryRow("SELECT padre_id FROM tutor_hijos WHERE hijo_id = $1 LIMIT 1", req.HijoID).Scan(&padreID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "El niño no tiene historial ni tutor asignado"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar historial"})
			return
		}
	}

	query := `
        INSERT INTO asistencia (hijo_id, padre_id, guarderia_id, tipo_movimiento, fecha_hora, observaciones)
        VALUES ($1, $2, $3, $4, $5, $6)`

	observacion := "Actualizado por Admin"

	_, err = s.DB.Exec(query, req.HijoID, padreID, gID, req.Movimiento, ahora, observacion)
	if err != nil {
		fmt.Println("Error al forzar estatus:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar el movimiento"})
		return
	}

	go s.notificarEvento(req.HijoID, req.Movimiento, "")

	c.JSON(http.StatusOK, gin.H{
		"message": "Estatus actualizado correctamente",
		"detalles": gin.H{
			"movimiento": req.Movimiento,
			"hora_local": ahora.Format("15:04:05"),
		},
	})
}
