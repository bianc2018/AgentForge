# Formato: Requisitos Funcionais (EARS)

> Este arquivo define o formato de saída esperado para o artefato `requirements.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Os quatro padrões EARS

| Padrão | Quando usar | Sintaxe |
|--------|-------------|---------|
| **Ubíquo** | Comportamento sempre obrigatório | `The system shall <behavior>.` |
| **Evento** | Evento externo dispara comportamento | `When <event>, the system shall <behavior>.` |
| **Estado** | Sistema em estado específico | `While <state>, the system shall <behavior>.` |
| **Indesejado** | Condição de falha ou erro | `If <condition>, the system shall <behavior>.` |

> Regra: erros e falhas usam `If`, nunca `When`.

---

## Estrutura do arquivo

```markdown
# Requisitos Funcionais — <Nome da Feature>

## <Grupo 1, ex: Fluxo Principal>

**REQ-1**: When <event>, the system shall <behavior>.

> Fonte: Fluxo Principal do PRD / Cenário BDD "Título do cenário"

## <Grupo 2, ex: Validação de Entrada>

**REQ-3**: If <condition>, the system shall <behavior>.

> Fonte: Fluxo Alternativo Y do PRD / Estória Z, critério N
```

---

## Checklist por requisito

**Clareza:**
- [ ] Descreve um único comportamento
- [ ] Usa `shall` — nunca "should", "may", "can" ou "must"
- [ ] Sujeito é sempre "the system"

**Testabilidade:**
- [ ] Possível escrever um teste que prove atendimento ou violação
- [ ] Sem termos vagos: "corretamente", "adequada", "quando necessário", "rápido"
- [ ] Comportamento observável externamente

**Independência de tecnologia:**
- [ ] Sem linguagem, framework, biblioteca ou banco de dados
- [ ] Sem JWT, bcrypt, MySQL, Redis, Next.js, React
- [ ] Vocabulário de domínio do negócio

**Rastreabilidade:**
- [ ] Campo `> Fonte:` identifica a origem (PRD, Story ou cenário BDD)
- [ ] Cada cenário BDD tem ao menos um requisito correspondente

---

## Critério de qualidade do conjunto

| Critério | Verificação |
|---|---|
| Cobertura | Todo cenário BDD tem ao menos 1 requisito |
| Completude | Todo critério de aceitação das Stories está coberto |
| Consistência | Vocabulário uniforme com o PRD |
| Não-redundância | Dois requisitos não descrevem o mesmo comportamento |
| Escopo | Nenhum requisito cobre itens do "Fora do Escopo" do PRD |

---

## Regras de formato

- **Arquivo:** `docs/features/<slug>/requirements.md`
- **Idioma dos requisitos:** inglês (padrão EARS)
- **Comentários e anotações:** português (`> Fonte:`, títulos `##`)
- **Numeração:** sequencial global — REQ-1, REQ-2... (não reinicia por grupo)
- **Agrupamento:** cabeçalhos `##` por categoria (Fluxo Principal, Validação, Segurança...)
