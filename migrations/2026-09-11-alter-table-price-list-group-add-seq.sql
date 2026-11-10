ALTER TABLE price_list_group
ADD COLUMN seq INTEGER;

WITH numbered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (ORDER BY group_name ASC, id ASC) AS seq
    FROM price_list_group
)
UPDATE price_list_group p
SET seq = n.seq
FROM numbered n
WHERE p.id = n.id;