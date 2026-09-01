package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/applog"
	"biometrico/internal/middleware"
)

// guarderia.go -- configuración general de LA PROPIA guardería que su admin
// puede editar desde /panel/configuracion, DISTINTO de plataforma.go (que
// es el dueño de la plataforma administrando/bloqueando TODAS las
// guarderías). Empieza solo con el nombre; es el lugar natural para sumar
// más adelante otros campos editables de la guardería (logo, dirección,
// etc.) si hacen falta.

// maxLongitudNombreGuarderia coincide con guarderias.nombre VARCHAR(100)
// (migración 000001) -- se cuenta en runas, no en bytes, porque un nombre
// con acentos/ñ pesa más bytes que caracteres en UTF-8, y VARCHAR(100) de
// Postgres limita por caracteres.
const maxLongitudNombreGuarderia = 100

func (s *Server) registrarRutasGuarderia(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	// RequireAdmin, no RequireArea: igual que el Aviso de Privacidad,
	// renombrar la guardería es exclusivo del admin -- el staff no debe
	// poder tocarlo ni con permisos personalizados.
	admin := middleware.RequireAdmin()
	r.PUT("/admin/guarderia", auth, admin, s.handleActualizarGuarderia)
}

func (s *Server) handleActualizarGuarderia(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var input struct {
		Nombre string `json:"nombre"`
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

	if _, err := s.DB.Exec(`UPDATE guarderias SET nombre = $1 WHERE id = $2`, nombre, gID); err != nil {
		s.logError(c, "No se pudo actualizar el nombre de la guardería", err, "guarderia_id", gID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el nombre"})
		return
	}

	applog.Info("Nombre de guardería actualizado desde Configuración", "guarderia_id", gID)
	c.JSON(http.StatusOK, gin.H{"nombre": nombre})
}
