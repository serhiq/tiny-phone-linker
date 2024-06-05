CREATE DATABASE IF NOT EXISTS sbot_db DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE sbot_db;

SET NAMES utf8mb4;

select 'START MIGRATION' AS '';

CREATE TABLE user_phone_mapping (
                                    phone_number VARCHAR(16) NOT NULL PRIMARY KEY,
                                    telegram_id int(10) NOT NULL,
                                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                    deleted_at TIMESTAMP NULL DEFAULT NULL
);
select 'END MIGRATION' AS '';


