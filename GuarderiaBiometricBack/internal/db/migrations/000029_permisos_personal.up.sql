-- Permisos personalizados por docente: reemplaza el PIN compartido
-- (todo-o-nada) por una lista explícita de qué secciones protegidas puede
-- tocar cada cuenta de staff. NULL (el valor de toda cuenta existente hasta
-- que un admin la configure) significa "sin personalizar": acceso completo,
-- el mismo comportamiento de siempre. Un array, incluso vacío, reemplaza esa
-- regla con la lista exacta de áreas permitidas.
ALTER TABLE usuarios ADD COLUMN permisos TEXT[];
