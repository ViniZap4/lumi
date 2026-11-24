# Livros: vários servidores, várias contas, estado de sincronização visível

- Status: proposto (wave 2)
- Data: 22/11/2026
- Base: porvia ADRs 0033, 0034 e 0035 e `docs/sincronizacao.md`; lumi `SPEC.md` (v2) e `SPEC-V3.md`; `wave1/CRDT-CLIENT-PROTOCOL.md`. Este documento não contradiz nenhum deles. Onde houver dúvida, vale o protocolo CRDT.
- Pilares: segurança/LGPD → performance → DX → escala → UX → UI → QA.

## Contexto

Hoje o lumi tem três clientes com três modelos diferentes:

- **Web:** um único `API_URL`, definido no build (`api.ts`), e uma única sessão em `localStorage['lumi.v2.session']`. A sessão é cifrada com AES-GCM, mas a chave sai de constantes do código, então isso é só ofuscação.
- **Apple:** uma única `currentSession` (`lumi.auth.session.v1`). O token fica no Keychain com `account = serverURL`, e o `RemoteVaultsStore` usa um cliente só. Existem dois registros de vaults: o `VaultRecord` no SwiftData e o `vaults.yaml`.
- **TUI:** é o único cliente que já fala com vários servidores. Usa `accounts.yaml` (uma conta por servidor, 0600) e `vaults.yaml` (`server` e `account` em cada linha). Esse `vaults.yaml` não guarda o id do vault no servidor, porque `lumi vault link` gera um id local novo. Restam ainda tokens soltos em `<vault>/.lumi/config.yaml` e um cliente `sync/` que fala com o `/ws` da v1.

O servidor só tem o WebSocket Yjs por nota. Não existe feed por vault nem por conta: nenhum cliente fica sabendo que uma nota foi criada, que alguém o removeu de um vault ou que um vault foi apagado. Por isso nenhum cliente consegue mostrar se um livro está atualizado. A federação piora o quadro: o mesmo `vault_id` existe no servidor de origem e nos seguidores, e um cliente logado nos dois mostraria o livro duas vezes.

## Decisão (resumo)

1. **Livro = vault.** Nada muda no código, na API nem no banco: tudo continua `vault`. "Livro" é só o nome que aparece na interface, como no porvia (ADR 0033).
2. **Um registro de contas e um registro de livros em cada cliente**, com o mesmo modelo de dados nos três. No Apple e no TUI o registro de livros é o mesmo arquivo (`~/.config/lumi/books.yaml`), e os dois produzem os mesmos bytes. Os tokens ficam no armazenamento de cada plataforma.
3. **Um livro ligado aponta para um único par (servidor, conta).** Uma cópia federada do mesmo `vault_id` em outro servidor vira uma *alternativa* desse livro, nunca um segundo livro.
4. **Feed por conta via SSE:** `GET /api/events`, com autenticação por cabeçalho. O stream é lido com `fetch`/`URLSession`/`net/http`, nunca com `EventSource`, e o token nunca vai na URL.
5. **`seq` monotônico por vault em cada servidor**, guardado em `vault_heads.seq` e em `notes.change_seq`, com tombstones para notas apagadas. Junto vem `GET /api/vaults/:vault/changes?after=N` para buscar só o que mudou.
6. **Ao conectar, o cliente reconcilia tudo.** O evento `hello` traz o `seq` de cada vault. Nenhum cliente pode depender de não perder um evento.
7. **Estado de sincronização por livro**, com sete estados derivados de feed, sessão do editor e saúde do REST. Aparece num indicador na barra e num painel de detalhes.
8. **Seletor de livros no topo da barra lateral** (web e Apple) e num picker no TUI. Atalhos: ⌃⌘1–9 no Apple, `g1`–`g9` no web e no TUI.

---

## 1. Glossário

| Termo (UI) | Código | Significado |
|---|---|---|
| **Livro** | `Book` (cliente), `vault` (servidor) | Espaço separado de notas. No servidor é exatamente um vault. No cliente é uma linha do registro de livros. |
| **Servidor** | `server_url` | Uma instalação do `lumi-server`, identificada pela URL base normalizada (§2.1) e pela chave pública Ed25519 de `GET /api/info`. |
| **Conta** | `Account` | Um usuário num servidor: o par (`server_url`, `username`). Pode haver mais de uma conta no mesmo servidor. Não existe identidade central. |
| **Livro local** | `kind: local` | Existe só como pasta neste aparelho (Apple, TUI). Não tem servidor nem feed. O estado é sempre "Não ligado". |
| **Livro ligado** | `kind: bound` | Preso a um único (`server_url`, `account`, `vault_id`). Pode ou não ter uma pasta espelho local (`path`): o TUI sempre tem, Apple e web normalmente não. |
| **Livro ativo** | `active_book_id` | O livro aberto agora. Fica salvo por aparelho. Telas, busca, editor e atalhos olham só para ele. |
| **Alternativa** | `alternates` | Outro servidor (federado) onde o mesmo `vault_id` também está acessível para uma conta deste cliente. |
| **Cabeça (head)** | `seq` | O maior `seq` do vault *naquele servidor*. Não se compara entre servidores. |
| **Cursor** | `cursor_seq` | O maior `seq` que o cliente já aplicou ao cache daquele livro. |

## 2. Modelo no cliente

### 2.1 Normalização de URL

`server_url` = esquema e host em minúsculas, sem porta padrão, sem barra final e sem query. Um caminho base é aceito (`https://x.com/lumi`). `http://` só vale para `localhost`, IPs privados e `*.local`, e sempre com um aviso fixo que o usuário pode ocultar por servidor (porvia 0034 §10). Duas URLs que respondem a mesma `public_key` em `GET /api/info` são o mesmo servidor. Nesse caso o cliente avisa e usa a URL que já estava cadastrada.

### 2.2 Registro de contas

Campos comuns: `server_url`, `username`, `user_id`, `display_name`, `added_at` e `expires_at`. A chave é (`server_url`, `username`). O segredo é só o token de sessão. A senha nunca é guardada.

| Plataforma | Metadados | Token |
|---|---|---|
| **Apple** | `UserDefaults['lumi.accounts.v1']` (JSON) | Keychain. `service = com.vinizap.lumi.auth`, `account = "<server_url>#<username>"`, `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`: o feed continua com a tela bloqueada, mas o token não vai para backup nem para o iCloud Keychain. |
| **TUI** | `~/.config/lumi/accounts.yaml`, agora com `version: 2` | Na mesma linha, `token:`. O arquivo é 0600, o diretório 0700 e a escrita é atômica, como já é hoje. |

No macOS, o registro e os tokens seguem o §8.
| **Web** | IndexedDB `lumi`, store `accounts` | Cifrado com AES-GCM por uma `CryptoKey` **não extraível**, gerada uma vez por navegador e guardada na store `keys` (`device-key`). |

