# Formato: Documentação de Telas (Views)

> Este arquivo define o formato de saída esperado para o artefato `tela.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Estrutura obrigatória do `tela.md` (5 seções, nesta ordem)

```markdown
# <Nome de Exibição da Tela>

## Visão Geral

## Componentes

## Estados

## Considerações

## Referências Visuais
```

---

## Seção 1 — Visão Geral

**Campos obrigatórios:**

| Campo | Descrição |
|---|---|
| **Nome** | Nome de exibição da tela, em português, como aparece nos cenários ou PRD |
| **Slug** | Identificador canônico em kebab-case, sem acentos (ex: `pagina-de-cadastro`) |
| **URL** | Caminho da rota (ex: `/register`). Use `—` se não determinada pelos artefatos |
| **Objetivo** | 1–2 frases orientadas ao usuário: o que o usuário faz/consegue nessa tela |
| **Scenarios relacionados** | Lista dos títulos **exatos** do `.feature` que envolvem esta tela |

**Checklist:**
- [ ] O nome é consistente com o que aparece nos Scenarios e no PRD
- [ ] O objetivo é orientado ao usuário — não descreve implementação
- [ ] Todos os Scenarios que mencionam esta tela estão listados
- [ ] A URL é derivada de trecho explícito nos artefatos (ou marcada como `—`)

---

## Seção 2 — Componentes

**Subseções (inclua apenas as que se aplicam):**

### Campos de formulário
| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| Nome do campo | text / email / password / date / file / etc. | Sim / Não | O que o usuário informa ou seleciona |

### Botões
| Rótulo | Tipo | Comportamento |
|---|---|---|
| Texto do botão como aparece na UI | submit / button / link-button | O que acontece ao clicar |

### Links de navegação
| Rótulo | Destino | Contexto |
|---|---|---|
| Texto do link | URL ou nome da tela destino | Quando/onde aparece |

### Conteúdo e mensagens fixas
- `<texto exato ou descrição do conteúdo>`

**Checklist:**
- [ ] Todos os campos mencionados nos Scenarios estão na tabela
- [ ] Rótulos dos botões como aparecem na UI, não nomes técnicos
- [ ] Comportamento dos botões derivado dos `When` e `Then` dos Scenarios
- [ ] Links de navegação derivados dos `Then ... é levado para` dos Scenarios
- [ ] Sem nomes de componentes de biblioteca (não use `Input`, `Button`, `Link`)
- [ ] Sem código fonte ou props de componentes

---

## Seção 3 — Estados

**Formato:** Incluir diagrama `stateDiagram-v2` com todas as transições antes de descrever cada estado.

**Estados a documentar (inclua apenas os que se aplicam):**

- **Padrão (initial)** — o que o usuário vê ao acessar pela primeira vez
- **Carregamento (loading)** — o que o usuário vê durante operação assíncrona
- **Erro** — mensagens literais dos `Then` dos Scenarios, organizadas por causa
- **Sucesso** — o que o usuário vê após ação bem-sucedida
- **Vazio** — quando não há dados a exibir (listas, histórico, resultados de busca)

**Checklist:**
- [ ] Inclui diagrama `stateDiagram-v2` com todos os estados e transições
- [ ] Estado de Erro tem mensagens literais dos `Then` — não paráfrases
- [ ] Estado de Sucesso descreve o que o usuário vê/recebe
- [ ] Estado de Carregamento descrito quando há operação assíncrona
- [ ] Estado Padrão descreve o que o usuário vê ao acessar pela primeira vez

---

## Seção 4 — Considerações

**Subseções sugeridas:**

- **Validações** — regras derivadas dos Scenarios de erro e requirements
- **Acessibilidade** — aspectos identificados no contexto (labels, foco, leitores de tela)
- **Responsividade** — comportamento em diferentes tamanhos de tela, se mencionado
- **Outros** — considerações específicas não cobertas acima

**Checklist:**
- [ ] Validações derivadas dos Scenario Outline / Examples e dos requirements
- [ ] Sem suposições não rastreáveis a um artefato
- [ ] Acessibilidade mencionada apenas se há âncora nos artefatos

---

## Seção 5 — Referências Visuais

**Esta seção é SEMPRE deixada vazia pelo agente**, com exatamente estes 3 placeholders:

```markdown
## Referências Visuais

### Wireframe
_A preencher manualmente._

### Mockup
_A preencher manualmente._

### Protótipo interativo
_A preencher manualmente._
```

---

## Regras gerais de formato

| Regra | Valor |
|---|---|
| **Arquivo de saída** | `docs/features/<slug>/views/<nome-canonico>/tela.md` |
| **Idioma** | Português. Termos técnicos consagrados ficam em inglês (token, cookie, upload, etc.) |
| **Rastreabilidade** | Todo conteúdo rastreável a Scenario BDD, PRD ou Stories |
| **Sem código fonte** | Nenhum trecho de código, JSX, props ou nomes de componentes de biblioteca |
| **Rótulos** | Como aparecem nos cenários e no PRD — nunca inventados |
| **Mensagens de erro** | Cópias literais dos `Then` do `.feature` |
| **Separador de nome** | Kebab-case sem acentos para slugs e nomes de pasta |
