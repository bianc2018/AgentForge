# Formato: Constitution

> Este arquivo define o formato de saída esperado para o artefato `constitution.md`.
> Edite este arquivo para adaptar o formato ao seu projeto ou empresa.

---

## Estrutura obrigatória (5 seções, nesta ordem)

1. Purpose (Propósito)
2. Must Do (Obrigações)
3. Ask Before Proceeding (Pergunte Antes de Prosseguir)
4. Never Do (Proibições)
5. Enforcement (Execução)

---

## Seções e critérios de qualidade

### 1. Purpose

**Formato:** Texto corrido (sem bullets, sem tabelas).

**Checklist de qualidade:**
- [ ] Declara que o documento define regras não-negociáveis
- [ ] Menciona o escopo (projeto, sistema, feature)
- [ ] Instrui que ambiguidades devem ser resolvidas explicitamente — nunca assumidas
- [ ] Tem no máximo 3 frases

---

### 2. Must Do

**Formato:** Lista numerada. Cada item começa com "All [sujeito] must [verbo]" (ou equivalente em português).

**Checklist de qualidade:**
- [ ] Entre 4 e 8 regras
- [ ] Cada regra cobre um aspecto diferente: camadas, erros, validação, logs, rastreabilidade
- [ ] Cada regra é verificável — dá para dizer "esta implementação segue ou não esta regra"
- [ ] Cada regra usa linguagem imperativa ("must", "deve")
- [ ] Nenhuma regra repete ou sobrepõe outra da mesma seção ou de outra seção

**Cobertura mínima esperada:**
- Separação de camadas (business logic, transport, persistence)
- Propagação de erros com estrutura padronizada
- Validação de entradas no limite do sistema
- Logging de operações críticas
- Rastreabilidade para requisito ou BDD

---

### 3. Ask Before Proceeding

**Formato:** Lista numerada. Cada item começa com "If [condição], [ação]."

**Checklist de qualidade:**
- [ ] Entre 3 e 6 regras
- [ ] Cada regra tem uma condição de disparo clara (se X então perguntar)
- [ ] Cobre ao menos: requisito ambíguo, decisão arquitetural com múltiplas opções, mudança de contrato/API, conflito com a própria constituição
- [ ] Cada ação instrui explicitamente a parar e pedir clareza — não continuar com suposições
- [ ] Nenhuma regra é vaga demais para ser acionada (ex: "se houver dúvida" não serve)

---

### 4. Never Do

**Formato:** Lista numerada. Cada item começa com "Never [verbo]" (ou equivalente em português).

**Checklist de qualidade:**
- [ ] Entre 4 e 8 regras
- [ ] Cada regra é uma proibição absoluta, sem exceções
- [ ] Cobre ao menos: acesso direto a dados em camadas erradas, lógica de negócio fora do domínio, erros silenciosos, acoplamento com frameworks, suposições de requisitos
- [ ] Nenhuma regra se contradiz com uma regra do Must Do
- [ ] Cada regra descreve um anti-pattern concreto, não uma generalização

---

### 5. Enforcement

**Formato:** Lista numerada. Cada item descreve uma consequência ou gate de validação.

**Checklist de qualidade:**
- [ ] Entre 2 e 4 regras
- [ ] Ao menos uma regra exige que planos de implementação descrevam conformidade com a constituição
- [ ] Ao menos uma regra especifica que violações invalidam a implementação e devem ser corrigidas antes do merge
- [ ] Ao menos uma regra bloqueia implementação quando há clareza faltando
- [ ] As consequências são concretas (invalidar, bloquear, exigir correção) — não vagas (recomendar, sugerir)

---

## Regras gerais de formato

- **Idioma:** português por padrão. Termos técnicos (controller, repository, adapter, middleware) ficam em inglês.
- **Numeração global:** as regras são numeradas sequencialmente ao longo de todo o documento (1, 2, 3...) — não reinicia por seção.
- **Sem seções vazias:** se uma seção não for aplicável, registrar explicitamente por quê.
- **Sem repetição:** se uma regra já foi dita em uma seção, não repita em outra.
- **Tom:** objetivo e direto. Sem justificativas longas — a regra deve ser autoexplicativa.
- **Separadores:** usar `---` entre seções para facilitar leitura.