**Ameaças no web, sem rodeios.** A chave não extraível impede que o token apareça em claro numa cópia do `localStorage`, num dump de devtools, numa extensão que só lê o storage ou num log. Ela **não protege contra XSS**: um script na página pode chamar `decrypt`. Também não protege contra uma cópia do perfil inteiro aberta em outro navegador. A proteção real vem de outro lugar: CSP estrita (`script-src 'self'`), nenhuma dependência carregada em runtime, TTL deslizante do servidor, revogação e o botão "Sair deste aparelho". O formulário de login tem "Lembrar neste aparelho", ligado por padrão. Desligado, o token fica só em memória. A chave fixa de `crypto.ts` sai do código.

O TUI passa a aceitar várias contas no mesmo servidor. `Upsert` e `Find` usam (`server`, `username`). Um arquivo sem `version` é lido como v1 (uma linha por servidor, sem perda) e regravado como v2 no próximo save.

### 2.3 Registro de livros

```yaml
# ~/.config/lumi/books.yaml — lumi book registry. Format: SPEC-LIVROS §2.3.
version: 1
active: 0192f7a1-…            # somente TUI; Apple/web guardam o ativo à parte
books:
  - id: 0192f7a1-…            # UUID local, estável; = vault_id quando o livro nasceu de um clone
    name: "Trabalho"
    kind: bound               # local | bound
    server_url: "https://lumi.work.com"   # null quando local
    account: "alice"          # username; null quando local
    vault_id: 8c5a1d9f-…      # id no servidor; null quando local
    path: "~/notes/work"      # null se não houver pasta (Apple/web remoto)
    sort: 10                  # ordem do usuário; passos de 10
    added_at: 2026-11-22T10:00:00Z
    last_opened_at: 2026-11-22T11:00:00Z
```

- **Bytes iguais.** Ordem das chaves fixa (a acima), livros ordenados por `sort` e depois por `id`, timestamps em UTC com segundos e `Z`, `null` literal, strings com aspas duplas, LF, uma linha final. Apple e TUI testam contra o mesmo fixture `fixtures/books.v1.yaml`, que fica no monorepo e é copiado para cada submódulo (§6).
- **Arquivo novo, não `vaults.yaml` v2.** Os leitores atuais de `vaults.yaml` (TUI `parseVaultsYAML` e Apple `readFromDisk`) jogam fora campos que não conhecem e regravam o arquivo. Um cliente antigo apagaria `vault_id`, `kind` e `sort` em silêncio. Com um arquivo novo, cliente antigo e novo convivem sem se estragar.
- **No web** o registro tem o mesmo formato em JSON, na store `books` do IndexedDB, sem `path`, e só aceita `kind: bound`. A web não tem livro local nesta versão.
- **No Apple** os bookmarks de segurança das pastas (`bookmarkData`) ficam num side-store SwiftData `BookBookmark {book_id, bookmark}`. `VaultRecord` deixa de ser a lista canônica. No iOS e no visionOS, `books.yaml` fica no container do app, com o mesmo formato.
- **Nome:** num livro ligado, `name` é um cache do nome no servidor, atualizado por `vault.updated`. Renomear é `PATCH /api/vaults/:vault` e exige `vault.manage`. Não existe apelido local.
- **Livro ativo:** Apple em `UserDefaults['lumi.activeBookID']`, web em `localStorage['lumi.activeBook']` (não é segredo), TUI no campo `active` do arquivo. Se faltar ou apontar para um livro removido, abre o livro com o menor `sort` (porvia 0033 §2).

### 2.4 Regra de deduplicação (cópias federadas)

1. O cliente junta os `GET /api/vaults` de todas as contas. O registro garante **no máximo um livro ligado por `vault_id`**.
2. Um `vault_id` que ainda não está no registro e aparece por uma conta só vira livro sem perguntar nada.
3. Quando o mesmo `vault_id` aparece por duas ou mais contas (servidores federados, ou duas contas no mesmo servidor), o padrão é ligar ao servidor com `federation.role = "home"`. Se a origem não estiver entre as contas, liga à conta mais antiga. A interface mostra uma faixa uma vez: *"Este livro também está em notes.friend.org. Ele sincroniza por lumi.work.com."* com os botões **[Manter]** e **[Trocar servidor…]**.
4. As outras origens ficam em memória como `alternates` e aparecem no painel de detalhes. Elas não abrem feed próprio para esse livro: o feed da conta continua aberto, mas os eventos daquele `vault_id` que chegam por uma conta alternativa são ignorados.
5. **Trocar servidor** muda `server_url`, `account` e o cursor (vai para 0, porque `seq` é por servidor) e apaga o cache de lista. Os Y.Docs que o cliente persistiu localmente **são mantidos**: a federação replica o mesmo log Yjs a partir de um único semeador (CRDT §10 S5). Reconectar a outro servidor é um handshake normal (CRDT §5), não uma semeadura.

### 2.5 Migração

| De | Para | Como |
|---|---|---|
| Web `lumi.v2.session` | conta + livros | No boot, decifra com a chave antiga e cria a conta (`server_url` = `API_URL`). Cifra de novo com a `device-key` e apaga a chave antiga. O primeiro `GET /api/vaults` cria os livros, ordenados por nome. |
| Apple `lumi.auth.session.v1` + Keychain(`account = serverURL`) | `lumi.accounts.v1` + Keychain(`server#username`) | Lê, regrava com a chave nova e apaga o item antigo. Se falhar, a conta vai para "Precisa entrar". |
| Apple `VaultRecord` | `books.yaml` + `BookBookmark` | Pasta local → `kind: local`. `serverEndpoint` → `bound`, com `vault_id` resolvido pelo slug no primeiro `GET /api/vaults`. |
| `vaults.yaml` (Apple/TUI) | `books.yaml` | Importação única, só leitura, quando `books.yaml` ainda não existe. O `vaults.yaml` fica onde está, para clientes antigos. Linha com `server` → `bound`, com `vault_id` lido de `<path>/.lumi/vault.yaml` (`server.vault_id`). Se não houver, `vault_id: null` e o estado vira "Erro: ligação incompleta", com o botão **[Religar]**. |
| TUI `<vault>/.lumi/config.yaml` (`server_url`, `server_token`) | `accounts.yaml` | Chama `GET /api/users/me` para descobrir o `username`, grava a conta e **remove `server_token` do arquivo da pasta**. Pastas de notas costumam passar por iCloud, Syncthing ou Git, e um token ali é um vazamento à espera de acontecer. |

## 3. Contrato do servidor

### 3.1 SSE, não WebSocket

O tráfego só vai do servidor para o cliente. SSE é HTTP comum: passa por proxy reverso e Tailscale, usa o mesmo middleware de autenticação e de CORS do REST e reconecta com pouco código. Um WS por conta seria um segundo protocolo binário, com pings próprios e outra superfície de upgrade, ao lado do WS Yjs, que continua por nota e sem mudanças. Foi o mesmo motivo do porvia (0034 §4).

### 3.2 Endpoint e autenticação

```
GET /api/events
Authorization: Bearer <token>      (ou X-Lumi-Token)
Accept: text/event-stream
```

