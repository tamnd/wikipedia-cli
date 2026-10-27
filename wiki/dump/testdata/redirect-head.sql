/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19  Distrib 10.11.18-MariaDB, for debian-linux-gnu (x86_64)
--
-- Host: dbstore1008.eqiad.wmnet    Database: enwiki
-- ------------------------------------------------------
-- Server version	10.11.13-MariaDB-log

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `redirect`
--

DROP TABLE IF EXISTS `redirect`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `redirect` (
  `rd_from` int(8) unsigned NOT NULL DEFAULT 0,
  `rd_namespace` int(11) NOT NULL DEFAULT 0,
  `rd_title` varbinary(255) NOT NULL DEFAULT '',
  `rd_interwiki` varbinary(32) DEFAULT NULL,
  `rd_fragment` varbinary(255) DEFAULT NULL,
  PRIMARY KEY (`rd_from`),
  KEY `rd_ns_title` (`rd_namespace`,`rd_title`,`rd_from`)
) ENGINE=InnoDB DEFAULT CHARSET=binary ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `redirect`
--

/*!40000 ALTER TABLE `redirect` DISABLE KEYS */;
INSERT INTO `redirect` VALUES
(10,0,'Computer_accessibility','',''),
(13,0,'History_of_Afghanistan','',''),
(14,0,'Geography_of_Afghanistan','',''),
(15,0,'Demographics_of_Afghanistan','',''),
(18,0,'Communications_in_Afghanistan','',''),
(19,0,'Transport_in_Afghanistan','',''),
(20,0,'Afghan_Armed_Forces','',''),
(21,0,'Foreign_relations_of_Afghanistan','',''),
(23,0,'Assistive_technology','',''),
(24,0,'Amoeba','',''),
(25,0,'Autism','',''),
(27,0,'History_of_Albania','',''),
(29,0,'Demographics_of_Albania','',''),
(30,0,'As_We_May_Think','',''),
(35,0,'Politics_of_Albania','',''),
(36,0,'Economy_of_Albania','',''),
(40,0,'Afroasiatic_languages','',''),
(42,0,'Constructed_language','',''),
(301,0,'Automorphism','','');
/*!40000 ALTER TABLE `redirect` ENABLE KEYS */;
