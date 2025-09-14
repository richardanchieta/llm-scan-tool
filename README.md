# llm-scan-tool

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/richardanchieta/llm-scan-tool)](https://goreportcard.com/report/github.com/richardanchieta/llm-scan-tool)
[![Build](https://github.com/richardanchieta/llm-scan-tool/actions/workflows/go.yml/badge.svg)](https://github.com/richardanchieta/llm-scan-tool/actions)


⚡ Summarize your monorepo for humans & LLMs — fast, structured, and context-ready.



## Visão Geral
O **llm-scan-tool** é uma ferramenta em Go que varre um repositório (monorepo ou projeto único) e gera um **artefato otimizado para consumo por LLMs**.  
Ele coleta metadados técnicos, estatísticas de código e resumos de documentação, consolidando em relatórios Markdown e JSON.

Esse artefato serve como **contexto de trabalho** para agentes de IA, tech leads e desenvolvedores que precisam entender rapidamente a estrutura, dependências e decisões de arquitetura do projeto.

---

## Funcionalidades
- **Descoberta de módulos Go** (`go.mod`) e dependências.
- **Parsing de Protobufs**: pacotes, serviços e RPCs definidos.
- **Detecção de Make targets** e comandos úteis.
- **SQL migrations** (via Atlas/Goose) listadas por ordem.
- **Dockerfiles** e configs relevantes.
- **ADRs e decisões técnicas** (resumidas por arquivo).
- **READMEs**: extração de título, primeiro parágrafo e seção *Objetivo*.
- **Estatísticas técnicas** por extensão de arquivo (`.go`, `.proto`, `.sql`, `.md`, etc).
- **Árvore de diretórios** limitada em profundidade.
- Saída em **Markdown** (`LLM_SUMMARY.md`) e **JSON** (`LLM_SUMMARY.md.json`).

---

## Exemplo de Uso

```bash
# build
make build

# scan um monorepo a partir da raiz
./llm-scan -root /path/to/monorepo -out LLM_SUMMARY.md -tree-depth 3

# saída paralela em JSON
cat LLM_SUMMARY.md.json | jq .
```

Saída esperada (trecho):

```json
{
  "root": "/home/richardanchieta/repos/baseron",
  "generated_at": "2025-09-14T18:00:00Z",
  "go_modules": [
    {
      "path": "go.mod",
      "module": "github.com/richardanchieta/baseron",
      "requires": [
        "go.uber.org/fx",
        "github.com/nats-io/nats.go",
        "google.golang.org/protobuf",
        "github.com/sqlc-dev/sqlc",
        "github.com/jackc/pgx/v5"
      ]
    }
  ],
  "proto": [
    {
      "file": "proto/agent/v1/agent.proto",
      "package": "agent.v1",
      "services": ["AgentService"],
      "rpcs": [
        "CreateAgent",
        "ListAgents",
        "ExecuteRecipe"
      ]
    }
  ],
  "make_targets": [
    "build",
    "lint",
    "test",
    "proto-gen",
    "sqlc-gen",
    "docker-build",
    "docker-run"
  ],
  "dockerfiles": [
    "Dockerfile",
    "services/agent/Dockerfile",
    "services/recipe/Dockerfile"
  ],
  "sql_migrations": [
    "db/migrations/0001_init_schema.sql",
    "db/migrations/0002_agents.sql",
    "db/migrations/0003_recipes.sql"
  ],
  "decisions": [
    {
      "file": "docs/decisions/adr-001-monorepo.md",
      "title": "Adotar monorepo com Go",
      "summary": "Optamos por monorepo para compartilhar módulos internos (proto, libs) e facilitar CI/CD unificado."
    },
    {
      "file": "docs/decisions/adr-002-agent-recipes.md",
      "title": "Recipes declarativos em YAML",
      "summary": "Definição de recipes como YAML declarativo, permitindo instanciar execuções sem acoplamento de lógica ao core."
    }
  ],
  "env_examples": [
    ".env.example"
  ],
  "licenses": [
    "LICENSE"
  ],
  "readmes": [
    "README.md",
    "services/agent/README.md",
    "services/recipe/README.md"
  ],
  "readme_summaries": {
    "README.md": {
      "file": "README.md",
      "title": "Baseron",
      "first_para": "Framework aberto para orquestração de agentes multi-linguagem com suporte a Go, Python e TypeScript.",
      "objective": "Fornecer uma base mínima para construção de agentes que combinam LLMs, ferramentas determinísticas e memória."
    },
    "services/agent/README.md": {
      "file": "services/agent/README.md",
      "title": "Agent Service",
      "first_para": "Serviço responsável pelo ciclo de vida de agentes (criação, execução, monitoramento).",
      "objective": "Gerenciar agentes, expor APIs gRPC/REST e publicar eventos para o bus de mensagens."
    },
    "services/recipe/README.md": {
      "file": "services/recipe/README.md",
      "title": "Recipe Service",
      "first_para": "Serviço que interpreta arquivos YAML declarativos e instancia execuções de recipes.",
      "objective": "Separar a definição declarativa (recipes) da execução orquestrada, permitindo versionamento e reprocesso."
    }
  },
  "tech_stats": {
    ".go": 180,
    ".proto": 6,
    ".sql": 3,
    ".md": 8,
    ".yaml": 4,
    ".dockerfile": 3,
    "(none)": 5
  },
  "tree": [
    "baseron",
    "  cmd",
    "    agent",
    "    recipe",
    "  internal",
    "    agent",
    "    recipe",
    "    infra",
    "  proto",
    "    agent",
    "      v1",
    "  db",
    "    migrations",
    "  docs",
    "    decisions",
    "  Makefile",
    "  go.mod",
    "  README.md"
  ],
  "notable_configs": []
}


```

LLM_SUMMARY.md
```markdown
# LLM Scan Summary

- **Root:** /home/richardanchieta/repos/baseron  
- **Generated at:** 2025-09-14T18:00:00Z

---

## Go Modules

- **go.mod** → `github.com/richardanchieta/baseron`  
  Requires:  
  - go.uber.org/fx  
  - github.com/nats-io/nats.go  
  - google.golang.org/protobuf  
  - github.com/sqlc-dev/sqlc  
  - github.com/jackc/pgx/v5  

---

## Proto Definitions

- **proto/agent/v1/agent.proto**  
  Package: `agent.v1`  
  Services: `AgentService`  
  RPCs: `CreateAgent`, `ListAgents`, `ExecuteRecipe`

---

## Makefile Targets

`build`, `lint`, `test`, `proto-gen`, `sqlc-gen`, `docker-build`, `docker-run`

---

## Dockerfiles

- Dockerfile  
- services/agent/Dockerfile  
- services/recipe/Dockerfile  

---

## SQL Migrations

- db/migrations/0001_init_schema.sql  
- db/migrations/0002_agents.sql  
- db/migrations/0003_recipes.sql  

---

## ADRs (Architecture Decision Records)

- **adr-001-monorepo.md** — *Adotar monorepo com Go*  
  Optamos por monorepo para compartilhar módulos internos (proto, libs) e facilitar CI/CD unificado.

- **adr-002-agent-recipes.md** — *Recipes declarativos em YAML*  
  Definição de recipes como YAML declarativo, permitindo instanciar execuções sem acoplamento de lógica ao core.

---

## Environment Examples

- .env.example

---

## Licenses

- LICENSE

---

## READMEs

- README.md  
- services/agent/README.md  
- services/recipe/README.md  

---

## README Summaries

### README.md
- **Title:** Baseron  
- **Objective:** Fornecer uma base mínima para construção de agentes que combinam LLMs, ferramentas determinísticas e memória.  
- **Summary:** Framework aberto para orquestração de agentes multi-linguagem com suporte a Go, Python e TypeScript.

### services/agent/README.md
- **Title:** Agent Service  
- **Objective:** Gerenciar agentes, expor APIs gRPC/REST e publicar eventos para o bus de mensagens.  
- **Summary:** Serviço responsável pelo ciclo de vida de agentes (criação, execução, monitoramento).

### services/recipe/README.md
- **Title:** Recipe Service  
- **Objective:** Separar a definição declarativa (recipes) da execução orquestrada, permitindo versionamento e reprocesso.  
- **Summary:** Serviço que interpreta arquivos YAML declarativos e instancia execuções de recipes.

---

## Tech Stats

- .go → 180  
- .proto → 6  
- .sql → 3  
- .md → 8  
- .yaml → 4  
- .dockerfile → 3  
- (none) → 5  

---

## Tree (depth=3)

    baseron
      cmd
        agent
        recipe
      internal
        agent
        recipe
        infra
      proto
        agent
          v1
      db
        migrations
      docs
        decisions
      Makefile
      go.mod
      README.md

---

## Notable Configs

(none)
```


## Estrutura do Projeto

```bash
llm-scan-tool/
  cmd/llm-scan/         # CLI principal
  internal/collect/     # Parsers e coletores (Go, proto, SQL, etc.)
  internal/render/      # Renderização para Markdown/JSON
  internal/files/       # Utilitários de leitura de arquivos
  docs/decisions/       # ADRs do próprio llm-scan (opcional)
  Makefile
  go.mod
```

## .gitignore

Veja [.gitignore](.gitignore)
 incluso para ignorar binários, caches e relatórios (`LLM_SUMMARY.md*`).

## Roadmap

- [x] Parsing de Go, Protobuf, SQL e Docker.

- [x] Resumos automáticos de READMEs.

- [ ] Suporte a múltiplas linguagens (Python, JS/TS).

- [ ] Enriquecimento semântico com embeddings para queries de LLM.

- [ ] Integração direta com agentes de IA.

 ## Licença

 Apache 2.0 — veja [LICENSE](LICENSE)
