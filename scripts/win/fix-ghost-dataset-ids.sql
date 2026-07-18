-- 清理 Chatflow「知识库实验辅助」知识检索节点幽灵 dataset_ids
-- 仅保留仍存在的 qiyuehui-kb：2456acc6-91c7-4bcf-8632-471527c9d986
-- 见 docs/palantier/10_v01/10f-知识检索幽灵dataset_ids说明.md

UPDATE workflows
SET graph = (
  WITH nodes AS (
    SELECT jsonb_agg(
      CASE
        WHEN elem->'data'->>'type' = 'knowledge-retrieval'
        THEN jsonb_set(
          elem,
          '{data,dataset_ids}',
          '["2456acc6-91c7-4bcf-8632-471527c9d986"]'::jsonb
        )
        ELSE elem
      END
      ORDER BY ordinality
    ) AS new_nodes
    FROM jsonb_array_elements(graph::jsonb->'nodes') WITH ORDINALITY AS t(elem, ordinality)
  )
  SELECT jsonb_set(graph::jsonb, '{nodes}', new_nodes)::text FROM nodes
),
updated_at = CURRENT_TIMESTAMP(0)
WHERE app_id = '804cfc3a-088f-4cb4-9588-dd613568438e'
  AND (
    version = 'draft'
    OR id = '7c2f9632-0350-4450-b752-213e8087895c'
  );

SELECT version, graph::jsonb->'nodes'->1->'data'->'dataset_ids' AS ds
FROM workflows
WHERE app_id = '804cfc3a-088f-4cb4-9588-dd613568438e'
  AND (
    version = 'draft'
    OR id = '7c2f9632-0350-4450-b752-213e8087895c'
  );
