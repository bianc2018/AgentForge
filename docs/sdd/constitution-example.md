# Exemplo Anotado — constitution.md

Este arquivo mostra um exemplo de constitution.md com qualidade suficiente, com anotações explicando por que cada parte funciona.

---

## constitution.md

### Purpose

<!-- ✅ Duas frases diretas. Declara o escopo ("this system"), o tipo de regras ("non-negotiable technical rules") e a instrução de uso ("Any ambiguity must be resolved explicitly"). -->

This document defines the non-negotiable technical rules that govern all implementations in this system.
Any ambiguity must be resolved explicitly — never assumed.

---

### Must Do

<!-- ✅ 5 regras cobrindo: separação de camadas, propagação de erros, validação de entradas, logging e rastreabilidade. Cada uma começa com "All [sujeito] must [verbo]" e é verificável mecanicamente. -->

1. All business logic must reside in the service layer and remain independent from transport and persistence layers.

   <!-- ✅ Camada especificada (service layer). Proibição implícita de mistura com transport e persistence. Verificável: ao revisar um PR, é possível checar se há lógica de negócio fora do serviço. -->

2. All errors must be propagated using a standardized structure and include contextual metadata (requestId, timestamp, error code).

   <!-- ✅ Define o formato esperado (estrutura padronizada) e os campos obrigatórios (requestId, timestamp, error code). Verificável: o objeto de erro pode ser inspecionado. -->

3. All external inputs must be validated at system boundaries before reaching domain logic.

   <!-- ✅ Define onde a validação acontece (system boundaries) e o invariante (antes do domínio). Verificável: ao rastrear o fluxo de uma requisição, a validação deve ser anterior ao serviço. -->

4. All state-changing operations must be logged using structured logging.

   <!-- ✅ Define o escopo (state-changing operations) e o formato (structured logging). Verificável: ao checar operações de write, cada uma deve ter um log correspondente. -->

5. All changes must be traceable back to a functional requirement or BDD scenario.

   <!-- ✅ Define rastreabilidade como obrigatória e especifica os destinos válidos (requisito funcional ou BDD). Verificável: commits ou PRs devem referenciar um requisito ou cenário. -->

---

### Ask Before Proceeding

<!-- ✅ 4 regras cobrindo os quatro gates de aprovação mais comuns: requisito ambíguo, decisão arquitetural, mudança de contrato e conflito com a própria constituição. Cada uma tem condição (If X) e ação (must Y). -->

6. If a requirement is ambiguous or incomplete, the implementation must stop and request clarification before proceeding.

   <!-- ✅ Condição clara (ambíguo ou incompleto). Ação clara (parar e pedir clareza). Não deixa margem para suposição. -->

7. If multiple architectural approaches are possible (e.g., sync vs async, cache vs direct query), a justification must be provided and validated.

   <!-- ✅ Condição com exemplos concretos (sync/async, cache/query). Ação exige justificativa E validação — não apenas documentar a escolha. -->

8. If a change impacts data model, contracts, or public APIs, confirmation must be obtained before implementation.

   <!-- ✅ Define o escopo do risco (modelo de dados, contratos, APIs públicas). Ação exige confirmação prévia — não posterior. -->

9. If a requirement appears to conflict with an existing rule in this constitution, the conflict must be explicitly raised.

   <!-- ✅ Fecha o loop de auto-consistência: a constituição também pode ser questionada. Evita que um desenvolvedor silencie um conflito real. -->

---

### Never Do

<!-- ✅ 5 proibições absolutas cobrindo: acesso direto ao banco, lógica de negócio fora do domínio, suposições de requisito, erros silenciosos e acoplamento com frameworks. Cada uma começa com "Never" e descreve um anti-pattern específico. -->

10. Never access the database directly from controllers or UI layers.

    <!-- ✅ Nomeia as camadas proibidas (controllers, UI). Verificável: imports ou chamadas de ORM/query em controllers são uma violação direta. -->

11. Never embed business logic inside controllers, repositories, or external adapters.

    <!-- ✅ Lista as camadas proibidas explicitamente (controllers, repositories, adapters). Complementa a regra 1 pelo lado negativo. -->

12. Never assume missing requirements or infer business rules without explicit specification.

    <!-- ✅ Proíbe suposições e inferências. Complementa a regra 6 pelo lado negativo. -->

13. Never swallow errors silently or return generic error responses without context.

    <!-- ✅ Dois anti-patterns em uma regra: silenciar erros e responder genericamente sem contexto. Complementa a regra 2. -->

14. Never couple domain logic to specific frameworks, libraries, or infrastructure details.

    <!-- ✅ Generaliza o acoplamento a qualquer framework ou lib — não apenas ao banco. Mantém o domínio portável. -->

---

### Enforcement

<!-- ✅ 3 regras de execução cobrindo: gate pré-implementação, consequência de violação e bloqueio por clareza faltando. As consequências são concretas (invalidar, bloquear, corrigir). -->

15. Any implementation plan must explicitly describe how it complies with this constitution.

    <!-- ✅ Gate pré-implementação: o plano deve justificar conformidade antes de começar. Aplicável em code reviews ou documentos de design. -->

16. Any violation of these rules invalidates the implementation and must be corrected before merge.

    <!-- ✅ Consequência clara e concreta (invalida + bloqueia merge). Sem exceções documentadas ou workarounds. -->

17. Any missing clarification must block implementation until resolved.

    <!-- ✅ Reforça as regras 6-9 com uma consequência: clareza faltando = bloqueio total. Previne o "vou fazer e ver o que acontece". -->

---

## O que torna este exemplo de qualidade

1. **Numeração global**: as regras são numeradas de 1 a 17 continuamente — facilita referência ("violação da regra 13")
2. **Complementaridade**: Must Do e Never Do cobrem os mesmos temas pelos dois lados (ex: regra 1 + regra 11 = separação de camadas)
3. **Sem repetição**: cada regra existe em apenas uma seção
4. **Sem vagueza**: "structured logging", "requestId, timestamp, error code" — sem "boas práticas" ou "adequadamente"
5. **Enforcement concreto**: "invalidates the implementation", "must be corrected before merge" — não "recomenda-se revisar"
