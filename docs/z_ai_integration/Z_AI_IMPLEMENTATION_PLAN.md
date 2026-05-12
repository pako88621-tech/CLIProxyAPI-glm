# Piano di Implementazione Super Dettagliato per Z.ai

Questo piano guiderà la futura implementazione di Z.ai come nuovo provider per CLIProxyAPI.

## Fase 1: Reverse Engineering e Analisi API

Poiché le API di `chat.z.ai` non sono pubbliche, è necessario analizzare il traffico di rete dal client web.
- **Azione:** Aprire `https://chat.z.ai` nel browser, aprire i Developer Tools (Network tab) e registrare un account.
- **Dati da raccogliere:**
  - Endpoint di Login/Refresh Token e il formato della Request/Response.
  - Endpoint di creazione chat (presumibilmente `https://api.z.ai/.../chats/new` o simile).
  - Endpoint di invio messaggio (presumibilmente `https://api.z.ai/.../chats/...`).
  - Struttura del payload (Headers, Body JSON) per l'invio del prompt.
  - Struttura della risposta in streaming (Event stream / SSE format).
  - Parametri supportati (es. gestione allegati/immagini per `image.z.ai` o `ocr.z.ai`).

## Fase 2: Sviluppo Modulo Auth (`internal/auth/zai`)

1. **Creare `internal/auth/zai/token.go`**:
   - Definire le struct `ZaiTokenData`, `ZaiTokenStorage` in base al JSON del token che il sito utilizza (es. `access_token`, `refresh_token`, `expires_in`).
   - Implementare i metodi di archiviazione come `SaveTokenToFile`, `NeedsRefresh`.

2. **Creare `internal/auth/zai/zai.go`**:
   - Implementare il gestore del Device Flow OAuth (se disponibile come per Claude/Kimi) o il gestore del refresh del token, affinché gli utenti possano autenticarsi via CLIProxyAPI e ottenere il token valido per Z.ai.

## Fase 3: Sviluppo Modulo Translator (`internal/translator/zai`)

1. **Creare `internal/translator/zai/request.go`**:
   - Scrivere funzioni per convertire il formato unificato di messaggi (OpenAI style) nel payload richiesto da Z.ai.
   - Es: mappare i ruoli "user", "assistant", "system".

2. **Creare `internal/translator/zai/response.go`**:
   - Scrivere un parser per la risposta in streaming SSE di Z.ai e tradurla nel formato `chat.completion.chunk` di OpenAI.
   - Gestire i metadati (es. conteggio token se fornito) e la chiusura dello stream.

## Fase 4: Sviluppo Runtime Executor (`internal/runtime/executor/zai_executor.go`)

1. **Implementare la logica dell'executor**:
   - Creare la struct `ZaiExecutor`.
   - Implementare l'interfaccia principale per gestire le HTTP request.
   - Aggiungere il recupero del token (da `internal/auth/zai`).
   - Costruire la request HTTP per `api.z.ai` applicando il payload dal translator.
   - Gestire l'invio e leggere lo stream di risposta.

2. **Gestione Errori e Retry**:
   - Implementare logica per gestire i token scaduti (e avviare il refresh automaticamente).
   - Logica per eventuali rate-limits specifici di Z.ai (es. HTTP 429).

## Fase 5: Integrazione Centrale e Registrazione Modelli

1. **Modificare `internal/registry/models.go` (o i file pertinenti alla lista dei modelli)**:
   - Registrare i modelli di Z.ai, come `glm-5`, `glm-4.7`, `glm-4.5`.
   - Mapparli affinché CLIProxyAPI sappia instradare le richieste con prefisso `zai-` o per i modelli GLM direttamente al `ZaiExecutor`.

2. **Modificare l'inizializzazione server (`internal/api/routes.go` o moduli simli)**:
   - Registrare i nuovi endpoint per l'autenticazione OAuth/Device Flow di Z.ai nel middleware o nei routes pubblici.

## Fase 6: Testing e QA

1. **Unit Test**:
   - Scrivere test per `translator/zai` verificando che i payload finti si traducano correttamente nel formato Z.ai e viceversa.
   - Scrivere test per l'executor per verificare mock HTTP requests.

2. **Test di integrazione**:
   - Lanciare CLIProxyAPI con l'auth Z.ai, collegare un client come `claude code` o `curl` locale e verificare che le chiamate passino interamente fino al server Z.ai e la risposta arrivi fluida in streaming.
   - Verificare lo spezzettamento dei token in console e il calcolo dei costi/utilizzi.

## Fase 7: Documentazione

- Aggiornare `README.md` (e le versioni tradotte `README_CN.md`, `README_JA.md`) per annunciare il supporto a Z.ai.
- Aggiungere una guida in `docs/` su come ottenere il token e configurare i modelli Z.ai in CLIProxyAPI.
