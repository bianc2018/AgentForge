# Formato: Design Técnico

> Este arquivo define o formato de saída esperado para o artefato `design.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Estrutura obrigatória (6 seções, nesta ordem)

1. Visão Geral Técnica
2. Arquitetura de Componentes
3. Modelo de Dados
4. API / Contratos
5. Fluxo de Execução
6. Decisões Técnicas

---

## Checklists por seção

### 1. Visão Geral Técnica
_Formato: texto corrido, 2-4 frases._
- [ ] Descreve a abordagem técnica principal (não o que a feature faz ao usuário)
- [ ] Menciona tecnologias-chave (ex: OTP, bcrypt, Redis, serviço de email)
- [ ] Referencia camada arquitetural ou padrão (ex: hexagonal)
- [ ] Consistente com as restrições da `docs/constitution.md`

### 2. Arquitetura de Componentes
_Formato: diagrama `flowchart TD` seguido de um bloco `### NomeComponente` por componente, com camada, responsabilidade e dependências._
- [ ] Inclui diagrama `flowchart TD` mostrando dependências entre todos os componentes (infra → aplicação → domínio)
- [ ] Cada componente tem responsabilidade única e clara
- [ ] Camadas seguem a arquitetura da `docs/constitution.md`
- [ ] Dependências apontam para dentro (infra → aplicação → domínio)
- [ ] Todos os requisitos funcionais são endereçados por ao menos um componente
- [ ] Serviços externos isolados em adapters de infraestrutura
- [ ] Novos componentes sinalizados explicitamente

### 3. Modelo de Dados
_Formato: diagrama `erDiagram` seguido de tabela `Campo | Tipo | Descrição` por entidade, com relações explícitas._
- [ ] Inclui diagrama `erDiagram` mostrando todas as entidades e suas relações
- [ ] Entidades deriváveis dos requisitos ou cenários BDD — nenhuma inventada
- [ ] Campos de expiração e uso único presentes onde NFRs exigem
- [ ] Campos de auditoria declarados se o padrão do projeto exige
- [ ] Sem DDL ou migração (apenas modelo lógico)

### 4. API / Contratos
_Formato: `### MÉTODO /path` com autenticação, request body, response 200 e tabela de erros._
- [ ] Cada endpoint corresponde a ao menos um requisito ou cenário BDD
- [ ] Payloads de entrada com campos e tipos definidos
- [ ] Todos os cenários de erro têm código HTTP correspondente
- [ ] Bloqueios por rate limit (NFRs) têm código 429
- [ ] Autenticação de cada endpoint declarada

### 5. Fluxo de Execução
_Formato: `### Fluxo: <nome do Scenario>`, passos numerados com componente responsável, seguidos de `sequenceDiagram` Mermaid._
- [ ] Um fluxo por Scenario do `.feature`
- [ ] Caminho feliz passo a passo do request até a resposta
- [ ] Cada passo nomeia o componente responsável
- [ ] Cada fluxo inclui `sequenceDiagram` com participantes e mensagens correspondendo aos passos numerados
- [ ] Fluxos alternativos cobrem todos os cenários BDD de erro
- [ ] Nenhum passo vago ("o sistema processa" sem especificar o quê)

### 6. Decisões Técnicas
_Formato: `### DT-N: <título>` com problema, alternativas, decisão, justificativa e requisito relacionado._
- [ ] Apenas decisões com trade-off real (não as óbvias)
- [ ] Ao menos duas alternativas por decisão
- [ ] Justificativa menciona o trade-off
- [ ] Cada decisão rastreada a um REQ ou NFR
- [ ] Nenhuma decisão contradiz a `docs/constitution.md`

---

## Regras de formato

- **Idioma:** português. Nomes de componentes, campos e endpoints ficam em inglês.
- **Rastreabilidade:** sempre referencie REQ-N ou NFR-N que motivou a decisão.
- **Sem código:** sem código fonte; pseudocódigo aceitável apenas em Fluxo de Execução.
- **Sem repetição:** se a informação está no PRD ou nos requisitos, referencie — não copie.
- **Separadores:** usar `---` entre seções.
