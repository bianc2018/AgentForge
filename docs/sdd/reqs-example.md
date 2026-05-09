# Exemplo Anotado — Requisitos Funcionais (EARS)

Este arquivo demonstra o formato esperado e os critérios de qualidade aplicados
a requisitos funcionais de uma feature de **login de entregador**.
Use como régua de qualidade ao revisar rascunhos.

---

# Requisitos Funcionais — Login de Entregador

## Fluxo Principal

**REQ-1**: When a delivery driver submits their credentials, the system shall validate the provided email and password.

> Fonte: Fluxo Principal do PRD (passo 2) / Cenário BDD "Login com credenciais válidas"

<!-- ✅ Bom: evento claro (submissão de credenciais), comportamento verificável (validação),
         vocabulário de domínio, sem tecnologia, padrão `When` correto para ação do usuário -->

**REQ-2**: When valid credentials are submitted, the system shall authenticate the delivery driver.

> Fonte: Fluxo Principal do PRD (passo 3) / Cenário BDD "Login com credenciais válidas" / Estória 1, critério 1

<!-- ✅ Bom: distinção clara entre validar (REQ-1) e autenticar (REQ-2),
         "valid credentials" conecta com o `Given` do cenário BDD -->

**REQ-3**: After successful authentication, the system shall create an active session for the delivery driver.

> Fonte: Critério de Sucesso do PRD "Sessão iniciada" / Estória 1, critério 2

<!-- ✅ Bom: comportamento de pós-autenticação separado em requisito próprio,
         não menciona JWT, cookie ou Redis — decisão de design, não de requisito -->

**REQ-4**: After successful authentication, the system shall redirect the delivery driver to the main dashboard.

> Fonte: Fluxo Principal do PRD (passo 4) / Estória 1, critério 3

<!-- ✅ Bom: resultado observável (redirecionamento), não menciona rota específica ou framework -->

---

## Validação de Entrada

**REQ-5**: If the submitted email does not match any registered account, the system shall reject the authentication request.

> Fonte: Fluxo Alternativo "E-mail não cadastrado" do PRD / Cenário BDD "Login com e-mail não cadastrado"

<!-- ✅ Bom: padrão `If` correto para condição de falha,
         "reject" é comportamento verificável sem ambiguidade -->

**REQ-6**: If an invalid password is submitted, the system shall reject the authentication request.

> Fonte: Fluxo Alternativo "Senha incorreta" do PRD / Cenário BDD "Login com senha incorreta"

<!-- ✅ Bom: separado do REQ-5 — e-mail inválido e senha inválida são condições distintas -->

**REQ-7**: If authentication fails, the system shall display an error message to the delivery driver without revealing which credential is incorrect.

> Fonte: Objetivo "Rejeitar credenciais inválidas sem revelar qual campo está errado" do PRD / Estória 2, critério 2

<!-- ✅ Bom: "without revealing which credential is incorrect" captura um requisito de segurança
         de forma não-técnica e verificável -->

---

## Segurança

**REQ-8**: The system shall log all authentication attempts, whether successful or failed.

> Fonte: Requisito de auditoria do PRD / Estória 3, critério 1

<!-- ✅ Bom: padrão ubíquo correto (acontece sempre), comportamento de log é observável
         via interface de auditoria -->

**REQ-9**: If a delivery driver submits five consecutive failed authentication attempts, the system shall temporarily block further attempts from that account.

> Fonte: Risco "Ataques de força bruta" do PRD / Cenário BDD "Bloqueio após múltiplas tentativas falhas"

<!-- ✅ Bom: condição precisa ("five consecutive failed attempts"), comportamento claro ("temporarily block"),
         "that account" limita o escopo sem mencionar implementação -->

---

## Exemplos de requisitos com problemas (NÃO USE)

**❌ Tecnologia exposta:**
```
The system shall store the JWT token in the Redis cache with a TTL of 3600 seconds.
```
*Problema: menciona JWT, Redis e TTL — decisões de implementação, não de requisito.*

**❌ Comportamento do usuário, não do sistema:**
```
The delivery driver shall enter their email and password in the login form.
```
*Problema: descreve ação do usuário. O sujeito do requisito deve ser sempre "The system".*

**❌ `When` para condição de falha:**
```
When invalid credentials are submitted, the system shall display an error.
```
*Problema: condição de falha deve usar `If`, não `When`. Use: `If invalid credentials are submitted, the system shall...`*

**❌ Comportamento vago:**
```
The system shall handle authentication errors appropriately.
```
*Problema: "appropriately" é subjetivo e não verificável. O que exatamente o sistema deve fazer?*

**❌ Dois comportamentos em um requisito:**
```
When valid credentials are submitted, the system shall authenticate the driver and redirect them to the dashboard.
```
*Problema: autenticação e redirecionamento são comportamentos distintos — separar em REQ-2 e REQ-4.*
