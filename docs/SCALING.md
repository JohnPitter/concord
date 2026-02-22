# Concord — Plano de Escala para 100k Usuarios

## Visao Geral

Este documento descreve as mudancas necessarias para escalar o Concord de 0 a 100.000 usuarios registrados (~15k concorrentes). Organizado em 4 fases incrementais.

---

## Premissas

| Metrica | Valor |
|---------|-------|
| Usuarios registrados | 100.000 |
| Concorrentes (15%) | 15.000 |
| Em canais de texto ativos | 5.000 |
| Em canais de voz simultaneos | 2.000 |
| Mensagens/segundo (pico) | 500 |
| Voice channels ativos | 200 |

---

## Fase 1 — Validacao (0-5k usuarios)

**Objetivo:** Single VPS rodando tudo. Preparar observabilidade e config para escala.

**VPS:** 4 vCPU, 16GB RAM, 200GB NVMe SSD (~$40/mes Hetzner)

### Mudancas no codigo (esta fase):

1. **Prometheus metrics wired** — Conectar metricas HTTP ja definidas ao middleware
2. **Rate limiting configuravel** — Usar `config.Security.RateLimitAPI` em vez de hardcoded 100 RPS
3. **CSP corrigido** — Permitir WebSocket connections (`connect-src`)
4. **Rate limit headers** — `X-RateLimit-Remaining`, `X-RateLimit-Reset`
5. **Docker compose producao** — Nginx reverse proxy, resource limits, PostgreSQL tuning
6. **Graceful shutdown melhorado** — Drain de requests in-flight, shutdown ordenado
7. **Gzip compression** — Responses comprimidas no Nginx

### Distribuicao de memoria (16GB):
- PostgreSQL: 4GB (shared_buffers=1GB, effective_cache_size=3GB)
- Redis: 2GB
- App Server (Go): 4GB
- Nginx + OS: 2GB
- Reserva: 4GB

---

## Fase 2 — Crescimento (5k-20k usuarios)

**VPS:** 8 vCPU, 32GB RAM, 400GB NVMe SSD (~$100/mes)

### Mudancas necessarias:

1. **SFU para voice** — Integrar Pion ion-sfu ou LiveKit para voice channels com >10 users
2. **Redis Pub/Sub para signaling** — Distribuir eventos entre instancias futuras
3. **Rate limiting distribuido** — Migrar de in-memory para Redis sliding window (`INCR` + `EXPIRE`)
4. **Session storage no Redis** — Preparar para multi-instancia
5. **Connection pool tuning** — Bulkheads para separar queries de leitura/escrita
6. **Background health checks** — Nao depender apenas de `/health` requests
7. **Request tracing** — OpenTelemetry para distributed tracing

### Distribuicao de memoria (32GB):
- PostgreSQL: 8GB (shared_buffers=2GB, effective_cache_size=6GB)
- Redis: 4GB
- App Server (Go): 6GB
- SFU (Pion): 8GB
- Nginx + OS + Monitoring: 6GB

---

## Fase 3 — Escala (20k-50k usuarios)

**Infra:** 2 app servers + 1 DB + 1 Redis + 1 SFU (~$300/mes)

### Mudancas necessarias:

1. **Load balancer** — Nginx com sticky sessions para WebSocket
2. **PostgreSQL read replicas** — 1 primary (writes) + 1 replica (reads)
3. **Message queue** — NATS ou Redis Streams para delivery assincrono
4. **Cache invalidation broadcast** — Redis Pub/Sub em operacoes de escrita
5. **Database partitioning** — Particionar `messages` por `created_at` (range mensal)
6. **TURN server dedicado** — Para NATs simetricos que falham hole-punch
7. **Auto-scaling config** — Docker Swarm ou Kubernetes basics

### Topologia:
```
                    ┌─── App Server 1 ───┐
Internet → Nginx →──┤                    ├── PostgreSQL Primary
                    └─── App Server 2 ───┘        │
                              │              PG Replica
                           Redis ←── SFU
```

---

## Fase 4 — 100k (50k-100k usuarios)

**Infra:** 3 app + DB primary/replica + Redis cluster + SFU cluster (~$600/mes)

### Mudancas necessarias:

1. **Horizontal auto-scaling** — Kubernetes com HPA (Horizontal Pod Autoscaler)
2. **PostgreSQL read replicas x2** — Distribuir queries de leitura
3. **Redis Cluster** — 3 nodes para HA e throughput
4. **SFU cluster** — 2+ instancias com load balancing por channel
5. **CDN** — Arquivos estaticos e avatares via CDN (Cloudflare R2 ou S3)
6. **Message retention policy** — Arquivar mensagens >6 meses
7. **Full-text search dedicado** — Elasticsearch ou Meilisearch para busca de mensagens
8. **Service mesh** — Consul ou Istio para service discovery e circuit breaking

### Topologia:
```
                         ┌─── App 1 ───┐
Internet → LB (Nginx) →─┤─── App 2 ───┤── PG Primary ─── PG Replica 1
                         └─── App 3 ───┘       │          PG Replica 2
                               │
                          Redis Cluster    SFU 1 ─── SFU 2
                           (3 nodes)
                               │
                         Prometheus + Grafana
```

---

## VPS Recomendada por Fase

| Fase | Users | Spec | Custo/mes |
|------|-------|------|-----------|
| 1 | 0-5k | 4 vCPU, 16GB RAM, 200GB NVMe | ~$40 |
| 2 | 5k-20k | 8 vCPU, 32GB RAM, 400GB NVMe | ~$100 |
| 3 | 20k-50k | Multi-server (5 VPS) | ~$300 |
| 4 | 50k-100k | Cluster (8+ VPS ou K8s) | ~$600 |

**Providers recomendados:** Hetzner (melhor custo-beneficio), OVH, Vultr, DigitalOcean.

---

## Prioridade de Implementacao

```
[CRITICO] SFU para voice (Fase 2) — sem isso, voice com >10 users falha
[CRITICO] Redis Pub/Sub signaling (Fase 2) — permite multi-instancia
[ALTO]    Rate limiting distribuido (Fase 2) — protecao contra abuse
[ALTO]    DB read replicas (Fase 3) — performance de leitura
[MEDIO]   Message queue (Fase 3) — delivery assincrono
[MEDIO]   Metrics wiring (Fase 1) — observabilidade real  ← DONE
[BAIXO]   Particionamento messages (Fase 3) — so com >50M mensagens
[BAIXO]   TURN server dedicado (Fase 3) — para NATs simetricos
```

---

## Estado Atual (Pre-Fase 1)

### O que ja funciona bem:
- Clean Architecture com separacao clara de camadas
- Auth seguro (GitHub Device Flow + JWT + refresh token encriptado)
- P2P excelente (libp2p com QUIC, hole-punch, relay, mDNS)
- SQLite client exemplar (WAL, mmap, FK, pragmas otimizados)
- Health checks concorrentes com cache
- Prometheus metrics definidos (67 metricas em 10 subsistemas)
- Docker Compose funcional com PG 17 + Redis 7

### O que precisa melhorar (Fase 1):
- [x] Metrics wired no middleware HTTP
- [x] Rate limiting configuravel (nao hardcoded)
- [x] CSP corrigido para WebSocket
- [x] Docker compose com Nginx e resource limits
- [x] Graceful shutdown com drain de requests
- [x] Rate limit headers na response
