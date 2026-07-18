-- Rename workspace to Chinese (see 10g)
UPDATE tenants
SET name = '栖月汇的工作区'
WHERE id = '7ae527da-ab07-4f3c-9cce-148bae909b69';

SELECT id, name FROM tenants;