- A rota fica no grupo autenticado `/api` e **não** entra em `queryTokenPathPrefixes`. Um `?token=` em `/api/events` é ignorado e resulta em 401, com teste. Os clientes leem o stream com `fetch` + `ReadableStream` (web), `URLSession.bytes(for:)` (Apple) ou `net/http` (TUI). Como o `EventSource` do navegador não manda cabeçalhos, ele não é usado. Por isso não é preciso ticket curto.
- Cabeçalhos da resposta: `Content-Type: text/event-stream`, `Cache-Control: no-store`, `X-Accel-Buffering: no`. O CORS segue `LUMI_ALLOWED_ORIGINS`, que já libera `Authorization`.
- O servidor não manda `id:` e ignora `Last-Event-ID`. **Não há replay.**

### 3.3 Eventos

```
event: hello
data: {"protocol":1,"server_time":"2026-11-22T12:00:00Z","user_id":"…","vaults":[{"vault_id":"…","seq":42,"role":"Editor","can_edit":true}]}

event: vault.head
data: {"vault_id":"…","seq":43}

event: vault.access
data: {"vault_id":"…","role":"Viewer","can_edit":false}

event: vault.updated
data: {"vault_id":"…","fields":["name","federation"]}

event: vault.revoked
data: {"vault_id":"…","reason":"removed"}

event: session.ended
data: {"reason":"revoked"}

event: reconnect
data: {"after_ms":0}

: ping
```

| Evento | Quando | O cliente faz |
|---|---|---|
| `hello` | Primeiro evento de toda conexão. Lista todos os vaults que a conta pode ler. | Reconcilia: livros novos (§2.4), papéis e `changes` de cada livro com `seq > cursor`. Um livro ligado a esta conta que não aparece = `revoked`. |
| `vault.head` | Qualquer mudança em notas (criar, editar conteúdo, mover, renomear, apagar, copiar para dentro), venha de onde vier: cliente, fs-watcher ou federação. | Se `seq > cursor`, chama `GET …/changes?after=cursor`. |
| `vault.access` | A conta ganhou acesso (convite aceito em outro aparelho, cópia recebida, transferência) ou o papel mudou. | Cria o livro, ou atualiza `can_edit` e o estado "Somente leitura". |
| `vault.updated` | Mudou nome, dono, lista de federações ou status de federação (`severed`). | Refaz `GET /api/vaults/:vault`. |
| `vault.revoked` | `reason`: `removed` (saiu dos membros), `deleted`, `severed_unreadable` (o seguidor perdeu a réplica). | Fecha o editor, marca o livro e oferece **[Remover da lista]**. Uma pasta espelho, se houver, fica. |
| `session.ended` | Sessão revogada, expirada ou usuário apagado (LGPD). O stream é fechado logo depois. | Conta → "Precisa entrar". Não reconecta. |
| `reconnect` | Desligamento do servidor ou fim do tempo máximo de vida (30 min). | Reconecta depois de `after_ms` mais jitter. |
| `: ping` | A cada 25 s. | Sem bytes por 60 s = conexão morta → reconecta. |

Os eventos levam **só ids, `seq` e papel**. Nenhum título, caminho ou id de nota passa pelo feed, para minimizar dados em logs de proxy. O que mudou se busca por REST, com o mesmo controle de acesso de sempre.

### 3.4 Definição de `seq`

Migração `0006_vault_feed`:

```sql
CREATE TABLE vault_heads (
  vault_id        UUID PRIMARY KEY REFERENCES vaults(id) ON DELETE CASCADE,
  seq             BIGINT NOT NULL DEFAULT 0,
  tombstone_floor BIGINT NOT NULL DEFAULT 0,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE notes ADD COLUMN change_seq BIGINT NOT NULL DEFAULT 0;
CREATE INDEX notes_vault_change_seq_idx ON notes (vault_id, change_seq);
CREATE TABLE note_tombstones (
  vault_id   UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  note_id    TEXT NOT NULL,
  seq        BIGINT NOT NULL,
  deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (vault_id, note_id, seq)
);
CREATE INDEX note_tombstones_vault_seq_idx ON note_tombstones (vault_id, seq);
-- backfill: uma linha por vault; seq = 1 e notes.change_seq = 1 onde houver notas
```

- **Toda escrita de metadado de nota** (create, patch/move/rename, delete, copy, criação vinda da federação) faz `UPDATE vault_heads SET seq = seq + 1, updated_at = now() WHERE vault_id = $1 RETURNING seq` **na mesma transação** e grava esse valor em `notes.change_seq`, ou numa nova linha de `note_tombstones` quando a nota é apagada. Um rename que troca o id gera tombstone do id antigo mais uma linha nova. O `UPDATE … RETURNING` serializa as escritas só dentro do mesmo vault (porvia 0035).
- **Mudança de conteúdo** (update Yjs aplicado no Room) não sobe o `seq` a cada tecla. Um *coalescer* por nota dentro do hub agenda `bumpContent(vault, note)` e roda no máximo uma vez por segundo por nota. Ele sobe `vault_heads.seq`, grava `notes.change_seq` e `notes.updated_at` (o que conserta, de quebra, o `updated_at` que hoje não muda em edições ao vivo). Se o servidor cair, o bump pendente se perde. Isso é aceitável: o conteúdo já está no log Yjs, e a regra "reconciliar ao conectar" cobre o caso. Na pior hipótese a lista fica com um `updated_at` atrasado até a próxima edição.
- O `seq` **não vem** de `note_yjs_updates.id`. Esse é um BIGSERIAL global: entregaria a atividade de outros vaults e some na compactação.
- Membros, papéis e nome **não** sobem `seq`. Eles têm eventos próprios (`vault.access`, `vault.updated`), porque o cliente refaz o detalhe inteiro.
- As tombstones são podadas depois de 30 dias. A poda grava o maior `seq` removido em `tombstone_floor`.

### 3.5 REST novo ou alterado

| Rota | Mudança |
|---|---|
| `GET /api/info` (pública) | `{"name","version","protocol":1,"public_key","registration"}`. Reaproveita a chave de `server_keys` e substitui `/api/federation/identity` para os clientes. Usada para adicionar servidor e para detectar URLs duplicadas. |
| `GET /api/vaults` | Cada vault ganha `head`, `role`, `can_edit` e `federation: {"role":"home"\|"follower"\|null,"home_url","status"}`. |
| `GET /api/vaults/:vault/changes?after=N&limit=500` | `{"notes":[NoteSummary + "change_seq"],"deleted":[{"id","seq"}],"head","more"}` em ordem de `seq`. `limit` máximo 1000, paginação por cursor, nunca `OFFSET`. Exige `note.read`. Se `N < tombstone_floor`, responde `410 {"error":"resync_required","head"}`. O cliente então lê `head` **antes**, lista tudo com `GET …/notes` e grava o cursor = esse `head`. As mudanças que caírem no meio voltam no próximo `changes`, e reaplicar é idempotente. `after=0` = tudo. |
| `DELETE /api/auth/sessions/current` | Já existe como logout. Passa a publicar `session.ended` no barramento interno. |

### 3.6 Lado do servidor: pacote `internal/feed`

