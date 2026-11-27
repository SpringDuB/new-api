-- Stolas PostgreSQL only. psql -X -f this-file        = read-only preview
--                        psql -X -v apply=on -f ... = apply reviewed plan
-- Frozen snapshot: 2026-11-27 11:01:30 Asia/Shanghai, default group, closed hours.
-- Match logs to tasks and final GIN responses by request_id, not error-row count.
-- Include both image routes. Exclude retry attempts, eventual successes from
-- the error set, and the three Sunburst 400s. Preserve throughput/TTFT fields.
-- Missing latency is an estimate: final request_policy elapsed_ms for errors;
-- for the two successes, client access-log durations (122520 + 308115 ms).
\set ON_ERROR_STOP on
\if :{?apply}
\else
  \set apply off
\endif
\if :apply
  BEGIN;
  SET LOCAL lock_timeout = '3s';
  LOCK TABLE perf_metrics IN SHARE ROW EXCLUSIVE MODE;
\else
  BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
\endif
SET LOCAL statement_timeout = '15s';
SET LOCAL TIME ZONE 'Asia/Shanghai';

-- A static psql fragment lets preview, precondition checks and the write use
-- exactly the same manifest. No input from the operator is interpolated in it.
SELECT $plan$(
  SELECT m.model_name, 'default' AS grp, m.hour, p.request_count AS old_req,
         p.success_count AS old_ok, m.req + m.add_req AS new_req,
         m.ok + m.add_ok AS new_ok, m.lat + m.add_ms AS new_latency_ms,
         CASE
           WHEN (p.request_count,p.success_count,p.total_latency_ms) =
                (m.req+m.add_req,m.ok+m.add_ok,m.lat+m.add_ms) THEN 'already applied'
           WHEN (p.id IS NULL AND m.req=0) OR (m.req>0 AND
                (p.request_count,p.success_count,p.total_latency_ms)=(m.req,m.ok,m.lat)) THEN 'ready'
           ELSE 'DRIFT: re-audit'
         END AS state
  FROM (VALUES
    ('ty-seedream-5-5',       timestamptz '2026-11-24 16:00+08',  3, 3, 1168911, 1,0,  73637),
    ('ty-seedream-5-5',       timestamptz '2026-11-25 10:00+08',  0, 0,       0, 2,2, 430635),
    ('ty-seedream-5-5',       timestamptz '2026-11-25 13:00+08', 30,30, 4939919, 3,0, 445047),
    ('ty-seedream-5-5',       timestamptz '2026-11-25 14:00+08', 40,40, 9477505, 3,0, 198936),
    ('ac-image-2-5-sunburst', timestamptz '2026-11-24 08:00+08',  6, 6,  294038, 1,0,  49041),
    ('ac-image-2-5-sunburst', timestamptz '2026-11-24 21:00+08', 24,24, 1146490, 8,0,1019139),
    ('ac-image-2-5-sunburst', timestamptz '2026-11-26 01:00+08',  0, 0,       0, 1,0,   1375)
  ) m(model_name,hour,req,ok,lat,add_req,add_ok,add_ms)
  LEFT JOIN perf_metrics p ON p.model_name=m.model_name AND p."group"='default'
                          AND p.bucket_ts=extract(epoch FROM m.hour)::bigint
) $plan$ AS plan \gset
SELECT * FROM :plan AS preview ORDER BY model_name, hour;
\if :apply
  -- Fail the transaction rather than applying even one bucket on a mismatch.
  SELECT EXISTS(SELECT FROM :plan AS checked WHERE state LIKE 'DRIFT%') AS drift \gset
  \if :drift
    DO $$ BEGIN RAISE EXCEPTION 'Bucket drift: re-audit before applying this manifest'; END $$;
  \endif
  INSERT INTO perf_metrics(model_name,"group",bucket_ts,request_count,success_count,total_latency_ms)
    SELECT model_name,grp,extract(epoch FROM hour)::bigint,new_req,new_ok,new_latency_ms
    FROM :plan AS desired WHERE state='ready'
    ON CONFLICT (model_name,"group",bucket_ts) DO UPDATE SET
      request_count=excluded.request_count, success_count=excluded.success_count,
      total_latency_ms=excluded.total_latency_ms;
  COMMIT;
\else
  ROLLBACK;
\endif
