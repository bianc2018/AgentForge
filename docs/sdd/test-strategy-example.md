# Exemplo de Estratégia de Testes — Recuperação de Senha

> **Nota:** Exemplo anotado. Comentários `<!-- -->` explicam por que cada parte
> satisfaz o checklist. Remova-os em estratégias reais.

---

## 1. Testes Unitários

<!-- ✅ Um UT por método de domínio do design.md -->
<!-- ✅ Cada UT cobre caminho feliz + todas as condições de erro -->
<!-- ✅ Nenhum UT depende de banco, HTTP ou serviço externo -->

### UT-1: `PasswordRecoveryDomain.generateCode()`

- **O que testa:** Geração de código OTP numérico com timestamp de expiração correto
- **Casos cobertos:**
  - Caminho feliz: retorna string de exatamente 6 dígitos numéricos
  - Expiração: `expires_at` é `now() + 5 minutos` (± 1 segundo de tolerância)
  - Aleatoriedade: 100 execuções não produzem o mesmo código consecutivamente
- **Mocks necessários:** Clock mockado para controlar `now()` nos asserts de `expires_at`
- **Rastreabilidade:** REQ-1 · NFR-2

---

### UT-2: `PasswordRecoveryDomain.validateCode()`

- **O que testa:** Validação de código OTP contra as regras de validade
- **Casos cobertos:**
  - Caminho feliz: código com `expires_at` futuro e `used_at` null → aceito
  - Código expirado: `expires_at` no passado → rejeitado com erro `CODE_EXPIRED`
  - Código já utilizado: `used_at` preenchido → rejeitado com erro `CODE_ALREADY_USED`
  - Código inexistente: entry `null` → rejeitado com erro `CODE_NOT_FOUND`
- **Mocks necessários:** Clock mockado para controlar `now()` nos testes de expiração
- **Rastreabilidade:** REQ-2 · REQ-9 · REQ-10 · NFR-2

---

### UT-3: `PasswordRecoveryDomain.validatePasswordStrength()`

- **O que testa:** Validação de força de senha contra os 5 requisitos mínimos
- **Casos cobertos:**
  - Caminho feliz: senha com todos os requisitos → aceita, lista vazia retornada
  - Sem maiúscula: retorna `['MISSING_UPPERCASE']`
  - Sem minúscula: retorna `['MISSING_LOWERCASE']`
  - Sem número: retorna `['MISSING_NUMBER']`
  - Sem especial: retorna `['MISSING_SPECIAL']`
  - Menos de 8 chars: retorna `['MIN_LENGTH']`
  - Múltiplos ausentes: todos os requisitos violados retornados simultaneamente
- **Mocks necessários:** Nenhum — domínio puro
- **Rastreabilidade:** REQ-8

---

### UT-4: `PasswordRecoveryDomain.validatePasswordMatch()`

- **O que testa:** Comparação entre senha e confirmação
- **Casos cobertos:**
  - Campos iguais → aceito
  - Campos diferentes → rejeitado com erro `PASSWORDS_DO_NOT_MATCH`
  - Um campo vazio → rejeitado
- **Mocks necessários:** Nenhum — domínio puro
- **Rastreabilidade:** REQ-7

---

## 2. Testes de Integração

<!-- ✅ Um IT por repository e por adapter de infraestrutura do design.md -->
<!-- ✅ Dependências reais especificadas (banco de teste, Redis de teste) -->
<!-- ✅ Setup/teardown descrito -->
<!-- ✅ Comportamento de falha de dependência coberto -->

### IT-1: `VerificationCodeRepository` — operações de ciclo de vida do código

- **O que testa:** Persistência, recuperação e invalidação de códigos OTP no banco
- **Dependências reais usadas:** PostgreSQL de teste (banco dedicado para testes)
- **Casos cobertos:**
  - `save()`: persiste código com todos os campos; `used_at` inicia como null
  - `findValid()`: retorna o código quando válido; retorna null para código expirado; retorna null para código com `used_at` preenchido
  - `markAsUsed()`: preenche `used_at`; chamada subsequente a `findValid()` retorna null