- `Bus` em processo, que recebe `Publish(VaultEvent | UserEvent)`. Quem publica: serviço de notas, coalescer do hub, `members`, `invites`, `vaults` (rename, delete, transfer, copy), `auth` (logout, erasure) e o *apply* do control state da federação no seguidor.
- `Broker` com assinantes por usuário. Cada assinante tem um conjunto de `vault_id`s legíveis, carregado no `hello` com `ListForUser` e atualizado pelos próprios eventos de membership. **O filtro de capacidade é a regra:** um evento só sai para quem tem papel no vault com `note.read`. Teste obrigatório: o token da conta B nunca recebe nada do vault de A.
- A sessão é validada de novo a cada 60 s. Hoje um WS continua vivo com o token revogado (CRDT S6); o feed não repete esse erro. O `session.ended` chega na hora pelo barramento, e o recheck cobre o resto.
- **Limites:**
  - 16 streams simultâneos por usuário: o 17º leva `429 too_many_streams`.
  - Abertura de streams: balde de 20 por minuto por usuário, `429` com `Retry-After`.
  - Buffer de 128 eventos por assinante. `vault.head` é coalescido por vault (vale o último) e enviado no máximo a cada 250 ms.
  - Se o buffer estourar, a conexão é fechada. O cliente reconecta e reconcilia.
- Escala: uma instância hoje, com broker em memória. Com várias instâncias, `LISTEN/NOTIFY` (`vault_head`, payload `vault_id:seq`) espalha os eventos, como no porvia 0035 §22. Fica anotado, não implementado.

### 3.7 Federação

Cada servidor tem o seu `seq` e o seu feed, só para as **contas locais**. Um update que chega pelo relay passa pelo Room do hub (o único doc autoritativo, branch `fix/wave1-security`), cai no coalescer e vira `vault.head` para os assinantes locais daquele servidor. Mudança de membros no seguidor chega como control state assinado. O *apply* compara o conjunto anterior com o novo e publica `vault.access` ou `vault.revoked` para os usuários locais afetados. Um `severed` publica `vault.updated{fields:["federation"]}`. Os clientes mostram a faixa *"Este livro deixou de sincronizar com <origem>"*, como exige o SPEC-V3. Como o cursor é por servidor, um livro nunca ouve dois servidores (§2.4).

## 4. Estado de sincronização por livro

### 4.1 Estados (rótulos de UI pt-BR / en)

| Estado | Condição | Indicador |
|---|---|---|
| **Não ligado** / Local | `kind: local` | ícone de pasta, sem ponto |
| **Precisa entrar** / Sign in | Livro sem conta neste cliente, token ausente, 401 ou `session.ended` | ponto âmbar + botão "Entrar" |
| **Erro** / Error | 3 falhas seguidas não transitórias (5xx, JSON inválido, `vault_id` nulo, `vault.revoked`), com motivo legível | ponto vermelho |
| **Offline** | Feed da conta desconectado, ou rede indisponível | ponto cinza vazado |
| **Sincronizando** / Syncing | Feed conectado e (`changes` em andamento, ou `head > cursor`, ou editor ativo em CONNECTING/SYNCING, ou edições locais ainda não confirmadas) | ponto pulsando |
| **Somente leitura** / Read-only | Atualizado e `can_edit = false` | cadeado |
| **Atualizado** / Up to date | Feed conectado, `cursor = head`, editor LIVE ou fechado, nada pendente | ponto verde |

**Precedência** (vence o primeiro que valer): Não ligado → Precisa entrar → Erro → Offline → Sincronizando → Somente leitura → Atualizado.

### 4.2 Cálculo

O estado é uma função pura, `bookStatus(book, accountFeed, catchUp, editor, restHealth)`, com teste de tabela nos três clientes e o mesmo conjunto de casos. As entradas são:

- **`accountFeed`** (um por conta, não por livro): `connecting`, `open` (depois do `hello`), `backoff`, `auth_dead` ou `disabled`. Backoff começa em 1 s, multiplica por 2, para em 30 s e tem jitter. Respeita `Retry-After`.
- **`catchUp`**: `idle`, `running` ou `failed(n, reason)`, junto com `cursor` e `head`.
- **`editor`**: o estado da sessão do CRDT-PROTOCOL §2.1. "Pendente" quer dizer edições locais feitas enquanto OFFLINE e ainda sem SyncStep2 depois de reconectar. No TUI é o `.base` em `.lumi/cache` com um envio que ainda não voltou 2xx (CRDT §7.1).
- **`restHealth`**: a última falha de REST por conta. Um erro de rede manda para Offline sem esperar o feed.

`last_synced_at` = a última vez que o livro chegou a "Atualizado". Fica em memória e é salvo junto do cache, nunca em `books.yaml`.

### 4.3 Onde aparece

- **Web:** na `topbar` de `VaultHomeView`, à direita, antes de "+ New", vai um ponto com o rótulo curto. Clicar abre um popover com:
  - servidor, conta, papel e estado;
  - "última sincronização há 2 min";
  - `cursor/head`;
  - alternativas (§2.4);
  - os últimos 5 erros;
  - os botões **[Tentar agora]**, **[Entrar]**, **[Trocar servidor…]** e **[Desligar]**.

  No seletor de livros, cada linha mostra o seu ponto.
- **Apple:** item de toolbar com o mesmo popover (como no porvia 0034 §9). Os pontos também aparecem nas linhas do seletor. O `ReconnectStatus` do editor passa a alimentar o estado, em vez de ter um banner próprio.
- **TUI:** um segmento na status line (`● atualizado`, `◌ offline`, `↻ sincronizando`). `gs` abre um modal de detalhes com as mesmas informações.

## 5. UX

**Seletor.** No web e no Apple fica no topo da barra lateral: um botão com o nome do livro ativo e o ponto de estado. Ele abre uma lista agrupada por servidor, com os livros locais primeiro. No fim da lista vêm "Novo livro…", "Ligar pasta…" (só Apple), "Adicionar servidor…" e "Gerenciar livros…". No web, `VaultsView` vira esse seletor em tela cheia (mesma navegação j/k/Enter) e "← Vaults" vira "← Livros". No TUI, o picker atual (`cmd_picker.go`) passa a ler `books.yaml`, mostra servidor e estado em cada linha e abre com `gb` em qualquer tela. Escolher um livro reinicia o model com o novo `rootDir`. Um livro ligado sem `path` oferece "Clonar para…".

**Atalhos.** Os nove primeiros livros seguem a ordem do seletor (`sort`).
- Apple: ⌃⌘1–9 trocam de livro, ⌃⌘0 abre o seletor. Ficam num `CommandMenu("Livro")`.
- Web: `g1`–`g9` e `gb`, como acorde em contexto não editável, com o mesmo `isEditableTarget` que já existe (`gg` continua sendo "topo"). No macOS, ⌃⌘1–9 também, porque não conflitam com a troca de abas.
- TUI: `g1`–`g9`, `gb` e `gs` no prefixo `g` de `keys_tree.go`.

