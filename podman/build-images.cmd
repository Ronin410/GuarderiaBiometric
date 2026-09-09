@echo off
REM Construye las dos imagenes con los valores correctos siempre -- evita
REM errores de dedo al escribir el --build-arg (protocolo/puerto) a mano.
REM Uso: corre este archivo desde dentro de la carpeta podman\ (doble clic
REM tambien funciona, o "podman\build-images.cmd" desde otra carpeta).

echo ==^> Construyendo imagen del backend...
podman build -t guarderia-backend:local -f ../GuarderiaBiometricBack/Dockerfile ../GuarderiaBiometricBack
if errorlevel 1 goto :error

echo ==^> Construyendo imagen del frontend...
podman build -t guarderia-frontend:local --build-arg VITE_API_URL=https://localhost:8099 -f ../GuarderiaBiometricFront/Dockerfile ../GuarderiaBiometricFront
if errorlevel 1 goto :error

echo.
echo Listo. Ahora corre: podman play kube kube.yaml
goto :eof

:error
echo.
echo Hubo un error construyendo una de las imagenes -- revisa el mensaje de arriba.
exit /b 1
