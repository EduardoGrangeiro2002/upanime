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

## Fontes de frames (implementadas no professor)

1. WD14 com threshold baixo (0.05) — filtro grosso de volume
2. Amostra aleatória (~7,5%) dos frames rejeitados pelo WD14 — classes cegas + negativos (`class=none`, máscara vazia)
3. Timestamps manuais (`--timestamps "54.3,251.7"`)

## O professor (`ml/` — pacote `upanime-teacher`)

Pipeline composto que gera as máscaras candidatas e posta nesta API:
**GroundingDINO** (prompts de efeito, score ≥ 0.2, rótulo não-vazio) + **SAM** por caixa,
∪ **picos da máscara fotométrica HSV → SAM por ponto** (cobre orbes/beams que o DINO não vê,
classe `energy`, `source=hsv`), ∪ negativos amostrados. Uma amostra por classe por frame.

### Autenticação de máquina

As rotas `/api/dataset/*` aceitam sessão OU `Authorization: Bearer $DATASET_INGEST_TOKEN`
(env do servidor; comparação em tempo constante; bearer desabilitado se a env não existir).
Gere um token longo (`openssl rand -hex 32`) e adicione ao `PROD_ENV`.

### Rodando

```bash
cd ml
TEACHER_API_BASE=https://<host> TEACHER_API_TOKEN=<token> \
  .venv/bin/python -m upanime_teacher.cli <episodio.mp4|url-presignada> \
  --anime "Slayers" --episode "S1E04"
```

Env vars em `docker.env.example` (`TEACHER_DEVICE=auto` usa CUDA se existir; senão CPU).

### RunPod (pod avulso, não serverless)

Imagem pública `alkindar/upanime-teacher` (deploy: `ml/deploy.sh --minor`, pesos DINO+SAM+WD14
baked). Num pod GPU qualquer (4090/A40):

```bash
docker run --gpus all --env-file docker.env \
  alkindar/upanime-teacher:latest <url-presignada-do-episodio> \
  --anime "Slayers" --episode "S1E04"
```

### Limitação conhecida

Com `STORAGE_TYPE=local` (dev), `frameUrl`/`maskUrl` da fila são caminhos de filesystem —
a triagem no navegador só renderiza com storage R2 (prod), onde as URLs são presignadas.
