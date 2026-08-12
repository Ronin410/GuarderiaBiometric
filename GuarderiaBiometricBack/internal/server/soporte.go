package server

import (
	"bytes"
	"context"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/robfig/cron/v3"
)

// bucketFotos es el bucket privado donde se guardan las fotos de la
// bitácora. NO debe tener objetos públicos: el "Block Public Access" a
// nivel de bucket debe estar activado en la consola de AWS (ver
// GuarderiaBiometricBack/README.md). El backend nunca sube nada como
// público — solo sirve fotos a través de firmarURLFoto.
const bucketFotos = "biosafe-storage-fotos"

// ttlURLFoto es cuánto dura vigente una URL firmada antes de dejar de
// funcionar. Basta para que se cargue la foto en el navegador; no es un
// enlace permanente como lo era la URL pública anterior.
const ttlURLFoto = time.Hour

// uploadToS3 sube un archivo (foto de bitácora o documento de inscripción)
// al bucket privado y regresa la key del objeto (no una URL: el bucket no
// permite lectura pública).
func (s *Server) uploadToS3(fileHeader *multipart.FileHeader, key string, contentType string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	_, err = s.S3.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketFotos),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}

	return key, nil
}

// borrarDeS3 elimina un objeto del bucket privado. Se usa para no dejar
// huérfano el archivo anterior cuando se reemplaza un documento de
// inscripción, o al eliminar uno explícitamente. Es "fire and forget"
// (mismo criterio que registrarAcceso): si falla, queda en el log de la
// aplicación pero no tumba la respuesta al usuario — un objeto huérfano en
// S3 es un problema de limpieza, no de correctitud de los datos en Postgres.
func (s *Server) borrarDeS3(key string) {
	_, err := s.S3.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketFotos),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("borrarDeS3: no se pudo borrar %q: %v", key, err)
	}
}

// firmarURLFoto genera una URL temporal (presigned) para leer una foto del
// bucket privado. Acepta tanto una key nueva (lo que guarda uploadToS3 desde
// este cambio) como una URL pública completa de una foto subida antes de
// este cambio (fotos_seguimiento.url guarda ambos formatos según cuándo se
// subió la foto) — en ambos casos extrae la key y firma sobre ella.
func (s *Server) firmarURLFoto(valorGuardado string) (string, error) {
	presignClient := s3.NewPresignClient(s.S3)
	req, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketFotos),
		Key:    aws.String(extraerKeyS3(valorGuardado)),
	}, s3.WithPresignExpires(ttlURLFoto))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// extraerKeyS3 recibe lo que haya en fotos_seguimiento.url y regresa la key
// del objeto en S3. Si es una URL pública completa (formato anterior a este
// cambio), recorta todo lo anterior a ".amazonaws.com/"; si ya es una key
// (formato nuevo), la regresa tal cual.
func extraerKeyS3(valorGuardado string) string {
	const marcador = ".amazonaws.com/"
	if idx := strings.Index(valorGuardado, marcador); idx != -1 {
		return valorGuardado[idx+len(marcador):]
	}
	return valorGuardado
}

// IniciarTareasProgramadas arranca el cron de cierre automático nocturno: a
// las 23:00 (zona de la guardería) marca SALIDA a cualquier niño que quedó
// con ENTRADA abierta ese día.
func (s *Server) IniciarTareasProgramadas() {
	location := zonaMazatlan()

	c := cron.New(cron.WithLocation(location))

	_, err := c.AddFunc("0 23 * * *", func() {
		ahora := time.Now().In(location)
		inicioDia := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, location)
		finDia := inicioDia.Add(24 * time.Hour)

		log.Printf("Iniciando cierre automático [%s] entre %v y %v",
			ahora.Format("15:04:05"), inicioDia.Format("2006-01-02"), finDia.Format("2006-01-02"))

		query := `
            INSERT INTO asistencia (hijo_id, padre_id, guarderia_id, tipo_movimiento, fecha_hora, observaciones)
            SELECT DISTINCT ON (a1.hijo_id)
                a1.hijo_id,
                a1.padre_id,
                a1.guarderia_id,
                'SALIDA',
                $1::timestamp with time zone,
                'Cierre automático nocturno'
            FROM asistencia a1
            WHERE a1.tipo_movimiento = 'ENTRADA'
            AND a1.fecha_hora >= $2 AND a1.fecha_hora < $3
            AND NOT EXISTS (
                SELECT 1 FROM asistencia a2
                WHERE a2.hijo_id = a1.hijo_id
                AND a2.tipo_movimiento = 'SALIDA'
                AND a2.fecha_hora >= $2 AND a2.fecha_hora < $3
            )
            ORDER BY a1.hijo_id, a1.fecha_hora DESC`

		result, err := s.DB.Exec(query, ahora, inicioDia, finDia)
		if err != nil {
			log.Printf("FALLO en el cierre: %v", err)
			return
		}

		filas, _ := result.RowsAffected()
		log.Printf("Cierre completado. Niños actualizados: %d", filas)
	})
	if err != nil {
		log.Printf("Error registrando la tarea de cierre automático: %v", err)
		return
	}

	c.Start()
	log.Println("Cron iniciado con rango de fechas seguro")
}
