# Formato: PRD (Product Requirements Document)

> Este arquivo define o formato de saída esperado para o artefato `prd.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Estrutura obrigatória (10 seções, nesta ordem)

1. Visão Geral
2. Problema
3. Usuário-Alvo
4. Objetivos
5. Critérios de Sucesso
6. Fora do Escopo
7. Fluxo Principal
8. Fluxo Alternativo
9. Dependências
10. Riscos

---

## Checklists por seção

### 1. Visão Geral
_Formato: texto corrido, 1-3 frases._
- [ ] Responde "o quê" (o que o usuário consegue fazer)
- [ ] Responde "como" (mecanismo principal, ex: JWT, cookie httpOnly)
- [ ] Sem detalhes de implementação (nomes de funções, estrutura de tabelas)
- [ ] Não começa com frase meta ("Este PRD descreve...")

### 2. Problema
_Formato: texto corrido, 1-3 parágrafos._
- [ ] Articula a situação atual (o que acontece hoje sem a feature)
- [ ] Explica o impacto concreto para o usuário
- [ ] Não descreve a solução

### 3. Usuário-Alvo
_Formato: 1-2 frases específicas._
- [ ] Específico (não "qualquer usuário")
- [ ] Menciona pré-condições se houver (ex: "com conta previamente criada via register-user")

### 4. Objetivos
_Formato: lista numerada, cada item começa com verbo no infinitivo._
- [ ] 3 a 7 itens
- [ ] Cada item verificável (dá para dizer "implementado ou não")
- [ ] Nenhum item se repete ou sobrepõe outro
- [ ] Não confunde objetivo com critério de sucesso

### 5. Critérios de Sucesso
_Formato: tabela `Critério | Medida`._
- [ ] 3 a 6 linhas
- [ ] Coluna "Medida" é quantificável ou verificável (%, tempo, boolean)
- [ ] Cobre ao menos: happy path, erro principal, e um requisito não-funcional

### 6. Fora do Escopo
_Formato: lista de bullets._
- [ ] Ao menos 3 itens
- [ ] Itens que o usuário poderia razoavelmente esperar mas não serão entregues
- [ ] Sem itens óbvios que nunca estariam no escopo

### 7. Fluxo Principal
_Formato: diagrama `flowchart LR` Mermaid, cada nó é uma ação ou decisão distinta._
- [ ] Começa no ponto de entrada do usuário
- [ ] Termina no resultado final
- [ ] Sem bifurcações (erros ficam no Fluxo Alternativo)
- [ ] Cada passo é uma ação ou decisão distinta
- [ ] Usa `flowchart LR` com nós descritivos em linguagem de produto (sem código)

### 8. Fluxo Alternativo
_Formato: mesmo formato do Fluxo Principal (`flowchart LR`). Um sub-diagrama por fluxo alternativo distinto. Pode ser "N/A" com justificativa._
- [ ] Presente se houver fluxo secundário que o produto precisa garantir
- [ ] Claramente diferente do Fluxo Principal
- [ ] Mesmo nível de detalhe

### 9. Dependências
_Formato: lista de bullets._
- [ ] Menciona features do produto que precisam existir antes
- [ ] Menciona bibliotecas externas não-triviais
- [ ] Menciona dados ou estruturas de banco que precisam existir
- [ ] Sem dependências genéricas óbvias (ex: "Next.js", "banco de dados")

### 10. Riscos
_Formato: tabela `Risco | Mitigação`._
- [ ] 2 a 5 linhas
- [ ] Cada risco específico desta feature
- [ ] Cada mitigação concreta e acionável
- [ ] Cobre ao menos um risco de segurança se feature lidar com autenticação ou dados sensíveis

---

## Regras de formato

- **Tom:** orientado a produto. Um PM deve entender sem saber código.
- **Idioma:** português. Termos técnicos consagrados (JWT, OAuth, bcrypt) ficam em inglês.
- **Título:** `# PRD — <Nome da Feature>` (travessão, não hífen)
- **Seções vazias:** nunca em branco — se não se aplica, escrever por quê.