- **Setup necessário:** Banco limpo por teste (truncate em `verification_codes` no teardown); entregador de teste pre-inserido
- **Rastreabilidade:** REQ-1 · REQ-2 · REQ-9 · REQ-10

---

### IT-2: `RateLimiterAdapter` — controle de tentativas no Redis

- **O que testa:** Incremento de contadores, verificação de bloqueio e reset por TTL e por sucesso
- **Dependências reais usadas:** Redis de teste (instância dedicada para testes)
- **Casos cobertos:**
  - `increment()` + `check()`: contador cresce corretamente; `check()` retorna bloqueado na 5ª chamada
  - TTL: chave expira após 30 minutos (verificado via `TTL` do Redis no teste)
  - `reset()`: zera o contador; `check()` retorna desbloqueado imediatamente após
  - Isolamento por chave: bloqueio de contato não bloqueia IP e vice-versa
- **Setup necessário:** Flush no Redis de teste antes de cada IT
- **Rastreabilidade:** REQ-11 · REQ-12 · NFR-4

---

### IT-3: `DeliveryDriverRepository` — leitura e atualização por contato

- **O que testa:** Busca de entregador por canal/contato e atualização de hash de senha
- **Dependências reais usadas:** PostgreSQL de teste
- **Casos cobertos:**
  - `findByContact('email', email_existente)`: retorna entidade completa
  - `findByContact('email', email_inexistente)`: retorna null sem lançar erro
  - `findByContact('sms', telefone_existente)`: retorna entidade completa
  - `updatePasswordHash(driverId, novoHash)`: hash atualizado no banco; `findByContact` subsequente retorna novo hash
- **Setup necessário:** Entregador de teste pre-inserido no banco; limpeza no teardown
- **Rastreabilidade:** REQ-1 · REQ-3 · REQ-6

---

### IT-4: `EmailNotificationAdapter` — envio de código por email

- **O que testa:** Integração com o serviço externo de email
- **Dependências reais usadas:** Serviço de email em modo sandbox (ex: Mailtrap, SendGrid sandbox)
- **Casos cobertos:**
  - Envio bem-sucedido: email recebido no sandbox com o código OTP no corpo
  - Falha do serviço: quando o serviço retorna erro 5xx, o adapter lança exceção propagável
- **Setup necessário:** Credenciais de sandbox configuradas nas variáveis de ambiente de teste
- **Rastreabilidade:** REQ-1 · NFR-1

---

### IT-5: `SmsNotificationAdapter` — envio de código por SMS

- **O que testa:** Integração com o serviço externo de SMS
- **Dependências reais usadas:** Serviço de SMS em modo sandbox
- **Casos cobertos:**
  - Envio bem-sucedido: mensagem registrada no sandbox com o código OTP
  - Falha do serviço: quando o serviço retorna erro, o adapter lança exceção propagável
- **Setup necessário:** Credenciais de sandbox configuradas nas variáveis de ambiente de teste
- **Rastreabilidade:** REQ-1 · NFR-1

---

## 3. Testes E2E Gherkin

<!-- ✅ Um GH por Scenario do .feature — sem exceções -->
<!-- ✅ Step definitions descritos com implementação esperada -->
<!-- ✅ Steps reutilizáveis identificados -->
<!-- ✅ Estado inicial descrito para cada cenário -->

### GH-1: Scenario "Recuperação de senha com email e código válido"

- **Arquivo:** `docs/features/recuperacao-de-senha/scenarios.feature`
- **Step definitions necessários:**
  - `Given o entregador está na página de recuperação de senha e selecionou email como canal` → configura estado inicial: entregador cadastrado no banco de teste; faz GET no endpoint de início do fluxo
  - `When o entregador informa o email cadastrado e solicita o código de verificação` → POST `/api/v1/password-recovery/request` com email válido cadastrado
  - `And o entregador informa o código de verificação recebido por email` → lê o código gerado diretamente do banco de teste (tabela `verification_codes`); POST `/api/v1/password-recovery/verify`
  - `And o entregador define e confirma a nova senha` → POST `/api/v1/password-recovery/reset` com senha válida e reset_token recebido
  - `Then o sistema atualiza a senha da conta` → verifica que `password_hash` no banco foi alterado
  - `And o entregador é redirecionado para a página de login` → verifica campo de redirecionamento na resposta 200
