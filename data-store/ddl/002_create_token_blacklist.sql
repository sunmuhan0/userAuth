-- Token黑名单表
CREATE TABLE IF NOT EXISTS `token_blacklist` (
  `token_hash` VARCHAR(64) NOT NULL COMMENT 'token SHA256 hash',
  `expires_at` DATETIME NOT NULL COMMENT '过期时间（到期后可清理）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`token_hash`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Token黑名单';
