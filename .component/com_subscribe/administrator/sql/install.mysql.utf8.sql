CREATE TABLE IF NOT EXISTS `#__subscription_products` (
`id` int(11) UNSIGNED NOT NULL AUTO_INCREMENT,

`state` TINYINT(1)  NULL  DEFAULT 1,
`ordering` INT(11)  NULL  DEFAULT 0,
`checked_out` INT(11)  UNSIGNED,
`checked_out_time` DATETIME NULL  DEFAULT NULL ,
`created_by` INT(11)  NULL  DEFAULT 0,
`modified_by` INT(11)  NULL  DEFAULT 0,
`name` VARCHAR(255)  NOT NULL ,
`cost` DOUBLE NOT NULL DEFAULT 0,
`description` TEXT NOT NULL ,
`period` VARCHAR(255)  NOT NULL  DEFAULT "monthly",
`stripe_product_id` VARCHAR(255)  NULL  DEFAULT "",
`test_mode` VARCHAR(255)  NULL  DEFAULT "",
`stripe_test_product_id` VARCHAR(255)  NULL  DEFAULT "",
PRIMARY KEY (`id`)
,KEY `idx_state` (`state`)
,KEY `idx_checked_out` (`checked_out`)
,KEY `idx_created_by` (`created_by`)
,KEY `idx_modified_by` (`modified_by`)
) DEFAULT COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `#__subscription_subscribers` (
`id` int(11) UNSIGNED NOT NULL AUTO_INCREMENT,

`state` TINYINT(1)  NULL  DEFAULT 1,
`ordering` INT(11)  NULL  DEFAULT 0,
`checked_out` INT(11)  UNSIGNED,
`checked_out_time` DATETIME NULL  DEFAULT NULL ,
`created_by` INT(11)  NULL  DEFAULT 0,
`modified_by` INT(11)  NULL  DEFAULT 0,
`user_id` INT(11)  NULL  DEFAULT 0,
`start_date` DATETIME NULL  DEFAULT NULL ,
`end_date` DATETIME NULL  DEFAULT NULL ,
`product_id` INT(10)  NULL  DEFAULT 0,
`stripe_subscription_id` VARCHAR(255)  NULL  DEFAULT "",
`stripe_subscription_secret` VARCHAR(255)  NULL  DEFAULT "",
`stripe_customer_id` VARCHAR(255)  NULL  DEFAULT "",
PRIMARY KEY (`id`)
,KEY `idx_state` (`state`)
,KEY `idx_checked_out` (`checked_out`)
,KEY `idx_created_by` (`created_by`)
,KEY `idx_modified_by` (`modified_by`)
) DEFAULT COLLATE=utf8mb4_unicode_ci;