**Criar, ligar, desligar, renomear.**
- *Novo livro:* escolher onde. "Neste aparelho" (Apple/TUI) cria uma pasta. Uma conta → `POST /api/vaults`.
- *Ligar* (local → servidor): escolher uma conta, criar o vault, enviar as notas pelo fluxo de `lumi vault link`, gravar `vault_id`.
- *Desligar* (ligado → local): com pasta, ela vira um livro local e o servidor fica intacto. Sem pasta, "Remover da lista", com a opção **[Baixar cópia antes]**. Nunca apaga o vault no servidor. "Excluir livro no servidor" é uma ação separada, com confirmação dupla e `vault.manage`.
- *Renomear:* livro local muda `name`. Livro ligado chama `PATCH`, e o campo fica desabilitado sem `vault.manage`, com a explicação ao lado.
- *Reordenar:* arrastar no "Gerenciar livros" (web/Apple). No TUI, `J`/`K` no picker.

**Adicionar servidor.**
1. URL.
2. `GET /api/info` mostra o nome do servidor, a versão e o aviso de HTTP quando houver. No web, uma falha de CORS mostra: *"Este servidor não aceita acesso a partir de <origem>. Peça ao administrador para incluir <origem> em LUMI_ALLOWED_ORIGINS."*
3. Usuário e senha, ou um convite.
4. "Lembrar neste aparelho" (web).
5. Os livros da conta entram na lista pela regra §2.4, com uma faixa: *"3 livros adicionados de lumi.work.com"*.

No web o campo de URL já vem com `VITE_LUMI_SERVER_URL` ou, se não houver, com `location.origin`. Uma página HTTPS não fala com servidor `http://` (mixed content), e a mensagem diz isso.

**Primeira vez.** Um cartão aparece uma vez (`lumi.onboarding.books.v1`): *"Cada livro é um espaço separado de notas. Ele pode ficar só neste aparelho ou ligado a um servidor, e cada livro ligado sincroniza com um único servidor."*

**Estados vazios.**
- Web sem conta: tela de login com "Adicionar servidor".
- Sem livros: *"Nenhum livro ainda"* com [Novo livro] e [Adicionar servidor].
- Livro revogado: *"Você não tem mais acesso a este livro"* com [Remover da lista].
- "Precisa entrar": *"Entre em lumi.work.com para sincronizar"* com [Entrar].

Tudo segue a linguagem visual atual: monoespaçada, rótulos em minúsculas, `link-button` no web, `theme.textDim` no Apple, estilos de `ui/styles.go` no TUI.

## 6. Itens de trabalho

**Contrato primeiro (C0, 1 dia).** Congelar §3.3 e §3.5 como fixtures: `fixtures/feed/*.sse` (streams gravados, inclusive com cortes de chunk no meio do evento), `fixtures/changes/*.json` e `fixtures/books.v1.yaml`. Todos ficam no monorepo e são copiados para cada submódulo, com um script `make fixtures-check` que compara hashes. Cada cliente sobe um mock de feed a partir desses fixtures e não precisa esperar o servidor.

### Servidor

| # | Item | Depende de | Aceite e testes |
|---|---|---|---|
| S0 | Spike: SSE no Fiber/fasthttp (`SetBodyStreamWriter` + flush) contra `WriteTimeout: 60s` | — | Um stream fica de pé 10 min com ping. Se o prazo do fasthttp for absoluto, o tempo máximo de vida cai para 50 s com `reconnect`, ou o SSE vai para um listener próprio. A decisão fica registrada. |
| S1 | `0006_vault_feed`, bumps transacionais nas escritas de metadado, tombstones e poda, `GET /changes`, campos novos em `GET /api/vaults`, `GET /api/info` | C0 | `seq` estritamente crescente com 50 escritas concorrentes no mesmo vault. `changes` pagina sem buracos. `410` abaixo do floor. Teste de autorização com o token de outra conta: 404. Backfill idempotente. |
| S2 | `internal/feed` (Bus, Broker, Handler): `hello`, `vault.access`, `vault.updated`, `vault.revoked`, `session.ended`, ping, limites, recheck de sessão | S0, S1 | B nunca recebe eventos de A. Remover um membro produz `revoked` em menos de 1 s. Logout fecha o stream na hora. Um token revogado por fora cai em até 60 s. `?token=` é recusado. O 17º stream leva 429. Um assinante lento é desconectado. Wire bate com os fixtures (golden). |
| S3 | Coalescer de conteúdo no hub → `vault.head` | S2, **`fix/wave1-security`** (S2 do protocolo: toda escrita passa pelo Room autoritativo, e ApplyDiff respeita a base) | Edição ao vivo, `/diff` REST e fs-watcher produzem **um** `vault.head` por segundo por nota. Um update no-op (CRDT S3) não produz evento. |
| S4 | Federação: o *apply* do control state publica access e revoked, `severed` → `vault.updated` | S2; teste de dois servidores depende de S3 | Teste de integração com dois servidores (`-tags=integration`): editar no seguidor sobe o head nos assinantes do seguidor e da origem. Remover na origem gera `revoked` no seguidor. |

### Web, Apple e TUI (em paralelo depois de C0)

| # | Item | Depende de | Aceite e testes |
|---|---|---|---|
| W1 | `accounts` em IndexedDB com `device-key`, migração de `lumi.v2.session`, `api.ts` com um cliente por conta (`createClient(serverUrl, tokenRef)`), fim do `API_URL` global | C0 | Vitest: a migração preserva a sessão, a chave fixa some, um 401 numa conta não derruba as outras. |
| W2 | Registro `books`, dedupe §2.4, seletor e `VaultsView`, atalhos, livro ativo, onboarding e estados vazios | W1 | Vitest: dedupe com servidores de origem e seguidor (fixtures), ordem de `g1`–`g9`. |
| W3 | Leitor SSE com `fetch` (parser incremental: CRLF, comentários, `data` em várias linhas, chunks cortados), `FeedConnection` por conta, catch-up com `changes`, reducer `bookStatus` e popover | W1; S2 real, ou o mock C0 | Vitest com os fixtures `.sse` e a tabela de estados. Manual: derrubar o servidor → Offline em menos de 60 s, voltar → Atualizado. |
| A1 | `AccountsStore` com várias contas no Keychain e migração. `LumiAPIClient` por conta. `RemoteVaultsStore` deixa de depender de `currentSession` | C0 | XCTest: a migração do item antigo, duas contas no mesmo servidor. |
| A2 | `BooksRegistry` (`books.yaml` + `BookBookmark`), migração de `VaultRecord` e `vaults.yaml`, seletor na `VaultSidebar`, ⌃⌘1–9 | A1 | XCTest: round-trip byte a byte contra `fixtures/books.v1.yaml`, o mesmo arquivo do T2. |
| A3 | Feed com `URLSession.bytes`, um por conta, catch-up, reducer e item de toolbar com popover | A1; S2 ou mock | XCTest com fixtures `.sse` e a tabela de estados. Feed vivo com a tela bloqueada (macOS). |
| T1 | `accounts.yaml` v2 com várias contas e importação de `.lumi/config.yaml` (token removido da pasta) | C0 | Go: v1 → v2 sem perda, 0600 depois de toda escrita, a pasta fica sem `server_token`. |
| T2 | `books.yaml` (leitor e escritor), importação de `vaults.yaml`, picker com `gb`/`g1`–`g9`, comandos `lumi books list\|link\|unlink\|rename\|use` (`lumi vaults` vira alias) | T1 | Go: golden byte a byte contra o fixture compartilhado. |
| T3 | Feed com `net/http` no lugar do `sync/` antigo (que falava com `/ws`), status line e `gs`. Puxar notas mudadas para a pasta espelho só quando a nota não está suja | T1; S2 ou mock. O estado "pendente" do editor depende de **`fix/wave1-security`** (ApplyDiff com a base, CRDT §7.2) | Go: parser contra os fixtures, reducer. Manual: editar no web → o arquivo atualiza em menos de 2 s no TUI. |

