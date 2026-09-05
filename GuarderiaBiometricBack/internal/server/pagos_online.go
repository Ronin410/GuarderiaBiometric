package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"

	"biometrico/internal/applog"
	"biometrico/internal/middleware"
)

// pagos_online.go es "Pagos móviles" del PDF: pagar la colegiatura en línea
// (tarjeta) en vez de que el papá lleve efectivo/transferencia y el staff lo
// capture a mano en /pagos. Se deja preparado a propósito SIN activar: sin
// STRIPE_SECRET_KEY configurada, StripeHabilitado() es false y estas rutas
// responden "no disponible" en vez de fallar -- el resto de la app (incluido
// el registro manual de pagos que ya existía) sigue exactamente igual.
//
// Se eligió Stripe Checkout (sesión hospedada por Stripe, el navegador del
// papá se redirige ahí y vuelve) en vez de Stripe Elements/Payment Element
// embebido: no exige tarjeta ni datos de pago pasando por este backend en
// ningún momento (menor alcance de PCI-DSS), y no requiere agregar el SDK de
// JS de Stripe al frontend hasta que de verdad se vaya a usar.
//
// V1 solo cubre el concepto "Colegiatura" (el recurrente, el que ya calcula
// un saldo pendiente vía colegiatura_mensual - lo pagado del periodo) -- los
// demás conceptos (Material, Inscripción, Otro) se quedan como pago manual
// por ahora; extenderlo es agregar el mismo cálculo de saldo para ellos.
func (s *Server) registrarRutasPagosOnline(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)

	if s.StripeHabilitado() {
		// stripe.Key es una variable global del SDK (no hay un cliente por
		// request) -- se fija una sola vez aquí, al registrar las rutas.
		stripe.Key = s.StripeSecretKey
	}

	r.GET("/pagos-online/config", auth, s.handleConfigPagosOnline)
	r.POST("/padre/pagos-online/checkout", auth, s.handleCrearCheckoutColegiatura)
	// Sin auth a propósito: lo llama Stripe, no un usuario logueado --
	// webhook.ConstructEvent (más abajo) es lo que autentica la petición,
	// verificando la firma HMAC del header Stripe-Signature.
	r.POST("/webhooks/stripe", s.handleWebhookStripe)
}

// handleConfigPagosOnline le dice al frontend si mostrar o no el botón de
// "Pagar en línea" -- publicable_key es pública por diseño de Stripe (viaja
// en el HTML de cualquier página de Checkout), nunca la secreta.
func (s *Server) handleConfigPagosOnline(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"habilitado":     s.StripeHabilitado(),
		"publicable_key": s.StripePublishableKey,
	})
}