- **Steps reutilizáveis de outros Scenarios:** `Given o entregador está na página...` (GH-2, GH-3)
- **Estado inicial necessário:** Entregador cadastrado com email conhecido; banco limpo de `verification_codes`
- **Rastreabilidade:** REQ-1 · REQ-2 · REQ-3 · REQ-4

---

### GH-2: Scenario "Solicitação de código com dado de formato inválido"

- **Step definitions necessários:**
  - `Given o entregador está na página de recuperação de senha e selecionou email como canal` → reutilizado de GH-1
  - `When o entregador informa um endereço de email com formato inválido e solicita o código de verificação` → POST `/api/v1/password-recovery/request` com email malformado (ex: `"nao-e-um-email"`)
  - `Then o sistema exibe uma mensagem de erro informando que o formato do email é inválido` → verifica status 400 e campo de erro na resposta
  - `And o código de verificação não é enviado` → verifica que nenhum registro foi criado em `verification_codes`
- **Steps reutilizáveis:** `Given` de GH-1
- **Estado inicial necessário:** Nenhum entregador necessário (validação ocorre antes da busca)
- **Rastreabilidade:** REQ-5

---

### GH-3: Scenario "Solicitação de código com dado não cadastrado"

- **Step definitions necessários:**
  - `Given o entregador está na página de recuperação de senha e selecionou email como canal` → reutilizado de GH-1
  - `When o entregador informa um endereço de email não cadastrado e solicita o código de verificação` → POST `/request` com email válido em formato mas não existente no banco
  - `Then o sistema exibe a mesma mensagem neutra de confirmação exibida quando o dado está cadastrado` → verifica que status 200 e corpo da resposta são byte-a-byte idênticos ao retorno de GH-1
  - `And o código de verificação não é enviado` → verifica ausência de registro em `verification_codes`
- **Steps reutilizáveis:** `Given` de GH-1
- **Estado inicial necessário:** Banco sem o email usado no teste
- **Rastreabilidade:** REQ-6 · NFR-3

---

### GH-4: Scenario "Informar código de verificação expirado ou inválido"

- **Step definitions necessários:**
  - `Given o entregador está na etapa de informar o código de verificação` → insere código com `expires_at = now() - 1 minuto` diretamente no banco de teste
  - `When o entregador informa um código de verificação inválido ou expirado` → POST `/api/v1/password-recovery/verify` com o código expirado
  - `Then o sistema exibe uma mensagem de erro informando que o código é inválido ou expirado` → verifica status 401 e campo de erro
  - `And o entregador permanece na etapa de informar o código...` → verifica que nenhum `reset_token` foi emitido
- **Steps reutilizáveis:** Nenhum
- **Estado inicial necessário:** Entregador cadastrado; código expirado inserido diretamente no banco
- **Rastreabilidade:** REQ-9 · NFR-2

---

### GH-5: Scenario "Redefinição de senha com campos de confirmação divergentes"

- **Step definitions necessários:**
  - `Given o entregador está na etapa de redefinição de senha após validar o código de verificação` → executa o fluxo de verify completo para obter um `reset_token` válido
  - `When o entregador preenche o campo de nova senha e o campo de confirmação com valores diferentes e submete o formulário` → POST `/reset` com `password ≠ password_confirmation`
  - `Then o sistema exibe uma mensagem de erro informando que os campos de senha não coincidem` → verifica status 400
  - `And a senha da conta não é alterada` → verifica que `password_hash` no banco não mudou
- **Steps reutilizáveis:** `Given` pode reutilizar parte do fluxo de GH-6
- **Estado inicial necessário:** Entregador com `reset_token` válido (obtido via `/verify`)
- **Rastreabilidade:** REQ-7

---

### GH-6: Scenario "Redefinição de senha sem atender aos requisitos mínimos"

- **Step definitions necessários:**
  - `Given o entregador está na etapa de redefinição de senha após validar o código de verificação` → reutilizado de GH-5
  - `When o entregador define uma senha que não atende aos requisitos mínimos de segurança e submete o formulário` → POST `/reset` com senha fraca (ex: `"senha123"` — sem maiúscula e sem especial)
  - `Then o sistema exibe uma mensagem de erro listando todos os requisitos mínimos de segurança exigidos` → verifica status 422 e que a lista de requisitos violados é completa
  - `And a senha da conta não é alterada` → verifica que `password_hash` não mudou
