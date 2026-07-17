# Auditoria do gate de efeitos — Job 29 (Slayers ep. 4)

**Data:** 16/jul/2026 · **Worker:** 1.19.0 · **Modo:** prévia sem upscale (`skipUpscale`) · **Threshold:** 0.25

Primeira auditoria com o datalog `effects_log.json` (timestamps + prob máxima das effect tags do WD14, mesmo abaixo do threshold, para todo cut/recheck).

## Números

| Métrica | Valor |
|---|---|
| Entradas no log | 1.122 (84 cuts, 1.038 rechecks) |
| Disparos do gate | 16 |
| Falsos positivos (nível gate) | 0 — precisão 100% |
| Falsos negativos | 4 clusters (~25s de efeito sem tratamento) |
| Gates vindos de recheck (não de cut) | 13 de 16 |

## Falsos negativos encontrados

1. **00:43.8** — beam da Lina no OP (frame 3-painéis dilui a prob; máx 0.196)
2. **03:59–04:12** — guerra do flashback: energy ball escura deu **0.073** (cegueira em cena azul dessaturada); aura subiu 0.135→0.283 e só disparou às 04:12.8
3. **04:48–05:08** — revival + luta demônio×dragão + selo de gelo (pico 0.211, resto ≤0.07)
4. **14:07.7** — cauda do blast: corte no meio do efeito criou shot novo com 0.118; rechecks seguintes já eram pós-blast (≤0.033)

## Falso positivo em nível de MÁSCARA (reportado pelo usuário)

`evidencia-cabelo-gourry-op.png`: cabelo loiro do Gourry com bloom nos créditos do OP (~01:00).
Mecânica confirmada no log: shot 9 gated às 00:54.3 (relâmpago, 0.42) e o próximo corte só veio às
01:07.9 — o OP usa **dissolves**, invisíveis pro detector de corte (diff de luma), então o plano do
relâmpago e o do grupo viraram um shot só e o comp ficou ligado por 14s. Dentro da janela aprovada,
cabelo brilhante+saturado é alvo legítimo da máscara fotométrica (classe "cabelo da Lina" conhecida).

## Padrões sistêmicos

1. **Quem segura o gate é o recheck**: o frame do cut cai em transição/whiteout e dá prob baixa;
   o recheck de 24 frames pega 1–4s depois. Custo: início de plano sem comp.
2. **`comp_active` nunca desliga dentro do plano** — recheck só existe para LIGAR. Causa direta do caso do cabelo.
3. **Threshold 0.25 alto para vídeo**: todos os negativos ≥0.15 eram efeito real; falsos candidatos
   só aparecem ≤0.14 (dragões cômicos 0.14, título "Slayers" 0.13, bola de cristal 0.13).
   `WORKER_TAGGER_THRESHOLD=0.15` ganharia os FNs 1–3 sem FP novo neste episódio.

## Limitações conhecidas (nem threshold 0.15 salva)

- Energy ball escura em cena dessaturada (0.073) e luta escura de flashback (0.05–0.07) —
  limite do vocabulário WD14 (10 tags, sem "beam") em conteúdo escuro. Precisa de exemplos
  rotulados / modelo próprio de segmentação.

## Arquivos

- `effects_log.json` — datalog bruto do job 29
- `gated_grid.jpg` / `nearmiss_grid.jpg` — grades rotuladas (timestamp + prob + tags) dos 16 disparos e 19 quase-disparos
- `sheets/` — 12 folhas de contato do episódio inteiro (1 frame a cada 4s, grade 6×5; thumb k = k×4s)
- `frames-gated/`, `frames-nearmiss/` — frames individuais extraídos
- `evidencia-cabelo-gourry-op.png` — screenshot do Compare (usuário) com o FP de máscara
- `montage.py` — gerador das grades (usa o cv2 do venv do worker; ffmpeg local não tem drawtext)

> Imagens desta pasta ficam FORA do git (repo público — frames de anime são material com copyright).
> Só este README e o `effects_log.json` são versionados.
