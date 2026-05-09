# Formato: Cenários BDD (Gherkin)

> Este arquivo define o formato de saída esperado para o artefato `scenarios.feature`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Estrutura obrigatória do arquivo `.feature`

```gherkin
Feature: <Nome da Feature>
  <Descrição opcional de uma linha explicando o objetivo da feature>

  Scenario: <Título do cenário>
    Given <pré-condição ou estado do sistema>
    When <ação realizada pelo usuário ou evento>
    Then <resultado observável esperado>
    And <resultado adicional, se necessário>

  Scenario: <Outro cenário>
    ...
```

---

## Critérios de qualidade por parte do cenário

### `Feature`
- [ ] O nome da feature é consistente com o PRD e as User Stories
- [ ] A descrição (opcional) é orientada ao negócio, não à implementação

### `Scenario`
- [ ] O título é descritivo e específico — identifica o que está sendo testado sem ler os passos
- [ ] Cada cenário descreve **um único comportamento** (não mistura happy path com erro no mesmo cenário)
- [ ] O conjunto inclui: 1 cenário de sucesso (happy path), 2–3 cenários de erro, ao menos 1 edge case
- [ ] Cenários de erro têm título no padrão: `<Ação> com <condição de falha>` (ex: "Login com senha incorreta")

### `Given`
- [ ] Descreve um estado do sistema ou pré-condição, não uma ação do usuário
- [ ] Usa linguagem de domínio, não termos técnicos de implementação
- [ ] Não menciona banco de dados, JWT, bcrypt, tabelas ou bibliotecas
- [ ] Se a pré-condição é óbvia (ex: "sistema está disponível"), pode ser omitida para manter o cenário limpo

### `When`
- [ ] Descreve uma única ação do usuário ou evento externo
- [ ] Usa verbos de ação no presente ("o entregador envia", "o sistema recebe")
- [ ] Não menciona implementação (não usa "chama endpoint", "faz POST", "executa query")

### `Then`
- [ ] Descreve o resultado do ponto de vista do usuário ou de um observador externo
- [ ] É verificável — dá para dizer "passou" ou "falhou" ao observar o sistema
- [ ] Não menciona estado interno do sistema que não seja observável externamente
- [ ] `And` é usado apenas para resultados adicionais do mesmo cenário, não para novas ações

---

## Critério de qualidade do conjunto de cenários

| Critério | Verificação |
|---|---|
| **Cobertura** | Cobre happy path, principais erros e ao menos um edge case |
| **Clareza** | Um QA consegue executar os testes apenas lendo os cenários |
| **Independência** | Cada cenário funciona isoladamente, sem depender da ordem de execução |
| **Linguagem de domínio** | Qualquer pessoa do time entende os cenários sem saber código |
| **Rastreabilidade** | Cada cenário tem origem clara no PRD ou nas User Stories |

---

## Tipos de cenário e quando criar cada um

- **Sucesso (happy path)** — Âncora: Fluxo Principal do PRD ou critérios de aceitação da estória de caminho feliz.
- **Erro de entrada** — Âncora: seção Objetivos com "Rejeitar"/"Validar" ou critérios de estórias de validação.
- **Estado inválido** — Âncora: seção Fluxos Alternativos do PRD.
- **Edge case** — Âncora: seção Riscos do PRD ou critérios de aceitação de estórias de segurança.

---

## Regras gerais de formato

- **Arquivo de saída:** `docs/features/<slug>/scenarios.feature`
- **Indentação:** 2 espaços para os passos dentro de `Scenario`
- **Idioma dos passos:** português. Termos técnicos consagrados ficam em inglês (token, JWT, cookie, OAuth)
- **Separação:** linha em branco entre cenários
- **Sem comentários inline** nos passos — o passo deve ser autoexplicativo
- **Sem `Background`** a menos que explicitamente solicitado — preferir `Given` explícito em cada cenário
