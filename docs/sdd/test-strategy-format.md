# Formato: Estratégia de Testes

> Este arquivo define o formato de saída esperado para o artefato `test-strategy.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Estrutura obrigatória (6 seções, nesta ordem)

```markdown
# Estratégia de Testes — <Nome da Feature>

## 1. Testes Unitários
...

---

## 2. Testes de Integração
...

---

## 3. Testes E2E Gherkin
...

---

## 4. Testes de Performance
...

---

## 5. Testes de Segurança
...

---

## Resumo de Cobertura
...
```

---

## Formato por tipo de teste

### 1. Testes Unitários

```markdown
### UT-<N>: <NomeDoComponente>.<método>()

- **O que testa:** <comportamento esperado>
- **Casos cobertos:**
    - Caminho feliz: <descrição>
    - <Condição de erro 1>: <comportamento esperado>
    - <Condição de borda>: <comportamento esperado>
- **Mocks necessários:** <lista ou "nenhum — domínio puro">
- **Rastreabilidade:** <REQ-N> · <NFR-N>
```

**Checklist:**
- [ ] Um UT por método de domínio identificado no `design.md`
- [ ] Cada UT cobre ao menos: caminho feliz + cada condição de erro do método
- [ ] Nenhum UT depende de banco, HTTP ou serviço externo
- [ ] Mocks estão declarados quando necessário
- [ ] Cada UT rastreia ao menos um REQ

---

### 2. Testes de Integração

```markdown
### IT-<N>: <NomeDoComponente> — <método ou comportamento>

- **O que testa:** <comportamento com dependência real>
- **Dependências reais usadas:** <banco de teste | Redis de teste | serviço externo mockado>
- **Casos cobertos:**
    - <Caso 1>: <comportamento esperado>
    - <Caso 2>: <comportamento esperado>
- **Setup necessário:** <estado inicial do banco/Redis para o teste>
- **Rastreabilidade:** <REQ-N> · <NFR-N>
```

**Checklist:**
- [ ] Um IT por repository (cobrindo seus métodos principais)
- [ ] Um IT por adapter de serviço externo
- [ ] Cada IT especifica quais dependências são reais e quais são mockadas
- [ ] Setup/teardown do estado externo está descrito
- [ ] Comportamentos de falha de dependência estão cobertos

---

### 3. Testes E2E Gherkin

```markdown
### GH-<N>: Scenario "<nome exato do Scenario>"

- **Arquivo:** `docs/features/<slug>/scenarios.feature`
- **Step definitions necessários:**
    - `Given <texto exato do step>` → <o que o step deve fazer>
    - `When <texto exato do step>` → <o que o step deve fazer>
    - `Then <texto exato do step>` → <o que o step deve verificar>
- **Steps reutilizáveis de outros Scenarios:** <lista ou "nenhum">
- **Estado inicial necessário:** <dados no banco, mocks ativos>
- **Rastreabilidade:** <REQ-N> · <NFR-N>
```

**Checklist:**
- [ ] Um GH por Scenario do arquivo `.feature` — sem exceções
- [ ] Cada GH é explicitamente automatizável
- [ ] Cada step definition está descrito com sua implementação esperada
- [ ] Steps reutilizáveis entre Scenarios estão identificados
- [ ] Estado inicial necessário para cada cenário está descrito

---

### 4. Testes de Performance

```markdown
### PT-<N>: <Título descritivo>

- **O que mede:** <métrica: latência p95, tempo de operação, throughput>
- **Threshold:** <valor do NFR, ex: ≤ 30s, ≥ 100ms, ≤ 200ms p95>
- **Método de medição:** <benchmark local | teste de carga | medição em CI>
- **Número de execuções:** <ex: 10 execuções, 100 rps por 60s>
- **Rastreabilidade:** <NFR-N>
```

**Checklist:**
- [ ] Um PT por NFR com critério mensurável
- [ ] Threshold derivado diretamente do NFR (não inventado)
- [ ] Método de medição é executável em CI
- [ ] Número de execuções é suficiente para resultado estatisticamente significativo

---

### 5. Testes de Segurança

```markdown
### ST-<N>: <Título descritivo do ataque ou vulnerabilidade>

- **O que verifica:** <comportamento de segurança esperado>
- **Vetor de ataque simulado:** <ex: enumeração de contas, brute force, replay de token>
- **Casos cobertos:**
    - <Caso 1>: <comportamento esperado do sistema>
    - <Caso 2>: <comportamento esperado do sistema>
- **Rastreabilidade:** <NFR-N> · <Risco do PRD>
```

**Checklist:**
- [ ] Um ST por NFR de segurança
- [ ] Um ST por risco do PRD que tem mitigação técnica verificável
- [ ] Cada ST descreve o vetor de ataque simulado (não apenas "testar segurança")
- [ ] O comportamento esperado é específico (status HTTP, mensagem, ausência de dado sensível)

---

### Resumo de Cobertura

```markdown
## Resumo de Cobertura

| Requisito | Unitário | Integração | E2E Gherkin | Performance | Segurança |
| --------- | -------- | ---------- | ----------- | ----------- | --------- |
| REQ-1     | UT-1     | IT-1, IT-2 | GH-1        | PT-1        | —         |
| NFR-1     | —        | —          | —           | PT-1        | —         |
```

**Checklist:**
- [ ] Cada REQ tem ao menos 1 teste E2E Gherkin E ao menos 1 teste unitário ou de integração
- [ ] Cada NFR mensurável tem ao menos 1 teste de performance
- [ ] Cada NFR de segurança tem ao menos 1 teste de segurança
- [ ] Nenhuma linha da tabela está completamente vazia

---

## Cobertura mínima obrigatória

- [ ] Cada método de domínio do `design.md` tem ao menos 1 UT
- [ ] Cada repository do `design.md` tem ao menos 1 IT
- [ ] Cada adapter de serviço externo do `design.md` tem ao menos 1 IT
- [ ] Cada Scenario do `.feature` tem 1 GH correspondente
- [ ] Cada NFR com valor mensurável tem 1 PT
- [ ] Cada NFR de segurança e cada risco mitigável do PRD tem 1 ST

---

## Regras gerais de formato

- **IDs por tipo:** UT-N, IT-N, GH-N, PT-N, ST-N — sequenciais dentro de cada tipo
- **Rastreabilidade sempre presente:** todo teste aponta ao REQ, NFR ou Scenario que originou
- **Sem código:** o `test-strategy.md` descreve o que testar, não como implementar
- **Separadores:** `---` entre seções de tipo
- **Idioma:** português para títulos e descrições; nomes de componentes e métodos em inglês
