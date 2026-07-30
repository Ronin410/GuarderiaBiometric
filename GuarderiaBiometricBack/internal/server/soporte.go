package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/robfig/cron/v3"
)

// uploadToS3 sube una foto de la bitácora al bucket público y regresa su URL.
func (s *Server) uploadToS3(fileHeader *multipart.FileHeader, fileName string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}

	client := s3.NewFromConfig(cfg)
	bucketName := "biosafe-storage-fotos"

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String("image/jpeg"),
		ACL:         s3types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", err
	}

	region := s.AWSRegion
	if region == "" {
		region = "us-east-1"
	}

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, fileName)
	return url, nil
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
