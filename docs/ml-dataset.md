# Dataset de segmentação de efeitos

Infraestrutura da fase P2 do roadmap de efeitos (ver artefato da estratégia). **Totalmente
isolada do catálogo**: banco SQLite próprio + prefixo próprio no R2 — nenhuma FK ou tabela
compartilhada com o catálogo.

## Armazenamento

| O quê | Onde |
|---|---|
| Metadados + vereditos | `ml_dataset.db` (SQLite separado; `ML_DATASET_PATH`, default ao lado do `DATABASE_PATH`) |
| Frames | `ml-dataset/frames/{id}.jpg` no storage (R2 em prod) |
| Máscaras | `ml-dataset/masks/{id}.png` no storage |

## Contrato da máscara

PNG **com canal alpha**: opaco = pixel de efeito, transparente = fundo. A UI de triagem
renderiza via CSS `mask-image`, que usa o alpha — máscara branco-sobre-preto sem alpha
**não funciona**. Mesma resolução do frame.

## API (rotas autenticadas)

### Ingestão — `POST /api/dataset/samples` (multipart)

| Campo | Tipo | Obrigatório |
|---|---|---|
| `class` | `fire` \| `lightning` \| `energy` \| `aura` \| `dark_magic` \| `beam` \| `none` | sim |
| `frame` | arquivo jpg/png | sim |
| `mask` | arquivo png com alpha | sim |
| `source` | texto (default `teacher`) | não |
| `animeTitle`, `episode` | texto livre (sem vínculo com o catálogo) | não |
| `timestampS`, `teacherProb` | float | não |

```bash
curl -b cookies.txt -X POST https://<host>/api/dataset/samples \
  -F class=fire -F animeTitle=Slayers -F episode=S1E04 \
  -F timestampS=54.3 -F teacherProb=0.42 \
  -F frame=@frame.jpg -F mask=@mask.png
```

### Demais rotas

- `GET /api/dataset/samples/queue?limit=50` — amostras pendentes (mais antigas primeiro) com URLs de frame/máscara
- `POST /api/dataset/samples/{id}/verdict` — body `{"verdict":"approved"|"rejected"|"needs_edit"}`
- `GET /api/dataset/stats` — totais por status + quebra por classe/status

## Triagem

Página **Dataset** na UI (`#/dataset`): frame com a máscara tingida pela cor da classe por cima.
Atalhos: **A** aprova · **R** rejeita · **E** marca para ajuste manual · **M** liga/desliga a máscara.
Amostras `needs_edit` são a fila do editor de polígonos (Label Studio/CVAT) — exportação fica
para quando o pipeline do professor existir (P1).

## Fontes de frames planejadas (P2)

1. WD14 com threshold baixo (0.05) — filtro grosso de volume
2. Amostra aleatória (~5–10%) dos frames rejeitados pelo WD14 — classes cegas + negativos (`class=none`, máscara vazia)
3. Cenas apontadas manualmente
