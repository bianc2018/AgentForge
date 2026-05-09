# Exemplo Anotado — Requisitos Não Funcionais

Feature de referência: **Recuperação de Senha**

Este exemplo mostra RNFs bem escritos, com anotações que explicam cada decisão de qualidade.

---

# Requisitos Não Funcionais — Recuperação de Senha

## Performance

**NFR-1**: O sistema deve processar a solicitação de recuperação de senha em até 3 segundos para 95% das requisições em condições normais de carga.

> Fonte: PRD — Critérios de Sucesso: "link de recuperação enviado em até 3 segundos"

<!--
✅ Bom: tem métrica (3 segundos), tem percentil (95%), tem condição (carga normal), rastreável ao PRD
❌ Evite: "O sistema deve processar rapidamente" — vago, não testável
-->

---

## Segurança

**NFR-2**: O sistema deve expirar links de recuperação de senha após 30 minutos da geração.

> Fonte: PRD — Fluxos Alternativos: "link expirado deve ser rejeitado"

<!--
✅ Bom: métrica clara (30 minutos), comportamento verificável, vinculado ao PRD
❌ Evite: "links devem expirar em tempo razoável" — sem métrica, não testável
-->

**NFR-3**: O sistema deve bloquear temporariamente novas solicitações de recuperação de senha para o mesmo endereço de e-mail após 3 tentativas em um intervalo de 10 minutos.

> Fonte: PRD — Seção de Riscos: "prevenção de abuso por envio massivo de e-mails"

<!--
✅ Bom: dois limites definidos (3 tentativas, 10 minutos), finalidade clara (previne abuso)
❌ Evite: "O sistema deve ter rate limiting" — não especifica limites, não testável
-->

**NFR-4**: O sistema deve utilizar algoritmo de hash seguro com custo computacional adequado para armazenar senhas, sem expor a senha em texto plano em nenhum ponto do fluxo.

> Fonte: Story US-3 — Critério: "senha nunca deve ser armazenada em texto plano"

<!--
✅ Bom: descreve o comportamento esperado sem mencionar tecnologia específica
❌ Evite: "O sistema deve usar bcrypt" — amarra a implementação a uma biblioteca
-->

---

## Disponibilidade

**NFR-5**: O sistema deve ter disponibilidade mínima de 99,5% ao mês para o fluxo de recuperação de senha.

> Fonte: PRD — Objetivos: "garantir acesso contínuo à plataforma de delivery"

<!--
✅ Bom: percentual definido (99,5%), escopo delimitado ao fluxo da feature
Nota: 99,5% = ~3,6 horas de downtime por mês — calibrado para feature não-crítica
-->

---

## Observabilidade

**NFR-6**: O sistema deve registrar em log todas as solicitações de recuperação de senha, incluindo: data, hora, endereço de e-mail solicitante e resultado (sucesso, e-mail não encontrado, bloqueado por rate limit).

> Fonte: PRD — Seção de Riscos: "auditoria de tentativas suspeitas"

<!--
✅ Bom: especifica os campos a logar e os estados possíveis — não deixa ambiguidade
❌ Evite: "O sistema deve registrar logs" — sem especificar o quê, não diz nada
-->

**NFR-7**: O sistema deve reter logs de auditoria do fluxo de recuperação de senha por no mínimo 90 dias.

> Fonte: Story US-4 — Critério: "histórico de recuperações disponível para auditoria"

<!--
✅ Bom: métrica clara (90 dias), rastreável à Story
-->

---

## Escalabilidade

**NFR-8**: O sistema deve suportar até 500 solicitações simultâneas de recuperação de senha sem degradação perceptível no tempo de resposta.

> Fonte: BDD — Cenário "Múltiplas solicitações simultâneas de recuperação"

<!--
✅ Bom: derivado de cenário BDD, tem número concreto (500), critério de sucesso definido
Nota: número calibrado com o usuário durante a entrevista — não inventado
-->

---

## Checklist de qualidade aplicado neste exemplo

| Critério | Status | Observação |
|---|---|---|
| Todos os RNFs têm métrica objetiva | ✅ | Tempos, percentuais, limites, contagens |
| Nenhum RNF usa termos vagos | ✅ | Sem "rápido", "seguro", "muito" |
| Todos os RNFs têm fonte rastreável | ✅ | PRD, Story ou BDD em cada `> Fonte:` |
| Categorias relevantes cobertas | ✅ | Performance, Segurança, Disponibilidade, Observabilidade, Escalabilidade |
| Nenhum RNF menciona tecnologia | ✅ | Sem bcrypt, JWT, MySQL, Redis |