**Ordem:** C0 → (S0 ∥ W1 ∥ A1 ∥ T1) → (S1 ∥ W2 ∥ A2 ∥ T2) → S2 → (W3 ∥ A3 ∥ T3, já contra o servidor real) → S3 (quando `fix/wave1-security` entrar na `main`) → S4. Os clientes só esperam o servidor para o teste ponta a ponta, não para desenvolver.

## Segurança e privacidade (LGPD)

- **Tokens:**
  - Apple: Keychain `ThisDeviceOnly`.
  - TUI: arquivo 0600 e nenhum token em pasta de notas.
  - Web: chave não extraível, com a ameaça descrita sem exagero (§2.2).
  - Nenhum token em URL, em `books.yaml` ou em log.
- **Feed minimizado:** só ids, `seq` e papel. Toda leitura de conteúdo continua pelo REST com controle de acesso.
- **Erasure:** apagar o usuário publica `session.ended` e fecha os streams. O registro local de livros (URLs e usernames) é dado pessoal do próprio usuário, fica só no aparelho e some com "Remover conta neste aparelho".
- O feed não grava auditoria por conexão, para não fazer um log de presença. Só métricas agregadas.

## Performance

- Uma conexão por conta, não por livro.
- `changes` incremental com índice `(vault_id, change_seq)`: abrir um livro grande depois de horas offline busca só o que mudou.
- O bump de conteúdo coalescido custa no máximo um `UPDATE` por segundo por nota ativa.
- `vault.head` coalescido a cada 250 ms por assinante.
- Nenhum polling.

## Alternativas

- **WebSocket para o feed:** rejeitado (§3.1).
- **`EventSource` com ticket curto em `?ticket=`:** funcionaria, mas leva um segredo para a URL e precisa de uma rota a mais. `fetch` com streaming funciona nos três alvos.
- **SSE por vault:** N conexões por usuário. Rejeitado.
- **Polling de `GET /api/vaults`:** latência e custo sem motivo.
- **`seq` = `note_yjs_updates.id`:** global, vaza atividade entre vaults e some na compactação.
- **Livro ligado a vários servidores:** a redundância é papel da federação, como no porvia 0034 §8.
- **`vaults.yaml` v2:** clientes antigos estragariam o arquivo (§2.3).
- **Livro local no web via OPFS:** adiado. Não há como abrir a pasta no TUI nem no Apple.
- **Apelido local para livro ligado:** dois nomes para a mesma coisa confundem.

## Consequências

- Positivas: os três clientes passam a ter o mesmo modelo e a mesma tabela de estados. A federação deixa de duplicar livros. O usuário sempre sabe se está atualizado.
- Negativas:
  - Toda escrita nova de nota no servidor precisa subir o `seq`, com teste que cobre cada rota que escreve.
  - O web multi-servidor depende de CORS configurado em cada servidor.
  - Fica um terceiro arquivo de configuração no `~/.config/lumi`.

## 7. Fora do escopo (por enquanto)

**Espelhos de mão única.** Um servidor espelho seria um seguidor com `federation.role = "mirror"` que recusa escrita. O desenho já tem os encaixes. Os usuários do espelho recebem `can_edit = false` no `hello` e no `vault.access`, e o livro cai naturalmente em "Somente leitura". O `seq` já é por servidor, então o espelho mantém o seu, sem precisar copiar o da origem. "Promover espelho" seria só um `vault.updated{fields:["federation"]}` seguido de `vault.access{can_edit:true}`. Para o cliente, trocar a origem é o "Trocar servidor" do §2.4, que já preserva os Y.Docs.

**Livros selados (E2E).** Um livro selado teria `sealed: true` no registro e conteúdo opaco no servidor, como os espaços do porvia. O feed já não carrega nada além de `vault_id` e `seq`, então um livro selado não vaza nada novo por ele. O `seq` por vault, o `changes?after=` e as tombstones são o mesmo modelo de "registros depois de N" do porvia, com bytes cifrados no lugar de `NoteSummary`. O nome do livro passaria a ser só local (`name` sem `PATCH`). A chave do livro ficaria no Keychain, no arquivo 0600 ou atrás da `device-key`. Entra um estado a mais na precedência, logo depois de "Precisa entrar": **"Precisa da senha de cifra"**. O conflito com o CRDT (um servidor que não lê não faz merge Yjs nem mirror em disco) é o motivo de isso ser outra versão do protocolo.

## Decisões do dono (22/11/2026)
- Várias contas por servidor: **sim** (conta = `(server_url, username)`).
- Livro federado: liga por padrão ao **servidor de origem**; os demais ficam como alternativas em "Trocar servidor".
- Web, "Lembrar neste aparelho": **desmarcado por padrão** (sessão só na aba; com a opção marcada, token no IndexedDB).
- Renomear livro ligado: renomeia para todos (exige `vault.manage`), sem apelido local.
- Painel de administração do servidor: fica para a **onda 3**, no modelo do porvia (ADR 0037).

## 8. Login compartilhado entre o app e o TUI no Mac

- Status: decidido (24/11/2026). Pedido do dono: "o sistema de login e server pode ser o mesmo do mac instalado e o tui".
- Substitui, no macOS, as linhas Apple e TUI da tabela do §2.2. Linux e Windows seguem o mesmo arquivo de registro; só o lugar do token do TUI muda (§8.3).

### 8.1 Decisão

1. **Um registro de contas só**, `~/.config/lumi/accounts.yaml` `version: 3`, lido e escrito pelo app e pelo TUI. Ele **não guarda segredo**. Entrar, sair, adicionar ou remover uma conta num cliente aparece no outro em menos de 1 s (o app observa o diretório; o TUI confere o arquivo a cada 1 s enquanto está aberto e em todo comando).
2. **Uma sessão por cliente, não um token compartilhado.** O app guarda o seu token no Keychain (como hoje). O TUI guarda o seu em `~/.config/lumi/tokens.yaml` (0600). Quando um cliente vê no registro uma conta para a qual não tem sessão, ele pede ao outro um **link de aparelho** (§8.4): o cliente que tem sessão pede ao servidor um código de uso único, ligado a um desafio PKCE do cliente que pede, e o cliente que pede troca o código por uma **sessão nova, própria e revogável**. A senha é digitada uma vez só, em qualquer um dos dois.
3. **Quem confirma.** Se quem tem a sessão é o TUI, ele atende o pedido do app sem perguntar. Se quem tem a sessão é o app, ele **sempre pergunta** ("O lumi no Terminal quer entrar como alice em lumi.work.com. [Permitir] [Não permitir]"). O motivo está em §8.2.
4. **Sair.** "Sair deste app" encerra só a sessão daquele cliente. "Sair neste Mac" encerra a sessão dele **e todas as sessões ligadas a ela** no servidor (`POST /api/auth/logout` com `{"scope":"device"}`), e marca `signed_out_at` no registro para o outro cliente apagar o próprio token mesmo que as sessões não estejam ligadas. "Remover conta" faz o mesmo que "Sair neste Mac" e tira a linha do registro.

