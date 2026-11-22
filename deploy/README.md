# Deploy do lumi pelo Gitea, no servidor de casa, com Tailscale

Pipeline que publica a **web** e a **API** (com Postgres) do lumi no servidor de casa
(ZimaOS) a cada push na `main` do guarda-chuva no Gitea. Mesmo desenho do porvia
(ADR 0048 do porvia); as diferenças estão em [Desvios do porvia](#desvios-do-porvia).

- Workflow: [`.gitea/workflows/deploy.yml`](../.gitea/workflows/deploy.yml)
- Compose de produção: [`deploy/compose.prod.yaml`](compose.prod.yaml) (projeto fixo `lumi-prod`)
- Acompanhantes do Tailscale: [`deploy/tailscale/`](tailscale/) (config do Serve embutida na imagem)

Fora do deploy: TUI, site e Apple (não há runner macOS).

## Como funciona

| Nó na tailnet | Serviço | Destino dentro do nó |
|---|---|---|
| `https://lumi.<tailnet>.ts.net` | web (nginx com o bundle Svelte) | `127.0.0.1:80` |
| `https://lumi-api.<tailnet>.ts.net` | lumi-server (REST + WebSocket) | `127.0.0.1:8080` |

- Cada serviço público roda com `network_mode: service:<acompanhante>`: divide a pilha
  de rede com um contêiner do Tailscale que entra na tailnet como nó próprio, com HTTPS
  na 443 e certificado do Tailscale (`${TS_CERT_DOMAIN}` no Serve). Funnel desligado.
- **Nenhuma porta é publicada no host.** O Postgres só existe na rede interna do compose.
- O servidor escuta só em `127.0.0.1` (dentro do nó `lumi-api`), por isso
  `LUMI_REQUIRE_TLS=true` continua valendo: o TLS termina no Serve.
- A web é construída com `VITE_LUMI_SERVER_URL=https://lumi-api.<tailnet>.ts.net`
  (o WebSocket vira `wss://` automaticamente) e o servidor aceita só a origem
  `LUMI_ALLOWED_ORIGINS=https://lumi.<tailnet>.ts.net` (CORS e upgrade do WS).
- Migrações: one-shot `migrate` (CLI golang-migrate v4.19.1, mesma versão e mesma
  tabela `schema_migrations` da lib do servidor). `LUMI_AUTO_MIGRATE=false`.
- `vaults-init` ajusta o dono do volume das vaults para o usuário nonroot do servidor.
- Antes de cada deploy (se o banco já existe), `backup` grava em um volume um
  `pg_dump` e um `tar.gz` das vaults, guardando os 7 mais recentes de cada.

### Passos do job
1. Confere se as variáveis e os segredos da organização existem (falha com a lista do que falta).
2. Clona o guarda-chuva por `LAN_GIT_URL` (o Gitea se anuncia aos jobs como
   `http://gitea:3000`, que não resolve), com o token do job; valida que o commit está
   no histórico da `main`. Os submódulos `server` e `web-client` vêm por SSH
   (`LAN_GIT_SSH_URL`) com uma chave de deploy somente leitura; a troca
   `https://github.com/ViniZap4/` → `ssh://.../lumi/` vale só dentro do job.
   O submódulo aninhado `server/third_party/y-crdt` vem do GitHub público (ou de
   `YCRDT_GIT_URL`, se definido).
3. `docker compose build --pull` no Docker do próprio servidor (sem registro).
4. Backup, `docker compose up -d --remove-orphans`, espera `/readyz` da API e a
   página da web responderem por dentro dos acompanhantes, mostra `tailscale serve status`.
5. Limpeza da chave SSH e do cabeçalho de autenticação.

## Segredos e variáveis (organização `lumi` no Gitea)

Configuração → Ações → Segredos / Variáveis **da organização**, nunca no repositório.
O Gitea recusa nomes começando com `GITEA_`.

### Segredos
| Nome | Obrigatório | O que é | Como obter |
|---|---|---|---|
| `DEPLOY_SSH_KEY` | sim | Chave privada SSH (ed25519) só leitura para clonar `lumi-server` e `lumi-web` | `ssh-keygen -t ed25519 -N '' -C lumi-deploy -f lumi_deploy`; a pública vai como **chave de deploy somente leitura** em `lumi/lumi-server` e `lumi/lumi-web`; a privada (arquivo inteiro) vira o segredo |
| `TS_OAUTH_SECRET` | sim | Segredo de um cliente OAuth do Tailscale que registra os nós | Admin do Tailscale → Settings → OAuth clients: escopo **Auth Keys (write)**, tag `tag:lumi`. Copie o *client secret* (`tskey-client-...`) |
| `LUMI_DB_PASSWORD` | sim | Senha do Postgres (usuário `lumi`) | `openssl rand -hex 32` (o job exige 16+ caracteres em `[A-Za-z0-9._~-]`, porque a senha entra crua na URL) |
| `LUMI_ADMIN_USERNAME` | não | Admin criado no primeiro boot (só se a tabela de usuários estiver vazia) | Escolha livre; defina junto com o de baixo |
| `LUMI_ADMIN_PASSWORD` | não | Senha desse admin | Gerenciador de senhas. Depois do primeiro deploy, apague os dois e redeploye |

### Variáveis
| Nome | Obrigatória | Exemplo | O que é |
|---|---|---|---|
| `LAN_GIT_URL` | sim | `http://192.168.68.54:3001` | Endereço HTTP do Gitea alcançável de dentro do job (sem `/` no fim). Preferir o IP da rede local; o da tailnet (`http://100.66.72.35:3001`) só funciona se o contêiner do job rotear para a tailnet |
| `LAN_GIT_SSH_URL` | sim | `ssh://git@192.168.68.54:2222` | Base SSH do Gitea na rede local, sem a organização (o job acrescenta `/lumi/`) |
| `GIT_SSH_KNOWN_HOSTS` | sim | `[192.168.68.54]:2222 ssh-ed25519 AAAA...` | Chave do servidor SSH do Gitea, fixada: `ssh-keyscan -p 2222 -t ed25519 192.168.68.54` (confira a impressão digital no Gitea antes) |
| `TAILNET_DOMAIN` | sim | `tail1234.ts.net` | Nome da tailnet (Admin do Tailscale → DNS → Tailnet name) |
| `YCRDT_GIT_URL` | não | `ssh://git@192.168.68.54:2222/lumi/y-crdt.git` | Espelho do y-crdt, se não quiser depender do GitHub no build |
| `LUMI_REGISTRATION` | não | `invite-only` | `invite-only` (padrão) ou `open` |
| `LUMI_PUBLIC_BASE_URL` | não | `https://lumi-api.tail1234.ts.net` | Identidade de federação e base dos links de convite. Padrão: a URL da API |
| `LUMI_TOS_VERSION` / `LUMI_PRIVACY_VERSION` | não | `2026-04-28` | Liga o registro de consentimento no cadastro (os dois juntos) |
| `LUMI_LOG_LEVEL` | não | `info` | `debug`, `info`, `warn`, `error` |

## Antes da primeira execução

1. **Remotes**: o guarda-chuva e os repositórios `lumi-server` e `lumi-web` precisam
   existir na organização `lumi` do Gitea com as mesmas versões referenciadas pelos
   ponteiros dos submódulos (faça push dos submódulos antes do guarda-chuva).
2. **Chave de deploy**: gere e cadastre como descrito em `DEPLOY_SSH_KEY`.
3. **Tailscale**:
   - Na política da tailnet, declare a tag e quem pode aplicá-la, e libere só a 443:
     ```jsonc
     "tagOwners": { "tag:lumi": ["autogroup:admin"] },
     // ACL (ou grants equivalentes):
     { "action": "accept", "src": ["autogroup:member"], "dst": ["tag:lumi:443"] }
     ```
   - HTTPS e MagicDNS ligados (Admin → DNS → HTTPS Certificates).
   - Crie o cliente OAuth com a tag `tag:lumi` (o do porvia só pode aplicar `tag:porvia`).
   - Garanta que não há outro nó chamado `lumi` ou `lumi-api` (senão vira `lumi-1`).
4. **Segredos e variáveis** da organização (tabelas acima).
5. **Runner**: o `zima-runner` (rótulo `ubuntu-latest`) precisa enxergar a organização `lumi`
   (runner de instância, ou registrar outro runner na organização).
6. **Ações habilitadas** no repositório `lumi/lumi` (Configurações → Unidades → Ações).
7. Disparo: push na `main` ou Ações → deploy → *Run workflow* (deixe `ref` vazio).

O primeiro pedido HTTPS a cada nó demora (~20–30 s) pela emissão do certificado.

## Rollback

Ações → **deploy** → *Run workflow* no branch `main`, com `ref` = SHA ou tag do
guarda-chuva que estava bom. O job recusa refs fora do histórico da `main`.
O commit do guarda-chuva fixa as versões de servidor e web que funcionam juntas.

**Banco:** as migrações não são revertidas automaticamente. Se a versão nova aplicou
uma migração incompatível com a antiga, restaure o dump anterior:

```sh
cd <checkout do guarda-chuva no servidor>
export TAILNET_DOMAIN=... LUMI_DB_PASSWORD=... TS_OAUTH_SECRET=...
docker compose -f deploy/compose.prod.yaml stop server
docker compose -f deploy/compose.prod.yaml --profile ops run --rm --entrypoint sh backup -c \
  'ls -1t /backups; pg_restore -h postgres -U lumi -d lumi --clean --if-exists /backups/db-AAAAMMDD-HHMMSS.dump'
# vaults, se necessário (o volume de vaults está read-only no serviço backup; use um contêiner próprio):
docker run --rm -v lumi-prod_vaults:/vaults -v lumi-prod_backups:/backups:ro alpine:3.24.4 \
  sh -c 'tar -C /vaults -xzf /backups/vaults-AAAAMMDD-HHMMSS.tar.gz'
```
Depois rode o deploy da versão desejada.

## Como verificar

- No log do job: `tailscale serve status` dos dois nós e a tabela `docker compose ps`
  com `server`, `web`, `postgres`, `ts-api`, `ts-web` em `running`.
- De um aparelho na tailnet:
  ```sh
  curl -fsS https://lumi-api.<tailnet>.ts.net/healthz   # {"status":"ok"}
  curl -fsS https://lumi-api.<tailnet>.ts.net/readyz    # {"status":"ok"} (banco ok)
  curl -fsSI -H 'Origin: https://lumi.<tailnet>.ts.net' https://lumi-api.<tailnet>.ts.net/healthz \
    | grep -i access-control-allow-origin                 # CORS da web
  ```
  e abra `https://lumi.<tailnet>.ts.net` no navegador, faça login e edite uma nota
  (testa REST + WebSocket).
- No servidor: `docker compose -p lumi-prod logs --tail 100 server`.

## Segurança

- O runner é root com o socket do Docker do host: qualquer workflow da instância controla a
  máquina. Organização e repositórios privados, só o dono com acesso.
- Deploy só da `main`, nunca de pull request; `ref` do rollback validado (regex e
  ancestralidade na `main`); entradas do usuário nunca interpoladas direto no shell.
- Segredos só via ambiente das etapas, sem `set -x` e sem `docker compose config` no job
  (imprimiria os valores). A senha do banco não aparece em linha de comando de contêiner.
- Imagens base fixadas por versão exata; nenhuma ação de terceiros (o checkout é feito com git).
- Os nomes `lumi` e `lumi-api` + o nome da tailnet ficam em logs públicos de certificados
  (Certificate Transparency). Não revelam dados, só que os nós existem.
- O segredo OAuth só consegue registrar nós com `tag:lumi`.

## Desvios do porvia

- **Compose de produção próprio** (`deploy/compose.prod.yaml`) em vez de `include` dos
  compose dos submódulos: o `docker-compose.yml` do guarda-chuva é de desenvolvimento
  (senha padrão `lumi`, portas no host) e o de `server/` publica o Postgres na 5432.
- **Nome do projeto `lumi-prod`**, não `lumi`: o `docker-compose.yml` sem `name:` assume o
  nome do diretório (`lumi`) e os volumes poderiam colidir.
- **Sem entrada de reserva na rede local**: a API escuta só em loopback dentro do nó e
  `LUMI_REQUIRE_TLS=true`; a web não teria HTTPS na LAN. Só tailnet.
- **Segredos via ambiente da etapa**, sem gravar `.env` em disco.
- **Falha explícita** se faltar configuração (o porvia sobe parcialmente e sai com sucesso).
- **Migração e dono das vaults em one-shots** porque a imagem atual do servidor não traz
  `migrations/` nem `/vaults` com dono nonroot (patch proposto ao `lumi-server`).
- **Backup inclui as vaults** (os markdown são a fonte da verdade), não só o banco.
- **Rollback por `ref`** validado contra a `main` (o porvia usa só o disparo manual do branch).