// handleCrearCheckoutColegiatura crea una Stripe Checkout Session por la
// colegiatura pendiente de un hijo en un periodo, y regresa la URL a la que
// el frontend debe redirigir al papá. El TOPE del monto siempre se calcula
// aquí desde la BD (colegiatura_mensual - lo ya pagado) -- nunca se confía
// ciegamente en lo que mande el cliente; dentro de ese tope, el papá puede
// pedir pagar menos (monto opcional en el body, ver más abajo).
func (s *Server) handleCrearCheckoutColegiatura(c *gin.Context) {
	if !s.StripeHabilitado() {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Los pagos en línea todavía no están disponibles en esta guardería."})
		return
	}

	gID, _ := c.Get("guarderia_id")
	padreID, _ := c.Get("user_id")

	var input struct {
		HijoID  string  `json:"hijo_id"`
		Periodo string  `json:"periodo"` // YYYY-MM
		Monto   float64 `json:"monto"`   // opcional -- "cuánto puede pagar" el papá; si se omite o es <= 0, se cobra el saldo pendiente completo
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.HijoID == "" || len(input.Periodo) != 7 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hijo_id y periodo (YYYY-MM) son obligatorios"})
		return
	}

	if !s.hijoPerteneceAPadre(input.HijoID, padreID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Ese niño no pertenece a tu cuenta"})
		return
	}

	var nombreHijo string
	var colegiatura, totalPagado float64
	err := s.DB.QueryRow(`
        SELECT h.nombre_niño, h.colegiatura_mensual,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Colegiatura'), 0)
        FROM hijos h
        LEFT JOIN pagos p ON p.hijo_id = h.id AND p.periodo = $2
        WHERE h.id = $1 AND h.guarderia_id = $3
        GROUP BY h.id, h.nombre_niño, h.colegiatura_mensual`,
		input.HijoID, input.Periodo, gID,
	).Scan(&nombreHijo, &colegiatura, &totalPagado)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró al niño"})
		return
	}

	saldo := colegiatura - totalPagado
	if saldo <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No hay saldo pendiente de colegiatura para ese periodo"})
		return
	}

	// "Quiero que el papá pueda seleccionar cuánto puede pagar" -- puede
	// pagar MENOS del saldo completo (ej. solo tiene $1000 de los $2000);
	// el resto sigue viendo "parcial"/"pendiente" ese periodo y entra a la
	// deuda acumulada del mes siguiente automáticamente, igual que un pago
	// parcial capturado a mano por staff (ver DeudaAcumulada en pagos.go).
	// Lo que NO puede es pagar de MÁS por esta vía -- si el monto pedido
	// pasa del saldo, se rechaza en vez de aceptar un excedente sin dueño
	// claro; pagar deuda de meses viejos lo sigue capturando staff a mano
	// contra ese periodo específico (ver el comentario de V1 más arriba).
	monto := saldo
	if input.Monto > 0 {
		if input.Monto > saldo+0.005 { // tolerancia de centavos por redondeo
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("El monto no puede ser mayor al saldo pendiente ($%.2f)", saldo)})
			return
		}
		monto = input.Monto
	}
	// Piso de $10: además de no tener sentido cobrar centavos por tarjeta,
	// Stripe rechaza montos por debajo de su mínimo por moneda -- más claro
	// avisarlo aquí con un mensaje en español que dejar que la API de
	// Stripe regrese un error genérico más abajo.
	if monto < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El monto mínimo para pagar en línea es $10"})
		return
	}

	// success_url/cancel_url regresan al portal del papá con un query param
	// que DashboardPadre.jsx lee para mostrar un aviso de éxito/cancelado.
	// Apuntan a /panel/identificar y NO a "/" a propósito: la ruta "/" del
	// frontend es un <Navigate to="/panel/identificar" replace /> (ver
	// App.jsx), y ese tipo de redirección de React Router NO conserva el
	// query string -- el aviso nunca se vería a tiempo de por medio.
	baseURL := s.FrontendURL
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(baseURL + "/panel/identificar?pago_colegiatura=exito"),
		CancelURL:         stripe.String(baseURL + "/panel/identificar?pago_colegiatura=cancelado"),
		ClientReferenceID: stripe.String(fmt.Sprintf("colegiatura:hijo:%s:periodo:%s", input.HijoID, input.Periodo)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(s.StripeCurrency),
				UnitAmount: stripe.Int64(int64(monto*100 + 0.5)), // pesos -> centavos
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(fmt.Sprintf("Colegiatura %s — %s", nombreHijo, input.Periodo)),
					Description: stripe.String("Pago en línea de colegiatura"),
				},
			},
		}},
		Metadata: map[string]string{
			"guarderia_id": fmt.Sprintf("%v", gID),
			"hijo_id":      input.HijoID,
			"padre_id":     fmt.Sprintf("%v", padreID),
			"periodo":      input.Periodo,
			"concepto":     "Colegiatura",
		},
	}

	sesion, err := session.New(params)
	if err != nil {
		s.logError(c, "Error al crear Checkout Session de Stripe", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "No se pudo iniciar el pago en línea. Intenta de nuevo más tarde."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": sesion.URL})
}

// handleWebhookStripe recibe la confirmación de Stripe de que un Checkout se
// completó y registra el pago en la misma tabla `pagos` que usa la captura
// manual de staff -- así /pagos, /pagos/estado y el portal del papá lo ven
// igual sin importar cómo se haya pagado. Idempotente por diseño: si Stripe
// reintenta el mismo evento (lo hace si no respondemos 2xx rápido), el
// índice único sobre stripe_checkout_session_id evita duplicar el pago.
func (s *Server) handleWebhookStripe(c *gin.Context) {
	if !s.StripeHabilitado() || s.StripeWebhookSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook de Stripe no configurado"})
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se pudo leer el cuerpo de la petición"})
		return
	}

	// IgnoreAPIVersionMismatch: true -- la cuenta de Stripe puede estar
	// pineada a una versión de API distinta a la que trae compilada esta
	// versión del SDK (normal, sobre todo recién creada la cuenta); eso no
	// debe tirar la confirmación de un pago real. La firma SÍ se sigue
	// verificando siempre, sin excepción -- eso es lo que autentica que el
	// webhook de verdad viene de Stripe.
	event, err := webhook.ConstructEventWithOptions(
		payload, c.GetHeader("Stripe-Signature"), s.StripeWebhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		s.logError(c, "Webhook de Stripe con firma inválida", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Firma inválida"})
		return
	}

	if event.Type != "checkout.session.completed" {
		// Cualquier otro tipo de evento (los hay de sobra -- Stripe manda
		// todo lo que la cuenta tenga suscrito) se reconoce con 200 sin
		// hacer nada: no es un error, simplemente no nos interesa.
		c.JSON(http.StatusOK, gin.H{"recibido": true})
		return
	}

	var sesion stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sesion); err != nil {
		s.logError(c, "No se pudo interpretar el checkout.session.completed de Stripe", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Evento inválido"})
		return
	}

	if sesion.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		// ej. modo "setup" o pago diferido que aún no se liquida -- todavía
		// no hay nada que asentar en `pagos`.
		c.JSON(http.StatusOK, gin.H{"recibido": true})
		return
	}

	gID := sesion.Metadata["guarderia_id"]
	hijoID := sesion.Metadata["hijo_id"]
	periodo := sesion.Metadata["periodo"]
	concepto := sesion.Metadata["concepto"]
	if gID == "" || hijoID == "" || periodo == "" || concepto == "" {
		applog.Warn("checkout.session.completed sin metadata esperada", "stripe_session_id", sesion.ID)
		c.JSON(http.StatusOK, gin.H{"recibido": true})
		return
	}

	var paymentIntentID *string
	if sesion.PaymentIntent != nil && sesion.PaymentIntent.ID != "" {
		paymentIntentID = &sesion.PaymentIntent.ID
	}

	_, err = s.DB.Exec(`
        INSERT INTO pagos (guarderia_id, hijo_id, monto, concepto, periodo, fecha_pago, metodo_pago, observaciones, stripe_checkout_session_id, stripe_payment_intent_id)
        VALUES ($1, $2, $3, $4, $5, $6, 'Stripe', 'Pago en línea vía Stripe', $7, $8)
        ON CONFLICT (stripe_checkout_session_id) WHERE stripe_checkout_session_id IS NOT NULL DO NOTHING`,
		gID, hijoID, float64(sesion.AmountTotal)/100, concepto, periodo,
		time.Now().In(zonaMazatlan()), sesion.ID, paymentIntentID,
	)
	if err != nil {
		s.logError(c, "No se pudo registrar el pago de Stripe", err, "stripe_session_id", sesion.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar el pago"})
		return
	}

	applog.Info("Pago en línea registrado vía Stripe", "guarderia_id", gID, "hijo_id", hijoID, "periodo", periodo, "stripe_session_id", sesion.ID)
	c.JSON(http.StatusOK, gin.H{"recibido": true})
}