### 8.2 Por que não compartilhar o token pelo Keychain

Os dois binários são assinados ad hoc: `/Applications/Lumi.app` e `~/.local/bin/lumi`, recompilados com frequência.

- **(a) Um item do Keychain com ACL para os dois caminhos.** No Keychain de login (o "file-based", o único que um binário sem team id usa), a ACL e a partition list guardam o *designated requirement* de cada app. Num binário ad hoc, o DR é o `cdhash`, que muda a cada build. Depois de cada recompilação o macOS pergunta de novo ("lumi quer usar informações confidenciais…", às vezes pedindo a senha de login). Isso treina o usuário a clicar "Sempre permitir", e é o oposto de segurança. O atalho de ler o item via `/usr/bin/security` coloca na ACL um binário da Apple que qualquer processo pode chamar, o que equivale a não ter ACL.
- **(b) Access group compartilhado.** Exige o Keychain de proteção de dados (`kSecUseDataProtectionKeychain`) e o entitlement `keychain-access-groups`/`application-identifier`, que só valem com um team id e um provisioning profile. Assinatura ad hoc não tem team id, e uma ferramenta de linha de comando fora de um bundle não carrega provisioning profile. Fica para quando houver Developer ID; mesmo então, (d) continua melhor porque cada cliente fica revogável sozinho.
- **(c) O TUI pede o token ao app por XPC ou URL.** URL scheme abre e traz o app para a frente. Um serviço XPC/Mach precisa de registro no launchd e de validar quem chama pela assinatura, que no ad hoc não identifica nada. E o resultado seria o mesmo token nos dois lugares: revogar um derruba o outro, e o segredo do Keychain passa a morar num arquivo.
- **(d) Link de aparelho (escolhida).** Nenhum segredo atravessa de um cliente para o outro. O que passa pelo disco é um código de uso único, válido por 120 s, que **só funciona com o verificador PKCE que nunca sai da memória de quem pediu**. Cada cliente tem a própria sessão, com o próprio TTL deslizante, e pode sair sozinho. Não há prompt do Keychain além dos que o app já tem hoje.

**Por que o app pergunta e o TUI não.** Qualquer processo do mesmo usuário consegue escrever na pasta de pedidos. Se o app atendesse sozinho, qualquer programa do usuário ganharia uma sessão nova a qualquer momento, e a proteção do Keychain deixaria de valer. Já o token do TUI mora num arquivo 0600 que esse mesmo programa consegue ler, então o TUI atender sozinho não abre nada que já não estivesse aberto. Pelo mesmo motivo o app não "empurra" sessão para o TUI: o TUI pede, e o app confirma uma vez.

### 8.3 Arquivos

**`~/.config/lumi/accounts.yaml` v3** (0600, diretório 0700). Formato canônico, com bytes iguais no Swift e no Go, testado contra `fixtures/accounts/accounts.v3.yaml`:

```yaml
# ~/.config/lumi/accounts.yaml — lumi account registry, no secrets. Format: SPEC-LIVROS §8.3.
version: 3
accounts:
  - server: "https://lumi.work.com"
    username: "alice"
    user_id: 0192f7a1-3b4c-7d5e-8f60-718293a4b5c6
    display_name: "Alice"
    web_origin: "https://notes.work.com"
    added_at: 2026-11-20T08:30:00Z
    expires_at: 2026-12-20T08:30:00Z
    signed_out_at: null
```

- Ordem das chaves fixa (a acima). Contas ordenadas por `added_at`, depois `server`, depois `username` (bytes). `accounts: []` quando vazio.
- `server`, `username`, `display_name`: sempre entre aspas duplas (`display_name` pode ser `""`). `web_origin`: aspas ou `null`. `user_id`: sem aspas quando é UUID canônico em minúsculas, entre aspas quando não é, `null` quando vazio. Datas em UTC com segundos e `Z`, ou `null`. Escape, LF e linha final iguais ao §2.3.
- A chave YAML continua `server` (é o `server_url` do §2.2). Um TUI antigo que regravar o arquivo mantém a URL de cada conta. No máximo ele põe de volta um `token:`, que o próximo load v3 tira dali.
- `username` em minúsculas, sem espaços. `server` normalizado (§2.1). Linhas duplicadas: fica a de `added_at` mais recente; empate, a última do arquivo.
- `expires_at` é a última validade conhecida de qualquer cliente, só para a UI. `signed_out_at` diz "toda sessão deste Mac anterior a este instante está encerrada". Entrar de novo não apaga o campo: a sessão nova é mais recente que ele.
- O leitor é tolerante (comentários, CRLF, BOM, aspas simples, escalares sem aspas, `~`, frações de segundo, fuso). Um arquivo com `version` maior que 3 é só leitura.

**Escrita concorrente.** Toda escrita é ler, mudar e gravar sob `flock(LOCK_EX)` no arquivo irmão `accounts.yaml.lock` (0600): trava, relê do disco, aplica a mudança, grava num temporário 0600 no mesmo diretório, faz `fsync`, `rename` e destrava. A leitura não trava, porque o `rename` é atômico. Nenhum dos dois usa `NSFileCoordinator` neste arquivo. No Windows, onde não há `flock`, o TUI usa um lockfile com `O_EXCL` e expiração de 10 s.

**Tokens do TUI: `~/.config/lumi/tokens.yaml`** (0600, só o TUI lê e escreve, com o próprio lock):

```yaml
version: 1
tokens:
  - server: https://lumi.work.com
    username: alice
    token: "…"
    expires_at: 2026-12-20T08:30:00Z
    signed_in_at: 2026-11-20T08:30:00Z
opt_out: []        # contas em que o usuário saiu só do TUI: não pedir link sozinho
```

No macOS o TUI continua num arquivo 0600, e não no Keychain, pelo motivo de (a): cada recompilação geraria um prompt. Quando o TUI tiver assinatura estável, ele pode passar a usar um item próprio no Keychain sem mudar mais nada daqui. No Linux e no Windows é o mesmo arquivo; não há regressão em relação ao `accounts.yaml` v2.

**Token do app:** Keychain, como no §2.2 (`service = com.vinizap.lumi.auth`, `account = "<server>#<username>"`, `AfterFirstUnlockThisDeviceOnly`). O instante de entrada fica em `UserDefaults['lumi.accounts.signedInAt.v1']`, e as contas em que o usuário saiu só do app em `UserDefaults['lumi.accounts.linkOptOut.v1']`.

