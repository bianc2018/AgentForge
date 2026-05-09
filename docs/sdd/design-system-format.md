# Formato: Design System

> Este arquivo define o formato de saída esperado para os artefatos em `docs/design-system/`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Ordem de criação obrigatória

```
colors.md → typography.md → spacing.md → components.md → themes.md
```

Cada arquivo referencia os anteriores. Criar fora de ordem gera inconsistências.

---

## Estrutura por arquivo

### colors.md

```markdown
# Cores

## Paleta Semântica
| Token | Valor | Classe Tailwind | Uso |
|-------|-------|----------------|-----|
| primary | #... | `bg-primary` / `text-primary` | Ações principais, CTAs |
| primary-foreground | ... | `text-primary-foreground` | Texto sobre fundo primary |
| secondary | ... | `bg-secondary` | Ações secundárias |
| destructive | ... | `bg-destructive` | Ações destrutivas, erros |
| muted | ... | `bg-muted` | Fundos sutis, desabilitados |
| accent | ... | `bg-accent` | Destaques, hover states |
| background | ... | `bg-background` | Fundo principal da página |
| foreground | ... | `text-foreground` | Texto principal |
| border | ... | `border-border` | Bordas de componentes |
| ring | ... | `ring` | Foco visível (acessibilidade) |

## Paleta de Suporte
...

## Regras de Uso
- Nunca usar valores hex diretamente no código — sempre via token semântico

## Acessibilidade
- primary sobre background: contraste <N>:1 (mínimo WCAG AA: 4.5:1)
```

**Checklist:**
- [ ] Tokens semânticos cobrem: ação principal, ação destrutiva, fundo, texto principal e borda
- [ ] Cada token tem valor, referência de uso e descrição
- [ ] Contraste mínimo WCAG AA declarado para combinações primárias
- [ ] Regra explícita de não usar valores de cor diretamente no código
- [ ] Paleta derivada das configurações encontradas no projeto — não inventada

---

### typography.md

```markdown
# Tipografia

## Famílias de Fonte
| Família | Variável CSS / Classe | Uso |
|---------|----------------------|-----|
| <nome> | `font-sans` | Texto geral |
| <nome> | `font-mono` | Código, dados técnicos |

## Escala Tipográfica
| Nível | Classe Tailwind | Tamanho | Peso | Line Height | Uso |
|-------|----------------|---------|------|-------------|-----|
| H1 | `text-3xl font-bold` | 30px | 700 | 1.25 | Título principal de página |
| Body | `text-base` | 16px | 400 | 1.5 | Texto corrido |
| Caption | `text-xs` | 12px | 400 | 1.4 | Legendas, tooltips |

## Regras de Uso
...
```

**Checklist:**
- [ ] Ao menos uma família de fonte declarada com referência de uso
- [ ] Escala com ao menos 5 níveis
- [ ] Cada nível tem referência de uso, tamanho em px, peso e uso claro

---

### spacing.md

```markdown
# Espaçamento

## Escala Base
| Token | Valor | Referência de uso | Uso típico |
|-------|-------|------------------|------------|
| xs | 4px | `p-1` | Espaçamento mínimo interno |
| sm | 8px | `p-2` | Espaçamento interno de componentes pequenos |
| md | 12px | `p-3` | Padding de inputs |
| lg | 16px | `p-4` | Padding padrão de cards |
| xl | 24px | `p-6` | Padding de containers principais |

## Regras de Uso
- Nunca usar valores arbitrários — ajustar para o token mais próximo

## Grade e Layout
- Container máximo: <valor>
```

**Checklist:**
- [ ] Escala base declarada — derivada das configurações do projeto
- [ ] Ao menos 8 tokens com valor em px, referência de uso e uso típico
- [ ] Regras claras de quando usar cada faixa da escala
- [ ] Regra explícita contra valores arbitrários

---

### components.md

```markdown
# Componentes

## Componentes Base

### Button
**Variantes:**
| Variante | Referência | Quando usar |
|----------|-----------|-------------|
| primary | `variant="default"` | Ação principal — máximo 1 por seção |
| destructive | `variant="destructive"` | Ações irreversíveis |

**Estados:** Loading, Disabled

### Input
**Estados obrigatórios:** Default, Focus, Error, Disabled, Loading

## Componentes Compostos Reutilizáveis
...
```

**Checklist:**
- [ ] Cada componente da biblioteca detectada tem entrada
- [ ] Button tem todas as variantes com quando usar cada uma
- [ ] Todos os estados de Input estão declarados
- [ ] Regra de acessibilidade de label está explícita
- [ ] Componentes compostos têm interface, uso e regras declarados

---

### themes.md

```markdown
# Temas

## Temas Disponíveis
| Tema | Classe CSS | Ativo quando |
|------|-----------|--------------|
| Light | `:root` | Padrão |
| Dark | `.dark` | Preferência do sistema |

## Variáveis CSS por Tema
| Variável | Light | Dark | Token semântico |
|----------|-------|------|----------------|
| `--background` | ... | ... | `background` |

## Como Adicionar um Novo Tema
...

## Como o Projeto Ativa/Troca o Tema
...
```

**Checklist:**
- [ ] Todos os temas disponíveis declarados
- [ ] Tabela de variáveis CSS cobre todos os tokens de `colors.md`
- [ ] Instrução de como adicionar novo tema
- [ ] Documentado como o projeto ativa/troca o tema

---

## Regras gerais de formato

- **Idioma:** português para títulos e descrições; nomes de classes, tokens e props em inglês
- **Sem valores inventados:** tudo derivado das configurações e código encontrados no projeto
- **Referências cruzadas:** `typography.md` referencia cores de `colors.md`; `components.md` referencia ambos
- **Tom:** prescritivo — "usar X para Y", "nunca Z"
