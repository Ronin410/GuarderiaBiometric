package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

// PinRequest es el cuerpo esperado por /verificar-pin.
type PinRequest struct {
	Pin string `json:"pin"`
}

func (s *Server) registrarRutasAuth(r *gin.Engine) {
	r.POST("/usuarios/registro", s.handleRegistroUsuario)
	r.POST("/login", s.loginLimiter.Middleware(), s.handleLogin)
	r.POST("/verificar-pin", middleware.Auth(s.JWTKey), s.pinLimiter.Middleware(), s.handleVerificarPin)
}

func (s *Server) handleRegistroUsuario(c *gin.Context) {
	var nuevoUsuario struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		GuarderiaID int    `json:"guarderia_id"`
		Rol         string `json:"rol"`
		PinAdmin    string `json:"pin_admin"`
	}

	if err := c.ShouldBindJSON(&nuevoUsuario); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// El costo 10 es el estándar recomendado para el plan Professional que manejas
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(nuevoUsuario.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar seguridad"})
		return
	}

	query := `
        INSERT INTO usuarios (username, password_hash, guarderia_id, rol, pin_admin)
        VALUES ($1, $2, $3, $4, $5)`

	_, err = s.DBAuth.Exec(query,
		nuevoUsuario.Username,
		string(hashedPassword),
		nuevoUsuario.GuarderiaID,
		nuevoUsuario.Rol,
		nuevoUsuario.PinAdmin,
	)
	if err != nil {
		fmt.Printf("Error al insertar usuario: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el usuario"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Usuario creado exitosamente con hash de seguridad"})
}

func (s *Server) handleLogin(c *gin.Context) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	var id, gID int
	var passHash, rol, pinAdmin, gNombre, gSlug string
	// Nota: pin_admin se consulta solo para mantener la forma de la fila; nunca se
	// expone en la respuesta. La verificación del PIN se hace en /verificar-pin.
	query := `
		SELECT
            u.id, u.guarderia_id, u.password_hash, u.rol, u.pin_admin,
            g.nombre, g.slug
        FROM usuarios u
        INNER JOIN guarderias g ON u.guarderia_id = g.id
        WHERE u.username = $1`

	err := s.DBAuth.QueryRow(query, creds.Username).Scan(&id, &gID, &passHash, &rol, &pinAdmin, &gNombre, &gSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("Usuario no encontrado: %s\n", creds.Username)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no existe"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error de BD"})
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passHash), []byte(creds.Password))
	if err != nil {
		log.Printf("Intento de login inválido para usuario %s", creds.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Contraseña incorrecta"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &middleware.Claims{
		UserID:      id,
		GuarderiaID: gID,
		Rol:         rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.JWTKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
		return
	}

	log.Printf("Login exitoso: %s", creds.Username)
	c.JSON(http.StatusOK, gin.H{
		"token":            tokenStr,
		"user_id":          id,
		"guarderia_id":     gID,
		"guarderia_nombre": gNombre,
		"guarderia_slug":   gSlug,
		"rol":              rol,
		"username":         creds.Username,
	})
}

func (s *Server) handleVerificarPin(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req PinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de PIN inválido"})
		return
	}

	var pinDB string
	err := s.DBAuth.QueryRow("SELECT pin_admin FROM usuarios WHERE id = $1", userID).Scan(&pinDB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al verificar PIN"})
		return
	}

	if strings.TrimSpace(pinDB) != strings.TrimSpace(req.Pin) {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false, "error": "PIN incorrecto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "PIN confirmado",
	})
}
