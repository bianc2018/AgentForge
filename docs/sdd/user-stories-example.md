# Exemplo Anotado de Estórias de Usuário — Login de Entregador

> Este é um exemplo de estórias de usuário de alta qualidade para a feature `login-entregador`.
> Cada estória inclui anotações explicando as decisões tomadas.
> Use como régua ao avaliar e refinar rascunhos durante a entrevista.

---

# Estórias de Usuário — Login de Entregador

## Estoria 1 – Autenticação com credenciais válidas

Como entregador com conta previamente criada
Eu quero me autenticar na plataforma informando meu identificador e senha
Para que eu possa acessar as funcionalidades restritas do sistema

_Critérios de aceitação_:

- O sistema deve aceitar email, telefone ou login único como identificador, sem que o entregador precise selecionar o tipo
- O sistema deve autenticar o entregador e iniciar uma sessão válida quando as credenciais estiverem corretas
- O sistema deve redirecionar o entregador para o dashboard após autenticação bem-sucedida
- A sessão iniciada deve permitir acesso a funcionalidades protegidas sem nova autenticação a cada requisição

> **Anotação — Por que essa estória?**
> Âncora no PRD: Objetivos 1, 2 e 3 ("Autenticar o entregador", "Emitir tokens", "Detectar o tipo de identificador")
> e o Fluxo Principal completo. Esta é a estória central da feature — o happy path de ponta a ponta.
>
> **Anotação — Persona**
> "Com conta previamente criada" é uma pré-condição crítica derivada diretamente da seção
> Usuário-Alvo do PRD. Sem ela, a estória fica incompleta e pode gerar confusão sobre quem
> pode usar a feature.
>
> **Anotação — "Eu quero"**
> Usa linguagem de negócio ("me autenticar"), não técnica ("receber JWT").
> O mecanismo interno (JWT, cookie httpOnly, sliding window) não pertence à estória —
> pertence ao design técnico.
>
> **Anotação — Critérios**
> O critério "email, telefone ou login único" deriva do Objetivo 3 do PRD.
> O critério "sessão permite acesso" deriva do Objetivo 5.
> Todos são verificáveis sem conhecer a implementação interna.

---

## Estoria 2 – Rejeição de credenciais inválidas

Como entregador com conta previamente criada
Eu quero ser informado quando minhas credenciais estiverem incorretas
Para que eu possa corrigir o erro e tentar novamente

_Critérios de aceitação_:

- O sistema deve rejeitar a autenticação quando o identificador ou a senha estiverem incorretos
- O sistema deve exibir uma mensagem de erro genérica, sem indicar qual campo específico está errado
- O sistema não deve criar sessão nem redirecionar o entregador em caso de credenciais inválidas
- O entregador deve poder tentar novamente após o erro, sem precisar recarregar a página

> **Anotação — Por que essa estória?**
> Âncora no PRD: Objetivo 4 ("Rejeitar credenciais inválidas com mensagem genérica") e
> o Critério de Sucesso "Credenciais inválidas resultam em mensagem genérica".
>
> **Anotação — Critério de mensagem genérica**
> O critério "sem indicar qual campo específico está errado" é um requisito de segurança
> que vem diretamente do PRD. É explícito porque um desenvolvedor poderia facilmente
> exibir mensagens diferentes para "email não encontrado" vs "senha errada" sem essa instrução,
> o que comprometeria a segurança ao revelar se um email está cadastrado.
>
> **Anotação — Separação da Estória 1**
> Esta estória é independente da Estória 1: cobre um fluxo completamente distinto
> (falha de autenticação vs. sucesso) e pode ser implementada e testada separadamente.

---

## Estoria 3 – Manutenção da sessão ativa durante o uso

Como entregador autenticado
Eu quero permanecer autenticado durante o uso contínuo da plataforma
Para que eu não precise me autenticar novamente a cada acesso

_Critérios de aceitação_:

- O sistema deve manter a sessão do entregador ativa durante o uso normal, sem exigir nova autenticação
- O sistema deve renovar a sessão automaticamente quando necessário, de forma transparente ao entregador
- O entregador não deve perceber interrupções durante o uso contínuo dentro do período de sessão ativa

> **Anotação — Por que essa estória?**
> Âncora no PRD: Objetivo 5 ("Renovar o refresh token, mantendo sessões ativas")
> e o Fluxo de Refresh descrito nos Fluxos Alternativos.
>
> **Anotação — "Eu quero" sem jargão técnico**
> O rascunho inicial poderia ser "Eu quero que meu refresh token seja renovado via sliding window".
> Isso está errado — o sliding window é detalhe de implementação.
> A versão correta expressa o que o entregador experimenta: "permanecer autenticado".
>
> **Anotação — Critérios sem jargão técnico**
> "Renovar automaticamente quando necessário" cobre o comportamento do refresh sem mencionar
> access token, cookie ou sliding window. Esses detalhes pertencem ao design técnico,
> não à estória de usuário.

---

## Estoria 4 – Proteção contra tentativas excessivas de login

Como entregador tentando acessar a plataforma
Eu quero que o sistema me proteja contra tentativas abusivas de acesso à minha conta
Para que minha conta não seja comprometida por ataques de força bruta

_Critérios de aceitação_:

- O sistema deve detectar tentativas excessivas de autenticação com credenciais inválidas a partir do mesmo origem
- O sistema deve bloquear temporariamente novas tentativas após atingir o limite definido
- O sistema deve informar o entregador sobre o bloqueio temporário e quando poderá tentar novamente

> **Anotação — Por que essa estória?**
> Âncora no PRD: seção Riscos ("Brute-force de credenciais — Rate limit por IP + por identificador")
> e Critérios de Sucesso ("Brute-force mitigado").
>
> **Anotação — Estória de segurança com impacto visível**
> Nem todo risco do PRD vira estória. Este vira porque tem comportamento visível ao entregador:
> o bloqueio temporário é algo que o usuário experimenta diretamente. Riscos puramente técnicos
> (ex: "JWT assinado com secret de 256 bits") não viram estórias de usuário.
>
> **Anotação — Persona alternativa**
> A persona aqui é "tentando acessar", não "com conta previamente criada", porque
> o bloqueio acontece antes da autenticação bem-sucedida — inclusive para usuários
> que não têm conta. Ajustar a persona ao contexto específico da estória é correto
> e esperado.

---

## O que NÃO virou estória nesta feature

As situações abaixo **não geraram estórias** por razões explícitas:

- **"Como entregador, eu quero que meu JWT seja assinado com HS256"** — detalhe técnico de implementação, não comportamento de usuário. Pertence ao design técnico.
- **"Como entregador, eu quero fazer logout"** — explicitamente listado em "Fora do Escopo" no PRD.
- **"Como entregador, eu quero usar login social"** — explicitamente listado em "Fora do Escopo" no PRD.
- **"Como entregador, eu quero que meu token fique em cookie httpOnly"** — detalhe técnico de segurança; o benefício para o usuário é "permanecer autenticado com segurança", que já está coberto na Estória 3.

> **Anotação — Por que documentar o que não virou estória?**
> Torna explícito o raciocínio durante a revisão e evita que revisores ou desenvolvedores
> questionem se esses itens foram esquecidos. É especialmente útil em features de segurança
> onde muitos detalhes técnicos poderiam parecer candidatos a estórias.
