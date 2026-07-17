# P1 — Go/no-go do professor zero-shot

**Data:** 17/jul/2026 · **Modelos:** GroundingDINO base + SAM vit-base (HuggingFace, CPU local)
· **Conjunto:** 12 frames do job 29 (7 acertos do gate WD14, 3 pontos cegos, 2 negativos)
· **Harness:** `ml/teacher_probe.py` (env `PROBE_PROMPT` / `PROBE_THRESHOLD` / `PROBE_SUFFIX`)

## Veredito: GO — com pipeline composto, não GroundingDINO sozinho

### O que o professor acerta (e muito)

| Caso | Resultado |
|---|---|
| Brilho de magia escura 04:51 (WD14 deu 0.073 — cego) | **Cravado**: "magic aura" 0.39, máscara justa no blob (8,7%) |
| Fogo da vila 01:41 | **Perfeito**: fire 0.63, máscaras exatas nas duas chamas (1,7%) |
| Fogo da taverna 01:35 | Pego na v2 (threshold 0.15), máscara na chama |
| Flare da Lina 18:18 | Núcleo do flare bem mascarado; uma máscara extra vazada |
| Cauda do blast 14:07 (WD14 0.118) | Detectado, máscara vaza pra floresta (65%) |
| Aura do demônio 04:12 | Detecta, mas máscara cobre o frame inteiro (99%) |
| Negativos (diálogo, floresta) | **Limpos** com filtro: só ruído de rótulo vazio < 0.2 |

### O que o professor NÃO vê (gap real)

Energy ball na mão da Lina, orbe do Zolf e beam do OP: **zero detecção mesmo a 0.15 com
prompts "fireball / glowing orb / light beam"**. GroundingDINO é cego pra orbes/beams
estilizados de anime — e são cenas de poder comuns.

### Por que ainda é GO

1. **Filtro simples mantém precisão**: exigir rótulo não-vazio + score ≥ 0.2 elimina o ruído
   dos negativos sem perder os acertos.
2. **O gap dos orbes tem complemento em casa**: a máscara fotométrica HSV acende exatamente
   em orbe/beam (brilhante+saturado). Pico da máscara HSV → point prompt no SAM → máscara
   candidata da classe que o DINO não vê. As três fontes se compõem.
3. **WD14 continua como rede de segurança**: frame que o WD14 marca (energy_ball 0.89!) e o
   DINO não detecta → vai direto pra fila "precisa de ajuste" da triagem, com um clique
   assistido por SAM em vez de desenho manual.
4. Cobertura estimada de auto-rotulagem: **~60–70% dos frames de efeito** com máscara
   utilizável (aprovável ou quase) — muito acima do custo de desenhar tudo à mão.

### Custo medido (CPU M-series)

~2–2,5s GroundingDINO + ~1,5–3s SAM por frame. Fábrica offline de milhares de frames = horas
locais ou minutos em GPU. Viável.

### Decisão para o P2

Professor = **GroundingDINO(≥0.2, rótulo não-vazio) + SAM box-prompt**, complementado por
**HSV-pico → SAM point-prompt** (orbes/beams) e WD14 como roteador de fila. Humano tria na
interface já construída; "precisa de ajuste" vira clique assistido, não desenho.

## Arquivos

- `montage_v1.jpg` — passada 1 (threshold 0.25, prompt base)
- `montage_v2.jpg` — passada 2 (threshold 0.15, prompt expandido)
- `report.json` / `report_v2.json` — scores, áreas de máscara e latências por frame
