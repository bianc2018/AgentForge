# Exemplo anotado de cenários BDD de qualidade
# Feature de referência: Autenticação de Entregador
#
# Anotações (prefixadas com #) explicam as decisões de design.
# Remova-as ao criar cenários reais.

Feature: Autenticação de entregador
  # O título da feature reflete o objetivo do PRD, não a tecnologia usada.
  # Descrição opcional — orientada ao negócio.

  # --- CENÁRIO 1: Happy path ---
  # Âncora: Fluxo Principal do PRD + Estória 1 (happy path)
  # Título específico: identifica que são credenciais válidas.
  Scenario: Login com credenciais válidas
    # Given: estado do sistema antes da ação. Sem ação do usuário.
    Given um entregador com conta cadastrada na plataforma
    # When: única ação do usuário. Linguagem de domínio, sem mencionar HTTP.
    When o entregador envia suas credenciais corretas
    # Then: resultado observável. O entregador vê o dashboard — verificável na UI.
    Then o entregador acessa o painel principal
    # And: resultado adicional do mesmo cenário (não uma nova ação).
    And uma sessão ativa é registrada para o entregador

  # --- CENÁRIO 2: Erro de entrada ---
  # Âncora: Fluxo Alternativo "senha incorreta" + critério de aceitação da Estória 2
  # Título no padrão: <Ação> com <condição de falha>
  Scenario: Login com senha incorreta
    Given um entregador com conta cadastrada na plataforma
    When o entregador envia uma senha incorreta
    # Then: mensagem de erro é o resultado observável. Sem mencionar HTTP 401 ou banco.
    Then o sistema exibe uma mensagem de erro de autenticação
    And o entregador permanece na tela de login

  # --- CENÁRIO 3: Erro de estado ---
  # Âncora: Fluxo Alternativo "conta inexistente" + critério de aceitação da Estória 2
  Scenario: Login com conta inexistente
    # Given: o estado relevante aqui é a ausência de conta — explicitado.
    Given nenhuma conta cadastrada para o identificador informado
    When o entregador envia suas credenciais
    # Then: mesmo resultado genérico do erro anterior — intencional por segurança (não revelar se conta existe).
    Then o sistema exibe uma mensagem de erro de autenticação

  # --- CENÁRIO 4: Edge case ---
  # Âncora: seção Riscos do PRD (bloqueio por tentativas excessivas)
  # Edge case de segurança: comportamento limite após múltiplas falhas.
  Scenario: Bloqueio após exceder tentativas de login
    Given um entregador com conta cadastrada na plataforma
    And o entregador já atingiu o limite de tentativas com falha
    When o entregador tenta fazer login novamente
    Then o sistema bloqueia temporariamente o acesso
    And exibe uma mensagem informando o tempo de espera

  # --- CENÁRIO 5: Edge case de estado ---
  # Âncora: critério de aceitação da Estória 1 ("entregador já autenticado")
  Scenario: Acesso à tela de login com sessão já ativa
    # Given: pré-condição específica — sessão ativa é relevante aqui.
    Given o entregador já possui uma sessão ativa
    When o entregador tenta acessar a tela de login
    # Then: redirecionamento é o comportamento observável esperado.
    Then o sistema redireciona o entregador para o painel principal
