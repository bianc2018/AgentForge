# Formato: Estórias de Usuário

> Este arquivo define o formato de saída esperado para o artefato `stories.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Estrutura obrigatória de cada estória

```markdown
## Estoria <N> – <Título>

Como <persona>
Eu quero <ação ou capacidade>
Para que <benefício ou resultado>

_Critérios de aceitação_:

- <critério 1>
- <critério 2>
- <critério N>
```

---

## Critérios de qualidade por parte da estória

### Cabeçalho (`Como / Eu quero / Para que`)

**Checklist de qualidade:**
- [ ] **Como** — é uma persona específica, não genérica ("entregador com conta criada", não "o usuário" ou "alguém")
- [ ] **Como** — a persona é consistente com a seção Usuário-Alvo do PRD da feature
- [ ] **Eu quero** — expressa uma capacidade ou ação do usuário, não um detalhe técnico de implementação
- [ ] **Eu quero** — usa linguagem de negócio ("permanecer autenticado"), não técnica ("receber refresh token")
- [ ] **Para que** — expressa um benefício real ou resultado concreto para o usuário
- [ ] **Para que** — não é trivialmente óbvio (evitar "Para que eu possa usar o sistema")
- [ ] O trio "Como / Eu quero / Para que" forma uma frase coerente quando lida em sequência

### Critérios de aceitação

**Formato:** Lista de bullets. Cada item começa com "O sistema deve..." ou "O sistema não deve...".

**Checklist de qualidade:**
- [ ] Entre 2 e 6 critérios por estória
- [ ] Cada critério é verificável — dá para dizer "passou" ou "falhou" em um teste
- [ ] Cobre o caminho feliz (o que acontece quando tudo dá certo)
- [ ] Cobre pelo menos um cenário de validação ou erro relevante para esta estória
- [ ] Nenhum critério repete literalmente o "Eu quero" da estória
- [ ] Nenhum critério menciona detalhes de implementação (nome de função, estrutura de banco, biblioteca)
- [ ] Os critérios são coerentes com o PRD — não contradizem objetivos, fluxos ou fora do escopo

---

## Critério de qualidade do conjunto (INVEST adaptado)

| Letra | Princípio | Verificação |
|---|---|---|
| **I** | Independente | Cada estória pode ser implementada sem depender da sequência das outras |
| **N** | Negociável | O "como" e "para que" foram validados com o usuário — não inventados |
| **V** | Valiosa | Cada estória entrega valor perceptível ao usuário final |
| **E** | Estimável | Um desenvolvedor consegue estimar o esforço com base nos critérios |
| **S** | Small (pequena) | Cada estória cobre um único fluxo ou responsabilidade |
| **T** | Testável | Todos os critérios de aceitação são verificáveis por um teste |

---

## Tipos de estória e quando usar cada um

- **Happy path** — Âncora: Fluxo Principal do PRD ou itens de Objetivos com verbo "Permitir".
- **Validação** — Âncora: Objetivos do PRD com verbos "Validar", "Rejeitar" ou "Detectar".
- **Caso de falha** — Âncora: seção Fluxos Alternativos do PRD.
- **Segurança / restrição** — Âncora: seção Riscos ou Critérios de Sucesso quando mencionam segurança.

---

## Regras gerais de formato

- **Título da seção:** `## Estoria <N> – <Título>` (com travessão `–`, não hífen `-`)
- **Numeração:** sequencial a partir de 1, sem pular números
- **Tom:** orientado ao usuário, não à implementação. Um PM deve entender sem saber código.
- **Idioma:** português. Termos técnicos consagrados ficam em inglês (JWT, cookie, OAuth, bcrypt).
- **Título do arquivo:** `# Estórias de Usuário — <Nome da Feature>` (com travessão)
- **Separação:** linha em branco entre estórias
- **Critérios:** nunca deixar uma estória sem critérios de aceitação