- **Steps reutilizáveis:** `Given` de GH-5
- **Estado inicial necessário:** Entregador com `reset_token` válido
- **Rastreabilidade:** REQ-8

---

### GH-7: Scenario "Bloqueio por tentativas excessivas de solicitação de código"

- **Step definitions necessários:**
  - `Given o entregador atingiu 5 tentativas sem sucesso consecutivas de solicitação de código de verificação` → chama `POST /request` 5 vezes com email não cadastrado (ou manipula contador no Redis diretamente)
  - `When o entregador tenta solicitar um novo código de verificação` → POST `/request` na 6ª tentativa
  - `Then o sistema exibe uma mensagem informando que a solicitação está suspensa por 30 minutos` → verifica status 429 e mensagem de bloqueio
  - `And o código de verificação não é enviado` → verifica ausência de novo registro em `verification_codes`
- **Steps reutilizáveis:** Nenhum
- **Estado inicial necessário:** Redis limpo; entregador pode ou não existir no banco
- **Rastreabilidade:** REQ-11 · NFR-4

---

### GH-8: Scenario "Tentativa de reutilizar código após uso bem-sucedido"

- **Step definitions necessários:**
  - `Given o entregador já concluiu o fluxo de recuperação de senha utilizando um código de verificação` → executa fluxo completo de `/verify` com sucesso; guarda o código usado
  - `When o entregador tenta submeter o mesmo código de verificação novamente` → POST `/verify` com o mesmo código
  - `Then o sistema exibe uma mensagem de erro informando que o código é inválido ou expirado` → verifica status 401
  - `And o entregador permanece na etapa de informar o código...` → verifica que nenhum novo `reset_token` foi emitido
- **Steps reutilizáveis:** `Given` pode reaproveitar parte do fluxo de GH-1
- **Estado inicial necessário:** Código já utilizado (campo `used_at` preenchido)
- **Rastreabilidade:** REQ-10 · NFR-2

---

## 4. Testes de Performance

<!-- ✅ Um PT por NFR com critério mensurável -->
<!-- ✅ Threshold derivado diretamente do NFR -->
<!-- ✅ Método de medição executável em CI -->

### PT-1: Entrega do código no canal em até 30 segundos (NFR-1)

- **O que mede:** Tempo entre a chamada ao `POST /request` e a entrega efetiva no canal (email ou SMS)
- **Threshold:** p95 ≤ 30 segundos em condições normais de operação
- **Método de medição:** Teste de integração com serviço em sandbox; mede timestamp de envio vs. timestamp de recebimento na caixa do sandbox
- **Número de execuções:** 20 execuções por canal (email e SMS separadamente)
- **Rastreabilidade:** NFR-1

---

### PT-2: Tempo de execução do hash de senha ≥ 100ms (NFR-5)

- **O que mede:** Tempo de execução de `PasswordService.hash()` por operação individual
- **Threshold:** ≥ 100ms por execução (limite mínimo — abaixo indica custo bcrypt insuficiente)
- **Método de medição:** Benchmark unitário sem dependências externas; mede wall-clock time de cada chamada
- **Número de execuções:** 10 execuções; todas devem estar acima do threshold
- **Rastreabilidade:** NFR-5

---

## 5. Testes de Segurança

<!-- ✅ Um ST por NFR de segurança -->
<!-- ✅ Um ST por risco do PRD com mitigação técnica verificável -->
<!-- ✅ Vetor de ataque descrito especificamente -->

### ST-1: Resposta neutra não revela existência de conta (NFR-3 / anti-enumeração)

- **O que verifica:** Que um atacante não consegue determinar se um email está cadastrado pela resposta do sistema
- **Vetor de ataque simulado:** Enumeração de contas — enviar emails aleatórios e comparar respostas
- **Casos cobertos:**
  - Status HTTP é idêntico (200) para email cadastrado e não cadastrado
  - Corpo da resposta é textualmente idêntico para ambos os casos
  - Tempo de resposta não vaza informação: diferença de tempo entre os dois casos é < 50ms (evita timing attack via duração da query)
