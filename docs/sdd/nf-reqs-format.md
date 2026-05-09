# Formato: Requisitos Não Funcionais

> Este arquivo define o formato de saída esperado para o artefato `nf-requirements.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Características de bons RNFs

RNFs devem ser:

- **Mensuráveis**: expressos com métricas objetivas (ex.: "< 2s", "99,9% uptime")
- **Testáveis**: possível validar com um teste objetivo
- **Específicos**: vinculados a um contexto ou operação clara
- **Rastreáveis**: derivados de uma fonte (PRD, Story, BDD, requisito funcional)

Evite termos vagos: "rápido", "seguro", "alta disponibilidade", "boa performance".

---

## Categorias de RNFs

- **Performance** — Tempo de resposta e throughput de operações do sistema
- **Escalabilidade** — Capacidade de suportar crescimento de usuários ou volume
- **Disponibilidade** — Uptime e tolerância a falhas
- **Segurança** — Autenticação, autorização, criptografia e proteção de dados
- **Observabilidade** — Logs, métricas e rastreamento
- **Usabilidade** — Tempo de aprendizado e acessibilidade

---

## Estrutura obrigatória do arquivo

```markdown
# Requisitos Não Funcionais — <Nome da Feature>

## Performance

**NFR-1**: Em condições normais, o sistema deve <comportamento> em até <métrica> para <condição>.

> Fonte: <âncora nos artefatos>

## Segurança

**NFR-2**: O sistema deve <comportamento>.

> Fonte: <âncora nos artefatos>
```

---

## Formato de cada RNF

```
[Condição opcional,] o sistema deve [comportamento] [métrica].
```

Exemplos bem escritos:
- `Em condições normais, o sistema deve responder requisições de autenticação em até 2 segundos para 95% dos casos.`
- `O sistema deve suportar até 5.000 usuários simultâneos sem degradação perceptível.`
- `O sistema deve ter disponibilidade mínima de 99,9% ao mês.`
- `O sistema deve registrar em log todas as tentativas de autenticação, incluindo as falhas.`
- `O sistema deve bloquear temporariamente contas após 5 tentativas de login malsucedidas consecutivas.`

---

## Critérios de qualidade por RNF

**Mensurabilidade:**
- [ ] O RNF contém uma métrica objetiva (número, percentagem, tempo, quantidade)
- [ ] A métrica é verificável sem ambiguidade
- [ ] A condição de medição está especificada (carga normal? pico? percentil?)

**Testabilidade:**
- [ ] É possível escrever um teste que prove que o RNF foi atendido ou violado
- [ ] O RNF não usa termos vagos: "rápido", "seguro", "performático", "alto"
- [ ] O comportamento descrito é observável externamente

**Rastreabilidade:**
- [ ] O campo `> Fonte:` identifica de onde o RNF foi derivado (PRD, Story, BDD ou requisito funcional)
- [ ] O RNF está vinculado a uma categoria coerente

---

## Critério de qualidade do conjunto

| Critério | Verificação |
|---|---|
| **Cobertura por categoria** | As categorias relevantes para a feature estão cobertas |
| **Mensurabilidade** | Todo RNF tem métrica objetiva |
| **Consistência** | O vocabulário é uniforme com o PRD e os requisitos funcionais |
| **Não-redundância** | Dois RNFs não descrevem a mesma restrição de qualidade |
| **Relevância** | Nenhum RNF é genérico demais ou desconectado da feature |

---

## Regras gerais de formato

- **Arquivo de saída:** `docs/features/<slug>/nf-requirements.md`
- **Idioma dos RNFs:** português
- **Numeração:** sequencial global — NFR-1, NFR-2, NFR-3... (não reinicia por categoria)
- **Agrupamento:** use cabeçalhos `##` por categoria
- **Separação:** linha em branco entre RNFs
