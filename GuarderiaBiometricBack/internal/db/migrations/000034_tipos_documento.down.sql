ALTER TABLE documentos_nino DROP CONSTRAINT IF EXISTS documentos_nino_tipo_fk;
ALTER TABLE documentos_nino ADD CONSTRAINT documentos_nino_tipo_check CHECK (tipo IN (
    'acta_nacimiento', 'curp', 'comprobante_domicilio',
    'cartilla_vacunacion', 'identificacion_tutor', 'otro'
));

DROP TABLE IF EXISTS tipos_documento;
