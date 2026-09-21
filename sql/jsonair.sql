--
-- 'keys'
--

DROP TABLE IF EXISTS `keys`;

CREATE TABLE `keys` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uuid` varchar(36) NOT NULL,
  `name` varchar(64) NOT NULL,
  `token` varchar(255) COLLATE utf8mb4_bin NOT NULL,
  `created` timestamp NOT NULL,
  `last_login` timestamp NOT NULL,
  PRIMARY KEY (`id`,`uuid`),
  UNIQUE KEY `idx_token` (`token`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- 'Test' key
--

LOCK TABLES `keys` WRITE;
INSERT INTO `keys` VALUES
(1,'4a972bc2-dd43-4068-863c-b52242c2d3f4','Test API Key - NOT FOR PROD','TESTKEY123', NOW(), NOW());
UNLOCK TABLES;

--
-- `configurations`
--

DROP TABLE IF EXISTS `configurations`;

CREATE TABLE `configurations` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uuid` varchar(36) NOT NULL,
  `reload` varchar(255) NOT NULL,
  `debug` varchar(128) NOT NULL,
  `type` varchar(128) NOT NULL,
  `name` varchar(127) NOT NULL,
  `config_data` mediumtext NOT NULL,
  `created` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`,`uuid`,`name`),
  UNIQUE KEY `idx_uuid_type_name` (`uuid`,`type`,`name`),
  KEY `idx_reload` (`reload`),
  KEY `idx_debug` (`debug`),
  KEY `idx_type` (`type`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


LOCK TABLES `configurations` WRITE;

INSERT INTO `configurations` VALUES
(1,'','RELOADKEY','DEBUGLEVEL','testsub','test.config','{\"config\":\"some value\"}', NOW(), NOW() );

UNLOCK TABLES;


--
-- `write_keys`
--
-- PATs for jsonair-write (the write-only API).  These are completely separate
-- from `keys`: a read PAT is never accepted by jsonair-write, and a write PAT
-- is never accepted by jsonair.
--
--   uuid           Configurations this key writes.  Matches `configurations`.`uuid`,
--                  the same way `keys`.`uuid` does for reads.  It is never taken
--                  from a request.
--   token          HMAC-SHA256 (hex) of the PAT, using WRITE_TOKEN_HMAC_SECRET.
--   allowed_types  Comma separated list of `type` values this key may write.
--   allowed_names  Comma separated list of `name` values this key may write.
--
-- Both allowed_* lists accept an exact value, a prefix ending in '*' (web-*), or
-- a bare '*' for anything.  An empty list allows nothing.  Changes to a key's
-- scope take effect when its current JWT expires (WRITE_JWT_TOKEN_EXPIRE).
--
-- This uses IF NOT EXISTS (no DROP) so it is safe to run against an existing database.
--

CREATE TABLE IF NOT EXISTS `write_keys` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uuid` varchar(36) NOT NULL,
  `name` varchar(64) NOT NULL,
  `token` varchar(255) COLLATE utf8mb4_bin NOT NULL,
  `allowed_types` varchar(512) NOT NULL,
  `allowed_names` varchar(512) NOT NULL,
  `created` timestamp NOT NULL DEFAULT current_timestamp(),
  `last_login` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`,`uuid`),
  UNIQUE KEY `idx_token` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Example (compute the token with:
--   echo -n "YOUR_PAT" | openssl dgst -sha256 -hmac "YOUR_WRITE_TOKEN_HMAC_SECRET" ):
--
--   INSERT INTO `write_keys` (`uuid`,`name`,`token`,`allowed_types`,`allowed_names`)
--   VALUES ('4a972bc2-dd43-4068-863c-b52242c2d3f4','billing web service','<hex hmac>','web','billing-*');


-- Upgrading an existing database
--
-- `idx_uuid_type_name` is required so a configuration is unique per
-- (uuid, type, name).  On an existing database, first look for duplicates:
--
--   SELECT `uuid`,`type`,`name`,COUNT(*) AS n FROM `configurations`
--     GROUP BY `uuid`,`type`,`name` HAVING n > 1;
--
-- Remove or rename any rows returned (the read API returns whichever row
-- MySQL finds first), then add the key:
--
--   ALTER TABLE `configurations`
--     ADD UNIQUE KEY `idx_uuid_type_name` (`uuid`,`type`,`name`);


-- Using the API
--
-- Authentication is two steps: exchange the plain-text PAT (whose HMAC-SHA256
-- hash is stored in `keys`.`token`) for a short-lived JWT, then send the JWT as
-- a Bearer token.  See docs/3.1-authentication.md.
--
--   TOKEN=$(curl -s http://localhost:9191/api/v1/jsonair/auth/token \
--             -d '{"token":"YOUR_PAT"}' | jq -r .access_token)
--
-- Get a configuration (add "decode":true to have the server base64 decode it):
--
--   curl -H "Authorization: Bearer $TOKEN" http://localhost:9191/api/v1/jsonair/config \
--        -X GET -d '{"type":"testsub","name":"test.config"}'
--
-- Get 'reload':
--
--   curl -H "Authorization: Bearer $TOKEN" http://localhost:9191/api/v1/jsonair/reload \
--        -X GET -d '{"type":"testsub","name":"test.config"}'
--
-- Get 'debug':
--
--   curl -H "Authorization: Bearer $TOKEN" http://localhost:9191/api/v1/jsonair/debug \
--        -X GET -d '{"type":"testsub","name":"test.config"}'
