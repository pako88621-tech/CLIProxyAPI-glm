# Piano di Implementazione Master: Advanced Z.ai Integration

Questo documento rappresenta il piano operativo super dettagliato (Master Plan) per portare il provider Z.ai da una chat base a un provider agentico di livello enterprise per CLIProxyAPI. Il piano si articola in Task e Sotto-Task, stabilendo obiettivi e procedure esatte basate sulle documentazioni tecniche.

---

## FASE 1: Calibrazione Modelli e Tuning Registri
**Obiettivo:** Garantire che il middleware instradi correttamente le richieste senza troncare token, rispecchiando le soglie native dei modelli Zhipu GLM.

### Task 1.1: Aggiornamento JSON del Registry
- **File target:** `internal/registry/models/models.json`
- **Procedura:**
  1. Individuare il nodo `"zai"`.
  2. Per `glm-5.1`: Modificare `"context_length"` a `200000`, `"max_completion_tokens"` a `8192`.
  3. Aggiungere blocco `"thinking"`:
     ```json
     "thinking": {
       "min": 1024,
       "max": 64000,
       "zero_allowed": true
     }
     ```
  4. Per `glm-5`, `glm-4.7`, e `glm-4.5`: Impostare `"context_length"` a `128000`, `"max_completion_tokens"` a `4096`. Aggiungere `"thinking"` max a `32000`.

### Task 1.2: Allineamento Strutture Golang
- **File target:** `internal/registry/model_definitions.go`
- **Procedura:** Verificare che `GetZaiModels()` sia testato e clonato correttamente, e che il watcher riesca a ricaricare i modelli on-the-fly senza riavviare il proxy.

---

## FASE 2: Precision Thinking Budget e Reasoning
**Obiettivo:** Mappare e controllare l'espressione del ragionamento logico (System 2 thinking) di GLM attraverso configurazioni utente esplicite.

### Task 2.1: Traduzione del Request Budget
- **File target:** `internal/translator/zai/request.go`
- **Procedura:**
  1. All'interno di `TranslateRequest`, estrarre dal root JSON (formato OpenAI) la chiave `thinking.budget_tokens` (se presente).
  2. Implementare logica di switch:
     - Se `budget_tokens` è `0` -> `"enable_thinking": false` in `ZaiChat`.
     - Se `budget_tokens` > `0` -> `"enable_thinking": true` e mappare il valore numerico su stringa (es. `"params": {"thinking_effort": "high"}`) o iniettarlo direttamente se `budget_tokens` è nativamente supportato da API z.ai.
  3. Scrivere Unit Test: `TestTranslateRequest_ThinkingBudget`.

### Task 2.2: Stabilizzazione Parsing Streaming
- **File target:** `internal/translator/zai/response.go`
- **Procedura:**
  1. Analizzare i chunk in ingresso con `phase == "thinking"`.
  2. Assicurarsi che i chunk Thinking siano inviati al client (OpenAI/Claude) tramite il field `reasoning_content` del Delta.
  3. Scrivere Unit Test con un file JSON di dump reale di GLM-5 per verificare che non ci siano bleeding tra il testo di ragionamento e il contenuto effettivo.

---

## FASE 3: Automazione Autenticazione & Gestione Account
**Obiettivo:** Rimuovere l'intervento manuale per l'acquisizione del token, supportando logiche scalabili in background.

### Task 3.1: Guest Token Fetcher Dinamico
- **File target:** `internal/auth/zai/zai.go`
- **Procedura:**
  1. Implementare un modulo interno `FetchDynamicGuestToken()`.
  2. Questa funzione avvierà una simulazione HTTP per ottenere il cookie di sessione da `https://chat.z.ai`, per poi chiamare l'endpoint nascosto di registrazione guest (es. `/api/v1/auths/guest`).
  3. Estrarre il JWT dall'Header `Authorization` o dai cookies di risposta.
  4. Salvare automaticamente il token su disco chiamando `SaveTokenToFile()`.

### Task 3.2: OTP Login TUI (Per utenti Pro / GLM-5.1)
- **File target:** `internal/tui/auth_tab.go` e `internal/auth/zai/zai.go`
- **Procedura:**
  1. Creare l'interfaccia a terminale che chieda: "Inserisci numero/email per Z.ai".
  2. `ZaiAuth.SendOTP(ctx, account)`
  3. Chiedere il codice OTP.
  4. `ZaiAuth.VerifyOTP(ctx, code)` -> salva Access + Refresh token nel file system.
  5. Integrazione con `auto_refresh_loop.go` per rinfrescare i JWT in background quando `NeedsRefresh()` restituisce true.

---

## FASE 4: Integrazione Capacità Agentiche (Tool Calling)
**Obiettivo:** Renderizzare Z.ai un provider agentico compatibile con Antigravity / Kilo Code permettendo l'invocazione locale di script Bash, File System etc.

### Task 4.1: Mappatura Request Tools
- **File target:** `internal/translator/zai/request.go`
- **Procedura:**
  1. Se `root.Get("tools").Exists()` -> Estrapolare l'array dei tool di OpenAI.
  2. Implementare **Opzione B (Prompt Injection)** se l'API non ha mapping nativo 1:1:
     - Formattare una stringa System Prompt nascosta: `"Sei un agente in grado di usare strumenti. Se vuoi usare lo strumento X, rispondi ESATTAMENTE in formato XML <tool_call><name>X</name><args>...</args></tool_call>"`.
     - Iniettarla come messaggio `role: user` (nascosto prima del vero prompt dell'utente) nella history di Z.ai.
  3. Mappare i messaggi passati di tipo `role: tool` in stringhe formattate come risultati.

### Task 4.2: Parsing Risposte Tool (Synthetic Calling)
- **File target:** `internal/translator/zai/response.go`
- **Procedura:**
  1. Introdurre una Struct di buffer (`ToolCallBuffer`) nel traduttore di streaming.
  2. Quando lo streaming di Z.ai invia `phase: content`, monitorare i caratteri. Se inizia con `<tool_call>`, mettere in pausa l'invio al client.
  3. Accumulare i chunk fino al tag di chiusura `</tool_call>`.
  4. Convertire l'intera stringa XML catturata in un oggetto `choices[0].delta.tool_calls` e inoltrarla al client (compatibile con OpenAI stream).

### Task 4.3: Testing Intensivo Tool Calling
- **File target:** `test/zai_tool_calling_test.go`
- **Procedura:**
  1. Creare un test E2E simulando Kilo Code che chiede: "Qual è la cartella corrente?".
  2. Mappare il finto chunk XML e verificare che `response.go` lo traduca in `tool_calls` correttamente strutturato, con il giusto `toolu_id`.

---

## FASE 5: Validazione, QA e Rilascio
**Obiettivo:** Pre-commit steps e deployment.
- **Task 5.1:** Eseguire `gofmt` e `go test ./...` in tutto l'ambiente locale.
- **Task 5.2:** Test manuale con un frontend (es. Kilo Code).
- **Task 5.3:** Creazione PR con la nomenclatura: `feat: full enterprise integration of Z.ai provider (models, auth, thinking, tools)`.
