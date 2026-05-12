# Piano di Implementazione Super Dettagliato per Z.ai

Questo piano guiderà la futura implementazione di Z.ai come nuovo provider per CLIProxyAPI.

## Fase 1: Reverse Engineering e Analisi API

Dall'analisi interattiva condotta senza login, Z.ai supporta l'autenticazione tramite un Guest token JWT generato al volo dal frontend se non si è loggati.
- L'URL usato per la creazione della chat è: `https://chat.z.ai/api/v1/chats/new`.
- L'URL usato per i messaggi stream è `https://chat.z.ai/api/v2/chat/completions?timestamp=...&signature_timestamp=...&token=...` (usando una combinazione di parametri in querystring incluso il token).
- Il Payload inviato a `/chats/new` ha il seguente formato:
  ```json
  {
    "chat": {
      "id": "",
      "title": "New Chat",
      "models": ["glm-4.7"],
      "params": {},
      "history": {
        "messages": {
          "uuid-1": {
            "id": "uuid-1",
            "parentId": null,
            "childrenIds": [],
            "role": "user",
            "content": "hello",
            "timestamp": 1778549942,
            "models": ["glm-4.7"]
          }
        },
        "currentId": "uuid-1"
      },
      "tags": [],
      "flags": [],
      "features": [],
      "mcp_servers": [],
      "enable_thinking": true,
      "auto_web_search": false,
      "message_version": 1,
      "extra": {},
      "timestamp": 1778549942195,
      "type": "default"
    }
  }
  ```
- La risposta per l'endpoint `chat/completions` restituisce un SSE standard: `data: {"type":"chat:completion","data":{"delta_content":"...","phase":"thinking"}}`.

## Fase 2: Sviluppo Modulo Auth (`internal/auth/zai`)

1. **Creare `internal/auth/zai/token.go`**:
   - Definire le struct `ZaiTokenData`, `ZaiTokenStorage`.
   - Se l'utente non ha impostato un token o esegue il login, si può recuperare il guest token facendo una prima request di scraping o intercettazione sul dominio (es. chiamando `https://chat.z.ai` e leggendo i JWT nei cookie o nell'html).
   - Implementare i metodi di archiviazione come `SaveTokenToFile`, `NeedsRefresh`.

2. **Creare `internal/auth/zai/zai.go`**:
   - Implementare il gestore del Guest login o, se si vuole supportare l'account vero, il prompt per incollare il Bearer token manualmente.

## Fase 3: Sviluppo Modulo Translator (`internal/translator/zai`)

1. **Creare `internal/translator/zai/request.go`**:
   - Scrivere funzioni per convertire i messaggi (OpenAI style) nel payload `chat.history.messages` e `chat.models` richiesto da Z.ai per la creazione della chat.

2. **Creare `internal/translator/zai/response.go`**:
   - Scrivere un parser per la risposta in streaming SSE di Z.ai (che ha eventi `chat:completion` e campi `delta_content` e `phase`).
   - Tradurla nel formato `chat.completion.chunk` di OpenAI gestendo le frasi di thinking se presenti.

## Fase 4: Sviluppo Runtime Executor (`internal/runtime/executor/zai_executor.go`)

1. **Implementare la logica dell'executor**:
   - Creare la struct `ZaiExecutor`.
   - Chiamata a `/api/v1/chats/new` per creare la sessione, estrarre l'ID dalla risposta.
   - Usare l'ID per chiamare `/api/v2/chat/completions?timestamp=...&token=...` costruendo l'URL completo con le firme necessarie (potrebbe essere necessario analizzare come Z.ai calcola `signature_timestamp` e altri parametri).
   - Leggere lo stream di risposta con SSE e passarlo al translator.

## Fase 5: Integrazione Centrale e Registrazione Modelli

1. **Modificare i modelli**:
   - Registrare `glm-5` e `glm-4.7` come modelli instradati al `ZaiExecutor`.

## Fase 6: Testing e QA

- Scrivere test di base per i translator con i payload noti.