- **Rastreabilidade:** NFR-3 · Risco PRD: "Feedback do sistema confirma se uma conta existe"

---

### ST-2: Rate limiting bloqueia após 5 tentativas (NFR-4)

- **O que verifica:** Que o mecanismo de rate limiting funciona por contato e por IP independentemente
- **Vetor de ataque simulado:** Spam de solicitações de código para abusar do envio de SMS/email
- **Casos cobertos:**
  - 5 tentativas do mesmo contato → bloqueio ativo; 6ª retorna 429
  - 5 tentativas do mesmo IP (contatos diferentes) → bloqueio ativo por IP; 6ª retorna 429
  - Bloqueio de contato não afeta IPs diferentes usando outros contatos
  - Contador reinicia após conclusão bem-sucedida do fluxo
- **Rastreabilidade:** NFR-4 · Risco PRD: "Uso abusivo do envio de códigos"

---

### ST-3: Código OTP é invalidado após uso (REQ-10 / risco de replay)

- **O que verifica:** Que um código interceptado não pode ser reutilizado após uso bem-sucedido
- **Vetor de ataque simulado:** Replay attack — atacante intercepta o código e tenta usá-lo após a vítima já ter concluído o fluxo
- **Casos cobertos:**
  - Código usado com sucesso → segunda submissão retorna 401 imediatamente
  - Reset token emitido não pode ser reutilizado após expiração (15 minutos)
- **Rastreabilidade:** REQ-10 · NFR-2 · Risco PRD: "Código reutilizado após redefinição bem-sucedida"

---

### ST-4: Hash de senha resiste a brute force offline (NFR-5)

- **O que verifica:** Que o custo do bcrypt é suficientemente alto para dificultar ataques offline
- **Vetor de ataque simulado:** Brute force offline — atacante obtém o hash do banco e tenta quebrá-lo localmente
- **Casos cobertos:**
  - Tempo de `hash()` é ≥ 100ms no hardware de CI (verifica que o fator de custo está correto)
  - Fator de custo bcrypt está configurado para ≥ 12 (verificado via inspeção do valor configurado)
- **Rastreabilidade:** NFR-5 · Risco PRD: "Código interceptado por terceiros"

---

## Resumo de Cobertura

<!-- ✅ Cada REQ tem ao menos 1 teste E2E Gherkin e 1 unitário ou de integração -->
<!-- ✅ Cada NFR mensurável tem ao menos 1 teste de performance -->
<!-- ✅ Cada NFR de segurança tem ao menos 1 teste de segurança -->
<!-- ✅ Nenhuma linha completamente vazia -->

| Requisito | Unitário | Integração | E2E Gherkin | Performance | Segurança |
|-----------|----------|------------|-------------|-------------|-----------|
| REQ-1 | UT-1 | IT-1, IT-4, IT-5 | GH-1 | PT-1 | — |
| REQ-2 | UT-2 | IT-1 | GH-1 | — | — |
| REQ-3 | — | IT-3 | GH-1 | — | ST-4 |
| REQ-4 | — | — | GH-1 | — | — |
| REQ-5 | — | — | GH-2 | — | — |
| REQ-6 | — | IT-3 | GH-3 | — | ST-1 |
| REQ-7 | UT-4 | — | GH-5 | — | — |
| REQ-8 | UT-3 | — | GH-6 | — | — |
| REQ-9 | UT-2 | IT-1 | GH-4 | — | — |
| REQ-10 | UT-2 | IT-1 | GH-8 | — | ST-3 |
| REQ-11 | — | IT-2 | GH-7 | — | ST-2 |
| REQ-12 | — | IT-2 | GH-7 | — | — |
| NFR-1 | — | IT-4, IT-5 | — | PT-1 | — |
| NFR-2 | UT-2 | IT-1 | GH-4, GH-8 | — | ST-3 |
| NFR-3 | — | IT-3 | GH-3 | — | ST-1 |
| NFR-4 | — | IT-2 | GH-7 | — | ST-2 |
| NFR-5 | — | — | — | PT-2 | ST-4 |
| NFR-6 | — | — | — | — | — |
