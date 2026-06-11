-- Уникальность public_ip для идемпотентной регистрации узла оркестратором.
ALTER TABLE nodes ADD CONSTRAINT nodes_public_ip_key UNIQUE (public_ip);
