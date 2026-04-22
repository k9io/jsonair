--
-- 'keys'
--

DROP TABLE IF EXISTS `keys`;

CREATE TABLE `keys` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uuid` varchar(36) NOT NULL,
  `name` varchar(64) NOT NULL,
  `key` varchar(255) COLLATE utf8mb4_bin NOT NULL,
  `created` timestamp NOT NULL,
  `last_login` timestamp NOT NULL,
  PRIMARY KEY (`id`,`uuid`),
  UNIQUE KEY `idx_key` (`key`)
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
  KEY `idx_reload` (`reload`),
  KEY `idx_debug` (`debug`),
  KEY `idx_type` (`type`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


LOCK TABLES `configurations` WRITE;

INSERT INTO `configurations` VALUES
(1,'','RELOADKEY','DEBUGLEVEL','testsub','test.config','{\"config\":\"some value\"}', NOW(), NOW() )

UNLOCK TABLES;


-- Get's configurations
--
-- curl -H 'API_KEY: 4a972bc2-dd43-4068-863c-b52242c2d3f4:TESTKEY123' http://localhost:9191/config -XPOST -d'{"type":"testsub","name":"test.config"}'
 
-- Gets 'reload'
--
-- curl -H 'API_KEY: 4a972bc2-dd43-4068-863c-b52242c2d3f4:TESTKEY123' http://localhost:9191/reload -XPOST -d'{"type":"testsub","name":"test.config"}'

-- Gets 'debug'
--
-- curl -H 'API_KEY: 4a972bc2-dd43-4068-863c-b52242c2d3f4:TESTKEY123' http://localhost:9191/debug -XPOST -d'{"type":"testsub","name":"test.config"}'


