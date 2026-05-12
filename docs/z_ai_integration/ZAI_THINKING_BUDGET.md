# Integrazione e Gestione del "Thinking" per GLM

I modelli GLM moderni (come GLM-5) sono dotati di capacità di pensiero esteso (System 2 Thinking). Questa guida spiega come estrarre il pensiero generato da Z.ai e come controllare quanto budget dedicarvi.

## 1. Estrazione Corretta del Thinking (Response Translation)

Nell'API di Z.ai, i flussi in streaming (SSE) indicano se stanno inviando "ragionamento" o "testo finale" usando il campo `phase`.

- **Testo normare:** `{"phase": "content", "delta_content": "Ciao!"}`
- **Testo di ragionamento:** `{"phase": "thinking", "delta_content": "Devo salutare l'utente..."}`

Per essere compatibili con client avanzati (Claude Code, Antigravity, Kilo Code), in `internal/translator/zai/response.go` il flusso deve essere mappato allo standard OpenAI (usato nativamente in CLIProxyAPI per l'intermediazione).

**Flusso implementato:**
Se `phase == "thinking"`, il contenuto viene inserito nel campo `reasoning_content` del Delta. Se `phase == "content"`, va nel campo `content`.
Questa separazione permette ai client di renderizzare blocchi grigi/espandibili nell'interfaccia utente (UI) per il ragionamento logico.

## 2. Gestione del Thinking Budget (Request Translation)

Per decidere "quanto" far ragionare il modello (es. "Low", "Medium", "High" oppure un numero specifico di token come `budget_tokens: 4096`), è necessario iniettare la configurazione nella richiesta HTTP verso `api.z.ai`.

### Mappatura delle Richieste
Quando un utente configura un budget (es. passando `{"thinking": {"budget_tokens": 4096}}` o usando i parametri di CLIProxyAPI), il traduttore (`request.go`) deve convertire questa richiesta.

In Z.ai, la configurazione si riflette nel payload iniziale inviato a `/chats/new`:
```json
{
  "chat": {
    "enable_thinking": true,
    "params": {
      "thinking_effort": "high" // o parametro equivalente basato sul reverse engineering approfondito
    }
  }
}
```

### Prossime Implementazioni di Codice
In `internal/translator/zai/request.go`:
1. Estrarre `budget_tokens` dalla radice JSON (se inviato nel formato OpenAI-o1).
2. Se `budget_tokens` > 0, impostare `enable_thinking = true`.
3. Tradurre la metrica di `budget` in una logica accettata da GLM (se GLM accetta solo stringhe come `low`/`high`, fare un mapping basato su soglie numeriche, come fa Claude).
