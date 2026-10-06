-- Название банка хранится отдельно от идентификатора счёта. Старые демо-записи
-- остаются работоспособны после обновления.
ALTER TABLE participants ADD COLUMN bank_name VARCHAR(120) NOT NULL DEFAULT 'Демо-банк';
