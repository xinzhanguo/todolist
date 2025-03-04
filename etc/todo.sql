CREATE TABLE `todos` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uid` char(128) NOT NULL DEFAULT '' COMMENT 'The uid',
  `content` text,
  `token` char(128) NOT NULL DEFAULT '' COMMENT 'The token',
  `tokey` char(128) NOT NULL DEFAULT '' COMMENT 'The tokey',
  `code` char(128) NOT NULL DEFAULT '' COMMENT 'The share code',
  `style` char(128) NOT NULL DEFAULT '' COMMENT 'The style',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_index` (`uid`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='todos table';


ALTER TABLE todos ADD COLUMN `token` char(128) NOT NULL DEFAULT '' COMMENT 'The token';
ALTER TABLE todos ADD COLUMN `tokey` char(128) NOT NULL DEFAULT '' COMMENT 'The key';
ALTER TABLE todos ADD COLUMN `code` char(128) NOT NULL DEFAULT '' COMMENT 'The code';
ALTER TABLE todos ADD COLUMN `style` char(128) NOT NULL DEFAULT '' COMMENT 'The style';
