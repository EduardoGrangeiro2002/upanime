# Deploy em VPS (Hostinger) com CI/CD

Depois de configurar uma vez, o deploy é automático: **`git push` na `main`** → o GitHub Actions builda a imagem, publica no Docker Hub (imagem **pública**) e reinicia o app no VPS por SSH. Zero comando manual.

## Como o app é servido

O backend Go serve **tanto a API quanto o front** (o build do React é embutido na imagem). Não há dois serviços nem CORS: o navegador fala só com um domínio, e o Traefik (proxy reverso já rodando no VPS) aplica HTTPS na borda e roteia para o app via labels no compose.

```
Navegador → https://anime.eduardograngeiro.com.br → Traefik (HTTPS) → app:7891 (Go serve front + API)
                                                          │
                                                          ├── RunPod (upscale, via polling)
                                                          └── Cloudflare R2 (vídeos)
```

O worker de upscale roda no RunPod (imagem separada, `worker/deploy.sh`). O VPS **não precisa ser acessível de fora** pelo RunPod — o app acompanha os jobs por polling e o worker baixa os vídeos direto do R2.

## Configuração inicial (uma vez)

### 1. Preparar o VPS

Acesse por SSH e instale o Docker (a imagem é pública, não precisa de `docker login`):

```bash
ssh root@SEU_IP_VPS
curl -fsSL https://get.docker.com | sh
```

Crie a pasta do app e coloque nela o `docker-compose.prod.yml` (copie do repo, ou `git clone`):

```bash
mkdir -p /opt/upanime && cd /opt/upanime
```

Crie o `.env` de produção:

```bash
nano .env
```

| Variável | Valor |
|---|---|
| `AUTH_SECRET` | `openssl rand -base64 32` |
| `STORAGE_TYPE` | `r2` |
| `R2_*` | Credenciais do bucket Cloudflare R2 |
| `RUNPOD_ENDPOINT_ID` / `RUNPOD_API_KEY` | Endpoint serverless do worker |
| `SMTP_*` | Servidor de email (MFA e convites) |
| `OPENROUTER_API_KEY` | Opcional (classificação de gêneros) |
| `UPANIME_IMAGE` | `seu-usuario/upanime:latest` |

> `AUTH_COOKIE_SECURE=1` já vem forçado no compose (HTTPS).

### 2. Apontar o domínio

No painel do domínio, crie um registro **A** para o IP do VPS. O domínio já está definido nas labels do Traefik no `docker-compose.prod.yml` (`anime.eduardograngeiro.com.br`); o Traefik emite o certificado HTTPS sozinho (Let's Encrypt) na primeira requisição. Requer o Traefik já rodando no VPS com o entrypoint `websecure` e o certresolver `letsencrypt`.

### 3. Primeira subida (manual, só desta vez)

```bash
cd /opt/upanime
docker compose -f docker-compose.prod.yml up -d
```

> A imagem precisa já existir no Docker Hub. Se ainda não publicou, configure o CI/CD (passo 5) e dispare o workflow uma vez em **Actions → Deploy app → Run workflow**.

### 4. Criar seu usuário admin

```bash
docker compose -f docker-compose.prod.yml exec app ./upanime-api create-user voce@exemplo.com
```

Acesse o domínio, entre com a senha temporária, troque a senha e confirme o código de MFA (por email).

### 5. Configurar o CI/CD (segredos do GitHub)

Em **Settings → Secrets and variables → Actions** do repositório, adicione:

| Secret | O que é |
|---|---|
| `DOCKERHUB_USERNAME` | Seu usuário do Docker Hub |
| `DOCKERHUB_TOKEN` | Access token do Docker Hub (Account Settings → Security) |
| `VPS_HOST` | IP do VPS |
| `VPS_USER` | Usuário SSH (ex: `root`) |
| `VPS_SSH_KEY` | Chave **privada** SSH com acesso ao VPS |
| `VPS_PORT` | Porta SSH (`22` por padrão) |
| `VPS_APP_DIR` | Caminho no VPS (ex: `/opt/upanime`) |

> Gere um par de chaves dedicado pro deploy (`ssh-keygen -t ed25519 -f deploy_key`), adicione a **pública** em `~/.ssh/authorized_keys` no VPS e cole a **privada** em `VPS_SSH_KEY`.

Marque o repositório `seu-usuario/upanime` como **Public** no Docker Hub (a imagem não contém segredos — o `.env` é montado em runtime, nunca copiado pra imagem).

## Uso no dia a dia

```bash
git push
```

É isso. O workflow `CI` roda os testes (Go + client); passando, o `Deploy app` builda, publica e reinicia o VPS. Acompanhe em **Actions** no GitHub.

Para forçar um deploy sem alterar código: aba **Actions → Deploy app → Run workflow**.

## Comandos úteis (no VPS)

```bash
docker compose -f docker-compose.prod.yml logs -f app   # logs ao vivo
docker compose -f docker-compose.prod.yml restart        # reiniciar
docker compose -f docker-compose.prod.yml down           # parar tudo
```

## Notas

- **Dados persistentes**: banco, downloads e Redis ficam em volumes Docker nomeados (`upanime-data`, `redis-data`), sobrevivem a atualizações de imagem. Faça backup do volume `upanime-data`.
- **Worker de upscale**: é publicado separadamente para o RunPod via `worker/deploy.sh` — não faz parte deste deploy.
- **Recursos**: o scraper usa Chromium (Playwright), que consome RAM — VPS com 2 GB+ recomendado. O Chromium já vem na imagem.
