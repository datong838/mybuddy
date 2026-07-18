UPDATE tenants
SET custom_config = '{"remove_webapp_brand": true}'
WHERE id = '7ae527da-ab07-4f3c-9cce-148bae909b69';

SELECT id, name, custom_config FROM tenants;
