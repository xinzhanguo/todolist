CREATE TABLE `todos` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uid` char(128) NOT NULL DEFAULT '' COMMENT 'The uid',
  `content` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_index` (`uid`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='todos table';
