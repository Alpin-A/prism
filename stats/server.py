"""gRPC server for statistical significance computation."""

from __future__ import annotations

import logging
import os
import signal
import sys
from concurrent import futures

import grpc
import psycopg2
import psycopg2.extras
import psycopg2.pool

import stats_pb2
import stats_pb2_grpc
from engine import ExperimentResult, VariantStats, compute

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger(__name__)

_GRPC_PORT = int(os.environ.get("STATS_GRPC_PORT", "50051"))

_pool: psycopg2.pool.ThreadedConnectionPool | None = None


def _init_pool() -> None:
    global _pool
    _pool = psycopg2.pool.ThreadedConnectionPool(
        minconn=1,
        maxconn=10,
        host=os.environ.get("DB_HOST", "localhost"),
        port=int(os.environ.get("DB_PORT", "5432")),
        user=os.environ["DB_USER"],
        password=os.environ["DB_PASSWORD"],
        dbname=os.environ["DB_NAME"],
    )


def _load_variants(
    conn: psycopg2.extensions.connection,
    experiment_id: str,
    event_type: str,
) -> tuple[list[VariantStats], str | None]:
    with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cur:
        cur.execute(
            """
            SELECT
                v.id                            AS variant_id,
                v.weight,
                COUNT(DISTINCT e.user_id)       AS n_users,
                COALESCE(am.n_events, 0)        AS n_events
            FROM variants v
            LEFT JOIN exposures e
                   ON e.experiment_id = v.experiment_id
                  AND e.variant_id    = v.id
            LEFT JOIN agg_metrics am
                   ON am.experiment_id = v.experiment_id
                  AND am.variant_id    = v.id
                  AND am.event_type    = %s
            WHERE v.experiment_id = %s
            GROUP BY v.id, v.weight
            ORDER BY v.weight ASC, v.id ASC
            """,
            (event_type, experiment_id),
        )
        rows = cur.fetchall()

    if not rows:
        return [], None

    control_id: str = rows[0]["variant_id"]

    variant_stats = [
        VariantStats(
            variant_id=row["variant_id"],
            n_users=int(row["n_users"]),
            n_events=int(row["n_events"]),
        )
        for row in rows
    ]
    return variant_stats, control_id


class StatsServiceServicer(stats_pb2_grpc.StatsServiceServicer):
    def GetExperimentResult(self, request, context):
        experiment_id = request.experiment_id
        event_type = request.event_type or "conversion"

        if not experiment_id:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("experiment_id is required")
            return stats_pb2.ExperimentResultResponse()

        try:
            conn = _pool.getconn()
        except Exception as exc:
            log.error("db pool error: %s", exc)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("database unavailable")
            return stats_pb2.ExperimentResultResponse()

        try:
            variant_stats, control_id = _load_variants(conn, experiment_id, event_type)
        except Exception as exc:
            log.error("query error: %s", exc)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("failed to load metrics")
            return stats_pb2.ExperimentResultResponse()
        finally:
            _pool.putconn(conn)

        if not variant_stats:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("experiment not found or no variants")
            return stats_pb2.ExperimentResultResponse()

        try:
            result: ExperimentResult = compute(variant_stats, control_id)
        except Exception as exc:
            log.error("compute error: %s", exc)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("statistical computation failed")
            return stats_pb2.ExperimentResultResponse()

        proto_variants = [
            stats_pb2.VariantResult(
                variant_id=vr.variant_id,
                n_users=vr.n_users,
                n_events=vr.n_events,
                rate=vr.rate,
                ci_lower=vr.ci_lower,
                ci_upper=vr.ci_upper,
            )
            for vr in result.variants
        ]

        return stats_pb2.ExperimentResultResponse(
            experiment_id=experiment_id,
            event_type=event_type,
            variants=proto_variants,
            p_value=result.p_value,
            is_significant=result.is_significant,
            control_id=result.control_id,
        )


def serve() -> None:
    _init_pool()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    stats_pb2_grpc.add_StatsServiceServicer_to_server(StatsServiceServicer(), server)
    addr = f"[::]:{_GRPC_PORT}"
    server.add_insecure_port(addr)
    server.start()
    log.info("stats-grpc listening on %s", addr)

    def _shutdown(sig, frame):
        log.info("shutting down...")
        server.stop(5).wait()
        sys.exit(0)

    signal.signal(signal.SIGTERM, _shutdown)
    signal.signal(signal.SIGINT, _shutdown)
    server.wait_for_termination()


if __name__ == "__main__":
    serve()