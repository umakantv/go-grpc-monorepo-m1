-- Creates the per-service databases inside the shared Postgres container.
-- Postgres runs every .sql file in /docker-entrypoint-initdb.d/ on first init.

CREATE DATABASE userdb;
CREATE DATABASE authdb;
CREATE DATABASE filedb;
CREATE DATABASE notificationdb;