### 8.4 Link de aparelho

**Servidor.**

```
POST /api/auth/device-links            (autenticado)
{"challenge": "<base64url sem padding de SHA-256(verifier), 43 chars>", "label": "lumi TUI"}
→ 201 {"code": "<64 hex>", "expires_at": "…"}

POST /api/auth/device-links/redeem     (público)
{"code": "…", "verifier": "<43–128 chars base64url>", "label": "Lumi.app"}
→ 200 {"token": "…", "expires_at": "…", "user": {…}}     (mesmo corpo do login)
→ 401 {"error": "invalid_link"}                           (código desconhecido, usado, vencido, verificador errado ou sessão emissora encerrada)

POST /api/auth/logout                  (autenticado)
{"scope": "device"}  → 204   (sem corpo, ou "session": só a sessão atual, como hoje)
```

- Migração `0011_device_links`: `device_link_codes(code_hash PK, user_id, issuer_token_hash → sessions ON DELETE CASCADE, challenge, label, created_at, expires_at)` e `session_links(token_hash PK → sessions ON DELETE CASCADE, link_group, created_at)`. O código vale 120 s, guarda só o SHA-256 e é consumido na primeira tentativa, mesmo quando o verificador está errado. Se a sessão que emitiu o código acaba, o código acaba junto.
- O resgate cria a sessão nova e põe a emissora e a nova no mesmo `link_group`. `scope=device` apaga a sessão atual e todas as do grupo, sempre do mesmo usuário.
- Limites: 10 emissões por minuto por usuário e 30 resgates por minuto por IP. Auditoria: `auth.device_link_issue`, `auth.device_link_redeem`, `auth.device_link_failed` e `auth.logout_device`, com o `label` e sem código ou token.

**Canal local:** `~/.config/lumi/link/` (0700; arquivos 0600, escrita atômica). Os nomes de cliente são `apple` e `tui`.

| Arquivo | Quem escreve | Conteúdo |
|---|---|---|
| `<cliente>.alive` | o próprio cliente, enquanto roda (o app sempre, o TUI só na tela interativa) | `{"v":1,"pid":123,"started_at":"…"}` |
| `req-<id>.json` | quem pede | `{"v":1,"id":"<uuid>","from":"tui","to":"apple","server":"…","username":"alice","challenge":"…","label":"…","pid":123,"created_at":"…"}` |
| `resp-<id>.json` | quem tem a sessão | `{"v":1,"id":"…","code":"…","expires_at":"…"}` ou `{"v":1,"id":"…","error":"denied"\|"no_session"\|"failed"}` |

Fluxo:

1. Quem pede gera um `verifier` com 32 bytes aleatórios, que **fica só na memória**, e grava o `req`.
2. Quem tem a sessão vê o `req` (o app observa o diretório; o TUI confere ao abrir, a cada 1 s na tela interativa e depois de todo `lumi login`). Ele confere `v`, `to`, se a conta está no registro, o formato do `challenge` e a idade do pedido: até 5 min, ou mais se o `pid` de quem pediu ainda está vivo. Depois atende (TUI) ou pergunta (app), chama `POST /device-links` e grava o `resp`.
3. Quem pede lê o `resp`, resgata o código com o `verifier`, guarda o token no próprio armazenamento e apaga `req` e `resp`.

Um código lido do disco por outro processo não serve para nada sem o verificador. Pedido negado vira `opt_out` de quem pediu e não se repete sozinho.

**Quando cada cliente pede.**
- **TUI:** ao abrir a tela interativa e em todo comando que precisa de uma conta sem token. Só pede se o app está vivo (`apple.alive` com `pid` vivo) e a conta não está em `opt_out`. No terminal aparece "Aprove no Lumi.app… (Ctrl-C cancela)", com espera de até 120 s. `lumi accounts link [<server>] [--account u]` pede de forma explícita e limpa o `opt_out`.
- **Rótulo:** o código emitido leva o `label` do pedido; o resgate leva o rótulo de quem resgata (`lumi TUI` ou `Lumi.app`).
- **App:** pede para toda conta do registro sem token no Keychain que não está em `linkOptOut`. O pedido fica valendo enquanto o app roda e é atendido na próxima vez que o TUI abrir. A conta aparece como "Precisa entrar — ou abra o lumi no Terminal para ligar".

### 8.5 Propagação

| Ação num cliente | Registro | Servidor | O outro cliente |
|---|---|---|---|
| Entrar com senha | upsert da linha | sessão nova | vê a conta; se não tem sessão, pede link |
| Ligar por link | upsert da linha | sessão nova no mesmo `link_group` | — |
| Sair deste app | nada | revoga a própria sessão | continua conectado |
| Sair neste Mac | `signed_out_at = agora` | `logout {scope: device}` | apaga o próprio token se o seu `signed_in_at`, truncado ao segundo, não for posterior a `signed_out_at` (uma entrada no mesmo segundo perde), e chama `logout` por garantia |
| Remover conta | linha removida | `logout {scope: device}` | linha sumiu → apaga o token e chama `logout` |
| 401 / `session.ended` | nada | — | não muda |

### 8.6 Migração

| De | Para | Como |
|---|---|---|
| TUI `accounts.yaml` v1/v2 (com `token:`) | `tokens.yaml` + `accounts.yaml` v3 | Sob os dois locks: grava `tokens.yaml` primeiro, depois o registro v3. Uma queda no meio deixa o token duplicado, mas nunca o perde. |
| Apple `UserDefaults['lumi.accounts.v1']` | `accounts.yaml` v3 | Faz o merge das linhas que faltam no registro e só apaga a chave depois de gravar. O Keychain não muda (mesmo `service`/`account`). |
| `accounts.yaml` v1/v2 encontrado pelo app | `accounts.v2.yaml` + v3 | O app **não lê nem copia tokens**. Ele renomeia o arquivo para `accounts.v2.yaml` (0600), grava o v3 com as mesmas linhas e mais as suas. O TUI importa os tokens de `accounts.v2.yaml` para `tokens.yaml` e apaga o arquivo. |

### 8.7 Testes

- Go e Swift: golden byte a byte de `accounts.v3.yaml`; normalização de `accounts.v3.noncanonical.yaml`; leitura de `accounts.v2.yaml` (saída do yaml.v3) igual a `accounts.v2.migrated.yaml`; dois escritores concorrentes (processos ou goroutines/threads) sem perder linha; 0600 depois de toda escrita.
- Fixtures de pedido e resposta em `fixtures/accounts/link-*.json`, decodificados pelos dois lados.
- Servidor: emitir, resgatar, código reusado, vencido, verificador errado (queima o código), sessão emissora encerrada, `scope=device` revoga só o grupo, `scope` ausente segue como hoje.
- TUI: `httptest` com emissão e resgate de ponta a ponta através do canal local, propagação de `signed_out_at` e de linha removida, migração v2 → v3 e importação de `accounts.v2.yaml`.
- Apple: `TokenStore` injetado (nunca o Keychain real), `HOME` e `UserDefaults` temporários, propagação de saída e migração do `UserDefaults`.
